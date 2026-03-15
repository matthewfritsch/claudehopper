package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/matthewfritsch/claudehopper/internal/config"
)

// PathHealth describes the link health of a single managed or shared path.
// Status is one of: "linked", "shared", "conflict", "not-linked", "broken".
type PathHealth struct {
	Name   string
	Status string
	Detail string // e.g. source profile name for "shared", target for "broken"
}

// ProfileStatusInfo holds display information for a profile's full status.
type ProfileStatusInfo struct {
	Name        string
	Description string
	Paths       []PathHealth
}

// GetProfileStatus checks each managed and shared path in manifest and reports
// link health relative to claudeDir. All parameters accept filesystem paths so
// the function is testable with t.TempDir() without touching real config dirs.
//
// Health states:
//   - "linked"    — symlink in claudeDir points into profileDir
//   - "shared"    — symlink in claudeDir points into sharedDir
//   - "broken"    — symlink exists but target does not exist (dangling)
//   - "conflict"  — a real file or directory (not a symlink) exists at the link location
//   - "not-linked" — nothing exists at the link location
func GetProfileStatus(profileDir, claudeDir, sharedDir string, m config.Manifest) ProfileStatusInfo {
	info := ProfileStatusInfo{
		Description: m.Description,
	}

	// Extract profile name from the directory path
	info.Name = filepath.Base(profileDir)

	var paths []PathHealth

	// Track which paths we've already processed to avoid duplicates.
	// Shared paths may appear in both ManagedPaths and SharedPaths.
	seen := make(map[string]bool)

	// Check managed_paths, annotating shared ones from the SharedPaths map
	for _, name := range m.ManagedPaths {
		seen[name] = true
		linkPath := filepath.Join(claudeDir, name)
		ph := checkLinkHealth(name, linkPath, profileDir, sharedDir)
		if source, isShared := m.SharedPaths[name]; isShared && ph.Status == "shared" {
			ph.Detail = source
		}
		paths = append(paths, ph)
	}

	// Check any shared_paths not already in managed_paths
	for name, sourceProfile := range m.SharedPaths {
		if seen[name] {
			continue
		}
		linkPath := filepath.Join(claudeDir, name)
		ph := checkLinkHealth(name, linkPath, profileDir, sharedDir)
		if ph.Status == "shared" {
			ph.Detail = sourceProfile
		}
		paths = append(paths, ph)
	}

	if paths == nil {
		paths = []PathHealth{}
	}
	info.Paths = paths
	return info
}

// checkLinkHealth inspects a single link location and returns a PathHealth.
func checkLinkHealth(name, linkPath, profileDir, sharedDir string) PathHealth {
	fi, err := os.Lstat(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return PathHealth{Name: name, Status: "not-linked"}
		}
		return PathHealth{Name: name, Status: "not-linked", Detail: err.Error()}
	}

	// If it's not a symlink, it's a real file — conflict
	if fi.Mode()&os.ModeSymlink == 0 {
		return PathHealth{Name: name, Status: "conflict"}
	}

	// It's a symlink — read the target
	target, err := os.Readlink(linkPath)
	if err != nil {
		return PathHealth{Name: name, Status: "broken", Detail: err.Error()}
	}

	// Check if target exists (dangling symlink detection)
	if _, err := os.Stat(linkPath); err != nil {
		if os.IsNotExist(err) {
			return PathHealth{Name: name, Status: "broken", Detail: target}
		}
	}

	// Classify by target location
	if strings.HasPrefix(target, profileDir) {
		return PathHealth{Name: name, Status: "linked", Detail: target}
	}
	if sharedDir != "" && strings.HasPrefix(target, sharedDir) {
		return PathHealth{Name: name, Status: "shared", Detail: target}
	}

	// Symlink points somewhere else — treat as broken/unknown; use "broken"
	// because from this profile's perspective it is not correctly linked
	return PathHealth{Name: name, Status: "broken", Detail: target}
}

// FormatProfileStatus formats a ProfileStatusInfo for display.
// When verbose is false, only a summary line is shown if all paths are healthy;
// unhealthy paths are always listed individually.
// When verbose is true, every path is listed with its health indicator.
func FormatProfileStatus(info ProfileStatusInfo, verbose bool) string {
	var sb strings.Builder

	descPart := ""
	if info.Description != "" {
		descPart = " - " + info.Description
	}
	fmt.Fprintf(&sb, "Profile: %s%s\n", info.Name, descPart)

	if len(info.Paths) == 0 {
		sb.WriteString("  (no managed paths)\n")
		return sb.String()
	}

	if verbose {
		for _, ph := range info.Paths {
			fmt.Fprintf(&sb, "  %s  %s\n", ph.Name, formatHealthLabel(ph))
		}
		return sb.String()
	}

	// Compact mode: summarize healthy paths, list unhealthy ones individually.
	var linked, shared int
	var unhealthy []PathHealth
	for _, ph := range info.Paths {
		switch ph.Status {
		case "linked":
			linked++
		case "shared":
			shared++
		default:
			unhealthy = append(unhealthy, ph)
		}
	}

	healthy := linked + shared
	total := len(info.Paths)

	if len(unhealthy) == 0 {
		// All paths healthy — single summary line.
		if shared > 0 {
			fmt.Fprintf(&sb, "  %d paths linked (%d shared)\n", total, shared)
		} else {
			fmt.Fprintf(&sb, "  %d paths linked\n", total)
		}
	} else {
		// Mixed: summary + individual unhealthy paths.
		fmt.Fprintf(&sb, "  %d/%d paths healthy\n", healthy, total)
		for _, ph := range unhealthy {
			fmt.Fprintf(&sb, "  %s  %s\n", ph.Name, formatHealthLabel(ph))
		}
	}

	return sb.String()
}

// formatHealthLabel returns the display label for a single path's health.
func formatHealthLabel(ph PathHealth) string {
	switch ph.Status {
	case "linked":
		return "[linked]"
	case "shared":
		if ph.Detail != "" {
			return fmt.Sprintf("[linked, shared from %s]", ph.Detail)
		}
		return "[linked, shared]"
	case "conflict":
		return "[CONFLICT]"
	case "not-linked":
		return "[not linked]"
	case "broken":
		return "[broken]"
	default:
		return fmt.Sprintf("[%s]", ph.Status)
	}
}
