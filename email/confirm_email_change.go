package email

import (
	"fmt"

	"github.com/ognev-dev/goplease/app"
)

// ConfirmEmailChange ...
type ConfirmEmailChange struct {
	Username    string
	Token       string
	ProjectName string
}

// Subject ...
func (p ConfirmEmailChange) Subject() string {
	return "Confirm Your New Email Address"
}

// TemplateName ...
func (p ConfirmEmailChange) TemplateName() string {
	return "confirm_email_change"
}

// Variables ...
func (p ConfirmEmailChange) Variables() map[string]any {
	return map[string]any{
		"Username":    p.Username,
		"Link":        fmt.Sprintf("%schange-email/%s/", app.Config().Server.Addr, p.Token),
		"ProjectName": app.Config().App.Name,

		// Token var is not used in email template, it is here to make testing easier.
		"token": p.Token,
	}
}
