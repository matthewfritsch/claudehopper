package profile_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/matthewfritsch/claudehopper/internal/profile"
)

// setupProfile creates a profile directory with a manifest in profilesDir.
func setupProfile(t *testing.T, profilesDir, name string, shared map[string]string, createdFrom string) {
	t.Helper()
	dir := filepath.Join(profilesDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("setup profile %q: %v", name, err)
	}
	if shared == nil {
		shared = map[string]string{}
	}
	m := struct {
		ManagedPaths []string          `json:"managed_paths"`
		SharedPaths  map[string]string `json:"shared_paths"`
		Description  string            `json:"description"`
		CreatedFrom  string            `json:"created_from,omitempty"`
	}{
		ManagedPaths: []string{},
		SharedPaths:  shared,
		Description:  name,
		CreatedFrom:  createdFrom,
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest for %q: %v", name, err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".hop-manifest.json"), data, 0644); err != nil {
		t.Fatalf("write manifest for %q: %v", name, err)
	}
}

func TestDeleteProfile_RemovesDirectory(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")

	setupProfile(t, profilesDir, "work", nil, "")
	setupProfile(t, profilesDir, "personal", nil, "")

	err := profile.DeleteProfile(profilesDir, "personal", "work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// personal dir should be gone
	if _, err := os.Stat(filepath.Join(profilesDir, "personal")); !os.IsNotExist(err) {
		t.Error("expected personal profile directory to be removed")
	}
	// work dir should still exist
	if _, err := os.Stat(filepath.Join(profilesDir, "work")); err != nil {
		t.Errorf("work profile should still exist: %v", err)
	}
}

func TestDeleteProfile_RefusesActiveProfile(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")

	setupProfile(t, profilesDir, "work", nil, "")

	err := profile.DeleteProfile(profilesDir, "work", "work")
	if err == nil {
		t.Fatal("expected error when deleting active profile")
	}
	// Profile dir should still exist
	if _, err := os.Stat(filepath.Join(profilesDir, "work")); err != nil {
		t.Errorf("active profile directory should not have been removed: %v", err)
	}
}

func TestDeleteProfile_WithDependents_ReturnsDependentError(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")

	// work is the target; personal shares from work
	setupProfile(t, profilesDir, "work", nil, "")
	setupProfile(t, profilesDir, "personal", map[string]string{"themes": "work"}, "")

	err := profile.DeleteProfile(profilesDir, "work", "other")
	if err == nil {
		t.Fatal("expected DependentError when dependents exist")
	}

	var depErr *profile.DependentError
	if !errors.As(err, &depErr) {
		t.Fatalf("expected *profile.DependentError, got %T: %v", err, err)
	}
	if depErr.Profile != "work" {
		t.Errorf("expected Profile=work, got %q", depErr.Profile)
	}
	if len(depErr.Dependents) != 1 || depErr.Dependents[0] != "personal" {
		t.Errorf("expected Dependents=[personal], got %v", depErr.Dependents)
	}
}

func TestFindDependents_NoDependents(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")

	setupProfile(t, profilesDir, "work", nil, "")
	setupProfile(t, profilesDir, "personal", nil, "")

	deps, err := profile.FindDependents(profilesDir, "work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("expected no dependents, got %v", deps)
	}
}

func TestFindDependents_SharedPathsReference(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")

	setupProfile(t, profilesDir, "work", nil, "")
	// personal shares "themes" from work
	setupProfile(t, profilesDir, "personal", map[string]string{"themes": "work"}, "")
	// freelance shares multiple from work
	setupProfile(t, profilesDir, "freelance", map[string]string{"settings.json": "work", "snippets": "other"}, "")

	deps, err := profile.FindDependents(profilesDir, "work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 2 {
		t.Errorf("expected 2 dependents (personal, freelance), got %v", deps)
	}
	// Should be sorted
	if deps[0] != "freelance" || deps[1] != "personal" {
		t.Errorf("expected sorted [freelance personal], got %v", deps)
	}
}

func TestFindDependents_CreatedFromReference(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")

	setupProfile(t, profilesDir, "base", nil, "")
	// derived was created from base
	setupProfile(t, profilesDir, "derived", nil, "base")

	deps, err := profile.FindDependents(profilesDir, "base")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 1 || deps[0] != "derived" {
		t.Errorf("expected [derived], got %v", deps)
	}
}

func TestDependentError_Message(t *testing.T) {
	err := &profile.DependentError{
		Profile:    "work",
		Dependents: []string{"personal", "freelance"},
	}
	msg := err.Error()
	if msg == "" {
		t.Error("DependentError.Error() returned empty string")
	}
}
