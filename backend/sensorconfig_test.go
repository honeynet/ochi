package backend

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/honeynet/ochi/backend/entities"
	"github.com/honeynet/ochi/backend/repos"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const (
	sensorMine   = "11111111-1111-1111-1111-111111111111"
	sensorTheirs = "22222222-2222-2222-2222-222222222222"
	tokenMine    = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	tokenTheirs  = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
)

// roundTripperFunc adapts a function to an http.RoundTripper, so a test can stand in for
// the network the Glutton defaults are fetched over.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// failingClient makes every outbound request fail, so gluttonDefault falls back to the
// embedded copy. This keeps the tests that check served content off the network.
func failingClient() *http.Client {
	return &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network disabled in test")
	})}
}

// newSensorConfigTestServer serves one sensor for owner-1 and one for owner-2, each with a
// known token. The returned caller authenticates with a given token (empty for none) and
// applies any extra request headers. httpClient stands in for the Glutton fetch.
func newSensorConfigTestServer(t *testing.T, httpClient *http.Client) func(method, target, token string, headers ...string) *httptest.ResponseRecorder {
	t.Helper()
	tmp := t.TempDir()
	db, err := sqlx.Connect("sqlite", filepath.Join(tmp, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	sensorRepo, err := repos.NewSensorRepo(db)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "build"), 0755))
	require.NoError(t, sensorRepo.AddSensors(entities.Sensor{ID: sensorMine, Name: "mine", UserID: "owner-1", Token: tokenMine}))
	require.NoError(t, sensorRepo.AddSensors(entities.Sensor{ID: sensorTheirs, Name: "theirs", UserID: "owner-2", Token: tokenTheirs}))

	cs := &server{
		sensorRepo: sensorRepo,
		fs:         os.DirFS(tmp),
		httpClient: httpClient,
		cfg:        Config{JWTSecret: "test-secret", PublishToken: "s3cr3t-token"},
	}
	mux, err := newRouter(cs)
	require.NoError(t, err)

	return func(method, target, token string, headers ...string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, target, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		for i := 0; i+1 < len(headers); i += 2 {
			req.Header.Set(headers[i], headers[i+1])
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		return w
	}
}

// A sensor fetches its own files with its token; the token must match the sensor in the
// path, and a request without a valid one learns nothing.
func TestSensorConfig_SensorTokenAuth(t *testing.T) {
	do := newSensorConfigTestServer(t, failingClient())

	for _, path := range []string{"/api/sensors/" + sensorMine + "/config", "/api/sensors/" + sensorMine + "/rules"} {
		assert.Equal(t, http.StatusUnauthorized, do(http.MethodGet, path, "").Code, "no token: "+path)
		assert.Equal(t, http.StatusUnauthorized, do(http.MethodGet, path, "not-a-real-token").Code, "unknown token: "+path)
	}

	// A valid token, but for a different sensor than the one in the path, is reported like
	// a sensor that does not exist, so a token cannot read another sensor's config.
	crossed := do(http.MethodGet, "/api/sensors/"+sensorMine+"/config", tokenTheirs)
	assert.Equal(t, http.StatusNotFound, crossed.Code)
	assert.NotContains(t, crossed.Body.String(), "s3cr3t-token", "a refused request must not leak the token")

	assert.Equal(t, http.StatusBadRequest, do(http.MethodGet, "/api/sensors/not-a-uuid/config", tokenMine).Code)
}

