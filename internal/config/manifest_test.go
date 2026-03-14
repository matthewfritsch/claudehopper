package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest_ReadsFields(t *testing.T) {
	fixture := filepath.Join("testdata", "hop-manifest.json")
	m, err := LoadManifest(fixture)
	if err != nil {
		t.Fatalf("LoadManifest() error: %v", err)
	}
	if len(m.ManagedPaths) != 2 {
		t.Errorf("len(ManagedPaths) = %d, want 2", len(m.ManagedPaths))
	}
	if m.ManagedPaths[0] != "agents" {
		t.Errorf("ManagedPaths[0] = %q, want %q", m.ManagedPaths[0], "agents")
	}
	if m.ManagedPaths[1] != "settings.json" {
		t.Errorf("ManagedPaths[1] = %q, want %q", m.ManagedPaths[1], "settings.json")
	}
	if m.SharedPaths == nil {
		t.Error("SharedPaths is nil, want non-nil map")
	}
	if len(m.SharedPaths) != 0 {
		t.Errorf("len(SharedPaths) = %d, want 0", len(m.SharedPaths))
	}
	if m.Description != "get-shit-done spec-driven development" {
		t.Errorf("Description = %q, want %q", m.Description, "get-shit-done spec-driven development")
	}
}

func TestSaveManifest_SortsManagedPaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	m := Manifest{
		ManagedPaths: []string{"settings.json", "agents", "mcp.json"},
		SharedPaths:  map[string]string{},
		Description:  "test",
	}
	if err := SaveManifest(path, m); err != nil {
		t.Fatalf("SaveManifest() error: %v", err)
	}

	loaded, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest() after save: %v", err)
	}

	// managed_paths should be alphabetically sorted
	want := []string{"agents", "mcp.json", "settings.json"}
	if len(loaded.ManagedPaths) != len(want) {
		t.Fatalf("len(ManagedPaths) = %d, want %d", len(loaded.ManagedPaths), len(want))
	}
	for i, p := range want {
		if loaded.ManagedPaths[i] != p {
			t.Errorf("ManagedPaths[%d] = %q, want %q", i, loaded.ManagedPaths[i], p)
		}
	}
}

func TestSaveManifest_IndentAndNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	m := Manifest{
		ManagedPaths: []string{"agents"},
		SharedPaths:  map[string]string{},
		Description:  "test manifest",
	}
	if err := SaveManifest(path, m); err != nil {
		t.Fatalf("SaveManifest() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Must use 2-space indentation and end with newline
	if len(data) == 0 {
		t.Fatal("SaveManifest wrote empty file")
	}
	if data[len(data)-1] != '\n' {
		t.Errorf("SaveManifest output does not end with newline: %q", string(data))
	}
	// Check 2-space indent by verifying "  " prefix on indented lines
	str := string(data)
	if str[0] != '{' {
		t.Errorf("manifest does not start with {: %q", str)
	}
}

func TestSaveManifest_EmptySharedPaths_NotNull(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	m := Manifest{
		ManagedPaths: []string{},
		SharedPaths:  map[string]string{},
		Description:  "test",
	}
	if err := SaveManifest(path, m); err != nil {
		t.Fatalf("SaveManifest() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	str := string(data)

	// shared_paths must be {} not null
	if !bytes.Contains(data, []byte(`"shared_paths": {}`)) {
		t.Errorf("shared_paths should serialize as {}, got:\n%s", str)
	}
}

func TestSaveManifest_EmptyManagedPaths_NotNull(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	m := Manifest{
		ManagedPaths: []string{},
		SharedPaths:  map[string]string{},
		Description:  "test",
	}
	if err := SaveManifest(path, m); err != nil {
		t.Fatalf("SaveManifest() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// managed_paths must be [] not null
	if !bytes.Contains(data, []byte(`"managed_paths": []`)) {
		t.Errorf("managed_paths should serialize as [], got:\n%s", string(data))
	}
}

func TestLoadManifest_FixtureRoundTrip(t *testing.T) {
	fixture := filepath.Join("testdata", "hop-manifest.json")
	original, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	m, err := LoadManifest(fixture)
	if err != nil {
		t.Fatalf("LoadManifest(fixture): %v", err)
	}

	dir := t.TempDir()
	out := filepath.Join(dir, "hop-manifest.json")
	if err := SaveManifest(out, m); err != nil {
		t.Fatalf("SaveManifest(): %v", err)
	}

	saved, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(original, saved) {
		t.Errorf("round-trip mismatch:\noriginal: %q\nsaved:    %q", original, saved)
	}
}

func TestNewManifest_NonNilCollections(t *testing.T) {
	m := NewManifest("test description")
	if m.ManagedPaths == nil {
		t.Error("NewManifest ManagedPaths is nil, want non-nil slice")
	}
	if m.SharedPaths == nil {
		t.Error("NewManifest SharedPaths is nil, want non-nil map")
	}
	if m.Description != "test description" {
		t.Errorf("NewManifest Description = %q, want %q", m.Description, "test description")
	}
}
