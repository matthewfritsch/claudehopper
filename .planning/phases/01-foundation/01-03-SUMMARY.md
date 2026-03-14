---
phase: 01-foundation
plan: "03"
subsystem: config
tags: [go, json, xdg, config, manifest, paths, serialization]

# Dependency graph
requires:
  - phase: 01-foundation plan 01
    provides: go.mod with module path github.com/matthewfritsch/claudehopper

provides:
  - XDG-compliant config path resolution via os.UserConfigDir()
  - Python-compatible config.json Load/Save with byte-level fixture round-trip
  - Python-compatible .hop-manifest.json Load/Save with sorted managed_paths
  - NewManifest constructor with non-nil empty collections
  - ConfigDir, ProfilesDir, ProfileDir, ConfigFilePath functions
affects: [02-profile-ops, 03-symlinks, 04-cli-commands]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "os.UserConfigDir() for XDG-compliant path resolution — never store tilde strings"
    - "json.MarshalIndent with 2-space indent + trailing newline for Python format compatibility"
    - "sort.Strings before SaveManifest to maintain alphabetical managed_paths"
    - "Non-nil empty collections ([]string{} and map[string]string{}) to prevent null in JSON"
    - "TDD: failing test commit first, then implementation commit"

key-files:
  created:
    - internal/config/paths.go
    - internal/config/paths_test.go
    - internal/config/config.go
    - internal/config/config_test.go
    - internal/config/manifest.go
    - internal/config/manifest_test.go
    - internal/config/testdata/config.json
    - internal/config/testdata/hop-manifest.json
  modified: []

key-decisions:
  - "Use os.UserConfigDir() (stdlib) rather than adrg/xdg — sufficient for Linux XDG_CONFIG_HOME support without extra dependency"
  - "LoadConfig returns zero Config (Active='') for missing file rather than error — simplifies caller code for first-run initialization"
  - "SaveManifest sorts ManagedPaths on write (not in struct) to keep struct mutable while ensuring deterministic output"
  - "Fixture testdata files contain exact Python-format JSON for byte-level round-trip testing"

patterns-established:
  - "Pattern: Always use os.UserConfigDir() for config path base — never construct tilde paths"
  - "Pattern: ManagedPaths sorted alphabetically on every SaveManifest call"
  - "Pattern: Empty slice/map fields initialized as non-nil so JSON never produces null"
  - "Pattern: Round-trip fixture test reads original bytes, load+save, byte-compare to verify Python compatibility"

requirements-completed: [SAFE-03]

# Metrics
duration: 7min
completed: 2026-03-14
---

# Phase 01 Plan 03: Config Path Resolution and JSON Serialization Summary

**XDG-compliant config paths and byte-for-byte Python-compatible config.json and .hop-manifest.json serialization with fixture round-trip tests**

## Performance

- **Duration:** ~7 min
- **Started:** 2026-03-14T17:43:35Z
- **Completed:** 2026-03-14T17:50:00Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments

- ConfigDir/ProfilesDir/ProfileDir/ConfigFilePath using os.UserConfigDir() with full XDG_CONFIG_HOME support; paths are always absolute, never contain tilde
- Config struct + LoadConfig/SaveConfig producing 2-space-indented JSON with trailing newline matching Python json.dumps output
- Manifest struct + LoadManifest/SaveManifest with alphabetically sorted managed_paths, non-null empty collections, and byte-level fixture round-trip confirmation
- 18 tests, all passing; go vet clean

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: Failing tests for config path resolution** - `1a15565` (test)
2. **Task 1 GREEN: Config path resolution with XDG support** - `905fb2d` (feat)
3. **Task 2 RED: Failing tests for config.json and manifest serialization** - `a069c6d` (test)
4. **Task 2 GREEN: Python-compatible config.json and manifest serialization** - `79f6980` (feat)

_Note: TDD tasks produce test + feat commits per task_

## Files Created/Modified

- `internal/config/paths.go` - ConfigDir, ProfilesDir, ProfileDir, ConfigFilePath using os.UserConfigDir()
- `internal/config/paths_test.go` - XDG override, default, no-tilde, absolute-path, ProfilesDir, ProfileDir, ConfigFilePath tests
- `internal/config/config.go` - Config struct, LoadConfig (zero on missing file), SaveConfig (2-space indent + newline)
- `internal/config/config_test.go` - Load, save format, fixture round-trip, missing file tests
- `internal/config/manifest.go` - Manifest struct, NewManifest, LoadManifest, SaveManifest (sorted paths, non-null collections)
- `internal/config/manifest_test.go` - Field reads, sorted paths, indent/newline, null-safety, fixture round-trip, NewManifest tests
- `internal/config/testdata/config.json` - Python-format fixture: `{"active": "gsd"}` with 2-space indent + newline
- `internal/config/testdata/hop-manifest.json` - Python-format fixture with managed_paths array, shared_paths object, description

## Decisions Made

- Used os.UserConfigDir() (stdlib) rather than adding adrg/xdg dependency — stdlib is sufficient for the Linux/macOS XDG_CONFIG_HOME behavior required
- LoadConfig returns zero Config for missing file (not an error) to simplify first-run initialization patterns
- SaveManifest sorts ManagedPaths on write rather than keeping them sorted in the struct — callers can freely append without thinking about order
- Fixture files contain exact Python output as ground truth for byte-level round-trip tests

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

Minor: test file initially imported `"filepath"` (not a valid package path) instead of `"path/filepath"` — caught immediately when running the RED phase tests and fixed before the test commit.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- internal/config package is complete and ready for use by profile operations (Phase 2)
- ConfigDir, ProfilesDir, ProfileDir functions are the canonical path API for all subsequent phases
- LoadManifest/SaveManifest handle the .hop-manifest.json format Phase 2 will read and write per-profile

---
*Phase: 01-foundation*
*Completed: 2026-03-14*
