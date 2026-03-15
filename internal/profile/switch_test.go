package profile

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/matthewfritsch/claudehopper/internal/config"
)

// ---- ValidatePreflight ----

func TestValidatePreflight_AllPresent(t *testing.T) {
	profileDir := t.TempDir()
	claudeDir := t.TempDir()

	// Create managed files in profileDir
	managed := []string{"settings.json", "keybindings.json"}
	for _, name := range managed {
		if err := os.WriteFile(filepath.Join(profileDir, name), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	m := config.Manifest{ManagedPaths: managed, SharedPaths: map[string]string{}}

	actions, err := ValidatePreflight(profileDir, claudeDir, m)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if len(actions) == 0 {
		t.Fatal("expected some planned actions, got none")
	}
}

func TestValidatePreflight_MissingPaths(t *testing.T) {
	profileDir := t.TempDir()
	claudeDir := t.TempDir()

	// Only create one of the two managed files
	managed := []string{"settings.json", "missing.json"}
	if err := os.WriteFile(filepath.Join(profileDir, "settings.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	m := config.Manifest{ManagedPaths: managed, SharedPaths: map[string]string{}}

	_, err := ValidatePreflight(profileDir, claudeDir, m)
	if err == nil {
		t.Fatal("expected error for missing paths, got nil")
	}
	if !strings.Contains(err.Error(), "missing.json") {
		t.Errorf("expected error to mention missing.json, got: %v", err)
	}
}

// ---- backupPath ----

func TestBackupPath_NoExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	got := backupPath(path)
	want := path + ".hop-backup"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBackupPath_OneExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	// Create the .hop-backup
	if err := os.WriteFile(path+".hop-backup", []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	got := backupPath(path)
	want := path + ".hop-backup.1"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBackupPath_TwoExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	// Create both .hop-backup and .hop-backup.1
	if err := os.WriteFile(path+".hop-backup", []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".hop-backup.1", []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	got := backupPath(path)
	want := path + ".hop-backup.2"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBackupPath_DanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	// Create a dangling symlink at .hop-backup
	if err := os.Symlink("/nonexistent/target", path+".hop-backup"); err != nil {
		t.Fatal(err)
	}
	// Lstat should detect the dangling symlink as existing
	got := backupPath(path)
	want := path + ".hop-backup.1"
	if got != want {
		t.Errorf("dangling symlink not detected by Lstat; got %q, want %q", got, want)
	}
}

// ---- linkManagedPath ----

func TestLinkManagedPath_RegularFile(t *testing.T) {
	profileDir := t.TempDir()
	claudeDir := t.TempDir()

	// Create a regular file in profileDir
	content := []byte("profile content")
	if err := os.WriteFile(filepath.Join(profileDir, "settings.json"), content, 0644); err != nil {
		t.Fatal(err)
	}

	backedUp, err := linkManagedPath(profileDir, claudeDir, "settings.json")
	if err != nil {
		t.Fatalf("linkManagedPath: %v", err)
	}
	if backedUp {
		t.Error("expected backedUp=false for new link, got true")
	}

	// Verify symlink created in claudeDir
	linkPath := filepath.Join(claudeDir, "settings.json")
	fi, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("Lstat link: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink at claudeDir/settings.json, got regular file")
	}

	// Verify symlink target is absolute path of profileDir/settings.json
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatal(err)
	}
	wantTarget, _ := filepath.Abs(filepath.Join(profileDir, "settings.json"))
	if target != wantTarget {
		t.Errorf("symlink target: got %q, want %q", target, wantTarget)
	}
}

func TestLinkManagedPath_PreservesSymlinkTarget(t *testing.T) {
	profileDir := t.TempDir()
	claudeDir := t.TempDir()
	sharedDir := t.TempDir()

	// Create a file in shared dir
	sharedFile := filepath.Join(sharedDir, "settings.json")
	if err := os.WriteFile(sharedFile, []byte("shared"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create a symlink in profileDir -> sharedFile
	if err := os.Symlink(sharedFile, filepath.Join(profileDir, "settings.json")); err != nil {
		t.Fatal(err)
	}

	_, err := linkManagedPath(profileDir, claudeDir, "settings.json")
	if err != nil {
		t.Fatalf("linkManagedPath: %v", err)
	}

	linkPath := filepath.Join(claudeDir, "settings.json")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	// Target should be sharedFile, not profileDir/settings.json
	if target != sharedFile {
		t.Errorf("expected target %q (from profile symlink), got %q", sharedFile, target)
	}
}

func TestLinkManagedPath_BacksUpConflictingFile(t *testing.T) {
	profileDir := t.TempDir()
	claudeDir := t.TempDir()

	// Create profile file
	if err := os.WriteFile(filepath.Join(profileDir, "settings.json"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create conflicting real file in claudeDir
	conflictPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(conflictPath, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	backedUp, err := linkManagedPath(profileDir, claudeDir, "settings.json")
	if err != nil {
		t.Fatalf("linkManagedPath: %v", err)
	}
	if !backedUp {
		t.Error("expected backedUp=true for conflicting file, got false")
	}

	// Backup should exist
	backupFile := conflictPath + ".hop-backup"
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		t.Error("expected .hop-backup file to exist, but it does not")
	}

	// Symlink should now exist at original path
	fi, err := os.Lstat(conflictPath)
	if err != nil {
		t.Fatalf("Lstat after link: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink after backup, got regular file")
	}
}

func TestLinkManagedPath_BacksUpConflictingDir(t *testing.T) {
	profileDir := t.TempDir()
	claudeDir := t.TempDir()

	// Create profile file
	if err := os.WriteFile(filepath.Join(profileDir, "mydir"), []byte("profile"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create conflicting directory in claudeDir
	conflictPath := filepath.Join(claudeDir, "mydir")
	if err := os.MkdirAll(conflictPath, 0755); err != nil {
		t.Fatal(err)
	}

	backedUp, err := linkManagedPath(profileDir, claudeDir, "mydir")
	if err != nil {
		t.Fatalf("linkManagedPath: %v", err)
	}
	if !backedUp {
		t.Error("expected backedUp=true for conflicting dir, got false")
	}

	// Backup directory should exist
	backupPath := conflictPath + ".hop-backup"
	fi, err := os.Lstat(backupPath)
	if err != nil {
		t.Fatalf("Lstat backup: %v", err)
	}
	if !fi.IsDir() {
		t.Error("expected backup to be a directory")
	}
}

// ---- DetectUnmanaged ----

func TestDetectUnmanaged_FiltersCorrectly(t *testing.T) {
	claudeDir := t.TempDir()
	sharedDir := t.TempDir()

	// Create various files
	files := map[string]string{
		"myext.json":               "unmanaged - should be returned",
		"settings.json":            "managed - should be filtered",
		".credentials.json":        "protected - should be filtered",
		".hop-foo":                 ".hop- prefix - should be filtered",
		".ccswap123":               ".ccswap prefix - should be filtered",
		"file.hop-backup":          "hop-backup pattern - should be filtered",
		"file.hop-backup.1":        "hop-backup.1 pattern - should be filtered",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(claudeDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create a symlink pointing into sharedDir
	sharedTarget := filepath.Join(sharedDir, "shared.json")
	if err := os.WriteFile(sharedTarget, []byte("shared"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sharedTarget, filepath.Join(claudeDir, "shared.json")); err != nil {
		t.Fatal(err)
	}

	managedPaths := []string{"settings.json"}
	unmanaged, err := DetectUnmanaged(claudeDir, sharedDir, managedPaths)
	if err != nil {
		t.Fatalf("DetectUnmanaged: %v", err)
	}

	// Only "myext.json" should be returned
	if len(unmanaged) != 1 || unmanaged[0] != "myext.json" {
		t.Errorf("expected [myext.json], got %v", unmanaged)
	}
}

func TestDetectUnmanaged_SkipsSharedSymlinks(t *testing.T) {
	claudeDir := t.TempDir()
	sharedDir := t.TempDir()

	// Create a symlink pointing into sharedDir
	sharedTarget := filepath.Join(sharedDir, "settings.json")
	if err := os.WriteFile(sharedTarget, []byte("shared"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sharedTarget, filepath.Join(claudeDir, "settings.json")); err != nil {
		t.Fatal(err)
	}

	unmanaged, err := DetectUnmanaged(claudeDir, sharedDir, []string{})
	if err != nil {
		t.Fatalf("DetectUnmanaged: %v", err)
	}

	for _, name := range unmanaged {
		if name == "settings.json" {
			t.Error("shared symlink should be filtered from unmanaged list")
		}
	}
}

func TestDetectUnmanaged_ReturnsSorted(t *testing.T) {
	claudeDir := t.TempDir()
	sharedDir := t.TempDir()

	// Create some unmanaged files (not protected, not .hop-, not managed)
	for _, name := range []string{"zzz.json", "aaa.json", "mmm.json"} {
		if err := os.WriteFile(filepath.Join(claudeDir, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	unmanaged, err := DetectUnmanaged(claudeDir, sharedDir, []string{})
	if err != nil {
		t.Fatalf("DetectUnmanaged: %v", err)
	}

	if !sort.StringsAreSorted(unmanaged) {
		t.Errorf("DetectUnmanaged result not sorted: %v", unmanaged)
	}
}
