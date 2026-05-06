package repo

import (
	"context"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/ognev-dev/goplease/app/ds"
)

// CreatePasswordResetToken inserts a new password reset token into the database.
func (r *Repo) CreatePasswordResetToken(ctx context.Context, t *ds.PasswordResetToken) error {
	_, span := r.tracer.Start(ctx, "CreatePasswordResetToken")
	defer span.End()

	if t.ID.IsNil() {
		t.ID = ds.NewID()
	}

	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}

	return r.insert(ctx, "password_reset_tokens", data{
		"id":         t.ID,
		"user_id":    t.UserID,
		"token":      t.Token,
		"expires_at": t.ExpiresAt,
		"created_at": t.CreatedAt,
	})
}

// GetPasswordResetToken retrieves a password reset token from the database by the token string.
// If the token is not found, it returns ErrPasswordResetTokenNotFound.
func (r *Repo) GetPasswordResetToken(ctx context.Context, token string) (*ds.PasswordResetToken, error) {
	_, span := r.tracer.Start(ctx, "GetPasswordResetToken")
	defer span.End()

	t := new(ds.PasswordResetToken)
	err := pgxscan.Get(ctx, r.getDB(ctx), t, `SELECT * FROM password_reset_tokens WHERE token = $1`, token)
	if noRows(err) {
		return nil, ErrPasswordResetTokenNotFound
	}
	return t, err
}

// DeletePasswordResetToken removes a password reset token from the database by its ID.
func (r *Repo) DeletePasswordResetToken(ctx context.Context, id ds.ID) error {
	_, span := r.tracer.Start(ctx, "DeletePasswordResetToken")
	defer span.End()

	return r.hardDelete(ctx, "password_reset_tokens", id)
}

// DeletePasswordResetTokensByUser removes a password reset tokens that belongs to specific user.
func (r *Repo) DeletePasswordResetTokensByUser(ctx context.Context, userID ds.ID) error {
	_, span := r.tracer.Start(ctx, "DeletePasswordResetTokensByUser")
	defer span.End()

	return r.exec(ctx, `DELETE FROM password_reset_tokens WHERE user_id = $1`, userID)
}
