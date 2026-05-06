package ds

import (
	"context"
	"time"
)

const (
	userSessionCtxKey ctxKey = "user_session"
)

// UserSession represents an active session for a logged-in user.
type UserSession struct {
	ID        ID         `json:"id"`
	UserID    ID         `json:"user_id"`
	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
	ExpiresAt time.Time  `json:"-"`
}

// ToContext adds the given user session object to the provided context.
func (s *UserSession) ToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, userSessionCtxKey, s)
}

// UserSessionFromContext attempts to retrieve the user session object from the context.
func UserSessionFromContext(ctx context.Context) *UserSession {
	if v := ctx.Value(userSessionCtxKey); v != nil {
		if session, ok := v.(*UserSession); ok {
			return session
		}
	}

	return nil
}
