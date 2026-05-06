package validation_test

import (
	"strings"
	"testing"

	"github.com/ognev-dev/goplease/app/service"
)

func TestValidateRegisterUserInput(t *testing.T) {
	t.Parallel()

	validUsername := strings.Repeat("a", service.UsernameMinLen)
	validEmail := "mail@ognev.dev"
	validPassword := strings.Repeat("a", service.UserPasswordMinLen)
	cases := []struct {
		name      string
		valid     bool
		expectErr string
		argName   string
		data      service.RegisterUserInput
	}{
		{
			name:      "empty username",
			expectErr: "Username is required",
			argName:   "username",
			data:      service.RegisterUserInput{"", validEmail, validPassword},
		},
		{
			name:      "username too short",
			expectErr: "Username must be at least 2 characters",
			argName:   "username",
			data:      service.RegisterUserInput{strings.Repeat("a", service.UsernameMinLen-1), validEmail, validPassword},
		},
		{
			name:      "username too long",
			expectErr: "Username must be at most 30 characters",
			argName:   "username",
			data:      service.RegisterUserInput{strings.Repeat("a", service.UsernameMaxLen+1), validEmail, validPassword},
		},
		{
			name:      "username contains forbidden chars",
			expectErr: "Username can only contain letters, numbers, dots, underscores, and dashes",
			argName:   "username",
			data:      service.RegisterUserInput{"a@?$", validEmail, validPassword},
		},
		{
			name:      "username contains more than two special chars",
			expectErr: "Username cannot contain more than two dots, underscores, or dashes",
			argName:   "username",
			data:      service.RegisterUserInput{"a.a.a.a", validEmail, validPassword},
		},
		{
			name:      "invalid email",
			expectErr: "must be a valid email",
			argName:   "email",
			data:      service.RegisterUserInput{validUsername, "aaa", validPassword},
		},
		{
			name:      "empty email",
			expectErr: "Email is required",
			argName:   "email",
			data:      service.RegisterUserInput{validUsername, "", validPassword},
		},
		{
			name:      "empty email having whitespace",
			expectErr: "Email is required",
			argName:   "email",
			data:      service.RegisterUserInput{validUsername, "          ", validPassword},
		},
		{
			name:      "empty password",
			expectErr: "Password is required",
			argName:   "password",
			data:      service.RegisterUserInput{validUsername, validEmail, ""},
		},
		{
			name:      "short password",
			expectErr: "Password must be at least 6 characters",
			argName:   "password",
			data:      service.RegisterUserInput{validUsername, validEmail, "123"},
		},
		{
			valid: true,
			name:  "valid input",
			data:  service.RegisterUserInput{validUsername, validEmail, validPassword},
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
