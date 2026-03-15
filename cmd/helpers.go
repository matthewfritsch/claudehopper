package cmd

import (
	"os"
	"path/filepath"
)

// isInteractive returns true when stdin is connected to a terminal (TTY).
// This is used to gate interactive prompts — piped/scripted invocations skip
// prompts silently so automation never blocks.
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// claudeDir returns the path to the ~/.claude directory, which is the
// managed directory that claudehopper creates symlinks in.
func claudeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback if $HOME is unset
		return filepath.Join(os.Getenv("HOME"), ".claude")
	}
	return filepath.Join(home, ".claude")
}
