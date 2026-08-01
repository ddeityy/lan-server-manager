package config

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

//go:embed config.toml
var defaultConfig []byte

// ServerPreset describes a server to open automatically on startup.
type ServerPreset struct {
	Address      string `toml:"address"`
	RCONPassword string `toml:"rcon_password"`
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
		return Config{}
	}
	return cfg
}

// Load reads a TOML configuration file and falls back to Default() values
// for any missing top-level keys.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	return LoadBytes(data)
}

// LoadBytes parses TOML data and applies defaults when a list is empty.
func LoadBytes(data []byte) (Config, error) {
	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("parse config file: %w", err)
	}

	if len(c.Maps) == 0 {
		c.Maps = Default().Maps
	}
	if len(c.Configs) == 0 {
		c.Configs = Default().Configs
	}

	return c, nil
}
