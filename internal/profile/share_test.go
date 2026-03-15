package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matthewfritsch/claudehopper/internal/config"
)

// makeEmptyProfile creates a minimal profile directory with an empty manifest.
func makeEmptyProfile(t *testing.T, profilesDir, name string) string {
	t.Helper()
	dir := filepath.Join(profilesDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("makeEmptyProfile MkdirAll: %v", err)
	}
	m := config.NewManifest("")
	mPath := filepath.Join(dir, ".hop-manifest.json")
	if err := config.SaveManifest(mPath, m); err != nil {
		t.Fatalf("makeEmptyProfile SaveManifest: %v", err)
	}
	return dir
}

// shareTestWriteFile creates a regular file in dir with given name and content.
func shareTestWriteFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("shareTestWriteFile: %v", err)
	}
	return path
}

// shareTestLoadManifest is a test helper that loads manifest or fails.
func shareTestLoadManifest(t *testing.T, dir string) config.Manifest {
	t.Helper()
	m, err := config.LoadManifest(filepath.Join(dir, ".hop-manifest.json"))
	if err != nil {
		t.Fatalf("shareTestLoadManifest: %v", err)
	}
	return m
}

func TestShareFiles(t *testing.T) {
	profilesDir := t.TempDir()
	srcDir := makeEmptyProfile(t, profilesDir, "src")
	tgtDir := makeEmptyProfile(t, profilesDir, "tgt")

	// Create a file in src
	shareTestWriteFile(t, srcDir, "agent.md", "hello")

	shared, err := ShareFiles(profilesDir, "src", "tgt", []string{"agent.md"}, false)
	if err != nil {
		t.Fatalf("ShareFiles: %v", err)
	}
	if len(shared) != 1 {
		t.Fatalf("expected 1 shared path, got %d", len(shared))
	}

	// Verify symlink exists in target
	linkPath := filepath.Join(tgtDir, "agent.md")
	fi, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("Lstat link: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected symlink, got %v", fi.Mode())
	}

	// Verify symlink points to the real source file
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	srcFile := filepath.Join(srcDir, "agent.md")
	if target != srcFile {
		t.Fatalf("symlink target = %q, want %q", target, srcFile)
	}

	// Verify manifest updated
	m := shareTestLoadManifest(t, tgtDir)
	if m.SharedPaths["agent.md"] != "src" {
		t.Fatalf("SharedPaths[agent.md] = %q, want %q", m.SharedPaths["agent.md"], "src")
	}
	// managed_paths should contain agent.md
	found := false
	for _, p := range m.ManagedPaths {
		if p == "agent.md" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("agent.md not in ManagedPaths: %v", m.ManagedPaths)
	}
}

func TestShareFiles_ResolvesChainedSymlinks(t *testing.T) {
	profilesDir := t.TempDir()
	srcDir := makeEmptyProfile(t, profilesDir, "src")
	_ = makeEmptyProfile(t, profilesDir, "tgt")

	// Create a real file and a symlink pointing to it in src
	realFile := shareTestWriteFile(t, srcDir, "real.md", "real content")
	symlinkInSrc := filepath.Join(srcDir, "link.md")
	if err := os.Symlink(realFile, symlinkInSrc); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	shared, err := ShareFiles(profilesDir, "src", "tgt", []string{"link.md"}, false)
	if err != nil {
		t.Fatalf("ShareFiles: %v", err)
	}
	if len(shared) != 1 {
		t.Fatalf("expected 1 shared path, got %d", len(shared))
	}

	// The target symlink should resolve to the real file, not be chained
	tgtDir := filepath.Join(profilesDir, "tgt")
	linkPath := filepath.Join(tgtDir, "link.md")
	resolved, err := filepath.EvalSymlinks(linkPath)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	if resolved != realFile {
		t.Fatalf("resolved = %q, want %q", resolved, realFile)
	}
}

func TestShareFiles_DryRun(t *testing.T) {
	profilesDir := t.TempDir()
	srcDir := makeEmptyProfile(t, profilesDir, "src")
	tgtDir := makeEmptyProfile(t, profilesDir, "tgt")

	shareTestWriteFile(t, srcDir, "agent.md", "hello")

	shared, err := ShareFiles(profilesDir, "src", "tgt", []string{"agent.md"}, true)
	if err != nil {
		t.Fatalf("ShareFiles dry-run: %v", err)
	}
	if len(shared) != 1 {
		t.Fatalf("expected 1 path in dry-run result, got %d", len(shared))
	}

	// No symlink should have been created
	linkPath := filepath.Join(tgtDir, "agent.md")
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Fatalf("dry-run should not create symlink, but Lstat returned: %v", err)
	}

	// Manifest should not have been updated
	m := shareTestLoadManifest(t, tgtDir)
	if len(m.SharedPaths) != 0 {
		t.Fatalf("dry-run should not update manifest SharedPaths: %v", m.SharedPaths)
	}
}

