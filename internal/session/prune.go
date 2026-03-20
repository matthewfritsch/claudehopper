package session

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PruneResult tracks what was deleted.
type PruneResult struct {
	Deleted    []Session
	BytesFreed int64
	Errors     []error
}

// Prune removes sessions older than the given duration.
func Prune(claudeDir string, olderThan time.Duration, dryRun bool) (*PruneResult, error) {
	sessions, err := ScanAll(claudeDir)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().Add(-olderThan)
	result := &PruneResult{}

	for _, sess := range sessions {
		activity := sess.LastActivity
		if activity.IsZero() {
			activity = sess.StartedAt
		}
		if activity.IsZero() || activity.After(cutoff) {
			continue
		}

		if dryRun {
			result.Deleted = append(result.Deleted, sess)
			result.BytesFreed += sess.FileSize
			continue
		}

		if err := deleteSession(sess); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("deleting %s: %w", sess.ID, err))
			continue
		}

		result.Deleted = append(result.Deleted, sess)
		result.BytesFreed += sess.FileSize
	}

	return result, nil
}

// PruneSessions removes specific sessions by ID.
func PruneSessions(claudeDir string, ids []string, dryRun bool) (*PruneResult, error) {
	result := &PruneResult{}

	for _, id := range ids {
		sess, err := FindByID(claudeDir, id)
		if err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		if dryRun {
			result.Deleted = append(result.Deleted, *sess)
			result.BytesFreed += sess.FileSize
			continue
		}

		if err := deleteSession(*sess); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("deleting %s: %w", sess.ID, err))
			continue
		}

		result.Deleted = append(result.Deleted, *sess)
		result.BytesFreed += sess.FileSize
	}

	return result, nil
}

func deleteSession(sess Session) error {
	if err := os.Remove(sess.FilePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	subagentDir := filepath.Join(filepath.Dir(sess.FilePath), sess.ID)
	if info, err := os.Stat(subagentDir); err == nil && info.IsDir() {
		if err := os.RemoveAll(subagentDir); err != nil {
			return fmt.Errorf("removing subagent dir: %w", err)
		}
	}
	return nil
}
