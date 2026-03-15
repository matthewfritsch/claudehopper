package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateBlank_CreatesProfileDir(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	sharedDir := filepath.Join(tmp, "shared")
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := CreateBlank(profilesDir, sharedDir, "test", "a test profile"); err != nil {
		t.Fatalf("CreateBlank() error: %v", err)
	}

	profileDir := filepath.Join(profilesDir, "test")
	fi, err := os.Stat(profileDir)
	if err != nil {
		t.Fatalf("profile dir not created: %v", err)
	}
	if !fi.IsDir() {
		t.Error("expected directory")
	}

	// settings.json should exist
	settingsPath := filepath.Join(profileDir, "settings.json")
	if _, err := os.Stat(settingsPath); err != nil {
		t.Errorf("settings.json not created: %v", err)
	}

	// manifest should exist
	manifestPath := filepath.Join(profileDir, ".hop-manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Errorf(".hop-manifest.json not created: %v", err)
	}
}

func TestCreateBlank_ManifestContents(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	sharedDir := filepath.Join(tmp, "shared")
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := CreateBlank(profilesDir, sharedDir, "mytest", "desc"); err != nil {
		t.Fatalf("CreateBlank() error: %v", err)
	}

	manifestPath := filepath.Join(profilesDir, "mytest", ".hop-manifest.json")
	m, err := loadManifestFromPath(manifestPath)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}

	if m.Description != "desc" {
		t.Errorf("Description = %q, want %q", m.Description, "desc")
	}
	if len(m.ManagedPaths) != 1 || m.ManagedPaths[0] != "settings.json" {
		t.Errorf("ManagedPaths = %v, want [settings.json]", m.ManagedPaths)
	}
}

func TestCreateBlank_ErrorOnExistingProfile(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	sharedDir := filepath.Join(tmp, "shared")
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := CreateBlank(profilesDir, sharedDir, "existing", "first"); err != nil {
		t.Fatalf("first CreateBlank() error: %v", err)
	}
	if err := CreateBlank(profilesDir, sharedDir, "existing", "second"); err == nil {
		t.Error("expected error on duplicate profile name, got nil")
	}
}

func TestCreateBlank_NormalizesName(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	sharedDir := filepath.Join(tmp, "shared")
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := CreateBlank(profilesDir, sharedDir, "MyProfile", "mixed case name"); err != nil {
		t.Fatalf("CreateBlank() error: %v", err)
	}

	// Should be stored as lowercase
	profileDir := filepath.Join(profilesDir, "myprofile")
	if _, err := os.Stat(profileDir); err != nil {
		t.Errorf("expected profile dir at lowercase path: %v", err)
	}
}

func TestCreateFromCurrent_CapturesFiles(t *testing.T) {
	tmp := t.TempDir()
	claudeDir := filepath.Join(tmp, "claude")
	profilesDir := filepath.Join(tmp, "profiles")
	sharedDir := filepath.Join(tmp, "shared")

	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "keybindings.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CreateFromCurrent(profilesDir, claudeDir, sharedDir, "captured", "from current"); err != nil {
		t.Fatalf("CreateFromCurrent() error: %v", err)
	}

	profileDir := filepath.Join(profilesDir, "captured")
	if _, err := os.Stat(filepath.Join(profileDir, "settings.json")); err != nil {
		t.Errorf("settings.json not captured: %v", err)
	}
	if _, err := os.Stat(filepath.Join(profileDir, "keybindings.json")); err != nil {
		t.Errorf("keybindings.json not captured: %v", err)
	}
}

