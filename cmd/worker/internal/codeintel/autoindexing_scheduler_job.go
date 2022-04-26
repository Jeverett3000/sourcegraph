package codeintel

import (
	"context"

	"github.com/sourcegraph/sourcegraph/cmd/worker/job"
	"github.com/sourcegraph/sourcegraph/internal/codeintel/autoindexing/background/scheduler"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
)

type autoindexingScheduler struct{}

func NewAutoindexingSchedulerJob() job.Job {
	return &autoindexingScheduler{}
}

func (j *autoindexingScheduler) Config() []env.Config {
	return []env.Config{
		scheduler.ConfigInst,
	}
}

func (j *autoindexingScheduler) Routines(ctx context.Context) ([]goroutine.BackgroundRoutine, error) {
	return []goroutine.BackgroundRoutine{
		scheduler.NewScheduler(),
	}, nil
}
