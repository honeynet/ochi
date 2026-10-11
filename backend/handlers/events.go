package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/honeynet/ochi/backend/entities"

	"github.com/go-chi/chi/v5"
)

// CreateEvent creates a new event
func (h *Handlers) CreateEvent(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	var event entities.Event
	if err := decoder.Decode(&event); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	event.OwnerID = userID
	event.DstHost = nil
	var err error
	event, err = h.Events.Create(event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(event); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// DeleteEvent deletes an event making sure the user owns the event.
func (h *Handlers) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	id := chi.URLParam(r, "id")
	ownerID, err := h.Events.OwnerID(id)
	if err != nil {
		if isNotFoundError(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if userID != ownerID {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	err = h.Events.Delete(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

const (
	defaultEventsLimit = 20
	maxEventsLimit     = 100
	maxBulkDeleteIDs   = 100
)

// GetEvents returns one page of the user's events; the overall count goes in X-Total-Count.
func (h *Handlers) GetEvents(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	q := r.URL.Query()

	limit := defaultEventsLimit
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxEventsLimit {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = n
	}
	offset := 0
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			http.Error(w, "invalid offset", http.StatusBadRequest)
			return
		}
		offset = n
	}

	total, err := h.Events.CountByOwnerId(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	events, err := h.Events.FindPageByOwnerId(userID, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for i := range events {
		events[i].DstHost = nil
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(events); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) DeleteEvents(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())

	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.IDs) == 0 || len(req.IDs) > maxBulkDeleteIDs {
		http.Error(w, fmt.Sprintf("ids must contain between 1 and %d entries", maxBulkDeleteIDs), http.StatusBadRequest)
		return
	}

	deleted, err := h.Events.DeleteByOwner(userID, req.IDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(map[string]int64{"deleted": deleted}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetEventByID returns an event with the given ID.
func (h *Handlers) GetEventByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	event, err := h.Events.GetByID(id)
	if err != nil {
		if isNotFoundError(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	event.DstHost = nil
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(event); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
