package cmd

import (
	"fmt"

	"github.com/matthewfritsch/claudehopper/internal/config"
	"github.com/matthewfritsch/claudehopper/internal/profile"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff PROFILE_A PROFILE_B",
	Short: "Compare two profiles",
	Long: `Compare the managed paths of two profiles side by side.

Shows paths unique to each profile and common paths with content comparison
(identical or different).`,
	Args: cobra.ExactArgs(2),
	RunE: runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
}

func runDiff(_ *cobra.Command, args []string) error {
	nameA := profile.NormalizeProfileName(args[0])
	nameB := profile.NormalizeProfileName(args[1])

	profilesDir, err := config.ProfilesDir()
	if err != nil {
		return fmt.Errorf("resolve profiles dir: %w", err)
	}

	result, err := profile.DiffProfiles(profilesDir, nameA, nameB)
	if err != nil {
		return fmt.Errorf("diff profiles: %w", err)
	}

	fmt.Print(profile.FormatDiff(result, nameA, nameB))
	return nil
}
