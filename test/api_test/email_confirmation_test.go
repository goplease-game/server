package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/server/handler"
	"github.com/ognev-dev/goplease/server/request"
	"github.com/ognev-dev/goplease/server/response"
	"github.com/ognev-dev/goplease/test"
	"github.com/stretchr/testify/assert"
)

func TestResendEmailConfirmationCode(t *testing.T) {
	user := create(t, ds.User{EmailConfirmed: false})
	loginAs(t, user)

	_, err := tt.DB.Exec(context.TODO(), "UPDATE users SET email_confirmed = false WHERE id = $1", user.ID)
	test.CheckErr(t, err)

	var resp response.Status
	Request(t, RequestArgs{
		method:       http.MethodPost,
		path:         "/users/email-confirmation-code/",
		bindResponse: &resp,
		assertStatus: http.StatusOK,
	})

	vars := test.LoadEmailVars(t, user.Email)

	assert.Equal(t, user.Username, app.String(vars["username"]))
	assert.Equal(t, user.Email, app.String(vars["email"]))

	test.AssertInDB(t, tt.DB, "email_confirmations", test.Data{
		"user_id": user.ID,
		"code":    vars["code"],
	})

	t.Run("too many attempts", func(t *testing.T) {
		var resp handler.Error
		Request(t, RequestArgs{
			method:       http.MethodPost,
			path:         "/users/email-confirmation-code/",
			bindResponse: &resp,
			assertStatus: http.StatusTooManyRequests,
		})
	})
}

func TestUserConfirmEmail(t *testing.T) {
	ec := create[ds.EmailConfirmation](t)

	req := request.ConfirmEmail{
		Code: ec.Code,
	}

	var resp response.Status
	Request(t, RequestArgs{
		method:       http.MethodPost,
		path:         "/users/confirm-email",
		body:         req,
		bindResponse: &resp,
		assertStatus: http.StatusOK,
	})

	test.AssertInDB(t, tt.DB, "users", test.Data{
		"id":              ec.UserID,
		"email_confirmed": true,
	})

	test.AssertNotInDB(t, tt.DB, "email_confirmations", test.Data{
		"code": ec.Code,
	})

	test.AssertInDB(t, tt.DB, "event_logs", test.Data{
		"user_id":   ec.UserID,
		"type":      ds.EventLogUserEmailConfirmed,
		"is_public": false,
	})
	test.AssertInDB(t, tt.DB, "event_logs", test.Data{
		"user_id":   ec.UserID,
		"type":      ds.EventLogUserAccountActivated,
		"is_public": true,
	})
}
