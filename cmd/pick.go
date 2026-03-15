package cmd

import (
	"fmt"

	"github.com/matthewfritsch/claudehopper/internal/config"
	"github.com/matthewfritsch/claudehopper/internal/profile"
	"github.com/spf13/cobra"
)

var pickCmd = &cobra.Command{
	Use:   "pick FILE... --from SOURCE [--to TARGET]",
	Short: "Copy files from one profile to another as independent copies",
	Long: `Copy files from a source profile into a target profile as independent files.

Unlike share, the target profile gets its own copy of the files and is not
linked to the source. Symlinks in the source are preserved as symlinks in
the target.

Use --dry-run to preview changes without writing anything.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runPick,
}

var (
	pickFrom   string
	pickTo     string
	pickDryRun bool
)

func init() {
	pickCmd.Flags().StringVar(&pickFrom, "from", "", "Source profile name (required)")
	pickCmd.Flags().StringVar(&pickTo, "to", "", "Target profile name (defaults to active profile)")
	pickCmd.Flags().BoolVar(&pickDryRun, "dry-run", false, "Preview changes without writing")
	_ = pickCmd.MarkFlagRequired("from")
	rootCmd.AddCommand(pickCmd)
}

func runPick(_ *cobra.Command, args []string) error {
	profilesDir, err := config.ProfilesDir()
	if err != nil {
		return fmt.Errorf("resolve profiles dir: %w", err)
	}

	configPath, err := config.ConfigFilePath()
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}

	from := profile.NormalizeProfileName(pickFrom)

	to := pickTo
	if to == "" {
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if cfg.Active == "" {
			return fmt.Errorf("no active profile — specify --to")
		}
		to = cfg.Active
	}
	to = profile.NormalizeProfileName(to)

	picked, err := profile.PickFiles(profilesDir, from, to, args, pickDryRun)
	if err != nil {
		return err
	}

	if pickDryRun {
		fmt.Printf("Would pick %d file(s) from %q to %q:\n", len(picked), from, to)
		for _, p := range picked {
			fmt.Printf("  %s\n", p)
		}
		return nil
	}

	fmt.Printf("Picked %d file(s) from %q to %q\n", len(picked), from, to)

	// Re-link active profile if target is active
	cfg, err := config.LoadConfig(configPath)
	if err == nil && cfg.Active == to {
		sharedDir, err := profile.SharedDir()
		if err != nil {
			return fmt.Errorf("resolve shared dir: %w", err)
		}
		cDir := claudeDir()
		opts := profile.SwitchOptions{Force: true}
		if _, err := profile.DoSwitch(profilesDir, cDir, configPath, sharedDir, to, to, opts); err != nil {
			return fmt.Errorf("re-link active profile: %w", err)
		}
	}

	return nil
}