func TestCreateFromCurrent_ExcludesProtectedPaths(t *testing.T) {
	tmp := t.TempDir()
	claudeDir := filepath.Join(tmp, "claude")
	profilesDir := filepath.Join(tmp, "profiles")
	sharedDir := filepath.Join(tmp, "shared")

	for _, d := range []string{claudeDir, profilesDir, sharedDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Protected file
	if err := os.WriteFile(filepath.Join(claudeDir, ".credentials.json"), []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	// .hop- prefix file
	if err := os.WriteFile(filepath.Join(claudeDir, ".hop-manifest.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	// .ccswap prefix file
	if err := os.WriteFile(filepath.Join(claudeDir, ".ccswap-temp"), []byte("tmp"), 0644); err != nil {
		t.Fatal(err)
	}
	// Normal file that should be captured
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CreateFromCurrent(profilesDir, claudeDir, sharedDir, "filtered", "test"); err != nil {
		t.Fatalf("CreateFromCurrent() error: %v", err)
	}

	profileDir := filepath.Join(profilesDir, "filtered")
	// Protected file should NOT be in profile
	if _, err := os.Stat(filepath.Join(profileDir, ".credentials.json")); err == nil {
		t.Error(".credentials.json should not be captured")
	}
	// .hop- files from claudeDir should NOT be added to managed_paths.
	// Note: the profile dir will have a .hop-manifest.json written by CreateFromCurrent
	// itself, but its content should be the generated manifest, not the source one.
	// Check that managed_paths does not include .hop-manifest.json.
	manifestPath := filepath.Join(profileDir, ".hop-manifest.json")
	m, err := loadManifestFromPath(manifestPath)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	for _, p := range m.ManagedPaths {
		if p == ".hop-manifest.json" {
			t.Error(".hop-manifest.json from claudeDir should not be in managed_paths")
		}
	}
	// .ccswap files should NOT be in profile
	if _, err := os.Stat(filepath.Join(profileDir, ".ccswap-temp")); err == nil {
		t.Error(".ccswap-temp should not be captured")
	}
	// Normal file should be captured
	if _, err := os.Stat(filepath.Join(profileDir, "settings.json")); err != nil {
		t.Errorf("settings.json should be captured: %v", err)
	}
}

func TestCreateFromCurrent_PreservesSymlinks(t *testing.T) {
	tmp := t.TempDir()
	claudeDir := filepath.Join(tmp, "claude")
	profilesDir := filepath.Join(tmp, "profiles")
	sharedDir := filepath.Join(tmp, "shared")

	for _, d := range []string{claudeDir, profilesDir, sharedDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create a target file and a symlink in claudeDir
	targetFile := filepath.Join(tmp, "real-settings.json")
	if err := os.WriteFile(targetFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetFile, filepath.Join(claudeDir, "settings.json")); err != nil {
		t.Fatal(err)
	}

	if err := CreateFromCurrent(profilesDir, claudeDir, sharedDir, "symtest", "symlink test"); err != nil {
		t.Fatalf("CreateFromCurrent() error: %v", err)
	}

	linkPath := filepath.Join(profilesDir, "symtest", "settings.json")
	fi, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("settings.json not present in profile: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlink to be preserved as symlink")
	}
}

func TestCreateFromCurrent_RecordsSymlinksInSharedPaths(t *testing.T) {
	tmp := t.TempDir()
	claudeDir := filepath.Join(tmp, "claude")
	profilesDir := filepath.Join(tmp, "profiles")
	sharedDir := filepath.Join(tmp, "shared")

	for _, d := range []string{claudeDir, profilesDir, sharedDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create a symlink in claudeDir
	targetFile := filepath.Join(tmp, "real-settings.json")
	if err := os.WriteFile(targetFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetFile, filepath.Join(claudeDir, "settings.json")); err != nil {
		t.Fatal(err)
	}

	if err := CreateFromCurrent(profilesDir, claudeDir, sharedDir, "symsrc", "test"); err != nil {
		t.Fatalf("CreateFromCurrent() error: %v", err)
	}

	manifestPath := filepath.Join(profilesDir, "symsrc", ".hop-manifest.json")
	m, err := loadManifestFromPath(manifestPath)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}

	if _, ok := m.SharedPaths["settings.json"]; !ok {
		t.Errorf("expected settings.json in SharedPaths, got: %v", m.SharedPaths)
	}
}

func TestCreateFromProfile_CopiesFiles(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	sharedDir := filepath.Join(tmp, "shared")

	for _, d := range []string{profilesDir, sharedDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create source profile with CreateBlank
	if err := CreateBlank(profilesDir, sharedDir, "source", "source profile"); err != nil {
		t.Fatalf("CreateBlank source: %v", err)
	}
	// Add extra file
	srcFile := filepath.Join(profilesDir, "source", "keybindings.json")
	if err := os.WriteFile(srcFile, []byte(`{"key":"val"}`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CreateFromProfile(profilesDir, sharedDir, "source", "clone", "cloned"); err != nil {
		t.Fatalf("CreateFromProfile() error: %v", err)
	}

	cloneDir := filepath.Join(profilesDir, "clone")
	if _, err := os.Stat(filepath.Join(cloneDir, "keybindings.json")); err != nil {
		t.Errorf("keybindings.json not cloned: %v", err)
	}
}

func TestCreateFromProfile_SetsCreatedFrom(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	sharedDir := filepath.Join(tmp, "shared")

	for _, d := range []string{profilesDir, sharedDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	if err := CreateBlank(profilesDir, sharedDir, "original", "orig"); err != nil {
		t.Fatalf("CreateBlank: %v", err)
	}

	if err := CreateFromProfile(profilesDir, sharedDir, "original", "derived", "derived profile"); err != nil {
		t.Fatalf("CreateFromProfile() error: %v", err)
	}

	manifestPath := filepath.Join(profilesDir, "derived", ".hop-manifest.json")
	m, err := loadManifestFromPath(manifestPath)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}

	if m.CreatedFrom != "original" {
		t.Errorf("CreatedFrom = %q, want %q", m.CreatedFrom, "original")
	}
	if m.Description != "derived profile" {
		t.Errorf("Description = %q, want %q", m.Description, "derived profile")
	}
}

func TestCreateFromProfile_PreservesSymlinks(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	sharedDir := filepath.Join(tmp, "shared")

	for _, d := range []string{profilesDir, sharedDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	sourceDir := filepath.Join(profilesDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a real file and a symlink in source profile
	realFile := filepath.Join(tmp, "real.json")
	if err := os.WriteFile(realFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realFile, filepath.Join(sourceDir, "settings.json")); err != nil {
		t.Fatal(err)
	}

	// Write a manifest for the source
	testWriteManifest(t, sourceDir, nil, map[string]string{"settings.json": "(shared)"}, "source")

	if err := CreateFromProfile(profilesDir, sharedDir, "source", "clonedsym", "test"); err != nil {
		t.Fatalf("CreateFromProfile() error: %v", err)
	}

	cloneDir := filepath.Join(profilesDir, "clonedsym")
	linkPath := filepath.Join(cloneDir, "settings.json")
	fi, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("settings.json not in clone: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlink to be preserved as symlink")
	}
}
