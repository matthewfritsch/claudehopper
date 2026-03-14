package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "claudehopper",
	Short: "Switch Claude Code configuration profiles",
	Long: `claudehopper manages multiple Claude Code configuration profiles,
allowing you to instantly switch between different setups — each with
its own API keys, MCP servers, and settings.

Use 'hop' as a convenient alias for 'claudehopper'.`,
}

// SetVersionInfo sets the version string on the root command using the provided
// version, commit hash, and build date. This is called from main() with values
// injected via ldflags at build time.
func SetVersionInfo(version, commit, date string) {
	rootCmd.Version = fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)
}

// Execute runs the root command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}
