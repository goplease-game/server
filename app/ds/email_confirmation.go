package ds

import (
	"time"
)

// EmailConfirmation represents a record used to verify a user's email address
// after registration or a change request.
type EmailConfirmation struct {
	ID          ID
	UserID      ID
	Code        string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	ConfirmedAt *time.Time
}

// Invalid checks if the confirmation record is expired.
// Returns true if the token is no longer valid, false otherwise.
func (c *EmailConfirmation) Invalid() bool {
	return c.ExpiresAt.Before(time.Now())
}
