package usage_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matthewfritsch/claudehopper/internal/usage"
)

// TestRecordUsage verifies that RecordUsage appends a valid JSON line to usage.jsonl
// with the expected profile, timestamp, and action fields.
func TestRecordUsage(t *testing.T) {
	dir := t.TempDir()
	usage.RecordUsage(dir, "myprofile", "switch")

	data, err := os.ReadFile(filepath.Join(dir, "usage.jsonl"))
	if err != nil {
		t.Fatalf("expected usage.jsonl to be created: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	var entry usage.UsageEntry
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if entry.Profile != "myprofile" {
		t.Errorf("expected profile %q, got %q", "myprofile", entry.Profile)
	}
	if entry.Action != "switch" {
		t.Errorf("expected action %q, got %q", "switch", entry.Action)
	}
	if entry.Timestamp == "" {
		t.Errorf("expected non-empty timestamp")
	}
}

// TestRecordUsage_CreatesDir verifies that RecordUsage creates configDir via os.MkdirAll if missing.
func TestRecordUsage_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "nested", "subdir")

	// Ensure subDir does not exist
	if _, err := os.Stat(subDir); !os.IsNotExist(err) {
		t.Fatalf("expected subDir to not exist")
	}

	usage.RecordUsage(subDir, "testprofile", "create")

	if _, err := os.Stat(subDir); err != nil {
		t.Errorf("expected subDir to be created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(subDir, "usage.jsonl")); err != nil {
		t.Errorf("expected usage.jsonl to be created in subDir: %v", err)
	}
}

// TestRecordUsage_NoError verifies that RecordUsage never panics or returns error
// even when given an impossible/invalid path.
func TestRecordUsage_NoError(t *testing.T) {
	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RecordUsage panicked: %v", r)
		}
	}()
	usage.RecordUsage("/dev/null/impossible", "profile", "delete")
}

// TestRecordUsage_MultipleAppends verifies that recording 3 entries produces 3 lines.
func TestRecordUsage_MultipleAppends(t *testing.T) {
	dir := t.TempDir()
	usage.RecordUsage(dir, "profile1", "switch")
	usage.RecordUsage(dir, "profile2", "create")
	usage.RecordUsage(dir, "profile3", "delete")

	data, err := os.ReadFile(filepath.Join(dir, "usage.jsonl"))
	if err != nil {
		t.Fatalf("expected usage.jsonl to be created: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	actions := []string{"switch", "create", "delete"}
	profiles := []string{"profile1", "profile2", "profile3"}
	for i, line := range lines {
		var entry usage.UsageEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("line %d: expected valid JSON: %v", i, err)
			continue
		}
		if entry.Profile != profiles[i] {
			t.Errorf("line %d: expected profile %q, got %q", i, profiles[i], entry.Profile)
		}
		if entry.Action != actions[i] {
			t.Errorf("line %d: expected action %q, got %q", i, actions[i], entry.Action)
		}
	}
}

// TestReadUsage_FileNotExist verifies that ReadUsage returns empty slice, nil error
// when usage.jsonl does not exist.
func TestReadUsage_FileNotExist(t *testing.T) {
	dir := t.TempDir()
	entries, err := usage.ReadUsage(dir)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty slice, got %d entries", len(entries))
	}
}

