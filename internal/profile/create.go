package profile

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/matthewfritsch/claudehopper/internal/config"
	"github.com/matthewfritsch/claudehopper/internal/fs"
)

// CreateBlank creates a new empty profile with a settings.json placeholder
// and a manifest. It accepts explicit directory paths so it can be called
// from tests using t.TempDir() without touching real config dirs.
//
// The profile name is normalized (trimmed + lowercased) before use.
// Returns an error if the profile already exists.
func CreateBlank(profilesDir, sharedDir, name, description string) error {
	name = NormalizeProfileName(name)
	if err := ValidateProfileName(name); err != nil {
		return err
	}

	profileDir := filepath.Join(profilesDir, name)
	if _, err := os.Stat(profileDir); err == nil {
		return fmt.Errorf("profile %q already exists", name)
	}
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}

	// Write empty settings.json placeholder
	settingsPath := filepath.Join(profileDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte("{}\n"), 0644); err != nil {
		return fmt.Errorf("write settings.json: %w", err)
	}

	// Bootstrap shared defaults
	sharedM, err := LinkDefaultsIntoProfile(profileDir, sharedDir, "")
	if err != nil {
		return fmt.Errorf("link defaults: %w", err)
	}

	// Build manifest
	m := config.NewManifest(description)
	m.ManagedPaths = []string{"settings.json"}
	// Merge shared_paths from LinkDefaultsIntoProfile
	for k, v := range sharedM.SharedPaths {
		m.SharedPaths[k] = v
	}

	manifestPath := filepath.Join(profileDir, ".hop-manifest.json")
	if err := config.SaveManifest(manifestPath, m); err != nil {
		return fmt.Errorf("save manifest: %w", err)
	}
	return nil
}

// CreateFromCurrent captures the current ~/.claude/ directory contents into
// a new profile. It accepts explicit paths for testing.
//
// Files are filtered:
//   - IsProtected names (credentials, history, etc.) are skipped
//   - Names with .hop- prefix are skipped
//   - Names with .ccswap prefix are skipped
//
// Symlinks are preserved as symlinks and recorded in shared_paths with value
// "(shared)". Regular files/dirs are copied into the profile dir and added
// to managed_paths.
func CreateFromCurrent(profilesDir, claudeDir, sharedDir, name, description string) error {
	name = NormalizeProfileName(name)
	if err := ValidateProfileName(name); err != nil {
		return err
	}

	profileDir := filepath.Join(profilesDir, name)
	if _, err := os.Stat(profileDir); err == nil {
		return fmt.Errorf("profile %q already exists", name)
	}
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}

	m := config.NewManifest(description)

	entries, err := os.ReadDir(claudeDir)
	if err != nil {
		return fmt.Errorf("read claude dir: %w", err)
	}

	for _, entry := range entries {
		entryName := entry.Name()

		// Skip protected, .hop- prefixed, and .ccswap prefixed files
		if fs.IsProtected(entryName) {
			continue
		}
		if strings.HasPrefix(entryName, ".hop-") {
			continue
		}
		if strings.HasPrefix(entryName, ".ccswap") {
			continue
		}

		srcPath := filepath.Join(claudeDir, entryName)
		dstPath := filepath.Join(profileDir, entryName)

		// Use Lstat to detect symlinks without following them
		fi, err := os.Lstat(srcPath)
		if err != nil {
			continue
		}

		if fi.Mode()&os.ModeSymlink != 0 {
			// Preserve symlink as symlink
			target, err := os.Readlink(srcPath)
			if err != nil {
				return fmt.Errorf("readlink %s: %w", srcPath, err)
			}
			if err := os.Symlink(target, dstPath); err != nil {
				return fmt.Errorf("symlink %s: %w", entryName, err)
			}
			m.SharedPaths[entryName] = "(shared)"
		} else {
			// Copy regular file/dir
			if fi.IsDir() {
				if err := copyDirRecursive(srcPath, dstPath); err != nil {
					return fmt.Errorf("copy dir %s: %w", entryName, err)
				}
			} else {
				if err := copyFile(srcPath, dstPath); err != nil {
					return fmt.Errorf("copy file %s: %w", entryName, err)
				}
			}
			m.ManagedPaths = append(m.ManagedPaths, entryName)
		}
	}

	// Bootstrap shared defaults (seed from newly created profile dir)
	sharedM, err := LinkDefaultsIntoProfile(profileDir, sharedDir, profileDir)
	if err != nil {
		return fmt.Errorf("link defaults: %w", err)
	}
	for k, v := range sharedM.SharedPaths {
		m.SharedPaths[k] = v
	}

	manifestPath := filepath.Join(profileDir, ".hop-manifest.json")
	return config.SaveManifest(manifestPath, m)
}

