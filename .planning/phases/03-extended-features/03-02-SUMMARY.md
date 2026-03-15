---
phase: 03-extended-features
plan: 02
subsystem: profile
tags: [go, cobra, symlinks, sharing, manifest]

requires:
  - phase: 02-core-profile-operations
    provides: DoSwitch, LoadManifest/SaveManifest, copyFile helpers, profile management

provides:
  - ShareFiles function: symlink-based file sharing between profiles with manifest tracking
  - PickFiles function: independent file copy between profiles (symlink-preserving, recursive dirs)
  - UnshareFiles function: materialize shared symlinks back to owned file copies
  - hop share command: CLI for symlink-based file sharing
  - hop pick command: CLI for independent file copy between profiles
  - hop unshare command: CLI for materializing shared symlinks

affects:
  - 03-extended-features
  - cmd layer for all future commands that mutate profile state

tech-stack:
  added: []
  patterns:
    - "share.go business logic pattern: explicit profilesDir params, no internal config calls, dryRun flag"
    - "cmd re-link pattern: after mutation, DoSwitch --force re-links active profile"
    - "TDD pattern: tests in share_test.go use makeEmptyProfile/shareTestWriteFile/shareTestLoadManifest helpers to avoid naming conflicts with existing test helpers"

key-files:
  created:
    - internal/profile/share_test.go
    - cmd/share.go
    - cmd/pick.go
    - cmd/unshare.go
  modified: []

key-decisions:
  - "ShareFiles uses filepath.EvalSymlinks on source before creating target symlink — prevents chained symlinks when source is itself a symlink"
  - "UnshareFiles with empty paths unshares ALL shared_paths from manifest — provides bulk materialization"
  - "All three cmd commands re-link active profile via DoSwitch(Force=true) after manifest mutation — keeps ~/.claude/ in sync"
  - "share_test.go uses prefixed helpers (makeEmptyProfile, shareTestWriteFile, shareTestLoadManifest) to avoid redeclaration conflicts with existing test helpers in switch_test.go, diff_test.go, tree_test.go"

patterns-established:
  - "Pattern: share.go functions accept explicit profilesDir, avoid internal config calls — matches DoSwitch/CreateProfile pattern for test isolation"
  - "Pattern: addIfAbsent dedup inline in share.go (no separate helper) — manifest ManagedPaths dedup without import overhead"

requirements-completed: [SHAR-01, SHAR-02, SHAR-03]

duration: 20min
completed: 2026-03-14
---

# Phase 3 Plan 02: Share/Pick/Unshare Summary

**Profile file sharing via atomic symlinks (share), independent copy (pick), and symlink materialization (unshare) with three Cobra commands and 11 TDD tests**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-03-14
- **Completed:** 2026-03-14
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Implemented 11 TDD tests covering all ShareFiles/PickFiles/UnshareFiles behaviors including symlink chain resolution, dry-run, dedup, directory recursion, graceful missing-target handling
- Three Cobra commands (hop share, hop pick, hop unshare) wired to business logic with --from/--to/--profile/--dry-run flags and active profile re-link
- All tests pass; `go build ./...` and `go vet ./...` clean

## Task Commits

Each task was committed atomically:

1. **Task 1: TDD tests for ShareFiles, PickFiles, UnshareFiles** - `08c6b2f` (test)
2. **Task 2: Cobra commands for share, pick, unshare** - `ec3c7c5` (feat)

## Files Created/Modified

- `internal/profile/share_test.go` - 11 TDD tests covering all 3 functions and edge cases
- `cmd/share.go` - hop share command (--from required, --to optional, --dry-run)
- `cmd/pick.go` - hop pick command (--from required, --to optional, --dry-run)
- `cmd/unshare.go` - hop unshare command (--profile optional, --dry-run, empty args = all)

## Decisions Made

- ShareFiles uses `filepath.EvalSymlinks` to resolve source before creating target symlink — prevents chained symlinks (A->B->C becomes A->C directly)
- UnshareFiles with empty paths argument unshares everything in shared_paths — consistent with Python behavior
- All three Cobra commands re-link the active profile via DoSwitch(Force:true) after mutation to keep ~/.claude/ in sync with updated manifest
- Test helpers prefixed (makeEmptyProfile, shareTestWriteFile, shareTestLoadManifest) to avoid redeclaration conflicts with existing test helpers across the package

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Pre-existing diff_test.go/tree_test.go referenced unimplemented functions**
- **Found during:** Task 1 (TDD RED phase)
- **Issue:** diff_test.go referenced DiffProfiles/DiffResult/FormatDiff and tree_test.go referenced BuildTree/RenderTree/TreeJSON — none implemented. Package would not compile, blocking our test run.
- **Fix:** Confirmed these were already implemented in feat(03-03) commit (27167db) as part of a prior plan execution. share.go was also already committed. The only new work needed was share_test.go and cmd files.
- **Files modified:** none (implementations already existed)
- **Verification:** go build ./internal/profile/ passes; tests compile
- **Committed in:** Pre-existing (27167db)

**2. [Rule 1 - Bug] Test helper name conflicts with existing package helpers**
- **Found during:** Task 1 (TDD RED phase compilation)
- **Issue:** makeProfile() and writeFile() already declared in switch_test.go and diff_test.go with different signatures
- **Fix:** Renamed to makeEmptyProfile(), shareTestWriteFile(), shareTestLoadManifest() to avoid redeclaration
- **Files modified:** internal/profile/share_test.go
- **Verification:** Package compiles cleanly
- **Committed in:** 08c6b2f (Task 1)

---

**Total deviations:** 2 auto-handled (1 pre-existing resolved, 1 naming conflict fixed)
**Impact on plan:** No scope change; all fixes necessary for compilation and correctness.

## Issues Encountered

- share.go was already implemented by a prior plan execution (feat(03-03)); this plan focused on adding tests and the Cobra commands which were the remaining unimplemented artifacts.

## Next Phase Readiness

- ShareFiles, PickFiles, UnshareFiles fully tested and functional
- Three Cobra commands registered: hop share, hop pick, hop unshare
- Requirements SHAR-01, SHAR-02, SHAR-03 complete

---
*Phase: 03-extended-features*
*Completed: 2026-03-14*
