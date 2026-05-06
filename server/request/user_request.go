//nolint:gosec
package request

// UserSignUp holds the data required to register a new user.
type UserSignUp struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ConfirmEmail holds the confirmation code for email verification.
type ConfirmEmail struct {
	Code string `json:"code"`
}

// UserSignIn holds the credentials required to authenticate a user.
type UserSignIn struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ChangeUsername holds the data required to update a username.
type ChangeUsername struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// DeleteUser holds the data required to delete a user account.
type DeleteUser struct {
	Password string `json:"password"`
}
