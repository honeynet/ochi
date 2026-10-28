package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"github.com/honeynet/ochi/backend/entities"
	"github.com/honeynet/ochi/backend/handlers"
	"github.com/honeynet/ochi/backend/repos"
	"github.com/honeynet/ochi/backend/services"
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
		PublishFunc:              cs.publish,
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
		r := httptest.NewRequest("POST", "/api/publish", strings.NewReader(body(sensorID)))
		h.Publish(w, r)
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

// A refused token is a client error, not a server fault: every rejection path must
// answer 401, and must not report why the token was refused. jwtauth's own
// Authenticator writes err.Error() into the body, which is why handlers.Authenticator
// replaces it.
func TestAuthenticator_RejectsCredentialsWith401(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "build"), 0755))

	cs := &server{
		fs:  os.DirFS(tmp),
		cfg: Config{JWTSecret: "test-secret"},
	}
	mux, err := newRouter(cs)
	require.NoError(t, err)

	expiredClaims := map[string]any{"user_id": "owner-1"}
	jwtauth.SetExpiry(expiredClaims, time.Now().Add(-time.Hour))
	_, expiredToken, err := entities.NewTokenAuth("test-secret").Encode(expiredClaims)
	require.NoError(t, err)

	forgedToken, err := entities.NewToken(entities.NewTokenAuth("not-the-server-secret"), entities.User{ID: "owner-1"})
	require.NoError(t, err)

	tests := []struct {
		name   string
		header string
	}{
		{"Expired", "Bearer " + expiredToken},
		{"WrongSecret", "Bearer " + forgedToken},
		{"Malformed", "Bearer not-a-jwt"},
		{"WrongScheme", "Basic " + expiredToken},
		{"MissingHeader", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Equal(t, http.StatusText(http.StatusUnauthorized), strings.TrimSpace(w.Body.String()),
				"must not disclose token validation details")
		})
	}
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
	token, err := entities.NewToken(entities.NewTokenAuth("test-secret"), entities.User{ID: owners[0]})
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

	token, err := entities.NewToken(entities.NewTokenAuth("test-secret"), entities.User{ID: "owner-1"})
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
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "build", "bundle.js"), []byte("console.log(1)"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "index.html"), []byte("<!doctype html>"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "global.css"), []byte("body {}"), 0644))

	cs := &server{fs: os.DirFS(tmp), cfg: Config{JWTSecret: "test-secret"}}
	mux, err := newRouter(cs)
	require.NoError(t, err)

	for _, path := range []string{"/", "/myqueries", "/myevents", "/mysensors", "/users", "/events/abc"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		assert.Equal(t, http.StatusOK, w.Code, "expected 200 for %s", path)
		assert.Equal(t, "<!doctype html>", w.Body.String())
		assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
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

	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/global.css", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/css", w.Header().Get("Content-Type"))
	assert.Equal(t, "body {}", w.Body.String())
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

	statsService := services.NewStatsService(sensorRepo)
	require.NoError(t, statsService.Start(context.Background()))
	defer func() { _ = statsService.Stop() }()

	cs := &server{
		sensorRepo:     sensorRepo,
		statsService:   statsService,
		fs:             os.DirFS(tmp),
		cfg:            Config{JWTSecret: "test-secret", PublishToken: "token"},
		publishLimiter: rate.NewLimiter(rate.Inf, 1),
	}
	mux, err := newRouter(cs)
	require.NoError(t, err)

	token1, err := entities.NewToken(entities.NewTokenAuth("test-secret"), entities.User{ID: "owner-1"})
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

	statsService := services.NewStatsService(repo)
	require.NoError(t, statsService.Start(context.Background()))

	const sID = "33333333-3333-3333-3333-333333333333"
	statsService.IncrementStat(sID, "2026-10-11T12:00:00Z")
	statsService.IncrementStat(sID, "2026-10-11T12:01:00Z")

	// Flush to SQLite
	require.NoError(t, statsService.Flush())

	// Verify row in SQLite
	var row entities.SensorStat
	require.NoError(t, db.Get(&row, "SELECT sensor_id, event_count, last_event_at FROM sensor_stats WHERE sensor_id=?", sID))
	assert.Equal(t, int64(2), row.EventCount)
	assert.Equal(t, "2026-10-11T12:01:00Z", row.LastEventAt)

	// Flush again with no changes does not fail or duplicate
	require.NoError(t, statsService.Flush())
	require.NoError(t, db.Get(&row, "SELECT sensor_id, event_count, last_event_at FROM sensor_stats WHERE sensor_id=?", sID))
	assert.Equal(t, int64(2), row.EventCount)

	// Stop previous service
	require.NoError(t, statsService.Stop())

	// Simulate restart: new repo and service loads from DB
	repo2, err := repos.NewSensorRepo(db)
	require.NoError(t, err)
	statsService2 := services.NewStatsService(repo2)
	require.NoError(t, statsService2.Start(context.Background()))
	defer func() { _ = statsService2.Stop() }()

	stats := statsService2.GetStatsForSensors([]string{sID})
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

	statsService := services.NewStatsService(sensorRepo)
	statsService.SetBroadcastInterval(50 * time.Millisecond)
	require.NoError(t, statsService.Start(context.Background()))
	defer func() { _ = statsService.Stop() }()

	cs := &server{
		sensorRepo:     sensorRepo,
		statsService:   statsService,
		fs:             os.DirFS(tmp),
		cfg:            Config{JWTSecret: "test-secret", PublishToken: "token"},
		publishLimiter: rate.NewLimiter(rate.Inf, 1),
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

	// Trigger a stat increment
	const sID = "ws-sensor-12345678"
	statsService.IncrementStat(sID, "2026-10-11T12:00:00Z")

	// Read message from WS client
	_, msg, err := c.Read(ctx)
	require.NoError(t, err)

	var received []entities.SensorStat
	require.NoError(t, json.Unmarshal(msg, &received))
	require.Len(t, received, 1)
	assert.Equal(t, sID, received[0].SensorID)
	assert.Equal(t, int64(1), received[0].EventCount)
}

