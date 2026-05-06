package worker_test

import (
	"testing"
	"time"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/test"
	"github.com/ognev-dev/goplease/test/factory"
	"github.com/ognev-dev/goplease/worker/cleanup_change_email_requests"
)

func TestCleanupChangeEmailRequests(t *testing.T) {
	// create user
	user := create[ds.User](t)

	// create 10 "email change requests" that is expired
	_, err := factory.Ten(tt.Factory.CreateChangeEmailRequest, ds.ChangeEmailRequest{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(-time.Hour),
	})
	test.CheckErr(t, err)

	// run worker's job
	runJob(t, cleanupchangeemailrequests.NewJob())

	// check that job did what expected
	test.AssertNotInDB(t, tt.DB, "change_email_requests", test.Data{
		"user_id": user.ID,
	})

	// do it again, against false-positive case
	// 1. create "email change requests" that is fresh
	// 2. run job
	// 3. "email change requests" should not be removed (yet)
	req := create[ds.ChangeEmailRequest](t)
	runJob(t, cleanupchangeemailrequests.Job{})
	test.AssertInDB(t, tt.DB, "change_email_requests", test.Data{"id": req.ID})
}
