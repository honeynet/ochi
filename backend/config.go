package backend

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

const defaultConfigPath = "config.yaml"

// devModeEnv lets a local setup keep the published secrets below. The check is on the
// secret itself rather than on the bind address, so the recommended "reverse proxy in
// front, server on localhost" deployment is not rejected for listening on localhost.
const devModeEnv = "OCHI_DEV"

// knownSecrets are the values this project shipped as defaults. They are in the README
// and in config.docker.yaml, so anyone can read them: a deployment still using one is
// not configured, whatever the file says.
var knownSecrets = map[string]bool{"token": true, "secret": true}

// newSecret returns 32 random bytes, encoded for a YAML scalar.
func newSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func devMode() bool { return os.Getenv(devModeEnv) != "" }

// Config holds service configuration loaded from a YAML file.
type Config struct {
	ListenAddress     string  `yaml:"listen_address"`
	PublishToken      string  `yaml:"publish_token"`
	JWTSecret         string  `yaml:"jwt_secret"`
	DatabasePath      string  `yaml:"database_path"`
	PublishRatePerSec float64 `yaml:"publish_rate_per_sec"`
	PublishBurst      int     `yaml:"publish_burst"`
	// RequireRegisteredSensors rejects published events whose sensor UUID is not
	// in the sensors table. Off by default: sensors that were never registered
	// through /sensors, including ones provisioned by /download, would stop being
	// accepted.
	RequireRegisteredSensors bool `yaml:"require_registered_sensors"`
	// AdminEmails are granted the admin role when they log in. Removing an
	// address does not revoke a role that was already granted.
	AdminEmails []string `yaml:"admin_emails,omitempty"`
}

// defaultConfig deliberately carries no secrets. It is also the base the config file is
// unmarshalled over, so a secret named here would be inherited in silence by any config
// that happens to leave that key out.
func defaultConfig() Config {
	return Config{
		ListenAddress:            "localhost:3000",
		DatabasePath:             "./data.db",
		PublishRatePerSec:        100,
		PublishBurst:             50,
		RequireRegisteredSensors: false,
	}
}

func (c Config) validate() error {
	if c.ListenAddress == "" {
		return fmt.Errorf("listen_address is required")
	}
	if c.PublishToken == "" {
		return fmt.Errorf("publish_token is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("jwt_secret is required")
	}
	if !devMode() {
		for field, value := range map[string]string{
			"publish_token": c.PublishToken,
			"jwt_secret":    c.JWTSecret,
		} {
			if knownSecrets[value] {
				return fmt.Errorf(
					"%s is one of this project's published default secrets; replace it, or set %s to keep it for local development",
					field, devModeEnv)
			}
		}
	}
	if c.DatabasePath == "" {
		return fmt.Errorf("database_path is required")
	}
	if c.PublishRatePerSec <= 0 {
		return fmt.Errorf("publish_rate_per_sec must be positive")
	}
	if c.PublishBurst < 1 {
		return fmt.Errorf("publish_burst must be at least 1")
	}
	return nil
}

func writeDefaultConfig(path string) error {
	cfg := defaultConfig()
	var err error
	if cfg.PublishToken, err = newSecret(); err != nil {
		return err
	}
	if cfg.JWTSecret, err = newSecret(); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal default config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write default config %q: %w", path, err)
	}
	return nil
}

// LoadConfig reads configuration from path. If the file does not exist, it is
// created with defaults suitable for local testing.
func LoadConfig(path string) (Config, error) {
	if path == "" {
		path = defaultConfigPath
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := writeDefaultConfig(path); err != nil {
			return Config{}, err
		}
	} else if err != nil {
		return Config{}, fmt.Errorf("stat config %q: %w", path, err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	cfg := defaultConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	// A config that leaves a secret out gets a generated one for this run rather than a
	// value an attacker could look up. It is not written back: the file belongs to whoever
	// wrote it, and in the container it is not ours to write. Sessions and the publish
	// token therefore last only as long as the process, which the log says plainly.
	for _, secret := range []struct {
		field string
		into  *string
	}{{"publish_token", &cfg.PublishToken}, {"jwt_secret", &cfg.JWTSecret}} {
		if *secret.into != "" {
			continue
		}
		if *secret.into, err = newSecret(); err != nil {
			return Config{}, err
		}
		log.Printf("config %q sets no %s: generated one for this run only; set it in the file to keep it across restarts",
			path, secret.field)
	}
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config %q: %w", path, err)
	}
	return cfg, nil
}
