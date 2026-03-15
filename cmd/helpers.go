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

// claudeDir returns the path to the Claude Code config directory.
// If CLAUDE_DIR is set, it is used as-is (for development/testing).
// Otherwise defaults to ~/.claude.
func claudeDir() string {
	if override := os.Getenv("CLAUDE_DIR"); override != "" {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.Getenv("HOME"), ".claude")
	}
	return filepath.Join(home, ".claude")
}
