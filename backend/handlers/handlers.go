package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/honeynet/ochi/backend/entities"
	"github.com/honeynet/ochi/backend/repos"

	"github.com/julienschmidt/httprouter"
	"google.golang.org/api/idtoken"
)

// Handlers holds the dependencies shared by the HTTP handlers.
type Handlers struct {
	FS         fs.FS
	JWTSecret  string
	HTTPClient *http.Client

	Users   *repos.UserRepo
	Queries *repos.QueryRepo
	Events  *repos.EventRepo
	Sensors *repos.SensorRepo

	// Publish fans a message out to subscribers; it returns false when rate limited.
	Publish func(msg []byte) bool
}

func (h *Handlers) IndexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fh, err := h.FS.Open("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(w, fh); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) CSSHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fh, err := h.FS.Open("global.css")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "text/css")
	if _, err := io.Copy(w, fh); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

const PublishMaxBodyBytes = 2 << 20 // 2 MiB; multi-frame decoded sessions exceed 8 KiB

// PublishHandler reads the request body with a limit of 2 MiB and then publishes
// the received message. sensorID is truncated to 8 characters for display.
// dstHost (honeypot sensor IP) is stripped so it is never sent to subscribers.
func (h *Handlers) PublishHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	body := http.MaxBytesReader(w, r.Body, PublishMaxBodyBytes)
	msg, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		return
	}

	// Use map[string]any so numeric fields (dstPort) and nested objects (decoded) survive.
	var event map[string]any
	if err := json.Unmarshal(msg, &event); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sensorID, ok := event["sensorID"].(string)
	if !ok || sensorID == "" {
		http.Error(w, "sensor id does not exists", http.StatusBadRequest)
		return
	}
	if len(sensorID) < 8 {
		http.Error(w, "sensor id must have at least 8 characters", http.StatusBadRequest)
		return
	}
	event["sensorID"] = sensorID[:8]
	delete(event, "dstHost")

	alteredMsg, err := json.Marshal(event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !h.Publish(alteredMsg) {
		http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

type response struct {
	User  entities.User `json:"user,omitempty"`
	Token string        `json:"token,omitempty"`
}

// SessionHandler creates a new token for the user
func (h *Handlers) SessionHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	userID := userIDFromCtx(r.Context())
	user, err := h.Users.Get(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := entities.NewToken(h.JWTSecret, user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(response{user, token}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// LoginHandler validates a token with Google
func (h *Handlers) LoginHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

	token, err := entities.NewToken(h.JWTSecret, user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(response{user, token}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// query handlers

// GetQueriesHandler returns a list of queries belonging to ther user.
func (h *Handlers) GetQueriesHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

// CreateQueryHandler creates a new query.
func (h *Handlers) CreateQueryHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	userID := userIDFromCtx(r.Context())
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	var t entities.Query
	err := decoder.Decode(&t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
func (h *Handlers) UpdateQueryHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	userID := userIDFromCtx(r.Context())
	id := p.ByName("id")
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
		http.Error(w, "Unauthorized", http.StatusInternalServerError)
		return
	}
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err = decoder.Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

// DeleteQueryHandler deletes a query making sure the user owns the query.
func (h *Handlers) DeleteQueryHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	userID := userIDFromCtx(r.Context())
	id := p.ByName("id")
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
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	err = h.Queries.Delete(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// event handlers

// CreateEventHandler creates a new event
func (h *Handlers) CreateEventHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

// DeleteEventHandler deletes an event making sure the user owns the event.
func (h *Handlers) DeleteEventHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	userID := userIDFromCtx(r.Context())
	id := p.ByName("id")
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

// GetEventsHandler returns one page of the user's events; the overall count goes in X-Total-Count.
func (h *Handlers) GetEventsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

func (h *Handlers) DeleteEventsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

// GetEventByIDHandler returns an event with the given ID.
func (h *Handlers) GetEventByIDHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("id")

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

func (h *Handlers) GetSensorsByUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	userId := userIDFromCtx(r.Context())
	events, err := h.Sensors.GetSensorsByOwnerId(userId)
	if err != nil {
		if isNotFoundError(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	if err = json.NewEncoder(w).Encode(events); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) AddSensor(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	userId := userIDFromCtx(r.Context())
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	var sensor entities.Sensor
	if err := decoder.Decode(&sensor); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sensor.UserID = userId

	if err := h.Sensors.AddSensors(sensor); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(sensor); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// DownloadBinaryHandler serves the binary for the requested architecture with a new sensor UUID injected.
func (h *Handlers) DownloadBinaryHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	osType := p.ByName("os")
	arch := p.ByName("arch")

	validOS := map[string]bool{
		"linux":   true,
		"darwin":  true,
		"windows": true,
		"openbsd": true,
	}

	validArch := map[string]bool{
		"amd64": true,
		"arm64": true,
		"386":   true,
	}

	if !validOS[osType] || !validArch[arch] {
		http.Error(w, "Invalid parameters", http.StatusBadRequest)
		return
	}

	binaryPath := "bin/sensor-" + osType + "-" + arch

	data, err := os.ReadFile(binaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Binary not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newUUID := uuid.New().String()
	placeholderUUID := "00000000-0000-0000-0000-000000000000"

	modifiedData, ok := IndexReplace(data, []byte(placeholderUUID), []byte(newUUID))
	if !ok {
		http.Error(w, "Placeholder UUID not found in binary", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\"sensor-"+osType+"-"+arch+"\"")

	if _, err := w.Write(modifiedData); err != nil {
		log.Printf("failed to write binary: %v", err)
	}
}
