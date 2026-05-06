package request

// ChangePassword represents the request body for changing a user's password.
type ChangePassword struct {
	OldPassword string `json:"old_password" z:"OldPassword"`
	NewPassword string `json:"new_password" z:"NewPassword"`
}
