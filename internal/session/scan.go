package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Event represents a single line from a session JSONL file.
type Event struct {
	SessionID string          `json:"sessionId"`
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	UUID      string          `json:"uuid"`
	CWD       string          `json:"cwd"`
	Version   string          `json:"version"`
	GitBranch string          `json:"gitBranch"`
	Message   json.RawMessage `json:"message"`
}

// MessageContent represents parsed message content.
type MessageContent struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// Session holds metadata extracted from a session transcript.
type Session struct {
	ID           string
	Project      string // decoded project path
	ProjectKey   string // encoded project directory name
	FilePath     string // path to the .jsonl file
	FirstMessage string // first user message (topic)
	StartedAt    time.Time
	LastActivity time.Time
	MessageCount int
	FileSize     int64
	Version      string
	GitBranch    string
}

// decodeProjectPath converts an encoded project directory name back to a path.
// e.g., "-home-matthew-Programming" -> "/home/matthew/Programming"
func decodeProjectPath(encoded string) string {
	if encoded == "" {
		return ""
	}
	parts := strings.Split(encoded, "-")
	if len(parts) == 0 {
		return encoded
	}
	var pathParts []string
	for _, p := range parts {
		if p != "" {
			pathParts = append(pathParts, p)
		}
	}
	return "/" + strings.Join(pathParts, "/")
}

// ScanAll finds all sessions across all projects.
func ScanAll(claudeDir string) ([]Session, error) {
	return ScanFilter(claudeDir, "")
}

// ScanFilter finds sessions, optionally filtering by project path substring.
func ScanFilter(claudeDir, projectFilter string) ([]Session, error) {
	projDir := filepath.Join(claudeDir, "projects")
	entries, err := os.ReadDir(projDir)
	if err != nil {
		return nil, fmt.Errorf("reading projects dir %s: %w", projDir, err)
	}

	var sessions []Session

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectPath := decodeProjectPath(entry.Name())
		if projectFilter != "" && !strings.Contains(strings.ToLower(projectPath), strings.ToLower(projectFilter)) {
			continue
		}

		projSessions, err := scanProject(projDir, entry.Name(), projectPath)
		if err != nil {
			continue
		}
		sessions = append(sessions, projSessions...)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastActivity.After(sessions[j].LastActivity)
	})

	return sessions, nil
}

// scanProject scans a single project directory for sessions.
func scanProject(projDir, encodedName, projectPath string) ([]Session, error) {
	dir := filepath.Join(projDir, encodedName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var sessions []Session
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		sessionID := strings.TrimSuffix(entry.Name(), ".jsonl")
		filePath := filepath.Join(dir, entry.Name())

		info, err := entry.Info()
		if err != nil {
			continue
		}

		sess := Session{
			ID:         sessionID,
			Project:    projectPath,
			ProjectKey: encodedName,
			FilePath:   filePath,
			FileSize:   info.Size(),
		}

		if err := extractMetadata(&sess); err != nil {
			sess.LastActivity = info.ModTime()
			sess.StartedAt = info.ModTime()
		}

		sessions = append(sessions, sess)
	}

	return sessions, nil
}

// extractMetadata reads a session file to extract topic, timestamps, and message count.
func extractMetadata(sess *Session) error {
	f, err := os.Open(sess.FilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var (
		firstUserMsg string
		firstTime    time.Time
		lastTime     time.Time
		msgCount     int
		version      string
		gitBranch    string
		lastLine     string
	)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		lastLine = line
		msgCount++

		var ev Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}

		if t, err := parseTimestamp(ev.Timestamp); err == nil {
			if firstTime.IsZero() {
				firstTime = t
			}
			lastTime = t
		}

		if version == "" && ev.Version != "" {
			version = ev.Version
		}
		if gitBranch == "" && ev.GitBranch != "" {
			gitBranch = ev.GitBranch
		}

		if firstUserMsg == "" && len(ev.Message) > 0 {
			var msg MessageContent
			if err := json.Unmarshal(ev.Message, &msg); err == nil {
				if msg.Role == "user" {
					firstUserMsg = extractTextContent(msg.Content)
				}
			}
		}
	}

	if !lastTime.IsZero() || lastLine != "" {
		var ev Event
		if err := json.Unmarshal([]byte(lastLine), &ev); err == nil {
			if t, err := parseTimestamp(ev.Timestamp); err == nil {
				lastTime = t
			}
		}
	}

	sess.FirstMessage = truncate(firstUserMsg, 120)
	sess.StartedAt = firstTime
	sess.LastActivity = lastTime
	sess.MessageCount = msgCount
	sess.Version = version
	sess.GitBranch = gitBranch

	return nil
}

func parseTimestamp(ts string) (time.Time, error) {
	if ts == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02T15:04:05.000Z", ts); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unknown timestamp format: %s", ts)
}

func extractTextContent(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				return b.Text
			}
		}
	}
	return ""
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// FindByID finds a session by its ID (or prefix).
func FindByID(claudeDir, id string) (*Session, error) {
	all, err := ScanAll(claudeDir)
	if err != nil {
		return nil, err
	}

	var matches []Session
	for _, s := range all {
		if s.ID == id {
			return &s, nil
		}
		if strings.HasPrefix(s.ID, id) {
			matches = append(matches, s)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no session found matching %q", id)
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("ambiguous session ID %q matches %d sessions", id, len(matches))
	}
}

// GroupByProject groups sessions by their project path.
func GroupByProject(sessions []Session) map[string][]Session {
	groups := make(map[string][]Session)
	for _, s := range sessions {
		groups[s.Project] = append(groups[s.Project], s)
	}
	return groups
}

// FormatSize returns a human-readable file size.
func FormatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// FormatAge returns a human-readable relative time.
func FormatAge(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		months := int(d.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
}
