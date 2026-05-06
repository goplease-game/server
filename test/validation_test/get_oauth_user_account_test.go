package validation_test

import (
	"testing"

	"github.com/ognev-dev/goplease/app/service"
	"github.com/ognev-dev/goplease/oauth/provider"
)

func TestGetOAuthUserAccount(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		valid     bool
		expectErr string
		argName   string
		data      service.GetOAuthUserAccountInput
	}{
		{
			name:      "invalid provider",
			expectErr: "Invalid provider type",
			argName:   "provider",
			data: service.GetOAuthUserAccountInput{
				Provider:       provider.New("invalid"),
				ProviderUserID: "123",
			},
		},
		{
			name:      "empty provider user id",
			expectErr: "provider_user_id is required",
			argName:   "provider_user_id",
			data: service.GetOAuthUserAccountInput{
				Provider:       provider.GitHub,
				ProviderUserID: "   ",
			},
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