func newRunnableServer(t *testing.T) *server {
	t.Helper()
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "build"), 0755))
	db, err := openSQLite(filepath.Join(tmp, "data.db"))
	require.NoError(t, err)

	cs := &server{
		cfg:                     Config{JWTSecret: "test-secret"},
		subscriberMessageBuffer: 64,
		subscribers:             make(map[*subscriber]struct{}),
		publishLimiter:          rate.NewLimiter(rate.Inf, 1),
		db:                      db,
		fs:                      os.DirFS(tmp),
	}
	cs.uRepo, err = repos.NewUserRepo(db)
	require.NoError(t, err)
	cs.queryRepo, err = repos.NewQueryRepo(db)
	require.NoError(t, err)
	cs.eventRepo, err = repos.NewEventRepo(db)
	require.NoError(t, err)
	cs.sensorRepo, err = repos.NewSensorRepo(db)
	require.NoError(t, err)
	cs.statsService = services.NewStatsService(cs.sensorRepo)
	require.NoError(t, cs.statsService.Start(context.Background()))
	cs.mux, err = newRouter(cs)
	require.NoError(t, err)
	return cs
}

// A cancelled context (as a signal produces) drains the server and closes the
// shared database. Before this change only uRepo.Close was called, and it was a
// no-op because NewUserRepo never stored the handle, so the database stayed open.
func TestServer_ShutdownClosesDatabase(t *testing.T) {
	cs := newRunnableServer(t)
	db := cs.db
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- cs.serve(ctx, l) }()

	base := "http://" + l.Addr().String()
	require.Eventually(t, func() bool {
		resp, err := http.Get(base + "/api/events/missing")
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return true
	}, 3*time.Second, 20*time.Millisecond, "server never started serving")

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return after the context was cancelled")
	}
	require.Error(t, db.Ping(), "the shared database must be closed after shutdown")
}

