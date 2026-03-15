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

// ---- helpers for DoSwitch tests ----

// makeProfile creates a profile directory with the given managed files and a .hop-manifest.json.
func makeProfile(t *testing.T, profilesDir, name string, managed []string) string {
	t.Helper()
	dir := filepath.Join(profilesDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, f := range managed {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("content:"+f), 0644); err != nil {
			t.Fatal(err)
		}
	}
	testWriteManifest(t, dir, managed, nil, "test profile "+name)
	return dir
}

// readConfig loads config.json from configPath.
func readConfig(t *testing.T, configPath string) config.Config {
	t.Helper()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	return cfg
}

// ---- DoSwitch ----

func TestDoSwitch_DryRun_NoFilesystemChanges(t *testing.T) {
	profilesDir := t.TempDir()
	claudeDir := t.TempDir()
	sharedDir := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")

	makeProfile(t, profilesDir, "alpha", []string{"settings.json"})
	makeProfile(t, profilesDir, "beta", []string{"settings.json", "keybindings.json"})

	// Capture claudeDir state before
	before, _ := os.ReadDir(claudeDir)

	result, err := DoSwitch(profilesDir, claudeDir, configPath, sharedDir, "beta", "alpha", SwitchOptions{DryRun: true})
	if err != nil {
		t.Fatalf("DoSwitch dry-run: %v", err)
	}
	if len(result.Actions) == 0 {
		t.Error("dry-run should return planned actions")
	}

	// claudeDir should be unchanged
	after, _ := os.ReadDir(claudeDir)
	if len(before) != len(after) {
		t.Errorf("dry-run modified claudeDir: before %d entries, after %d entries", len(before), len(after))
	}

	// config should not have been written
	if _, err := os.Stat(configPath); err == nil {
		t.Error("dry-run should not write config.json")
	}
}

func TestDoSwitch_AlreadyActive_ErrorWithoutForce(t *testing.T) {
	profilesDir := t.TempDir()
	claudeDir := t.TempDir()
	sharedDir := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")

	makeProfile(t, profilesDir, "alpha", []string{"settings.json"})

	_, err := DoSwitch(profilesDir, claudeDir, configPath, sharedDir, "alpha", "alpha", SwitchOptions{})
	if err == nil {
		t.Fatal("expected error for same-profile switch, got nil")
	}
}

