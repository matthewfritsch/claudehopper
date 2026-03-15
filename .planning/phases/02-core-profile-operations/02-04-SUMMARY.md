---
phase: 02-core-profile-operations
plan: "04"
subsystem: cli
tags: [cobra, cli, profile-management, switch, create, list, status, delete]

# Dependency graph
requires:
  - phase: 02-core-profile-operations
    plan: "01"
    provides: "CreateBlank, CreateFromCurrent, CreateFromProfile, ValidateProfileName, NormalizeProfileName"
  - phase: 02-core-profile-operations
    plan: "02"
    provides: "ListProfiles, FormatProfileList, GetProfileStatus, FormatProfileStatus, DeleteProfile, FindDependents"
  - phase: 02-core-profile-operations
    plan: "03"
    provides: "DoSwitch, SwitchOptions, SwitchResult, DetectUnmanaged"
provides:
  - "Five Cobra CLI commands: create, list, status, switch, delete"
  - "cmd/helpers.go: isInteractive() TTY check, claudeDir() helper"
  - "cmd/create.go: --from-current, --from-profile, --activate, --description flags with Python-format output"
  - "cmd/list.go: profile list with no-profiles hint"
  - "cmd/status.go: active profile link health display"
  - "cmd/switch.go: --dry-run, --force with adopt-on-switch TTY prompt"
  - "cmd/delete.go: --yes flag with dependent warning and interactive prompt"
affects: [03-polish-and-packaging, 04-distribution]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Thin Cobra command wrappers — RunE delegates immediately to internal/profile"
    - "Flag variables as package-level vars, reset between tests via rootCmd.SetArgs"
    - "isInteractive() gates all prompts — non-TTY silently skips"
    - "filepath.Join for all path construction in cmd layer"

key-files:
  created:
    - cmd/helpers.go
    - cmd/create.go
    - cmd/list.go
    - cmd/status.go
    - cmd/switch.go
    - cmd/delete.go
    - cmd/create_test.go
    - cmd/list_test.go
    - cmd/switch_test.go
  modified: []

key-decisions:
  - "isInteractive() uses os.Stdin.Stat() + ModeCharDevice — standard Unix TTY detection, no external dependency"
  - "claudeDir() uses os.UserHomeDir() with os.Getenv fallback for robustness"
  - "forceDelete on dependent profiles: --yes bypasses DependentError check by directly calling os.RemoveAll"
  - "Dry-run output format: 'would link/backup/unlink: path' matching Python CLI style"
  - "Adopt prompt: single bulk y/N for all unmanaged files — not per-file for UX simplicity"

patterns-established:
  - "Test via cobra cmd path (rootCmd.SetArgs + Execute) not direct RunE calls — avoids arg[0] panics"
  - "Command registration verified in tests via rootCmd.Commands() iteration"

requirements-completed: [PROF-01, PROF-02, PROF-03, PROF-04, PROF-05, PROF-06, PROF-07, SWCH-01, SWCH-02, SWCH-03, SWCH-04, SWCH-05, SWCH-06, SAFE-02, SHAR-04]

# Metrics
duration: 3min
completed: 2026-03-14
---

# Phase 2 Plan 04: CLI Command Wiring Summary

**Five Cobra commands (create/list/status/switch/delete) wired as thin wrappers over internal/profile business logic with TTY-aware adopt prompts, Python-format dry-run output, and dependent-warning on delete**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-14T~00:12:37Z
- **Completed:** 2026-03-14T~00:15:37Z
- **Tasks:** 2 (+ 1 auto-approved checkpoint)
- **Files modified:** 9

## Accomplishments
- All five CLI commands registered and delegating to tested internal/profile package
- TTY-aware adopt-on-switch prompt with silent skip for non-interactive mode
- Python-compatible dry-run output format (file-by-file action list)
- `--from-current` displays each captured file path then summary (matches Python output)
- Delete command warns about dependents with interactive confirmation or `--yes` force

## Task Commits

1. **Task 1: create, list, status, delete Cobra commands** - `5405363` (feat)
2. **Task 2: switch Cobra command with dry-run and adopt prompt** - `4878450` (feat)

**Plan metadata:** (docs commit follows)

## Files Created/Modified
- `cmd/helpers.go` - isInteractive() TTY check and claudeDir() path helper
- `cmd/create.go` - create with --from-current, --from-profile, --activate, --description
- `cmd/list.go` - list profiles, no-profiles hint message
- `cmd/status.go` - active profile link health via GetProfileStatus/FormatProfileStatus
- `cmd/switch.go` - switch with --dry-run, --force, adopt-on-switch TTY prompt
- `cmd/delete.go` - delete with --yes and DependentError warning prompt
- `cmd/create_test.go` - arg validation and flag registration tests
- `cmd/list_test.go` - command registration test
- `cmd/switch_test.go` - flag and arg validation tests

## Decisions Made
- `isInteractive()` uses `os.Stdin.Stat()` + `ModeCharDevice` — standard Unix TTY detection
- Adopt prompt is a single bulk y/N for all unmanaged files (not per-file)
- `forceDelete` on dependent profiles bypasses `DeleteProfile` and calls `os.RemoveAll` directly when `--yes` is set
- Dry-run output format: `would link: path` / `would backup: path` matching Python style

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Test for `TestCreateCmd_NoArgs` initially called `runCreate` directly with empty args, causing a panic on `args[0]`. Fixed by testing via `rootCmd.SetArgs + Execute` instead (cobra arg validation fires before RunE). No behavior change.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All profile management commands fully wired and tested
- Binary builds and all five commands appear in `--help`
- Full test suite passes with `-race`
- Ready for Phase 3: polish, shell completions, goreleaser distribution

---
*Phase: 02-core-profile-operations*
*Completed: 2026-03-14*
