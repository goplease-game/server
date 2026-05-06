package factory

import (
	"context"
	"strings"

	fake "github.com/brianvoe/gofakeit/v7"
	"github.com/ognev-dev/goplease/app/ds"
)

// NewPage constructs a new Page model with default values.
func (f *Factory) NewPage(overrideOpt ...ds.Page) (m *ds.Page) {
	text := strings.Repeat(fake.Paragraph(), 5) //nolint:mnd

	m = &ds.Page{
		Entity: f.NewEntity(ds.Entity{
			Visibility: ds.EntityVisibilityPublic,
			Status:     ds.EntityStatusApproved,
			Type:       ds.EntityTypePage,
		}),

		ContentRaw: text,
		Content:    text,
	}

	if len(overrideOpt) == 1 {
		o := overrideOpt[0]
		merge(m, o)

		if o.Entity == nil {
			o.Entity = &ds.Entity{}
		} else {
			merge(m.Entity, o.Entity)
		}
	}

	return
}

// CreatePage creates and persists a Page domain model using the factory.
func (f *Factory) CreatePage(overrideOpt ...ds.Page) (m *ds.Page, err error) {
	m = f.NewPage(overrideOpt...)

	m.Type = ds.EntityTypePage
	m.Entity, err = f.CreateEntity(*m.Entity)
	if err != nil {
		return
	}

	err = f.repo.CreatePage(context.Background(), m)
	return
}
