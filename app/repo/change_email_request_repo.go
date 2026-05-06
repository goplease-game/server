package repo

import (
	"context"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/ognev-dev/goplease/app/ds"
)

// CreateChangeEmailRequest inserts a new change email request into the database.
func (r *Repo) CreateChangeEmailRequest(ctx context.Context, req *ds.ChangeEmailRequest) error {
	_, span := r.tracer.Start(ctx, "CreateChangeEmailRequest")
	defer span.End()

	if req.ID.IsNil() {
		req.ID = ds.NewID()
	}

	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now()
	}

	return r.insert(ctx, "change_email_requests", data{
		"id":         req.ID,
		"user_id":    req.UserID,
		"new_email":  req.NewEmail,
		"token":      req.Token,
		"expires_at": req.ExpiresAt,
		"created_at": req.CreatedAt,
	})
}

// GetChangeEmailRequestByToken retrieves a change email request from the database by its token.
// If the token is not found, it returns ErrChangeEmailRequestNotFound.
func (r *Repo) GetChangeEmailRequestByToken(ctx context.Context, token string) (*ds.ChangeEmailRequest, error) {
	_, span := r.tracer.Start(ctx, "GetChangeEmailRequestByToken")
	defer span.End()

	req := new(ds.ChangeEmailRequest)
	err := pgxscan.Get(ctx, r.getDB(ctx), req, `SELECT * FROM change_email_requests WHERE token = $1`, token)
	if noRows(err) {
		return nil, ErrChangeEmailRequestNotFound
	}
	return req, err
}

// DeleteChangeEmailRequest removes a change email request from the database by its ID.
func (r *Repo) DeleteChangeEmailRequest(ctx context.Context, id ds.ID) error {
	_, span := r.tracer.Start(ctx, "DeleteChangeEmailRequest")
	defer span.End()

	return r.hardDelete(ctx, "change_email_requests", id)
}

// DeleteChangeEmailRequestsByUser removes a change email request for specific user.
func (r *Repo) DeleteChangeEmailRequestsByUser(ctx context.Context, userID ds.ID) error {
	_, span := r.tracer.Start(ctx, "DeleteChangeEmailRequest")
	defer span.End()

	return r.exec(ctx, `DELETE FROM change_email_requests WHERE user_id = $1`, userID)
}
