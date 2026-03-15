// Package profile provides the business logic for creating, switching, and
// managing claudehopper profiles.
package profile

import (
	"fmt"
	"regexp"
	"strings"
)

// validProfileName matches names that are safe to use as directory names.
// Names must start with a lowercase letter or digit and contain only
// lowercase letters, digits, hyphens, and underscores.
var validProfileName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// ValidateProfileName validates that name is a safe, directory-friendly
// profile name. It normalizes (trims and lowercases) the name before
// checking. Returns an error describing the violation, or nil if valid.
func ValidateProfileName(name string) error {
	normalized := NormalizeProfileName(name)
	if normalized == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	if !validProfileName.MatchString(normalized) {
		return fmt.Errorf("profile name %q is invalid: must start with a letter or digit and contain only letters, digits, hyphens, and underscores", name)
	}
	return nil
}

// NormalizeProfileName trims whitespace and lowercases name.
func NormalizeProfileName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
