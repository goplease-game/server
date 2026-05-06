package request

// EmailChangeRequest represents the request body for initiating an email change.
type EmailChangeRequest struct {
	Email string `json:"email"`
}

// EmailChangeConfirm represents the request body for confirming an email change.
type EmailChangeConfirm struct {
	Token string `json:"token"`
}
