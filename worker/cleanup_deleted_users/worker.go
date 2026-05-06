// Package cleanupdeletedusers ...
package cleanupdeletedusers

import (
	"context"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/app/service"
	"golang.org/x/sync/errgroup"
)

// Job implements the worker.Job interface for cleaning up deleted user accounts.
type Job struct{}

// NewJob ...
func NewJob() *Job {
	return &Job{}
}

// Name returns the unique name of the job.
func (w Job) Name() string {
	return "CLEANUP:DELETED_USER_ACCOUNTS"
}

// Schedule defines when the job should run.
// This job is scheduled to run once daily at midnight.
func (w Job) Schedule() gocron.JobDefinition {
	return gocron.DailyJob(1,
		gocron.NewAtTimes(gocron.NewAtTime(0, 0, 0)),
	)
}

// Do executes the job's task, which is cleanup users data who have been
// soft-deleted for more than a certain period.
func (w Job) Do(ctx context.Context, s *service.Service, _ *app.DB) (err error) {
	users, _, err := s.FilterUsers(ctx, ds.UsersFilter{
		DeletedAt:  ds.DtBefore(time.Now().Add(-ds.CleanupDeletedUserAfter)),
		NotCleaned: true,
		PerPage:    ds.PerPageNoLimit,
	})
	if err != nil {
		return
	}

	var eg errgroup.Group
	for _, user := range users {
		eg.Go(func() error {
			return s.CleanupDeletedUser(ctx, user.ID)
		})
	}

	return eg.Wait()
}
