package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// DependentError is returned by DeleteProfile when other profiles reference
// the target profile via shared_paths or created_from. The CLI layer can
// type-assert to DependentError to display a structured warning and prompt.
type DependentError struct {
	Profile    string
	Dependents []string
}

func (e *DependentError) Error() string {
	return fmt.Sprintf(
		"profile %q has %d dependent profile(s): %v — delete them first or use --force",
		e.Profile, len(e.Dependents), e.Dependents,
	)
}

// manifestForDependents is a minimal struct for scanning dependent info.
// We don't use config.LoadManifest here to avoid importing a full Manifest
// when we only need shared_paths and created_from.
type manifestForDependents struct {
	SharedPaths map[string]string `json:"shared_paths"`
	CreatedFrom string            `json:"created_from"`
}

// FindDependents scans all profiles in profilesDir and returns a sorted list
// of profile names whose shared_paths values reference targetName, or whose
// created_from field equals targetName.
func FindDependents(profilesDir, targetName string) ([]string, error) {
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read profiles dir: %w", err)
	}

	var dependents []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == targetName {
			continue
		}

		manifestPath := filepath.Join(profilesDir, name, ".hop-manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			// Not a profile directory, skip
			continue
		}

		var m manifestForDependents
		if err := json.Unmarshal(data, &m); err != nil {
			// Corrupt manifest — skip
			continue
		}

		isDependent := false

		// Check shared_paths values
		for _, sourceProfile := range m.SharedPaths {
			if sourceProfile == targetName {
				isDependent = true
				break
			}
		}

		// Check created_from
		if !isDependent && m.CreatedFrom == targetName {
			isDependent = true
		}

		if isDependent {
			dependents = append(dependents, name)
		}
	}

	sort.Strings(dependents)
	if dependents == nil {
		dependents = []string{}
	}
	return dependents, nil
}

// DeleteProfile deletes the named profile from profilesDir.
//
// Guards:
//   - Returns an error if name == activeName (cannot delete the active profile)
//   - Returns a *DependentError if other profiles reference the target via
//     shared_paths or created_from (caller decides whether to proceed with --force)
//
// If no guard applies, the profile directory is removed recursively.
func DeleteProfile(profilesDir, name, activeName string) error {
	if name == activeName {
		return fmt.Errorf("cannot delete active profile %q — switch to another profile first", name)
	}

	deps, err := FindDependents(profilesDir, name)
	if err != nil {
		return fmt.Errorf("scan dependents: %w", err)
	}
	if len(deps) > 0 {
		return &DependentError{
			Profile:    name,
			Dependents: deps,
		}
	}

	profilePath := filepath.Join(profilesDir, name)
	if err := os.RemoveAll(profilePath); err != nil {
		return fmt.Errorf("remove profile %q: %w", name, err)
	}
	return nil
}
