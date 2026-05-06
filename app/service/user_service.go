package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	z "github.com/Oudwins/zog"
	"github.com/google/uuid"
	"github.com/markbates/goth"
	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/app/repo"
	"github.com/ognev-dev/goplease/app/session"
	"github.com/ognev-dev/goplease/email"
	"github.com/ognev-dev/goplease/oauth/provider"
	"github.com/ognev-dev/goplease/test/factory/random"
	"golang.org/x/crypto/bcrypt"
)

var changePasswordInputRules = z.Shape{
	"UserID":      ds.IDInputRules,
	"OldPassword": z.String().Required(z.Message("Password is required")),
	"NewPassword": newPasswordInputRules,
}

var changeUsernameInputRules = z.Shape{
	"UserID":      ds.IDInputRules,
	"NewUsername": usernameInputRules,
	"Password":    z.String().Required(),
}

var confirmEmailChangeInputRules = z.Shape{
	"token": z.String().Required(z.Message("Token is required")),
}

var createChangeEmailRequestInputRules = z.Shape{
	"UserID":   ds.IDInputRules,
	"NewEmail": emailInputRules,
}

var createOAuthUserAccountInputRules = z.Shape{
	"UserID":         ds.IDInputRules,
	"Provider":       provider.TypeInputRules,
	"ProviderUserID": z.String().Required(z.Message("provider_user_id is required")),
}

var createPasswordResetRequestInputRules = z.Shape{
	"Email": emailInputRules,
}

var createUserSessionInputRules = z.Shape{
	"UserID": ds.IDInputRules,
}

var deleteUserInputRules = z.Shape{
	"UserID":   ds.IDInputRules,
	"Password": z.String().Required(),
}

var findPasswordResetByTokenInputRules = z.Shape{
	"Token": z.String().Required(z.Message("Token is required")),
}

var findUserByEmailInputRules = z.Shape{
	"Email": emailInputRules,
}

var findUserByIDInputRules = z.Shape{
	"ID": ds.IDInputRules,
}

var getOAuthUserAccountInputRules = z.Shape{
	"Provider":       provider.TypeInputRules,
	"ProviderUserID": z.String().Required(z.Message("provider_user_id is required")),
}

var hardDeleteUserInputRules = z.Shape{
	"UserID": ds.IDInputRules,
}

var getUserAndSessionFromJWTInputRules = z.Shape{
	"Token": z.String().Required(z.Message("Token is required")),
}

var confirmEmailInputRules = z.Shape{
	"code": z.String().Required(z.Message("Code is required")),
}

var usernameInputRules = z.String().Required().
	Min(UsernameMinLen, z.Message("Username must be at least 2 characters")).
	Max(UsernameMaxLen, z.Message("Username must be at most 30 characters")).
	Required(z.Message("Username is required")).
	Match(UsernameBasicRegex,
		z.Message("Username can only contain letters, numbers, dots, underscores, and dashes")).
	Match(UsernameSpecialCharsRegex,
		z.Message("Username cannot contain more than two dots, underscores, or dashes"))

var registerUserInputRules = z.Shape{
	"Username": usernameInputRules,
	"Email":    emailInputRules,
	"Password": newPasswordInputRules,
}

var resetPasswordInputRules = z.Shape{
	"Token":    z.String().Required(z.Message("Token is required")),
	"Password": newPasswordInputRules,
}

var (
	// UsernameBasicRegex defines the basic character set allowed in a username (letters, numbers, dot, underscore, dash).
	UsernameBasicRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

	// UsernameSpecialCharsRegex enforces a limit on the maximum number of special characters (dot, underscore, dash).
	UsernameSpecialCharsRegex = regexp.MustCompile(`^[^._-]*([._-][^._-]*){0,2}$`)
)

const (
	// UserWithThisEmailAlreadyExists is the specific error message for email validation failure during registration.
	UserWithThisEmailAlreadyExists = "User with this email already exists."

	// UsernameAlreadyTaken is the specific error message for username validation failure during registration.
	UsernameAlreadyTaken = "Username already taken"

	passwordResetTokenLength = 32
	emailChangeTokenLength   = 32
)

