package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/matthewfritsch/claudehopper/internal/config"
	"github.com/matthewfritsch/claudehopper/internal/usage"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show profile usage statistics",
	Args:  cobra.NoArgs,
	RunE:  runStats,
}

var (
	statsJSON    bool
	statsSince   string
	statsProfile string
)

func init() {
	statsCmd.Flags().BoolVar(&statsJSON, "json", false, "Output statistics as JSON")
	statsCmd.Flags().StringVar(&statsSince, "since", "", "Filter entries on or after YYYY-MM-DD")
	statsCmd.Flags().StringVar(&statsProfile, "profile", "", "Filter entries for a specific profile name")
	rootCmd.AddCommand(statsCmd)
}

func runStats(_ *cobra.Command, _ []string) error {
	configDir, err := config.ConfigDir()
	if err != nil {
		return fmt.Errorf("resolve config dir: %w", err)
	}

	result, err := usage.AggregateStats(configDir, statsSince, statsProfile)
	if err != nil {
		return fmt.Errorf("aggregate stats: %w", err)
	}

	if statsJSON {
		out, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal JSON: %w", err)
		}
		fmt.Println(string(out))
		return nil
	}

	sinceLabel := ""
	if statsSince != "" {
		sinceLabel = "since " + statsSince
	}
	fmt.Print(usage.FormatStats(result, sinceLabel))
	return nil
}
