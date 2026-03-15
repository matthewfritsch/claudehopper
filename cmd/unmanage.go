package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/matthewfritsch/claudehopper/internal/config"
	"github.com/matthewfritsch/claudehopper/internal/profile"
	"github.com/spf13/cobra"
)

var unmanageCmd = &cobra.Command{
	Use:   "unmanage",
	Short: "Stop using claudehopper and materialize all symlinks",
	Long: `Replace all symlinks in ~/.claude/ with real file copies and deactivate claudehopper.

This operation materializes every symlink managed by claudehopper back to a real
file or directory, making ~/.claude/ fully self-contained again. The active profile
setting is cleared from config.json.

This is a one-way operation. Use --dry-run to preview what would be changed.`,
	Args: cobra.NoArgs,
	RunE: runUnmanage,
}

var unmanagedDryRun bool

func init() {
	unmanageCmd.Flags().BoolVar(&unmanagedDryRun, "dry-run", false, "Preview changes without writing")
	rootCmd.AddCommand(unmanageCmd)
}

func runUnmanage(_ *cobra.Command, _ []string) error {
	if !unmanagedDryRun && isInteractive() {
		fmt.Fprint(os.Stderr, "This will materialize all symlinks in ~/.claude/ and deactivate claudehopper. Continue? [y/N] ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if answer != "y" && answer != "yes" {
				fmt.Fprintln(os.Stderr, "Aborted.")
				return nil
			}
		}
	}

	configPath, err := config.ConfigFilePath()
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}

	cDir := claudeDir()

	materialized, err := profile.UnmanageActive(cDir, configPath, unmanagedDryRun)
	if err != nil {
		return err
	}

	if unmanagedDryRun {
		fmt.Printf("Would materialize:\n")
		for _, p := range materialized {
			fmt.Printf("  %s\n", p)
		}
		return nil
	}

	fmt.Printf("Materialized %d symlinks. claudehopper is now inactive.\n", len(materialized))
	return nil
}
