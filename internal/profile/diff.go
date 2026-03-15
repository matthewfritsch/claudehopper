package profile

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/matthewfritsch/claudehopper/internal/config"
)

// DiffResult holds the categorized comparison between two profiles' managed paths.
type DiffResult struct {
	OnlyA     []string // paths only in profile A's managed_paths
	OnlyB     []string // paths only in profile B's managed_paths
	Identical []string // common paths with identical file content
	Different []string // common paths with different file content (or one missing on disk)
}

// DiffProfiles compares the managed_paths of two profiles and categorizes each
// path as: only in A, only in B, common with identical content, or common with
// different content. File content is compared byte-by-byte for regular files.
// Directories are compared by existence only. All result slices are sorted.
func DiffProfiles(profilesDir, nameA, nameB string) (*DiffResult, error) {
	dirA := filepath.Join(profilesDir, nameA)
	dirB := filepath.Join(profilesDir, nameB)

	mA, err := config.LoadManifest(filepath.Join(dirA, ".hop-manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("load manifest for %q: %w", nameA, err)
	}
	mB, err := config.LoadManifest(filepath.Join(dirB, ".hop-manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("load manifest for %q: %w", nameB, err)
	}

	setA := make(map[string]bool, len(mA.ManagedPaths))
	for _, p := range mA.ManagedPaths {
		setA[p] = true
	}
	setB := make(map[string]bool, len(mB.ManagedPaths))
	for _, p := range mB.ManagedPaths {
		setB[p] = true
	}

	result := &DiffResult{}

	// Paths only in A
	for p := range setA {
		if !setB[p] {
			result.OnlyA = append(result.OnlyA, p)
		}
	}
	// Paths only in B
	for p := range setB {
		if !setA[p] {
			result.OnlyB = append(result.OnlyB, p)
		}
	}

	// Common paths: compare content
	for p := range setA {
		if !setB[p] {
			continue
		}
		pathA := filepath.Join(dirA, p)
		pathB := filepath.Join(dirB, p)

		statA, errA := os.Stat(pathA)
		statB, errB := os.Stat(pathB)

		if errA != nil || errB != nil {
			// One or both missing on disk — treat as different
			result.Different = append(result.Different, p)
			continue
		}

		if statA.IsDir() || statB.IsDir() {
			// Directories: compare by existence only — both exist so identical
			result.Identical = append(result.Identical, p)
			continue
		}

		// Both regular files — byte comparison
		if fileContentsEqual(pathA, pathB) {
			result.Identical = append(result.Identical, p)
		} else {
			result.Different = append(result.Different, p)
		}
	}

	// Sort all slices alphabetically for deterministic output
	sort.Strings(result.OnlyA)
	sort.Strings(result.OnlyB)
	sort.Strings(result.Identical)
	sort.Strings(result.Different)

	return result, nil
}

// fileContentsEqual returns true if both files have identical byte content.
func fileContentsEqual(pathA, pathB string) bool {
	a, err := os.ReadFile(pathA)
	if err != nil {
		return false
	}
	b, err := os.ReadFile(pathB)
	if err != nil {
		return false
	}
	return bytes.Equal(a, b)
}

// FormatDiff formats a DiffResult for display, matching the Python output format.
// Empty sections are omitted. Common section merges identical and different
// entries sorted alphabetically.
//
//	Only in 'nameA':
//	  file1
//	Only in 'nameB':
//	  file2
//	Common:
//	  file3  [identical]
//	  file4  [different]
func FormatDiff(result *DiffResult, nameA, nameB string) string {
	var sb strings.Builder

	if len(result.OnlyA) > 0 {
		fmt.Fprintf(&sb, "Only in '%s':\n", nameA)
		for _, p := range result.OnlyA {
			fmt.Fprintf(&sb, "  %s\n", p)
		}
	}

	if len(result.OnlyB) > 0 {
		fmt.Fprintf(&sb, "Only in '%s':\n", nameB)
		for _, p := range result.OnlyB {
			fmt.Fprintf(&sb, "  %s\n", p)
		}
	}

	hasCommon := len(result.Identical) > 0 || len(result.Different) > 0
	if hasCommon {
		fmt.Fprintf(&sb, "Common:\n")
		// Merge and sort common entries for stable output
		type commonEntry struct {
			path  string
			label string
		}
		var entries []commonEntry
		for _, p := range result.Identical {
			entries = append(entries, commonEntry{p, "[identical]"})
		}
		for _, p := range result.Different {
			entries = append(entries, commonEntry{p, "[different]"})
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].path < entries[j].path
		})
		for _, e := range entries {
			fmt.Fprintf(&sb, "  %s  %s\n", e.path, e.label)
		}
	}

	return sb.String()
}
