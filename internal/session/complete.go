package session

// SessionIDs returns all session IDs (for shell completion).
func SessionIDs(claudeDir string) []string {
	sessions, err := ScanAll(claudeDir)
	if err != nil {
		return nil
	}
	ids := make([]string, len(sessions))
	for i, s := range sessions {
		ids[i] = s.ID
	}
	return ids
}
