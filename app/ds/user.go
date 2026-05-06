package ds

import (
	"context"
	"encoding/json"
	"time"
)

// DeletedUsername is the placeholder username for soft-deleted users.
const DeletedUsername = "deleted"

const (
	userCtxKey ctxKey = "user"

	// CleanupDeletedUserAfter ...
	CleanupDeletedUserAfter = 30 * 24 * time.Hour
)

// User represents a user account in the system.
type User struct {
	ID             ID         `json:"id"`
	Username       string     `json:"username"`
	Email          string     `json:"email"`
	EmailConfirmed bool       `json:"-"`
	Password       string     `json:"-"`
	CreatedAt      time.Time  `json:"-"`
	UpdatedAt      *time.Time `json:"-"`
	DeletedAt      *time.Time `json:"-"`
	CleanedAt      *time.Time `json:"-"`

	// IsAdmin is true if the user ID is listed in "admins" key in config file.
	// This field is set by the auth middleware.
	// Until proper RBAC/ACL is implemented, we trust authority generously granted by the devs themselves.
	IsAdmin bool `json:"-"`
}

// MarshalJSON implements custom JSON serialization for User.
func (u *User) MarshalJSON() ([]byte, error) {
	type Alias User
	a := Alias(*u)

	if u.Deleted() {
		a.Username = DeletedUsername
	}

	return json.Marshal(&a)
}

// UsersFilter is used to filter and paginate user queries.
type UsersFilter struct {
	Page           int
	PerPage        int
	WithCount      bool
	CreatedAt      *FilterDT
	DeletedAt      *FilterDT
	Deleted        bool
	OrderBy        string
	OrderDirection string
	NotCleaned     bool
}

// Deleted reports whether the user has been soft-deleted.
func (u *User) Deleted() bool {
	return u.DeletedAt != nil
}

// ToContext adds the given user object to the provided context.
func (u *User) ToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, userCtxKey, u)
}

// UserFromContext attempts to retrieve user object from the context.
func UserFromContext(ctx context.Context) *User {
	if v := ctx.Value(userCtxKey); v != nil {
		if user, ok := v.(*User); ok {
			return user
		}
	}

	return nil
}
