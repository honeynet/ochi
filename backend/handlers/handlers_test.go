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
	"github.com/stretchr/testify/assert"
)

// withDownloadParams attaches the chi URL params DownloadBinaryHandler reads.
func withDownloadParams(r *http.Request, osType, arch string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("os", osType)
	rctx.URLParams.Add("arch", arch)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
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

	h := &Handlers{}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/download/"+osType+"/"+arch, nil)
	r = withDownloadParams(r, osType, arch)

	h.DownloadBinaryHandler(w, r)

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

	h := &Handlers{}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/download/"+tc.os+"/"+tc.arch, nil)
			r = withDownloadParams(r, tc.os, tc.arch)

			h.DownloadBinaryHandler(w, r)

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

	h := &Handlers{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/download/linux/amd64", nil)
	r = withDownloadParams(r, "linux", "amd64")

	h.DownloadBinaryHandler(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDownloadBinaryHandler_MissingPlaceholder(t *testing.T) {
	content := []byte("binary-without-placeholder")

	osType, arch, cleanup := setupDownloadBinaryTest(t, content)
	defer cleanup()

	h := &Handlers{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/download/"+osType+"/"+arch, nil)
	r = withDownloadParams(r, osType, arch)

	h.DownloadBinaryHandler(w, r)

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
