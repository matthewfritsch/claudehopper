// Package fs provides safety-critical filesystem primitives for claudehopper.
package fs

import "github.com/google/renameio/v2"

// AtomicSymlink atomically creates or replaces a symlink at linkPath pointing
// to targetPath. It never leaves the filesystem in a broken or inconsistent
// state: the link is either the old value or the new value, never missing.
//
// This is achieved via renameio which creates a temporary symlink and
// renames it into place. rename(2) is atomic on POSIX systems.
//
// Note: Not supported on Windows — renameio/v2 does not export Symlink on
// that platform. This is an accepted limitation; claudehopper targets
// macOS and Linux only.
func AtomicSymlink(targetPath, linkPath string) error {
	return renameio.Symlink(targetPath, linkPath)
}
