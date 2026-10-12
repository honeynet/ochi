package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

const PublishMaxBodyBytes = 2 << 20 // 2 MiB; multi-frame decoded sessions exceed 8 KiB

// Publish reads the request body with a limit of 2 MiB and then publishes
// the received message. sensorID is truncated to 8 characters for display.
// dstHost (honeypot sensor IP) is stripped so it is never sent to subscribers.
func (h *Handlers) Publish(w http.ResponseWriter, r *http.Request) {
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
	if !h.PublishFunc(alteredMsg) {
		http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
		return
	}
	if h.Stats != nil {
		now := time.Now().UTC().Format(time.RFC3339)
		h.Stats.IncrementStat(sensorID, now)
	}
	w.WriteHeader(http.StatusAccepted)
}
