// Package profile provides operations for managing claudehopper profiles.
// This includes listing profiles, viewing status with per-path link health,
// and deleting profiles with dependent-profile warnings.
package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/matthewfritsch/claudehopper/internal/config"
)

// ProfileSummary holds display information for a single profile.
type ProfileSummary struct {
	Name         string
	Description  string
	ManagedCount int
	SharedCount  int
	IsActive     bool
}

// ListProfiles reads profilesDir and returns a sorted slice of ProfileSummary.
// configPath is the path to config.json which records the active profile name.
// Directories without a .hop-manifest.json are silently skipped.
// If configPath does not exist, no profile is marked active (first-run safe).
func ListProfiles(profilesDir, configPath string) ([]ProfileSummary, error) {
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ProfileSummary{}, nil
		}
		return nil, fmt.Errorf("read profiles dir: %w", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	var summaries []ProfileSummary
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(profilesDir, entry.Name(), ".hop-manifest.json")
		m, err := config.LoadManifest(manifestPath)
		if err != nil {
			// Not a profile directory — skip silently
			continue
		}
		summaries = append(summaries, ProfileSummary{
			Name:         entry.Name(),
			Description:  m.Description,
			ManagedCount: len(m.ManagedPaths),
			SharedCount:  len(m.SharedPaths),
			IsActive:     entry.Name() == cfg.Active,
		})
	}

	// Sort alphabetically by name
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})

	if summaries == nil {
		summaries = []ProfileSummary{}
	}

	return summaries, nil
}

// FormatProfileList formats a slice of ProfileSummary for display.
// Format matches the Python version:
//
//	  name (active)  [N paths, M shared] - desc
//	  name           [N paths] - desc
func FormatProfileList(summaries []ProfileSummary) string {
	if len(summaries) == 0 {
		return "No profiles found.\n"
	}

	var sb strings.Builder
	for _, s := range summaries {
		// Active marker
		activeMarker := ""
		if s.IsActive {
			activeMarker = " (active)"
		}

		// Path counts
		var pathInfo string
		if s.SharedCount > 0 {
			pathInfo = fmt.Sprintf("[%d paths, %d shared]", s.ManagedCount, s.SharedCount)
		} else {
			pathInfo = fmt.Sprintf("[%d paths]", s.ManagedCount)
		}

		// Description part
		descPart := ""
		if s.Description != "" {
			descPart = " - " + s.Description
		}

		fmt.Fprintf(&sb, "  %s%s  %s%s\n", s.Name, activeMarker, pathInfo, descPart)
	}
	return sb.String()
}
