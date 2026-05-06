package factory

import (
	"context"
	"time"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/test/factory/random"
)

// NewPasswordResetToken ...
func (f *Factory) NewPasswordResetToken(overrideOpt ...ds.PasswordResetToken) (m *ds.PasswordResetToken) {
	m = &ds.PasswordResetToken{
		ID:        ds.NilID,
		UserID:    ds.NilID,
		Token:     random.String(),
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}

	if len(overrideOpt) == 1 {
		merge(m, overrideOpt[0])
	}

	return
}

// CreatePasswordResetToken ...
func (f *Factory) CreatePasswordResetToken(overrideOpt ...ds.PasswordResetToken) (
	m *ds.PasswordResetToken, err error) {
	m = f.NewPasswordResetToken(overrideOpt...)

	if m.UserID.IsNil() {
		u, err := f.CreateUser()
		if err != nil {
			return nil, err
		}

		m.UserID = u.ID
	}

	err = f.repo.CreatePasswordResetToken(context.Background(), m)
	return
}
