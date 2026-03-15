// Package usage provides usage tracking for claudehopper profile actions.
// Every switch, create, and delete operation is appended to usage.jsonl in
// the claudehopper config directory. Errors are swallowed — usage tracking is
// best-effort and must never interrupt normal CLI operation.
package usage

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// ProfileStats holds aggregated statistics for a single profile.
type ProfileStats struct {
	Name     string         `json:"name"`
	Switches int            `json:"switches"`
	LastUsed string         `json:"last_used"`
	Actions  map[string]int `json:"actions"`
}

// StatsResult holds the aggregated usage statistics across all profiles.
type StatsResult struct {
	TotalSwitches int            `json:"total_switches"`
	Profiles      []ProfileStats `json:"profiles"`
}

// AggregateStats reads all usage entries from configDir and returns aggregated
// statistics. sinceDate (YYYY-MM-DD) excludes entries before that date.
// profileFilter restricts results to a single profile name. Results are sorted
// by switch count descending, then alphabetically for ties.
func AggregateStats(configDir, sinceDate, profileFilter string) (*StatsResult, error) {
	entries, err := ReadUsage(configDir)
	if err != nil {
		return nil, err
	}

	// Build since cutoff string for lexicographic comparison (matches Python behavior)
	sincePrefix := ""
	if sinceDate != "" {
		sincePrefix = sinceDate + "T00:00:00"
	}

	// Aggregate: name -> stats
	type statsAccum struct {
		switches int
		lastUsed string
		actions  map[string]int
	}
	accum := make(map[string]*statsAccum)

	for _, e := range entries {
		// Apply since filter (lexicographic comparison on RFC3339 timestamps)
		if sincePrefix != "" && e.Timestamp < sincePrefix {
			continue
		}
		// Apply profile filter
		if profileFilter != "" && e.Profile != profileFilter {
			continue
		}

		a, ok := accum[e.Profile]
		if !ok {
			a = &statsAccum{actions: make(map[string]int)}
			accum[e.Profile] = a
		}
		a.actions[e.Action]++
		if e.Action == "switch" {
			a.switches++
		}
		if e.Timestamp > a.lastUsed {
			a.lastUsed = e.Timestamp
		}
	}

	// Build profile list
	profiles := make([]ProfileStats, 0, len(accum))
	totalSwitches := 0
	for name, a := range accum {
		profiles = append(profiles, ProfileStats{
			Name:     name,
			Switches: a.switches,
			LastUsed: a.lastUsed,
			Actions:  a.actions,
		})
		totalSwitches += a.switches
	}

	// Sort by switches descending, then name alphabetically
	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].Switches != profiles[j].Switches {
			return profiles[i].Switches > profiles[j].Switches
		}
		return profiles[i].Name < profiles[j].Name
	})

	return &StatsResult{
		TotalSwitches: totalSwitches,
		Profiles:      profiles,
	}, nil
}

// relativeTime returns a human-readable relative time string for a past timestamp.
// Format: <60min "Nm ago", <24h "Nh ago", <7d "Nd ago", else "Nw ago".
func relativeTime(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		d = 0
	}
	minutes := int(math.Round(d.Minutes()))
	hours := int(math.Round(d.Hours()))
	days := int(math.Round(d.Hours() / 24))
	weeks := int(math.Round(d.Hours() / (24 * 7)))

	switch {
	case minutes < 60:
		return fmt.Sprintf("%dm ago", minutes)
	case hours < 24:
		return fmt.Sprintf("%dh ago", hours)
	case days < 7:
		return fmt.Sprintf("%dd ago", days)
	default:
		return fmt.Sprintf("%dw ago", weeks)
	}
}

// FormatStats returns a human-readable formatted string for usage stats.
// sinceLabel is used in the header: "all time" if empty, otherwise the value
// (e.g. "since 2026-01-01").
func FormatStats(result *StatsResult, sinceLabel string) string {
	if sinceLabel == "" {
		sinceLabel = "all time"
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Profile usage (%s):\n", sinceLabel)

	// Find max name length for alignment
	maxName := 0
	for _, p := range result.Profiles {
		if len(p.Name) > maxName {
			maxName = len(p.Name)
		}
	}

	for _, p := range result.Profiles {
		lastStr := "never"
		if p.LastUsed != "" {
			if t, err := time.Parse(time.RFC3339, p.LastUsed); err == nil {
				lastStr = relativeTime(t)
			} else if t, err := time.Parse(time.RFC3339Nano, p.LastUsed); err == nil {
				lastStr = relativeTime(t)
			}
		}
		padding := strings.Repeat(" ", maxName-len(p.Name))
		fmt.Fprintf(&sb, "  %s%s  %3d switches  (last: %s)\n",
			p.Name, padding, p.Switches, lastStr)
	}

	profileCount := len(result.Profiles)
	fmt.Fprintf(&sb, "Total: %d switches across %d profiles\n",
		result.TotalSwitches, profileCount)

	return sb.String()
}
