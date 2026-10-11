package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/honeynet/ochi/backend/entities"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handlers) GetSensorsByUser(w http.ResponseWriter, r *http.Request) {
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

func (h *Handlers) AddSensor(w http.ResponseWriter, r *http.Request) {
	userId := userIDFromCtx(r.Context())
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	var sensor entities.Sensor
	if err := decoder.Decode(&sensor); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sensor.UserID = userId
	if strings.TrimSpace(sensor.Name) == "" {
		sensor.Name = entities.RandomName()
	}

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

// DownloadBinary serves the binary for the requested architecture with the sensor UUID injected.
// It is unauthenticated: the sensor id in the path and the owner's "user_id" query parameter
// are two secrets that must match a stored sensor.
func (h *Handlers) DownloadBinary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	osType := q.Get("os")
	arch := q.Get("arch")
	sensorID := chi.URLParam(r, "sensor_id")
	userID := q.Get("user_id")

	validOS := map[string]bool{
		"linux": true,
	}

	validArch := map[string]bool{
		"amd64": true,
	}

	// The id is written into a fixed-width slot in the binary, so only the canonical form is accepted.
	if parsed, err := uuid.Parse(sensorID); err != nil || parsed.String() != sensorID || userID == "" || !validOS[osType] || !validArch[arch] {
		http.Error(w, "Invalid parameters", http.StatusBadRequest)
		return
	}

	sensor, err := h.Sensors.GetSensor(sensorID, userID)
	if err != nil {
		if isNotFoundError(err) {
			http.Error(w, "Sensor not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	newUUID := sensor.ID
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
