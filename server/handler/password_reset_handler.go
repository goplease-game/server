package handler

import (
	"net/http"

	"github.com/ognev-dev/goplease/server/request"
)

// PasswordResetRequest handles the form submission for requesting a password reset.
func (h *Handler) PasswordResetRequest(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "PasswordResetRequest")
	defer span.End()

	var req request.PasswordResetRequest
	res := handleJSON(w, r, &req)
	if res.Aborted() {
		return
	}

	err := h.service.CreatePasswordResetRequest(ctx, req.Email)
	if err != nil {
		res.Abort(err)
		return
	}

	res.jsonSuccess()
}

// PasswordResetConfirm handles the form submission for resetting the password.
func (h *Handler) PasswordResetConfirm(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "PasswordResetConfirm")
	defer span.End()

	var req request.PasswordReset
	res := handleJSON(w, r, &req)
	if res.Aborted() {
		return
	}

	err := h.service.ResetPassword(ctx, req.Token, req.Password)
	if err != nil {
		res.Abort(err)
		return
	}

	res.jsonSuccess()
}
