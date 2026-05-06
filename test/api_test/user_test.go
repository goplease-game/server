package api_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/app/service"
	"github.com/ognev-dev/goplease/server/handler"
	"github.com/ognev-dev/goplease/server/request"
	"github.com/ognev-dev/goplease/server/response"
	"github.com/ognev-dev/goplease/test"
	"github.com/ognev-dev/goplease/test/factory/random"
	"github.com/stretchr/testify/assert"
)

func TestUserSignUp(t *testing.T) {
	req := request.UserSignUp{
		Username: random.String(),
		Email:    random.Email(),
		Password: random.String(),
	}

	var resp response.Status
	Request(t, RequestArgs{
		method:       http.MethodPost,
		path:         "/users/sign-up",
		body:         req,
		bindResponse: &resp,
		assertStatus: http.StatusOK,
	})

	test.AssertInDB(t, tt.DB, "users", test.Data{
		"username":        req.Username,
		"email":           req.Email,
		"email_confirmed": false,
	})

	vars := test.LoadEmailVars(t, req.Email)

	assert.Equal(t, req.Username, app.String(vars["username"]))
	assert.Equal(t, req.Email, app.String(vars["email"]))

	user, err := tt.Service.GetUserByEmail(context.Background(), req.Email)
	test.CheckErr(t, err)

	test.AssertInDB(t, tt.DB, "email_confirmations", test.Data{
		"user_id": user.ID,
		"code":    vars["code"],
	})

	test.AssertInDB(t, tt.DB, "event_logs", test.Data{
		"user_id":   user.ID,
		"type":      ds.EventLogUserAccountCreated,
		"is_public": false,
	})

	t.Run("username already taken", func(t *testing.T) {
		req := request.UserSignUp{
			Username: user.Username,
			Email:    random.Email(),
			Password: random.String(),
		}

		var resp handler.Error
		Request(t, RequestArgs{
			method:       http.MethodPost,
			path:         "/users/sign-up",
			body:         req,
			bindResponse: &resp,
			assertStatus: http.StatusUnprocessableEntity,
		})

		assert.Equal(t, service.UsernameAlreadyTaken, resp.InputErrors["username"])
	})

	t.Run("email already taken", func(t *testing.T) {
		req := request.UserSignUp{
			Username: random.String(),
			Email:    user.Email,
			Password: random.String(),
		}

		var resp handler.Error
		Request(t, RequestArgs{
			method:       http.MethodPost,
			path:         "/users/sign-up",
			body:         req,
			bindResponse: &resp,
			assertStatus: http.StatusUnprocessableEntity,
		})

		assert.Equal(t, service.UserWithThisEmailAlreadyExists, resp.InputErrors["email"])
	})
}

func TestUserSignIn(t *testing.T) {
	password := random.String()
	user := create(t, ds.User{
		Password: password,
	})

	req := request.UserSignIn{
		Email:    user.Email,
		Password: password,
	}

	var resp response.UserSignIn
	Request(t, RequestArgs{
		method:       http.MethodPost,
		path:         "/users/sign-in",
		body:         req,
		bindResponse: &resp,
		assertStatus: http.StatusOK,
	})
}

func TestChangePassword(t *testing.T) {
	oldPassword := random.String(10)
	newPassword := random.String(10)

	user := create(t, ds.User{Password: oldPassword, EmailConfirmed: true})

	_, token, err := tt.Service.AuthenticateUser(context.Background(), user.Email, oldPassword)
	if err != nil {
		t.Fatal(err)
	}

	req := request.ChangePassword{
		OldPassword: oldPassword,
		NewPassword: newPassword,
	}

	var resp response.Status
	Request(t, RequestArgs{
		method:       http.MethodPut,
		path:         "/users/password/",
		body:         req,
		authToken:    token,
		bindResponse: &resp,
		assertStatus: http.StatusOK,
	})

	// Login with the old password
	var signInResp handler.Error
	Request(t, RequestArgs{
		method: http.MethodPost,
		path:   "/users/sign-in/",
		body: request.UserSignIn{
			Email:    user.Email,
			Password: oldPassword,
		},
		bindResponse: &signInResp,
		assertStatus: http.StatusUnprocessableEntity,
	})

	test.AssertInDB(t, tt.DB, "event_logs", test.Data{
		"user_id":   user.ID,
		"type":      ds.EventLogUserPasswordChanged,
		"is_public": false,
	})

	t.Run("login with new password", func(t *testing.T) {
		var signInResp response.UserSignIn
		Request(t, RequestArgs{
			method: http.MethodPost,
			path:   "/users/sign-in/",
			body: request.UserSignIn{
				Email:    user.Email,
				Password: newPassword,
			},
			bindResponse: &signInResp,
			assertStatus: http.StatusOK,
		})
	})

	t.Run("incorrect old password", func(t *testing.T) {
		req := request.ChangePassword{
			OldPassword: "incorrect-password",
			NewPassword: newPassword,
		}

		var resp handler.Error
		Request(t, RequestArgs{
			method:       http.MethodPut,
			path:         "/users/password/",
			body:         req,
			authToken:    token,
			bindResponse: &resp,
			assertStatus: http.StatusUnprocessableEntity,
		})

		assert.Equal(t, resp.InputErrors["old_password"], service.ErrInvalidPassword.Error())
	})
}

