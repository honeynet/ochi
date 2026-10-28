package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/honeynet/ochi/backend/entities"
	"github.com/honeynet/ochi/backend/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyStatsRepo struct{}

func (d *dummyStatsRepo) LoadAllStats() ([]entities.SensorStat, error) {
	return nil, nil
}

func (d *dummyStatsRepo) BatchUpsertStats(deltas map[string]int64, lastSeen map[string]string) error {
	return nil
}

func TestPublish_AlwaysUsesCurrentUTCTime(t *testing.T) {
	statsService := services.NewStatsService(&dummyStatsRepo{})
	h := &Handlers{
		Stats: statsService,
		PublishFunc: func(msg []byte) bool {
			return true
		},
	}

	const sensorID = "11111111-2222-3333-4444-555555555555"
	// Event provides an untrusted timestamp in 2010
	payload := `{"sensorID":"` + sensorID + `","dstPort":80,"timestamp":"2010-01-01T00:00:00Z"}`

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/publish?token=token", bytes.NewBufferString(payload))

	before := time.Now().UTC().Add(-time.Second)
	h.Publish(w, r)
	after := time.Now().UTC().Add(time.Second)

	require.Equal(t, http.StatusAccepted, w.Code)

	stats := statsService.GetStatsForSensors([]string{sensorID})
	stat := stats[sensorID]
	assert.Equal(t, int64(1), stat.EventCount)

	// Verify that the sensor's spoofed timestamp was NOT used
	assert.NotEqual(t, "2010-01-01T00:00:00Z", stat.LastEventAt)

	// Verify that the recorded timestamp is current UTC time
	parsed, err := time.Parse(time.RFC3339, stat.LastEventAt)
	require.NoError(t, err)
	assert.True(t, parsed.After(before) || parsed.Equal(before))
	assert.True(t, parsed.Before(after) || parsed.Equal(after))
}
