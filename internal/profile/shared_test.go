package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureSharedDefaults_CreatesDir(t *testing.T) {
	// Override the config dir via environment for isolation
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	sharedPath, err := EnsureSharedDefaults()
	if err != nil {
		t.Fatalf("EnsureSharedDefaults() error: %v", err)
	}

	fi, err := os.Stat(sharedPath)
	if err != nil {
		t.Fatalf("shared dir not created: %v", err)
	}
	if !fi.IsDir() {
		t.Errorf("expected shared dir to be a directory")
	}
}

func TestEnsureSharedDefaults_IdempotentOnExisting(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// Call twice — should not error
	if _, err := EnsureSharedDefaults(); err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if _, err := EnsureSharedDefaults(); err != nil {
		t.Fatalf("second call error: %v", err)
	}
}

func TestLinkDefaultsIntoProfile_SymlinksExistingSharedFiles(t *testing.T) {
	tmp := t.TempDir()
	sharedDir := filepath.Join(tmp, "shared")
	profileDir := filepath.Join(tmp, "profiles", "work")

	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Place settings.json in shared dir
	sharedFile := filepath.Join(sharedDir, "settings.json")
	if err := os.WriteFile(sharedFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := LinkDefaultsIntoProfile(profileDir, sharedDir, "")
	if err != nil {
		t.Fatalf("LinkDefaultsIntoProfile() error: %v", err)
	}

	// Profile should have a symlink to settings.json
	linkPath := filepath.Join(profileDir, "settings.json")
	fi, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("symlink not created at %s: %v", linkPath, err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlink at %s", linkPath)
	}

	// Manifest should record settings.json in shared_paths
	if _, ok := m.SharedPaths["settings.json"]; !ok {
		t.Errorf("expected settings.json in SharedPaths, got: %v", m.SharedPaths)
	}
}

func TestLinkDefaultsIntoProfile_SeedsFromSource(t *testing.T) {
	tmp := t.TempDir()
	sharedDir := filepath.Join(tmp, "shared")
	profileDir := filepath.Join(tmp, "profiles", "work")
	fromSource := filepath.Join(tmp, "source")

	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fromSource, 0755); err != nil {
		t.Fatal(err)
	}

	// Place settings.json in source but NOT in shared
	srcFile := filepath.Join(fromSource, "settings.json")
	if err := os.WriteFile(srcFile, []byte(`{"theme":"dark"}`), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := LinkDefaultsIntoProfile(profileDir, sharedDir, fromSource)
	if err != nil {
		t.Fatalf("LinkDefaultsIntoProfile() error: %v", err)
	}

	// shared dir should now have settings.json
	sharedFile := filepath.Join(sharedDir, "settings.json")
	if _, err := os.Stat(sharedFile); err != nil {
		t.Fatalf("settings.json not copied to shared dir: %v", err)
	}

	// Profile should have symlink
	linkPath := filepath.Join(profileDir, "settings.json")
	fi, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("symlink not created: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlink")
	}

	// Manifest should have shared_paths entry
	if _, ok := m.SharedPaths["settings.json"]; !ok {
		t.Errorf("expected settings.json in SharedPaths")
	}
}

func TestLinkDefaultsIntoProfile_SkipsMissingFiles(t *testing.T) {
	tmp := t.TempDir()
	sharedDir := filepath.Join(tmp, "shared")
	profileDir := filepath.Join(tmp, "profiles", "blank")

	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Neither shared nor source has any DEFAULT_LINKED files
	m, err := LinkDefaultsIntoProfile(profileDir, sharedDir, "")
	if err != nil {
		t.Fatalf("LinkDefaultsIntoProfile() error: %v", err)
	}

	// No symlinks should be created
	entries, _ := os.ReadDir(profileDir)
	if len(entries) != 0 {
		t.Errorf("expected empty profileDir, got: %v", entries)
	}

	// No shared_paths entries
	if len(m.SharedPaths) != 0 {
		t.Errorf("expected empty SharedPaths, got: %v", m.SharedPaths)
	}
}