func TestPasswordReset(t *testing.T) {
	user := create[ds.User](t)

	// 1. RequestArgs password reset
	var reqResetResp response.Status
	Request(t, RequestArgs{
		method:       http.MethodPost,
		path:         "users/password-reset-request",
		body:         request.PasswordResetRequest{Email: user.Email},
		bindResponse: &reqResetResp,
		assertStatus: http.StatusOK,
	})

	test.AssertInDB(t, tt.DB, "password_reset_tokens", test.Data{"user_id": user.ID})

	test.AssertInDB(t, tt.DB, "event_logs", test.Data{
		"user_id":   user.ID,
		"type":      ds.EventLogUserRequestPasswordReset,
		"is_public": false,
	})

	emailVars := test.LoadEmailVars(t, user.Email)
	token := app.String(emailVars["token"])
	assert.NotEmpty(t, token)

	// 2. Successfully reset the password
	newPassword := random.String()
	var resetResp response.Status
	Request(t, RequestArgs{
		method: http.MethodPost,
		path:   "users/password-reset",
		body: request.PasswordReset{
			Token:    token,
			Password: newPassword,
		},
		bindResponse: &resetResp,
		assertStatus: http.StatusOK,
	})

	// Assert the authToken was deleted
	test.AssertNotInDB(t, tt.DB, "password_reset_tokens", test.Data{"token": token})

	test.AssertInDB(t, tt.DB, "event_logs", test.Data{
		"user_id":   user.ID,
		"type":      ds.EventLogUserPasswordChangedByResetRequest,
		"is_public": false,
	})

	// 3. Verify login with the new password
	var signInResp response.UserSignIn
	Request(t, RequestArgs{
		method: http.MethodPost,
		path:   "users/sign-in",
		body: request.UserSignIn{
			Email:    user.Email,
			Password: newPassword,
		},
		bindResponse: &signInResp,
		assertStatus: http.StatusOK,
	})

	// 4. Test failure cases
	t.Run("reset with invalid token", func(t *testing.T) {
		var errorResp handler.Error
		Request(t, RequestArgs{
			method: http.MethodPost,
			path:   "users/password-reset",
			body: request.PasswordReset{
				Token:    "invalid-token",
				Password: newPassword,
			},
			bindResponse: &errorResp,
			assertStatus: http.StatusUnprocessableEntity,
		})
	})

	t.Run("reset with password too short", func(t *testing.T) {
		prt := create(t, ds.PasswordResetToken{
			UserID: user.ID,
		})

		var errorResp handler.Error
		Request(t, RequestArgs{
			method: http.MethodPost,
			path:   "users/password-reset",
			body: request.PasswordReset{
				Token:    prt.Token,
				Password: strings.Repeat("a", service.UserPasswordMinLen-1),
			},
			bindResponse: &errorResp,
			assertStatus: http.StatusUnprocessableEntity,
		})
		assert.NotEmpty(t, errorResp.InputErrors["password"])
	})
}

