package handlers

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/go-chi/jwtauth/v5"
)

func TokenMiddleware(h http.HandlerFunc, secret string) http.HandlerFunc {
	want := []byte(secret)
	return func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(publishToken(r)), want) != 1 {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		h(w, r)
	}
}

// publishToken reads the publish token, preferring forms that keep it out of the
// request line (and therefore out of proxy and intermediary access logs): HTTP
// Basic auth password, then an Authorization: Bearer header. The ?token= query
// parameter is still accepted for backward compatibility, but a token in the URL
// is logged, so the header forms should be used.
func publishToken(r *http.Request) string {
	if _, password, ok := r.BasicAuth(); ok {
		return password
	}
	const prefix = "bearer "
	if auth := r.Header.Get("Authorization"); len(auth) > len(prefix) &&
		strings.EqualFold(auth[:len(prefix)], prefix) {
		return auth[len(prefix):]
	}
	return r.URL.Query().Get("token")
}

// Authenticator rejects any request that jwtauth.Verifier could not authenticate.
//
// It is the jwtauth.Authenticator flow with one change: the reason a token was
// refused is not written to the response. The upstream version passes err.Error()
// as the body, which tells an unauthenticated caller whether a token merely
// expired, was signed with the wrong key, or was not a JWT at all.
func Authenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, _, err := jwtauth.FromContext(r.Context())
		if err != nil || token == nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
