package cmd

import (
	"fmt"

	"github.com/matthewfritsch/claudehopper/internal/config"
	"github.com/matthewfritsch/claudehopper/internal/profile"
	"github.com/spf13/cobra"
)

var unshareCmd = &cobra.Command{
	Use:   "unshare [FILE...]",
	Short: "Materialize shared symlinks back to independent file copies",
	Long: `Replace shared symlinks in a profile with real file copies.

If no files are specified, all shared paths in the profile are materialized.
After unsharing, the files are owned independently by the profile and changes
to the original source profile will not affect this profile.

Use --dry-run to preview changes without writing anything.`,
	Args: cobra.ArbitraryArgs,
	RunE: runUnshare,
}

var (
	unshareProfile string
	unshareDryRun  bool
)

func init() {
	unshareCmd.Flags().StringVar(&unshareProfile, "profile", "", "Profile to unshare from (defaults to active profile)")
	unshareCmd.Flags().BoolVar(&unshareDryRun, "dry-run", false, "Preview changes without writing")
	rootCmd.AddCommand(unshareCmd)
}

func runUnshare(_ *cobra.Command, args []string) error {
	profilesDir, err := config.ProfilesDir()
	if err != nil {
		return fmt.Errorf("resolve profiles dir: %w", err)
	}

	configPath, err := config.ConfigFilePath()
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}

	profileName := unshareProfile
	if profileName == "" {
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if cfg.Active == "" {
			return fmt.Errorf("no active profile — specify --profile")
		}
		profileName = cfg.Active
	}
	profileName = profile.NormalizeProfileName(profileName)

	unshared, err := profile.UnshareFiles(profilesDir, profileName, args, unshareDryRun)
	if err != nil {
		return err
	}

	if unshareDryRun {
		fmt.Printf("Would unshare %d file(s) in %q:\n", len(unshared), profileName)
		for _, p := range unshared {
			fmt.Printf("  %s\n", p)
		}
		return nil
	}

	fmt.Printf("Unshared %d file(s) in %q\n", len(unshared), profileName)

	// Re-link active profile if target is active
	cfg, err := config.LoadConfig(configPath)
	if err == nil && cfg.Active == profileName {
		sharedDir, err := profile.SharedDir()
		if err != nil {
			return fmt.Errorf("resolve shared dir: %w", err)
		}
		cDir := claudeDir()
		opts := profile.SwitchOptions{Force: true}
		if _, err := profile.DoSwitch(profilesDir, cDir, configPath, sharedDir, profileName, profileName, opts); err != nil {
			return fmt.Errorf("re-link active profile: %w", err)
		}
	}

	return nil
}
