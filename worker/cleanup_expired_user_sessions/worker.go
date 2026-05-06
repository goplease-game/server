// Package cleanupexpiredusersessions ...
package cleanupexpiredusersessions

import (
	"context"

	"github.com/go-co-op/gocron/v2"
	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/service"
)

// Job implements the worker.Job interface for cleaning up expired user sessions.
type Job struct{}

// NewJob ...
func NewJob() *Job {
	return &Job{}
}

// Name returns the unique name of the job.
func (w Job) Name() string {
	return "CLEANUP:EXPIRED_USER_SESSIONS"
}

// Schedule defines when the job should run.
// This job is scheduled to run once daily at midnight.
func (w Job) Schedule() gocron.JobDefinition {
	return gocron.DailyJob(1,
		gocron.NewAtTimes(gocron.NewAtTime(0, 0, 0)),
	)
}

// Do executes the job's task, which is to delete all records
// from the user_sessions table where the expiration date is in the past.
func (w Job) Do(ctx context.Context, _ *service.Service, db *app.DB) (err error) {
	_, err = db.Exec(ctx, "DELETE FROM user_sessions WHERE expires_at < NOW()")
	return
}
