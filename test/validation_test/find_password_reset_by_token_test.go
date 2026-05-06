package validation_test

import (
	"testing"

	"github.com/ognev-dev/goplease/app/service"
)

func TestValidateFindPasswordResetByTokenInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		valid     bool
		expectErr string
		argName   string
		data      service.FindPasswordResetByTokenInput
	}{
		{
			name:      "empty token",
			expectErr: "Token is required",
			argName:   "token",
			data:      service.FindPasswordResetByTokenInput{""},
		},
		{
			valid: true,
			name:  "valid token",
			data:  service.FindPasswordResetByTokenInput{"valid-token"},
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