// TestReadUsage_SkipsMalformed verifies that ReadUsage skips malformed lines
// and returns only valid entries.
func TestReadUsage_SkipsMalformed(t *testing.T) {
	dir := t.TempDir()
	content := `{"profile":"p1","timestamp":"2026-01-01T00:00:00Z","action":"switch"}
not valid json at all
{"profile":"p2","timestamp":"2026-01-01T00:01:00Z","action":"create"}
{broken
`
	if err := os.WriteFile(filepath.Join(dir, "usage.jsonl"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	entries, err := usage.ReadUsage(dir)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 valid entries, got %d", len(entries))
	}
	if entries[0].Profile != "p1" {
		t.Errorf("expected first profile %q, got %q", "p1", entries[0].Profile)
	}
	if entries[1].Profile != "p2" {
		t.Errorf("expected second profile %q, got %q", "p2", entries[1].Profile)
	}
}

// writeTestEntries writes entries to usage.jsonl in dir.
func writeTestEntries(t *testing.T, dir string, entries []usage.UsageEntry) {
	t.Helper()
	var lines []string
	for _, e := range entries {
		b, err := json.Marshal(e)
		if err != nil {
			t.Fatalf("marshal entry: %v", err)
		}
		lines = append(lines, string(b))
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, "usage.jsonl"), []byte(content), 0644); err != nil {
		t.Fatalf("write usage.jsonl: %v", err)
	}
}

// TestAggregateStats_CountsSwitches verifies per-profile switch counts.
func TestAggregateStats_CountsSwitches(t *testing.T) {
	dir := t.TempDir()
	entries := []usage.UsageEntry{
		{Profile: "A", Timestamp: "2026-01-01T10:00:00Z", Action: "switch"},
		{Profile: "A", Timestamp: "2026-01-02T10:00:00Z", Action: "switch"},
		{Profile: "A", Timestamp: "2026-01-03T10:00:00Z", Action: "switch"},
		{Profile: "B", Timestamp: "2026-01-04T10:00:00Z", Action: "switch"},
		{Profile: "B", Timestamp: "2026-01-05T10:00:00Z", Action: "switch"},
	}
	writeTestEntries(t, dir, entries)

	result, err := usage.AggregateStats(dir, "", "")
	if err != nil {
		t.Fatalf("AggregateStats: %v", err)
	}
	if result.TotalSwitches != 5 {
		t.Errorf("TotalSwitches = %d, want 5", result.TotalSwitches)
	}
	if len(result.Profiles) != 2 {
		t.Fatalf("len(Profiles) = %d, want 2", len(result.Profiles))
	}
	// Sorted by count descending: A (3), B (2)
	if result.Profiles[0].Name != "A" || result.Profiles[0].Switches != 3 {
		t.Errorf("Profiles[0] = {%s, %d}, want {A, 3}", result.Profiles[0].Name, result.Profiles[0].Switches)
	}
	if result.Profiles[1].Name != "B" || result.Profiles[1].Switches != 2 {
		t.Errorf("Profiles[1] = {%s, %d}, want {B, 2}", result.Profiles[1].Name, result.Profiles[1].Switches)
	}
}

// TestAggregateStats_SinceFilter verifies that --since excludes entries before the date.
func TestAggregateStats_SinceFilter(t *testing.T) {
	dir := t.TempDir()
	entries := []usage.UsageEntry{
		{Profile: "A", Timestamp: "2026-01-01T10:00:00Z", Action: "switch"},
		{Profile: "A", Timestamp: "2026-01-15T10:00:00Z", Action: "switch"},
		{Profile: "A", Timestamp: "2026-02-01T10:00:00Z", Action: "switch"},
	}
	writeTestEntries(t, dir, entries)

	result, err := usage.AggregateStats(dir, "2026-01-10", "")
	if err != nil {
		t.Fatalf("AggregateStats: %v", err)
	}
	// Only entries on or after 2026-01-10 should be counted: 2 entries
	if result.TotalSwitches != 2 {
		t.Errorf("TotalSwitches = %d, want 2", result.TotalSwitches)
	}
}

