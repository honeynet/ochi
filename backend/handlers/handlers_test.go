package handlers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/honeynet/ochi/backend/entities"
	"github.com/honeynet/ochi/backend/repos"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"
)

// withDownloadParams attaches the chi URL params DownloadBinary reads.
func withDownloadParams(r *http.Request, sensorID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sensor_id", sensorID)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

const (
	testSensorID = "11111111-2222-3333-4444-555555555555"
	testUserID   = "user-1"
)

// newDownloadHandlers returns Handlers backed by a database holding one sensor owned by testUserID.
func newDownloadHandlers(t *testing.T) *Handlers {
	t.Helper()
	db, err := sqlx.Connect("sqlite", filepath.Join(t.TempDir(), "test.db"))
	assert.NoError(t, err)
	repo, err := repos.NewSensorRepo(db)
	assert.NoError(t, err)
	assert.NoError(t, repo.AddSensors(entities.Sensor{ID: testSensorID, UserID: testUserID, Name: "s"}))
	return &Handlers{Sensors: repo}
}

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

	h := newDownloadHandlers(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/download/"+testSensorID+"?os="+osType+"&arch="+arch+"&user_id="+testUserID, nil)
	r = withDownloadParams(r, testSensorID)

	h.DownloadBinary(w, r)

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

	assert.True(t, bytes.Contains(body, []byte(testSensorID)))
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

	h := newDownloadHandlers(t)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/api/download/"+testSensorID+"?os="+tc.os+"&arch="+tc.arch+"&user_id="+testUserID, nil)
			r = withDownloadParams(r, testSensorID)

			h.DownloadBinary(w, r)

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

	h := newDownloadHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/download/"+testSensorID+"?os=linux&arch=amd64&user_id="+testUserID, nil)
	r = withDownloadParams(r, testSensorID)

	h.DownloadBinary(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDownloadBinaryHandler_MissingPlaceholder(t *testing.T) {
	content := []byte("binary-without-placeholder")

	osType, arch, cleanup := setupDownloadBinaryTest(t, content)
	defer cleanup()

	h := newDownloadHandlers(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/download/"+testSensorID+"?os="+osType+"&arch="+arch+"&user_id="+testUserID, nil)
	r = withDownloadParams(r, testSensorID)

	h.DownloadBinary(w, r)

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

func TestDownloadBinaryHandler_Unauthorized(t *testing.T) {
	content := []byte("pre-00000000-0000-0000-0000-000000000000-post")
	osType, arch, cleanup := setupDownloadBinaryTest(t, content)
	defer cleanup()

	h := newDownloadHandlers(t)

	tests := []struct {
		name     string
		sensorID string
		query    string
		want     int
	}{
		{"WrongUser", testSensorID, "&user_id=other", http.StatusNotFound},
		{"UnknownSensor", "99999999-2222-3333-4444-555555555555", "&user_id=" + testUserID, http.StatusNotFound},
		{"MissingUser", testSensorID, "", http.StatusBadRequest},
		{"BadSensorID", "nope", "&user_id=" + testUserID, http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/api/download/"+tc.sensorID+"?os="+osType+"&arch="+arch+tc.query, nil)
			r = withDownloadParams(r, tc.sensorID)
			h.DownloadBinary(w, r)
			assert.Equal(t, tc.want, w.Code)
		})
	}
}