// A request already being handled when shutdown starts runs to completion.
func TestServer_ShutdownDrainsInflightRequests(t *testing.T) {
	cs := newRunnableServer(t)

	started := make(chan struct{})
	release := make(chan struct{})
	r := chi.NewRouter()
	r.Get("/slow", func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusOK)
	})
	cs.mux = r

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- cs.serve(ctx, l) }()

	status := make(chan int, 1)
	go func() {
		resp, err := http.Get("http://" + l.Addr().String() + "/slow")
		if err != nil {
			status <- -1
			return
		}
		defer resp.Body.Close()
		status <- resp.StatusCode
	}()

	<-started // the request is in the handler
	cancel()  // shutdown begins while it is in flight
	time.Sleep(100 * time.Millisecond)
	close(release) // let the handler finish

	assert.Equal(t, http.StatusOK, <-status, "an in-flight request must complete")
	require.NoError(t, <-done)
}

// The database is closed even when the server never starts, so a failed start
// does not leak the handle or leave the WAL unchecked.
func TestServer_RunClosesDatabaseOnListenError(t *testing.T) {
	cs := newRunnableServer(t)
	db := cs.db
	cs.cfg.ListenAddress = "127.0.0.1:-1" // not a valid port

	require.Error(t, cs.Run())
	require.Error(t, db.Ping(), "the database must be closed when the listener fails")
}

func TestPublish_FiltersByHandler(t *testing.T) {
	cs := &server{
		subscribers:    make(map[*subscriber]struct{}),
		publishLimiter: rate.NewLimiter(rate.Inf, 1),
	}
	all := &subscriber{msgs: make(chan []byte, 4)}
	smb := &subscriber{msgs: make(chan []byte, 4), handlers: parseHandlerFilter(url.Values{"handler": {"smb, ssh"}})}
	cs.addSubscriber(all)
	cs.addSubscriber(smb)

	require.True(t, cs.publish([]byte(`{"handler":"smb"}`)))
	require.True(t, cs.publish([]byte(`{"handler":"http"}`)))
	require.True(t, cs.publish([]byte(`{"handler":"ssh"}`)))
	require.True(t, cs.publish([]byte(`{}`)))

	assert.Len(t, all.msgs, 4)
	assert.Len(t, smb.msgs, 2)
}

func TestParseHandlerFilter(t *testing.T) {
	assert.Nil(t, parseHandlerFilter(url.Values{}))
	assert.Nil(t, parseHandlerFilter(url.Values{"handler": {" , "}}))
	got := parseHandlerFilter(url.Values{"handler": {"smb,ssh", "http"}})
	assert.Equal(t, map[string]struct{}{"smb": {}, "ssh": {}, "http": {}}, got)
}

func TestQueryHandlers_StatusCodes(t *testing.T) {
	tmp := t.TempDir()
	db, err := sqlx.Connect("sqlite", filepath.Join(tmp, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	queryRepo, err := repos.NewQueryRepo(db)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "build"), 0755))

	mine, err := queryRepo.Create("owner-1", "tcp.port == 80", "mine", true, nil)
	require.NoError(t, err)
	theirs, err := queryRepo.Create("owner-2", "tcp.port == 443", "theirs", true, nil)
	require.NoError(t, err)

	cs := &server{
		queryRepo: queryRepo,
		fs:        os.DirFS(tmp),
		cfg:       Config{JWTSecret: "test-secret"},
	}
	mux, err := newRouter(cs)
	require.NoError(t, err)

	token, err := entities.NewToken(entities.NewTokenAuth("test-secret"), entities.User{ID: "owner-1"})
	require.NoError(t, err)

	do := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		return w
	}

	// Create with invalid JSON -> 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, do(http.MethodPost, "/api/queries", "{invalid-json").Code)

	// Update another user's query -> 403 Forbidden
	updateBody := fmt.Sprintf(`{"id":%q,"content":"udp.port == 53"}`, theirs.ID)
	assert.Equal(t, http.StatusForbidden, do(http.MethodPatch, "/api/queries/"+theirs.ID, updateBody).Code)
	theirsAfterUpdate, err := queryRepo.GetByID(theirs.ID)
	require.NoError(t, err)
	assert.Equal(t, theirs.Content, theirsAfterUpdate.Content, "unauthorized update must not modify the query")

	// Update with invalid JSON -> 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, do(http.MethodPatch, "/api/queries/"+mine.ID, "{invalid-json").Code)

	// Delete another user's query -> 403 Forbidden
	assert.Equal(t, http.StatusForbidden, do(http.MethodDelete, "/api/queries/"+theirs.ID, "").Code)
	theirsAfterDelete, err := queryRepo.GetByID(theirs.ID)
	require.NoError(t, err)
	assert.Equal(t, theirs.ID, theirsAfterDelete.ID, "unauthorized delete must not remove the query")
}

