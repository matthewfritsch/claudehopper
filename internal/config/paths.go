// Package config provides path resolution and JSON serialization for
// claudehopper configuration files. Paths are resolved using
// os.UserConfigDir which respects XDG_CONFIG_HOME on Linux and never
// stores tilde (~) strings.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ConfigDir returns the claudehopper configuration directory.
// If CLAUDEHOPPER_HOME is set, it is used as-is (for development/testing).
// Otherwise, respects XDG_CONFIG_HOME on Linux; falls back to
// $HOME/.config/claudehopper. The returned path is always absolute.
func ConfigDir() (string, error) {
	if override := os.Getenv("CLAUDEHOPPER_HOME"); override != "" {
		return override, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine config dir: %w", err)
	}
	return filepath.Join(base, "claudehopper"), nil
}

// ProfilesDir returns the directory that contains all profile subdirectories.
func ProfilesDir() (string, error) {
	cfg, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "profiles"), nil
}

// ProfileDir returns the directory for the named profile. This is a
// convenience function used by Phase 2 profile operations.
func ProfileDir(name string) (string, error) {
	profiles, err := ProfilesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(profiles, name), nil
}

// ConfigFilePath returns the path to config.json within the config directory.
func ConfigFilePath() (string, error) {
	cfg, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "config.json"), nil
}