var (
	// ErrInvalidPasswordResetToken ...
	ErrInvalidPasswordResetToken = app.ErrUnprocessable("password reset request is either expired or invalid")

	// ErrInvalidPassword is returned when a user tries to change their password
	// but provides an incorrect old password.
	ErrInvalidPassword = app.ErrUnprocessable("invalid password")

	// ErrInvalidChangeEmailToken ...
	ErrInvalidChangeEmailToken = app.ErrUnprocessable("change email request is expired or invalid")

	// ErrChangeEmailToSameEmail ...
	ErrChangeEmailToSameEmail = app.ErrUnprocessable("you already use this email, no change needed")

	// ErrInvalidJWT is returned when an authentication token is malformed,
	// invalidly signed, or contains unexpected claims.
	ErrInvalidJWT = app.ErrForbidden("invalid token")

	// ErrSessionExpired is returned when a JWT is validly signed but the associated
	// database session has expired based on its timestamp.
	ErrSessionExpired = app.ErrForbidden("session expired")
)

// FilterUsers retrieves a filtered list of users based on the provided filter criteria.
func (s *Service) FilterUsers(ctx context.Context, f ds.UsersFilter) (data []ds.User, count int, err error) {
	ctx, span := s.tracer.Start(ctx, "FilterUsers")
	defer span.End()

	return s.db.FilterUsers(ctx, f)
}

// ChangeUserPassword handles the logic for an authenticated user to change their own password.
func (s *Service) ChangeUserPassword(ctx context.Context, userID ds.ID, oldPassword, newPassword string) (err error) {
	ctx, span := s.tracer.Start(ctx, "ChangeUserPassword")
	defer span.End()

	in := &ChangeUserPasswordInput{
		UserID:      userID,
		OldPassword: oldPassword,
		NewPassword: newPassword,
	}
	err = Normalize(in)
	if err != nil {
		return err
	}

	user, err := s.GetUserByID(ctx, in.UserID)
	if user == nil {
		return err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.OldPassword))
	if err != nil {
		return app.InputError{"old_password": ErrInvalidPassword.Error()}
	}

	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), app.DefaultBCryptCost)
	if err != nil {
		return err
	}

	err = s.db.UpdateUserPassword(ctx, user.ID, string(newPasswordHash))
	return err
}

// ChangeUserPasswordInput defines the input for changing a user's password.
type ChangeUserPasswordInput struct {
	UserID      ds.ID
	OldPassword string
	NewPassword string
}

// Sanitize trims whitespace from password fields.
func (in *ChangeUserPasswordInput) Sanitize() {
	in.OldPassword = strings.TrimSpace(in.OldPassword)
	in.NewPassword = strings.TrimSpace(in.NewPassword)
}

// Validate validates the change password input against defined rules.
func (in *ChangeUserPasswordInput) Validate() error {
	return validateInput(changePasswordInputRules, in)
}

// ChangeUsername handles the logic for changing a user's username.
func (s *Service) ChangeUsername(ctx context.Context, in ChangeUsernameInput) (err error) {
	ctx, span := s.tracer.Start(ctx, "ChangeUsername")
	defer span.End()

	err = Normalize(&in)
	if err != nil {
		return
	}

	user, err := s.db.GetUserByID(ctx, in.UserID)
	if err != nil {
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.Password))
	if err != nil {
		return app.InputError{"password": "Incorrect password"}
	}

	existingUser, err := s.db.GetUserByUsername(ctx, in.NewUsername)
	if errors.Is(err, repo.ErrUserNotFound) {
		err = nil
	}
	if err != nil {
		return
	}
	if existingUser != nil {
		return app.InputError{"username": UsernameAlreadyTaken}
	}

	err = s.db.UpdateUsername(ctx, user.ID, in.NewUsername)
	return
}

// ChangeUsernameInput defines the input for changing a user's username.
type ChangeUsernameInput struct {
	UserID      ds.ID
	NewUsername string
	Password    string //nolint:gosec
}

// Sanitize trims whitespace from the new username.
func (in *ChangeUsernameInput) Sanitize() {
	in.NewUsername = strings.TrimSpace(in.NewUsername)
}

// Validate validates the change username input against defined rules.
func (in *ChangeUsernameInput) Validate() error {
	return validateInput(changeUsernameInputRules, in)
}

