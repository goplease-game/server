package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/app/service"
	"github.com/ognev-dev/goplease/server/request"
	"github.com/ognev-dev/goplease/server/response"
)

// UserSignUp is the API handler for user registration.
//
//	@ID			UserSignUp
//	@Summary	User registration
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		request	body		request.UserSignUp	true	"Request body"
//	@Success	200		{object}	response.UserSignIn
//	@Failure	422		{object}	Error
//	@Failure	500		{object}	Error
//	@Router		/users/sign-up/ [post]
//	@Security	ApiKeyAuth
func (h *Handler) UserSignUp(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "UserSignUp")
	defer span.End()

	var req request.UserSignUp

	res := handleJSON(w, r, &req)
	if res.Aborted() {
		return
	}

	_, err := h.service.RegisterUser(ctx, req.Username, req.Email, req.Password)
	if err != nil {
		res.Abort(err)
		return
	}

	user, token, err := h.service.AuthenticateUser(ctx, req.Email, req.Password)
	if err != nil {
		res.Abort(err)
		return
	}

	setSessionCookie(w, token)

	res.jsonOK(response.UserSignIn{
		ID:       user.ID,
		Username: user.Username,
		Token:    token,
	})
}

// UserSignIn is the API handler for the user login endpoint.
//
//	@ID			UserSignIn
//	@Summary	User auth
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		request	body		request.UserSignIn	true	"Request body"
//	@Success	200		{object}	response.UserSignIn
//	@Failure	422		{object}	Error
//	@Failure	500		{object}	Error
//	@Router		/users/sign-in/ [post]
//	@Security	ApiKeyAuth
//
// TODO either email or username can be used to login.
func (h *Handler) UserSignIn(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "UserSignIn")
	defer span.End()

	var req request.UserSignIn

	res := handleJSON(w, r, &req)
	if res.Aborted() {
		return
	}

	user, token, err := h.service.AuthenticateUser(ctx, req.Email, req.Password)
	if err != nil {
		res.Abort(err)
		return
	}

	setSessionCookie(w, token)

	res.jsonOK(response.UserSignIn{
		ID:       user.ID,
		Username: user.Username,
		Token:    token,
	})
}

// ConfirmEmail is the API handler for confirming a user's email address via a confirmation code.
//
//	@ID			ConfirmEmail
//	@Summary	Confirm email
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		request	body		request.ConfirmEmail	true	"Request body"
//	@Success	200		{object}	response.Status
//	@Failure	422		{object}	Error
//	@Failure	500		{object}	Error
//	@Router		/users/confirm-email/ [post]
//	@Security	ApiKeyAuth
func (h *Handler) ConfirmEmail(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "ConfirmEmail")
	defer span.End()

	var req request.ConfirmEmail
	res := handleJSON(w, r, &req)
	if res.Aborted() {
		return
	}

	err := h.service.ConfirmEmail(ctx, req.Code)
	if err != nil {
		res.Abort(err)
		return
	}

	res.jsonSuccess()
}

// SendEmailConfirmationCode sends a confirmation code to the authenticated user's email.
//
//	@ID			SendEmailConfirmationCode
//	@Summary	SendE email confirmation code
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Success	200		{object}	response.Status
//	@Failure	422		{object}	Error
//	@Failure	500		{object}	Error
//	@Router		/users/confirm-email/ [post]
//	@Security	ApiKeyAuth
func (h *Handler) SendEmailConfirmationCode(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "SendEmailConfirmationCode")
	defer span.End()

	retryAfter, err := h.service.ResendConfirmationEmailCode(ctx)
	if errors.Is(err, service.ErrResendConfirmationEmailCodeTooManyRequest) {
		w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
	}
	if err != nil {
		Abort(w, r, err)
		return
	}

	jsonSuccess(w)
}

// UserSignOut handles user log-out by clearing the session cookie and deleting the session
// record from the database.
//
//	@ID			UserSignOut
//	@Summary	Logout
//	@Tags		users
//	@Produce	json
//	@Success	200		{object}	response.Status
//	@Failure	422		{object}	Error
//	@Failure	500		{object}	Error
//	@Router		/users/sign-out/ [post]
//	@Security	ApiKeyAuth
func (h *Handler) UserSignOut(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "UserSignOut")
	defer span.End()

	clearSessionCookie(w)

	session := ds.UserSessionFromContext(ctx)
	if session != nil {
		err := h.service.DeleteUserSession(ctx, session.ID)
		if err != nil {
			log.Println("delete user session: " + err.Error()) //nolint:gosec
		}
	}

	jsonOK(w, response.Success)
	return
}

