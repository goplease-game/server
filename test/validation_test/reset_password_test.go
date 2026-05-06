package validation_test

import (
	"strings"
	"testing"

	"github.com/ognev-dev/goplease/app/service"
)

func TestValidateResetPasswordInput(t *testing.T) {
	t.Parallel()

	validPassword := strings.Repeat("a", service.UserPasswordMinLen)
	cases := []struct {
		name      string
		valid     bool
		expectErr string
		argName   string
		data      service.ResetPasswordInput
	}{
		{
			name:      "empty token",
			expectErr: "Token is required",
			argName:   "token",
			data:      service.ResetPasswordInput{"", validPassword},
		},
		{
			name:      "empty password",
			expectErr: "Password is required",
			argName:   "password",
			data:      service.ResetPasswordInput{"aaaa", ""},
		},
		{
			name:      "short password",
			expectErr: "Password must be at least 6 characters",
			argName:   "password",
			data:      service.ResetPasswordInput{"aaa", "123"},
		},
		{
			valid: true,
			name:  "valid input",
			data:  service.ResetPasswordInput{"aaa", validPassword},
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
