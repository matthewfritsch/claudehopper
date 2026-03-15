package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/matthewfritsch/claudehopper/internal/config"
)

// testWriteManifest writes a manifest JSON file to the given dir.
func testWriteManifest(t *testing.T, dir string, managed []string, shared map[string]string, desc string) {
	t.Helper()
	if managed == nil {
		managed = []string{}
	}
	if shared == nil {
		shared = map[string]string{}
	}
	m := map[string]interface{}{
		"managed_paths": managed,
		"shared_paths":  shared,
		"description":   desc,
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dir, ".hop-manifest.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

// loadManifestFromPath is a test helper that loads a manifest from an explicit path.
func loadManifestFromPath(path string) (config.Manifest, error) {
	return config.LoadManifest(path)
}

// marshalTestManifest marshals any value to indented JSON bytes.
func marshalTestManifest(v interface{}) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
