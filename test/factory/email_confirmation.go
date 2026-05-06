package factory

import (
	"context"
	"time"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/test/factory/random"
)

// NewEmailConfirmation ...
func (f *Factory) NewEmailConfirmation(overrideOpt ...ds.EmailConfirmation) (m *ds.EmailConfirmation) {
	m = &ds.EmailConfirmation{
		ID:          ds.NilID,
		UserID:      ds.NilID,
		Code:        random.String(16), //nolint:mnd
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(time.Hour),
		ConfirmedAt: nil,
	}

	if len(overrideOpt) == 1 {
		merge(m, overrideOpt[0])
	}

	return
}

// CreateEmailConfirmation ...
func (f *Factory) CreateEmailConfirmation(overrideOpt ...ds.EmailConfirmation) (m *ds.EmailConfirmation, err error) {
	m = f.NewEmailConfirmation(overrideOpt...)

	if m.UserID.IsNil() {
		u, err := f.CreateUser()
		if err != nil {
			return nil, err
		}

		m.UserID = u.ID
	}

	err = f.repo.CreateEmailConfirmation(context.Background(), m)
	return
}
