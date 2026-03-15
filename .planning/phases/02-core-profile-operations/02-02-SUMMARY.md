---
phase: 02-core-profile-operations
plan: "02"
subsystem: profile
tags: [go, profile, list, status, delete, symlinks]

requires:
  - phase: 01-foundation
    provides: config.LoadManifest, config.LoadConfig, fs.IsProtected, config paths
  - phase: 02-core-profile-operations
    plan: "01"
    provides: CreateBlank, ValidateProfileName, shared defaults

provides:
  - ListProfiles returning sorted ProfileSummary with active marker and path counts
  - GetProfileStatus returning per-path link health (linked/shared/conflict/not-linked/broken)
  - DeleteProfile with active-profile guard and DependentError for dependent profiles
  - FindDependents scanning shared_paths and created_from across all profiles
  - FormatProfileList and FormatProfileStatus output matching Python behavior

affects: [03-profile-switching, 04-cli-commands]

tech-stack:
  added: []
  patterns:
    - os.Lstat for symlink interrogation (never os.Stat for managed paths)
    - DependentError as structured error type for CLI-layer type assertion
    - Accept explicit directory paths in all functions (testable with t.TempDir)
    - TDD: RED commit then GREEN commit per feature group

key-files:
  created:
    - internal/profile/list.go
    - internal/profile/list_test.go
    - internal/profile/status.go
    - internal/profile/status_test.go
    - internal/profile/delete.go
    - internal/profile/delete_test.go
  modified:
    - internal/profile/testhelpers_test.go

key-decisions:
  - "DependentError returned by DeleteProfile lets CLI layer decide whether to prompt or force-delete"
  - "GetProfileStatus uses os.Lstat + os.Readlink + strings.HasPrefix for target classification"
  - "ListProfiles silently skips non-profile dirs (no manifest) to tolerate extra directories"
  - "FindDependents checks both shared_paths values AND created_from field for complete dependency graph"

patterns-established:
  - "PathHealth.Status string enum: linked/shared/conflict/not-linked/broken"
  - "ProfileSummary includes ManagedCount + SharedCount for quick display without full manifest read"
  - "DependentError: type DependentError struct { Profile string; Dependents []string }"

requirements-completed: [PROF-04, PROF-05, PROF-06]

duration: 5min
completed: 2026-03-14
---

# Phase 02 Plan 02: Profile List, Status, and Delete Summary

**Read-only and destructive profile query operations: list with active marker, per-path link health status, and delete with dependent-profile detection via DependentError**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-03-14T23:58:14Z
- **Completed:** 2026-03-14T23:03:00Z
- **Tasks:** 2 (TDD: 4 commits total)
- **Files modified:** 7

## Accomplishments
- ListProfiles returns sorted ProfileSummary slice with name, description, path counts, and active marker from config.json
- GetProfileStatus classifies each managed/shared path as linked/shared/conflict/not-linked/broken using os.Lstat
- DeleteProfile blocks deletion of active profile and returns typed DependentError when other profiles reference the target
- FindDependents scans shared_paths values and created_from fields across all profiles for complete dependency graph
- FormatProfileList and FormatProfileStatus output format matching Python claudehopper

## Task Commits

1. **Task 1 RED: List + Status failing tests** - `da7a90f` (test)
2. **Task 1 GREEN: ListProfiles and GetProfileStatus** - `dbc8b39` (feat)
3. **Task 2 RED: Delete failing tests** - `f7f3eda` (test)
4. **Task 2 GREEN: DeleteProfile, FindDependents** - `5f3f966` (feat)

## Files Created/Modified
- `internal/profile/list.go` - ProfileSummary struct, ListProfiles, FormatProfileList
- `internal/profile/list_test.go` - empty dir, multiple profiles, active marker, non-profile skip tests
- `internal/profile/status.go` - PathHealth, ProfileStatusInfo, GetProfileStatus, FormatProfileStatus
- `internal/profile/status_test.go` - linked/not-linked/conflict/broken/shared state tests
- `internal/profile/delete.go` - DependentError, FindDependents, DeleteProfile
- `internal/profile/delete_test.go` - remove dir, refuse active, DependentError, no-dependents tests
- `internal/profile/testhelpers_test.go` - added loadManifestFromPath and marshalTestManifest helpers

## Decisions Made
- DependentError is a typed error (not a string error) so CLI callers can type-assert and get the Dependents slice for structured prompts or --force handling
- GetProfileStatus accepts sharedDir as explicit parameter — when symlink target starts with sharedDir prefix, status is "shared"; when it starts with profileDir prefix, it's "linked"
- ListProfiles silently skips directories without .hop-manifest.json (tolerates accidental non-profile subdirs)
- FindDependents checks both shared_paths values AND created_from so that cloned profiles that inherit from a deleted profile also show as dependents

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Plan 02-01 test files existed but shared.go/create.go/testhelpers_test.go were missing helper functions**
- **Found during:** Task 2 (delete tests execution)
- **Issue:** `go test` build failed: `shared_test.go` and `create_test.go` referenced `EnsureSharedDefaults`, `LinkDefaultsIntoProfile`, `CreateBlank`, `CreateFromCurrent`, `CreateFromProfile`, `loadManifestFromPath`, `marshalTestManifest` — all undefined
- **Fix:** Discovered plan 02-01 had already been committed (`aff3559`) with those implementations. Added `loadManifestFromPath` and `marshalTestManifest` to `testhelpers_test.go`
- **Files modified:** `internal/profile/testhelpers_test.go`
- **Verification:** `go test ./internal/profile/... -count=1` — all 46 tests pass
- **Committed in:** `5f3f966`

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Auto-fix was necessary to unblock the build. No scope creep — helpers are small test utilities.

## Issues Encountered
- Plan 02-01 create_test.go had `TestCreateFromCurrent_ExcludesProtectedPaths` checking `os.Stat(profileDir/.hop-manifest.json) == nil` but CreateFromCurrent always writes a manifest there. The test was already fixed in the committed version of create_test.go (the 02-01 agent updated it). No change needed.

## Next Phase Readiness
- All profile query and delete operations implemented and tested
- DependentError type ready for CLI --force flag pattern in phase 04
- GetProfileStatus ready for use in profile switch safety checks in phase 03

---
*Phase: 02-core-profile-operations*
*Completed: 2026-03-14*
