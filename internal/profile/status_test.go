package profile_test

import (
	"os"
	"path/filepath"
	"strings"
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

func TestFormatProfileStatus_Verbose(t *testing.T) {
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
	result := profile.FormatProfileStatus(info, true)
	if result == "" {
		t.Fatal("FormatProfileStatus returned empty string")
	}
	// Verbose mode should list every path
	for _, name := range []string{"settings.json", "themes", "keybindings.json", "snippets"} {
		if !strings.Contains(result, name) {
			t.Errorf("verbose output missing path %q", name)
		}
	}
}

func TestFormatProfileStatus_Compact_AllHealthy(t *testing.T) {
	info := profile.ProfileStatusInfo{
		Name: "work",
		Paths: []profile.PathHealth{
			{Name: "CLAUDE.md", Status: "linked"},
			{Name: "commands", Status: "linked"},
			{Name: "settings.json", Status: "shared", Detail: "(shared)"},
		},
	}
	result := profile.FormatProfileStatus(info, false)
	// Should show summary, not individual paths
	if !strings.Contains(result, "3 paths linked") {
		t.Errorf("expected summary line, got:\n%s", result)
	}
	if !strings.Contains(result, "1 shared") {
		t.Errorf("expected shared count, got:\n%s", result)
	}
	// Individual path names should NOT appear
	if strings.Contains(result, "CLAUDE.md") {
		t.Errorf("compact healthy output should not list individual paths, got:\n%s", result)
	}
}

func TestFormatProfileStatus_Compact_WithUnhealthy(t *testing.T) {
	info := profile.ProfileStatusInfo{
		Name: "work",
		Paths: []profile.PathHealth{
			{Name: "CLAUDE.md", Status: "linked"},
			{Name: "settings.json", Status: "shared", Detail: "(shared)"},
			{Name: "keybindings.json", Status: "not-linked"},
			{Name: "snippets", Status: "conflict"},
		},
	}
	result := profile.FormatProfileStatus(info, false)
	// Should show healthy count summary
	if !strings.Contains(result, "2/4 paths healthy") {
		t.Errorf("expected healthy/total summary, got:\n%s", result)
	}
	// Unhealthy paths should be listed
	if !strings.Contains(result, "keybindings.json") {
		t.Errorf("expected unhealthy path listed, got:\n%s", result)
	}
	if !strings.Contains(result, "snippets") {
		t.Errorf("expected unhealthy path listed, got:\n%s", result)
	}
	// Healthy paths should NOT be listed
	if strings.Contains(result, "CLAUDE.md") {
		t.Errorf("healthy paths should not be listed in compact mode, got:\n%s", result)
	}
}
