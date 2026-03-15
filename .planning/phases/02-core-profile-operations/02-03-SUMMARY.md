---
phase: 02-core-profile-operations
plan: "03"
subsystem: profile
tags: [go, switch, symlink, atomic, backup, preflight, dry-run]

# Dependency graph
requires:
  - phase: 02-core-profile-operations
    plan: "01"
    provides: config.Manifest, config.LoadManifest, config.SaveManifest, fs.AtomicSymlink, fs.IsProtected, profile.SharedDir
  - phase: 01-foundation
    provides: config.Config, config.LoadConfig, config.SaveConfig, fs.AtomicSymlink, fs.IsProtected
provides:
  - DoSwitch with preflight validation, dry-run, backup, unlink-old, link-new, config save
  - ValidatePreflight checks all managed paths exist in target profile dir
  - backupPath generates unique .hop-backup / .hop-backup.N suffixes via os.Lstat
  - linkManagedPath handles regular files, dirs, and symlinks with backup on conflict
  - DetectUnmanaged filters protected, .hop-, .ccswap, backup, and shared symlinks
  - AdoptUnmanaged moves claudeDir files into departing profile dir and updates manifest
  - SwitchOptions (DryRun, Force, AdoptFiles), SwitchAction, SwitchResult types
affects:
  - 02-04-delete (may call DoSwitch if deleting active profile)
  - cmd/hop switch command (consumes DoSwitch + DetectUnmanaged)
  - Phase 03 CLI (switch subcommand implementation)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "os.Lstat everywhere symlinks are interrogated — never os.Stat for managed paths"
    - "backupPath collision avoidance via Lstat loop — .hop-backup, .hop-backup.1, .hop-backup.2 ..."
    - "linkManagedPath symlink-target preservation: if profile entry is symlink, use Readlink target"
    - "DoSwitch sequence: guard -> load manifest -> preflight -> (dry-run exit) -> unlink-current -> link-target -> save config"
    - "DetectUnmanaged triple-filter: IsProtected || .hop- prefix || .ccswap prefix || .hop-backup substring || shared symlink"

key-files:
  created:
    - internal/profile/switch.go
    - internal/profile/switch_test.go
  modified: []

key-decisions:
  - "linkManagedPath uses os.Readlink on profile-dir entry when it is itself a symlink — preserves shared-dir indirection through the switch"
  - "DoSwitch skips unlinking real files from current profile (only removes symlinks) — protects any file the user placed directly"
  - "AdoptUnmanaged signature takes claudeDir explicitly rather than deriving it — consistent with explicit-dir-params pattern"
  - "backupPath uses os.Lstat not os.Stat — correctly treats dangling symlinks as occupying the path"

patterns-established:
  - "Switch sequence: validate preflight first (fail fast), then unlink, then link, then save config — never save config on error"
  - "Dry-run = return planned actions from ValidatePreflight, nothing else — zero filesystem writes"
  - "Backup before link: os.Rename (not copy) to move conflicting real files — atomic, no duplicate data"

requirements-completed: [SWCH-01, SWCH-02, SWCH-03, SWCH-04, SWCH-05, SWCH-06]

# Metrics
duration: 3min
completed: 2026-03-15
---

# Phase 2 Plan 03: Profile Switch Engine Summary

**Atomic profile switch engine with preflight validation, dry-run preview, .hop-backup collision-safe backups, unmanaged file detection, and adoption into the departing profile**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-15T00:07:07Z
- **Completed:** 2026-03-15T00:10:11Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Implemented ValidatePreflight: fails fast listing every missing managed path before any write occurs
- Implemented backupPath: .hop-backup / .hop-backup.N via os.Lstat loop (handles dangling symlinks)
- Implemented linkManagedPath: real-file/dir backup, wrong-symlink removal, Readlink preservation for shared symlinks
- Implemented DetectUnmanaged: filters five categories of non-user-owned entries, returns sorted list
- Implemented DoSwitch: full switch sequence with already-active guard, dry-run shortcircuit, unlink-old, link-new, config save
- Implemented AdoptUnmanaged: os.Rename from claudeDir to profileDir + manifest update + save

## Task Commits

Each task was committed atomically:

1. **Task 1: Preflight validation, backup, linkManagedPath, DetectUnmanaged** - `efbe26e` (feat)
2. **Task 2: DoSwitch orchestrator with dry-run, force, adopt, and backup** - `c2c31c8` (feat)

**Plan metadata:** TBD (docs: complete plan)

_Note: TDD tasks — tests written first, implementation confirmed GREEN before commit_

## Files Created/Modified

- `internal/profile/switch.go` - DoSwitch, ValidatePreflight, DetectUnmanaged, AdoptUnmanaged, backupPath, linkManagedPath, SwitchAction, SwitchOptions, SwitchResult
- `internal/profile/switch_test.go` - 21 tests covering all behavioral contracts: ValidatePreflight, backupPath, linkManagedPath, DetectUnmanaged, DoSwitch (dry-run, force, unlink-old, link-new, save config, backup conflicts), AdoptUnmanaged

## Decisions Made

- linkManagedPath uses os.Readlink on profile-dir entry when it is itself a symlink — preserves shared-dir indirection so settings.json in claudeDir still points to shared/ after switch
- DoSwitch only removes symlinks from current profile's managed paths (skips real files) — protects any file the user placed directly in claudeDir
- AdoptUnmanaged accepts claudeDir explicitly (not derived from config) — consistent with explicit-dir-params test isolation pattern established in Plan 01
- backupPath uses os.Lstat not os.Stat — dangling symlinks occupy the backup slot and must be skipped

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- DoSwitch is the core daily-use operation — ready for CLI wiring in Phase 03
- DetectUnmanaged + AdoptUnmanaged ready for the interactive adoption prompt in the switch subcommand
- All six SWCH requirements satisfied

---
*Phase: 02-core-profile-operations*
*Completed: 2026-03-15*

## Self-Check: PASSED

- internal/profile/switch.go: FOUND
- internal/profile/switch_test.go: FOUND
- .planning/phases/02-core-profile-operations/02-03-SUMMARY.md: FOUND
- commit efbe26e: FOUND
- commit c2c31c8: FOUND
