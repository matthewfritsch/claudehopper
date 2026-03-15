package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/matthewfritsch/claudehopper/internal/config"
	"github.com/matthewfritsch/claudehopper/internal/profile"
	"github.com/matthewfritsch/claudehopper/internal/usage"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create NAME",
	Short: "Create a new profile",
	Long: `Create a new claudehopper profile.

Without flags, creates a blank profile with a placeholder settings.json.
Use --from-current to capture the current ~/.claude/ configuration.
Use --from-profile=SOURCE to clone an existing profile.
Use --activate to switch to the new profile immediately after creation.`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

var (
	createFromCurrent bool
	createFromProfile string
	createActivate    bool
	createDescription string
)

func init() {
	createCmd.Flags().BoolVar(&createFromCurrent, "from-current", false, "Capture current ~/.claude/ config into the new profile")
	createCmd.Flags().StringVar(&createFromProfile, "from-profile", "", "Clone an existing profile")
	createCmd.Flags().BoolVar(&createActivate, "activate", false, "Switch to the new profile after creating it")
	createCmd.Flags().StringVar(&createDescription, "description", "", "Description for the new profile")
	rootCmd.AddCommand(createCmd)
}

func runCreate(cmd *cobra.Command, args []string) error {
	if createFromCurrent && createFromProfile != "" {
		return fmt.Errorf("--from-current and --from-profile are mutually exclusive")
	}

	name := profile.NormalizeProfileName(args[0])
	if err := profile.ValidateProfileName(name); err != nil {
		return err
	}

	profilesDir, err := config.ProfilesDir()
	if err != nil {
		return fmt.Errorf("resolve profiles dir: %w", err)
	}
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		return fmt.Errorf("create profiles dir: %w", err)
	}

	sharedDir, err := profile.SharedDir()
	if err != nil {
		return fmt.Errorf("resolve shared dir: %w", err)
	}
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		return fmt.Errorf("create shared dir: %w", err)
	}

	switch {
	case createFromCurrent:
		cDir := claudeDir()
		if err := profile.CreateFromCurrent(profilesDir, cDir, sharedDir, name, createDescription); err != nil {
			return err
		}
		// Display captured files matching Python output: list each file then summary
		profileDir, err := config.ProfileDir(name)
		if err != nil {
			return fmt.Errorf("resolve profile dir: %w", err)
		}
		manifestPath := profileDir + "/.hop-manifest.json"
		m, err := config.LoadManifest(manifestPath)
		if err != nil {
			return fmt.Errorf("load manifest: %w", err)
		}
		// Collect all unique captured file names from managed_paths + shared_paths
		seen := make(map[string]bool)
		var capturedFiles []string
		for _, f := range m.ManagedPaths {
			if !seen[f] {
				seen[f] = true
				capturedFiles = append(capturedFiles, f)
			}
		}
		for k := range m.SharedPaths {
			if !seen[k] {
				seen[k] = true
				capturedFiles = append(capturedFiles, k)
			}
		}
		sort.Strings(capturedFiles)
		for _, f := range capturedFiles {
			fmt.Println(f)
		}
		total := len(capturedFiles)
		fmt.Printf("Created profile %q from current config (%d files captured)\n", name, total)

	case createFromProfile != "":
		sourceName := profile.NormalizeProfileName(createFromProfile)
		if err := profile.CreateFromProfile(profilesDir, sharedDir, sourceName, name, createDescription); err != nil {
			return err
		}
		fmt.Printf("Created profile %q from %q\n", name, sourceName)

	default:
		if err := profile.CreateBlank(profilesDir, sharedDir, name, createDescription); err != nil {
			return err
		}
		fmt.Printf("Created profile %q\n", name)
	}

	cfgDir, _ := config.ConfigDir()
	usage.RecordUsage(cfgDir, name, "create")

	if createActivate {
		configPath, err := config.ConfigFilePath()
		if err != nil {
			return fmt.Errorf("resolve config path: %w", err)
		}
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		currentName := cfg.Active

		opts := profile.SwitchOptions{Force: true}
		if _, err := profile.DoSwitch(profilesDir, claudeDir(), configPath, sharedDir, name, currentName, opts); err != nil {
			return fmt.Errorf("switch to new profile: %w", err)
		}
		fmt.Printf("Switched to %q\n", name)
	}

	return nil
}
