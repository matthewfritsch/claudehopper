---
phase: 02-core-profile-operations
plan: "01"
subsystem: profile
tags: [go, profile, manifest, create, validation, symlinks]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: config.Manifest, config.SaveManifest/LoadManifest, fs.IsProtected, fs.AtomicSymlink, config.ConfigDir
provides:
  - Manifest struct with CreatedFrom field (Python compat)
  - ValidateProfileName, NormalizeProfileName in internal/profile
  - CreateBlank, CreateFromCurrent, CreateFromProfile in internal/profile
  - EnsureSharedDefaults, LinkDefaultsIntoProfile, SharedDir in internal/profile
affects:
  - 02-02-list (consumes profile package)
  - 02-03-status (consumes profile package)
  - 02-04-switch (consumes profile package, CreateFromProfile for clone)
  - 02-05-delete (consumes profile package)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "TDD: write failing tests first, implement to pass, no separate refactor needed"
    - "Explicit dir params pattern: create functions accept profilesDir/sharedDir/claudeDir directly, never call config.ProfilesDir internally — enables t.TempDir() isolation"
    - "Manifest round-trip: every struct change paired with fixture+test before serialization"

key-files:
  created:
    - internal/config/testdata/hop-manifest-with-lineage.json
    - internal/profile/validate.go
    - internal/profile/validate_test.go
    - internal/profile/shared.go
    - internal/profile/shared_test.go
    - internal/profile/create.go
    - internal/profile/create_test.go
    - internal/profile/testhelpers_test.go
  modified:
    - internal/config/manifest.go
    - internal/config/manifest_test.go

key-decisions:
  - "CreatedFrom placed first in Manifest struct for Python field-order parity — Go serializes struct fields in declaration order"
  - "LinkDefaultsIntoProfile accepts explicit sharedDir parameter instead of calling SharedDir() internally — keeps function testable"
  - "CreateFromCurrent records existing symlinks in SharedPaths with value (shared) rather than managed_paths — symlinks are shared, not owned"
  - "Test for .hop-manifest.json exclusion checks managed_paths not file presence — SaveManifest legitimately writes .hop-manifest.json to profile dir"

patterns-established:
  - "Explicit dir params: all create/shared functions take explicit dir paths, never resolve from config internally"
  - "Symlink preservation: os.Lstat to detect symlinks, os.Readlink + os.Symlink to copy — never follow through symlinks during profile operations"
  - "Filter triple: IsProtected || HasPrefix(.hop-) || HasPrefix(.ccswap) for all from-current captures"

requirements-completed: [PROF-01, PROF-02, PROF-03, PROF-07, SAFE-02, SHAR-04]

# Metrics
duration: 35min
completed: 2026-03-14
---

# Phase 2 Plan 01: Profile Create Foundation Summary

**Manifest CreatedFrom field for Python round-trip compatibility + three profile create modes (blank, from-current, from-profile) with shared defaults bootstrapping via symlinks**

## Performance

- **Duration:** 35 min
- **Started:** 2026-03-14T23:40:00Z
- **Completed:** 2026-03-14T23:15:00Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments

- Added CreatedFrom to Manifest struct with omitempty; existing fixture still round-trips byte-identically
- Implemented ValidateProfileName (regex: ^[a-z0-9][a-z0-9_-]*$) and NormalizeProfileName (trim + lowercase)
- Implemented CreateBlank, CreateFromCurrent, CreateFromProfile in internal/profile/create.go — all accept explicit dir params for test isolation
- Implemented EnsureSharedDefaults, LinkDefaultsIntoProfile, SharedDir in internal/profile/shared.go
- All three create modes call LinkDefaultsIntoProfile to bootstrap DEFAULT_LINKED (settings.json, settings.local.json, .mcp.json) as shared symlinks
- CreateFromCurrent correctly filters protected, .hop-, .ccswap paths while preserving existing symlinks as symlinks

## Task Commits

Each task was committed atomically:

1. **Task 1: Add CreatedFrom to Manifest + profile name validation** - `c5535a2` (feat)
2. **Task 2: Profile create (blank, from-current, from-profile) + shared defaults** - `aff3559` (feat)

**Plan metadata:** TBD (docs: complete plan)

_Note: TDD tasks have single commits (RED phase confirmed build failure, GREEN phase produced passing tests)_

## Files Created/Modified

- `internal/config/manifest.go` - Added CreatedFrom field with omitempty, updated SaveManifest copy
- `internal/config/manifest_test.go` - Added CreatedFrom round-trip tests
- `internal/config/testdata/hop-manifest-with-lineage.json` - Python-format fixture with created_from field
- `internal/profile/validate.go` - ValidateProfileName, NormalizeProfileName
- `internal/profile/validate_test.go` - Table-driven tests for name validation
- `internal/profile/shared.go` - DefaultLinked, SharedDir, EnsureSharedDefaults, LinkDefaultsIntoProfile, copyFile
- `internal/profile/shared_test.go` - Tests for shared defaults bootstrapping
- `internal/profile/create.go` - CreateBlank, CreateFromCurrent, CreateFromProfile, copyDirContents, copyDirRecursive
- `internal/profile/create_test.go` - Tests for all three create modes
- `internal/profile/testhelpers_test.go` - testWriteManifest, loadManifestFromPath, marshalTestManifest

## Decisions Made

- CreatedFrom placed first in Manifest struct so Go's struct-order serialization matches Python key ordering
- LinkDefaultsIntoProfile accepts explicit sharedDir to enable t.TempDir() isolation in tests
- symlinks encountered during from-current capture are recorded in shared_paths (not managed_paths) with value "(shared)"
- Test for .hop- file exclusion validates managed_paths rather than file presence — SaveManifest legitimately writes .hop-manifest.json itself

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed test for .hop-manifest.json exclusion**
- **Found during:** Task 2 (profile create tests)
- **Issue:** Test `TestCreateFromCurrent_ExcludesProtectedPaths` checked `os.Stat(profileDir/.hop-manifest.json)` but CreateFromCurrent itself writes a manifest there, so the file always exists — test would always fail
- **Fix:** Changed assertion to verify .hop-manifest.json is not in manifest's managed_paths rather than checking file absence
- **Files modified:** internal/profile/create_test.go
- **Verification:** All tests pass
- **Committed in:** aff3559 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking test correction)
**Impact on plan:** Test semantics fix only — production code unaffected. No scope creep.

## Issues Encountered

- Pre-existing `list.go` and `status.go` stub files were already fully implemented in the profile package; no stubs needed to be created

## Next Phase Readiness

- Profile foundation complete: CreateBlank, CreateFromCurrent, CreateFromProfile all tested and passing
- Manifest CreatedFrom field ready for clone lineage tracking
- All three create modes call LinkDefaultsIntoProfile — shared defaults bootstrapped uniformly
- Ready for Plan 02-02 (list profiles) and 02-03 (profile status)

---
*Phase: 02-core-profile-operations*
*Completed: 2026-03-14*
