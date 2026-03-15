package cmd

import (
	"context"
	"fmt"

	"github.com/matthewfritsch/claudehopper/internal/config"
	"github.com/matthewfritsch/claudehopper/internal/updater"
	"github.com/spf13/cobra"
)

var updateCheckOnly bool

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update claudehopper to the latest version",
	Long: `Update claudehopper to the latest version from GitHub Releases.

With --check, only checks for updates without installing.

For source installs (installed via 'go install'), this runs 'go install' with
the latest version tag. For binary installs, this replaces the current binary
in-place with the downloaded release asset.`,
	Args: cobra.NoArgs,
	RunE: runUpdate,
}

func init() {
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check", false, "Check for updates without installing")
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(_ *cobra.Command, _ []string) error {
	if updateCheckOnly {
		configDir, err := config.ConfigDir()
		if err != nil {
			return err
		}
		info, err := updater.CheckForUpdate(context.Background(), configDir, Version)
		if err != nil {
			return err
		}
		if info == nil {
			fmt.Printf("claudehopper %s is up to date.\n", Version)
			return nil
		}
		fmt.Printf("Update available: %s → %s\n", Version, info.Version)
		fmt.Println("Run 'hop update' to install.")
		return nil
	}
	return updater.PerformUpdate(context.Background(), Version)
}
