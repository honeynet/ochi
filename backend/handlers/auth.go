package handlers

import (
	"crypto/subtle"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/jwtauth/v5"
)

// warnQueryTokenOnce keeps the deprecation notice below to one line per process, so a
// sensor still using the query form does not flood the log on every publish.
var warnQueryTokenOnce sync.Once

func TokenMiddleware(h http.HandlerFunc, secret string) http.HandlerFunc {
	want := []byte(secret)
	return func(w http.ResponseWriter, r *http.Request) {
		// Fail closed when the server has no publish token. config.go rejects an empty
		// publish_token at load time, but the middleware does not rely on that: an empty
		// want would make subtle.ConstantTimeCompare("", "") report a match and let an
		// unauthenticated request through.
		if len(want) == 0 || !tokenMatches(r, want) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		h(w, r)
	}
}

// tokenMatches reports whether the request carries the publish token. Only the first
// form present is compared, in this order: the HTTP Basic password, an Authorization:
// Bearer header, then the ?token= query parameter. The two header forms keep the token
// out of the request line. Comparing a single form gives each request one guess, and
// since Basic and Bearer share the one Authorization header, a request never has both.
func tokenMatches(r *http.Request, want []byte) bool {
	if _, password, ok := r.BasicAuth(); ok {
		return secretEqual(password, want)
	}
	if bearer, ok := bearerToken(r); ok {
		return secretEqual(bearer, want)
	}
	query := r.URL.Query().Get("token")
	if query == "" || !secretEqual(query, want) {
		return false
	}
	// The query form ends up in access logs, which is the leak this handler exists to
	// close. It still authenticates so existing sensors keep working, but say once that
	// it should be replaced with the Basic or Bearer form.
	warnQueryTokenOnce.Do(func() {
		log.Print("publish token accepted from the ?token= query parameter; it is written to request lines and proxy logs, so move senders to HTTP Basic userinfo or an Authorization: Bearer header")
	})
	return true
}

// bearerToken returns the value of an Authorization: Bearer header, if present and
// non-empty. The scheme is matched case-insensitively, as RFC 7235 requires.
func bearerToken(r *http.Request) (string, bool) {
	const prefix = "bearer "
	auth := r.Header.Get("Authorization")
	if len(auth) > len(prefix) && strings.EqualFold(auth[:len(prefix)], prefix) {
		return auth[len(prefix):], true
	}
	return "", false
}

// secretEqual compares a candidate against the secret in constant time.
func secretEqual(got string, want []byte) bool {
	return subtle.ConstantTimeCompare([]byte(got), want) == 1
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
