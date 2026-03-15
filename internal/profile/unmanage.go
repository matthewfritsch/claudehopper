package profile

import (
	"os"
	"path/filepath"

	"github.com/matthewfritsch/claudehopper/internal/config"
	fs "github.com/matthewfritsch/claudehopper/internal/fs"
)

// UnmanageActive materializes all symlinks in claudeDir back to real files or
// directories, then clears the active profile in config.json. This is the
// "exit ramp" for claudehopper — after calling this, claudeDir is self-contained
// and no longer managed by claudehopper.
//
// If dryRun is true, the list of paths that would be materialized is returned
// without any filesystem changes and without clearing the config.
//
// Protected paths (credentials, history, etc.) are always skipped.
// Non-symlink entries are skipped — they are already real files.
//
// Returns the list of materialized paths.
func UnmanageActive(claudeDir, configPath string, dryRun bool) ([]string, error) {
	entries, err := os.ReadDir(claudeDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Empty dir — nothing to do, still clear config
			if !dryRun {
				if err := config.SaveConfig(configPath, config.Config{Active: ""}); err != nil {
					return nil, err
				}
			}
			return []string{}, nil
		}
		return nil, err
	}

	var materialized []string

	for _, entry := range entries {
		name := entry.Name()

		// Skip protected paths
		if fs.IsProtected(name) {
			continue
		}

		// Check if this is a symlink using os.Lstat
		fi, err := os.Lstat(filepath.Join(claudeDir, name))
		if err != nil {
			continue // skip on error
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			// Not a symlink — already a real file, skip
			continue
		}

		linkPath := filepath.Join(claudeDir, name)
		materialized = append(materialized, linkPath)

		if dryRun {
			continue
		}

		// Resolve the symlink target (follows all levels)
		realPath, err := filepath.EvalSymlinks(linkPath)
		if err != nil {
			// Dangling symlink — skip materialization but still track it
			continue
		}

		realFi, err := os.Stat(realPath)
		if err != nil {
			continue
		}

		// Remove the symlink
		if err := os.Remove(linkPath); err != nil {
			return materialized, err
		}

		if realFi.IsDir() {
			if err := copyDirRecursive(realPath, linkPath); err != nil {
				return materialized, err
			}
		} else {
			if err := copyFile(realPath, linkPath); err != nil {
				return materialized, err
			}
		}
	}

	if !dryRun {
		if err := config.SaveConfig(configPath, config.Config{Active: ""}); err != nil {
			return materialized, err
		}
	}

	if materialized == nil {
		materialized = []string{}
	}
	return materialized, nil
}