func TestSensorHandlers_StatusCodes(t *testing.T) {
	tmp := t.TempDir()
	db, err := sqlx.Connect("sqlite", filepath.Join(tmp, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	sensorRepo, err := repos.NewSensorRepo(db)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "build"), 0755))

	cs := &server{
		sensorRepo: sensorRepo,
		fs:         os.DirFS(tmp),
		cfg:        Config{JWTSecret: "test-secret"},
	}
	mux, err := newRouter(cs)
	require.NoError(t, err)

	token, err := entities.NewToken(entities.NewTokenAuth("test-secret"), entities.User{ID: "owner-1"})
	require.NoError(t, err)

	do := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		return w
	}

	// AddSensor with invalid JSON -> 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, do(http.MethodPost, "/api/sensors", "{bad-json").Code)
}

func TestEventHandlers_StatusCodes(t *testing.T) {
	do, repo, ids := newEventsTestServer(t, "owner-1", "owner-2")

	// CreateEvent with invalid JSON -> 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, do(http.MethodPost, "/api/events", "{bad-json").Code)

	// DeleteEvent of another user's event -> 403 Forbidden
	theirsID := ids["owner-2"][0]
	assert.Equal(t, http.StatusForbidden, do(http.MethodDelete, "/api/events/"+theirsID, "").Code)
	theirsAfterDelete, err := repo.GetByID(theirsID)
	require.NoError(t, err)
	assert.Equal(t, theirsID, theirsAfterDelete.ID, "unauthorized delete must not remove the event")
}

// Listing users and choosing a sensor ID are admin-only; everyone else gets a 403
// and their sensors get a server-generated ID.
func TestAdminPermissions(t *testing.T) {
	tmp := t.TempDir()
	db, err := sqlx.Connect("sqlite", filepath.Join(tmp, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	uRepo, err := repos.NewUserRepo(db)
	require.NoError(t, err)
	sensorRepo, err := repos.NewSensorRepo(db)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "build"), 0755))

	admin, err := uRepo.Find("admin@test")
	require.NoError(t, err)
	require.NoError(t, uRepo.GrantRole(admin.ID, entities.RoleAdmin))
	plain, err := uRepo.Find("plain@test")
	require.NoError(t, err)

	cs := &server{
		uRepo:      uRepo,
		sensorRepo: sensorRepo,
		fs:         os.DirFS(tmp),
		cfg:        Config{JWTSecret: "test-secret"},
	}
	mux, err := newRouter(cs)
	require.NoError(t, err)

	tokenAuth := entities.NewTokenAuth("test-secret")
	adminToken, err := entities.NewToken(tokenAuth, admin)
	require.NoError(t, err)
	plainToken, err := entities.NewToken(tokenAuth, plain)
	require.NoError(t, err)

	do := func(token, method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		return w
	}

	// List users
	assert.Equal(t, http.StatusForbidden, do(plainToken, http.MethodGet, "/api/users", "").Code)
	w := do(adminToken, http.MethodGet, "/api/users", "")
	require.Equal(t, http.StatusOK, w.Code)
	var users []entities.User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &users))
	require.Len(t, users, 2)
	assert.Equal(t, []entities.Role{entities.RoleAdmin}, users[0].Roles)

	// Add sensor by ID
	const chosen = "44444444-4444-4444-4444-444444444444"
	body := fmt.Sprintf(`{"id":%q,"name":"s"}`, chosen)
	assert.Equal(t, http.StatusForbidden, do(plainToken, http.MethodPost, "/api/sensors", body).Code)
	assert.False(t, sensorRepo.Exists(chosen), "a refused add must not register the sensor")

	assert.Equal(t, http.StatusBadRequest, do(adminToken, http.MethodPost, "/api/sensors", `{"id":"not-a-uuid"}`).Code)
	require.Equal(t, http.StatusOK, do(adminToken, http.MethodPost, "/api/sensors", body).Code)
	assert.True(t, sensorRepo.Exists(chosen))
	assert.Equal(t, http.StatusConflict, do(adminToken, http.MethodPost, "/api/sensors", body).Code)

	// Without an ID anyone may add a sensor, and the server picks the ID.
	w = do(plainToken, http.MethodPost, "/api/sensors", `{"name":"mine"}`)
	require.Equal(t, http.StatusOK, w.Code)
	var added entities.Sensor
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &added))
	assert.NotEmpty(t, added.ID)
	assert.Equal(t, plain.ID, added.UserID)
	assert.True(t, sensorRepo.Exists(added.ID))
}

