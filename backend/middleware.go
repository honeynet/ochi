package backend

import (
	"context"
	"net/http"
	"strings"

	"github.com/honeynet/ochi/backend/entities"
	"github.com/julienschmidt/httprouter"
)

func tokenMiddleware(h httprouter.Handle, secret string) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		token, ok := r.URL.Query()["token"]
		if !ok || len(token) == 0 || token[0] != secret {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		h(w, r, ps)
	}
}

type userID string

func bearerMiddleware(h httprouter.Handle, secret string) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		authHeader := r.Header.Get("Authorization")
		authFields := strings.Fields(authHeader)
		if len(authFields) != 2 || strings.ToLower(authFields[0]) != "bearer" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
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

		r = r.WithContext(context.WithValue(r.Context(), userID("userID"), claims.UserID))

		h(w, r, ps)
	}
}
