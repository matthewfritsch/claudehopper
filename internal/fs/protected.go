package fs

import "sort"

// sharedPaths is the set of paths that are shared across all Claude Code
// profiles. These paths are never touched during profile switches because
// they contain credentials, history, and shared data that belongs to the user
// rather than to any specific profile.
//
// If you add or remove entries here, update testdata/shared_paths.txt
// to match — TestIsProtected_MatchesFixture verifies they stay in sync.
var sharedPaths = map[string]struct{}{
	".credentials.json":  {},
	"history.jsonl":      {},
	"projects":           {},
	"cache":              {},
	"downloads":          {},
	"transcripts":        {},
	"shell-snapshots":    {},
	"file-history":       {},
	"backups":            {},
	"session-env":        {},
	".session-stats.json": {},
}

// IsProtected reports whether name is a shared path that must never be
// modified during profile switches. Only bare names (no path separators) are
// checked; names with slashes are always treated as unprotected because the
// protected set contains only top-level entries.
func IsProtected(name string) bool {
	if name == "" {
		return false
	}
	// Reject anything that looks like a path component — protected names are
	// bare (no directory separators).
	for _, c := range name {
		if c == '/' || c == '\\' {
			return false
		}
	}
	_, ok := sharedPaths[name]
	return ok
}

// ProtectedPaths returns a sorted slice of all protected path names.
// Useful for display, debugging, and drift-detection tests.
func ProtectedPaths() []string {
	names := make([]string, 0, len(sharedPaths))
	for name := range sharedPaths {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
