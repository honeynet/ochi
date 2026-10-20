package services

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/honeynet/ochi/backend/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	mu        sync.Mutex
	rows      []entities.SensorStat
	upserts   []map[string]int64
	failFlush bool
}

func (m *mockRepo) LoadAllStats() ([]entities.SensorStat, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.rows, nil
}

func (m *mockRepo) BatchUpsertStats(deltas map[string]int64, lastSeen map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failFlush {
		return errors.New("db error")
	}
	m.upserts = append(m.upserts, deltas)
	for id, delta := range deltas {
		found := false
		for i, r := range m.rows {
			if r.SensorID == id {
				m.rows[i].EventCount += delta
				m.rows[i].LastEventAt = lastSeen[id]
				found = true
				break
			}
		}
		if !found {
			m.rows = append(m.rows, entities.SensorStat{
				SensorID:    id,
				EventCount:  delta,
				LastEventAt: lastSeen[id],
			})
		}
	}
	return nil
}

func TestStatsService_IncrementAndQuery(t *testing.T) {
	repo := &mockRepo{}
	svc := NewStatsService(repo)

	const s1 = "11111111-1111-1111-1111-111111111111"
	stat1 := svc.IncrementStat(s1, "2026-10-12T12:00:00Z")
	assert.Equal(t, int64(1), stat1.EventCount)
	assert.Equal(t, s1, stat1.SensorID)
	assert.Equal(t, "2026-10-12T12:00:00Z", stat1.LastEventAt)

	// Increment again
	stat1 = svc.IncrementStat(s1, "2026-10-12T12:01:00Z")
	assert.Equal(t, int64(2), stat1.EventCount)
	assert.Equal(t, "2026-10-12T12:01:00Z", stat1.LastEventAt)

	// Query exact ID
	stats := svc.GetStatsForSensors([]string{s1, "unknown"})
	assert.Equal(t, int64(2), stats[s1].EventCount)
	assert.Equal(t, int64(0), stats["unknown"].EventCount)

	// Query 8-char prefix matches full UUID
	statsPrefix := svc.GetStatsForSensors([]string{"11111111"})
	assert.Equal(t, int64(2), statsPrefix["11111111"].EventCount)

	// Delete sensor removes from memory
	svc.DeleteSensor(s1)
	statsAfterDelete := svc.GetStatsForSensors([]string{s1})
	assert.Equal(t, int64(0), statsAfterDelete[s1].EventCount)
}

func TestStatsService_FlushAndRevertOnError(t *testing.T) {
	repo := &mockRepo{failFlush: true}
	svc := NewStatsService(repo)

	const sID = "s-test-retry"
	svc.IncrementStat(sID, "2026-10-12T12:00:00Z")

	// Flush fails -> deltas preserved
	err := svc.Flush()
	require.Error(t, err)

	// Recover DB
	repo.mu.Lock()
	repo.failFlush = false
	repo.mu.Unlock()

	// Flush again -> succeeds and commits the preserved delta
	require.NoError(t, svc.Flush())

	repo.mu.Lock()
	require.Len(t, repo.upserts, 1)
	assert.Equal(t, int64(1), repo.upserts[0][sID])
	repo.mu.Unlock()
}

func TestStatsService_LifecycleStartStop(t *testing.T) {
	repo := &mockRepo{
		rows: []entities.SensorStat{
			{SensorID: "init-1", EventCount: 10, LastEventAt: "2026-10-12T10:00:00Z"},
		},
	}
	svc := NewStatsService(repo)
	svc.SetFlushInterval(50 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, svc.Start(ctx))

	// Initial load verified
	stats := svc.GetStatsForSensors([]string{"init-1"})
	assert.Equal(t, int64(10), stats["init-1"].EventCount)

	// Increment and stop -> Stop performs final flush
	svc.IncrementStat("init-1", "2026-10-12T10:01:00Z")
	require.NoError(t, svc.Stop())

	repo.mu.Lock()
	require.NotEmpty(t, repo.upserts)
	repo.mu.Unlock()

	statsAfterStop := svc.GetStatsForSensors([]string{"init-1"})
	assert.Equal(t, int64(11), statsAfterStop["init-1"].EventCount)
}

func TestStatsService_WebSocketBroadcastAndSubscribe(t *testing.T) {
	svc := NewStatsService(&mockRepo{})
	svc.SetBroadcastInterval(50 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, svc.Start(ctx))
	defer func() { _ = svc.Stop() }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close(websocket.StatusInternalError, "")
		_ = svc.Subscribe(r.Context(), c)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConn, _, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	defer clientConn.Close(websocket.StatusNormalClosure, "")

	// Increment a sensor to trigger broadcast
	const sID = "ws-broadcast-sensor"
	svc.IncrementStat(sID, "2026-10-12T12:00:00Z")

	// Read message from WebSocket client
	readCtx, readCancel := context.WithTimeout(ctx, 3*time.Second)
	defer readCancel()

	_, msg, err := clientConn.Read(readCtx)
	require.NoError(t, err)

	var received []entities.SensorStat
	require.NoError(t, json.Unmarshal(msg, &received))
	require.NotEmpty(t, received)
	assert.Equal(t, sID, received[0].SensorID)
	assert.Equal(t, int64(1), received[0].EventCount)
}

func TestStatsService_BoundaryAndOverflowProtection(t *testing.T) {
	repo := &mockRepo{}
	svc := NewStatsService(repo)

	const sID = "s-boundary"

	// Preset totals to math.MaxInt64
	svc.mu.Lock()
	svc.totals[sID] = math.MaxInt64
	svc.deltas[sID] = math.MaxInt64
	svc.mu.Unlock()

	// Incrementing at math.MaxInt64 must not overflow to negative numbers
	stat := svc.IncrementStat(sID, "2026-10-12T12:00:00Z")
	assert.Equal(t, int64(math.MaxInt64), stat.EventCount)
	assert.True(t, stat.EventCount >= 0, "count must never be negative")

	// Query boundary
	stats := svc.GetStatsForSensors([]string{sID})
	assert.Equal(t, int64(math.MaxInt64), stats[sID].EventCount)
	assert.True(t, stats[sID].EventCount >= 0)

	// Revert delta boundary test
	svc.mu.Lock()
	svc.deltas[sID] = math.MaxInt64 - 10
	svc.mu.Unlock()

	svc.revertDeltas(map[string]int64{sID: 20})
	svc.mu.RLock()
	assert.Equal(t, int64(math.MaxInt64), svc.deltas[sID], "delta should cap at MaxInt64 without overflow")
	svc.mu.RUnlock()
}
