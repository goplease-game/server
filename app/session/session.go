// Package session provides primitives for managing user sessions using JSON Web Tokens (JWT).
package session

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/ds"
)

var (
	jwtSessionParam = "session"
	jwtUserParam    = "user"
)

var (
	// ErrInvalidJWT is returned when an authentication token is malformed,
	// invalidly signed, or contains unexpected claims.
	ErrInvalidJWT = app.ErrForbidden("invalid token")
)

// NewSignedJWT creates a new, signed JWT token containing the session ID and user ID claims.
// The token is signed using the secret key from the application configuration.
func NewSignedJWT(sessionID, userID ds.ID) (token string, err error) {
	jt := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			jwtSessionParam: sessionID,
			jwtUserParam:    userID,
		})

	return jt.SignedString([]byte(app.Config().Session.Key))
}

// UnpackFromJWT validates and parses a signed JWT string.
// It extracts the session ID (string) and user ID (int64) from the claims.
func UnpackFromJWT(jt string) (sessionID, userID ds.ID, err error) {
	token, err := jwt.Parse(jt, func(_ *jwt.Token) (any, error) {
		return []byte(app.Config().Session.Key), nil
	})
	if err != nil {
		return
	}

	if !token.Valid {
		err = ErrInvalidJWT
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		err = ErrInvalidJWT
		return
	}

	sessionIDStr, ok := claims[jwtSessionParam].(string)
	if !ok {
		err = ErrInvalidJWT
		return
	}

	sessionID, err = ds.ParseID(sessionIDStr)
	if err != nil {
		err = ErrInvalidJWT
		return
	}

	userIDStr, ok := claims[jwtUserParam].(string)
	if !ok {
		err = ErrInvalidJWT
		return
	}

	userID, err = ds.ParseID(userIDStr)
	if err != nil {
		err = ErrInvalidJWT
		return
	}

	return
}
