package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigDir_XDGOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	got, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error: %v", err)
	}
	want := filepath.Join(tmp, "claudehopper")
	if got != want {
		t.Errorf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestConfigDir_Default(t *testing.T) {
	// Clear XDG_CONFIG_HOME so os.UserConfigDir uses $HOME/.config
	t.Setenv("XDG_CONFIG_HOME", "")

	got, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error: %v", err)
	}
	if !strings.HasSuffix(got, "/claudehopper") {
		t.Errorf("ConfigDir() = %q, want suffix /claudehopper", got)
	}
}

func TestConfigDir_NoTilde(t *testing.T) {
	got, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error: %v", err)
	}
	if strings.Contains(got, "~") {
		t.Errorf("ConfigDir() = %q contains tilde character", got)
	}
}

func TestConfigDir_AbsolutePath(t *testing.T) {
	got, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("ConfigDir() = %q is not an absolute path", got)
	}
}

func TestProfilesDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	got, err := ProfilesDir()
	if err != nil {
		t.Fatalf("ProfilesDir() error: %v", err)
	}
	want := filepath.Join(tmp, "claudehopper", "profiles")
	if got != want {
		t.Errorf("ProfilesDir() = %q, want %q", got, want)
	}
}

func TestProfileDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	got, err := ProfileDir("myprofile")
	if err != nil {
		t.Fatalf("ProfileDir() error: %v", err)
	}
	want := filepath.Join(tmp, "claudehopper", "profiles", "myprofile")
	if got != want {
		t.Errorf("ProfileDir() = %q, want %q", got, want)
	}
}

func TestConfigFilePath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	got, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath() error: %v", err)
	}
	want := filepath.Join(tmp, "claudehopper", "config.json")
	if got != want {
		t.Errorf("ConfigFilePath() = %q, want %q", got, want)
	}
}
