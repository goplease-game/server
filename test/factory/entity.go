//nolint:mnd
package factory

import (
	"context"
	"fmt"
	"time"

	fake "github.com/brianvoe/gofakeit/v7"
	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/test/factory/random"
)

// NewEntity ...
func (f *Factory) NewEntity(overrideOpt ...ds.Entity) (m *ds.Entity) {
	text := fake.Paragraph()
	createdAt := fake.DateRange(time.Now().AddDate(0, -12, 0), time.Now())
	var publishedAt, updatedAt, deletedAt *time.Time

	status := random.Element(ds.EntityStatuses)
	if status == ds.EntityStatusApproved {
		publishedAt = &createdAt
		updatedAt = random.NilOrValue(fake.DateRange(createdAt.AddDate(0, -12, -25), createdAt), 50)
	}

	title := fake.BookTitle()

	m = &ds.Entity{
		ID:            ds.NewID(),
		PublicID:      app.Slug(title),
		OwnerID:       ds.NilID,
		PreviewFileID: ds.NilID,
		Title:         title,
		SummaryRaw:    text,
		Summary:       text,
		Visibility:    random.Element(ds.EntityVisibilities),
		Status:        status,
		PublishedAt:   publishedAt,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		DeletedAt:     deletedAt,
	}

	if len(overrideOpt) == 1 {
		merge(m, overrideOpt[0])
	}

	if m.Type == ds.EntityTypePage {
		m.Summary = ""
		m.SummaryRaw = ""
	}

	return
}

// CreateEntity ...
func (f *Factory) CreateEntity(overrideOpt ...ds.Entity) (m *ds.Entity, err error) {
	m = f.NewEntity(overrideOpt...)

	if m.OwnerID.IsNil() {
		u, err := f.CreateUser()
		if err != nil {
			return nil, err
		}

		m.OwnerID = u.ID
	}

createEntity:
	err = f.repo.CreateEntity(context.Background(), m)
	if column, ok := app.IsUniqueViolation(err); ok {
		switch column {
		case "public_id":
			m.PublicID, err = LookupUnique(context.Background(), f.db, "entities", "public_id", m.PublicID, func(s string) string {
				return s + "-" + fake.UrlSlug(1)
			})
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("%w: column: %q", app.ErrUniqueViolation, column)
		}

		goto createEntity
	}
	if err != nil {
		return
	}

	if len(m.Topics) > 0 {
		err = f.repo.AttachTopics(context.Background(), m.ID, m.Topics)
		if err != nil {
			return
		}
	}

	return m, nil
}
