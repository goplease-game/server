// Package deleteunconfirmedusers ...
package deleteunconfirmedusers

import (
	"context"

	"github.com/go-co-op/gocron/v2"
	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/service"
)

// Job implements the worker.Job interface for cleaning up users with unconfirmed emails.
type Job struct{}

// NewJob ...
func NewJob() *Job {
	return &Job{}
}

// Name returns the unique name of the job.
func (w Job) Name() string {
	return "DELETE:UNCONFIRMED_USERS"
}

// Schedule defines when the job should run.
// This job is scheduled to run once daily at midnight.
func (w Job) Schedule() gocron.JobDefinition {
	return gocron.DailyJob(1,
		gocron.NewAtTimes(gocron.NewAtTime(0, 0, 0)),
	)
}

// Do mark unconfirmed users as deleted (Another job will take care of cleanup of deleted users).
func (w Job) Do(ctx context.Context, _ *service.Service, db *app.DB) (err error) {
	_, err = db.Exec(ctx,
		"UPDATE users SET deleted_at = created_at "+
			"WHERE email_confirmed IS FALSE AND deleted_at IS NULL "+
			"AND created_at < NOW() - INTERVAL '24 hours'")
	return
}
