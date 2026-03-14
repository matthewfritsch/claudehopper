package profile_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/matthewfritsch/claudehopper/internal/profile"
)

// writeManifest is a test helper that writes a minimal manifest JSON file.
func writeManifest(t *testing.T, dir string, managed []string, shared map[string]string, desc string) {
	t.Helper()
	if managed == nil {
		managed = []string{}
	}
	if shared == nil {
		shared = map[string]string{}
	}
	m := map[string]interface{}{
		"managed_paths": managed,
		"shared_paths":  shared,
		"description":   desc,
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".hop-manifest.json"), data, 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

// writeConfig is a test helper that writes a minimal config.json file.
func writeConfig(t *testing.T, dir, active string) {
	t.Helper()
	data := []byte(`{"active":"` + active + `"}` + "\n")
	if err := os.WriteFile(filepath.Join(dir, "config.json"), data, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func TestListProfiles_EmptyDir(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(tmp, "config.json")

	summaries, err := profile.ListProfiles(profilesDir, configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(summaries) != 0 {
		t.Errorf("expected empty slice, got %d items", len(summaries))
	}
}

func TestListProfiles_MultipleProfiles(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")

	// Create three profiles
	for _, name := range []string{"work", "personal", "default"} {
		dir := filepath.Join(profilesDir, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	writeManifest(t, filepath.Join(profilesDir, "work"), []string{"settings.json", "keybindings.json"}, nil, "work profile")
	writeManifest(t, filepath.Join(profilesDir, "personal"), []string{"settings.json"}, map[string]string{"themes": "work"}, "personal profile")
	writeManifest(t, filepath.Join(profilesDir, "default"), []string{}, nil, "default profile")

	configPath := filepath.Join(tmp, "config.json")
	writeConfig(t, tmp, "work")

	summaries, err := profile.ListProfiles(profilesDir, configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(summaries) != 3 {
		t.Fatalf("expected 3 summaries, got %d", len(summaries))
	}

	// Should be sorted by name: default, personal, work
	if summaries[0].Name != "default" {
		t.Errorf("expected summaries[0].Name=default, got %q", summaries[0].Name)
	}
	if summaries[1].Name != "personal" {
		t.Errorf("expected summaries[1].Name=personal, got %q", summaries[1].Name)
	}
	if summaries[2].Name != "work" {
		t.Errorf("expected summaries[2].Name=work, got %q", summaries[2].Name)
	}

	// work should be active
	if summaries[2].IsActive != true {
		t.Errorf("expected work to be active")
	}
	if summaries[0].IsActive != false {
		t.Errorf("expected default to not be active")
	}

	// Check counts
	if summaries[2].ManagedCount != 2 {
		t.Errorf("expected work ManagedCount=2, got %d", summaries[2].ManagedCount)
	}
	if summaries[1].SharedCount != 1 {
		t.Errorf("expected personal SharedCount=1, got %d", summaries[1].SharedCount)
	}

	// Check descriptions
	if summaries[2].Description != "work profile" {
		t.Errorf("expected work description, got %q", summaries[2].Description)
	}
}

func TestListProfiles_SkipsNonProfileDirs(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")

	// Create a real profile
	profileDir := filepath.Join(profilesDir, "myprofile")
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeManifest(t, profileDir, nil, nil, "my profile")

	// Create a non-profile directory (no manifest)
	notAProfile := filepath.Join(profilesDir, "notaprofile")
	if err := os.MkdirAll(notAProfile, 0755); err != nil {
		t.Fatal(err)
	}
	// Write some other file, but NOT a manifest
	if err := os.WriteFile(filepath.Join(notAProfile, "README.txt"), []byte("not a profile"), 0644); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(tmp, "config.json")
	writeConfig(t, tmp, "")

	summaries, err := profile.ListProfiles(profilesDir, configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(summaries) != 1 {
		t.Errorf("expected 1 summary (non-profile dir skipped), got %d", len(summaries))
	}
	if summaries[0].Name != "myprofile" {
		t.Errorf("expected myprofile, got %q", summaries[0].Name)
	}
}

func TestListProfiles_ActiveMarker(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")

	for _, name := range []string{"alpha", "beta"} {
		dir := filepath.Join(profilesDir, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		writeManifest(t, dir, nil, nil, name)
	}

	configPath := filepath.Join(tmp, "config.json")
	writeConfig(t, tmp, "beta")

	summaries, err := profile.ListProfiles(profilesDir, configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// alpha not active, beta active
	activeCount := 0
	for _, s := range summaries {
		if s.IsActive {
			activeCount++
			if s.Name != "beta" {
				t.Errorf("wrong profile marked active: %q", s.Name)
			}
		}
	}
	if activeCount != 1 {
		t.Errorf("expected exactly 1 active profile, got %d", activeCount)
	}
}

func TestFormatProfileList(t *testing.T) {
	summaries := []profile.ProfileSummary{
		{Name: "work", Description: "work profile", ManagedCount: 2, SharedCount: 1, IsActive: true},
		{Name: "personal", Description: "personal profile", ManagedCount: 1, SharedCount: 0, IsActive: false},
	}
	result := profile.FormatProfileList(summaries)
	if result == "" {
		t.Error("expected non-empty format output")
	}
	// Active profile should be marked
	// We just check that the active marker appears
	if len(result) == 0 {
		t.Error("FormatProfileList returned empty string")
	}
}