// ChangePassword handles the API request for an authenticated user to change their password.
//
//	@ID			ChangePassword
//	@Summary	Change user password
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		request	body		request.ChangePassword	true	"Old and new passwords"
//	@Success	200		{object}	response.Status
//	@Failure	401		{object}	Error "Unauthorized"
//	@Failure	422		{object}	Error "Validation error or incorrect old password"
//	@Failure	500		{object}	Error
//	@Router		/users/password/ [post]
//	@Security	ApiKeyAuth
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "ChangePassword")
	defer span.End()

	var req request.ChangePassword
	user, res := handleAuthorizedJSON(w, r, &req)
	if res.Aborted() {
		return
	}

	err := h.service.ChangeUserPassword(ctx, user.ID, req.OldPassword, req.NewPassword)
	if err != nil {
		res.Abort(err)
		return
	}

	res.jsonSuccess()
}

// RequestEmailChange handles the request for an email change.
//
//	@ID			EmailChangeRequest
//	@Summary	Request to change user email
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		request	body		request.EmailChangeRequest	true	"Old and new passwords"
//	@Success	200		{object}	response.Status
//	@Failure	401		{object}	Error "Unauthorized"
//	@Failure	422		{object}	Error "Validation error"
//	@Failure	500		{object}	Error
//	@Router		/users/email/ [post]
//	@Security	ApiKeyAuth
func (h *Handler) RequestEmailChange(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "EmailChangeRequest")
	defer span.End()

	var req request.EmailChangeRequest
	user, res := handleAuthorizedJSON(w, r, &req)
	if res.Aborted() {
		return
	}

	err := h.service.CreateChangeEmailRequest(ctx, user.ID, req.Email)
	if err != nil {
		res.Abort(err)
		return
	}

	res.jsonSuccess()
}

// ConfirmEmailChange handles the confirmation for an email change.
//
//	@ID			EmailChangeConfirm
//	@Summary	Confirm changing user email
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		request	body		request.EmailChangeRequest	true	"Old and new passwords"
//	@Success	200		{object}	response.Status
//	@Failure	401		{object}	Error "Unauthorized"
//	@Failure	422		{object}	Error "Validation error"
//	@Failure	500		{object}	Error
//	@Router		/users/email/ [post]
//	@Security	ApiKeyAuth
func (h *Handler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "EmailChangeConfirm")
	defer span.End()

	var req request.EmailChangeConfirm
	res := handleJSON(w, r, &req)
	if res.Aborted() {
		return
	}

	err := h.service.ConfirmEmailChange(ctx, req.Token)
	if err != nil {
		res.Abort(err)
		return
	}

	res.jsonSuccess()
}

// ChangeUsername handles the API request for an authenticated user to change their username.
//
//	@ID			ChangeUsername
//	@Summary	Change username
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		request	body		request.ChangeUsername	true	"New username and password"
//	@Success	200		{object}	response.Status
//	@Failure	401		{object}	Error "Unauthorized"
//	@Failure	422		{object}	Error "Validation error or incorrect password"
//	@Failure	500		{object}	Error
//	@Router		/users/username/ [post]
//	@Security	ApiKeyAuth
func (h *Handler) ChangeUsername(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "ChangeUsername")
	defer span.End()

	var req request.ChangeUsername
	user, res := handleAuthorizedJSON(w, r, &req)
	if res.Aborted() {
		return
	}

	err := h.service.ChangeUsername(ctx, service.ChangeUsernameInput{
		UserID:      user.ID,
		NewUsername: req.Username,
		Password:    req.Password,
	})
	if err != nil {
		res.Abort(err)
		return
	}

	res.jsonSuccess()
}

// DeleteUser handles the API request for an authenticated user to delete their account.
//
//	@ID			DeleteUser
//	@Summary	Delete user account
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		request	body		request.DeleteUser	true	"Password"
//	@Success	200		{object}	response.Status
//	@Failure	401		{object}	Error "Unauthorized"
//	@Failure	422		{object}	Error "Validation error or incorrect password"
//	@Failure	500		{object}	Error
//	@Router		/users/ [delete]
//	@Security	ApiKeyAuth
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "DeleteUser")
	defer span.End()

	var req request.DeleteUser
	user, res := handleAuthorizedJSON(w, r, &req)
	if res.Aborted() {
		return
	}

	err := h.service.DeleteUser(ctx, user.ID, req.Password)
	if err != nil {
		res.Abort(err)
		return
	}

	clearSessionCookie(w)
	res.jsonSuccess()
}
