package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_ReadsActive(t *testing.T) {
	// Create a temp config file with known content
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"active": "gsd"}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if cfg.Active != "gsd" {
		t.Errorf("cfg.Active = %q, want %q", cfg.Active, "gsd")
	}
}

func TestSaveConfig_WritesCorrectFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := Config{Active: "gsd"}
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Must be 2-space indented with trailing newline
	want := "{\n  \"active\": \"gsd\"\n}\n"
	if string(data) != want {
		t.Errorf("SaveConfig output:\ngot:  %q\nwant: %q", string(data), want)
	}
}

func TestLoadConfig_FixtureRoundTrip(t *testing.T) {
	// Load from fixture, save to temp, compare bytes
	fixture := filepath.Join("testdata", "config.json")
	original, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	cfg, err := LoadConfig(fixture)
	if err != nil {
		t.Fatalf("LoadConfig(fixture): %v", err)
	}

	dir := t.TempDir()
	out := filepath.Join(dir, "config.json")
	if err := SaveConfig(out, cfg); err != nil {
		t.Fatalf("SaveConfig(): %v", err)
	}

	saved, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(original, saved) {
		t.Errorf("round-trip mismatch:\noriginal: %q\nsaved:    %q", original, saved)
	}
}

func TestLoadConfig_MissingFile_ReturnsZero(t *testing.T) {
	cfg, err := LoadConfig("/does/not/exist/config.json")
	if err != nil {
		t.Fatalf("LoadConfig(missing) error: %v, want nil", err)
	}
	if cfg.Active != "" {
		t.Errorf("cfg.Active = %q, want empty string for missing file", cfg.Active)
	}
}
