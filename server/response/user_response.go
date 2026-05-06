package response

import "github.com/ognev-dev/goplease/app/ds"

// UserSignIn contains user data returned after successful authentication.
type UserSignIn struct {
	ID       ds.ID  `json:"id"`
	Username string `json:"username"`
	Token    string `json:"token"`
}
