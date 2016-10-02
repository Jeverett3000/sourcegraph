package localstore

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/gitserver"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/store"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/store/mockstore"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/vcs"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/vcs/gitcmd"
	"sourcegraph.com/sourcegraph/sourcegraph/services/backend/accesscontrol"
)

var MockRepoVCS *mockstore.RepoVCS

// repoVCS is a local filesystem-backed implementation of the RepoVCS
// store interface.
type repoVCS struct{}

// getRepoDir gets the dir (relative to the base repo VCS storage dir)
// where the repo's git repository data lives.
func getRepoDir(ctx context.Context, repo int32) (string, error) {
	dir, err := appDBH(ctx).SelectStr("SELECT uri FROM repo WHERE id=$1;", repo)
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", grpc.Errorf(codes.NotFound, "repo not found (looking up dir): %d", repo)
	}
	return dir, nil
}

func (s *repoVCS) Open(ctx context.Context, repo int32) (vcs.Repository, error) {
	if MockRepoVCS != nil {
		return MockRepoVCS.Open(ctx, repo)
	}

	if err := accesscontrol.VerifyUserHasReadAccess(ctx, "RepoVCS.Open", repo); err != nil {
		return nil, err
	}
	dir, err := getRepoDir(ctx, repo)
	if err != nil {
		return nil, err
	}

	return gitcmd.Open(dir), nil
}

func (s *repoVCS) Clone(ctx context.Context, repo int32, info *store.CloneInfo) error {
	if MockRepoVCS != nil {
		return MockRepoVCS.Clone(ctx, repo, info)
	}

	if err := accesscontrol.VerifyUserHasWriteAccess(ctx, "RepoVCS.Clone", repo); err != nil {
		return err
	}
	dir, err := getRepoDir(ctx, repo)
	if err != nil {
		return err
	}

	return gitserver.DefaultClient.Clone(ctx, dir, info.CloneURL, &info.RemoteOpts)
}
