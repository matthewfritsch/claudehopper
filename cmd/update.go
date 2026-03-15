package cmd

import (
	"context"

	"github.com/matthewfritsch/claudehopper/internal/updater"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update claudehopper to the latest version",
	Long: `Update claudehopper to the latest version from GitHub Releases.

For source installs (installed via 'go install'), this runs 'go install' with
the latest version tag. For binary installs, this replaces the current binary
in-place with the downloaded release asset.`,
	Args: cobra.NoArgs,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(_ *cobra.Command, _ []string) error {
	return updater.PerformUpdate(context.Background(), Version)
}