// CreateFromProfile copies an existing profile into a new one, recording
// the lineage via the created_from field in the manifest.
//
// Symlinks in the source profile are preserved as symlinks.
func CreateFromProfile(profilesDir, sharedDir, sourceName, newName, description string) error {
	newName = NormalizeProfileName(newName)
	if err := ValidateProfileName(newName); err != nil {
		return err
	}

	sourceDir := filepath.Join(profilesDir, sourceName)
	if _, err := os.Stat(sourceDir); err != nil {
		return fmt.Errorf("source profile %q not found: %w", sourceName, err)
	}

	newDir := filepath.Join(profilesDir, newName)
	if _, err := os.Stat(newDir); err == nil {
		return fmt.Errorf("profile %q already exists", newName)
	}
	if err := os.MkdirAll(newDir, 0755); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}

	// Copy source dir contents (not the dir itself) preserving symlinks
	if err := copyDirContents(sourceDir, newDir); err != nil {
		return fmt.Errorf("copy source profile: %w", err)
	}

	// Load source manifest and update for the new profile
	srcManifestPath := filepath.Join(sourceDir, ".hop-manifest.json")
	srcM, err := config.LoadManifest(srcManifestPath)
	if err != nil {
		// Source may not have a manifest — start fresh
		srcM = config.NewManifest("")
	}
	srcM.CreatedFrom = sourceName
	if description != "" {
		srcM.Description = description
	}

	// Bootstrap shared defaults
	sharedM, err := LinkDefaultsIntoProfile(newDir, sharedDir, newDir)
	if err != nil {
		return fmt.Errorf("link defaults: %w", err)
	}
	for k, v := range sharedM.SharedPaths {
		srcM.SharedPaths[k] = v
	}

	newManifestPath := filepath.Join(newDir, ".hop-manifest.json")
	return config.SaveManifest(newManifestPath, srcM)
}

// copyDirContents copies all entries from srcDir into dstDir without creating
// a nested directory. Symlinks are preserved as symlinks.
func copyDirContents(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		src := filepath.Join(srcDir, entry.Name())
		dst := filepath.Join(dstDir, entry.Name())

		fi, err := os.Lstat(src)
		if err != nil {
			return err
		}

		switch {
		case fi.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(src)
			if err != nil {
				return err
			}
			if err := os.Symlink(target, dst); err != nil {
				return err
			}
		case fi.IsDir():
			if err := os.MkdirAll(dst, fi.Mode()); err != nil {
				return err
			}
			if err := copyDirContents(src, dst); err != nil {
				return err
			}
		default:
			if err := copyFile(src, dst); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyDirRecursive copies srcDir to dstDir as a new directory.
func copyDirRecursive(srcDir, dstDir string) error {
	fi, err := os.Stat(srcDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dstDir, fi.Mode()); err != nil {
		return err
	}
	return copyDirContents(srcDir, dstDir)
}

// copyFileWithPerm is an alias kept for clarity — same as copyFile.
func copyFileWithPerm(src, dst string) error {
	return copyFile(src, dst)
}

// copyFileIO copies src to dst using io.Copy (fallback if copyFile not used).
func copyFileIO(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
