package handlers

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenMiddleware(t *testing.T) {
	const secret = "s3cr3t-publish-token"
	reached := false
	h := TokenMiddleware(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusAccepted)
	}, secret)

	run := func(setup func(*http.Request)) *httptest.ResponseRecorder {
		reached = false
		r := httptest.NewRequest(http.MethodPost, "/api/publish", nil)
		setup(r)
		w := httptest.NewRecorder()
		h(w, r)
		return w
	}

	t.Run("accepts Basic auth password (what Glutton sends via userinfo)", func(t *testing.T) {
		w := run(func(r *http.Request) { r.SetBasicAuth("sensor", secret) })
		assert.Equal(t, http.StatusAccepted, w.Code)
		assert.True(t, reached)
	})

	t.Run("accepts Authorization Bearer", func(t *testing.T) {
		w := run(func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+secret) })
		assert.Equal(t, http.StatusAccepted, w.Code)
	})

	t.Run("accepts a lowercase bearer scheme", func(t *testing.T) {
		// RFC 7235 makes the scheme case-insensitive.
		w := run(func(r *http.Request) { r.Header.Set("Authorization", "bearer "+secret) })
		assert.Equal(t, http.StatusAccepted, w.Code)
	})

	t.Run("still accepts the query parameter for back-compat", func(t *testing.T) {
		w := run(func(r *http.Request) { r.URL.RawQuery = "token=" + secret })
		assert.Equal(t, http.StatusAccepted, w.Code)
	})

	t.Run("compares only the first form present, so each request gets one guess", func(t *testing.T) {
		// Once the header form is wrong, a valid ?token= is not tried as a second guess.
		for name, header := range map[string]func(*http.Request){
			"Basic":  func(r *http.Request) { r.SetBasicAuth("sensor", "nope") },
			"Bearer": func(r *http.Request) { r.Header.Set("Authorization", "Bearer nope") },
		} {
			w := run(func(r *http.Request) {
				header(r)
				r.URL.RawQuery = "token=" + secret
			})
			assert.Equal(t, http.StatusForbidden, w.Code, "wrong %s with a valid query token", name)
			assert.False(t, reached)
		}

		// The first form decides the other way too: a valid Basic password passes even
		// when a wrong query token comes with it.
		w := run(func(r *http.Request) {
			r.SetBasicAuth("sensor", secret)
			r.URL.RawQuery = "token=nope"
		})
		assert.Equal(t, http.StatusAccepted, w.Code)
	})

	t.Run("rejects a wrong token in each form", func(t *testing.T) {
		for _, setup := range []func(*http.Request){
			func(r *http.Request) { r.SetBasicAuth("sensor", "nope") },
			func(r *http.Request) { r.Header.Set("Authorization", "Bearer nope") },
			func(r *http.Request) { r.URL.RawQuery = "token=nope" },
		} {
			w := run(setup)
			assert.Equal(t, http.StatusForbidden, w.Code)
			assert.False(t, reached)
		}
	})

	t.Run("rejects a missing token", func(t *testing.T) {
		w := run(func(*http.Request) {})
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("rejects an empty Bearer value", func(t *testing.T) {
		w := run(func(r *http.Request) { r.Header.Set("Authorization", "Bearer ") })
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.False(t, reached)
	})

	t.Run("does not accept the token as the Basic username", func(t *testing.T) {
		// Only the password carries the secret; a bare userinfo must not authenticate.
		cred := base64.StdEncoding.EncodeToString([]byte(secret + ":"))
		w := run(func(r *http.Request) { r.Header.Set("Authorization", "Basic "+cred) })
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

// An empty secret must fail closed rather than let an empty comparison match. config.go
// rejects an empty publish_token at load time; this guards the middleware regardless.
func TestTokenMiddleware_EmptySecretRejectsEverything(t *testing.T) {
	reached := false
	h := TokenMiddleware(func(http.ResponseWriter, *http.Request) {
		reached = true
	}, "")

	for name, setup := range map[string]func(*http.Request){
		"no credentials":   func(*http.Request) {},
		"empty Basic":      func(r *http.Request) { r.SetBasicAuth("", "") },
		"empty query":      func(r *http.Request) { r.URL.RawQuery = "token=" },
		"empty Bearer":     func(r *http.Request) { r.Header.Set("Authorization", "Bearer ") },
		"a guessed secret": func(r *http.Request) { r.URL.RawQuery = "token=secret" },
	} {
		t.Run(name, func(t *testing.T) {
			reached = false
			r := httptest.NewRequest(http.MethodPost, "/api/publish", nil)
			setup(r)
			w := httptest.NewRecorder()
			h(w, r)
			assert.Equal(t, http.StatusForbidden, w.Code)
			assert.False(t, reached)
		})
	}
}