func TestShareFiles_Dedup(t *testing.T) {
	profilesDir := t.TempDir()
	srcDir := makeEmptyProfile(t, profilesDir, "src")
	tgtDir := makeEmptyProfile(t, profilesDir, "tgt")

	shareTestWriteFile(t, srcDir, "agent.md", "hello")

	// Share twice
	if _, err := ShareFiles(profilesDir, "src", "tgt", []string{"agent.md"}, false); err != nil {
		t.Fatalf("first ShareFiles: %v", err)
	}
	if _, err := ShareFiles(profilesDir, "src", "tgt", []string{"agent.md"}, false); err != nil {
		t.Fatalf("second ShareFiles: %v", err)
	}

	// managed_paths should not have duplicates
	m := shareTestLoadManifest(t, tgtDir)
	count := 0
	for _, p := range m.ManagedPaths {
		if p == "agent.md" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("managed_paths has %d copies of agent.md, want 1: %v", count, m.ManagedPaths)
	}
}

func TestPickFiles_RegularFile(t *testing.T) {
	profilesDir := t.TempDir()
	srcDir := makeEmptyProfile(t, profilesDir, "src")
	tgtDir := makeEmptyProfile(t, profilesDir, "tgt")

	shareTestWriteFile(t, srcDir, "config.json", `{"key":"value"}`)

	picked, err := PickFiles(profilesDir, "src", "tgt", []string{"config.json"}, false)
	if err != nil {
		t.Fatalf("PickFiles: %v", err)
	}
	if len(picked) != 1 {
		t.Fatalf("expected 1 picked path, got %d", len(picked))
	}

	// Verify file content in target
	dstPath := filepath.Join(tgtDir, "config.json")
	data, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("ReadFile dst: %v", err)
	}
	if string(data) != `{"key":"value"}` {
		t.Fatalf("content = %q, want %q", string(data), `{"key":"value"}`)
	}

	// Destination should be a real file, not a symlink
	fi, err := os.Lstat(dstPath)
	if err != nil {
		t.Fatalf("Lstat dst: %v", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("expected regular file, got symlink")
	}

	// Manifest updated
	m := shareTestLoadManifest(t, tgtDir)
	found := false
	for _, p := range m.ManagedPaths {
		if p == "config.json" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("config.json not in ManagedPaths: %v", m.ManagedPaths)
	}
}

func TestPickFiles_Symlink(t *testing.T) {
	profilesDir := t.TempDir()
	srcDir := makeEmptyProfile(t, profilesDir, "src")
	tgtDir := makeEmptyProfile(t, profilesDir, "tgt")

	// Create a real file and a symlink in src
	realFile := shareTestWriteFile(t, srcDir, "real.md", "content")
	symlinkInSrc := filepath.Join(srcDir, "link.md")
	if err := os.Symlink(realFile, symlinkInSrc); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	_, err := PickFiles(profilesDir, "src", "tgt", []string{"link.md"}, false)
	if err != nil {
		t.Fatalf("PickFiles: %v", err)
	}

	// Destination should also be a symlink
	dstPath := filepath.Join(tgtDir, "link.md")
	fi, err := os.Lstat(dstPath)
	if err != nil {
		t.Fatalf("Lstat dst: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected symlink in target, got %v", fi.Mode())
	}

	// Symlink target should match original
	target, err := os.Readlink(dstPath)
	if err != nil {
		t.Fatalf("Readlink dst: %v", err)
	}
	if target != realFile {
		t.Fatalf("symlink target = %q, want %q", target, realFile)
	}
}

func TestPickFiles_Directory(t *testing.T) {
	profilesDir := t.TempDir()
	srcDir := makeEmptyProfile(t, profilesDir, "src")
	tgtDir := makeEmptyProfile(t, profilesDir, "tgt")

	// Create a directory with files in src
	subDir := filepath.Join(srcDir, "mydir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	shareTestWriteFile(t, subDir, "a.txt", "aaa")
	shareTestWriteFile(t, subDir, "b.txt", "bbb")

	_, err := PickFiles(profilesDir, "src", "tgt", []string{"mydir"}, false)
	if err != nil {
		t.Fatalf("PickFiles directory: %v", err)
	}

	// Verify directory and contents in target
	dstDir := filepath.Join(tgtDir, "mydir")
	aPath := filepath.Join(dstDir, "a.txt")
	bPath := filepath.Join(dstDir, "b.txt")

	for _, p := range []string{aPath, bPath} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected file %s: %v", p, err)
		}
	}

	data, _ := os.ReadFile(aPath)
	if string(data) != "aaa" {
		t.Fatalf("a.txt content = %q, want %q", string(data), "aaa")
	}
}

