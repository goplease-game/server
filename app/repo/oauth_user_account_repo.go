package repo

import (
	"context"
	"errors"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/oauth/provider"
)

var (
	// ErrOAuthUserAccountNotFound is a sentinel error returned when a account lookup fails.
	ErrOAuthUserAccountNotFound = errors.New("oauth user account not found")
)

// CreateOAuthUserAccount inserts a new user record into the database.
func (r *Repo) CreateOAuthUserAccount(ctx context.Context, acc *ds.OAuthUserAccount) (err error) {
	_, span := r.tracer.Start(ctx, "CreateUser")
	defer span.End()

	if acc.ID.IsNil() {
		acc.ID = ds.NewID()
	}

	if acc.CreatedAt.IsZero() {
		acc.CreatedAt = time.Now()
	}

	return r.insert(ctx, "oauth_user_accounts", data{
		"id":               acc.ID,
		"user_id":          acc.UserID,
		"provider":         acc.Provider,
		"provider_user_id": acc.ProviderUserID,
		"created_at":       acc.CreatedAt,
	})
}

// GetOAuthUserAccount ...
func (r *Repo) GetOAuthUserAccount(
	ctx context.Context, prov provider.Type, provUserID string) (*ds.OAuthUserAccount, error) {
	_, span := r.tracer.Start(ctx, "FindOAuthUserAccount")
	defer span.End()

	account := new(ds.OAuthUserAccount)
	err := pgxscan.Get(ctx, r.getDB(ctx), account,
		`SELECT * FROM oauth_user_accounts WHERE provider = $1 AND provider_user_id = $2`, prov, provUserID)
	if noRows(err) {
		return nil, ErrOAuthUserAccountNotFound
	}

	return account, err
}
