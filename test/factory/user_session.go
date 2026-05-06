package factory

import (
	"context"
	"time"

	"github.com/ognev-dev/goplease/app/ds"
)

// NewUserSession ...
func (f *Factory) NewUserSession(overrideOpt ...ds.UserSession) (m *ds.UserSession) {
	m = &ds.UserSession{
		ID:        ds.NilID,
		UserID:    ds.NilID,
		CreatedAt: time.Now(),
		UpdatedAt: nil,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	if len(overrideOpt) == 1 {
		merge(m, overrideOpt[0])
	}

	return
}

// CreateUserSession ...
func (f *Factory) CreateUserSession(overrideOpt ...ds.UserSession) (m *ds.UserSession, err error) {
	m = f.NewUserSession(overrideOpt...)
	if m.UserID.IsNil() {
		u, err := f.CreateUser()
		if err != nil {
			return nil, err
		}

		m.UserID = u.ID
	}

	err = f.repo.CreateUserSession(context.Background(), m)
	return
}
