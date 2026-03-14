package fs_test

import (
	"bufio"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/matthewfritsch/claudehopper/internal/fs"
)

// TestIsProtected_True verifies each of the 11 Python SHARED_PATHS entries
// individually returns true from IsProtected.
func TestIsProtected_True(t *testing.T) {
	protected := []string{
		".credentials.json",
		"history.jsonl",
		"projects",
		"cache",
		"downloads",
		"transcripts",
		"shell-snapshots",
		"file-history",
		"backups",
		"session-env",
		".session-stats.json",
	}

	for _, name := range protected {
		t.Run(name, func(t *testing.T) {
			if !fs.IsProtected(name) {
				t.Errorf("IsProtected(%q) = false; want true", name)
			}
		})
	}
}

// TestIsProtected_False verifies non-protected paths return false.
func TestIsProtected_False(t *testing.T) {
	notProtected := []string{
		"settings.json",
		"agents",
		"random.txt",
		"CLAUDE.md",
		"claude_desktop_config.json",
	}

	for _, name := range notProtected {
		t.Run(name, func(t *testing.T) {
			if fs.IsProtected(name) {
				t.Errorf("IsProtected(%q) = true; want false", name)
			}
		})
	}
}

// TestIsProtected_MatchesPythonConstants verifies the Go sharedPaths map has
// exactly the same entries as the python_shared_paths.txt fixture — no extras,
// no missing. This is the drift-detection test.
func TestIsProtected_MatchesPythonConstants(t *testing.T) {
	f, err := os.Open("testdata/python_shared_paths.txt")
	if err != nil {
		t.Fatalf("Failed to open fixture: %v", err)
	}
	defer f.Close()

	var fixtureNames []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fixtureNames = append(fixtureNames, line)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("Scanner error: %v", err)
	}

	goPaths := fs.ProtectedPaths()

	// Build sets for bidirectional comparison
	fixtureSet := make(map[string]struct{}, len(fixtureNames))
	for _, n := range fixtureNames {
		fixtureSet[n] = struct{}{}
	}
	goSet := make(map[string]struct{}, len(goPaths))
	for _, n := range goPaths {
		goSet[n] = struct{}{}
	}

	// Check for entries in fixture but missing from Go
	var missingInGo []string
	for n := range fixtureSet {
		if _, ok := goSet[n]; !ok {
			missingInGo = append(missingInGo, n)
		}
	}
	// Check for entries in Go but missing from fixture
	var extraInGo []string
	for n := range goSet {
		if _, ok := fixtureSet[n]; !ok {
			extraInGo = append(extraInGo, n)
		}
	}

	if len(missingInGo) > 0 {
		sort.Strings(missingInGo)
		t.Errorf("Paths in Python fixture but missing from Go sharedPaths: %v", missingInGo)
	}
	if len(extraInGo) > 0 {
		sort.Strings(extraInGo)
		t.Errorf("Paths in Go sharedPaths but missing from Python fixture: %v", extraInGo)
	}
}

// TestIsProtected_EdgeCases verifies edge cases: empty string and paths with slashes.
func TestIsProtected_EdgeCases(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"", false},
		{"/", false},
		{"/.credentials.json", false},
		{".credentials.json/", false},
		{"some/path/history.jsonl", false},
		{"history.jsonl/extra", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := fs.IsProtected(tc.name)
			if got != tc.want {
				t.Errorf("IsProtected(%q) = %v; want %v", tc.name, got, tc.want)
			}
		})
	}
}
