package validation_test

import (
	"testing"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/app/service"
)

func TestValidateCreateEmailConfirmationInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		valid     bool
		expectErr string
		argName   string
		data      service.CreateEmailConfirmationInput
	}{
		{
			name:      "invalid ID",
			expectErr: "Invalid UUID",
			argName:   "user_id",
			data:      service.CreateEmailConfirmationInput{ds.NilID},
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
