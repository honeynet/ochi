package backend

import (
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
	sensorMine    = "11111111-1111-1111-1111-111111111111"
	sensorTheirs  = "22222222-2222-2222-2222-222222222222"
	sensorMissing = "33333333-3333-3333-3333-333333333333"
)

// newSensorConfigTestServer serves one sensor for owner-1 and one for owner-2. The
// returned caller authenticates as owner-1 and applies any extra request headers.
func newSensorConfigTestServer(t *testing.T) func(method, target string, authenticated bool, headers ...string) *httptest.ResponseRecorder {
	t.Helper()
	tmp := t.TempDir()
	db, err := sqlx.Connect("sqlite", filepath.Join(tmp, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	sensorRepo, err := repos.NewSensorRepo(db)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "build"), 0755))
	require.NoError(t, sensorRepo.AddSensors(entities.Sensor{ID: sensorMine, Name: "mine", UserID: "owner-1"}))
	require.NoError(t, sensorRepo.AddSensors(entities.Sensor{ID: sensorTheirs, Name: "theirs", UserID: "owner-2"}))

	cs := &server{
		sensorRepo: sensorRepo,
		fs:         os.DirFS(tmp),
		cfg:        Config{JWTSecret: "test-secret", PublishToken: "s3cr3t-token"},
	}
	mux, err := newRouter(cs)
	require.NoError(t, err)

	token, err := entities.NewToken(entities.NewTokenAuth("test-secret"), entities.User{ID: "owner-1"})
	require.NoError(t, err)

	return func(method, target string, authenticated bool, headers ...string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, target, nil)
		if authenticated {
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

// The config carries the publish token, which the whole instance shares, so knowing a
// sensor id must not be enough to read it.
func TestSensorConfig_OwnerOnly(t *testing.T) {
	do := newSensorConfigTestServer(t)

	for _, path := range []string{"/api/sensors/" + sensorMine + "/config", "/api/sensors/" + sensorMine + "/rules"} {
		assert.Equal(t, http.StatusUnauthorized, do(http.MethodGet, path, false).Code, path)
	}

	// Another user's sensor is reported like one that does not exist.
	theirs := do(http.MethodGet, "/api/sensors/"+sensorTheirs+"/config", true)
	assert.Equal(t, http.StatusNotFound, theirs.Code)
	assert.Equal(t, http.StatusNotFound, do(http.MethodGet, "/api/sensors/"+sensorMissing+"/config", true).Code)
	assert.NotContains(t, theirs.Body.String(), "s3cr3t-token", "a refused request must not leak the token")

	assert.Equal(t, http.StatusBadRequest, do(http.MethodGet, "/api/sensors/not-a-uuid/config", true).Code)
}

func TestSensorConfig_FilledInForThisInstance(t *testing.T) {
	do := newSensorConfigTestServer(t)

	w := do(http.MethodGet, "/api/sensors/"+sensorMine+"/config", true)
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
	do := newSensorConfigTestServer(t)

	w := do(http.MethodGet, "/api/sensors/"+sensorMine+"/config", true,
		"X-Forwarded-Proto", "https", "X-Forwarded-Host", "ochi.example.org")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "https://ochi.example.org/api/publish")
	assert.NotContains(t, w.Body.String(), "http://example.com/api/publish")
}

func TestSensorRules_ServesGluttonDefaults(t *testing.T) {
	do := newSensorConfigTestServer(t)

	w := do(http.MethodGet, "/api/sensors/"+sensorMine+"/rules", true)
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
