package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matthewfritsch/claudehopper/internal/config"
)

// writeFileAtPath creates a file with the given content at the absolute path (creating dirs as needed).
func writeFileAtPath(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func setupTwoProfiles(t *testing.T) (profilesDir string) {
	t.Helper()
	tmp := t.TempDir()
	profilesDir = filepath.Join(tmp, "profiles")
	return profilesDir
}

func TestDiffProfiles_OnlyA(t *testing.T) {
	profilesDir := setupTwoProfiles(t)

	mA := config.NewManifest("a")
	mA.ManagedPaths = []string{"CLAUDE.md", "only-in-a.txt"}
	writeManifest(t, profilesDir, "a", mA)
	// Create the actual files
	writeFileAtPath(t, filepath.Join(profilesDir, "a", "CLAUDE.md"), "content")
	writeFileAtPath(t, filepath.Join(profilesDir, "a", "only-in-a.txt"), "only a")

	mB := config.NewManifest("b")
	mB.ManagedPaths = []string{"CLAUDE.md"}
	writeManifest(t, profilesDir, "b", mB)
	writeFileAtPath(t, filepath.Join(profilesDir, "b", "CLAUDE.md"), "content")

	result, err := DiffProfiles(profilesDir, "a", "b")
	if err != nil {
		t.Fatalf("DiffProfiles: %v", err)
	}

	if len(result.OnlyA) != 1 || result.OnlyA[0] != "only-in-a.txt" {
		t.Errorf("expected OnlyA=[only-in-a.txt], got %v", result.OnlyA)
	}
	if len(result.OnlyB) != 0 {
		t.Errorf("expected empty OnlyB, got %v", result.OnlyB)
	}
}

func TestDiffProfiles_OnlyB(t *testing.T) {
	profilesDir := setupTwoProfiles(t)

	mA := config.NewManifest("a")
	mA.ManagedPaths = []string{"CLAUDE.md"}
	writeManifest(t, profilesDir, "a", mA)
	writeFileAtPath(t, filepath.Join(profilesDir, "a", "CLAUDE.md"), "content")

	mB := config.NewManifest("b")
	mB.ManagedPaths = []string{"CLAUDE.md", "only-in-b.txt"}
	writeManifest(t, profilesDir, "b", mB)
	writeFileAtPath(t, filepath.Join(profilesDir, "b", "CLAUDE.md"), "content")
	writeFileAtPath(t, filepath.Join(profilesDir, "b", "only-in-b.txt"), "only b")

	result, err := DiffProfiles(profilesDir, "a", "b")
	if err != nil {
		t.Fatalf("DiffProfiles: %v", err)
	}

	if len(result.OnlyB) != 1 || result.OnlyB[0] != "only-in-b.txt" {
		t.Errorf("expected OnlyB=[only-in-b.txt], got %v", result.OnlyB)
	}
	if len(result.OnlyA) != 0 {
		t.Errorf("expected empty OnlyA, got %v", result.OnlyA)
	}
}

func TestDiffProfiles_Identical(t *testing.T) {
	profilesDir := setupTwoProfiles(t)

	mA := config.NewManifest("a")
	mA.ManagedPaths = []string{"CLAUDE.md"}
	writeManifest(t, profilesDir, "a", mA)
	writeFileAtPath(t, filepath.Join(profilesDir, "a", "CLAUDE.md"), "same content")

	mB := config.NewManifest("b")
	mB.ManagedPaths = []string{"CLAUDE.md"}
	writeManifest(t, profilesDir, "b", mB)
	writeFileAtPath(t, filepath.Join(profilesDir, "b", "CLAUDE.md"), "same content")

	result, err := DiffProfiles(profilesDir, "a", "b")
	if err != nil {
		t.Fatalf("DiffProfiles: %v", err)
	}

	if len(result.Identical) != 1 || result.Identical[0] != "CLAUDE.md" {
		t.Errorf("expected Identical=[CLAUDE.md], got %v", result.Identical)
	}
	if len(result.Different) != 0 {
		t.Errorf("expected empty Different, got %v", result.Different)
	}
}

func TestDiffProfiles_Different(t *testing.T) {
	profilesDir := setupTwoProfiles(t)

	mA := config.NewManifest("a")
	mA.ManagedPaths = []string{"CLAUDE.md"}
	writeManifest(t, profilesDir, "a", mA)
	writeFileAtPath(t, filepath.Join(profilesDir, "a", "CLAUDE.md"), "content A")

	mB := config.NewManifest("b")
	mB.ManagedPaths = []string{"CLAUDE.md"}
	writeManifest(t, profilesDir, "b", mB)
	writeFileAtPath(t, filepath.Join(profilesDir, "b", "CLAUDE.md"), "content B")

	result, err := DiffProfiles(profilesDir, "a", "b")
	if err != nil {
		t.Fatalf("DiffProfiles: %v", err)
	}

	if len(result.Different) != 1 || result.Different[0] != "CLAUDE.md" {
		t.Errorf("expected Different=[CLAUDE.md], got %v", result.Different)
	}
	if len(result.Identical) != 0 {
		t.Errorf("expected empty Identical, got %v", result.Identical)
	}
}

func TestDiffProfiles_Mixed(t *testing.T) {
	profilesDir := setupTwoProfiles(t)

	mA := config.NewManifest("a")
	mA.ManagedPaths = []string{"common-identical.txt", "common-different.txt", "only-a.txt"}
	writeManifest(t, profilesDir, "a", mA)
	writeFileAtPath(t, filepath.Join(profilesDir, "a", "common-identical.txt"), "same")
	writeFileAtPath(t, filepath.Join(profilesDir, "a", "common-different.txt"), "in A")
	writeFileAtPath(t, filepath.Join(profilesDir, "a", "only-a.txt"), "only a")

	mB := config.NewManifest("b")
	mB.ManagedPaths = []string{"common-identical.txt", "common-different.txt", "only-b.txt"}
	writeManifest(t, profilesDir, "b", mB)
	writeFileAtPath(t, filepath.Join(profilesDir, "b", "common-identical.txt"), "same")
	writeFileAtPath(t, filepath.Join(profilesDir, "b", "common-different.txt"), "in B")
	writeFileAtPath(t, filepath.Join(profilesDir, "b", "only-b.txt"), "only b")

	result, err := DiffProfiles(profilesDir, "a", "b")
	if err != nil {
		t.Fatalf("DiffProfiles: %v", err)
	}

	if len(result.OnlyA) != 1 || result.OnlyA[0] != "only-a.txt" {
		t.Errorf("OnlyA: expected [only-a.txt], got %v", result.OnlyA)
	}
	if len(result.OnlyB) != 1 || result.OnlyB[0] != "only-b.txt" {
		t.Errorf("OnlyB: expected [only-b.txt], got %v", result.OnlyB)
	}
	if len(result.Identical) != 1 || result.Identical[0] != "common-identical.txt" {
		t.Errorf("Identical: expected [common-identical.txt], got %v", result.Identical)
	}
	if len(result.Different) != 1 || result.Different[0] != "common-different.txt" {
		t.Errorf("Different: expected [common-different.txt], got %v", result.Different)
	}
}

func TestFormatDiff(t *testing.T) {
	result := &DiffResult{
		OnlyA:     []string{"only-a.txt"},
		OnlyB:     []string{"only-b.txt"},
		Identical: []string{"same.txt"},
		Different: []string{"diff.txt"},
	}

	output := FormatDiff(result, "alpha", "beta")

	if !strings.Contains(output, "Only in 'alpha'") {
		t.Errorf("expected \"Only in 'alpha'\", got:\n%s", output)
	}
	if !strings.Contains(output, "Only in 'beta'") {
		t.Errorf("expected \"Only in 'beta'\", got:\n%s", output)
	}
	if !strings.Contains(output, "only-a.txt") {
		t.Errorf("expected 'only-a.txt' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "only-b.txt") {
		t.Errorf("expected 'only-b.txt' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "[identical]") {
		t.Errorf("expected '[identical]' marker, got:\n%s", output)
	}
	if !strings.Contains(output, "[different]") {
		t.Errorf("expected '[different]' marker, got:\n%s", output)
	}
}

func TestFormatDiff_EmptySections(t *testing.T) {
	// Empty sections should be skipped
	result := &DiffResult{
		Identical: []string{"same.txt"},
	}

	output := FormatDiff(result, "a", "b")

	if strings.Contains(output, "Only in") {
		t.Errorf("empty 'Only in' sections should be skipped, got:\n%s", output)
	}
	if !strings.Contains(output, "same.txt") {
		t.Errorf("expected 'same.txt' in output, got:\n%s", output)
	}
}
