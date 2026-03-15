// Package usage provides usage tracking for claudehopper profile actions.
// Every switch, create, and delete operation is appended to usage.jsonl in
// the claudehopper config directory. Errors are swallowed — usage tracking is
// best-effort and must never interrupt normal CLI operation.
package usage

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// UsageEntry records a single profile action. JSON field names match the
// Python claudehopper format exactly.
type UsageEntry struct {
	Profile   string `json:"profile"`
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
}

// RecordUsage appends a JSON line to configDir/usage.jsonl recording the
// given profileName and action. It creates configDir if it does not exist.
// All errors are swallowed — this function never returns an error or panics.
func RecordUsage(configDir, profileName, action string) {
	// Create configDir if missing (first-run scenario)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return
	}

	entry := UsageEntry{
		Profile:   profileName,
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Action:    action,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	usagePath := filepath.Join(configDir, "usage.jsonl")
	f, err := os.OpenFile(usagePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	// Write JSON line + newline; ignore write errors
	_, _ = f.Write(append(data, '\n'))
}

// ReadUsage reads all usage entries from configDir/usage.jsonl and returns
// them as a slice. If the file does not exist, an empty slice and nil error
// are returned. Lines that fail to parse as UsageEntry are silently skipped.
func ReadUsage(configDir string) ([]UsageEntry, error) {
	usagePath := filepath.Join(configDir, "usage.jsonl")

	f, err := os.Open(usagePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []UsageEntry{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []UsageEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry UsageEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			// Skip malformed lines silently
			continue
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if entries == nil {
		entries = []UsageEntry{}
	}
	return entries, nil
}
