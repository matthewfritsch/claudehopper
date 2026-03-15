---
phase: 03-extended-features
plan: 01
subsystem: usage
tags: [jsonl, usage-tracking, observability]

# Dependency graph
requires:
  - phase: 02-core-profile-operations
    provides: switch/create/delete commands to wire usage tracking into
provides:
  - internal/usage package with RecordUsage and ReadUsage
  - usage.jsonl append-only log on every profile action
affects: [03-extended-features plan 04 (stats/viz which reads usage.jsonl)]

# Tech tracking
tech-stack:
  added: []
  patterns: [best-effort side-effects swallow all errors, external test package for usage_test]

key-files:
  created:
    - internal/usage/usage.go
    - internal/usage/usage_test.go
  modified:
    - cmd/switch.go
    - cmd/create.go
    - cmd/delete.go

key-decisions:
  - "RecordUsage is void with no return value — callers never need to handle usage errors"
  - "External test package (usage_test) used to keep test imports explicit and match real caller usage"

patterns-established:
  - "Best-effort side-effect pattern: RecordUsage swallows all errors at every step — MkdirAll, Marshal, OpenFile, Write"
  - "configDir injected via config.ConfigDir() discarding error with _ — usage tracking degrades silently if config dir unresolvable"

requirements-completed: [OPS-03]

# Metrics
duration: 3min
completed: 2026-03-15
---

# Phase 3 Plan 01: Usage Tracking Summary

**append-only JSONL usage log (profile/timestamp/action) wired into switch, create, and delete via a void error-swallowing RecordUsage function**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-15T00:46:19Z
- **Completed:** 2026-03-15T00:48:53Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Created `internal/usage` package with `UsageEntry` struct, `RecordUsage` (void, error-swallowing), and `ReadUsage` (returns entries, handles missing file and malformed lines)
- Wired `RecordUsage` into all three mutating commands: switch (non-dry-run only), create, and delete (both normal and force paths)
- 6 tests cover all specified behaviors including first-run dir creation, multi-append, missing file, and malformed line skipping

## Task Commits

Each task was committed atomically:

1. **Task 1: Create internal/usage package (RED)** - `4b507ac` (test)
2. **Task 1: Create internal/usage package (GREEN)** - `1e6d091` (feat)
3. **Task 2: Wire RecordUsage into switch, create, delete** - `7ca70e2` (feat)

_Note: TDD task 1 has two commits (test RED → feat GREEN)_

## Files Created/Modified
- `internal/usage/usage.go` - UsageEntry struct, RecordUsage (void/swallows errors), ReadUsage (returns entries)
- `internal/usage/usage_test.go` - 6 tests for all specified behaviors using t.TempDir isolation
- `cmd/switch.go` - RecordUsage called after successful non-dry-run DoSwitch
- `cmd/create.go` - RecordUsage called after all three creation paths succeed
- `cmd/delete.go` - RecordUsage called in both runDelete success path and forceDelete

## Decisions Made
- RecordUsage is void (no return) — callers cannot misuse or forget to check it
- External test package `usage_test` used — tests reference exported symbols with package qualifier, matching real caller usage
- `cfgDir, _ := config.ConfigDir()` pattern used in all three command files — config dir resolution failure silently degrades tracking, never blocks the primary operation

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed external test package missing import**
- **Found during:** Task 1 (GREEN phase)
- **Issue:** Test file used `package usage_test` but referenced `RecordUsage`, `ReadUsage`, `UsageEntry` without package qualifier — build failed
- **Fix:** Added `"github.com/matthewfritsch/claudehopper/internal/usage"` import and prefixed all symbol references with `usage.`
- **Files modified:** internal/usage/usage_test.go
- **Verification:** All 6 tests pass
- **Committed in:** 1e6d091 (Task 1 feat commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - bug in test file package declaration)
**Impact on plan:** Minor — test file needed package import added. No scope change.

## Issues Encountered
- Initial test file used `package usage_test` (external test package) but referenced package symbols without import — fixed by adding the import and qualifying all symbol references.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `internal/usage` package is complete and tested — ready for Plan 04 (stats/viz) to call `ReadUsage`
- All profile actions (switch/create/delete) now append to `~/.config/claudehopper/usage.jsonl`

---
*Phase: 03-extended-features*
*Completed: 2026-03-15*

## Self-Check: PASSED

- internal/usage/usage.go: FOUND
- internal/usage/usage_test.go: FOUND
- commit 4b507ac (test RED): FOUND
- commit 1e6d091 (feat GREEN): FOUND
- commit 7ca70e2 (feat task 2): FOUND
