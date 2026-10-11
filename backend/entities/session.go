package entities

import (
	"time"

	"github.com/go-chi/jwtauth/v5"
)

// TokenTTL is how long a freshly issued session token stays valid.
const TokenTTL = 24 * time.Hour

// NewTokenAuth builds the verifier used both to sign session tokens and to
// authenticate requests, so the two can never drift apart.
func NewTokenAuth(secret string) *jwtauth.JWTAuth {
	return jwtauth.New("HS256", []byte(secret), nil)
}

// NewToken issues a session token for the user.
func NewToken(ja *jwtauth.JWTAuth, user User) (string, error) {
	claims := map[string]any{
		"user_id":    user.ID,
		"user_email": user.Email,
	}
	jwtauth.SetExpiryIn(claims, TokenTTL)

	_, tokenString, err := ja.Encode(claims)
	return tokenString, err
}