// ConfirmEmailChange handles the logic for finalizing an email change via a token.
func (s *Service) ConfirmEmailChange(ctx context.Context, token string) (err error) {
	ctx, span := s.tracer.Start(ctx, "ConfirmEmailChange")
	defer span.End()

	in := &ConfirmEmailChangeInput{Token: token}
	err = Normalize(in)
	if err != nil {
		return
	}

	req, err := s.db.GetChangeEmailRequestByToken(ctx, in.Token)
	if errors.Is(err, repo.ErrChangeEmailRequestNotFound) {
		return ErrInvalidChangeEmailToken
	}
	if err != nil {
		return
	}

	if req.Invalid() {
		return ErrInvalidChangeEmailToken
	}

	err = s.db.UpdateUserEmail(ctx, req.UserID, req.NewEmail)
	if err != nil {
		return
	}

	return s.db.DeleteChangeEmailRequest(ctx, req.ID)
}

// ConfirmEmailChangeInput defines the input for confirming an email change.
type ConfirmEmailChangeInput struct {
	Token string
}

// Sanitize trims whitespace from the token.
func (in *ConfirmEmailChangeInput) Sanitize() {
	in.Token = strings.TrimSpace(in.Token)
}

// Validate validates the email change confirmation input against defined rules.
func (in *ConfirmEmailChangeInput) Validate() error {
	return validateInput(confirmEmailChangeInputRules, in)
}

// CreateChangeEmailRequest handles the business logic for a user initiating an email change.
func (s *Service) CreateChangeEmailRequest(ctx context.Context, userID ds.ID, newEmail string) (err error) {
	ctx, span := s.tracer.Start(ctx, "CreateChangeEmailRequest")
	defer span.End()

	in := &CreateChangeEmailRequestInput{
		UserID:   userID,
		NewEmail: newEmail,
	}
	err = Normalize(in)
	if err != nil {
		return
	}

	user, err := s.db.GetUserByID(ctx, userID)
	if err != nil {
		return
	}

	if user.Email == in.NewEmail {
		return ErrChangeEmailToSameEmail
	}

	// Check if the new email is already taken by another user.
	existingUser, err := s.db.GetUserByEmail(ctx, in.NewEmail)
	if errors.Is(err, repo.ErrUserNotFound) {
		err = nil
	}
	if err != nil {
		return
	}
	if existingUser != nil && existingUser.ID != user.ID {
		return app.InputError{"email": UserWithThisEmailAlreadyExists}
	}

	token, err := app.Token(emailChangeTokenLength)
	if err != nil {
		return
	}

	req := &ds.ChangeEmailRequest{
		UserID:    user.ID,
		NewEmail:  in.NewEmail,
		Token:     token,
		ExpiresAt: time.Now().Add(time.Hour * 1),
		CreatedAt: time.Now(),
	}

	err = s.db.CreateChangeEmailRequest(ctx, req)
	if err != nil {
		return
	}

	return email.Send(in.NewEmail, email.ConfirmEmailChange{
		Username: user.Username,
		Token:    token,
	})
}

// CreateChangeEmailRequestInput defines the input for creating an email change request.
type CreateChangeEmailRequestInput struct {
	UserID   ds.ID
	NewEmail string
}

// Sanitize trims whitespace from the new email.
func (in *CreateChangeEmailRequestInput) Sanitize() {
	in.NewEmail = strings.TrimSpace(in.NewEmail)
}

// Validate validates the email change request input against defined rules.
func (in *CreateChangeEmailRequestInput) Validate() error {
	return validateInput(createChangeEmailRequestInputRules, in)
}

// CreateOAuthUserAccount creates a new user session object.
func (s *Service) CreateOAuthUserAccount(ctx context.Context, m *ds.OAuthUserAccount) (err error) {
	ctx, span := s.tracer.Start(ctx, "CreateOAuthUserAccount")
	defer span.End()

	in := &CreateOAuthUserAccountInput{m}
	err = Normalize(in)
	if err != nil {
		return
	}

	return s.db.CreateOAuthUserAccount(ctx, in.OAuthUserAccount)
}

// CreateOAuthUserAccountInput defines the input for creating an OAuth user account.
type CreateOAuthUserAccountInput struct {
	*ds.OAuthUserAccount
}

// Sanitize trims whitespace from the provider user ID.
func (in *CreateOAuthUserAccountInput) Sanitize() {
	in.ProviderUserID = strings.TrimSpace(in.ProviderUserID)
}

// Validate validates the OAuth user account input against defined rules.
func (in *CreateOAuthUserAccountInput) Validate() error {
	return validateInput(createOAuthUserAccountInputRules, in)
}

