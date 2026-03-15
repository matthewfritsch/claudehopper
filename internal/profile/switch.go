package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/matthewfritsch/claudehopper/internal/config"
	fs "github.com/matthewfritsch/claudehopper/internal/fs"
)

// SwitchAction describes a single planned or performed action during a profile switch.
// Action is one of: "link", "unlink", "backup", "orphan", "skip".
type SwitchAction struct {
	Action string
	Path   string
	Detail string
}

// SwitchOptions controls the behaviour of DoSwitch.
type SwitchOptions struct {
	DryRun     bool
	Force      bool
	AdoptFiles []string // populated by the CLI layer after prompting the user
}

// SwitchResult summarises what DoSwitch did.
type SwitchResult struct {
	Actions []SwitchAction
	BackedUp []string
	Adopted  []string
}

// ValidatePreflight checks that every managed path listed in manifest exists
// inside profileDir. It returns a list of planned SwitchActions on success, or
// an error listing every missing path.
//
// claudeDir is inspected to determine whether each managed path will need a
// plain link, an unlink-then-link, or a backup-then-link.
func ValidatePreflight(profileDir, claudeDir string, manifest config.Manifest) ([]SwitchAction, error) {
	var missing []string
	for _, name := range manifest.ManagedPaths {
		if _, err := os.Lstat(filepath.Join(profileDir, name)); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("target profile is missing managed paths: %s", strings.Join(missing, ", "))
	}

	// Build planned actions
	var actions []SwitchAction
	for _, name := range manifest.ManagedPaths {
		claudePath := filepath.Join(claudeDir, name)
		fi, err := os.Lstat(claudePath)
		if err != nil {
			// Does not exist yet — plain link
			actions = append(actions, SwitchAction{Action: "link", Path: name})
			continue
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			// Existing symlink — unlink then re-link
			actions = append(actions, SwitchAction{Action: "link", Path: name, Detail: "replace symlink"})
		} else {
			// Real file/dir — backup then link
			actions = append(actions, SwitchAction{Action: "backup", Path: name})
			actions = append(actions, SwitchAction{Action: "link", Path: name})
		}
	}

	return actions, nil
}

// backupPath returns a unique backup destination for path by appending
// ".hop-backup", ".hop-backup.1", ".hop-backup.2", etc.  It uses os.Lstat so
// that dangling symlinks are treated as existing.
func backupPath(path string) string {
	candidate := path + ".hop-backup"
	if _, err := os.Lstat(candidate); err != nil {
		// Does not exist (or unreadable) — use it
		return candidate
	}
	for i := 1; ; i++ {
		candidate = fmt.Sprintf("%s.hop-backup.%d", path, i)
		if _, err := os.Lstat(candidate); err != nil {
			return candidate
		}
	}
}

// linkManagedPath creates (or replaces) a symlink at claudeDir/name pointing to
// the appropriate target derived from profileDir/name.
//
//   - If profileDir/name is itself a symlink, the link target is re-used
//     (preserving shared-dir indirection).
//   - Otherwise the target is the absolute path of profileDir/name.
//
// If a real file or directory already exists at claudeDir/name it is moved to
// the path returned by backupPath and backedUp is set to true.
// An existing wrong symlink at claudeDir/name is removed silently.
func linkManagedPath(profileDir, claudeDir, name string) (backedUp bool, err error) {
	claudePath := filepath.Join(claudeDir, name)
	profilePath := filepath.Join(profileDir, name)

	// Check what exists at claudeDir/name
	fi, err := os.Lstat(claudePath)
	if err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			// Existing symlink — just remove it
			if err := os.Remove(claudePath); err != nil {
				return false, fmt.Errorf("remove existing symlink %s: %w", claudePath, err)
			}
		} else {
			// Real file or directory — back it up
			dest := backupPath(claudePath)
			if err := os.Rename(claudePath, dest); err != nil {
				return false, fmt.Errorf("backup %s -> %s: %w", claudePath, dest, err)
			}
			backedUp = true
		}
	}

	// Determine the symlink target
	var target string
	profileFi, err := os.Lstat(profilePath)
	if err != nil {
		return false, fmt.Errorf("stat profile path %s: %w", profilePath, err)
	}
	if profileFi.Mode()&os.ModeSymlink != 0 {
		// Profile entry is itself a symlink — preserve its target
		target, err = os.Readlink(profilePath)
		if err != nil {
			return false, fmt.Errorf("readlink %s: %w", profilePath, err)
		}
	} else {
		target, err = filepath.Abs(profilePath)
		if err != nil {
			return false, fmt.Errorf("abs %s: %w", profilePath, err)
		}
	}

	if err := fs.AtomicSymlink(target, claudePath); err != nil {
		return false, fmt.Errorf("symlink %s -> %s: %w", claudePath, target, err)
	}

	return backedUp, nil
}

