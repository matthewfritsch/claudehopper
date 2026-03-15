package config

import (
	"encoding/json"
	"errors"
	"os"
)

// Config holds the claudehopper top-level configuration stored in config.json.
// The JSON format uses 2-space-indented JSON with a trailing newline.
type Config struct {
	Active string `json:"active"`
}

// LoadConfig reads and parses config.json at path. If the file does not exist,
// LoadConfig returns a zero Config and nil error. Any other read or parse
// error is returned as-is.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// SaveConfig writes cfg to path as 2-space-indented JSON with a trailing newline.
func SaveConfig(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	// Append trailing newline for clean file endings
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}
