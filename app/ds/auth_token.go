package ds

import (
	"time"
)

// AuthToken represents container for authentication-related
// data.
type AuthToken struct {
	ID         ID
	UserID     int64
	User       *User
	ClientName string
	ClientIP   string
	UserAgent  string
	Token      string
	CreatedAt  time.Time
	ExpiresAt  time.Time
}
