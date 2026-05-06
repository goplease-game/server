package email

import (
	"path"

	"github.com/ognev-dev/goplease/app"
)

// ConfirmEmail ...
type ConfirmEmail struct {
	Username   string
	Email      string
	Code       string
	ConfirmURL string
}

// Subject ...
func (ConfirmEmail) Subject() string {
	return "Email confirmation"
}

// TemplateName ...
func (ConfirmEmail) TemplateName() string {
	return "confirm_email"
}

// Variables ...
func (c ConfirmEmail) Variables() map[string]any {
	return map[string]any{
		"username":    c.Username,
		"email":       c.Email,
		"code":        c.Code,
		"confirm_url": path.Join(app.Config().Server.Addr, "/users/confirm-email/"),
	}
}
