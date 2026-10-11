package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/honeynet/ochi/backend/entities"
	"github.com/honeynet/ochi/backend/handlers"
	"github.com/honeynet/ochi/backend/repos"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestPublishHandler_AcceptsTypedGluttonEvent(t *testing.T) {
	cs := &server{
		subscribers:    make(map[*subscriber]struct{}),
		publishLimiter: rate.NewLimiter(rate.Inf, 1),
	}
	sub := &subscriber{msgs: make(chan []byte, 1)}
	cs.addSubscriber(sub)

	payload := `{
		"sensorID":"abcd1234-ffff-ffff-ffff-ffffffffffff",
		"dstPort":80,
		"srcHost":"1.1.1.1",
		"srcPort":"4321",
		"timestamp":"2026-01-01T00:00:00Z",
		"payload":"dGVzdA==",
		"rule":"Rule: TCP",
		"transport":"tcp",
		"decoded":{"payload":"test"}
	}`

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/publish?token=token", bytes.NewBufferString(payload))
	(&handlers.Handlers{PublishFunc: cs.publish}).Publish(w, r)

	resp := w.Result()
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	select {
	case msg := <-sub.msgs:
		var event map[string]any
		require.NoError(t, json.Unmarshal(msg, &event))
		assert.Equal(t, "abcd1234", event["sensorID"])
		assert.Equal(t, float64(80), event["dstPort"])
		assert.Equal(t, "1.1.1.1", event["srcHost"])
		decoded, ok := event["decoded"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "test", decoded["payload"])
	case <-time.After(time.Second):
		t.Fatal("expected published message")
	}
}

func TestPublishHandler_RoundTripsNewEnvelope(t *testing.T) {
	cs := &server{
		subscribers:    make(map[*subscriber]struct{}),
		publishLimiter: rate.NewLimiter(rate.Inf, 1),
	}
	sub := &subscriber{msgs: make(chan []byte, 1)}
	cs.addSubscriber(sub)

	payload := `{
		"sensorID":"abcd1234-ffff-ffff-ffff-ffffffffffff",
		"dstPort":445,
		"srcHost":"203.0.113.10",
		"srcPort":"54321",
		"dstHost":"198.51.100.8",
		"timestamp":"2026-01-01T00:00:01Z",
		"startedAt":"2026-01-01T00:00:00Z",
		"durationMs":1500,
		"srcPtr":"scanner.example.",
		"sensorVersion":"1.2.3",
		"rule":"Rule: SMB",
		"ruleName":"smb",
		"handler":"smb",
		"transport":"tcp",
		"payload":"dGVzdA==",
		"payloadHash":"deadbeef",
		"frameCount":2,
		"endReason":"timeout",
		"decoded":[{"direction":"read","path":"IPC$","setup":"TRANS2_SESSION_SETUP","status":"STATUS_NOT_IMPLEMENTED"}]
	}`

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/publish?token=token", bytes.NewBufferString(payload))
	(&handlers.Handlers{PublishFunc: cs.publish}).Publish(w, r)

	resp := w.Result()
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	select {
	case msg := <-sub.msgs:
		var event map[string]any
		require.NoError(t, json.Unmarshal(msg, &event))
		assert.Equal(t, "abcd1234", event["sensorID"])
		_, hasDstHost := event["dstHost"]
		assert.False(t, hasDstHost)
		assert.Equal(t, "timeout", event["endReason"])
		assert.Equal(t, float64(2), event["frameCount"])
		assert.Equal(t, "deadbeef", event["payloadHash"])
		assert.Equal(t, "1.2.3", event["sensorVersion"])
		assert.Equal(t, "smb", event["ruleName"])
		decoded, ok := event["decoded"].([]any)
		require.True(t, ok)
		require.Len(t, decoded, 1)
		frame, ok := decoded[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "IPC$", frame["path"])
	case <-time.After(time.Second):
		t.Fatal("expected published message")
	}
}