func TestSensorConfig_FilledInForThisInstance(t *testing.T) {
	do := newSensorConfigTestServer(t, failingClient())

	w := do(http.MethodGet, "/api/sensors/"+sensorMine+"/config", tokenMine)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Header().Get("Content-Type"), "yaml")
	assert.Contains(t, w.Header().Get("Content-Disposition"), `filename="config.yaml"`)

	var config struct {
		Producers struct {
			Enabled bool `yaml:"enabled"`
			HTTP    struct {
				Enabled bool   `yaml:"enabled"`
				Remote  string `yaml:"remote"`
			} `yaml:"http"`
		} `yaml:"producers"`
		RulesPath string `yaml:"rules_path"`
		Ports     struct {
			TCP int `yaml:"tcp"`
		} `yaml:"ports"`
	}
	require.NoError(t, yaml.Unmarshal(w.Body.Bytes(), &config))

	// Publishing is switched on and aimed at this instance, so the file works as it is.
	assert.True(t, config.Producers.Enabled)
	assert.True(t, config.Producers.HTTP.Enabled)
	assert.Equal(t, "http://example.com/api/publish?token=s3cr3t-token", config.Producers.HTTP.Remote)

	// The rest of Glutton's defaults are untouched.
	assert.Equal(t, "config/rules.yaml", config.RulesPath)
	assert.Equal(t, 5000, config.Ports.TCP)

	// Glutton's own comments survive, so the file is still readable.
	assert.Contains(t, w.Body.String(), "# bytes")
}

// Behind a reverse proxy the sensor has to publish to the public address, not to the one
// the proxy forwarded to.
func TestSensorConfig_UsesForwardedHost(t *testing.T) {
	do := newSensorConfigTestServer(t, failingClient())

	w := do(http.MethodGet, "/api/sensors/"+sensorMine+"/config", tokenMine,
		"X-Forwarded-Proto", "https", "X-Forwarded-Host", "ochi.example.org")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "https://ochi.example.org/api/publish")
	assert.NotContains(t, w.Body.String(), "http://example.com/api/publish")
}

func TestSensorRules_ServesGluttonDefaults(t *testing.T) {
	do := newSensorConfigTestServer(t, failingClient())

	w := do(http.MethodGet, "/api/sensors/"+sensorMine+"/rules", tokenMine)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Disposition"), `filename="rules.yaml"`)
	assert.NotContains(t, w.Body.String(), "s3cr3t-token", "the rules hold no secrets")

	var rules struct {
		Rules []struct {
			Match  string `yaml:"match"`
			Type   string `yaml:"type"`
			Target string `yaml:"target"`
		} `yaml:"rules"`
	}
	require.NoError(t, yaml.Unmarshal(w.Body.Bytes(), &rules))
	require.NotEmpty(t, rules.Rules)

	// The file is Glutton's, so it carries its handlers.
	var targets []string
	for _, rule := range rules.Rules {
		targets = append(targets, rule.Target)
	}
	assert.Contains(t, targets, "telnet")
	assert.True(t, strings.HasPrefix(rules.Rules[0].Match, "tcp "), rules.Rules[0].Match)
}

// The defaults are fetched from the Glutton repo, and only fall back to the embedded copy
// when the fetch cannot be made.
func TestSensorConfig_FetchesFromGluttonWithFallback(t *testing.T) {
	// A client that serves recognisable content for the fetch, so a served file that
	// carries the marker came from the fetch rather than the embedded fallback.
	embeddedConfig, err := os.ReadFile(filepath.Join("handlers", "sensordefaults", "config.yaml"))
	require.NoError(t, err)
	fetched := append([]byte("# fetched-from-glutton\n"), embeddedConfig...)

	serving := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		require.True(t, strings.HasPrefix(r.URL.String(), "https://raw.githubusercontent.com/mushorg/glutton/"), r.URL.String())
		body := fetched
		if strings.HasSuffix(r.URL.Path, "rules.yaml") {
			body = []byte("rules:\n  - match: tcp dst port 23\n    type: conn_handler\n    target: telnet\n")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}

	do := newSensorConfigTestServer(t, serving)
	w := do(http.MethodGet, "/api/sensors/"+sensorMine+"/config", tokenMine)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "# fetched-from-glutton", "served the fetched file")
	assert.Contains(t, w.Body.String(), "/api/publish?token=s3cr3t-token", "still filled in for this instance")

	// With the fetch failing, the same endpoint serves the embedded copy, which does not
	// carry the marker, so the server keeps working with no network.
	doOffline := newSensorConfigTestServer(t, failingClient())
	off := doOffline(http.MethodGet, "/api/sensors/"+sensorMine+"/config", tokenMine)
	require.Equal(t, http.StatusOK, off.Code)
	assert.NotContains(t, off.Body.String(), "# fetched-from-glutton")
	assert.Contains(t, off.Body.String(), "# bytes", "the embedded default")
}
