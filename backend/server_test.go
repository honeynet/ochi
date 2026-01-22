package backend

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

func TestDownloadBinaryHandler(t *testing.T) {

	osType := "linux"
	arch := "amd64"
	binaryName := "sensor-" + osType + "-" + arch
	binDir := "bin"

	tmpDir := t.TempDir()

	originalWD, err := os.Getwd()
	assert.NoError(t, err)
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)
	defer os.Chdir(originalWD)

	err = os.MkdirAll(binDir, 0755)
	assert.NoError(t, err)

	placeHolderUUID := "00000000-0000-0000-0000-000000000000"
	prefix := "some-prefix-bytes-"
	suffix := "-some-suffix-bytes"
	content := []byte(prefix + placeHolderUUID + suffix)
	filePath := filepath.Join(binDir, binaryName)
	err = os.WriteFile(filePath, content, 0644)
	assert.NoError(t, err)

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
	assert.False(t, bytes.Contains(body, []byte(placeHolderUUID)))
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
