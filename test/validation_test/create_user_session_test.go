package validation_test

import (
	"testing"

	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/app/service"
)

func TestValidateCreateUserSessionInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		valid     bool
		expectErr string
		argName   string
		data      service.CreateUserSessionInput
	}{
		{
			name:      "invalid userID",
			expectErr: "Invalid UUID",
			argName:   "user_id",
			data:      service.CreateUserSessionInput{ds.NilID},
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
