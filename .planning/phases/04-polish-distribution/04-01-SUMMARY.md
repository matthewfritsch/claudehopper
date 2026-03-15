---
phase: 04-polish-distribution
plan: 01
subsystem: infra
tags: [go-selfupdate, updater, self-update, ttl-cache, goroutine, cobra]

requires:
  - phase: 03-extended-features
    provides: Complete CLI with status, share, usage commands and config infrastructure

provides:
  - internal/updater package with CheckForUpdate (24h TTL stamp) and PerformUpdate (binary/source strategy)
  - hop update command for in-place binary upgrades
  - Non-blocking update notice in hop status with 3s timeout

affects:
  - 04-02-goreleaser-release (release artifacts needed for binary update to work end-to-end)

tech-stack:
  added: [github.com/creativeprojects/go-selfupdate v1.5.2, github.com/Masterminds/semver/v3 v3.4.0]
  patterns:
    - detectFunc package-level seam for mocking GitHub API calls in tests
    - Silent degradation pattern for update checks (nil,nil on network errors)
    - goroutine + select + time.After for non-blocking background checks in CLI commands

key-files:
  created:
    - internal/updater/updater.go
    - internal/updater/updater_test.go
    - cmd/update.go
  modified:
    - cmd/root.go
    - cmd/status.go
    - go.mod
    - go.sum

key-decisions:
  - "detectFunc seam for test isolation: package-level var replaces real GitHub call, no mock framework needed"
  - "Silent degradation on network errors in CheckForUpdate: return nil,nil so status never fails due to GitHub outage"
  - "isSourceInstall checks GOPATH/bin prefix via 'go env GOPATH': go install strategy for source, UpdateTo for binary"
  - "var Version string in cmd package set by SetVersionInfo: avoids re-parsing the formatted rootCmd.Version string"
  - "Update notice to stderr, not stdout: status output stays clean for scripted use"

patterns-established:
  - "Background goroutine pattern: launch check in goroutine, use select with time.After(3s) timeout"
  - "TTL stamp file: os.WriteFile empty file, os.Stat ModTime comparison against ttlDuration constant"

requirements-completed:
  - OPS-02

duration: 4min
completed: 2026-03-15
---

# Phase 4 Plan 1: Update Checker and hop update Command Summary

**24h TTL-cached GitHub release check in internal/updater with detectFunc seam for test isolation, plus hop update command using go-selfupdate v1.5.2 for source/binary install strategy**

## Performance

- **Duration:** ~4 min
- **Started:** 2026-03-15T01:21:18Z
- **Completed:** 2026-03-15T01:25:02Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Created `internal/updater` package with `CheckForUpdate` (stamp-file TTL, detectFunc seam) and `PerformUpdate` (source vs binary detection)
- Added `hop update` cobra command that calls `PerformUpdate`
- Wired non-blocking update check into `hop status` via goroutine + 3s timeout select, printing to stderr when newer version available
- Added go-selfupdate v1.5.2 dependency; all 5 TTL stamp tests pass with -race

## Task Commits

Each task was committed atomically:

1. **Task 1: Create internal/updater package with TTL-cached update check** - `5303d1d` (feat)
2. **Task 2: Wire update check into cmd/status.go and create cmd/update.go** - `01dd4a5` (feat)

## Files Created/Modified

- `internal/updater/updater.go` - CheckForUpdate (TTL stamp), PerformUpdate (source/binary), detectFunc seam, stripV, writeStamp
- `internal/updater/updater_test.go` - 5 TTL tests: SkipsWithinTTL, StampMissing, CallsAPIAfterTTL, AlreadyLatest, WritesStampAfterCheck
- `cmd/update.go` - `hop update` cobra command wired to updater.PerformUpdate
- `cmd/root.go` - Added `var Version string` set by SetVersionInfo for access from update/status commands
- `cmd/status.go` - Added goroutine + select update check with 3s timeout after profile status display
- `go.mod` / `go.sum` - Added go-selfupdate v1.5.2 and transitive dependencies

## Decisions Made

- **detectFunc seam:** package-level `var detectFunc func(...)` defaults to real `detectLatest`, overridden in tests. No mock framework needed, just function assignment.
- **Silent degradation:** `CheckForUpdate` returns `nil, nil` on network errors — status command must never fail because GitHub is down.
- **isSourceInstall via GOPATH/bin prefix:** `go env GOPATH` + filepath.Join gives portable detection without parsing build info.
- **var Version in cmd package:** SetVersionInfo stores bare version separately from the formatted `rootCmd.Version` string so update/status can use it without re-parsing.
- **Update notice to stderr:** Keeps stdout clean for scripts that parse `hop status` output.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required. Update check degrades silently if GitHub is unreachable.

## Next Phase Readiness

- `hop update` command is functional but binary download strategy requires goreleaser release artifacts (checksums.txt, named asset files) — covered in 04-02.
- Source install path (`go install`) is fully working for current dev users.

---
*Phase: 04-polish-distribution*
*Completed: 2026-03-15*

## Self-Check: PASSED

- internal/updater/updater.go: FOUND
- internal/updater/updater_test.go: FOUND
- cmd/update.go: FOUND
- 04-01-SUMMARY.md: FOUND
- Commit 5303d1d: FOUND
- Commit 01dd4a5: FOUND
