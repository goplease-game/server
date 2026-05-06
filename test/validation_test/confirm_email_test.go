package validation_test

import (
	"testing"

	"github.com/ognev-dev/goplease/app/service"
)

func TestValidateConfirmEmailInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		valid     bool
		expectErr string
		argName   string
		data      service.ConfirmEmailInput
	}{
		{
			name:      "empty code",
			expectErr: "Code is required",
			argName:   "code",
			data:      service.ConfirmEmailInput{""},
		},
		{
			name:  "valid code",
			valid: true,
			data:  service.ConfirmEmailInput{"some-valid-code"},
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
