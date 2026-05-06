//nolint:dupl
package validation_test

import (
	"testing"

	"github.com/markbates/goth"
	"github.com/ognev-dev/goplease/app/service"
	"github.com/ognev-dev/goplease/test/factory/random"
)

func TestResolveUserFromOAuthAccount(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		valid     bool
		expectErr string
		argName   string
		data      service.ResolveUserFromOAuthAccountInput
	}{
		{
			name:      "missing user id",
			expectErr: "user_id is required",
			argName:   "user_id",
			data: service.ResolveUserFromOAuthAccountInput{&goth.User{
				UserID:   "  ",
				Provider: "google",
				Email:    random.Email(),
			}},
		},
		{
			name:      "missing provider",
			expectErr: "provider is required",
			argName:   "provider",
			data: service.ResolveUserFromOAuthAccountInput{&goth.User{
				UserID:   "123",
				Provider: "  ",
				Email:    random.Email(),
			}},
		},
		{
			name:      "missing email",
			expectErr: "Email is required",
			argName:   "email",
			data: service.ResolveUserFromOAuthAccountInput{&goth.User{
				UserID:   "123",
				Provider: "google",
				Email:    "  ",
			}},
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