// CreatePasswordResetRequest handles the logic for initiating a password reset.
func (s *Service) CreatePasswordResetRequest(ctx context.Context, emailAddr string) (err error) {
	ctx, span := s.tracer.Start(ctx, "CreatePasswordResetRequest")
	defer span.End()

	in := &CreatePasswordResetRequestInput{Email: emailAddr}
	err = Normalize(in)
	if err != nil {
		return
	}

	user, err := s.db.GetUserByEmail(ctx, in.Email)
	if err != nil {
		// If the user is not found, we don't return an error to prevent email enumeration attacks.
		if errors.Is(err, repo.ErrUserNotFound) {
			err = nil
		}

		return
	}

	resetToken, err := app.Token(passwordResetTokenLength)
	if err != nil {
		return
	}

	token := &ds.PasswordResetToken{
		UserID:    user.ID,
		Token:     resetToken,
		ExpiresAt: time.Now().Add(time.Hour * 1),
		CreatedAt: time.Now(),
	}

	err = s.db.CreatePasswordResetToken(ctx, token)
	if err != nil {
		return
	}

	return email.Send(user.Email, email.PasswordResetRequest{
		Username: user.Username,
		Token:    resetToken,
	})
}

// CreatePasswordResetRequestInput defines the input for creating a password reset request.
type CreatePasswordResetRequestInput struct {
	Email string
}

// Sanitize trims whitespace from the email.
func (in *CreatePasswordResetRequestInput) Sanitize() {
	in.Email = strings.TrimSpace(in.Email)
}

// Validate validates the password reset request input against defined rules.
func (in *CreatePasswordResetRequestInput) Validate() error {
	return validateInput(createPasswordResetRequestInputRules, in)
}

// CreateUserSession creates a new user session object.
func (s *Service) CreateUserSession(ctx context.Context, userID ds.ID) (sess *ds.UserSession, err error) {
	ctx, span := s.tracer.Start(ctx, "CreateUserSession")
	defer span.End()

	in := &CreateUserSessionInput{UserID: userID}
	err = Normalize(in)
	if err != nil {
		return
	}

	sess = &ds.UserSession{
		ID:        ds.NewID(),
		UserID:    in.UserID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour * time.Duration(app.Config().Session.DurationHours)),
	}

	err = s.db.CreateUserSession(ctx, sess)
	if err != nil {
		return
	}

	return
}

// CreateUserSessionInput defines the input for creating a user session.
type CreateUserSessionInput struct {
	UserID ds.ID
}

// Sanitize performs no sanitization for this input.
func (in *CreateUserSessionInput) Sanitize() {}

// Validate validates the user session input against defined rules.
func (in *CreateUserSessionInput) Validate() error {
	return validateInput(createUserSessionInputRules, in)
}

// DeleteUser handles the logic for soft-deleting a user account.
func (s *Service) DeleteUser(ctx context.Context, userID ds.ID, password string) (err error) {
	ctx, span := s.tracer.Start(ctx, "DeleteUser")
	defer span.End()

	in := DeleteUserInput{
		UserID:   userID,
		Password: password,
	}
	err = Normalize(&in)
	if err != nil {
		return
	}

	user, err := s.db.GetUserByID(ctx, in.UserID)
	if err != nil {
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.Password))
	if err != nil {
		return app.InputError{"password": "Incorrect password"}
	}

	err = s.db.DeleteUser(ctx, user.ID)
	if err != nil {
		return
	}

	err = s.db.DeleteSessionsByUserID(ctx, user.ID)
	if err != nil {
		return
	}

	return
}

// DeleteUserInput defines the input for deleting a user.
type DeleteUserInput struct {
	UserID   ds.ID
	Password string //nolint:gosec
}

// Sanitize performs no sanitization for this input.
func (in *DeleteUserInput) Sanitize() {
}

// Validate validates the delete user input against defined rules.
func (in *DeleteUserInput) Validate() error {
	return validateInput(deleteUserInputRules, in)
}

// DeleteUserSession removes a user session record from the database using its ID.
func (s *Service) DeleteUserSession(ctx context.Context, id ds.ID) (err error) {
	ctx, span := s.tracer.Start(ctx, "DeleteUserSession")
	defer span.End()

	in := &DeleteUserSessionInput{ID: id}
	err = Normalize(in)
	if err != nil {
		return
	}

	return s.db.DeleteUserSession(ctx, in.ID)
}

