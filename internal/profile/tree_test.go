package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matthewfritsch/claudehopper/internal/config"
)

// writeManifest is a test helper that writes a manifest to a profile dir.
func writeManifest(t *testing.T, profilesDir, name string, m config.Manifest) {
	t.Helper()
	dir := filepath.Join(profilesDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := config.SaveManifest(filepath.Join(dir, ".hop-manifest.json"), m); err != nil {
		t.Fatalf("save manifest %s: %v", name, err)
	}
}

// writeConfig writes a config.json to configPath with the given active profile.
func writeConfig(t *testing.T, configPath, active string) {
	t.Helper()
	data := []byte(`{"active":"` + active + `"}`)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func TestBuildTree_SingleRoot(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	configPath := filepath.Join(tmp, "config.json")

	m := config.NewManifest("solo profile")
	m.ManagedPaths = []string{"CLAUDE.md", "settings.json"}
	writeManifest(t, profilesDir, "solo", m)
	writeConfig(t, configPath, "")

	roots, err := BuildTree(profilesDir, configPath)
	if err != nil {
		t.Fatalf("BuildTree: %v", err)
	}
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	if roots[0].Name != "solo" {
		t.Errorf("expected name 'solo', got %q", roots[0].Name)
	}
	if roots[0].ManagedCount != 2 {
		t.Errorf("expected ManagedCount=2, got %d", roots[0].ManagedCount)
	}
	if len(roots[0].Children) != 0 {
		t.Errorf("expected no children, got %d", len(roots[0].Children))
	}
}

func TestBuildTree_ParentChild(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	configPath := filepath.Join(tmp, "config.json")

	mParent := config.NewManifest("parent profile")
	mParent.ManagedPaths = []string{"CLAUDE.md"}
	writeManifest(t, profilesDir, "parent", mParent)

	mChild := config.NewManifest("child profile")
	mChild.ManagedPaths = []string{"CLAUDE.md", "commands/"}
	mChild.CreatedFrom = "parent"
	writeManifest(t, profilesDir, "child", mChild)
	writeConfig(t, configPath, "parent")

	roots, err := BuildTree(profilesDir, configPath)
	if err != nil {
		t.Fatalf("BuildTree: %v", err)
	}
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	if roots[0].Name != "parent" {
		t.Errorf("expected root 'parent', got %q", roots[0].Name)
	}
	if !roots[0].Active {
		t.Error("expected root to be active")
	}
	if len(roots[0].Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(roots[0].Children))
	}
	if roots[0].Children[0].Name != "child" {
		t.Errorf("expected child 'child', got %q", roots[0].Children[0].Name)
	}
}

func TestBuildTree_MultipleRoots(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	configPath := filepath.Join(tmp, "config.json")

	writeManifest(t, profilesDir, "alpha", config.NewManifest("alpha"))
	writeManifest(t, profilesDir, "beta", config.NewManifest("beta"))
	writeConfig(t, configPath, "")

	roots, err := BuildTree(profilesDir, configPath)
	if err != nil {
		t.Fatalf("BuildTree: %v", err)
	}
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(roots))
	}
	// Roots should be sorted alphabetically
	if roots[0].Name != "alpha" || roots[1].Name != "beta" {
		t.Errorf("expected sorted roots [alpha, beta], got [%s, %s]", roots[0].Name, roots[1].Name)
	}
}

func TestBuildTree_Cycle(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	configPath := filepath.Join(tmp, "config.json")

	// A created_from B, B created_from A — cycle
	mA := config.NewManifest("a")
	mA.CreatedFrom = "b"
	writeManifest(t, profilesDir, "a", mA)

	mB := config.NewManifest("b")
	mB.CreatedFrom = "a"
	writeManifest(t, profilesDir, "b", mB)
	writeConfig(t, configPath, "")

	// Must not loop indefinitely; just complete
	roots, err := BuildTree(profilesDir, configPath)
	if err != nil {
		t.Fatalf("BuildTree with cycle: %v", err)
	}
	// With a cycle both profiles reference each other; one will be root (the one
	// whose parent doesn't exist or is in the cycle) — we just verify it completes
	// and returns some roots without hanging.
	if len(roots) == 0 {
		t.Error("expected at least one root even in cyclic case")
	}
}

