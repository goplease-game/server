package service

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"time"

	z "github.com/Oudwins/zog"
	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/app/repo"
	"github.com/ognev-dev/goplease/email"
)

const emailConfirmationRetryAfterTolerance = 15 * time.Second

var (
	// ErrInvalidConfirmationCode is the specific error returned
	// when an email confirmation code is invalid or expired.
	ErrInvalidConfirmationCode = app.InputError{"code": "Invalid confirmation code"}

	// ErrEmailAlreadyConfirmed is returned when the user's email is already confirmed.
	ErrEmailAlreadyConfirmed = app.ErrUnprocessable("email already confirmed")

	// ErrResendConfirmationEmailCodeTooManyRequest is returned when the user requests a new confirmation email too soon.
	ErrResendConfirmationEmailCodeTooManyRequest = app.ErrTooManyRequests("We already sent you confirmation email recently.")
)

var createEmailConfirmationInputRules = z.Shape{
	"UserID": ds.IDInputRules,
}

const (
	emailConfirmationTTL        = time.Hour * 24
	emailConfirmationRetryAfter = time.Minute * 5
	emailConfirmationCodeLen    = 6
)

// ConfirmEmail confirms an email address by validating the provided code,
// setting the email_confirmed flag for the associated user, and then deleting the used confirmation record.
func (s *Service) ConfirmEmail(ctx context.Context, code string) (err error) {
	ctx, span := s.tracer.Start(ctx, "ConfirmEmail")
	defer span.End()

	in := &ConfirmEmailInput{Code: code}
	err = Normalize(in)
	if err != nil {
		return
	}

	ec, err := s.db.GetEmailConfirmationByCode(ctx, in.Code)
	if errors.Is(err, repo.ErrEmailConfirmationNotFound) {
		return ErrInvalidConfirmationCode
	}
	if err != nil {
		return err
	}

	if ec.Invalid() {
		return ErrInvalidConfirmationCode
	}

	err = s.db.SetUserEmailConfirmed(ctx, ec.UserID)
	if err != nil {
		return
	}

	err = s.db.DeleteEmailConfirmation(ctx, ec.ID)
	return err
}

// ResendConfirmationEmailCode sends a new confirmation email to the authenticated user.
func (s *Service) ResendConfirmationEmailCode(ctx context.Context) (retryAfter time.Duration, err error) {
	ctx, span := s.tracer.Start(ctx, "ResendConfirmationEmailCode")
	defer span.End()

	user := ds.UserFromContext(ctx)
	if user == nil {
		err = app.ErrUnauthorized()
		return
	}

	if user.EmailConfirmed {
		err = ErrEmailAlreadyConfirmed
		return
	}

	retryAfter, err = s.GetConfirmationEmailRetryAfter(ctx, user.ID)
	if err != nil {
		return
	}
	if retryAfter > emailConfirmationRetryAfterTolerance {
		err = ErrResendConfirmationEmailCodeTooManyRequest
		return
	}

	emailConfirmCode, err := s.CreateEmailConfirmation(ctx, user.ID)
	if err != nil {
		return
	}

	err = email.Send(user.Email, email.ConfirmEmail{
		Username: user.Username,
		Email:    user.Email,
		Code:     emailConfirmCode,
	})

	return
}

// GetConfirmationEmailRetryAfter returns the remaining cooldown duration before the user can request a new confirmation email.
func (s *Service) GetConfirmationEmailRetryAfter(ctx context.Context, userID ds.ID) (retryAfter time.Duration, err error) {
	ctx, span := s.tracer.Start(ctx, "GetConfirmationEmailRetryAfter")
	defer span.End()

	ec, err := s.db.GetLatestEmailConfirmationByUserID(ctx, userID)
	if errors.Is(err, repo.ErrEmailConfirmationNotFound) {
		return 0, nil
	}
	if err != nil || ec == nil {
		return 0, err
	}

	resendAt := ec.CreatedAt.Add(emailConfirmationRetryAfter)
	if time.Now().Before(resendAt) {
		retryAfter = time.Until(resendAt)
	}

	return retryAfter, nil
}

// ConfirmEmailInput defines the input for email confirmation.
type ConfirmEmailInput struct {
	Code string
}

// Sanitize trims whitespace from the confirmation code.
func (in *ConfirmEmailInput) Sanitize() {
	in.Code = strings.TrimSpace(in.Code)
}

// Validate validates the email confirmation input against defined rules.
func (in *ConfirmEmailInput) Validate() error {
	return validateInput(confirmEmailInputRules, in)
}

// CreateEmailConfirmation creates a new ds.EmailConfirmation for given user.
func (s *Service) CreateEmailConfirmation(ctx context.Context, userID ds.ID) (code string, err error) {
	ctx, span := s.tracer.Start(ctx, "CreateEmailConfirmation")
	defer span.End()

	in := &CreateEmailConfirmationInput{UserID: userID}
	err = Normalize(in)
	if err != nil {
		return
	}

	code, err = s.newEmailConfirmationCode(ctx)
	if err != nil {
		return
	}

	ec := &ds.EmailConfirmation{
		UserID:    in.UserID,
		Code:      code,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(emailConfirmationTTL),
	}

	err = s.db.CreateEmailConfirmation(ctx, ec)
	return
}

// CreateEmailConfirmationInput defines the input for creating an email confirmation.
type CreateEmailConfirmationInput struct {
	UserID ds.ID
}

// Sanitize performs no sanitization for this input.
func (in *CreateEmailConfirmationInput) Sanitize() {
}

// Validate validates the email confirmation input against defined rules.
func (in *CreateEmailConfirmationInput) Validate() error {
	return validateInput(createEmailConfirmationInputRules, in)
}

// newEmailConfirmationCode generates a unique email confirmation code.
// It checks for collisions and increments the code length if necessary.
func (s *Service) newEmailConfirmationCode(ctx context.Context) (string, error) {
	chars := []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")

	length := emailConfirmationCodeLen
	newCode := func(length int) string {
		token := make([]byte, length)
		for i := range length {
			token[i] = chars[rand.Intn(len(chars))] //nolint:gosec
		}

		return string(token)
	}

	for {
		code := newCode(length)

		_, err := s.db.GetEmailConfirmationByCode(ctx, code)
		if errors.Is(err, repo.ErrEmailConfirmationNotFound) {
			return code, nil
		}
		if err != nil {
			return "", err
		}

		length++
	}
}
