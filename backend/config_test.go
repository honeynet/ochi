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

	assert.Equal(t, defaultConfig(), cfg)

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
listen_address: localhost:3000
publish_token: ""
jwt_secret: secret
database_path: ./data.db
`), 0o600))

	_, err := LoadConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "publish_token")
}

func TestLoadConfig_RejectsInvalidPublishRate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
listen_address: localhost:3000
publish_token: token
jwt_secret: secret
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
	assert.Equal(t, defaultConfig(), cfg)

	_, err = os.Stat(defaultConfigPath)
	require.NoError(t, err)
}
