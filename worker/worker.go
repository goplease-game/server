// Package worker provides a background job scheduler for running periodic maintenance tasks.
// Jobs are registered at startup and run according to their defined schedules.
package worker

import (
	"context"
	"log"

	"github.com/go-co-op/gocron/v2"
	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/service"
	"github.com/ognev-dev/goplease/tracing"
	"github.com/ognev-dev/goplease/worker/cleanup_change_email_requests"
	"github.com/ognev-dev/goplease/worker/cleanup_deleted_users"
	"github.com/ognev-dev/goplease/worker/cleanup_expired_password_change_requests"
	"github.com/ognev-dev/goplease/worker/cleanup_expired_user_sessions"
	"github.com/ognev-dev/goplease/worker/delete_unconfirmed_users"
)

// List of registered jobs.
var jobs = []Job{
	cleanupexpiredpasswordchangerequests.NewJob(),
	cleanupchangeemailrequests.NewJob(),
	deleteunconfirmedusers.NewJob(),
	cleanupexpiredusersessions.NewJob(),
	cleanupdeletedusers.NewJob(),
}

// Job defines the interface for a background worker job.
// Each job must provide a name, a schedule, and an execution function.
type Job interface {
	// Name returns the unique name of the job.
	Name() string

	// Schedule defines when the job should run.
	Schedule() gocron.JobDefinition

	// Do executes the job's task.
	Do(ctx context.Context, s *service.Service, db *app.DB) error
}

// Start initializes and starts the background worker scheduler.
// It sets up the database connection, tracer, and registers all defined jobs.
// The scheduler runs until the provided context is canceled.
func Start(ctx context.Context) error {
	tracer, err := tracing.New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	s, err := gocron.NewScheduler()
	if err != nil {
		return err
	}

	db, err := app.NewDB(ctx)
	if err != nil {
		return err
	}

	services := service.New(db, tracer)

	for _, j := range jobs {
		_, err = s.NewJob(j.Schedule(),
			gocron.NewTask(func() {
				println("[WORKER] Executing job:", j.Name())
				err := j.Do(ctx, services, db)
				if err != nil {
					println("[WORKER] [ERROR]", err.Error())
				}
			}),
		)
		if err != nil {
			println("[WORKER]", j.Name(), err.Error())
			return err
		}
		println(j.Name(), "worker's job registered")
	}

	s.Start()

	go func() {
		<-ctx.Done()

		db.Close()

		err = s.Shutdown()
		if err != nil {
			println("[WORKER] [SHUTDOWN]", err.Error())
		}
	}()

	return nil
}
