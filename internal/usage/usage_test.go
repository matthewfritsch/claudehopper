package usage_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
