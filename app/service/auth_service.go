package service

import (
	"context"
	"errors"
	"strings"

	z "github.com/Oudwins/zog"
	"github.com/markbates/goth"
	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/app/repo"
	"github.com/ognev-dev/goplease/app/session"
	"golang.org/x/crypto/bcrypt"
)

var authenticateUserInputRules = z.Shape{
	"Email":    z.String().Required(z.Message("Email is required")),
	"Password": z.String().Required(z.Message("Password is required")),
}

var authenticateOAuthUser = z.Shape{
	"Email":    emailInputRules,
	"Provider": z.String().Required(z.Message("provider is required")),
	"UserID":   z.String().Required(z.Message("user_id is required")),
}

var (
	// ErrInvalidEmailOrPassword is returned when a user attempts to log in with credentials
	// that do not match any record.
	ErrInvalidEmailOrPassword = app.ErrUnprocessable("invalid email or password")
)

// AuthenticateUser authenticates a user using their email and password.
func (s *Service) AuthenticateUser(ctx context.Context, email, password string) (
	user *ds.User, token string, err error) {
	ctx, span := s.tracer.Start(ctx, "AuthenticateUser")
	defer span.End()

	in := &AuthenticateUserInput{
		Email:    email,
		Password: password,
	}
	err = Normalize(in)
	if err != nil {
		return
	}

	user, err = s.db.GetUserByEmail(ctx, in.Email)
	if err != nil {
		if errors.Is(err, repo.ErrUserNotFound) {
			err = ErrInvalidEmailOrPassword
		}
		return
	}
	if user.Deleted() {
		err = ErrInvalidEmailOrPassword
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.Password))
	if err != nil {
		err = ErrInvalidEmailOrPassword
		return
	}

	token, err = s.newSignedSessionToken(ctx, user.ID)
	return
}

// newSignedSessionToken creates a new persistent session record in the database for the
// given userID and returns a signed JWT string for client-side authentication.
func (s *Service) newSignedSessionToken(ctx context.Context, userID ds.ID) (token string, err error) {
	sess, err := s.CreateUserSession(ctx, userID)
	if err != nil {
		return
	}

	return session.NewSignedJWT(sess.ID, userID)
}

// AuthenticateUserInput defines the input for user authentication.
type AuthenticateUserInput struct {
	Email, Password string //nolint:gosec
}

// Sanitize trims whitespace from email and password fields.
func (in *AuthenticateUserInput) Sanitize() {
	in.Email = strings.TrimSpace(in.Email)
	in.Password = strings.TrimSpace(in.Password)
}

// Validate validates the authentication input against defined rules.
func (in *AuthenticateUserInput) Validate() error {
	return validateInput(authenticateUserInputRules, in)
}

// AuthenticateOAuthUser authenticates a user via OAuth provider credentials.
// It resolves the user account from the OAuth data and creates a new session token.
func (s *Service) AuthenticateOAuthUser(ctx context.Context, authAcc goth.User) (token string, err error) {
	ctx, span := s.tracer.Start(ctx, "AuthenticateOAuthUser")
	defer span.End()

	in := &AuthenticateOAuthUserInput{&authAcc}
	user, err := s.ResolveUserFromOAuthAccount(ctx, *in.User)
	if err != nil {
		return
	}

	return s.newSignedSessionToken(ctx, user.ID)
}

// AuthenticateOAuthUserInput defines the input for OAuth user authentication.
type AuthenticateOAuthUserInput struct {
	*goth.User
}

// Sanitize trims whitespace from OAuth user fields.
func (in *AuthenticateOAuthUserInput) Sanitize() {
	in.UserID = strings.TrimSpace(in.UserID)
	in.Provider = strings.TrimSpace(in.Provider)
	in.Email = strings.TrimSpace(in.Email)
}

// Validate validates the OAuth authentication input against defined rules.
func (in *AuthenticateOAuthUserInput) Validate() error {
	return validateInput(authenticateOAuthUser, in)
}
