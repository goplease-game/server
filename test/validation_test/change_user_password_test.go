package validation_test

import (
	"testing"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/app/service"
)

func TestValidateChangeUserPasswordInput(t *testing.T) {
	t.Parallel()

	id := ds.NewID()

	cases := []struct {
		name      string
		valid     bool
		expectErr string
		argName   string
		data      service.ChangeUserPasswordInput
	}{
		{
			name:      "invalid ID",
			expectErr: "Invalid UUID",
			argName:   "user_id",
			data:      service.ChangeUserPasswordInput{ds.NilID, "aaa", "bbb"},
		},
		{
			name:      "empty old password",
			expectErr: "Password is required",
			argName:   "old_password",
			data:      service.ChangeUserPasswordInput{id, "", "bbb"},
		},
		{
			name:      "empty new password",
			expectErr: "Password is required",
			argName:   "new_password",
			data:      service.ChangeUserPasswordInput{id, "aaa", ""},
		},
		{
			name:      "new password too short",
			expectErr: "Password must be at least 6 characters",
			argName:   "new_password",
			data:      service.ChangeUserPasswordInput{id, "aaa", "bbb"},
		},
		{
			valid: true,
			name:  "valid input",
			data:  service.ChangeUserPasswordInput{id, "aaa", "new-password"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			err := service.Normalize(&c.data)
			checkValidatedInput(t, c.valid, err, c.argName, c.expectErr)
		})
	}
}
