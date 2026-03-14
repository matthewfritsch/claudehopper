package profile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matthewfritsch/claudehopper/internal/config"
	"github.com/matthewfritsch/claudehopper/internal/profile"
)

func TestGetProfileStatus_Linked(t *testing.T) {
	tmp := t.TempDir()
	profileDir := filepath.Join(tmp, "profiles", "work")
	claudeDir := filepath.Join(tmp, "claude")
	sharedDir := filepath.Join(tmp, "shared")

	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file in profileDir
	if err := os.WriteFile(filepath.Join(profileDir, "settings.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink in claudeDir pointing to the profile file
	if err := os.Symlink(
		filepath.Join(profileDir, "settings.json"),
		filepath.Join(claudeDir, "settings.json"),
	); err != nil {
		t.Fatal(err)
	}

	m := config.Manifest{
		ManagedPaths: []string{"settings.json"},
		SharedPaths:  map[string]string{},
		Description:  "work",
	}

	info := profile.GetProfileStatus(profileDir, claudeDir, sharedDir, m)
	if len(info.Paths) != 1 {
		t.Fatalf("expected 1 path health, got %d", len(info.Paths))
	}
	if info.Paths[0].Status != "linked" {
		t.Errorf("expected status=linked, got %q", info.Paths[0].Status)
	}
}

func TestGetProfileStatus_NotLinked(t *testing.T) {
	tmp := t.TempDir()
	profileDir := filepath.Join(tmp, "profiles", "work")
	claudeDir := filepath.Join(tmp, "claude")
	sharedDir := filepath.Join(tmp, "shared")

	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// File exists in profile but NOT linked in claudeDir
	if err := os.WriteFile(filepath.Join(profileDir, "settings.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	m := config.Manifest{
		ManagedPaths: []string{"settings.json"},
		SharedPaths:  map[string]string{},
		Description:  "work",
	}

	info := profile.GetProfileStatus(profileDir, claudeDir, sharedDir, m)
	if len(info.Paths) != 1 {
		t.Fatalf("expected 1 path health, got %d", len(info.Paths))
	}
	if info.Paths[0].Status != "not-linked" {
		t.Errorf("expected status=not-linked, got %q", info.Paths[0].Status)
	}
}

func TestGetProfileStatus_Conflict(t *testing.T) {
	tmp := t.TempDir()
	profileDir := filepath.Join(tmp, "profiles", "work")
	claudeDir := filepath.Join(tmp, "claude")
	sharedDir := filepath.Join(tmp, "shared")

	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// A REAL file (not symlink) exists at the link location
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("real file"), 0644); err != nil {
		t.Fatal(err)
	}

	m := config.Manifest{
		ManagedPaths: []string{"settings.json"},
		SharedPaths:  map[string]string{},
		Description:  "work",
	}

	info := profile.GetProfileStatus(profileDir, claudeDir, sharedDir, m)
	if len(info.Paths) != 1 {
		t.Fatalf("expected 1 path health, got %d", len(info.Paths))
	}
	if info.Paths[0].Status != "conflict" {
		t.Errorf("expected status=conflict, got %q", info.Paths[0].Status)
	}
}

func TestGetProfileStatus_Broken(t *testing.T) {
	tmp := t.TempDir()
	profileDir := filepath.Join(tmp, "profiles", "work")
	claudeDir := filepath.Join(tmp, "claude")
	sharedDir := filepath.Join(tmp, "shared")

	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a symlink pointing to a non-existent target
	if err := os.Symlink(
		filepath.Join(tmp, "nonexistent", "settings.json"),
		filepath.Join(claudeDir, "settings.json"),
	); err != nil {
		t.Fatal(err)
	}

	m := config.Manifest{
		ManagedPaths: []string{"settings.json"},
		SharedPaths:  map[string]string{},
		Description:  "work",
	}

	info := profile.GetProfileStatus(profileDir, claudeDir, sharedDir, m)
	if len(info.Paths) != 1 {
		t.Fatalf("expected 1 path health, got %d", len(info.Paths))
	}
	if info.Paths[0].Status != "broken" {
		t.Errorf("expected status=broken, got %q", info.Paths[0].Status)
	}
}

func TestGetProfileStatus_Shared(t *testing.T) {
	tmp := t.TempDir()
	profileDir := filepath.Join(tmp, "profiles", "work")
	sharedDir := filepath.Join(tmp, "shared")
	claudeDir := filepath.Join(tmp, "claude")

	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file in sharedDir
	if err := os.WriteFile(filepath.Join(sharedDir, "themes"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink in claudeDir pointing to shared
	if err := os.Symlink(
		filepath.Join(sharedDir, "themes"),
		filepath.Join(claudeDir, "themes"),
	); err != nil {
		t.Fatal(err)
	}

	m := config.Manifest{
		ManagedPaths: []string{},
		SharedPaths:  map[string]string{"themes": "other-profile"},
		Description:  "work",
	}

	info := profile.GetProfileStatus(profileDir, claudeDir, sharedDir, m)
	if len(info.Paths) != 1 {
		t.Fatalf("expected 1 path health (from shared_paths), got %d", len(info.Paths))
	}
	if info.Paths[0].Status != "shared" {
		t.Errorf("expected status=shared, got %q", info.Paths[0].Status)
	}
}

func TestFormatProfileStatus(t *testing.T) {
	info := profile.ProfileStatusInfo{
		Name:        "work",
		Description: "work profile",
		Paths: []profile.PathHealth{
			{Name: "settings.json", Status: "linked", Detail: ""},
			{Name: "themes", Status: "shared", Detail: "other-profile"},
			{Name: "keybindings.json", Status: "not-linked", Detail: ""},
			{Name: "snippets", Status: "conflict", Detail: ""},
		},
	}
	result := profile.FormatProfileStatus(info)
	if result == "" {
		t.Error("FormatProfileStatus returned empty string")
	}
}
