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

	// Check managed_paths
	for _, name := range m.ManagedPaths {
		linkPath := filepath.Join(claudeDir, name)
		ph := checkLinkHealth(name, linkPath, profileDir, sharedDir)
		paths = append(paths, ph)
	}

	// Check shared_paths
	for name, sourceProfile := range m.SharedPaths {
		linkPath := filepath.Join(claudeDir, name)
		ph := checkLinkHealth(name, linkPath, profileDir, sharedDir)
		// For shared paths, override detail with source profile name if shared
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
// Format matches the Python version per-path output.
func FormatProfileStatus(info ProfileStatusInfo) string {
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

	for _, ph := range info.Paths {
		var label string
		switch ph.Status {
		case "linked":
			label = "[linked]"
		case "shared":
			if ph.Detail != "" {
				label = fmt.Sprintf("[linked, shared from %s]", ph.Detail)
			} else {
				label = "[linked, shared]"
			}
		case "conflict":
			label = "[CONFLICT]"
		case "not-linked":
			label = "[not linked]"
		case "broken":
			label = "[broken]"
		default:
			label = fmt.Sprintf("[%s]", ph.Status)
		}
		fmt.Fprintf(&sb, "  %s  %s\n", ph.Name, label)
	}
	return sb.String()
}
