package email

import (
	"errors"
	"fmt"
	"sync"
)

var (
	// ErrDriverIsNil indicates that the global email sender driver has not been initialized.
	ErrDriverIsNil = errors.New("email driver is nil")

	// ErrDriverIsNotTestSender indicates that an attempt was made to load a test email,
	// but the currently active email driver is not the TestSender implementation.
	ErrDriverIsNotTestSender = errors.New("email driver is not TestSender")

	// ErrEmailNotExists indicates that no email was found in the TestSender's store for the given recipient address.
	ErrEmailNotExists = errors.New("email for recipient not found")

	// ErrEmailIsNotComposerType indicates that the value retrieved from the TestSender's storage
	// was not of the expected Composer interface type.
	ErrEmailIsNotComposerType = errors.New("email is not composer type")
)

// TestSender is an in-memory implementation of the email sender interface.
// It stores sent emails for later inspection in tests instead of actually sending them.
type TestSender struct {
	emails sync.Map
}

// Send records the email and its recipient into the in-memory store.
// In a test environment, this method returns nil to simulate a successful send operation.
func (t *TestSender) Send(to string, c Composer) (err error) {
	t.emails.Store(to, c)

	return nil
}

// LoadTestEmail retrieves the Composer content of an email sent to a specific recipient address
// from the currently active email driver.
//
// This function is intended for use in integration or unit tests to assert the content
// of an email sent during a test scenario.
// It will fail if the global driver is not a *TestSender.
func LoadTestEmail(to string) (c Composer, err error) {
	if driver == nil {
		err = ErrDriverIsNil

		return
	}

	sender, ok := driver.(*TestSender)
	if !ok {
		err = ErrDriverIsNotTestSender
		return
	}

	v, ok := sender.emails.Load(to)
	if !ok {
		err = fmt.Errorf("%s: %w", to, ErrEmailNotExists)
		return
	}

	c, ok = v.(Composer)
	if !ok {
		err = ErrEmailIsNotComposerType
		return
	}

	return c, nil
}