// DeleteUserSessionInput defines the input for deleting a user session.
type DeleteUserSessionInput struct {
	ID ds.ID
}

// Sanitize performs no sanitization for this input.
func (in *DeleteUserSessionInput) Sanitize() {}

// Validate validates the delete session input (no validation rules defined).
func (in *DeleteUserSessionInput) Validate() error {
	return nil
}

// GetPasswordResetByToken retrieves a password reset token from the database and validates it.
func (s *Service) GetPasswordResetByToken(ctx context.Context, token string) (prt *ds.PasswordResetToken, err error) {
	ctx, span := s.tracer.Start(ctx, "GetPasswordResetByToken")
	defer span.End()

	in := &FindPasswordResetByTokenInput{Token: token}
	err = Normalize(in)
	if err != nil {
		return
	}

	prt, err = s.db.GetPasswordResetToken(ctx, in.Token)
	if errors.Is(err, repo.ErrPasswordResetTokenNotFound) {
		err = ErrInvalidPasswordResetToken
		return
	}
	if err != nil {
		return
	}

	if prt.Invalid() {
		err = ErrInvalidPasswordResetToken
		return
	}

	return
}

// FindPasswordResetByTokenInput defines the input for finding a password reset token.
type FindPasswordResetByTokenInput struct {
	Token string
}

// Sanitize trims whitespace from the token.
func (in *FindPasswordResetByTokenInput) Sanitize() {
	in.Token = strings.TrimSpace(in.Token)
}

// Validate validates the password reset token input against defined rules.
func (in *FindPasswordResetByTokenInput) Validate() error {
	return validateInput(findPasswordResetByTokenInputRules, in)
}

// GetUserByEmail retrieves a user record from the database by their email address.
func (s *Service) GetUserByEmail(ctx context.Context, email string) (user *ds.User, err error) {
	ctx, span := s.tracer.Start(ctx, "GetUserByEmail")
	defer span.End()

	in := &FindUserByEmailInput{Email: email}
	err = Normalize(in)
	if err != nil {
		return
	}

	return s.db.GetUserByEmail(ctx, in.Email)
}

// FindUserByEmailInput defines the input for finding a user by email.
type FindUserByEmailInput struct {
	Email string
}

// Sanitize trims whitespace from the email.
func (in *FindUserByEmailInput) Sanitize() {
	in.Email = strings.TrimSpace(in.Email)
}

// Validate validates the find user by email input against defined rules.
func (in *FindUserByEmailInput) Validate() error {
	return validateInput(findUserByEmailInputRules, in)
}

// GetUserByID retrieves a user record from the database by their ID.
func (s *Service) GetUserByID(ctx context.Context, id ds.ID) (user *ds.User, err error) {
	ctx, span := s.tracer.Start(ctx, "GetUserByID")
	defer span.End()

	in := &FindUserByIDInput{ID: id}
	err = Normalize(in)
	if err != nil {
		return
	}

	return s.db.GetUserByID(ctx, in.ID)
}

// FindUserByIDInput defines the input for finding a user by ID.
type FindUserByIDInput struct {
	ID ds.ID
}

// Sanitize performs no sanitization for this input.
func (in *FindUserByIDInput) Sanitize() {}

// Validate validates the find user by ID input against defined rules.
func (in *FindUserByIDInput) Validate() error {
	return validateInput(findUserByIDInputRules, in)
}

// GetUserSessionByID retrieves a user session from the database using its ID.
func (s *Service) GetUserSessionByID(ctx context.Context, id ds.ID) (sess *ds.UserSession, err error) {
	ctx, span := s.tracer.Start(ctx, "GetUserSessionByID")
	defer span.End()

	in := &FindUserSessionByIDInput{ID: id}
	err = Normalize(in)
	if err != nil {
		return
	}

	return s.db.GetUserSessionByID(ctx, id)
}

// FindUserSessionByIDInput defines the input for finding a user session by ID.
type FindUserSessionByIDInput struct {
	ID ds.ID
}

// Sanitize performs no sanitization for this input.
func (in *FindUserSessionByIDInput) Sanitize() {}

// Validate validates the find session input (no validation rules defined).
func (in *FindUserSessionByIDInput) Validate() error {
	return nil
}

