package profile

import (
	"io"
	"os"
	"path/filepath"

	"github.com/matthewfritsch/claudehopper/internal/config"
)

// DefaultLinked contains the file names that are shared across profiles by
// default. This matches the Python DEFAULT_LINKED constant exactly.
var DefaultLinked = []string{
	"settings.json",
	"settings.local.json",
	".mcp.json",
}

// SharedDir returns the path to the shared directory inside the claudehopper
// config directory (e.g. ~/.config/claudehopper/shared on Linux).
func SharedDir() (string, error) {
	cfgDir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfgDir, "shared"), nil
}

// EnsureSharedDefaults creates the shared directory if it does not exist.
// It returns the absolute path to the shared directory.
func EnsureSharedDefaults() (string, error) {
	sharedDir, err := SharedDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		return "", err
	}
	return sharedDir, nil
}

// LinkDefaultsIntoProfile creates symlinks in profileDir for each
// DefaultLinked file. For each file:
//   - If it already exists in sharedDir: create symlink profileDir/file -> sharedDir/file
//   - If it does NOT exist in sharedDir but fromSource is non-empty and the file
//     exists in fromSource: copy fromSource/file to sharedDir/file, then symlink
//   - Otherwise: skip (no seed, blank profile)
//
// Returns an updated Manifest with SharedPaths entries for any linked files.
// The manifest's ManagedPaths and Description are left empty; callers may
// merge the returned SharedPaths into their own manifest.
func LinkDefaultsIntoProfile(profileDir, sharedDir, fromSource string) (config.Manifest, error) {
	m := config.NewManifest("")

	for _, name := range DefaultLinked {
		sharedFile := filepath.Join(sharedDir, name)
		linkTarget := sharedFile

		// Check if file exists in sharedDir
		if _, err := os.Stat(sharedFile); os.IsNotExist(err) {
			// Try to seed from fromSource
			if fromSource == "" {
				continue
			}
			srcFile := filepath.Join(fromSource, name)
			if _, err := os.Stat(srcFile); os.IsNotExist(err) {
				continue
			}
			// Copy srcFile to sharedDir
			if err := os.MkdirAll(sharedDir, 0755); err != nil {
				return m, err
			}
			if err := copyFile(srcFile, sharedFile); err != nil {
				return m, err
			}
		}

		// Create symlink in profileDir pointing to sharedFile
		linkPath := filepath.Join(profileDir, name)
		// Remove existing link/file if present (idempotent)
		_ = os.Remove(linkPath)
		if err := os.Symlink(linkTarget, linkPath); err != nil {
			return m, err
		}
		m.SharedPaths[name] = "(shared)"
	}

	return m, nil
}

// copyFile copies a regular file from src to dst preserving permissions.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	fi, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fi.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