func TestChangeEmail(t *testing.T) {
	user := create[ds.User](t, ds.User{EmailConfirmed: true})
	token := loginAs(t, user)

	newEmail := random.Email()
	var reqEmailChangeResp response.Status
	Request(t, RequestArgs{
		method:       http.MethodPost,
		path:         "/users/email/",
		body:         request.EmailChangeRequest{Email: newEmail},
		authToken:    token,
		bindResponse: &reqEmailChangeResp,
		assertStatus: http.StatusOK,
	})

	test.AssertInDB(t, tt.DB, "change_email_requests", test.Data{
		"user_id":   user.ID,
		"new_email": newEmail,
	})

	test.AssertInDB(t, tt.DB, "event_logs", test.Data{
		"user_id":   user.ID,
		"type":      ds.EventLogUserEmailChangeRequested,
		"is_public": false,
	})

	emailVars := test.LoadEmailVars(t, newEmail)
	confirmToken := app.String(emailVars["token"])
	assert.NotEmpty(t, confirmToken)

	// Confirm the email change with the confirmToken
	var confirmResp response.Status
	Request(t, RequestArgs{
		method: http.MethodPut,
		path:   "/users/email/",
		body: request.EmailChangeConfirm{
			Token: confirmToken,
		},
		authToken:    token,
		bindResponse: &confirmResp,
		assertStatus: http.StatusOK,
	})

	test.AssertInDB(t, tt.DB, "users", test.Data{
		"id":    user.ID,
		"email": newEmail,
	})

	test.AssertNotInDB(t, tt.DB, "change_email_requests", test.Data{
		"token": confirmToken,
	})

	test.AssertInDB(t, tt.DB, "event_logs", test.Data{
		"user_id":   user.ID,
		"type":      ds.EventLogUserEmailChanged,
		"meta":      map[string]any{"new_email": newEmail, "old_email": user.Email},
		"is_public": false,
	})

	// Test failure case: using the same authToken again
	var errorResp handler.Error
	Request(t, RequestArgs{
		method: http.MethodPut,
		path:   "/users/email/",
		body: request.EmailChangeConfirm{
			Token: confirmToken,
		},
		authToken:    token,
		bindResponse: &errorResp,
		assertStatus: http.StatusUnprocessableEntity,
	})

	assert.Equal(t, service.ErrInvalidChangeEmailToken.Error(), errorResp.Error)
}

func TestChangeUsername(t *testing.T) {
	password := random.String(10)
	user := create(t, ds.User{Password: password, EmailConfirmed: true})

	token := loginAs(t, user)
	newUsername := random.String(10)

	t.Run("successful username change", func(t *testing.T) {
		req := request.ChangeUsername{
			Username: newUsername,
			Password: password,
		}

		var resp response.Status
		Request(t, RequestArgs{
			method:       http.MethodPut,
			path:         "/users/username/",
			body:         req,
			authToken:    token,
			bindResponse: &resp,
			assertStatus: http.StatusOK,
		})

		test.AssertInDB(t, tt.DB, "users", test.Data{
			"id":       user.ID,
			"username": newUsername,
		})

		test.AssertInDB(t, tt.DB, "event_logs", test.Data{
			"user_id":   user.ID,
			"type":      ds.EventLogUserUsernameChanged,
			"meta":      map[string]any{"old_username": user.Username, "new_username": newUsername},
			"is_public": false,
		})
	})

	t.Run("incorrect password", func(t *testing.T) {
		req := request.ChangeUsername{
			Username: random.String(10),
			Password: "wrong-password",
		}

		var resp handler.Error
		Request(t, RequestArgs{
			method:       http.MethodPut,
			path:         "/users/username/",
			body:         req,
			authToken:    token,
			bindResponse: &resp,
			assertStatus: http.StatusUnprocessableEntity,
		})

		assert.Equal(t, "Incorrect password", resp.InputErrors["password"])
	})

	t.Run("username already taken", func(t *testing.T) {
		otherUser := create[ds.User](t)
		req := request.ChangeUsername{
			Username: otherUser.Username,
			Password: password,
		}

		var resp handler.Error
		Request(t, RequestArgs{
			method:       http.MethodPut,
			path:         "/users/username/",
			body:         req,
			authToken:    token,
			bindResponse: &resp,
			assertStatus: http.StatusUnprocessableEntity,
		})

		assert.Equal(t, service.UsernameAlreadyTaken, resp.InputErrors["username"])
	})
}

func TestDeleteUser(t *testing.T) {
	password := random.String(10)
	user := create(t, ds.User{Password: password, EmailConfirmed: true})

	token := loginAs(t, user)

	req := request.DeleteUser{
		Password: password,
	}

	var resp response.Status
	Request(t, RequestArgs{
		method:       http.MethodDelete,
		path:         "/users/",
		body:         req,
		authToken:    token,
		bindResponse: &resp,
		assertStatus: http.StatusOK,
	})

	test.AssertInDB(t, tt.DB, "users", test.Data{
		"id":         user.ID,
		"deleted_at": test.NotNull,
	})

	test.AssertNotInDB(t, tt.DB, "user_sessions", test.Data{
		"user_id": user.ID,
	})

	// try to login
	var signInResp handler.Error
	Request(t, RequestArgs{
		method: http.MethodPost,
		path:   "/users/sign-in/",
		body: request.UserSignIn{
			Email:    user.Email,
			Password: password,
		},
		bindResponse: &signInResp,
		assertStatus: http.StatusUnprocessableEntity,
	})
}
