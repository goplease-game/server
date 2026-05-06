package request

// PasswordResetRequest represents the request body for initiating a password reset.
type PasswordResetRequest struct {
	Email string `json:"email"`
}

// PasswordReset represents the request body for resetting a password with a token.
type PasswordReset struct {
	Token    string `json:"token"`
	Password string `json:"password"` //nolint:gosec
}
