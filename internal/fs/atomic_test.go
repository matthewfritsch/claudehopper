package fs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matthewfritsch/claudehopper/internal/fs"
)

func TestAtomicSymlink_CreatesNewSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")

	// Create target file so it's a valid dereferenceable link
	if err := os.WriteFile(target, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := fs.AtomicSymlink(target, link); err != nil {
		t.Fatalf("AtomicSymlink returned error: %v", err)
	}

	got, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("Readlink failed: %v", err)
	}
	if got != target {
		t.Errorf("Readlink = %q; want %q", got, target)
	}
}

func TestAtomicSymlink_ReplacesExistingSymlink(t *testing.T) {
	dir := t.TempDir()
	oldTarget := filepath.Join(dir, "old-target")
	newTarget := filepath.Join(dir, "new-target")
	link := filepath.Join(dir, "link")

	// Create both targets
	for _, p := range []string{oldTarget, newTarget} {
		if err := os.WriteFile(p, []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// First create a symlink pointing to oldTarget
	if err := os.Symlink(oldTarget, link); err != nil {
		t.Fatal(err)
	}

	// Now atomically replace with newTarget
	if err := fs.AtomicSymlink(newTarget, link); err != nil {
		t.Fatalf("AtomicSymlink returned error: %v", err)
	}

	got, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("Readlink failed: %v", err)
	}
	if got != newTarget {
		t.Errorf("Readlink = %q; want %q", got, newTarget)
	}
}

func TestAtomicSymlink_DanglingLinkIsValid(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "nonexistent-target")
	link := filepath.Join(dir, "link")

	// Target does not exist — dangling symlink should still be created
	if err := fs.AtomicSymlink(target, link); err != nil {
		t.Fatalf("AtomicSymlink returned error for dangling link: %v", err)
	}

	// Use Lstat (not Stat) to check that the symlink itself exists
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("Lstat failed: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("Expected symlink, got mode %v", info.Mode())
	}

	got, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("Readlink failed: %v", err)
	}
	if got != target {
		t.Errorf("Readlink = %q; want %q", got, target)
	}
}

func TestAtomicSymlink_AbsoluteTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "profile-dir")
	link := filepath.Join(dir, "current")

	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := fs.AtomicSymlink(target, link); err != nil {
		t.Fatalf("AtomicSymlink returned error: %v", err)
	}

	got, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("Readlink failed: %v", err)
	}
	if got != target {
		t.Errorf("Readlink = %q; want %q", got, target)
	}

	// Dereference via os.Stat (should succeed — not dangling)
	if _, err := os.Stat(link); err != nil {
		t.Errorf("Stat on symlink failed (dangling?): %v", err)
	}
}