func TestPublish_KicksSlowSubscriberOnce(t *testing.T) {
	cs := &server{
		subscribers:    make(map[*subscriber]struct{}),
		publishLimiter: rate.NewLimiter(rate.Inf, 0),
	}
	var kicks atomic.Int32
	sub := &subscriber{msgs: make(chan []byte, 1), closeSlow: func() { kicks.Add(1) }}
	cs.addSubscriber(sub)

	for i := 0; i < 20; i++ {
		require.True(t, cs.publish([]byte(`{"ok":true}`)))
	}
	assert.Eventually(t, func() bool { return kicks.Load() == 1 }, time.Second, 10*time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(1), kicks.Load(), "closeSlow must run once per subscriber, not once per dropped message")
}

// jwtauth.Verifier also reads a "jwt" cookie; the router restricts it to the
// Authorization header. A valid token presented only in a cookie must not
// authenticate, so wildcard CORS cannot be turned into a CSRF path.
func TestAuthenticator_IgnoresJWTCookie(t *testing.T) {
	tmp := t.TempDir()
	db, err := sqlx.Connect("sqlite", filepath.Join(tmp, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	eventRepo, err := repos.NewEventRepo(db)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "build"), 0755))

	cs := &server{eventRepo: eventRepo, fs: os.DirFS(tmp), cfg: Config{JWTSecret: "test-secret"}}
	mux, err := newRouter(cs)
	require.NoError(t, err)

	token, err := entities.NewToken(entities.NewTokenAuth("test-secret"), entities.User{ID: "owner-1"})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: token})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code, "a token in a cookie must not authenticate")

	// The same token in the Authorization header still works.
	req = httptest.NewRequest(http.MethodGet, "/api/events", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func newQueriesTestServer(t *testing.T) (do func(method, target, body string) *httptest.ResponseRecorder, repo *repos.QueryRepo) {
	t.Helper()
	tmp := t.TempDir()
	db, err := sqlx.Connect("sqlite", filepath.Join(tmp, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	queryRepo, err := repos.NewQueryRepo(db)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "build"), 0755))

	cs := &server{
		queryRepo: queryRepo,
		fs:        os.DirFS(tmp),
		cfg:       Config{JWTSecret: "test-secret"},
	}
	mux, err := newRouter(cs)
	require.NoError(t, err)
	token, err := entities.NewToken(entities.NewTokenAuth("test-secret"), entities.User{ID: "owner-1"})
	require.NoError(t, err)

	return func(method, target, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		return w
	}, queryRepo
}

func listQueries(t *testing.T, do func(method, target, body string) *httptest.ResponseRecorder) []entities.Query {
	t.Helper()
	w := do(http.MethodGet, "/api/queries", "")
	require.Equal(t, http.StatusOK, w.Code)
	var qs []entities.Query
	require.NoError(t, json.NewDecoder(w.Body).Decode(&qs))
	return qs
}

