package handlers

import (
	"net/http"

	"github.com/go-chi/jwtauth/v5"
)

func TokenMiddleware(h http.HandlerFunc, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := r.URL.Query()["token"]
		if !ok || len(token) == 0 || token[0] != secret {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		h(w, r)
	}
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
