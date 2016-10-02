package backend

import (
	"strings"
	"time"

	"github.com/AaronO/go-git-http"
	"gopkg.in/inconshreveable/log15.v2"

	"context"

	"sourcegraph.com/sourcegraph/sourcegraph/api/sourcegraph"
	authpkg "sourcegraph.com/sourcegraph/sourcegraph/pkg/auth"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/langp"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/store"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/vcs"
	"sourcegraph.com/sourcegraph/sourcegraph/services/backend/accesscontrol"
	"sourcegraph.com/sourcegraph/sourcegraph/services/backend/internal/localstore"
	"sourcegraph.com/sourcegraph/sourcegraph/services/events"
	"sourcegraph.com/sourcegraph/sourcegraph/services/ext/github"
	"sourcegraph.com/sourcegraph/sourcegraph/services/svc"
	"sourcegraph.com/sqs/pbtypes"
)

// emptyGitCommitID is used in githttp.Event objects in the Last (or
// Commit) field to signify that a branch was created (or deleted).
const emptyGitCommitID = "0000000000000000000000000000000000000000"

var MirrorRepos sourcegraph.MirrorReposServer = &mirrorRepos{}

type mirrorRepos struct{}

var _ sourcegraph.MirrorReposServer = (*mirrorRepos)(nil)

func (s *mirrorRepos) RefreshVCS(ctx context.Context, op *sourcegraph.MirrorReposRefreshVCSOp) (*pbtypes.Void, error) {
	ctx = context.WithValue(ctx, github.GitHubTrackingContextKey, "RefreshVCS")
	actor := authpkg.ActorFromContext(ctx)
	asUserUID := actor.UID

	// Only admin users and the repo updater are allowed to perform this operation
	// as a different user.
	canImpersonateUser := actor.HasScope("internal:repoupdater")
	if op.AsUser != nil && canImpersonateUser {
		asUserUID = op.AsUser.UID
	}

	// Use the auth token for asUserUID if it can be successfully looked up (it may fail if that user doesn't have one),
	// otherwise proceed without their credentials. It will work for public repos.
	remoteOpts := vcs.RemoteOpts{}
	if asUserUID != "" {
		extToken, err := authpkg.FetchGitHubToken(ctx, asUserUID)
		if err == nil {
			// Set the auth token to be used in repo VCS operations.
			remoteOpts.HTTPS = &vcs.HTTPSConfig{
				Pass: extToken.Token,
			}

			// Set a GitHub client authed as the user in the context, to be used to make GitHub API calls.
			ctx, err = github.NewContextWithAuthedClient(authpkg.WithActor(ctx, &authpkg.Actor{UID: asUserUID, GitHubToken: extToken.Token}))
			if err != nil {
				log15.Error("RefreshVCS: failed github authentication for user", "error", err, "uid", asUserUID)
				return nil, err
			}
		}
	}

	repo, err := localstore.Repos.Get(ctx, op.Repo)
	if err != nil {
		log15.Error("RefreshVCS: failed to get repo", "error", err, "repo", op.Repo)
		return nil, err
	}

	vcsRepo, err := localstore.RepoVCS.Open(ctx, repo.ID)
	if err != nil {
		log15.Error("RefreshVCS: failed to open VCS", "error", err, "URI", repo.URI)
		return nil, err
	}
	if err := s.updateRepo(ctx, repo, vcsRepo, remoteOpts); err != nil {
		if !vcs.IsRepoNotExist(err) {
			log15.Error("RefreshVCS: update repo failed unexpectedly", "error", err, "repo", repo.URI)
			return nil, err
		}
		if err.(vcs.RepoNotExistError).CloneInProgress {
			log15.Info("RefreshVCS: clone in progress, not updating", "repo", repo.URI)
			return &pbtypes.Void{}, nil
		}
		if err := s.cloneRepo(ctx, repo, remoteOpts); err != nil {
			log15.Info("RefreshVCS: cloneRepo failed", "error", err, "repo", repo.URI)
			return nil, err
		}
	}

	{
		now := time.Now()
		ctx2 := authpkg.WithActor(ctx, &authpkg.Actor{Scope: map[string]bool{"internal:repo-internal-update": true}})
		if err := localstore.Repos.InternalUpdate(ctx2, repo.ID, store.InternalRepoUpdate{VCSSyncedAt: &now}); err != nil {
			log15.Info("RefreshVCS: updating repo internal VCSSyncedAt failed", "err", err, "repo", repo.URI)
			return nil, err
		}
	}

	return &pbtypes.Void{}, nil
}

