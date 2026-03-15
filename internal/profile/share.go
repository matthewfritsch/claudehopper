package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/google/renameio/v2"
	"github.com/matthewfritsch/claudehopper/internal/config"
)

// ShareFiles creates a symlink in the target profile directory pointing to the
// source profile's file. The symlink target is resolved to avoid chained
// symlinks. The target manifest's shared_paths and managed_paths are updated
// and saved. If dryRun is true, only the list of paths that would be shared is
// returned without any filesystem or manifest changes.
//
// Returns the list of paths that were (or would be) shared.
func ShareFiles(profilesDir, srcName, tgtName string, paths []string, dryRun bool) ([]string, error) {
	srcDir := filepath.Join(profilesDir, srcName)
	tgtDir := filepath.Join(profilesDir, tgtName)

	if dryRun {
		return append([]string{}, paths...), nil
	}

	tgtManifestPath := filepath.Join(tgtDir, ".hop-manifest.json")
	tgtManifest, err := config.LoadManifest(tgtManifestPath)
	if err != nil {
		return nil, fmt.Errorf("load target manifest: %w", err)
	}

	var shared []string
	for _, p := range paths {
		src := filepath.Join(srcDir, p)
		dst := filepath.Join(tgtDir, p)

		// Resolve real target to avoid chained symlinks.
		// If src is itself a symlink, follow it; otherwise use src as-is.
		realTarget := src
		if resolved, err := filepath.EvalSymlinks(src); err == nil {
			realTarget = resolved
		}

		if err := renameio.Symlink(realTarget, dst); err != nil {
			return shared, fmt.Errorf("create symlink for %q: %w", p, err)
		}

		tgtManifest.SharedPaths[p] = srcName

		// Add to managed_paths if not already present (no duplicates)
		found := false
		for _, mp := range tgtManifest.ManagedPaths {
			if mp == p {
				found = true
				break
			}
		}
		if !found {
			tgtManifest.ManagedPaths = append(tgtManifest.ManagedPaths, p)
		}

		shared = append(shared, p)
	}

	if err := config.SaveManifest(tgtManifestPath, tgtManifest); err != nil {
		return shared, fmt.Errorf("save target manifest: %w", err)
	}

	return shared, nil
}

// PickFiles copies files from the source profile to the target profile.
// Regular files are copied byte-for-byte. Symlinks are reproduced as symlinks.
// Directories are copied recursively. The target manifest's managed_paths is
// updated. If dryRun is true, only the list of paths is returned without changes.
//
// Returns the list of paths that were (or would be) picked.
func PickFiles(profilesDir, srcName, tgtName string, paths []string, dryRun bool) ([]string, error) {
	srcDir := filepath.Join(profilesDir, srcName)
	tgtDir := filepath.Join(profilesDir, tgtName)

	if dryRun {
		return append([]string{}, paths...), nil
	}

	tgtManifestPath := filepath.Join(tgtDir, ".hop-manifest.json")
	tgtManifest, err := config.LoadManifest(tgtManifestPath)
	if err != nil {
		return nil, fmt.Errorf("load target manifest: %w", err)
	}

	var picked []string
	for _, p := range paths {
		src := filepath.Join(srcDir, p)
		dst := filepath.Join(tgtDir, p)

		fi, err := os.Lstat(src)
		if err != nil {
			return picked, fmt.Errorf("stat source %q: %w", p, err)
		}

		switch {
		case fi.Mode()&os.ModeSymlink != 0:
			// Preserve symlink: read target and create new symlink
			target, err := os.Readlink(src)
			if err != nil {
				return picked, fmt.Errorf("readlink %q: %w", p, err)
			}
			if err := os.Symlink(target, dst); err != nil {
				return picked, fmt.Errorf("symlink %q: %w", p, err)
			}
		case fi.IsDir():
			// Recursive directory copy
			if err := copyDirRecursive(src, dst); err != nil {
				return picked, fmt.Errorf("copy dir %q: %w", p, err)
			}
		default:
			// Regular file copy
			if err := copyFile(src, dst); err != nil {
				return picked, fmt.Errorf("copy file %q: %w", p, err)
			}
		}

		// Add to managed_paths without duplicates
		found := false
		for _, mp := range tgtManifest.ManagedPaths {
			if mp == p {
				found = true
				break
			}
		}
		if !found {
			tgtManifest.ManagedPaths = append(tgtManifest.ManagedPaths, p)
		}

		picked = append(picked, p)
	}

	if err := config.SaveManifest(tgtManifestPath, tgtManifest); err != nil {
		return picked, fmt.Errorf("save target manifest: %w", err)
	}

	return picked, nil
}

// UnshareFiles materializes shared symlinks in profileName back to independent
// file copies. For each path in shared_paths (or the specified subset if paths
// is non-empty): resolves the symlink target, replaces the symlink with a real
// copy, removes the path from shared_paths. If the symlink target no longer
// exists, the path is removed from shared_paths without materializing.
// If dryRun is true, only the list of paths is returned without changes.
//
// Returns the list of paths that were (or would be) unshared.
func UnshareFiles(profilesDir, profileName string, paths []string, dryRun bool) ([]string, error) {
	profileDir := filepath.Join(profilesDir, profileName)
	manifestPath := filepath.Join(profileDir, ".hop-manifest.json")

	m, err := config.LoadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("load manifest: %w", err)
	}

	// Determine which paths to unshare.
	var toUnshare []string
	if len(paths) == 0 {
		// All shared paths
		for p := range m.SharedPaths {
			toUnshare = append(toUnshare, p)
		}
		sort.Strings(toUnshare) // deterministic order
	} else {
		toUnshare = paths
	}

	if dryRun {
		return append([]string{}, toUnshare...), nil
	}

	var unshared []string
	for _, p := range toUnshare {
		linkPath := filepath.Join(profileDir, p)

		// Attempt to follow symlink to real content.
		realPath, err := filepath.EvalSymlinks(linkPath)
		if err != nil {
			// Target missing — still remove from shared_paths but skip copy.
			delete(m.SharedPaths, p)
			unshared = append(unshared, p)
			continue
		}

		// Remove the symlink then copy the real content.
		if err := os.Remove(linkPath); err != nil {
			return unshared, fmt.Errorf("remove symlink %q: %w", p, err)
		}

		realFi, err := os.Stat(realPath)
		if err != nil {
			// Real target disappeared between EvalSymlinks and here.
			delete(m.SharedPaths, p)
			unshared = append(unshared, p)
			continue
		}

		if realFi.IsDir() {
			if err := copyDirRecursive(realPath, linkPath); err != nil {
				return unshared, fmt.Errorf("copy dir %q: %w", p, err)
			}
		} else {
			if err := copyFile(realPath, linkPath); err != nil {
				return unshared, fmt.Errorf("copy file %q: %w", p, err)
			}
		}

		delete(m.SharedPaths, p)
		unshared = append(unshared, p)
	}

	if err := config.SaveManifest(manifestPath, m); err != nil {
		return unshared, fmt.Errorf("save manifest: %w", err)
	}

	return unshared, nil
}