func TestRenderTree(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	configPath := filepath.Join(tmp, "config.json")

	mParent := config.NewManifest("parent")
	mParent.ManagedPaths = []string{"CLAUDE.md"}
	writeManifest(t, profilesDir, "parent", mParent)

	mChild := config.NewManifest("child")
	mChild.ManagedPaths = []string{"CLAUDE.md", "commands/"}
	mChild.CreatedFrom = "parent"
	writeManifest(t, profilesDir, "child", mChild)
	writeConfig(t, configPath, "")

	roots, err := BuildTree(profilesDir, configPath)
	if err != nil {
		t.Fatalf("BuildTree: %v", err)
	}

	output := RenderTree(roots)

	// Verify box-drawing connectors are present
	if !strings.Contains(output, "└── ") && !strings.Contains(output, "├── ") {
		t.Errorf("expected box-drawing connectors in output, got:\n%s", output)
	}
	// Verify profile names appear
	if !strings.Contains(output, "parent") {
		t.Errorf("expected 'parent' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "child") {
		t.Errorf("expected 'child' in output, got:\n%s", output)
	}
	// Verify managed count annotation
	if !strings.Contains(output, "managed") {
		t.Errorf("expected 'managed' annotation in output, got:\n%s", output)
	}
}

func TestRenderTree_ActiveMarker(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	configPath := filepath.Join(tmp, "config.json")

	writeManifest(t, profilesDir, "myprofile", config.NewManifest("desc"))
	writeConfig(t, configPath, "myprofile")

	roots, err := BuildTree(profilesDir, configPath)
	if err != nil {
		t.Fatalf("BuildTree: %v", err)
	}

	output := RenderTree(roots)
	if !strings.Contains(output, "(active)") {
		t.Errorf("expected '(active)' marker in output, got:\n%s", output)
	}
}

func TestRenderTree_SharedAnnotation(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	configPath := filepath.Join(tmp, "config.json")

	m := config.NewManifest("profile with shared")
	m.ManagedPaths = []string{"CLAUDE.md"}
	m.SharedPaths = map[string]string{"settings.json": "source-profile"}
	writeManifest(t, profilesDir, "myprofile", m)
	writeConfig(t, configPath, "")

	roots, err := BuildTree(profilesDir, configPath)
	if err != nil {
		t.Fatalf("BuildTree: %v", err)
	}

	output := RenderTree(roots)
	if !strings.Contains(output, "shared from") {
		t.Errorf("expected 'shared from' annotation in output, got:\n%s", output)
	}
	if !strings.Contains(output, "source-profile") {
		t.Errorf("expected source profile name in output, got:\n%s", output)
	}
}

func TestTreeJSON(t *testing.T) {
	tmp := t.TempDir()
	profilesDir := filepath.Join(tmp, "profiles")
	configPath := filepath.Join(tmp, "config.json")

	m := config.NewManifest("work context")
	m.ManagedPaths = []string{"CLAUDE.md", "commands/"}
	m.SharedPaths = map[string]string{"settings.json": "base"}
	writeManifest(t, profilesDir, "work", m)
	writeConfig(t, configPath, "work")

	roots, err := BuildTree(profilesDir, configPath)
	if err != nil {
		t.Fatalf("BuildTree: %v", err)
	}

	data, err := TreeJSON(roots, "work")
	if err != nil {
		t.Fatalf("TreeJSON: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal JSON: %v", err)
	}

	// Verify top-level keys
	if result["active"] != "work" {
		t.Errorf("expected active='work', got %v", result["active"])
	}
	profiles, ok := result["profiles"].([]interface{})
	if !ok {
		t.Fatalf("expected profiles array, got %T", result["profiles"])
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}

	p := profiles[0].(map[string]interface{})
	if p["name"] != "work" {
		t.Errorf("expected name='work', got %v", p["name"])
	}
	if p["active"] != true {
		t.Errorf("expected active=true, got %v", p["active"])
	}
	// managed_count should be present
	if _, ok := p["managed_count"]; !ok {
		t.Error("expected managed_count field in JSON")
	}
	// shared_count should be present
	if _, ok := p["shared_count"]; !ok {
		t.Error("expected shared_count field in JSON")
	}
	// children should be present
	if _, ok := p["children"]; !ok {
		t.Error("expected children field in JSON")
	}
}
