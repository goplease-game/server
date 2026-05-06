// Package email ...
package email

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"sync"

	"github.com/ognev-dev/goplease/app"
)

var (
	// ErrInvalidDriver indicates that the configured email driver name is not recognized or supported.
	ErrInvalidDriver = errors.New("invalid email driver")
)

var initDriverOnce sync.Once
var driver Sender

const (
	// SMTPDriver is used to identify the standard SMTP email sending implementation.
	SMTPDriver = "smtp"

	// TestDriver is used to identify the in-memory email sender implementation for testing purposes.
	TestDriver = "test"
)

//go:embed *.html
var templateFiles embed.FS
var templates = template.Must(template.ParseFS(templateFiles, "*.html"))

// Sender is the interface defining the capability to send an email message.
type Sender interface {
	Send(to string, c Composer) error
}

// Composer is the interface defining the components required to compose a full email message.
type Composer interface {
	Subject() string
	TemplateName() string
	Variables() map[string]any
}

// Send initializes the appropriate email driver (if not already done) and dispatches the email.
func Send(to string, c Composer) (err error) {
	initDriverOnce.Do(func() {
		conf := app.Config().Email
		switch conf.Driver {
		case SMTPDriver:
			driver, err = NewSMTPSender()
		case TestDriver:
			driver = new(TestSender)
		default:
			err = fmt.Errorf("driver '%s': %w", conf.Driver, ErrInvalidDriver)
		}
	})

	if err != nil {
		return
	}

	return driver.Send(to, c)
}

// TemplateData represents the data that passed to the base email layout template.
type TemplateData struct {
	Subject string
	Body    template.HTML
}

func renderTemplate(c Composer) (result string, err error) {
	var buff bytes.Buffer

	err = templates.ExecuteTemplate(&buff, c.TemplateName()+".html", c.Variables())
	if err != nil {
		return
	}

	var tBuff bytes.Buffer

	err = templates.ExecuteTemplate(&tBuff, "template.html", TemplateData{
		Subject: c.Subject(),
		Body:    template.HTML(buff.String()), //nolint:gosec
	})
	if err != nil {
		return
	}

	return tBuff.String(), nil
}