// GetOAuthUserAccount retrieves an OAuth user account by provider and provider user ID.
func (s *Service) GetOAuthUserAccount(
	ctx context.Context, prov provider.Type, provUserID string) (m *ds.OAuthUserAccount, err error) {
	ctx, span := s.tracer.Start(ctx, "GetOAuthUserAccount")
	defer span.End()

	in := &GetOAuthUserAccountInput{
		Provider:       prov,
		ProviderUserID: provUserID,
	}
	err = Normalize(in)
	if err != nil {
		return
	}

	return s.db.GetOAuthUserAccount(ctx, in.Provider, in.ProviderUserID)
}

// GetOAuthUserAccountInput defines the input for getting an OAuth user account.
type GetOAuthUserAccountInput struct {
	Provider       provider.Type
	ProviderUserID string
}

// Sanitize trims whitespace from the provider user ID.
func (in *GetOAuthUserAccountInput) Sanitize() {
	in.ProviderUserID = strings.TrimSpace(in.ProviderUserID)
}

// Validate validates the OAuth user account input against defined rules.
func (in *GetOAuthUserAccountInput) Validate() error {
	return validateInput(getOAuthUserAccountInputRules, in)
}

// GetUserAndSessionFromJWT checks the associated session's validity and retrieves the corresponding user record.
func (s *Service) GetUserAndSessionFromJWT(ctx context.Context, token string) (
	user *ds.User, sess *ds.UserSession, err error) {
	ctx, span := s.tracer.Start(ctx, "GetUserAndSessionFromJWT")
	defer span.End()

	in := &GetUserAndSessionFromJWTInput{Token: token}
	err = Normalize(in)
	if err != nil {
		return
	}

	sessionID, userID, err := session.UnpackFromJWT(in.Token)
	if err != nil {
		return
	}

	sess, err = s.GetUserSessionByID(ctx, sessionID)
	if err != nil {
		return
	}

	if sess.UserID != userID {
		return nil, nil, ErrInvalidJWT
	}

	if sess.ExpiresAt.Before(time.Now()) {
		err = s.DeleteUserSession(ctx, sess.ID)
		if err != nil {
			return
		}

		err = ErrSessionExpired
		return
	}

	user, err = s.GetUserByID(ctx, sess.UserID)
	return
}

// GetUserAndSessionFromJWTInput defines the input for authenticating via JWT.
type GetUserAndSessionFromJWTInput struct {
	Token string
}

// Sanitize trims whitespace from the token.
func (in *GetUserAndSessionFromJWTInput) Sanitize() {
	in.Token = strings.TrimSpace(in.Token)
}

// Validate validates the JWT input against defined rules.
func (in *GetUserAndSessionFromJWTInput) Validate() error {
	return validateInput(getUserAndSessionFromJWTInputRules, in)
}

// CleanupDeletedUser handles the logic for deleting a user account and relations.
func (s *Service) CleanupDeletedUser(ctx context.Context, userID ds.ID) (err error) {
	ctx, span := s.tracer.Start(ctx, "CleanupDeletedUser")
	defer span.End()

	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return
	}

	// sessions
	err = s.db.DeleteSessionsByUserID(ctx, userID)
	if err != nil {
		return
	}

	// email confirmations
	err = s.db.DeleteEmailConfirmationByUser(ctx, userID)
	if err != nil {
		return
	}

	// password resets
	err = s.db.DeletePasswordResetTokensByUser(ctx, userID)
	if err != nil {
		return
	}

	// email change requests
	err = s.db.DeleteChangeEmailRequestsByUser(ctx, userID)
	if err != nil {
		return
	}

	user.Email = "deleted-" + random.String(16) + "-" + uuid.NewString()    //nolint:mnd
	user.Username = "deleted-" + random.String(16) + "-" + uuid.NewString() //nolint:mnd
	user.Password = "deleted-" + random.String(16)                          //nolint:mnd
	user.CleanedAt = new(time.Now())

	return s.db.UpdateUser(ctx, user)
}

// HardDeleteUserInput defines the input for hard deleting a user.
type HardDeleteUserInput struct {
	UserID int64
}

// Sanitize performs no sanitization for this input.
func (in *HardDeleteUserInput) Sanitize() {}

// Validate validates the hard delete user input against defined rules.
func (in *HardDeleteUserInput) Validate() error {
	return validateInput(hardDeleteUserInputRules, in)
}

