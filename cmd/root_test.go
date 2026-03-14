package cmd

import (
	"strings"
	"testing"
)

func TestRootCmdVersion_NotEmpty(t *testing.T) {
	SetVersionInfo("dev", "abc1234", "2026-01-01T00:00:00Z")
	if rootCmd.Version == "" {
		t.Fatal("expected rootCmd.Version to be non-empty after SetVersionInfo")
	}
}

func TestRootCmdVersion_NotDevel(t *testing.T) {
	SetVersionInfo("dev", "abc1234", "2026-01-01T00:00:00Z")
	if strings.Contains(rootCmd.Version, "(devel)") {
		t.Fatalf("expected rootCmd.Version not to contain '(devel)', got: %q", rootCmd.Version)
	}
}

func TestRootCmdUse(t *testing.T) {
	if rootCmd.Use != "claudehopper" {
		t.Fatalf("expected rootCmd.Use to be 'claudehopper', got %q", rootCmd.Use)
	}
}

func TestSetVersionInfo_FormatsCorrectly(t *testing.T) {
	SetVersionInfo("1.2.3", "abc1234", "2026-01-01T00:00:00Z")
	want := "1.2.3 (commit abc1234, built 2026-01-01T00:00:00Z)"
	if rootCmd.Version != want {
		t.Fatalf("expected version %q, got %q", want, rootCmd.Version)
	}
}
