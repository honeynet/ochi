package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	r := httptest.NewRequest("POST", "/publish?token=token", bytes.NewBufferString(payload))
	(&handlers.Handlers{Publish: cs.publish}).PublishHandler(w, r)

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
	r := httptest.NewRequest("POST", "/publish?token=token", bytes.NewBufferString(payload))
	(&handlers.Handlers{Publish: cs.publish}).PublishHandler(w, r)

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
	r := httptest.NewRequest("POST", "/publish", bytes.NewReader(body))
	(&handlers.Handlers{Publish: cs.publish}).PublishHandler(w, r)

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

func newPublishTestHandlers(t *testing.T, requireRegistered bool) (*handlers.Handlers, *repos.SensorRepo, *subscriber) {
	t.Helper()
	tmp := t.TempDir()
	db, err := sqlx.Connect("sqlite", filepath.Join(tmp, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	sensorRepo, err := repos.NewSensorRepo(db)
	require.NoError(t, err)

	cs := &server{
		subscribers:    make(map[*subscriber]struct{}),
		publishLimiter: rate.NewLimiter(rate.Inf, 1),
	}
	sub := &subscriber{msgs: make(chan []byte, 1)}
	cs.addSubscriber(sub)

	h := &handlers.Handlers{
		Publish:                  cs.publish,
		Sensors:                  sensorRepo,
		RequireRegisteredSensors: requireRegistered,
	}
	return h, sensorRepo, sub
}

func TestPublishHandler_RequiresRegisteredSensor(t *testing.T) {
	const registered = "abcd1234-ffff-ffff-ffff-ffffffffffff"
	const unknown = "99999999-0000-0000-0000-000000000000"
	body := func(sensorID string) string {
		return `{"sensorID":"` + sensorID + `","dstPort":80,"transport":"tcp"}`
	}
	publish := func(h *handlers.Handlers, sensorID string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/publish", strings.NewReader(body(sensorID)))
		h.PublishHandler(w, r)
		return w
	}

	t.Run("RejectsUnregisteredSensor", func(t *testing.T) {
		h, _, sub := newPublishTestHandlers(t, true)

		w := publish(h, unknown)
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.NotContains(t, w.Body.String(), unknown, "must not reveal which sensor ids exist")
		assert.Empty(t, sub.msgs, "a rejected event must not reach subscribers")
	})

	t.Run("AcceptsRegisteredSensor", func(t *testing.T) {
		h, sensorRepo, sub := newPublishTestHandlers(t, true)
		require.NoError(t, sensorRepo.AddSensors(entities.Sensor{
			ID: registered, Name: "sensor-1", UserID: "owner-1",
		}))

		require.Equal(t, http.StatusAccepted, publish(h, registered).Code)

		select {
		case msg := <-sub.msgs:
			var got map[string]any
			require.NoError(t, json.Unmarshal(msg, &got))
			assert.Equal(t, "abcd1234", got["sensorID"], "the uuid is still truncated for display")
		case <-time.After(time.Second):
			t.Fatal("expected published message")
		}
	})

	t.Run("DisabledByDefault", func(t *testing.T) {
		h, _, sub := newPublishTestHandlers(t, false)

		require.Equal(t, http.StatusAccepted, publish(h, unknown).Code)

		select {
		case <-sub.msgs:
		case <-time.After(time.Second):
			t.Fatal("unregistered sensors must still be accepted while the check is off")
		}
	})
}

func TestPublishHandler_RejectsMissingSensorID(t *testing.T) {
	cs := &server{
		subscribers:    make(map[*subscriber]struct{}),
		publishLimiter: rate.NewLimiter(rate.Inf, 1),
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/publish", bytes.NewBufferString(`{"dstPort":80}`))
	(&handlers.Handlers{Publish: cs.publish}).PublishHandler(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestPublishHandler_RateLimited(t *testing.T) {
	cs := &server{
		subscribers:    make(map[*subscriber]struct{}),
		publishLimiter: rate.NewLimiter(0, 0),
	}
	payload := `{"sensorID":"abcd1234-ffff-ffff-ffff-ffffffffffff","dstPort":80}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/publish", bytes.NewBufferString(payload))
	(&handlers.Handlers{Publish: cs.publish}).PublishHandler(w, r)
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
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "build", "bundle.js"), []byte("console.log(1)"), 0644))

	cs := &server{fs: os.DirFS(tmp), cfg: Config{JWTSecret: "test-secret"}}
	mux, err := newRouter(cs)
	require.NoError(t, err)

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
