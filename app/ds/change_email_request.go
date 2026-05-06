package ds

import (
	"time"
)

// ChangeEmailRequest represents a record in the change_email_requests table.
// It stores a single-use token for a user to confirm a change to their email address.
type ChangeEmailRequest struct {
	ID        ID        `json:"-"`
	UserID    ID        `json:"-"`
	NewEmail  string    `json:"-"`
	Token     string    `json:"-"`
	ExpiresAt time.Time `json:"-"`
	CreatedAt time.Time `json:"-"`
}

// Invalid returns true if token has expired.
func (r *ChangeEmailRequest) Invalid() bool {
	return r.ExpiresAt.Before(time.Now())
}
