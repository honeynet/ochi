package backend

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_CreatesDefaultWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg, err := LoadConfig(path)
	require.NoError(t, err)

	// Everything but the secrets comes from defaultConfig; the secrets are generated.
	want := defaultConfig()
	want.PublishToken, want.JWTSecret = cfg.PublishToken, cfg.JWTSecret
	assert.Equal(t, want, cfg)
	assertGenerated(t, cfg)

	// Written to the file, so they outlive the process instead of changing under sessions.
	written, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(written), cfg.PublishToken)
	assert.Contains(t, string(written), cfg.JWTSecret)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.True(t, info.Mode().Perm()&0o077 == 0, "config file should not be group/world readable")
}

func TestLoadConfig_ReadsExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
listen_address: 0.0.0.0:4000
publish_token: my-token
jwt_secret: my-secret
database_path: /tmp/ochi.db
`)
	require.NoError(t, os.WriteFile(path, content, 0o600))

	cfg, err := LoadConfig(path)
	require.NoError(t, err)

	assert.Equal(t, Config{
		ListenAddress:     "0.0.0.0:4000",
		PublishToken:      "my-token",
		JWTSecret:         "my-secret",
		DatabasePath:      "/tmp/ochi.db",
		PublishRatePerSec: 100,
		PublishBurst:      50,
	}, cfg)
}

func TestLoadConfig_RejectsInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("listen_address: [\n"), 0o600))

	_, err := LoadConfig(path)
	require.Error(t, err)
}

func TestLoadConfig_RejectsMissingRequiredFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
listen_address: ""
publish_token: a-real-token
jwt_secret: a-real-secret
database_path: ./data.db
`), 0o600))

	_, err := LoadConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listen_address")
}

// A config that leaves a secret out, or blanks it, is not configured. Generating one beats
// both refusing to start and falling back to a value published in this repository.
func TestLoadConfig_GeneratesSecretsTheFileDoesNotSet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
listen_address: localhost:3000
publish_token: ""
database_path: ./data.db
`), 0o600))

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	assertGenerated(t, cfg)

	// Not written back: the file belongs to whoever wrote it, and in the container it is
	// not ours to write.
	written, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(written), cfg.PublishToken)
}

func TestLoadConfig_RejectsPublishedDefaultSecrets(t *testing.T) {
	for field, body := range map[string]string{
		"publish_token": `
listen_address: localhost:3000
publish_token: token
jwt_secret: a-real-secret
database_path: ./data.db
`,
		"jwt_secret": `
listen_address: localhost:3000
publish_token: a-real-token
jwt_secret: secret
database_path: ./data.db
`,
	} {
		t.Run(field, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

			_, err := LoadConfig(path)
			require.Error(t, err)
			assert.Contains(t, err.Error(), field)
			assert.Contains(t, err.Error(), devModeEnv, "the error should say how to keep the value locally")
		})
	}
}

func TestLoadConfig_KeepsPublishedDefaultsInDevMode(t *testing.T) {
	t.Setenv(devModeEnv, "1")
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
listen_address: localhost:3000
publish_token: token
jwt_secret: secret
database_path: ./data.db
`), 0o600))

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "token", cfg.PublishToken)
	assert.Equal(t, "secret", cfg.JWTSecret)
}

func TestLoadConfig_GeneratesADifferentSecretEachTime(t *testing.T) {
	first, err := LoadConfig(filepath.Join(t.TempDir(), "config.yaml"))
	require.NoError(t, err)
	second, err := LoadConfig(filepath.Join(t.TempDir(), "config.yaml"))
	require.NoError(t, err)

	assert.NotEqual(t, first.PublishToken, second.PublishToken)
	assert.NotEqual(t, first.JWTSecret, second.JWTSecret)
}

// assertGenerated checks the secrets were generated rather than left at a published value.
func assertGenerated(t *testing.T, cfg Config) {
	t.Helper()
	for field, value := range map[string]string{
		"publish_token": cfg.PublishToken,
		"jwt_secret":    cfg.JWTSecret,
	} {
		assert.NotEmpty(t, value, field)
		assert.False(t, knownSecrets[value], "%s is a published default", field)
		assert.GreaterOrEqual(t, len(value), 32, "%s is too short to have been generated", field)
	}
	assert.NotEqual(t, cfg.PublishToken, cfg.JWTSecret, "the two secrets should differ")
}

func TestLoadConfig_RejectsInvalidPublishRate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
listen_address: localhost:3000
publish_token: a-real-token
jwt_secret: a-real-secret
database_path: ./data.db
publish_rate_per_sec: 0
`), 0o600))

	_, err := LoadConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "publish_rate_per_sec")
}

func TestLoadConfig_EmptyPathUsesDefaultName(t *testing.T) {
	dir := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	cfg, err := LoadConfig("")
	require.NoError(t, err)
	want := defaultConfig()
	want.PublishToken, want.JWTSecret = cfg.PublishToken, cfg.JWTSecret
	assert.Equal(t, want, cfg)
	assertGenerated(t, cfg)

	_, err = os.Stat(defaultConfigPath)
	require.NoError(t, err)
}
