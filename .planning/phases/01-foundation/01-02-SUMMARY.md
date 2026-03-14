---
phase: 01-foundation
plan: "02"
subsystem: filesystem
tags: [go, renameio, atomic-symlink, protected-paths, tdd]

# Dependency graph
requires:
  - phase: 01-foundation plan 01
    provides: go.mod with github.com/matthewfritsch/claudehopper module, go toolchain

provides:
  - AtomicSymlink function in internal/fs: wraps renameio/v2 for POSIX-atomic symlink create/replace
  - IsProtected function in internal/fs: O(1) map lookup for 11 Python-derived protected path names
  - ProtectedPaths function in internal/fs: sorted slice of all protected names for display/debug
  - testdata/python_shared_paths.txt: drift-detection fixture tied to Python SHARED_PATHS

affects:
  - all profile switch operations (use AtomicSymlink)
  - profile list/create/delete (use IsProtected to guard credentials)
  - Phase 2 and beyond (both primitives are called by every profile mutation)

# Tech tracking
tech-stack:
  added:
    - github.com/google/renameio/v2 v2.0.2
  patterns:
    - TDD red-green: tests written first, committed separately from implementation
    - Drift-detection fixture: Python constant mirrored in testdata/ and verified bidirectionally in test
    - Bare-name only protection: IsProtected rejects any name containing '/' or '\\'

key-files:
  created:
    - internal/fs/atomic.go
    - internal/fs/atomic_test.go
    - internal/fs/protected.go
    - internal/fs/protected_test.go
    - internal/fs/testdata/python_shared_paths.txt

key-decisions:
  - "Use renameio.Symlink directly — no wrapper complexity, thin package boundary"
  - "IsProtected accepts bare names only — paths with separators always return false, matching expected caller contract"
  - "ProtectedPaths() exported alongside IsProtected so callers can display the set without re-declaring it"
  - "Drift-detection test uses bidirectional set comparison (fixture vs Go map) to catch both missing and extra entries"

patterns-established:
  - "internal/fs package: safety primitives only, no business logic"
  - "All symlink operations must go through AtomicSymlink — never os.Remove + os.Symlink"
  - "All credential/history path checks must go through IsProtected — never hardcoded string comparisons at callsite"

requirements-completed: [SAFE-01]

# Metrics
duration: 5min
completed: 2026-03-14
---

# Phase 1 Plan 02: Atomic Symlink Engine and Protected-Paths Enforcement Summary

**POSIX-atomic symlink create/replace via renameio/v2 and O(1) protected-path lookup with Python SHARED_PATHS drift detection**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-03-14T17:43:39Z
- **Completed:** 2026-03-14T17:48:00Z
- **Tasks:** 2
- **Files modified:** 5 created + go.mod/go.sum updated

## Accomplishments

- AtomicSymlink wraps renameio/v2 so profile switches never leave a broken symlink state
- IsProtected guards 11 credential/history paths from being touched during profile operations
- TestIsProtected_MatchesPythonConstants performs bidirectional fixture comparison to prevent silent drift from Python source
- 8 tests total: 4 AtomicSymlink tests, 4 IsProtected test groups — all pass, go vet clean

## Task Commits

Each task was committed atomically using TDD (test commit then implementation commit):

1. **Task 1: AtomicSymlink RED** — `c981eeb` (test)
2. **Task 1: AtomicSymlink GREEN** — `6be4c6b` (feat)
3. **Task 2: IsProtected RED** — `9d1de58` (test)
4. **Task 2: IsProtected GREEN** — `1869e0d` (feat)

**Plan metadata:** (docs commit — created next)

_Note: TDD tasks have two commits each (test RED → feat GREEN)_

## Files Created/Modified

- `internal/fs/atomic.go` — AtomicSymlink wrapping renameio.Symlink; Windows limitation documented
- `internal/fs/atomic_test.go` — 4 tests: new symlink, replace existing, dangling link, absolute target
- `internal/fs/protected.go` — sharedPaths map (11 entries), IsProtected, ProtectedPaths
- `internal/fs/protected_test.go` — 4 test groups: True, False, MatchesPythonConstants (drift), EdgeCases
- `internal/fs/testdata/python_shared_paths.txt` — 11-line fixture copied from Python SHARED_PATHS
- `go.mod` / `go.sum` — Added github.com/google/renameio/v2 v2.0.2

## Decisions Made

- IsProtected only accepts bare names (no path separators). Callers must strip directory components before checking. This matches the expected use case: checking whether a top-level entry under ~/.claude should be protected.
- ProtectedPaths() exported so display/CLI code does not need to re-declare the set, preventing a second point of drift.
- Drift-detection test uses bidirectional comparison: entries in fixture but not Go map, and entries in Go map but not fixture, are both failures.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Both safety primitives are ready for consumption by Phase 2 profile management
- Any code that switches profiles must call AtomicSymlink — os.Remove + os.Symlink is never acceptable
- Any code that touches ~/.claude entries must call IsProtected before mutation — guarding .credentials.json, history.jsonl, projects, cache, downloads, transcripts, shell-snapshots, file-history, backups, session-env, .session-stats.json

---
*Phase: 01-foundation*
*Completed: 2026-03-14*
