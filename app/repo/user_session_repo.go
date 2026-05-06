package repo

import (
	"context"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/ds"
)

// ErrSessionNotFound ...
var ErrSessionNotFound = app.ErrUnauthorized()

// CreateUserSession inserts a new user session record into the database.
func (r *Repo) CreateUserSession(ctx context.Context, s *ds.UserSession) (err error) {
	_, span := r.tracer.Start(ctx, "CreateUserSession")
	defer span.End()

	if s.ID.IsNil() {
		s.ID = ds.NewID()
	}

	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}

	return r.insert(ctx, "user_sessions", data{
		"id":         s.ID,
		"user_id":    s.UserID,
		"created_at": s.CreatedAt,
		"expires_at": s.ExpiresAt,
	})
}

// GetUserSessionByID retrieves a user session record from the database using its unique ID.
func (r *Repo) GetUserSessionByID(ctx context.Context, id ds.ID) (sess *ds.UserSession, err error) {
	_, span := r.tracer.Start(ctx, "GetUserSessionByID")
	defer span.End()

	sess = new(ds.UserSession)
	err = pgxscan.Get(ctx, r.getDB(ctx), sess, `SELECT * FROM user_sessions WHERE id = $1`, id)
	if noRows(err) {
		return nil, ErrSessionNotFound
	}

	return
}

// ProlongUserSession updates the expiration timestamp of an existing user session.
func (r *Repo) ProlongUserSession(ctx context.Context, id ds.ID) (err error) {
	_, span := r.tracer.Start(ctx, "ProlongUserSession")
	defer span.End()

	expiresAt := time.Now().
		Add(time.Hour * time.Duration(app.Config().Session.DurationHours))

	return r.exec(ctx, `UPDATE user_sessions SET expires_at = $1 WHERE id = $2`,
		expiresAt, id,
	)
}

// DeleteUserSession removes a user session record from the database using its unique ID.
func (r *Repo) DeleteUserSession(ctx context.Context, id ds.ID) (err error) {
	_, span := r.tracer.Start(ctx, "DeleteUserSession")
	defer span.End()

	return r.hardDelete(ctx, "user_sessions", id)
}

// DeleteSessionsByUserID removes all session records for a specific user from the database.
func (r *Repo) DeleteSessionsByUserID(ctx context.Context, userID ds.ID) (err error) {
	_, span := r.tracer.Start(ctx, "DeleteSessionsByUserID")
	defer span.End()

	return r.exec(ctx, `DELETE FROM user_sessions WHERE user_id = $1`, userID)
}
