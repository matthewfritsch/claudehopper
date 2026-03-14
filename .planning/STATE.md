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

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Use Cobra for CLI framework (shell completions nearly free)
- Multi-file `cmd/` + `internal/` package structure (not monolithic)
- Maintain Python manifest/config format compatibility (fixture tests required before serialization)
- `google/renameio/v2` for atomic symlinks; never `os.Remove` + `os.Symlink`
- Use `os.Lstat` (not `os.Stat`) everywhere we interrogate managed symlinks

### Pending Todos

None yet.

### Blockers/Concerns

- Windows atomic symlinks: `google/renameio/v2` does not export `Symlink` on Windows — document limitation in Phase 1 rather than discover in production
- Dual binary `go install` UX: two entry points needed (`cmd/hop` and `cmd/claudehopper`) or build-tag approach — resolve during Phase 4

## Session Continuity

Last session: 2026-03-14
Stopped at: Roadmap created, STATE.md initialized — ready to plan Phase 1
Resume file: None
