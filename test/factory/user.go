package factory

import (
	"context"
	"time"

	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/test/factory/random"
	"golang.org/x/crypto/bcrypt"
)

// NewUser builds a new User struct with random data for testing purposes.
func (f *Factory) NewUser(overrideOpt ...ds.User) (m *ds.User) {
	m = &ds.User{
		ID:             ds.NewID(),
		Username:       random.String(),
		Email:          random.Email(),
		EmailConfirmed: false,
		Password:       "",
		CreatedAt:      time.Now(),
		UpdatedAt:      nil,
		DeletedAt:      nil,
		CleanedAt:      nil,
	}

	if len(overrideOpt) == 1 {
		merge(m, overrideOpt[0])
	}

	return
}

// CreateUser creates and persists a user to the database for testing purposes.
func (f *Factory) CreateUser(overrideOpt ...ds.User) (m *ds.User, err error) {
	m = f.NewUser(overrideOpt...)

	password := m.Password
	if password == "" {
		password = random.String()
	}

	passwordHashBytes, err := bcrypt.GenerateFromPassword([]byte(password), app.DefaultBCryptCost)
	if err != nil {
		return
	}

	m.Password = string(passwordHashBytes)

	err = f.repo.CreateUser(context.Background(), m)
	return
}
