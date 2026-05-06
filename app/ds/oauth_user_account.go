package ds

import (
	"time"

	"github.com/ognev-dev/goplease/oauth/provider"
)

// OAuthUserAccount links a local user to their external identity provider credentials.
type OAuthUserAccount struct {
	ID             ID            `json:"id"`
	UserID         ID            `json:"user_id"`
	Provider       provider.Type `json:"provider"`
	ProviderUserID string        `json:"provider_user_id"`
	CreatedAt      time.Time     `json:"created_at"`
}
