package validation_test

import (
	"testing"

	"github.com/ognev-dev/goplease/app/service"
)

func TestValidateFindUserByEmailInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		valid     bool
		expectErr string
		argName   string
		data      service.FindUserByEmailInput
	}{
		{
			name:      "invalid email",
			expectErr: "must be a valid email",
			argName:   "email",
			data:      service.FindUserByEmailInput{"invalid"},
		},
		{
			name:      "empty email",
			expectErr: "Email is required",
			argName:   "email",
			data:      service.FindUserByEmailInput{""},
		},
		{
			valid: true,
			name:  "valid input",
			data:  service.FindUserByEmailInput{"mail@ognev.dev"},
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