func TestPublishHandler_AcceptsLargeDecodedEvent(t *testing.T) {
	cs := &server{
		subscribers:    make(map[*subscriber]struct{}),
		publishLimiter: rate.NewLimiter(rate.Inf, 1),
	}
	sub := &subscriber{msgs: make(chan []byte, 1)}
	cs.addSubscriber(sub)

	framePayload := strings.Repeat("A", 2048)
	frames := make([]map[string]any, 10)
	for i := range frames {
		frames[i] = map[string]any{
			"direction": "read",
			"payload":   framePayload,
			"path":      "IPC$",
		}
	}
	event := map[string]any{
		"sensorID":  "abcd1234-ffff-ffff-ffff-ffffffffffff",
		"dstPort":   445,
		"srcHost":   "203.0.113.10",
		"srcPort":   "54321",
		"dstHost":   "198.51.100.8",
		"timestamp": "2026-01-01T00:00:00Z",
		"payload":   "dGVzdA==",
		"decoded":   frames,
	}
	body, err := json.Marshal(event)
	require.NoError(t, err)
	require.Greater(t, len(body), 8192)
	require.Less(t, len(body), handlers.PublishMaxBodyBytes)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/publish", bytes.NewReader(body))
	(&handlers.Handlers{PublishFunc: cs.publish}).Publish(w, r)

	resp := w.Result()
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	select {
	case msg := <-sub.msgs:
		var got map[string]any
		require.NoError(t, json.Unmarshal(msg, &got))
		decoded, ok := got["decoded"].([]any)
		require.True(t, ok)
		assert.Len(t, decoded, 10)
		_, hasDstHost := got["dstHost"]
		assert.False(t, hasDstHost)
	case <-time.After(time.Second):
		t.Fatal("expected published message")
	}
}

