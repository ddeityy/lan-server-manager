package config

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"

	"lan-server-manager/internal/logger"
)

//go:embed config.toml
var defaultConfig []byte

// ServerPreset describes a server to open automatically on startup.
type ServerPreset struct {
	Address       string `toml:"address"`
	RCONPassword  string `toml:"rcon_password"`
	Password      string `toml:"password"`
	ContainerName string `toml:"container_name"`
	SSHHost       string `toml:"ssh_host"`
	SSHUser       string `toml:"ssh_user"`
	SSHPassword   string `toml:"ssh_password"`
	SSHKeyPath    string `toml:"ssh_key_path"`
}

// Config holds runtime configuration loaded from a TOML file.
type Config struct {
	Maps    []string       `toml:"maps"`
	Configs []string       `toml:"configs"`
	Servers []ServerPreset `toml:"servers"`
}

// Default returns the embedded TOML configuration as a fallback.
func Default() Config {
	cfg, err := LoadBytes(defaultConfig)
	if err != nil {
		// The embedded file is known-good, so this should never happen.
		logger.Errorf("Failed to parse embedded default config: %v", err)
		return Config{}
	}
	logger.Infof("Loaded embedded default config")
	return cfg
}

// Load reads a TOML configuration file and falls back to Default() values
// for any missing top-level keys.
func Load(path string) (Config, error) {
	logger.Infof("Loading config from %s", path)
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	return LoadBytes(data)
}

// MapsOrDefault returns the configured map list, falling back to the
// compiled-in defaults when empty.
func (c Config) MapsOrDefault() []string {
	if len(c.Maps) > 0 {
		return c.Maps
	}
	return Default().Maps
}

// ConfigsOrDefault returns the configured exec config list, falling back to
// the compiled-in defaults when empty.
func (c Config) ConfigsOrDefault() []string {
	if len(c.Configs) > 0 {
		return c.Configs
	}
	return Default().Configs
}

// LoadBytes parses TOML data and applies defaults when a list is empty.
func LoadBytes(data []byte) (Config, error) {
	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("parse config file: %w", err)
	}

	if len(c.Maps) == 0 {
		logger.Warnf("No maps in config, using defaults")
		c.Maps = Default().Maps
	}
	if len(c.Configs) == 0 {
		logger.Warnf("No configs in config, using defaults")
		c.Configs = Default().Configs
	}

	logger.Infof("Parsed config with %d maps, %d configs, %d servers", len(c.Maps), len(c.Configs), len(c.Servers))
	return c, nil
}
