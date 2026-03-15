package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/matthewfritsch/claudehopper/internal/config"
	"github.com/matthewfritsch/claudehopper/internal/profile"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the active profile's link health",
	Long:  `Show the active profile name and the health of each managed symlink.`,
	Args:  cobra.NoArgs,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(_ *cobra.Command, _ []string) error {
	configPath, err := config.ConfigFilePath()
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.Active == "" {
		fmt.Println("No active profile")
		return nil
	}

	profilesDir, err := config.ProfilesDir()
	if err != nil {
		return fmt.Errorf("resolve profiles dir: %w", err)
	}

	profileDir := filepath.Join(profilesDir, cfg.Active)
	manifestPath := filepath.Join(profileDir, ".hop-manifest.json")
	m, err := config.LoadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("load manifest for %q: %w", cfg.Active, err)
	}

	sharedDir, err := profile.SharedDir()
	if err != nil {
		return fmt.Errorf("resolve shared dir: %w", err)
	}

	info := profile.GetProfileStatus(profileDir, claudeDir(), sharedDir, m)
	fmt.Print(profile.FormatProfileStatus(info))
	return nil
}
