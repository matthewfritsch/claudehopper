package cmd

import (
	"fmt"

	"github.com/matthewfritsch/claudehopper/internal/config"
	"github.com/matthewfritsch/claudehopper/internal/profile"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Long:  `List all claudehopper profiles, showing which is currently active.`,
	Args:  cobra.NoArgs,
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(_ *cobra.Command, _ []string) error {
	profilesDir, err := config.ProfilesDir()
	if err != nil {
		return fmt.Errorf("resolve profiles dir: %w", err)
	}

	configPath, err := config.ConfigFilePath()
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}

	summaries, err := profile.ListProfiles(profilesDir, configPath)
	if err != nil {
		return fmt.Errorf("list profiles: %w", err)
	}

	if len(summaries) == 0 {
		fmt.Println("No profiles found. Create one with: hop create NAME")
		return nil
	}

	fmt.Print(profile.FormatProfileList(summaries))
	return nil
}