// ProlongUserSession updates the expiration time of an existing user session in the database.
func (s *Service) ProlongUserSession(ctx context.Context, id ds.ID) (err error) {
	ctx, span := s.tracer.Start(ctx, "ProlongUserSession")
	defer span.End()

	in := &ProlongUserSessionInput{ID: id}
	err = Normalize(in)
	if err != nil {
		return
	}

	return s.db.ProlongUserSession(ctx, id)
}

// ProlongUserSessionInput defines the input for prolonging a user session.
type ProlongUserSessionInput struct {
	ID ds.ID
}

// Sanitize performs no sanitization for this input.
func (in *ProlongUserSessionInput) Sanitize() {}

// Validate validates the prolong session input (no validation rules defined).
func (in *ProlongUserSessionInput) Validate() error {
	return nil
}

// RegisterUser handles the complete user registration process.
func (s *Service) RegisterUser(ctx context.Context, username, emailAddr, password string) (user *ds.User, err error) {
	ctx, span := s.tracer.Start(ctx, "RegisterUser")
	defer span.End()

	in := &RegisterUserInput{
		Username: username,
		Email:    emailAddr,
		Password: password,
	}
	err = Normalize(in)
	if err != nil {
		return
	}

	_, err = s.db.GetUserByEmail(ctx, in.Email)
	if err == nil {
		err = app.InputError{"email": UserWithThisEmailAlreadyExists}
		return
	}
	if !errors.Is(err, repo.ErrUserNotFound) {
		return
	}

	_, err = s.db.GetUserByUsername(ctx, in.Username)
	if err == nil {
		err = app.InputError{"username": UsernameAlreadyTaken}
		return
	}
	if !errors.Is(err, repo.ErrUserNotFound) {
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(in.Password), app.DefaultBCryptCost)
	if err != nil {
		return
	}

	user = &ds.User{
		Username:       in.Username,
		Email:          in.Email,
		Password:       string(passwordHash),
		EmailConfirmed: false,
		CreatedAt:      time.Now(),
	}

	err = s.db.CreateUser(ctx, user)
	if err != nil {
		return
	}

	emailConfirmCode, err := s.CreateEmailConfirmation(ctx, user.ID)
	if err != nil {
		return
	}

	err = email.Send(user.Email, email.ConfirmEmail{
		Username: user.Username,
		Email:    in.Email,
		Code:     emailConfirmCode,
	})
	return
}

// RegisterUserInput defines the expected input parameters for the user registration process.
type RegisterUserInput struct {
	Username string
	Email    string
	Password string //nolint:gosec
}

// Sanitize trims whitespace from all registration fields.
func (in *RegisterUserInput) Sanitize() {
	in.Username = strings.TrimSpace(in.Username)
	in.Email = strings.TrimSpace(in.Email)
	in.Password = strings.TrimSpace(in.Password)
}

// Validate validates the registration input against defined rules.
func (in *RegisterUserInput) Validate() error {
	return validateInput(registerUserInputRules, in)
}

// ResetPassword handles the logic for resetting a user's password using a token.
func (s *Service) ResetPassword(ctx context.Context, token, password string) (err error) {
	ctx, span := s.tracer.Start(ctx, "ResetPassword")
	defer span.End()

	in := &ResetPasswordInput{
		Token:    token,
		Password: password,
	}
	err = Normalize(in)
	if err != nil {
		return
	}

	prt, err := s.db.GetPasswordResetToken(ctx, in.Token)
	if errors.Is(err, repo.ErrPasswordResetTokenNotFound) {
		err = ErrInvalidPasswordResetToken
		return
	}
	if err != nil {
		return
	}
	if prt.Invalid() {
		err = ErrInvalidPasswordResetToken
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(in.Password), app.DefaultBCryptCost)
	if err != nil {
		return err
	}

	err = s.db.UpdateUserPassword(ctx, prt.UserID, string(passwordHash))
	if err != nil {
		return err
	}

	return s.db.DeletePasswordResetToken(ctx, prt.ID)
}

// ResetPasswordInput defines the input for resetting a password.
type ResetPasswordInput struct {
	Token    string
	Password string //nolint:gosec
}

// Sanitize trims whitespace from token and password fields.
func (in *ResetPasswordInput) Sanitize() {
	in.Token = strings.TrimSpace(in.Token)
	in.Password = strings.TrimSpace(in.Password)
}

