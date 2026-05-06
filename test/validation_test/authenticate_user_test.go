package validation_test

import (
	"testing"

	"github.com/ognev-dev/goplease/app/service"
)

func TestValidateAuthenticateUserInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		valid     bool
		expectErr string
		argName   string
		data      service.AuthenticateUserInput
	}{
		{
			name:      "empty email",
			expectErr: "Email is required",
			argName:   "email",
			data:      service.AuthenticateUserInput{"", "bbb"},
		},
		{
			name:      "empty email having whitespace",
			expectErr: "Email is required",
			argName:   "email",
			data:      service.AuthenticateUserInput{"          ", "bbb"},
		},
		{
			name:      "empty password",
			expectErr: "Password is required",
			argName:   "password",
			data:      service.AuthenticateUserInput{"mail@ognev.dev", ""},
		},
		{
			valid: true,
			name:  "valid input",
			data:  service.AuthenticateUserInput{"mail@ognev.dev", "123"},
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
