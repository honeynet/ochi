package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"strings"

	"github.com/honeynet/ochi/backend/entities"
)

// RequirePermission rejects with 403 any caller whose roles do not grant perm.
// It must run behind Authenticator. Roles are read from the database on every
// request, not from the token, so granting or revoking one takes effect at once.
func (h *Handlers) RequirePermission(perm entities.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ok, err := h.hasPermission(r, perm)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if !ok {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// hasPermission reports whether the authenticated caller holds perm.
func (h *Handlers) hasPermission(r *http.Request, perm entities.Permission) (bool, error) {
	roles, err := h.Users.Roles(userIDFromCtx(r.Context()))
	if err != nil {
		return false, err
	}
	return entities.HasPermission(roles, perm), nil
}

// syncRoles grants the admin role to a user listed in AdminEmails and returns
// the user's roles.
func (h *Handlers) syncRoles(user entities.User) ([]entities.Role, error) {
	if user.Email != "" && slices.ContainsFunc(h.AdminEmails, func(e string) bool {
		return strings.EqualFold(strings.TrimSpace(e), user.Email)
	}) {
		if err := h.Users.GrantRole(user.ID, entities.RoleAdmin); err != nil {
			return nil, err
		}
	}
	return h.Users.Roles(user.ID)
}

// ListUsers returns every registered user with their roles.
func (h *Handlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.Users.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(users); err != nil {
		log.Printf("failed to write users: %v", err)
	}
}