// Validate validates the password reset input against defined rules.
func (in *ResetPasswordInput) Validate() error {
	return validateInput(resetPasswordInputRules, in)
}

var resolveUserFromOAuthAccount = z.Shape{
	"Email":    emailInputRules,
	"Provider": z.String().Required(z.Message("provider is required")),
	"UserID":   z.String().Required(z.Message("user_id is required")),
}

// ResolveUserFromOAuthAccount attempts to find an existing user associated with an OAuth provider.
// 1. If the OAuth account exists, it returns the linked user.
// 2. If the OAuth account is missing but the email exists, it links the OAuth account to that user.
// 3. If neither exists, it creates a new user (with a unique username) and links the OAuth account.
func (s *Service) ResolveUserFromOAuthAccount(ctx context.Context, authAcc goth.User) (user *ds.User, err error) {
	ctx, span := s.tracer.Start(ctx, "ResolveUserFromOAuthAccount")
	defer span.End()

	in := &AuthenticateOAuthUserInput{&authAcc}
	err = Normalize(in)
	if err != nil {
		return
	}

	acc, err := s.db.GetOAuthUserAccount(ctx, provider.New(in.Provider), in.UserID)
	if err == nil {
		return s.db.GetUserByID(ctx, acc.UserID)
	}
	if !errors.Is(err, repo.ErrOAuthUserAccountNotFound) {
		return
	}

	err = s.db.WithTx(ctx, func(ctx context.Context) error {
		// create new oauth account
		user, err = s.GetUserByEmail(ctx, in.Email)
		if err != nil && !errors.Is(err, repo.ErrUserNotFound) {
			return err
		}

		// create new user
		if errors.Is(err, repo.ErrUserNotFound) {
			username, err := s.selectUsernameForOAuthUser(ctx, in.NickName, in.Name, in.Email)
			if err != nil {
				return err
			}

			user = &ds.User{
				ID:             ds.NewID(),
				Username:       username,
				Email:          in.Email,
				EmailConfirmed: true,
				Password:       random.String(32), //nolint:mnd
				CreatedAt:      time.Now(),
				UpdatedAt:      nil,
				DeletedAt:      nil,
			}

			err = s.db.CreateUser(ctx, user)
			if err != nil {
				return err
			}
		}

		acc = &ds.OAuthUserAccount{
			ID:             ds.NewID(),
			UserID:         user.ID,
			Provider:       provider.New(in.Provider),
			ProviderUserID: in.UserID,
			CreatedAt:      time.Now(),
		}

		err = s.CreateOAuthUserAccount(ctx, acc)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return user, nil
}

// ResolveUserFromOAuthAccountInput defines the input for resolving a user from an OAuth account.
type ResolveUserFromOAuthAccountInput struct {
	*goth.User
}

// Sanitize trims whitespace from OAuth user fields.
func (in *ResolveUserFromOAuthAccountInput) Sanitize() {
	in.UserID = strings.TrimSpace(in.UserID)
	in.Provider = strings.TrimSpace(in.Provider)
	in.Email = strings.TrimSpace(in.Email)
}

// Validate validates the OAuth user input against defined rules.
func (in *ResolveUserFromOAuthAccountInput) Validate() error {
	return validateInput(resolveUserFromOAuthAccount, in)
}

// selectUsernameForOAuthUser picks the first available username from the provided candidates.
// If all candidates are taken, it appends random suffixes and retries until a unique name is found.
func (s *Service) selectUsernameForOAuthUser(ctx context.Context, maybeNames ...string) (username string, err error) {
	const atSign = "@"
	names := make([]string, 0, len(maybeNames))

	for _, n := range maybeNames {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}

		if ii := strings.Index(n, atSign); ii > -1 {
			n = n[:ii]
		}
		names = append(names, n)
	}

	if len(names) == 0 {
		names = append(names, random.String(6)) //nolint:mnd
	}

	// The first candidate that isn't found in the DB is returned immediately
	for {
		for _, name := range names {
			_, err := s.db.GetUserByUsername(ctx, name)
			if errors.Is(err, repo.ErrUserNotFound) {
				return name, nil
			}
			if err != nil {
				return "", err
			}
		}

		// none of provided usernames is available,
		// use random one (and only one)
		names = []string{names[0] + "-" + random.String(4)} //nolint:mnd
	}
}
