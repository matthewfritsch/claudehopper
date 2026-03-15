package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/matthewfritsch/claudehopper/internal/config"
	"github.com/matthewfritsch/claudehopper/internal/profile"
	"github.com/spf13/cobra"
)

var treeJSONFlag bool

var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Show profile lineage tree",
	Long: `Show all profiles in a lineage tree based on created_from relationships.

Profiles are shown with their managed path count, shared path count, and
an (active) marker for the currently active profile. Shared files are
annotated with their source profile.`,
	Args: cobra.NoArgs,
	RunE: runTree,
}

func init() {
	treeCmd.Flags().BoolVar(&treeJSONFlag, "json", false, "Output as JSON")
	rootCmd.AddCommand(treeCmd)
}

func runTree(_ *cobra.Command, _ []string) error {
	profilesDir, err := config.ProfilesDir()
	if err != nil {
		return fmt.Errorf("resolve profiles dir: %w", err)
	}

	configPath, err := config.ConfigFilePath()
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}

	roots, err := profile.BuildTree(profilesDir, configPath)
	if err != nil {
		return fmt.Errorf("build tree: %w", err)
	}

	if treeJSONFlag {
		// Get active profile name for JSON output
		cfg, _ := config.LoadConfig(configPath)
		data, err := profile.TreeJSON(roots, cfg.Active)
		if err != nil {
			return fmt.Errorf("serialize tree JSON: %w", err)
		}
		// Pretty print already handled by MarshalIndent, but ensure valid JSON
		var raw json.RawMessage = data
		fmt.Println(string(raw))
		return nil
	}

	fmt.Print(profile.RenderTree(roots))
	return nil
}