// DetectUnmanaged returns names of entries in claudeDir that are NOT managed,
// filtered by the following exclusion rules:
//   - name is in managedPaths
//   - fs.IsProtected(name) is true
//   - name has prefix ".hop-"
//   - name has prefix ".ccswap"
//   - name contains ".hop-backup" (backup files / dirs)
//   - name is a symlink whose Readlink target starts with sharedDir
//
// The returned slice is sorted.
func DetectUnmanaged(claudeDir, sharedDir string, managedPaths []string) ([]string, error) {
	entries, err := os.ReadDir(claudeDir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", claudeDir, err)
	}

	managed := make(map[string]struct{}, len(managedPaths))
	for _, p := range managedPaths {
		managed[p] = struct{}{}
	}

	var unmanaged []string
	for _, entry := range entries {
		name := entry.Name()

		// Managed
		if _, ok := managed[name]; ok {
			continue
		}
		// Protected system paths
		if fs.IsProtected(name) {
			continue
		}
		// .hop- prefix
		if strings.HasPrefix(name, ".hop-") {
			continue
		}
		// .ccswap prefix
		if strings.HasPrefix(name, ".ccswap") {
			continue
		}
		// backup files/dirs
		if strings.Contains(name, ".hop-backup") {
			continue
		}

		// Symlink pointing into sharedDir — skip
		if entry.Type()&os.ModeSymlink != 0 {
			target, err := os.Readlink(filepath.Join(claudeDir, name))
			if err == nil && strings.HasPrefix(target, sharedDir) {
				continue
			}
		}

		unmanaged = append(unmanaged, name)
	}

	sort.Strings(unmanaged)
	return unmanaged, nil
}

// AdoptUnmanaged moves files from claudeDir into profileDir and appends them
// to the manifest's ManagedPaths, then saves the updated manifest.
// This records the "adopting" profile as the owner of those files.
func AdoptUnmanaged(claudeDir, profileDir string, manifest *config.Manifest, files []string) error {
	for _, name := range files {
		src := filepath.Join(claudeDir, name)
		dst := filepath.Join(profileDir, name)
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("adopt %s: %w", name, err)
		}
		manifest.ManagedPaths = append(manifest.ManagedPaths, name)
	}
	manifestPath := filepath.Join(profileDir, ".hop-manifest.json")
	return config.SaveManifest(manifestPath, *manifest)
}

// DoSwitch performs a full profile switch from currentName to targetName.
//
// Sequence:
//  1. Guard: already-active check (skip if Force).
//  2. Load target manifest.
//  3. Preflight validation.
//  4. DryRun: return planned actions, no writes.
//  5. If currentName != "": adopt requested files, unlink current managed paths.
//  6. Link target managed paths (with backup on conflicts).
//  7. Save config with new active name.
func DoSwitch(profilesDir, claudeDir, configPath, sharedDir, targetName, currentName string, opts SwitchOptions) (*SwitchResult, error) {
	if currentName == targetName && !opts.Force {
		return nil, fmt.Errorf("already on %q — use --force to re-link", targetName)
	}

	targetProfileDir := filepath.Join(profilesDir, targetName)
	targetManifest, err := config.LoadManifest(filepath.Join(targetProfileDir, ".hop-manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("load target manifest: %w", err)
	}

	actions, err := ValidatePreflight(targetProfileDir, claudeDir, targetManifest)
	if err != nil {
		return nil, err
	}

	if opts.DryRun {
		return &SwitchResult{Actions: actions}, nil
	}

	result := &SwitchResult{Actions: actions}

	// Handle departing profile
	if currentName != "" {
		currentProfileDir := filepath.Join(profilesDir, currentName)
		currentManifest, err := config.LoadManifest(filepath.Join(currentProfileDir, ".hop-manifest.json"))
		if err != nil {
			return nil, fmt.Errorf("load current manifest: %w", err)
		}

		// Adopt unmanaged files if requested
		if len(opts.AdoptFiles) > 0 {
			if err := AdoptUnmanaged(claudeDir, currentProfileDir, &currentManifest, opts.AdoptFiles); err != nil {
				return nil, fmt.Errorf("adopt unmanaged: %w", err)
			}
			result.Adopted = append(result.Adopted, opts.AdoptFiles...)
		}

		// Unlink current profile's managed paths (only remove symlinks)
		for _, name := range currentManifest.ManagedPaths {
			claudePath := filepath.Join(claudeDir, name)
			fi, err := os.Lstat(claudePath)
			if err != nil {
				continue // already gone
			}
			if fi.Mode()&os.ModeSymlink != 0 {
				_ = os.Remove(claudePath)
			}
		}
	}

	// Link target managed paths
	for _, name := range targetManifest.ManagedPaths {
		backedUp, err := linkManagedPath(targetProfileDir, claudeDir, name)
		if err != nil {
			return nil, fmt.Errorf("link %s: %w", name, err)
		}
		if backedUp {
			result.BackedUp = append(result.BackedUp, name)
		}
	}

	// Persist new active profile
	cfg := config.Config{Active: targetName}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	return result, nil
}
