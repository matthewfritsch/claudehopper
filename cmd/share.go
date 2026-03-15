package cmd

import (
	"fmt"

	"github.com/matthewfritsch/claudehopper/internal/config"
	"github.com/matthewfritsch/claudehopper/internal/profile"
	"github.com/spf13/cobra"
)

var shareCmd = &cobra.Command{
	Use:   "share FILE... --from SOURCE [--to TARGET]",
	Short: "Share files between profiles via symlinks",
	Long: `Share files from a source profile into a target profile as symlinks.

The target profile will have symlinks pointing to the real source files.
Symlink chains are avoided by resolving the source to its canonical path.

Use --dry-run to preview changes without writing anything.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runShare,
}

var (
	shareFrom   string
	shareTo     string
	shareDryRun bool
)

func init() {
	shareCmd.Flags().StringVar(&shareFrom, "from", "", "Source profile name (required)")
	shareCmd.Flags().StringVar(&shareTo, "to", "", "Target profile name (defaults to active profile)")
	shareCmd.Flags().BoolVar(&shareDryRun, "dry-run", false, "Preview changes without writing")
	_ = shareCmd.MarkFlagRequired("from")
	rootCmd.AddCommand(shareCmd)
}

func runShare(_ *cobra.Command, args []string) error {
	profilesDir, err := config.ProfilesDir()
	if err != nil {
		return fmt.Errorf("resolve profiles dir: %w", err)
	}

	configPath, err := config.ConfigFilePath()
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}

	from := profile.NormalizeProfileName(shareFrom)

	to := shareTo
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

	shared, err := profile.ShareFiles(profilesDir, from, to, args, shareDryRun)
	if err != nil {
		return err
	}

	if shareDryRun {
		fmt.Printf("Would share %d file(s) from %q to %q:\n", len(shared), from, to)
		for _, p := range shared {
			fmt.Printf("  %s\n", p)
		}
		return nil
	}

	fmt.Printf("Shared %d file(s) from %q to %q\n", len(shared), from, to)

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
