package cmd

import (
	"fmt"
	"os"

	"github.com/matthewfritsch/claudehopper/internal/config"
	"github.com/matthewfritsch/claudehopper/internal/profile"
	"github.com/spf13/cobra"
)

var pathCmd = &cobra.Command{
	Use:   "path NAME",
	Short: "Print profile directory path",
	Long: `Print the absolute path to the named profile's directory.

Useful for scripting: $(hop path myprofile)/CLAUDE.md`,
	Args: cobra.ExactArgs(1),
	RunE: runPath,
}

func init() {
	rootCmd.AddCommand(pathCmd)
}

func runPath(_ *cobra.Command, args []string) error {
	name := profile.NormalizeProfileName(args[0])

	dir, err := config.ProfileDir(name)
	if err != nil {
		return fmt.Errorf("resolve profile dir: %w", err)
	}

	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("profile %q does not exist", name)
		}
		return fmt.Errorf("stat profile dir: %w", err)
	}

	fmt.Println(dir)
	return nil
}
