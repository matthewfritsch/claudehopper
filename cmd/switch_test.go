package cmd

import (
	"testing"
)

func TestSwitchCmd_Registered(t *testing.T) {
	// Verify switch is registered on rootCmd
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "switch" {
			return
		}
	}
	t.Fatal("switch command not registered on rootCmd")
}

func TestSwitchCmd_Flags(t *testing.T) {
	// Verify --dry-run and --force flags are defined
	flags := []string{"dry-run", "force"}
	for _, flagName := range flags {
		if switchCmd.Flags().Lookup(flagName) == nil {
			t.Errorf("flag --%s not defined on switch command", flagName)
		}
	}
}

func TestSwitchCmd_NoArgs(t *testing.T) {
	// Test that switch requires exactly one argument
	rootCmd.SetArgs([]string{"switch"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when switch called with no args")
	}
}
