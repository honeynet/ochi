package handlers

import (
	"bytes"
	"context"
	"strings"

	"github.com/go-chi/jwtauth/v5"
)

func isNotFoundError(e error) bool {
	return strings.Contains(e.Error(), "no rows in result set")
}

// userIDFromCtx reads the user id from the token jwtauth.Verifier put on the
// context. It is only meaningful behind Authenticator, which rejects the request
// before a handler runs when there is no valid token.
func userIDFromCtx(ctx context.Context) string {
	_, claims, err := jwtauth.FromContext(ctx)
	if err != nil {
		return ""
	}
	userID, _ := claims["user_id"].(string)
	return userID
}

// IndexReplace substitutes the first occurrence of old with new in b.
func IndexReplace(b, old, new []byte) ([]byte, bool) {
	i := bytes.Index(b, old)
	if i == -1 {
		return b, false
	}
	copy(b[i:], new)
	return b, true
}
