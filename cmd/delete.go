package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/matthewfritsch/claudehopper/internal/config"
	"github.com/matthewfritsch/claudehopper/internal/profile"
	"github.com/matthewfritsch/claudehopper/internal/usage"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete NAME",
	Short: "Delete a profile",
	Long: `Delete a claudehopper profile by name.

If other profiles depend on the target profile (via shared paths or lineage),
a warning is shown and confirmation is required unless --yes is provided.`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

var deleteYes bool

func init() {
	deleteCmd.Flags().BoolVar(&deleteYes, "yes", false, "Skip confirmation prompt")
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(_ *cobra.Command, args []string) error {
	name := profile.NormalizeProfileName(args[0])

	profilesDir, err := config.ProfilesDir()
	if err != nil {
		return fmt.Errorf("resolve profiles dir: %w", err)
	}

	configPath, err := config.ConfigFilePath()
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	err = profile.DeleteProfile(profilesDir, name, cfg.Active)
	if err == nil {
		cfgDir, _ := config.ConfigDir()
		usage.RecordUsage(cfgDir, name, "delete")
		fmt.Printf("Deleted profile %q\n", name)
		return nil
	}

	// Check if it's a DependentError
	depErr, ok := err.(*profile.DependentError)
	if !ok {
		return err
	}

	// DependentError: show warning and prompt
	fmt.Fprintf(os.Stderr, "Warning: profile %q has dependent profiles:\n", name)
	for _, dep := range depErr.Dependents {
		fmt.Fprintf(os.Stderr, "  - %s\n", dep)
	}

	if deleteYes {
		// --yes flag bypasses prompt — force delete by removing directory directly
		return forceDelete(profilesDir, name)
	}

	if !isInteractive() {
		// Non-TTY: abort silently for scripting safety
		return fmt.Errorf("aborting: profile %q has dependents (use --yes to force)", name)
	}

	// Interactive TTY: prompt user
	fmt.Fprintf(os.Stderr, "Delete anyway? [y/N] ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer == "y" || answer == "yes" {
			return forceDelete(profilesDir, name)
		}
	}
	fmt.Println("Aborted.")
	return nil
}

// forceDelete removes the profile directory regardless of dependents.
func forceDelete(profilesDir, name string) error {
	profilePath := filepath.Join(profilesDir, name)
	if err := os.RemoveAll(profilePath); err != nil {
		return fmt.Errorf("remove profile %q: %w", name, err)
	}
	cfgDir, _ := config.ConfigDir()
	usage.RecordUsage(cfgDir, name, "delete")
	fmt.Printf("Deleted profile %q\n", name)
	return nil
}