func TestUnshareFiles_ReplacesSymlink(t *testing.T) {
	profilesDir := t.TempDir()
	srcDir := makeEmptyProfile(t, profilesDir, "src")
	tgtDir := makeEmptyProfile(t, profilesDir, "tgt")

	// Create a real file in src
	shareTestWriteFile(t, srcDir, "agent.md", "shared content")

	// Share it first
	if _, err := ShareFiles(profilesDir, "src", "tgt", []string{"agent.md"}, false); err != nil {
		t.Fatalf("ShareFiles: %v", err)
	}

	// Now unshare
	unshared, err := UnshareFiles(profilesDir, "tgt", []string{"agent.md"}, false)
	if err != nil {
		t.Fatalf("UnshareFiles: %v", err)
	}
	if len(unshared) != 1 {
		t.Fatalf("expected 1 unshared path, got %d", len(unshared))
	}

	// Verify real file exists at target (not symlink)
	dstPath := filepath.Join(tgtDir, "agent.md")
	fi, err := os.Lstat(dstPath)
	if err != nil {
		t.Fatalf("Lstat after unshare: %v", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("expected real file after unshare, got symlink")
	}

	// Content should match
	data, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("ReadFile after unshare: %v", err)
	}
	if string(data) != "shared content" {
		t.Fatalf("content = %q, want %q", string(data), "shared content")
	}
}

func TestUnshareFiles_ManifestCleaned(t *testing.T) {
	profilesDir := t.TempDir()
	srcDir := makeEmptyProfile(t, profilesDir, "src")
	tgtDir := makeEmptyProfile(t, profilesDir, "tgt")

	shareTestWriteFile(t, srcDir, "agent.md", "content")

	if _, err := ShareFiles(profilesDir, "src", "tgt", []string{"agent.md"}, false); err != nil {
		t.Fatalf("ShareFiles: %v", err)
	}

	if _, err := UnshareFiles(profilesDir, "tgt", []string{"agent.md"}, false); err != nil {
		t.Fatalf("UnshareFiles: %v", err)
	}

	m := shareTestLoadManifest(t, tgtDir)
	if _, ok := m.SharedPaths["agent.md"]; ok {
		t.Fatalf("shared_paths should not contain agent.md after unshare: %v", m.SharedPaths)
	}

	// Should be in managed_paths
	found := false
	for _, p := range m.ManagedPaths {
		if p == "agent.md" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("agent.md should be in managed_paths after unshare: %v", m.ManagedPaths)
	}
}

func TestUnshareFiles_MissingTarget(t *testing.T) {
	profilesDir := t.TempDir()
	srcDir := makeEmptyProfile(t, profilesDir, "src")
	tgtDir := makeEmptyProfile(t, profilesDir, "tgt")

	shareTestWriteFile(t, srcDir, "agent.md", "content")

	if _, err := ShareFiles(profilesDir, "src", "tgt", []string{"agent.md"}, false); err != nil {
		t.Fatalf("ShareFiles: %v", err)
	}

	// Delete the source file so symlink target is missing
	if err := os.Remove(filepath.Join(srcDir, "agent.md")); err != nil {
		t.Fatalf("Remove source: %v", err)
	}

	// Unshare should handle gracefully (not return error)
	unshared, err := UnshareFiles(profilesDir, "tgt", []string{"agent.md"}, false)
	if err != nil {
		t.Fatalf("UnshareFiles with missing target: %v", err)
	}
	if len(unshared) != 1 {
		t.Fatalf("expected 1 unshared path even with missing target, got %d", len(unshared))
	}

	// Manifest should be cleaned
	m := shareTestLoadManifest(t, tgtDir)
	if _, ok := m.SharedPaths["agent.md"]; ok {
		t.Fatalf("shared_paths should be cleared: %v", m.SharedPaths)
	}
}

func TestUnshareFiles_AllPaths(t *testing.T) {
	profilesDir := t.TempDir()
	srcDir := makeEmptyProfile(t, profilesDir, "src")
	tgtDir := makeEmptyProfile(t, profilesDir, "tgt")

	shareTestWriteFile(t, srcDir, "a.md", "a")
	shareTestWriteFile(t, srcDir, "b.md", "b")

	if _, err := ShareFiles(profilesDir, "src", "tgt", []string{"a.md", "b.md"}, false); err != nil {
		t.Fatalf("ShareFiles: %v", err)
	}

	// Unshare with empty paths — should unshare all
	unshared, err := UnshareFiles(profilesDir, "tgt", []string{}, false)
	if err != nil {
		t.Fatalf("UnshareFiles all: %v", err)
	}
	if len(unshared) != 2 {
		t.Fatalf("expected 2 unshared paths, got %d", len(unshared))
	}

	m := shareTestLoadManifest(t, tgtDir)
	if len(m.SharedPaths) != 0 {
		t.Fatalf("shared_paths should be empty after unshare all: %v", m.SharedPaths)
	}
}
