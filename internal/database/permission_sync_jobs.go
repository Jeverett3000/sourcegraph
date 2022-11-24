package database

import (
	"context"
	"time"

	"github.com/keegancsmith/sqlf"
	"github.com/lib/pq"

	"github.com/sourcegraph/log"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	dbworkerstore "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

type PermissionSyncJobStore interface {
	basestore.ShareableStore
	With(other basestore.ShareableStore) PermissionSyncJobStore
	// Transact begins a new transaction and make a new PermsStore over it.
	Transact(ctx context.Context) (PermissionSyncJobStore, error)
	Done(err error) error

	CreateUserSyncJob(ctx context.Context, user int32, highPriority bool) error
	CreateRepoSyncJob(ctx context.Context, repo int32, highPriority bool) error
}

type permissionSyncJobStore struct {
	logger log.Logger
	*basestore.Store
}

var _ PermissionSyncJobStore = (*permissionSyncJobStore)(nil)

func PermissionSyncJobsWith(logger log.Logger, other basestore.ShareableStore) PermissionSyncJobStore {
	return &permissionSyncJobStore{logger: logger, Store: basestore.NewWithHandle(other.Handle())}
}

func (s *permissionSyncJobStore) With(other basestore.ShareableStore) PermissionSyncJobStore {
	return &permissionSyncJobStore{logger: s.logger, Store: s.Store.With(other)}
}

func (s *permissionSyncJobStore) Transact(ctx context.Context) (PermissionSyncJobStore, error) {
	return s.transact(ctx)
}

func (s *permissionSyncJobStore) transact(ctx context.Context) (*permissionSyncJobStore, error) {
	txBase, err := s.Store.Transact(ctx)
	return &permissionSyncJobStore{Store: txBase}, err
}

func (s *permissionSyncJobStore) Done(err error) error {
	return s.Store.Done(err)
}

func (s *permissionSyncJobStore) CreateUserSyncJob(ctx context.Context, user int32, highPriority bool) error {
	return s.createSyncJob(ctx, &PermissionSyncJob{
		UserID:       int(user),
		HighPriority: highPriority,
	})
}

func (s *permissionSyncJobStore) CreateRepoSyncJob(ctx context.Context, repo int32, highPriority bool) error {
	return s.createSyncJob(ctx, &PermissionSyncJob{
		RepositoryID: int(repo),
		HighPriority: highPriority,
	})
}

const permissionSyncJobCreateQueryFmtstr = `
INSERT INTO
	permission_sync_jobs (
		process_after,
		repository_id,
		user_id,
		high_priority,
		invalidate_caches,
	)
	VALUES (
		%s,
		%s,
		%s,
		%s,
		%s
	)
	RETURNING %s
`

func (s *permissionSyncJobStore) createSyncJob(ctx context.Context, job *PermissionSyncJob) error {
	q := sqlf.Sprintf(
		permissionSyncJobCreateQueryFmtstr,
		job.ProcessAfter,
		job.RepositoryID,
		job.UserID,
		dbutil.NewNullInt(int(job.RepositoryID)),
		dbutil.NewNullInt(int(job.UserID)),
		job.HighPriority,
		job.InvalidateCaches,
		sqlf.Join(PermissionSyncJobColumns, ", "),
	)

	row := s.QueryRow(ctx, q)
	if err := scanPermissionSyncJob(job, row); err != nil {
		return err
	}

	return nil
}

type PermissionSyncJob struct {
	ID              int
	State           string
	FailureMessage  *string
	QueuedAt        time.Time
	StartedAt       *time.Time
	FinishedAt      *time.Time
	ProcessAfter    *time.Time
	NumResets       int
	NumFailures     int
	LastHeartbeatAt time.Time
	ExecutionLogs   []workerutil.ExecutionLogEntry
	WorkerHostname  string
	Cancel          bool

	RepositoryID int
	UserID       int

	HighPriority     bool
	InvalidateCaches bool
}

func (j *PermissionSyncJob) RecordID() int { return j.ID }

var PermissionSyncJobColumns = []*sqlf.Query{
	sqlf.Sprintf("permission_sync_jobs.id"),
	sqlf.Sprintf("permission_sync_jobs.state"),
	sqlf.Sprintf("permission_sync_jobs.failure_message"),
	sqlf.Sprintf("permission_sync_jobs.queued_at"),
	sqlf.Sprintf("permission_sync_jobs.started_at"),
	sqlf.Sprintf("permission_sync_jobs.finished_at"),
	sqlf.Sprintf("permission_sync_jobs.process_after"),
	sqlf.Sprintf("permission_sync_jobs.num_resets"),
	sqlf.Sprintf("permission_sync_jobs.num_failures"),
	sqlf.Sprintf("permission_sync_jobs.last_heartbeat_at"),
	sqlf.Sprintf("permission_sync_jobs.execution_logs"),
	sqlf.Sprintf("permission_sync_jobs.worker_hostname"),
	sqlf.Sprintf("permission_sync_jobs.cancel"),

	sqlf.Sprintf("permission_sync_jobs.repository_id"),
	sqlf.Sprintf("permission_sync_jobs.user_id"),

	sqlf.Sprintf("permission_sync_jobs.high_priority"),
	sqlf.Sprintf("permission_sync_jobs.invalidate_caches"),
}

func ScanPermissionSyncJob(s dbutil.Scanner) (*PermissionSyncJob, error) {
	var job PermissionSyncJob
	if err := scanPermissionSyncJob(&job, s); err != nil {
		return nil, err
	}
	return &job, nil
}

func scanPermissionSyncJob(job *PermissionSyncJob, s dbutil.Scanner) error {
	var executionLogs []dbworkerstore.ExecutionLogEntry

	if err := s.Scan(
		&job.ID,
		&job.State,
		&job.FailureMessage,
		&job.QueuedAt,
		&job.StartedAt,
		&job.FinishedAt,
		&job.ProcessAfter,
		&job.NumResets,
		&job.NumFailures,
		&job.LastHeartbeatAt,
		pq.Array(&executionLogs),
		&job.WorkerHostname,
		&job.Cancel,
		&job.RepositoryID,
		&job.UserID,
		&job.HighPriority,
		&job.InvalidateCaches,
	); err != nil {
		return err
	}

	for _, entry := range executionLogs {
		job.ExecutionLogs = append(job.ExecutionLogs, workerutil.ExecutionLogEntry(entry))
	}
	return nil
}