package factory

import (
	"context"
	"time"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/test/factory/random"
)

// NewChangeEmailRequest ...
func (f *Factory) NewChangeEmailRequest(overrideOpt ...ds.ChangeEmailRequest) (m *ds.ChangeEmailRequest) {
	m = &ds.ChangeEmailRequest{
		ID:        ds.NilID,
		UserID:    ds.NilID,
		NewEmail:  random.Email(),
		Token:     random.String(),
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}

	if len(overrideOpt) == 1 {
		merge(m, overrideOpt[0])
	}

	return
}

// CreateChangeEmailRequest ...
func (f *Factory) CreateChangeEmailRequest(overrideOpt ...ds.ChangeEmailRequest) (
	m *ds.ChangeEmailRequest, err error) {
	m = f.NewChangeEmailRequest(overrideOpt...)

	if m.UserID.IsNil() {
		u, err := f.CreateUser()
		if err != nil {
			return nil, err
		}

		m.UserID = u.ID
	}

	err = f.repo.CreateChangeEmailRequest(context.Background(), m)
	return
}
