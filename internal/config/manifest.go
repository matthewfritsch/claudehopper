package config

import (
	"encoding/json"
	"os"
	"sort"
)

// Manifest holds the content of a profile's .hop-manifest.json file.
// JSON format:
//   - managed_paths is a sorted JSON array of strings
//   - shared_paths is an object mapping filename to source profile name
//   - description is a plain string
type Manifest struct {
	CreatedFrom  string            `json:"created_from,omitempty"`
	ManagedPaths []string          `json:"managed_paths"`
	SharedPaths  map[string]string `json:"shared_paths"`
	Description  string            `json:"description"`
}

// NewManifest creates a Manifest with the given description and non-nil,
// empty collections so that JSON serialization produces [] and {} rather than null.
func NewManifest(description string) Manifest {
	return Manifest{
		ManagedPaths: []string{},
		SharedPaths:  map[string]string{},
		Description:  description,
	}
}

// LoadManifest reads and parses a .hop-manifest.json file at path.
// SharedPaths is always returned as a non-nil map.
func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, err
	}

	// Ensure SharedPaths is never nil after load so callers can safely
	// range over it and so SaveManifest produces {} not null.
	if m.SharedPaths == nil {
		m.SharedPaths = map[string]string{}
	}
	// Ensure ManagedPaths is never nil for the same reason.
	if m.ManagedPaths == nil {
		m.ManagedPaths = []string{}
	}

	return m, nil
}

// SaveManifest writes m to path as 2-space-indented JSON with a trailing newline.
// ManagedPaths is sorted alphabetically before writing. SharedPaths and
// ManagedPaths never serialize as null even when empty.
func SaveManifest(path string, m Manifest) error {
	// Work on a copy to avoid mutating the caller's slice.
	out := Manifest{
		CreatedFrom: m.CreatedFrom,
		Description: m.Description,
	}

	// Sort ManagedPaths alphabetically
	paths := make([]string, len(m.ManagedPaths))
	copy(paths, m.ManagedPaths)
	sort.Strings(paths)
	out.ManagedPaths = paths

	// Ensure SharedPaths is non-nil so it serializes as {} not null.
	if m.SharedPaths != nil {
		out.SharedPaths = m.SharedPaths
	} else {
		out.SharedPaths = map[string]string{}
	}

	// ManagedPaths is already a non-nil slice from the copy above.
	// If the input was nil, make an explicit empty slice.
	if out.ManagedPaths == nil {
		out.ManagedPaths = []string{}
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	// Append trailing newline for clean file endings
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}
