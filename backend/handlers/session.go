package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/honeynet/ochi/backend/entities"

	"google.golang.org/api/idtoken"
)

type response struct {
	User        entities.User         `json:"user,omitempty"`
	Token       string                `json:"token,omitempty"`
	Permissions []entities.Permission `json:"permissions"`
}

// Session creates a new token for the user
func (h *Handlers) Session(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	user, err := h.Users.Get(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user.Roles, err = h.syncRoles(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := entities.NewToken(h.TokenAuth, user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(response{user, token, entities.PermissionsFor(user.Roles)}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Login validates a token with Google
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	body := http.MaxBytesReader(w, r.Body, 8192)
	data, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		return
	}

	ctx := context.Background()
	val, err := idtoken.NewValidator(ctx, idtoken.WithHTTPClient(h.HTTPClient))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	payload, err := val.Validate(ctx, string(data), "610036027764-0lveoeejd62j594aqab5e24o2o82r8uf.apps.googleusercontent.com")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var user entities.User
	if emailInt, ok := payload.Claims["email"]; ok {
		if email, ok := emailInt.(string); ok {
			user, err = h.Users.Find(email)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
	}
	user.Roles, err = h.syncRoles(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := entities.NewToken(h.TokenAuth, user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(response{user, token, entities.PermissionsFor(user.Roles)}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