func TestPublishHandler_RejectsMissingSensorID(t *testing.T) {
	cs := &server{
		subscribers:    make(map[*subscriber]struct{}),
		publishLimiter: rate.NewLimiter(rate.Inf, 1),
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/publish", bytes.NewBufferString(`{"dstPort":80}`))
	(&handlers.Handlers{PublishFunc: cs.publish}).Publish(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestPublishHandler_RateLimited(t *testing.T) {
	cs := &server{
		subscribers:    make(map[*subscriber]struct{}),
		publishLimiter: rate.NewLimiter(0, 0),
	}
	payload := `{"sensorID":"abcd1234-ffff-ffff-ffff-ffffffffffff","dstPort":80}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/publish", bytes.NewBufferString(payload))
	(&handlers.Handlers{PublishFunc: cs.publish}).Publish(w, r)
	assert.Equal(t, http.StatusTooManyRequests, w.Result().StatusCode)
}

func TestPublish_DoesNotHoldLockWhileRateLimited(t *testing.T) {
	cs := &server{
		subscribers:    make(map[*subscriber]struct{}),
		publishLimiter: rate.NewLimiter(0, 0),
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		assert.False(t, cs.publish([]byte(`{"ok":true}`)))
	}()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("publish blocked while rate limited; limiter must use Allow outside the subscriber lock")
	}
	// subscribe registry remains usable while publishes are rejected
	sub := &subscriber{msgs: make(chan []byte, 1)}
	cs.addSubscriber(sub)
	cs.deleteSubscriber(sub)
}

func TestPublish_FanoutUnderBurst(t *testing.T) {
	cs := &server{
		subscribers:             make(map[*subscriber]struct{}),
		subscriberMessageBuffer: 64,
		publishLimiter:          rate.NewLimiter(rate.Limit(1000), 50),
	}
	sub := &subscriber{msgs: make(chan []byte, 64)}
	cs.addSubscriber(sub)

	const n = 40
	for i := 0; i < n; i++ {
		require.True(t, cs.publish([]byte(fmt.Sprintf(`{"i":%d}`, i))))
	}
	for i := 0; i < n; i++ {
		select {
		case <-sub.msgs:
		case <-time.After(time.Second):
			t.Fatalf("missing message %d", i)
		}
	}
}

func TestGetSharedEventByID_NoAuth(t *testing.T) {
	tmp := t.TempDir()
	db, err := sqlx.Connect("sqlite", filepath.Join(tmp, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	eventRepo, err := repos.NewEventRepo(db)
	require.NoError(t, err)

	rule := "Rule: TCP"
	handler := "http"
	dstHost := "198.51.100.8"
	created, err := eventRepo.Create(entities.Event{
		OwnerID:   "owner-1",
		Payload:   "cGF5bG9hZA==",
		DstPort:   80,
		Rule:      &rule,
		Handler:   &handler,
		Transport: "tcp",
		SensorID:  "sensor-1",
		SrcHost:   "1.2.3.4",
		SrcPort:   "4321",
		DstHost:   &dstHost,
		Timestamp: "2026-01-01T00:00:00Z",
		Decoded:   types.JSONText(`{"payload":"test"}`),
	})
	require.NoError(t, err)

	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "build"), 0755))

	cs := &server{
		eventRepo: eventRepo,
		fs:        os.DirFS(tmp),
		cfg:       Config{JWTSecret: "test-secret"},
	}
	mux, err := newRouter(cs)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/events/"+created.ID, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var got entities.Event
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, created.SrcHost, got.SrcHost)
	assert.Equal(t, created.DstPort, got.DstPort)
	assert.Empty(t, got.DstHost)
}

func TestGetEventsList_RequiresAuth(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "build"), 0755))

	cs := &server{
		fs:  os.DirFS(tmp),
		cfg: Config{JWTSecret: "test-secret"},
	}
	mux, err := newRouter(cs)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
}

func newEventsTestServer(t *testing.T, owners ...string) (do func(method, target, body string) *httptest.ResponseRecorder, repo *repos.EventRepo, ids map[string][]string) {
	t.Helper()
	tmp := t.TempDir()
	db, err := sqlx.Connect("sqlite", filepath.Join(tmp, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo, err = repos.NewEventRepo(db)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "build"), 0755))

	ids = map[string][]string{}
	for n, owner := range owners {
		for i := 1; i <= 3; i++ {
			ev, err := repo.Create(entities.Event{
				OwnerID:   owner,
				Payload:   "cGF5bG9hZA==",
				DstPort:   80,
				Transport: "tcp",
				SensorID:  "sensor-1",
				SrcHost:   "1.2.3.4",
				SrcPort:   "4321",
				Timestamp: fmt.Sprintf("2026-01-0%dT00:00:00Z", i+n),
			})
			require.NoError(t, err)
			ids[owner] = append(ids[owner], ev.ID)
		}
	}

	cs := &server{
		eventRepo: repo,
		fs:        os.DirFS(tmp),
		cfg:       Config{JWTSecret: "test-secret"},
	}
	mux, err := newRouter(cs)
	require.NoError(t, err)
	token, err := entities.NewToken("test-secret", entities.User{ID: owners[0]})
	require.NoError(t, err)

	do = func(method, target, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		return w
	}
	return do, repo, ids
}

func TestEventsList_Paginated(t *testing.T) {
	do, _, _ := newEventsTestServer(t, "owner-1", "owner-2")

	w := do(http.MethodGet, "/api/events?limit=2&offset=0", "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "3", w.Header().Get("X-Total-Count"))
	var page []entities.Event
	require.NoError(t, json.NewDecoder(w.Body).Decode(&page))
	assert.Len(t, page, 2)
	for _, ev := range page {
		assert.Empty(t, ev.Payload, "list rows should omit payload")
		assert.Empty(t, ev.Decoded, "list rows should omit decoded")
		assert.Empty(t, ev.TLS, "list rows should omit tls")
	}

	w = do(http.MethodGet, "/api/events?limit=2&offset=2", "")
	require.Equal(t, http.StatusOK, w.Code)
	page = nil
	require.NoError(t, json.NewDecoder(w.Body).Decode(&page))
	assert.Len(t, page, 1)

	assert.Equal(t, http.StatusBadRequest, do(http.MethodGet, "/api/events?limit=0", "").Code)
	assert.Equal(t, http.StatusBadRequest, do(http.MethodGet, "/api/events?limit=101", "").Code)
	assert.Equal(t, http.StatusBadRequest, do(http.MethodGet, "/api/events?offset=-1", "").Code)
}

// Only the owner may delete a sensor; everything else is a 404, so the endpoint
// never reveals whether a sensor ID exists.
func TestDeleteSensor_OwnerOnly(t *testing.T) {
	tmp := t.TempDir()
	db, err := sqlx.Connect("sqlite", filepath.Join(tmp, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	sensorRepo, err := repos.NewSensorRepo(db)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "build"), 0755))

	const mine = "11111111-1111-1111-1111-111111111111"
	const theirs = "22222222-2222-2222-2222-222222222222"
	const missing = "33333333-3333-3333-3333-333333333333"
	require.NoError(t, sensorRepo.AddSensors(entities.Sensor{ID: mine, Name: "mine", UserID: "owner-1"}))
	require.NoError(t, sensorRepo.AddSensors(entities.Sensor{ID: theirs, Name: "theirs", UserID: "owner-2"}))

	cs := &server{
		sensorRepo: sensorRepo,
		fs:         os.DirFS(tmp),
		cfg:        Config{JWTSecret: "test-secret"},
	}
	mux, err := newRouter(cs)
	require.NoError(t, err)

	token, err := entities.NewToken("test-secret", entities.User{ID: "owner-1"})
	require.NoError(t, err)

	do := func(id string, authenticated bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodDelete, "/api/sensors/"+id, nil)
		if authenticated {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		return w
	}

	assert.Equal(t, http.StatusUnauthorized, do(mine, false).Code)

	assert.Equal(t, http.StatusNotFound, do(theirs, true).Code, "another user's sensor")
	assert.Equal(t, http.StatusNotFound, do(missing, true).Code, "unknown sensor")

	survivors, err := sensorRepo.GetSensorsByOwnerId("owner-2")
	require.NoError(t, err)
	assert.Len(t, survivors, 1, "a refused delete must leave the sensor in place")

	require.Equal(t, http.StatusOK, do(mine, true).Code)
	remaining, err := sensorRepo.GetSensorsByOwnerId("owner-1")
	require.NoError(t, err)
	assert.Empty(t, remaining)

	assert.Equal(t, http.StatusNotFound, do(mine, true).Code, "deleting twice is not a server error")
}

func TestOpenSQLite_EnablesWAL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wal.db")
	db, err := openSQLite(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	var mode string
	require.NoError(t, db.Get(&mode, "PRAGMA journal_mode"))
	assert.Equal(t, "wal", strings.ToLower(mode))

	var busy int
	require.NoError(t, db.Get(&busy, "PRAGMA busy_timeout"))
	assert.Equal(t, 5000, busy)
}

func TestEventsDelete_Bulk(t *testing.T) {
	do, repo, ids := newEventsTestServer(t, "owner-1", "owner-2")

	assert.Equal(t, http.StatusBadRequest, do(http.MethodDelete, "/api/events", `{"ids":[]}`).Code)

	foreign := ids["owner-2"][0]
	w := do(http.MethodDelete, "/api/events", `{"ids":["`+foreign+`"]}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"deleted":0}`, w.Body.String())
	_, err := repo.GetByID(foreign)
	require.NoError(t, err)

	mine := ids["owner-1"]
	w = do(http.MethodDelete, "/api/events", `{"ids":["`+mine[0]+`","`+mine[1]+`"]}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"deleted":2}`, w.Body.String())

	w = do(http.MethodGet, "/api/events", "")
	assert.Equal(t, "1", w.Header().Get("X-Total-Count"))
}

func TestRouter_CorsPreflightAndStaticFiles(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "build"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "index.html"), []byte("<!doctype html>"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "build", "bundle.js"), []byte("console.log(1)"), 0644))

	cs := &server{fs: os.DirFS(tmp), cfg: Config{JWTSecret: "test-secret"}}
	mux, err := newRouter(cs)
	require.NoError(t, err)

	for _, route := range []string{"/", "/myqueries", "/mysensors", "/myevents"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, route, nil))
		assert.Equal(t, http.StatusOK, w.Code, "route: %s", route)
		assert.Equal(t, "<!doctype html>", w.Body.String(), "route: %s", route)
	}

	req := httptest.NewRequest(http.MethodOptions, "/api/events/abc", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodDelete)
	req.Header.Set("Access-Control-Request-Headers", "Authorization")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, http.MethodDelete, w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Authorization", w.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodOptions, "/nope", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/build/bundle.js", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "X-Total-Count", w.Header().Get("Access-Control-Expose-Headers"))
	assert.Equal(t, "console.log(1)", w.Body.String())
}

func TestSensorStats_RESTEndpointAndIsolation(t *testing.T) {
	tmp := t.TempDir()
	db, err := sqlx.Connect("sqlite", filepath.Join(tmp, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	sensorRepo, err := repos.NewSensorRepo(db)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "build"), 0755))

	const s1 = "11111111-1111-1111-1111-111111111111"
	const s2 = "22222222-2222-2222-2222-222222222222"
	require.NoError(t, sensorRepo.AddSensors(entities.Sensor{ID: s1, Name: "sensor-1", UserID: "owner-1"}))
	require.NoError(t, sensorRepo.AddSensors(entities.Sensor{ID: s2, Name: "sensor-2", UserID: "owner-2"}))

	cs := &server{
		sensorRepo:     sensorRepo,
		fs:             os.DirFS(tmp),
		cfg:            Config{JWTSecret: "test-secret", PublishToken: "token"},
		publishLimiter: rate.NewLimiter(rate.Inf, 1),
	}
	mux, err := newRouter(cs)
	require.NoError(t, err)

	token1, err := entities.NewToken("test-secret", entities.User{ID: "owner-1"})
	require.NoError(t, err)

	// Publish 2 events for s1
	for i := 0; i < 2; i++ {
		payload := fmt.Sprintf(`{"sensorID":"%s","dstPort":80,"timestamp":"2026-01-01T00:00:00Z"}`, s1)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/api/publish?token=token", bytes.NewBufferString(payload))
		mux.ServeHTTP(w, r)
		require.Equal(t, http.StatusAccepted, w.Code)
	}

	// Publish 1 event for s2
	{
		payload := fmt.Sprintf(`{"sensorID":"%s","dstPort":80,"timestamp":"2026-01-01T00:00:00Z"}`, s2)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/api/publish?token=token", bytes.NewBufferString(payload))
		mux.ServeHTTP(w, r)
		require.Equal(t, http.StatusAccepted, w.Code)
	}

	// Unauthenticated GET /api/sensors/stats -> 401
	{
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/sensors/stats", nil)
		mux.ServeHTTP(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	}

	// Authenticated as owner-1 -> only sees s1
	{
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/sensors/stats", nil)
		r.Header.Set("Authorization", "Bearer "+token1)
		mux.ServeHTTP(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		var stats map[string]entities.SensorStat
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &stats))
		require.Contains(t, stats, s1)
		assert.Equal(t, int64(2), stats[s1].EventCount)
		assert.NotContains(t, stats, s2, "owner-1 must not see owner-2 stats")
	}
}

func TestSensorStats_FlushAndReload(t *testing.T) {
	tmp := t.TempDir()
	db, err := sqlx.Connect("sqlite", filepath.Join(tmp, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	repo, err := repos.NewSensorRepo(db)
	require.NoError(t, err)

	const sID = "33333333-3333-3333-3333-333333333333"
	repo.IncrementStat(sID, "2026-10-11T12:00:00Z")
	repo.IncrementStat(sID, "2026-10-11T12:01:00Z")

	// Flush to SQLite
	require.NoError(t, repo.FlushStats())

	// Verify row in SQLite
	var row entities.SensorStat
	require.NoError(t, db.Get(&row, "SELECT sensor_id, event_count, last_event_at FROM sensor_stats WHERE sensor_id=?", sID))
	assert.Equal(t, int64(2), row.EventCount)
	assert.Equal(t, "2026-10-11T12:01:00Z", row.LastEventAt)

	// Flush again with no changes does not fail or duplicate
	require.NoError(t, repo.FlushStats())
	require.NoError(t, db.Get(&row, "SELECT sensor_id, event_count, last_event_at FROM sensor_stats WHERE sensor_id=?", sID))
	assert.Equal(t, int64(2), row.EventCount)

	// Simulate restart: new repo loads from DB
	repo2, err := repos.NewSensorRepo(db)
	require.NoError(t, err)
	require.NoError(t, repo2.LoadStats())

	stats := repo2.GetStatsForSensors([]string{sID})
	assert.Equal(t, int64(2), stats[sID].EventCount)
	assert.Equal(t, "2026-10-11T12:01:00Z", stats[sID].LastEventAt)
}

func TestSensorStats_WebSocketBroadcast(t *testing.T) {
	tmp := t.TempDir()
	db, err := sqlx.Connect("sqlite", filepath.Join(tmp, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	sensorRepo, err := repos.NewSensorRepo(db)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "build"), 0755))

	cs := &server{
		sensorRepo:       sensorRepo,
		fs:               os.DirFS(tmp),
		cfg:              Config{JWTSecret: "test-secret", PublishToken: "token"},
		publishLimiter:   rate.NewLimiter(rate.Inf, 1),
		statsSubscribers: make(map[*subscriber]struct{}),
	}
	mux, err := newRouter(cs)
	require.NoError(t, err)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/sensors/stats/ws"
	c, _, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	defer c.Close(websocket.StatusNormalClosure, "")

	// Trigger a stat increment and broadcast
	const sID = "ws-sensor-12345678"
	sensorRepo.IncrementStat(sID, "2026-10-11T12:00:00Z")
	dirty := sensorRepo.PopDirtyStatsForBroadcast()
	require.NotEmpty(t, dirty)
	data, err := json.Marshal(dirty)
	require.NoError(t, err)
	cs.publishStats(data)

	// Read message from WS client
	_, msg, err := c.Read(ctx)
	require.NoError(t, err)

	var received []entities.SensorStat
	require.NoError(t, json.Unmarshal(msg, &received))
	require.Len(t, received, 1)
	assert.Equal(t, sID, received[0].SensorID)
	assert.Equal(t, int64(1), received[0].EventCount)
}
