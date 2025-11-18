package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/honeynet/ochi/backend/entities"
	"github.com/honeynet/ochi/backend/services"
)

type QueryHandler struct {
	service *services.QueryService
}

func NewQueryHandler(service *services.QueryService) *QueryHandler {
	return &QueryHandler{service: service}
}

// CreateQueryHandler handles creating a new query
func (h *QueryHandler) CreateQueryHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OwnerID     string         `json:"owner_id"`
		Content     string         `json:"content"`
		Description string         `json:"description"`
		Active      bool           `json:"active"`
		Tags        []entities.Tag `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	q, err := h.service.CreateQuery(req.OwnerID, req.Content, req.Description, req.Active, req.Tags)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(q)
}

// GetQueryByIDHandler returns a query by ID
func (h *QueryHandler) GetQueryByIDHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	q, err := h.service.GetQueryByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(q)
}

// UpdateQueryHandler updates a query
func (h *QueryHandler) UpdateQueryHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var req struct {
		Content     string         `json:"content"`
		Description string         `json:"description"`
		Active      bool           `json:"active"`
		Tags        []entities.Tag `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateQuery(id, req.Content, req.Description, req.Active, req.Tags); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteQueryHandler deletes a query
func (h *QueryHandler) DeleteQueryHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if err := h.service.DeleteQuery(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// FindQueriesByOwnerHandler returns queries by owner
func (h *QueryHandler) FindQueriesByOwnerHandler(w http.ResponseWriter, r *http.Request) {
	ownerID := mux.Vars(r)["owner_id"]

	qs, err := h.service.FindQueriesByOwner(ownerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(qs)
}
