package updater

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCheckForUpdate_SkipsWithinTTL verifies that when a stamp file is less than
// 24h old, CheckForUpdate returns nil without making any network call.
func TestCheckForUpdate_SkipsWithinTTL(t *testing.T) {
	configDir := t.TempDir()
	stampPath := filepath.Join(configDir, stampFileName)

	// Write a stamp file with a recent ModTime (1 hour ago)
	if err := os.WriteFile(stampPath, []byte{}, 0644); err != nil {
		t.Fatalf("setup: write stamp: %v", err)
	}
	recentTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(stampPath, recentTime, recentTime); err != nil {
		t.Fatalf("setup: chtimes: %v", err)
	}

	// Should return nil immediately without network — we pass a fake version
	// and the real CheckForUpdate must skip because TTL has not expired.
	result, err := CheckForUpdate(context.Background(), configDir, "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result within TTL window, got %+v", result)
	}
}

// TestCheckForUpdate_StampMissing verifies that when no stamp file exists the
// function proceeds with the check (first-run case).  We use a detectFunc
// override so no real network call is made.
func TestCheckForUpdate_StampMissing(t *testing.T) {
	configDir := t.TempDir()
	// No stamp file created — directory is empty.

	// Use detectFunc override to simulate a newer version being available.
	origDetect := detectFunc
	defer func() { detectFunc = origDetect }()
	detectFunc = func(_ context.Context, _ string) (*releaseInfo, error) {
		return &releaseInfo{version: "2.0.0", url: "https://example.com"}, nil
	}

	result, err := CheckForUpdate(context.Background(), configDir, "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected update info when newer version available, got nil")
	}
	if result.Version != "2.0.0" {
		t.Errorf("expected Version=2.0.0 got %q", result.Version)
	}
}

// TestCheckForUpdate_CallsAPIAfterTTL verifies that an expired stamp triggers a
// network call (simulated via detectFunc) and returns an UpdateInfo when a
// newer version exists.
func TestCheckForUpdate_CallsAPIAfterTTL(t *testing.T) {
	configDir := t.TempDir()
	stampPath := filepath.Join(configDir, stampFileName)

	// Stamp file is 25 hours old — TTL has expired.
	if err := os.WriteFile(stampPath, []byte{}, 0644); err != nil {
		t.Fatalf("setup: write stamp: %v", err)
	}
	oldTime := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(stampPath, oldTime, oldTime); err != nil {
		t.Fatalf("setup: chtimes: %v", err)
	}

	origDetect := detectFunc
	defer func() { detectFunc = origDetect }()
	detectFunc = func(_ context.Context, _ string) (*releaseInfo, error) {
		return &releaseInfo{version: "1.2.0", url: "https://github.com/releases/v1.2.0"}, nil
	}

	result, err := CheckForUpdate(context.Background(), configDir, "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected update info, got nil")
	}
	if result.Version != "1.2.0" {
		t.Errorf("expected Version=1.2.0 got %q", result.Version)
	}
}

// TestCheckForUpdate_AlreadyLatest verifies that when current version equals
// the latest release, CheckForUpdate returns nil.
func TestCheckForUpdate_AlreadyLatest(t *testing.T) {
	configDir := t.TempDir()

	origDetect := detectFunc
	defer func() { detectFunc = origDetect }()
	detectFunc = func(_ context.Context, _ string) (*releaseInfo, error) {
		return &releaseInfo{version: "1.0.0", url: "https://example.com"}, nil
	}

	result, err := CheckForUpdate(context.Background(), configDir, "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil when already at latest, got %+v", result)
	}
}

// TestCheckForUpdate_WritesStampAfterCheck verifies that after a successful
// check the stamp file is written/updated.
func TestCheckForUpdate_WritesStampAfterCheck(t *testing.T) {
	configDir := t.TempDir()
	stampPath := filepath.Join(configDir, stampFileName)

	origDetect := detectFunc
	defer func() { detectFunc = origDetect }()
	detectFunc = func(_ context.Context, _ string) (*releaseInfo, error) {
		return &releaseInfo{version: "1.0.0", url: ""}, nil
	}

	before := time.Now().Add(-time.Second)
	_, err := CheckForUpdate(context.Background(), configDir, "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := time.Now().Add(time.Second)

	fi, err := os.Stat(stampPath)
	if err != nil {
		t.Fatalf("stamp file not created: %v", err)
	}
	mt := fi.ModTime()
	if mt.Before(before) || mt.After(after) {
		t.Errorf("stamp ModTime %v is not in [%v, %v]", mt, before, after)
	}
}
