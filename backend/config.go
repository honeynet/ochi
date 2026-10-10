package backend

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const defaultConfigPath = "config.yaml"

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
}

func defaultConfig() Config {
	return Config{
		ListenAddress:            "localhost:3000",
		PublishToken:             "token",
		JWTSecret:                "secret",
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
	data, err := yaml.Marshal(defaultConfig())
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
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config %q: %w", path, err)
	}
	return cfg, nil
}
