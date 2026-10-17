package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/honeynet/ochi/backend/entities"

	"github.com/go-chi/chi/v5"
)

// GetQueries returns a list of queries belonging to ther user.
func (h *Handlers) GetQueries(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	queries, err := h.Queries.FindByOwnerId(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(queries); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// CreateQuery creates a new query.
func (h *Handlers) CreateQuery(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	var t entities.Query
	err := decoder.Decode(&t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	query, err := h.Queries.Create(userID, t.Content, t.Description, t.Active)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(query); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// udpateQueryHandler updates an existing query making sure the user owns the query.
func (h *Handlers) UpdateQuery(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	id := chi.URLParam(r, "id")
	q, err := h.Queries.GetByID(id)
	if err != nil {
		if isNotFoundError(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if userID != q.OwnerID {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err = decoder.Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if id != q.ID {
		http.Error(w, "Ids don't match", http.StatusBadRequest)
		return
	}
	err = h.Queries.Update(q.ID, q.Content, q.Description, q.Active)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// DeleteQuery deletes a query making sure the user owns the query.
func (h *Handlers) DeleteQuery(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	id := chi.URLParam(r, "id")
	q, err := h.Queries.GetByID(id)
	if err != nil {

		if isNotFoundError(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if userID != q.OwnerID {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	err = h.Queries.Delete(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
