package worker_test

import (
	"testing"
	"time"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/test"
	deleteunconfirmedusers "github.com/ognev-dev/goplease/worker/delete_unconfirmed_users"
)

func TestDeleteUnconfirmedUsers(t *testing.T) {
	user := create(t, ds.User{
		EmailConfirmed: false,
		CreatedAt:      time.Now().Add(-25 * time.Hour),
	})

	runJob(t, deleteunconfirmedusers.NewJob())

	test.AssertInDB(t, tt.DB, "users", test.Data{
		"id":              user.ID,
		"deleted_at":      test.NotNull,
		"email_confirmed": false,
	})
}
