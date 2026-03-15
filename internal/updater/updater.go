// Package updater implements TTL-cached update checking and self-update for
// claudehopper. It uses the go-selfupdate library for release detection and
// binary replacement, and caches the check result for 24h via a stamp file.
package updater

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	selfupdate "github.com/creativeprojects/go-selfupdate"
)

const (
	ttlDuration   = 24 * time.Hour
	stampFileName = "update-check.stamp"
	repoSlug      = "matthewfritsch/claudehopper"
)

// UpdateInfo holds information about an available update.
type UpdateInfo struct {
	Version    string
	ReleaseURL string
}

// releaseInfo is the internal struct returned by detectFunc.
type releaseInfo struct {
	version string
	url     string
}

// detectFunc is a package-level variable holding the function used to detect
// the latest release. It can be overridden in tests to avoid real network calls.
var detectFunc func(ctx context.Context, slug string) (*releaseInfo, error) = detectLatest

// detectLatest calls the go-selfupdate library to fetch the latest release.
// This is the production implementation of detectFunc.
func detectLatest(ctx context.Context, slug string) (*releaseInfo, error) {
	updater, err := selfupdate.NewUpdater(selfupdate.Config{})
	if err != nil {
		return nil, fmt.Errorf("create updater: %w", err)
	}
	release, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug(slug))
	if err != nil {
		return nil, fmt.Errorf("detect latest: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &releaseInfo{
		version: release.Version(),
		url:     release.URL,
	}, nil
}

// CheckForUpdate checks if a newer version of claudehopper is available on
// GitHub. It reads a stamp file at configDir/update-check.stamp to implement a
// 24h TTL cache — if the stamp is fresh the function returns nil immediately
// without any network call.
//
// On a cache miss (stale or missing stamp), it calls detectFunc to query the
// GitHub Releases API. After any live check (success or failure) the stamp file
// is written so the next call within 24h is a no-op.
//
// Returns a non-nil *UpdateInfo only when a newer version than currentVersion
// is available. Returns nil, nil in all other cases (up to date, check skipped,
// network error).
func CheckForUpdate(ctx context.Context, configDir, currentVersion string) (*UpdateInfo, error) {
	stampPath := filepath.Join(configDir, stampFileName)

	// Check TTL: if stamp exists and is recent, skip the network call.
	if fi, err := os.Stat(stampPath); err == nil {
		if time.Since(fi.ModTime()) < ttlDuration {
			return nil, nil
		}
	}

	// Cache miss — perform the live check.
	info, err := detectFunc(ctx, repoSlug)

	// Write the stamp regardless of detect outcome so we don't hammer GitHub
	// when the network is down.
	writeStamp(stampPath)

	if err != nil {
		// Degrade silently per project convention.
		return nil, nil //nolint:nilerr
	}
	if info == nil {
		return nil, nil
	}

	// Strip leading "v" from both versions before comparison.
	latest := stripV(info.version)
	current := stripV(currentVersion)
	if latest == current || latest == "" {
		return nil, nil
	}

	// Simple lexicographic semver comparison. go-selfupdate already uses
	// semver-aware detection so latest will only be populated when truly newer.
	if latest <= current {
		return nil, nil
	}

	return &UpdateInfo{
		Version:    latest,
		ReleaseURL: info.url,
	}, nil
}

// PerformUpdate downloads and installs the latest release of claudehopper.
// It detects whether the binary is a source install (in GOPATH/bin) or a
// binary install and uses the appropriate upgrade strategy:
//   - Source install: runs `go install github.com/matthewfritsch/claudehopper@vX.Y.Z`
//   - Binary install: uses go-selfupdate to replace the binary in-place
func PerformUpdate(ctx context.Context, currentVersion string) error {
	updater, err := selfupdate.NewUpdater(selfupdate.Config{})
	if err != nil {
		return fmt.Errorf("create updater: %w", err)
	}

	release, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug(repoSlug))
	if err != nil {
		return fmt.Errorf("detect latest: %w", err)
	}
	if !found {
		fmt.Println("Already at the latest version.")
		return nil
	}

	latest := release.Version()
	current := stripV(currentVersion)
	if stripV(latest) == current {
		fmt.Printf("Already at the latest version (%s).\n", current)
		return nil
	}

	fmt.Printf("Updating claudehopper %s -> %s...\n", current, latest)

	if isSourceInstall() {
		ref := "github.com/matthewfritsch/claudehopper@v" + stripV(latest)
		cmd := exec.CommandContext(ctx, "go", "install", ref)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go install %s: %w", ref, err)
		}
		fmt.Printf("Updated to %s via go install.\n", latest)
		return nil
	}

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	if err := updater.UpdateTo(ctx, release, exe); err != nil {
		return fmt.Errorf("update binary: %w", err)
	}
	fmt.Printf("Updated to %s.\n", latest)
	return nil
}

// isSourceInstall returns true when the running executable lives under
// GOPATH/bin, which indicates it was installed via `go install` rather than
// downloaded as a pre-built binary release.
func isSourceInstall() bool {
	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return false
	}

	// Determine GOPATH via `go env GOPATH`.
	out, err := exec.Command("go", "env", "GOPATH").Output()
	if err != nil {
		return false
	}
	gopath := strings.TrimSpace(string(out))
	if gopath == "" {
		return false
	}
	return strings.HasPrefix(exe, filepath.Join(gopath, "bin"))
}

// writeStamp creates or truncates the stamp file at path, updating its ModTime.
func writeStamp(path string) {
	_ = os.WriteFile(path, []byte{}, 0644)
}

// stripV removes a leading "v" from a version string (e.g. "v1.2.3" -> "1.2.3").
func stripV(v string) string {
	if len(v) > 0 && v[0] == 'v' {
		return v[1:]
	}
	return v
}