// TestAggregateStats_ProfileFilter verifies --profile filters to a single profile.
func TestAggregateStats_ProfileFilter(t *testing.T) {
	dir := t.TempDir()
	entries := []usage.UsageEntry{
		{Profile: "A", Timestamp: "2026-01-01T10:00:00Z", Action: "switch"},
		{Profile: "A", Timestamp: "2026-01-02T10:00:00Z", Action: "switch"},
		{Profile: "B", Timestamp: "2026-01-03T10:00:00Z", Action: "switch"},
	}
	writeTestEntries(t, dir, entries)

	result, err := usage.AggregateStats(dir, "", "A")
	if err != nil {
		t.Fatalf("AggregateStats: %v", err)
	}
	if result.TotalSwitches != 2 {
		t.Errorf("TotalSwitches = %d, want 2", result.TotalSwitches)
	}
	if len(result.Profiles) != 1 || result.Profiles[0].Name != "A" {
		t.Errorf("Profiles = %v, want [{A, 2}]", result.Profiles)
	}
}

// TestAggregateStats_SortedByCount verifies descending sort with alphabetical tiebreak.
func TestAggregateStats_SortedByCount(t *testing.T) {
	dir := t.TempDir()
	entries := []usage.UsageEntry{
		{Profile: "C", Timestamp: "2026-01-01T10:00:00Z", Action: "switch"},
		{Profile: "A", Timestamp: "2026-01-02T10:00:00Z", Action: "switch"},
		{Profile: "B", Timestamp: "2026-01-03T10:00:00Z", Action: "switch"},
		{Profile: "A", Timestamp: "2026-01-04T10:00:00Z", Action: "switch"},
	}
	writeTestEntries(t, dir, entries)

	result, err := usage.AggregateStats(dir, "", "")
	if err != nil {
		t.Fatalf("AggregateStats: %v", err)
	}
	// A=2, B=1, C=1 — tie between B and C, alphabetical: B before C
	if len(result.Profiles) != 3 {
		t.Fatalf("len(Profiles) = %d, want 3", len(result.Profiles))
	}
	if result.Profiles[0].Name != "A" {
		t.Errorf("Profiles[0].Name = %q, want A", result.Profiles[0].Name)
	}
	if result.Profiles[1].Name != "B" {
		t.Errorf("Profiles[1].Name = %q, want B", result.Profiles[1].Name)
	}
	if result.Profiles[2].Name != "C" {
		t.Errorf("Profiles[2].Name = %q, want C", result.Profiles[2].Name)
	}
}

// TestAggregateStats_EmptyFile verifies that no entries yields empty result.
func TestAggregateStats_EmptyFile(t *testing.T) {
	dir := t.TempDir()

	result, err := usage.AggregateStats(dir, "", "")
	if err != nil {
		t.Fatalf("AggregateStats: %v", err)
	}
	if result.TotalSwitches != 0 {
		t.Errorf("TotalSwitches = %d, want 0", result.TotalSwitches)
	}
	if len(result.Profiles) != 0 {
		t.Errorf("len(Profiles) = %d, want 0", len(result.Profiles))
	}
}

// TestFormatStats_Output verifies human-readable output format.
func TestFormatStats_Output(t *testing.T) {
	now := time.Now()
	result := &usage.StatsResult{
		TotalSwitches: 10,
		Profiles: []usage.ProfileStats{
			{
				Name:     "work",
				Switches: 8,
				LastUsed: now.Add(-2 * time.Hour).Format(time.RFC3339),
				Actions:  map[string]int{"switch": 8},
			},
			{
				Name:     "personal",
				Switches: 2,
				LastUsed: now.Add(-30 * time.Minute).Format(time.RFC3339),
				Actions:  map[string]int{"switch": 2},
			},
		},
	}

	out := usage.FormatStats(result, "")
	if !strings.Contains(out, "work") {
		t.Errorf("output missing 'work': %q", out)
	}
	if !strings.Contains(out, "personal") {
		t.Errorf("output missing 'personal': %q", out)
	}
	if !strings.Contains(out, "switches") {
		t.Errorf("output missing 'switches': %q", out)
	}
	if !strings.Contains(out, "all time") {
		t.Errorf("output missing 'all time': %q", out)
	}
	if !strings.Contains(out, "10 switches") {
		t.Errorf("output missing total '10 switches': %q", out)
	}
}
