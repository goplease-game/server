package service_test

import (
	"context"
	"testing"

	"github.com/markbates/goth"
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/app/session"
	"github.com/ognev-dev/goplease/oauth/provider"
	"github.com/ognev-dev/goplease/test"
	"github.com/ognev-dev/goplease/test/factory/random"
)

func TestAuthenticateOAuthUser(t *testing.T) {
	ctx := context.Background()

	newUser := func() goth.User {
		return goth.User{
			Provider: random.Element(provider.Types).String(),
			Email:    random.Email(),
			NickName: random.String(),
			UserID:   random.String(),
		}
	}

	// Case: We do not have user that associates with the OAuth account.
	// 1. New user should be created (with email_confirmed=true)
	// 2. New oauth user account linked to new user should be created
	// 3. New auth session should be created
	t.Run("new user", func(t *testing.T) {
		user := newUser()

		// make sure the user and oauth account doesn't exist yet
		test.AssertNotInDB(t, tt.DB, "users", test.Data{
			"email":    user.Email,
			"username": user.NickName,
		})
		test.AssertNotInDB(t, tt.DB, "oauth_user_accounts", test.Data{
			"provider":         user.Provider,
			"provider_user_id": user.UserID,
		})

		token, err := tt.Service.AuthenticateOAuthUser(ctx, user)
		test.CheckErr(t, err)

		newSessionID, newUserID, err := session.UnpackFromJWT(token)
		test.CheckErr(t, err)

		// check that new user created
		test.AssertInDB(t, tt.DB, "users", test.Data{
			"id":              newUserID,
			"email":           user.Email,
			"email_confirmed": true,
		})

		// check that new oauth user account created
		test.AssertInDB(t, tt.DB, "oauth_user_accounts", test.Data{
			"user_id":          newUserID,
			"provider":         user.Provider,
			"provider_user_id": user.UserID,
		})

		//  check that new session created
		test.AssertInDB(t, tt.DB, "user_sessions", test.Data{
			"id":      newSessionID,
			"user_id": newUserID,
		})
	})

	// Case: We have a user with same email as provided OAuth account, but we do not have (our) oauth account.
	// 1. New oauth user account linked to existing user should be created
	// 2. New auth session should be created
	t.Run("existing user without oauth account", func(t *testing.T) {
		oauthUser := newUser()
		user := create[ds.User](t)

		// make sure account doesn't exist yet
		test.AssertNotInDB(t, tt.DB, "oauth_user_accounts", test.Data{
			"user_id": user.ID,
		})

		token, err := tt.Service.AuthenticateOAuthUser(ctx, oauthUser)
		test.CheckErr(t, err)

		newSessionID, newUserID, err := session.UnpackFromJWT(token)
		test.CheckErr(t, err)

		// check that new oauth account created
		test.AssertInDB(t, tt.DB, "oauth_user_accounts", test.Data{
			"user_id":          newUserID,
			"provider":         oauthUser.Provider,
			"provider_user_id": oauthUser.UserID,
		})

		//  check that new session created
		test.AssertInDB(t, tt.DB, "user_sessions", test.Data{
			"id":      newSessionID,
			"user_id": newUserID,
		})
	})

	// Case: We have a user and account
	t.Run("existing user and account", func(t *testing.T) {
		oauthUser := newUser()
		user := create[ds.User](t)

		create(t, ds.OAuthUserAccount{
			UserID:         user.ID,
			Provider:       provider.New(oauthUser.Provider),
			ProviderUserID: oauthUser.UserID,
		})

		token, err := tt.Service.AuthenticateOAuthUser(ctx, oauthUser)
		test.CheckErr(t, err)

		newSessionID, newUserID, err := session.UnpackFromJWT(token)
		test.CheckErr(t, err)

		//  check that new session created
		test.AssertInDB(t, tt.DB, "user_sessions", test.Data{
			"id":      newSessionID,
			"user_id": newUserID,
		})
	})
}