func TestQueries_TagsThroughTheAPI(t *testing.T) {
	do, _ := newQueriesTestServer(t)

	w := do(http.MethodPost, "/api/queries", `{"content":"tcp.port == 445","description":"smb","active":true,
		"tags":[{"type":" malware ","value":"mirai"},{"type":"malware","value":"mirai"},{"type":"protocol","value":"smb"}]}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var created entities.Query
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	want := []entities.Tag{{Type: "malware", Value: "mirai"}, {Type: "protocol", Value: "smb"}}
	assert.Equal(t, want, created.Tags, "tags are trimmed and de-duplicated")
	assert.Equal(t, want, listQueries(t, do)[0].Tags)

	patch := func(body string) *httptest.ResponseRecorder {
		return do(http.MethodPatch, "/api/queries/"+created.ID, `{"id":"`+created.ID+`",`+body+`}`)
	}

	// Leaving tags out of the update keeps them, as it does for every other field.
	require.Equal(t, http.StatusOK, patch(`"description":"renamed"`).Code)
	got := listQueries(t, do)[0]
	assert.Equal(t, "renamed", got.Description)
	assert.Equal(t, want, got.Tags)

	require.Equal(t, http.StatusOK, patch(`"tags":[{"type":"severity","value":"high"}]`).Code)
	assert.Equal(t, []entities.Tag{{Type: "severity", Value: "high"}}, listQueries(t, do)[0].Tags)

	require.Equal(t, http.StatusOK, patch(`"tags":[]`).Code)
	assert.Empty(t, listQueries(t, do)[0].Tags)
}

func TestQueries_RejectsInvalidTags(t *testing.T) {
	do, _ := newQueriesTestServer(t)

	w := do(http.MethodPost, "/api/queries", `{"content":"c","description":"d","tags":[{"type":"t","value":""}]}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, listQueries(t, do), "a rejected create stores nothing")

	w = do(http.MethodPost, "/api/queries", `{"content":"c","description":"d","tags":[{"type":"t","value":"v"}]}`)
	require.Equal(t, http.StatusOK, w.Code)
	var q entities.Query
	require.NoError(t, json.NewDecoder(w.Body).Decode(&q))

	long := strings.Repeat("x", entities.MaxTagLength+1)
	w = do(http.MethodPatch, "/api/queries/"+q.ID, `{"id":"`+q.ID+`","description":"changed","tags":[{"type":"t","value":"`+long+`"}]}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	got := listQueries(t, do)[0]
	assert.Equal(t, "d", got.Description, "a rejected update changes nothing")
	assert.Equal(t, []entities.Tag{{Type: "t", Value: "v"}}, got.Tags)
}

// The update body is decoded over the stored query. Decoding a JSON array into a
// populated slice reuses its elements, so without clearing the stored tags first a
// tag sent with no value would quietly take the value of the tag it replaced.
func TestQueries_UpdatedTagsDoNotInheritStoredValues(t *testing.T) {
	do, _ := newQueriesTestServer(t)
	w := do(http.MethodPost, "/api/queries", `{"content":"c","description":"d","tags":[{"type":"a","value":"stored"}]}`)
	require.Equal(t, http.StatusOK, w.Code)
	var q entities.Query
	require.NoError(t, json.NewDecoder(w.Body).Decode(&q))

	w = do(http.MethodPatch, "/api/queries/"+q.ID, `{"id":"`+q.ID+`","tags":[{"type":"b"}]}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, "a tag without a value is invalid, not {b, stored}")
	assert.Equal(t, []entities.Tag{{Type: "a", Value: "stored"}}, listQueries(t, do)[0].Tags)
}

// Tags already stored are not validated again when an update leaves them out, so an
// edit to another field still works if they would fail today's limits.
func TestQueries_UpdateKeepsStoredTagsWithoutRevalidating(t *testing.T) {
	do, repo := newQueriesTestServer(t)
	long := []entities.Tag{{Type: "note", Value: strings.Repeat("x", entities.MaxTagLength+10)}}
	q, err := repo.Create("owner-1", "tcp.port == 22", "ssh", true, long)
	require.NoError(t, err)

	w := do(http.MethodPatch, "/api/queries/"+q.ID, `{"id":"`+q.ID+`","description":"renamed"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	got := listQueries(t, do)[0]
	assert.Equal(t, "renamed", got.Description)
	assert.Equal(t, long, got.Tags)
}