func TestDoSwitch_AlreadyActive_ForceRelinks(t *testing.T) {
	profilesDir := t.TempDir()
	claudeDir := t.TempDir()
	sharedDir := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")

	makeProfile(t, profilesDir, "alpha", []string{"settings.json"})

	result, err := DoSwitch(profilesDir, claudeDir, configPath, sharedDir, "alpha", "alpha", SwitchOptions{Force: true})
	if err != nil {
		t.Fatalf("DoSwitch --force: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// symlink should exist
	fi, err := os.Lstat(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatalf("Lstat after force switch: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink, got regular file")
	}
}

func TestDoSwitch_UnlinksOldLinksNewPaths(t *testing.T) {
	profilesDir := t.TempDir()
	claudeDir := t.TempDir()
	sharedDir := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")

	// alpha has keybindings.json, beta has settings.json
	makeProfile(t, profilesDir, "alpha", []string{"keybindings.json"})
	makeProfile(t, profilesDir, "beta", []string{"settings.json"})

	// First switch to alpha to set up initial links
	if _, err := DoSwitch(profilesDir, claudeDir, configPath, sharedDir, "alpha", "", SwitchOptions{}); err != nil {
		t.Fatalf("initial switch to alpha: %v", err)
	}

	// Now switch to beta
	if _, err := DoSwitch(profilesDir, claudeDir, configPath, sharedDir, "beta", "alpha", SwitchOptions{}); err != nil {
		t.Fatalf("switch to beta: %v", err)
	}

	// alpha's keybindings.json symlink should be gone
	if _, err := os.Lstat(filepath.Join(claudeDir, "keybindings.json")); err == nil {
		t.Error("keybindings.json should have been unlinked after switching away from alpha")
	}

	// beta's settings.json symlink should exist
	fi, err := os.Lstat(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatalf("settings.json not found after switch to beta: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("settings.json should be a symlink")
	}
}

func TestDoSwitch_SavesConfig(t *testing.T) {
	profilesDir := t.TempDir()
	claudeDir := t.TempDir()
	sharedDir := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")

	makeProfile(t, profilesDir, "alpha", []string{"settings.json"})
	makeProfile(t, profilesDir, "beta", []string{"settings.json"})

	if _, err := DoSwitch(profilesDir, claudeDir, configPath, sharedDir, "beta", "alpha", SwitchOptions{}); err != nil {
		t.Fatalf("DoSwitch: %v", err)
	}

	cfg := readConfig(t, configPath)
	if cfg.Active != "beta" {
		t.Errorf("config.Active: got %q, want %q", cfg.Active, "beta")
	}
}

func TestDoSwitch_BacksUpConflictingFiles(t *testing.T) {
	profilesDir := t.TempDir()
	claudeDir := t.TempDir()
	sharedDir := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")

	makeProfile(t, profilesDir, "beta", []string{"settings.json"})

	// Place a real file at the conflict location
	conflictPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(conflictPath, []byte("old real file"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := DoSwitch(profilesDir, claudeDir, configPath, sharedDir, "beta", "", SwitchOptions{})
	if err != nil {
		t.Fatalf("DoSwitch: %v", err)
	}

	if len(result.BackedUp) == 0 {
		t.Error("expected BackedUp to list conflict, got empty")
	}

	// Backup should exist
	if _, err := os.Stat(conflictPath + ".hop-backup"); os.IsNotExist(err) {
		t.Error("expected .hop-backup to exist")
	}
}

func TestDoSwitch_ReturnsSwitchResult(t *testing.T) {
	profilesDir := t.TempDir()
	claudeDir := t.TempDir()
	sharedDir := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")

	makeProfile(t, profilesDir, "beta", []string{"settings.json", "keybindings.json"})

	result, err := DoSwitch(profilesDir, claudeDir, configPath, sharedDir, "beta", "", SwitchOptions{})
	if err != nil {
		t.Fatalf("DoSwitch: %v", err)
	}
	if result == nil {
		t.Fatal("nil result")
	}
	if len(result.Actions) == 0 {
		t.Error("expected Actions in result")
	}
}

// ---- AdoptUnmanaged ----

func TestAdoptUnmanaged_MovesFilesAndUpdatesManifest(t *testing.T) {
	profileDir := t.TempDir()
	claudeDir := t.TempDir()

	// Create files in claudeDir to be adopted
	for _, name := range []string{"myext.json", "custom.md"} {
		if err := os.WriteFile(filepath.Join(claudeDir, name), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Write initial manifest
	testWriteManifest(t, profileDir, []string{"settings.json"}, nil, "test")
	manifest, err := config.LoadManifest(filepath.Join(profileDir, ".hop-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}

	if err := AdoptUnmanaged(claudeDir, profileDir, &manifest, []string{"myext.json", "custom.md"}); err != nil {
		t.Fatalf("AdoptUnmanaged: %v", err)
	}

	// Files should be in profileDir
	for _, name := range []string{"myext.json", "custom.md"} {
		if _, err := os.Stat(filepath.Join(profileDir, name)); err != nil {
			t.Errorf("expected %s in profileDir, got: %v", name, err)
		}
		// Should no longer be in claudeDir
		if _, err := os.Stat(filepath.Join(claudeDir, name)); err == nil {
			t.Errorf("expected %s to be gone from claudeDir", name)
		}
	}

	// Manifest should include adopted files
	updatedManifest, err := config.LoadManifest(filepath.Join(profileDir, ".hop-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	managedSet := make(map[string]struct{})
	for _, p := range updatedManifest.ManagedPaths {
		managedSet[p] = struct{}{}
	}
	for _, name := range []string{"myext.json", "custom.md"} {
		if _, ok := managedSet[name]; !ok {
			t.Errorf("expected %s in manifest managed_paths after adopt", name)
		}
	}
}
