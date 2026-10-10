package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/honeynet/ochi/backend/entities"
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

type UserID string

func BearerMiddleware(h http.HandlerFunc, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		authFields := strings.Fields(authHeader)
		if len(authFields) != 2 || strings.ToLower(authFields[0]) != "bearer" {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		token := authFields[1]

		claims, valid, err := entities.ValidateToken(token, secret)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !valid {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), UserID("userID"), claims.UserID))

		h(w, r)
	}
}
