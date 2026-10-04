package backend

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/honeynet/ochi/backend/entities"
	"github.com/honeynet/ochi/backend/repos"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func setupDownloadBinaryTest(t *testing.T, content []byte) (string, string, func()) {
	t.Helper()

	osType := "linux"
	arch := "amd64"
	binaryName := "sensor-" + osType + "-" + arch
	binDir := "bin"

	tmpDir := t.TempDir()

	originalWD, err := os.Getwd()
	assert.NoError(t, err)
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)

	cleanup := func() {
		os.Chdir(originalWD)
	}

	err = os.MkdirAll(binDir, 0755)
	assert.NoError(t, err)

	filePath := filepath.Join(binDir, binaryName)
	err = os.WriteFile(filePath, content, 0644)
	assert.NoError(t, err)

	return osType, arch, cleanup
}

func TestDownloadBinaryHandler(t *testing.T) {
	placeholderUUID := "00000000-0000-0000-0000-000000000000"
	prefix := "some-prefix-bytes-"
	suffix := "-some-suffix-bytes"
	content := []byte(prefix + placeholderUUID + suffix)

	osType, arch, cleanup := setupDownloadBinaryTest(t, content)
	defer cleanup()

	cs := &server{}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/download/"+osType+"/"+arch, nil)
	params := httprouter.Params{
		httprouter.Param{Key: "os", Value: osType},
		httprouter.Param{Key: "arch", Value: arch},
	}

	cs.downloadBinaryHandler(w, r, params)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	assert.Equal(t, "application/octet-stream", resp.Header.Get("Content-Type"))
	assert.Contains(t, resp.Header.Get("Content-Disposition"), "attachment; filename=\"sensor-"+osType+"-"+arch+"\"")
	assert.Equal(t, len(content), len(body))
	assert.False(t, bytes.Contains(body, []byte(placeholderUUID)))
	assert.True(t, bytes.Contains(body, []byte(prefix)))
	assert.True(t, bytes.Contains(body, []byte(suffix)))

	start := len(prefix)
	uuidLen := 36
	assert.GreaterOrEqual(t, len(body), start+uuidLen, "Response body is too short to contain the UUID")
	extractedUUID := string(body[start : start+uuidLen])
	_, err = uuid.Parse(extractedUUID)
	assert.NoError(t, err, "The injected string should be a valid UUID")
}

func TestDownloadBinaryHandler_InvalidParams(t *testing.T) {
	tests := []struct {
		name string
		os   string
		arch string
	}{
		{"InvalidOS", "invalid", "amd64"},
		{"InvalidArch", "linux", "invalid"},
		{"BothInvalid", "invalid", "invalid"},
	}

	cs := &server{}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/download/"+tc.os+"/"+tc.arch, nil)
			params := httprouter.Params{
				httprouter.Param{Key: "os", Value: tc.os},
				httprouter.Param{Key: "arch", Value: tc.arch},
			}

			cs.downloadBinaryHandler(w, r, params)

			resp := w.Result()
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestDownloadBinaryHandler_BinaryNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	originalWD, err := os.Getwd()
	assert.NoError(t, err)
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)
	defer os.Chdir(originalWD)

	cs := &server{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/download/linux/amd64", nil)
	params := httprouter.Params{
		httprouter.Param{Key: "os", Value: "linux"},
		httprouter.Param{Key: "arch", Value: "amd64"},
	}

	cs.downloadBinaryHandler(w, r, params)

	resp := w.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDownloadBinaryHandler_MissingPlaceholder(t *testing.T) {
	content := []byte("binary-without-placeholder")

	osType, arch, cleanup := setupDownloadBinaryTest(t, content)
	defer cleanup()

	cs := &server{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/download/"+osType+"/"+arch, nil)
	params := httprouter.Params{
		httprouter.Param{Key: "os", Value: osType},
		httprouter.Param{Key: "arch", Value: arch},
	}

	cs.downloadBinaryHandler(w, r, params)

	resp := w.Result()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestIndexReplace(t *testing.T) {
	placeholder := "00000000-0000-0000-0000-000000000000"
	newUUID := "11111111-2222-3333-4444-555555555555"
	data := []byte("prefix-" + placeholder + "-suffix")

	modified, ok := IndexReplace(data, []byte(placeholder), []byte(newUUID))
	assert.True(t, ok)
	assert.Equal(t, "prefix-"+newUUID+"-suffix", string(modified))

	_, ok = IndexReplace([]byte("no-match"), []byte(placeholder), []byte(newUUID))
	assert.False(t, ok)
}

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
	cs.publishHandler(w, r, nil)

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
	cs.publishHandler(w, r, nil)

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
	require.Less(t, len(body), publishMaxBodyBytes)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/publish", bytes.NewReader(body))
	cs.publishHandler(w, r, nil)

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
	r := httptest.NewRequest("POST", "/publish", bytes.NewBufferString(`{"dstPort":80}`))
	cs.publishHandler(w, r, nil)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestGetSharedEventByID_NoAuth(t *testing.T) {
	tmp := t.TempDir()
	db, err := sqlx.Connect("sqlite", filepath.Join(tmp, "test.db"))
	require.NoError(t, err)

	eventRepo, err := repos.NewEventRepo(db)
	require.NoError(t, err)

	created, err := eventRepo.Create(entities.Event{
		OwnerID:   "owner-1",
		Payload:   "cGF5bG9hZA==",
		DstPort:   80,
		Rule:      "Rule: TCP",
		Handler:   "http",
		Transport: "tcp",
		SensorID:  "sensor-1",
		SrcHost:   "1.2.3.4",
		SrcPort:   "4321",
		DstHost:   "198.51.100.8",
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
