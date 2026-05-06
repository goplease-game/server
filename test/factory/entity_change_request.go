package factory

import (
	"context"
	"time"

	"github.com/ognev-dev/goplease/app/ds"
)

// NewEntityChangeRequest creates a new EntityChangeRequest with default values.
func (f *Factory) NewEntityChangeRequest(overrideOpt ...ds.EntityChangeRequest) (m *ds.EntityChangeRequest) {
	m = &ds.EntityChangeRequest{
		ID:         ds.NewID(),
		Status:     ds.EntityChangePending,
		Diff:       make(map[string]any),
		Message:    "",
		Revision:   1,
		ReviewerID: nil,
		ReviewedAt: nil,
		ReviewNote: "",
		CreatedAt:  time.Now(),
		UpdatedAt:  nil,
	}

	if len(overrideOpt) == 1 {
		merge(m, overrideOpt[0])
	}

	return
}

// CreateEntityChangeRequest creates and persists a new EntityChangeRequest to the repository.
// Automatically creates associated Book and User entities if not provided in the override.
func (f *Factory) CreateEntityChangeRequest(overrideOpt ...ds.EntityChangeRequest) (
	m *ds.EntityChangeRequest, err error) {
	m = f.NewEntityChangeRequest(overrideOpt...)

	if m.EntityID.IsNil() {
		e, err := f.CreateBook()
		if err != nil {
			return nil, err
		}

		m.EntityID = e.ID
	}

	if m.UserID.IsNil() {
		u, err := f.CreateUser()
		if err != nil {
			return nil, err
		}

		m.UserID = u.ID
	}

	err = f.repo.CreateChangeRequest(context.Background(), m)
	return
}
