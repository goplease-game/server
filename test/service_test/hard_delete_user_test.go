package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/test"
	"github.com/ognev-dev/goplease/test/factory"
	"github.com/stretchr/testify/assert"
)

func TestCleanupDeletedUser(t *testing.T) {
	user := create[ds.User](t)
	ctx := context.Background()

	// user sessions
	_, err := factory.Five(tt.Factory.CreateUserSession, ds.UserSession{UserID: user.ID})
	test.CheckErr(t, err)

	// email confirmations
	_, err = factory.Five(tt.Factory.CreateEmailConfirmation, ds.EmailConfirmation{UserID: user.ID})
	test.CheckErr(t, err)

	// password resets
	_, err = factory.Five(tt.Factory.CreatePasswordResetToken, ds.PasswordResetToken{UserID: user.ID})
	test.CheckErr(t, err)

	// change email requests
	_, err = factory.Five(tt.Factory.CreateChangeEmailRequest, ds.ChangeEmailRequest{UserID: user.ID})
	test.CheckErr(t, err)

	err = tt.Service.CleanupDeletedUser(ctx, user.ID)
	test.CheckErr(t, err)

	test.AssertNotInDB(t, tt.DB, "user_sessions", test.Data{"user_id": user.ID})
	test.AssertNotInDB(t, tt.DB, "email_confirmations", test.Data{"user_id": user.ID})
	test.AssertNotInDB(t, tt.DB, "password_reset_tokens", test.Data{"user_id": user.ID})
	test.AssertNotInDB(t, tt.DB, "change_email_requests", test.Data{"user_id": user.ID})

	user, err = tt.Service.GetUserByID(ctx, user.ID)
	test.CheckErr(t, err)

	assert.True(t, strings.HasPrefix(user.Username, ds.DeletedUsername))
	assert.True(t, strings.HasPrefix(user.Email, ds.DeletedUsername))
	assert.True(t, strings.HasPrefix(user.Password, ds.DeletedUsername))
}
