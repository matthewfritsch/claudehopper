---
phase: 03-extended-features
plan: "04"
subsystem: usage-stats-unmanage
tags: [usage, stats, unmanage, exit-ramp, cobra]
dependency_graph:
  requires:
    - internal/usage/usage.go (ReadUsage, UsageEntry from plan 03-01)
    - internal/config/config.go (LoadConfig, SaveConfig)
    - internal/fs/protected.go (IsProtected)
    - internal/profile/shared.go (copyFile)
    - internal/profile/create.go (copyDirRecursive)
  provides:
    - usage.AggregateStats
    - usage.FormatStats
    - usage.StatsResult
    - usage.ProfileStats
    - profile.UnmanageActive
    - cmd/stats.go (hop stats command)
    - cmd/unmanage.go (hop unmanage command)
  affects:
    - cmd layer (two new registered subcommands)
tech_stack:
  added: []
  patterns:
    - TDD red/green cycle for both packages
    - Lexicographic timestamp comparison for since-filter (matches Python)
    - Protected-path guard using fs.IsProtected in unmanage path
    - copyFile/copyDirRecursive reuse from profile package (same package access)
key_files:
  created:
    - internal/profile/unmanage.go
    - internal/profile/unmanage_test.go
    - cmd/stats.go
    - cmd/unmanage.go
  modified:
    - internal/usage/usage.go (added AggregateStats, FormatStats, ProfileStats, StatsResult)
    - internal/usage/usage_test.go (added 6 AggregateStats/FormatStats tests)
decisions:
  - "Lexicographic since-filter: compare entry.Timestamp >= sinceDate+'T00:00:00' matching Python behavior — no time.Parse needed"
  - "UnmanageActive is in profile package (not new package) to access unexported copyFile and copyDirRecursive"
  - "FormatStats right-aligns switch counts using %3d and pads profile names to max length for clean columns"
  - "relativeTime uses math.Round to avoid off-by-one on boundary minutes/hours"
metrics:
  duration: "~10min"
  completed: "2026-03-15"
  tasks_completed: 2
  files_modified: 6
---

# Phase 03 Plan 04: Usage Statistics and Unmanage Summary

Implemented `hop stats` for usage analytics and `hop unmanage` as a clean exit ramp from claudehopper. Stats show per-profile switch counts with filters; unmanage materializes all symlinks back to real files.

## Tasks Completed

### Task 1: Implement AggregateStats + UnmanageActive (TDD)

**Commit:** `52a9960` (test RED), `b649553` (feat GREEN)

Added to `internal/usage/usage.go`:
- `ProfileStats` and `StatsResult` types with JSON tags
- `AggregateStats(configDir, sinceDate, profileFilter string)` — reads usage.jsonl, applies filters, aggregates per-profile switch counts, sorts descending
- `FormatStats(result, sinceLabel string)` — human-readable output with right-aligned switch counts and relative timestamps

Created `internal/profile/unmanage.go`:
- `UnmanageActive(claudeDir, configPath string, dryRun bool)` — iterates claudeDir entries, skips protected paths and real files, replaces symlinks with real copies (file or directory), clears active in config.json; dry-run returns list without modifying

Created `internal/profile/unmanage_test.go` — 6 tests covering materialization, skip-real-files, skip-protected, config-clearing, dry-run, directory symlinks.

Added 6 tests to `internal/usage/usage_test.go` covering all AggregateStats and FormatStats behaviors.

### Task 2: Create Cobra Commands

**Commit:** `320cd78`

Created `cmd/stats.go`:
- `hop stats [--json] [--since YYYY-MM-DD] [--profile NAME]`
- Calls `usage.AggregateStats`, outputs JSON or human-readable via `usage.FormatStats`

Created `cmd/unmanage.go`:
- `hop unmanage [--dry-run]`
- Interactive confirmation prompt in TTY mode
- Calls `profile.UnmanageActive`, reports materialized count

## Deviations from Plan

None - plan executed exactly as written.

## Verification

- `go test ./internal/usage/ ./internal/profile/ -run "TestAggregateStats|TestFormatStats|TestUnmanageActive"` — 12 tests pass
- `go build ./...` — compiles cleanly
- `go vet ./...` — no warnings
- `go test ./... -timeout 60s` — full suite passes (5 packages, 0 failures)

## Self-Check: PASSED