func (s *mirrorRepos) cloneRepo(ctx context.Context, repo *sourcegraph.Repo, remoteOpts vcs.RemoteOpts) error {
	if err := accesscontrol.VerifyUserHasWriteAccess(ctx, "MirrorRepos.cloneRepo", repo.URI); err != nil {
		return err
	}

	err := localstore.RepoVCS.Clone(elevatedActor(ctx), repo.ID, &store.CloneInfo{
		CloneURL:   repo.HTTPCloneURL,
		RemoteOpts: remoteOpts,
	})
	if err != nil && err != vcs.ErrRepoExist {
		return err
	}

	// We've just cloned the repository, do a sanity check to ensure we
	// can resolve the DefaultBranch.
	res, err := svc.Repos(ctx).ResolveRev(elevatedActor(ctx), &sourcegraph.ReposResolveRevOp{
		Repo: repo.ID,
		Rev:  repo.DefaultBranch,
	})
	if err != nil {
		return err
	}

	go func() {
		// Both of these are best effort, so we ignore the errors they
		// return + kick them off in a goroutine to not block the
		// clone

		// Ask the Language Processor to prepare the workspace. This is async
		_ = langp.DefaultClient.Prepare(ctx, &langp.RepoRev{
			// TODO(slimsag): URI is correct only where the repo URI and clone
			// URI are directly equal.. but CloneURI is only correct (for Go)
			// when it directly matches the package import path.
			Repo:   repo.URI,
			Commit: res.CommitID,
		})

		_, _ = svc.Async(ctx).RefreshIndexes(ctx, &sourcegraph.AsyncRefreshIndexesOp{
			Repo:   repo.ID,
			Source: "clone",
		})
	}()

	return nil
}

func (s *mirrorRepos) updateRepo(ctx context.Context, repo *sourcegraph.Repo, vcsRepo vcs.Repository, remoteOpts vcs.RemoteOpts) error {
	// TODO: Need to detect new tags and copy git_transport.go in event publishing
	// behavior.

	// Grab the current revision of every branch.
	branches, err := vcsRepo.Branches(ctx, vcs.BranchesOptions{})
	if err != nil {
		return err
	}

	// Update everything.
	updateResult, err := vcsRepo.UpdateEverything(ctx, remoteOpts)
	if err != nil {
		return err
	}

	forcePushes := make(map[string]bool)
	for _, change := range updateResult.Changes {
		switch change.Op {
		case vcs.NewOp, vcs.ForceUpdatedOp:
			// Skip refs that aren't branches, such as GitHub
			// "refs/pull/123/head" and "refs/pull/123/merge" refs
			// that are created for each pull request. In the future
			// we may want to handle these, but skipping them for now
			// is good because otherwise when we add a new mirror
			// repo, builds and notifications are triggered for all
			// historical PRs.
			if strings.HasPrefix(change.Branch, "refs/") {
				continue
			}

			// Determine the event type, and if it's a force push mark for later to
			// avoid additional work.
			eventType := events.GitCreateBranchEvent
			gitEventType := githttp.EventType(githttp.PUSH)
			if change.Op == vcs.ForceUpdatedOp {
				// Force push, remember for later.
				forcePushes[change.Branch] = true
				eventType = events.GitPushEvent
				gitEventType = githttp.PUSH_FORCE
			}

			// Determine the new branch head revision.
			head, err := vcsRepo.ResolveRevision(ctx, "refs/heads/"+change.Branch)
			if err != nil {
				return err
			}

			// Publish the event.
			// TODO: what about GitPayload.ContentEncoding field?
			events.Publish(eventType, events.GitPayload{
				Actor: *authpkg.ActorFromContext(ctx).UserSpec(),
				Repo:  repo.ID,
				Event: githttp.Event{
					Type:   gitEventType,
					Commit: string(head),
					Branch: change.Branch,
					Last:   emptyGitCommitID,
					// TODO: specify Dir, Tag, Error and Request fields somehow?
				},
			})
		}
	}

	// Find all new commits on each branch.
	for _, oldBranch := range branches {
		if _, ok := forcePushes[oldBranch.Name]; ok {
			// Already handled above.
			continue
		}

		// Determine new branch head revision.
		head, err := vcsRepo.ResolveRevision(ctx, "refs/heads/"+oldBranch.Name)
		if err == vcs.ErrRevisionNotFound {
			// Branch was deleted.
			// TODO: what about GitPayload.ContentEncoding field?
			events.Publish(events.GitDeleteBranchEvent, events.GitPayload{
				Actor: *authpkg.ActorFromContext(ctx).UserSpec(),
				Repo:  repo.ID,
				Event: githttp.Event{
					Type:   githttp.PUSH,
					Commit: emptyGitCommitID,
					Branch: oldBranch.Name,
					// TODO: specify Dir, Tag, Error and Request fields somehow?
				},
			})
			continue
		} else if err != nil {
			return err
		}
		if head == oldBranch.Head {
			continue // No new commits.
		}

		// Publish an event for the new commits pushed.
		// TODO: what about GitPayload.ContentEncoding field?
		events.Publish(events.GitPushEvent, events.GitPayload{
			Actor: *authpkg.ActorFromContext(ctx).UserSpec(),
			Repo:  repo.ID,
			Event: githttp.Event{
				Type:   githttp.PUSH,
				Commit: string(head),
				Last:   string(oldBranch.Head),
				Branch: oldBranch.Name,
				// TODO: specify Dir, Tag, Error and Request fields somehow?
			},
		})
	}
	return nil
}
