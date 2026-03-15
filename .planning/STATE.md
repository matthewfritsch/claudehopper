---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
status: planning
stopped_at: Completed 03-extended-features-02-PLAN.md
last_updated: "2026-03-15T00:54:29.385Z"
last_activity: 2026-03-14 — Roadmap created
progress:
  total_phases: 4
  completed_phases: 2
  total_plans: 11
  completed_plans: 10
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-14)

**Core value:** Instant, safe profile switching for Claude Code configs — users can swap their entire Claude Code setup with a single command without risking credentials or shared data.
**Current focus:** Phase 1 — Foundation

## Current Position

Phase: 1 of 4 (Foundation)
Plan: 0 of TBD in current phase
Status: Ready to plan
Last activity: 2026-03-14 — Roadmap created

Progress: [░░░░░░░░░░] 0%

## Performance Metrics

**Velocity:**
- Total plans completed: 0
- Average duration: —
- Total execution time: 0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| - | - | - | - |

**Recent Trend:**
- Last 5 plans: —
- Trend: —

*Updated after each plan completion*
| Phase 01-foundation P01 | 2 | 2 tasks | 7 files |
| Phase 01-foundation P02 | 5min | 2 tasks | 7 files |
| Phase 01-foundation P03 | 7 | 2 tasks | 8 files |
| Phase 02-core-profile-operations P01 | 35min | 2 tasks | 10 files |
| Phase 02-core-profile-operations P02 | 5min | 2 tasks | 7 files |
| Phase 02-core-profile-operations P03 | 3min | 2 tasks | 2 files |
| Phase 02-core-profile-operations P04 | 3min | 2 tasks | 9 files |
| Phase 03-extended-features P01 | 3min | 2 tasks | 5 files |
| Phase 03-extended-features P03 | 6min | 2 tasks | 8 files |
| Phase 03-extended-features P02 | 20min | 2 tasks | 4 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Use Cobra for CLI framework (shell completions nearly free)
- Multi-file `cmd/` + `internal/` package structure (not monolithic)
- Maintain Python manifest/config format compatibility (fixture tests required before serialization)
- `google/renameio/v2` for atomic symlinks; never `os.Remove` + `os.Symlink`
- Use `os.Lstat` (not `os.Stat`) everywhere we interrogate managed symlinks
- [Phase 01-foundation]: Single main.go entry point with goreleaser dual build stanzas producing both claudehopper and hop binary names
- [Phase 01-foundation]: No Windows builds in goreleaser — renameio/v2 does not support atomic symlinks on Windows
- [Phase 01-foundation]: IsProtected accepts bare names only — paths with separators return false, matching top-level caller contract
- [Phase 01-foundation]: Drift-detection test uses bidirectional fixture comparison for Python SHARED_PATHS parity
- [Phase 01-foundation]: os.UserConfigDir() for XDG paths (no extra dependency)
- [Phase 01-foundation]: LoadConfig returns zero Config for missing file — simplifies first-run initialization
- [Phase 01-foundation]: SaveManifest sorts ManagedPaths on write — callers can append without managing order
- [Phase 02-core-profile-operations]: CreatedFrom placed first in Manifest struct for Python field-order parity
- [Phase 02-core-profile-operations]: Explicit dir params pattern: create functions accept dir paths directly, never call config.ProfilesDir internally — enables t.TempDir() test isolation
- [Phase 02-core-profile-operations]: CreateFromCurrent records existing symlinks in shared_paths not managed_paths — symlinks are shared data, not owned by the profile
- [Phase 02-core-profile-operations]: DependentError returned by DeleteProfile lets CLI layer decide whether to prompt or force-delete
- [Phase 02-core-profile-operations]: GetProfileStatus uses os.Lstat + os.Readlink + strings.HasPrefix for link target classification (linked/shared/broken)
- [Phase 02-core-profile-operations]: FindDependents checks both shared_paths values AND created_from field for complete dependency scanning
- [Phase 02-core-profile-operations]: linkManagedPath uses os.Readlink on profile-dir symlinks to preserve shared-dir indirection through the switch
- [Phase 02-core-profile-operations]: DoSwitch only removes symlinks from current profile (skips real files) — protects directly placed user files
- [Phase 02-core-profile-operations]: backupPath uses os.Lstat not os.Stat — dangling symlinks occupy the backup slot
- [Phase 02-core-profile-operations]: isInteractive() uses os.Stdin.Stat() + ModeCharDevice for TTY detection in cmd layer — no external dependency
- [Phase 02-core-profile-operations]: Adopt prompt is a single bulk y/N for all unmanaged files — not per-file for UX simplicity
- [Phase 03-extended-features]: RecordUsage is void with no return value — callers never need to handle usage errors
- [Phase 03-extended-features]: configDir, _ = config.ConfigDir() pattern in cmd layer — usage tracking degrades silently if config dir unresolvable
- [Phase 03-extended-features]: Cycle detection fallback in BuildTree picks alphabetically first node as root when all nodes are in a mutual cycle
- [Phase 03-extended-features]: FormatDiff merges identical and different into single sorted Common section matching Python output format
- [Phase 03-extended-features]: ShareFiles uses filepath.EvalSymlinks to prevent chained symlinks
- [Phase 03-extended-features]: UnshareFiles with empty paths unshares all shared_paths — bulk materialization
- [Phase 03-extended-features]: share/pick/unshare cmd commands re-link active profile via DoSwitch(Force:true) after manifest mutation

### Pending Todos

None yet.

### Blockers/Concerns

- Windows atomic symlinks: `google/renameio/v2` does not export `Symlink` on Windows — document limitation in Phase 1 rather than discover in production
- Dual binary `go install` UX: two entry points needed (`cmd/hop` and `cmd/claudehopper`) or build-tag approach — resolve during Phase 4

## Session Continuity

Last session: 2026-03-15T00:54:29.382Z
Stopped at: Completed 03-extended-features-02-PLAN.md
Resume file: None
