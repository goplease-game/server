package worker_test

import (
	"testing"
	"time"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/test"
	"github.com/ognev-dev/goplease/test/factory"
	cleanupexpiredpasswordchangerequests "github.com/ognev-dev/goplease/worker/cleanup_expired_password_change_requests"
)

func TestCleanupExpiredPasswordChangeRequests(t *testing.T) {
	user := create[ds.User](t)

	_, err := factory.Ten(tt.Factory.CreatePasswordResetToken, ds.PasswordResetToken{
		UserID:    user.ID,
		ExpiresAt: time.Now().AddDate(0, 0, -1),
	})
	test.CheckErr(t, err)

	runJob(t, cleanupexpiredpasswordchangerequests.NewJob())

	test.AssertNotInDB(t, tt.DB, "password_reset_tokens", test.Data{
		"user_id": user.ID,
	})

	token := create[ds.PasswordResetToken](t)

	runJob(t, cleanupexpiredpasswordchangerequests.NewJob())

	test.AssertInDB(t, tt.DB, "password_reset_tokens", test.Data{
		"id": token.ID,
	})
}
