package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/matthewfritsch/claudehopper/internal/config"
	"github.com/matthewfritsch/claudehopper/internal/profile"
	"github.com/matthewfritsch/claudehopper/internal/usage"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch NAME",
	Short: "Switch to a profile",
	Long: `Switch the active claudehopper profile.

All managed paths in ~/.claude/ are re-linked to the target profile.
Any real files in ~/.claude/ that conflict with managed paths are backed up.

Use --dry-run to preview changes without writing anything.
Use --force to re-link even if the target is already active.`,
	Args: cobra.ExactArgs(1),
	RunE: runSwitch,
}

var (
	switchDryRun bool
	switchForce  bool
)

func init() {
	switchCmd.Flags().BoolVar(&switchDryRun, "dry-run", false, "Preview changes without writing")
	switchCmd.Flags().BoolVar(&switchForce, "force", false, "Re-link even if target is already active")
	rootCmd.AddCommand(switchCmd)
}

func runSwitch(_ *cobra.Command, args []string) error {
	name := profile.NormalizeProfileName(args[0])

	profilesDir, err := config.ProfilesDir()
	if err != nil {
		return fmt.Errorf("resolve profiles dir: %w", err)
	}

	configPath, err := config.ConfigFilePath()
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}

	sharedDir, err := profile.SharedDir()
	if err != nil {
		return fmt.Errorf("resolve shared dir: %w", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	currentName := cfg.Active

	cDir := claudeDir()

	opts := profile.SwitchOptions{
		DryRun: switchDryRun,
		Force:  switchForce,
	}

	// Adopt-on-switch: detect unmanaged files before switching
	if !switchDryRun && currentName != "" {
		currentManifest, err := config.LoadManifest(
			filepath.Join(profilesDir, currentName, ".hop-manifest.json"),
		)
		if err == nil && isInteractive() {
			unmanaged, err := profile.DetectUnmanaged(cDir, sharedDir, currentManifest.ManagedPaths)
			if err == nil && len(unmanaged) > 0 {
				fmt.Fprintf(os.Stderr, "Unmanaged files found in ~/.claude/:\n")
				for _, f := range unmanaged {
					fmt.Fprintf(os.Stderr, "  %s\n", f)
				}
				fmt.Fprintf(os.Stderr, "Adopt these %d file(s) into profile %q? [y/N] ", len(unmanaged), currentName)

				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() {
					answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
					if answer == "y" || answer == "yes" {
						opts.AdoptFiles = unmanaged
					}
				}
			}
		}
	}

	result, err := profile.DoSwitch(profilesDir, cDir, configPath, sharedDir, name, currentName, opts)
	if err != nil {
		return err
	}

	if !switchDryRun {
		cfgDir, _ := config.ConfigDir()
		usage.RecordUsage(cfgDir, name, "switch")
	}

	if switchDryRun {
		// Print dry-run action list
		for _, action := range result.Actions {
			switch action.Action {
			case "link":
				if action.Detail != "" {
					fmt.Printf("would link: %s (%s)\n", action.Path, action.Detail)
				} else {
					fmt.Printf("would link: %s\n", action.Path)
				}
			case "backup":
				fmt.Printf("would backup: %s\n", action.Path)
			case "unlink":
				fmt.Printf("would unlink: %s\n", action.Path)
			default:
				fmt.Printf("would %s: %s\n", action.Action, action.Path)
			}
		}
		return nil
	}

	// Normal switch: summary line
	linkedCount := 0
	for _, action := range result.Actions {
		if action.Action == "link" {
			linkedCount++
		}
	}
	fmt.Printf("Switched to %q (%d paths linked)\n", name, linkedCount)

	if len(result.BackedUp) > 0 {
		fmt.Printf("Backed up %d file(s):\n", len(result.BackedUp))
		for _, f := range result.BackedUp {
			fmt.Printf("  %s -> %s.hop-backup\n", f, f)
		}
	}

	if len(result.Adopted) > 0 {
		fmt.Printf("Adopted %d file(s) into %q\n", len(result.Adopted), currentName)
	}

	return nil
}
