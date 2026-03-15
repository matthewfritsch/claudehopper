package profile_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/matthewfritsch/claudehopper/internal/config"
	"github.com/matthewfritsch/claudehopper/internal/profile"
)

// makeConfigFile writes a config.json with the given active profile to configPath.
func makeConfigFile(t *testing.T, configPath, active string) {
	t.Helper()
	cfg := config.Config{Active: active}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// TestUnmanageActive_MaterializesSymlinks verifies that symlinks in claudeDir
// are replaced with real file copies.
func TestUnmanageActive_MaterializesSymlinks(t *testing.T) {
	// Create a "profiles" dir with a real file to link to
	realDir := t.TempDir()
	realFile := filepath.Join(realDir, "myfile.txt")
	if err := os.WriteFile(realFile, []byte("hello from real"), 0644); err != nil {
		t.Fatalf("create real file: %v", err)
	}

	// claudeDir contains a symlink pointing at realFile
	claudeDir := t.TempDir()
	linkPath := filepath.Join(claudeDir, "myfile.txt")
	if err := os.Symlink(realFile, linkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	configPath := filepath.Join(t.TempDir(), "config.json")
	makeConfigFile(t, configPath, "myprofile")

	materialized, err := profile.UnmanageActive(claudeDir, configPath, false)
	if err != nil {
		t.Fatalf("UnmanageActive: %v", err)
	}
	if len(materialized) != 1 {
		t.Fatalf("len(materialized) = %d, want 1", len(materialized))
	}

	// linkPath should now be a real file, not a symlink
	fi, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("Lstat: %v", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Errorf("expected real file, got symlink at %s", linkPath)
	}

	// Content should be preserved
	data, err := os.ReadFile(linkPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello from real" {
		t.Errorf("file content = %q, want %q", string(data), "hello from real")
	}
}

// TestUnmanageActive_SkipsRealFiles verifies that regular files in claudeDir are not touched.
func TestUnmanageActive_SkipsRealFiles(t *testing.T) {
	claudeDir := t.TempDir()
	realFile := filepath.Join(claudeDir, "realfile.txt")
	content := []byte("real file content")
	if err := os.WriteFile(realFile, content, 0644); err != nil {
		t.Fatalf("create real file: %v", err)
	}

	configPath := filepath.Join(t.TempDir(), "config.json")
	makeConfigFile(t, configPath, "myprofile")

	materialized, err := profile.UnmanageActive(claudeDir, configPath, false)
	if err != nil {
		t.Fatalf("UnmanageActive: %v", err)
	}
	// No symlinks to materialize
	if len(materialized) != 0 {
		t.Errorf("len(materialized) = %d, want 0", len(materialized))
	}

	// File should still be there and unchanged
	data, err := os.ReadFile(realFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("file content changed")
	}
}

// TestUnmanageActive_SkipsProtected verifies that protected paths (e.g., .credentials.json)
// are not materialized even if they are symlinks.
func TestUnmanageActive_SkipsProtected(t *testing.T) {
	realDir := t.TempDir()
	realCreds := filepath.Join(realDir, ".credentials.json")
	if err := os.WriteFile(realCreds, []byte(`{"key":"secret"}`), 0600); err != nil {
		t.Fatalf("create creds: %v", err)
	}

	claudeDir := t.TempDir()
	// Create a symlink to credentials — should be skipped
	if err := os.Symlink(realCreds, filepath.Join(claudeDir, ".credentials.json")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	// Also add a non-protected symlink that should be materialized
	realFile := filepath.Join(realDir, "agents")
	if err := os.WriteFile(realFile, []byte("agent data"), 0644); err != nil {
		t.Fatalf("create file: %v", err)
	}
	if err := os.Symlink(realFile, filepath.Join(claudeDir, "agents")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	configPath := filepath.Join(t.TempDir(), "config.json")
	makeConfigFile(t, configPath, "myprofile")

	materialized, err := profile.UnmanageActive(claudeDir, configPath, false)
	if err != nil {
		t.Fatalf("UnmanageActive: %v", err)
	}
	// Only "agents" should be materialized, not ".credentials.json"
	if len(materialized) != 1 {
		t.Errorf("len(materialized) = %d, want 1 (got %v)", len(materialized), materialized)
	}

	// .credentials.json should still be a symlink
	fi, err := os.Lstat(filepath.Join(claudeDir, ".credentials.json"))
	if err != nil {
		t.Fatalf("Lstat creds: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf(".credentials.json should still be a symlink")
	}
}

// TestUnmanageActive_ClearsConfig verifies that config.json has active="" after unmanage.
func TestUnmanageActive_ClearsConfig(t *testing.T) {
	claudeDir := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")
	makeConfigFile(t, configPath, "myprofile")

	_, err := profile.UnmanageActive(claudeDir, configPath, false)
	if err != nil {
		t.Fatalf("UnmanageActive: %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Active != "" {
		t.Errorf("Active = %q, want empty string", cfg.Active)
	}
}

// TestUnmanageActive_DryRun verifies that no filesystem changes occur in dry-run mode.
func TestUnmanageActive_DryRun(t *testing.T) {
	realDir := t.TempDir()
	realFile := filepath.Join(realDir, "myfile.txt")
	if err := os.WriteFile(realFile, []byte("content"), 0644); err != nil {
		t.Fatalf("create real file: %v", err)
	}

	claudeDir := t.TempDir()
	linkPath := filepath.Join(claudeDir, "myfile.txt")
	if err := os.Symlink(realFile, linkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	configPath := filepath.Join(t.TempDir(), "config.json")
	makeConfigFile(t, configPath, "myprofile")

	would, err := profile.UnmanageActive(claudeDir, configPath, true)
	if err != nil {
		t.Fatalf("UnmanageActive (dry-run): %v", err)
	}
	if len(would) != 1 {
		t.Errorf("len(would) = %d, want 1", len(would))
	}

	// linkPath should still be a symlink — dry-run made no changes
	fi, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("Lstat: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlink to remain in dry-run, got real file")
	}

	// Config should still have active profile
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Active != "myprofile" {
		t.Errorf("Active = %q, want myprofile in dry-run", cfg.Active)
	}
}

// TestUnmanageActive_DirectorySymlink verifies that a symlink to a directory is
// materialized as a real directory with copied contents.
func TestUnmanageActive_DirectorySymlink(t *testing.T) {
	// Create a real directory with some content
	realDir := t.TempDir()
	realSubDir := filepath.Join(realDir, "mydir")
	if err := os.MkdirAll(realSubDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(realSubDir, "inner.txt"), []byte("inner content"), 0644); err != nil {
		t.Fatalf("create inner file: %v", err)
	}

	// claudeDir has a symlink pointing to the directory
	claudeDir := t.TempDir()
	linkPath := filepath.Join(claudeDir, "mydir")
	if err := os.Symlink(realSubDir, linkPath); err != nil {
		t.Fatalf("create dir symlink: %v", err)
	}

	configPath := filepath.Join(t.TempDir(), "config.json")
	makeConfigFile(t, configPath, "myprofile")

	materialized, err := profile.UnmanageActive(claudeDir, configPath, false)
	if err != nil {
		t.Fatalf("UnmanageActive: %v", err)
	}
	if len(materialized) != 1 {
		t.Errorf("len(materialized) = %d, want 1", len(materialized))
	}

	// linkPath should now be a real directory, not a symlink
	fi, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("Lstat: %v", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Errorf("expected real dir, got symlink")
	}
	if !fi.IsDir() {
		t.Errorf("expected directory, got file")
	}

	// Inner content should be present
	innerData, err := os.ReadFile(filepath.Join(linkPath, "inner.txt"))
	if err != nil {
		t.Fatalf("ReadFile inner.txt: %v", err)
	}
	if string(innerData) != "inner content" {
		t.Errorf("inner content = %q, want %q", string(innerData), "inner content")
	}
}
