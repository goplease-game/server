package validation_test

import (
	"testing"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/app/service"
)

func TestValidateCreateChangeEmailRequestInput(t *testing.T) {
	t.Parallel()
	id := ds.NewID()

	cases := []struct {
		name      string
		valid     bool
		expectErr string
		argName   string
		data      service.CreateChangeEmailRequestInput
	}{
		{
			name:      "invalid ID",
			expectErr: "Invalid UUID",
			argName:   "user_id",
			data:      service.CreateChangeEmailRequestInput{ds.NilID, "mail@ognev.dev"},
		},
		{
			name:      "empty email",
			expectErr: "Email is required",
			argName:   "new_email",
			data:      service.CreateChangeEmailRequestInput{id, ""},
		},
		{
			name:      "invalid email",
			expectErr: "must be a valid email",
			argName:   "new_email",
			data:      service.CreateChangeEmailRequestInput{id, "aaa"},
		},
		{
			valid: true,
			name:  "valid input",
			data:  service.CreateChangeEmailRequestInput{id, "mail@ognev.dev"},
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
