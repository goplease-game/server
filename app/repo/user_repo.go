package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/ognev-dev/goplease/app/ds"
)

var (
	// ErrUserNotFound is a sentinel error returned when a user lookup fails.
	ErrUserNotFound = errors.New("user not found")

	// ErrPasswordResetTokenNotFound is a sentinel error returned when a password reset token is not found.
	ErrPasswordResetTokenNotFound = errors.New("password reset token not found")

	// ErrChangeEmailRequestNotFound ...
	ErrChangeEmailRequestNotFound = errors.New("change email request not found")
)

// GetUserByEmail retrieves a user from the database by their email address.
func (r *Repo) GetUserByEmail(ctx context.Context, email string) (*ds.User, error) {
	_, span := r.tracer.Start(ctx, "GetUserByEmail")
	defer span.End()

	user := new(ds.User)
	err := pgxscan.Get(ctx, r.getDB(ctx), user, `SELECT * FROM users WHERE email = $1`, email)
	if noRows(err) {
		return nil, ErrUserNotFound
	}

	return user, err
}

// GetUserByUsername retrieves a user from the database by their username.
func (r *Repo) GetUserByUsername(ctx context.Context, username string) (*ds.User, error) {
	_, span := r.tracer.Start(ctx, "GetUserByUsername")
	defer span.End()

	user := new(ds.User)
	err := pgxscan.Get(ctx, r.getDB(ctx), user, `SELECT * FROM users WHERE username = $1`, username)
	if noRows(err) {
		return nil, ErrUserNotFound
	}

	return user, err
}

// GetUserByID retrieves a user from the database by their ID.
func (r *Repo) GetUserByID(ctx context.Context, id ds.ID) (*ds.User, error) {
	_, span := r.tracer.Start(ctx, "GetUserByID")
	defer span.End()

	user := new(ds.User)
	err := pgxscan.Get(ctx, r.getDB(ctx), user, `SELECT * FROM users WHERE id = $1`, id)
	if noRows(err) {
		user = nil
		err = ErrUserNotFound
	}

	return user, err
}

// CreateUser inserts a new user record into the database.
func (r *Repo) CreateUser(ctx context.Context, u *ds.User) (err error) {
	_, span := r.tracer.Start(ctx, "CreateUser")
	defer span.End()

	if u.ID.IsNil() {
		u.ID = ds.NewID()
	}

	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now()
	}

	err = r.insert(ctx, "users", data{
		"id":              u.ID,
		"username":        u.Username,
		"email":           u.Email,
		"email_confirmed": u.EmailConfirmed,
		"password":        u.Password,
		"created_at":      u.CreatedAt,
		"updated_at":      u.UpdatedAt,
		"deleted_at":      u.DeletedAt,
	})

	return
}

// SetUserEmailConfirmed updates a user's record to set the email_confirmed flag to true
// and updates the updated_at timestamp.
func (r *Repo) SetUserEmailConfirmed(ctx context.Context, userID ds.ID) (err error) {
	_, span := r.tracer.Start(ctx, "SetUserEmailConfirmed")
	defer span.End()

	return r.exec(ctx, "UPDATE users SET email_confirmed = true, updated_at = NOW() WHERE id = $1", userID)
}

// UpdateUserPassword updates the password hash for a specific user.
func (r *Repo) UpdateUserPassword(ctx context.Context, userID ds.ID, password string) (err error) {
	_, span := r.tracer.Start(ctx, "UpdateUserPassword")
	defer span.End()

	return r.exec(ctx, "UPDATE users SET password = $1, updated_at = NOW() WHERE id = $2", password, userID)
}

// UpdateUserEmail updates the email for a specific user.
func (r *Repo) UpdateUserEmail(ctx context.Context, userID ds.ID, email string) (err error) {
	_, span := r.tracer.Start(ctx, "UpdateUserEmail")
	defer span.End()

	return r.exec(ctx, "UPDATE users SET email = $1, updated_at = NOW() WHERE id = $2", email, userID)
}

// UpdateUsername updates the username for a specific user.
func (r *Repo) UpdateUsername(ctx context.Context, userID ds.ID, username string) (err error) {
	_, span := r.tracer.Start(ctx, "UpdateUsername")
	defer span.End()

	return r.exec(ctx, "UPDATE users SET username = $1, updated_at = NOW() WHERE id = $2", username, userID)
}

// DeleteUser performs a soft delete on a user record by setting the deleted_at timestamp.
func (r *Repo) DeleteUser(ctx context.Context, userID ds.ID) (err error) {
	_, span := r.tracer.Start(ctx, "DeleteUser")
	defer span.End()

	return r.delete(ctx, "users", userID)
}

// FilterUsers retrieves a paginated, filtered list of users from the database.
func (r *Repo) FilterUsers(ctx context.Context, f ds.UsersFilter) (users []ds.User, count int, err error) {
	_, span := r.tracer.Start(ctx, "FilterUsers")
	defer span.End()

	count, err = r.filter("users").
		paginate(f.Page, f.PerPage).
		createdAt(f.CreatedAt).
		deletedAt(f.DeletedAt).
		deleted(f.Deleted).
		whereIf(f.NotCleaned, "cleaned_at IS NULL", nil).
		order(f.OrderBy, f.OrderDirection).
		withCount(f.WithCount).
		scan(ctx, &users)

	return
}

// UpdateUser updates user fields in the database.
func (r *Repo) UpdateUser(ctx context.Context, u *ds.User) error {
	_, span := r.tracer.Start(ctx, "UpdateUser")
	defer span.End()

	err := r.update(ctx, u.ID, "users", data{
		"email":      u.Email,
		"username":   u.Username,
		"password":   u.Password,
		"cleaned_at": u.CleanedAt,
	})
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}
