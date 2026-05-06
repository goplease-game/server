package request

import (
	"time"

	"github.com/ognev-dev/goplease/app/ds"
)

// CreatePage defines the request payload for creating a new page entity.
type CreatePage struct {
	PublicID string `json:"public_id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
}

// ToPage converts the CreatePage request into an Entity model.
func (r *CreatePage) ToPage() *ds.Page {
	return &ds.Page{
		Entity: &ds.Entity{
			ID:          ds.NewID(),
			OwnerID:     ds.NilID,
			Type:        ds.EntityTypePage,
			PublicID:    r.PublicID,
			Title:       r.Title,
			Summary:     "",
			Visibility:  ds.EntityVisibilityPublic,
			Status:      ds.EntityStatusUnderReview,
			PublishedAt: nil,
			CreatedAt:   time.Now(),
			UpdatedAt:   nil,
			DeletedAt:   nil,
		},

		ContentRaw: r.Content,
	}
}

// UpdatePage defines the request payload for updating an existing page.
// It reuses CreatePage fields as the updatable subset.
type UpdatePage struct {
	CreatePage
}
