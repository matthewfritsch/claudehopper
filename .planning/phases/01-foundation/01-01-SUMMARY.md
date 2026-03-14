---
phase: 01-foundation
plan: "01"
subsystem: infra
tags: [go, cobra, cli, makefile, goreleaser, tdd]

# Dependency graph
requires: []
provides:
  - Compilable Go module at github.com/matthewfritsch/claudehopper
  - Cobra root command with --help and --version flags
  - Dual binary build: claudehopper + hop symlink via Makefile
  - goreleaser config producing claudehopper and hop for linux/darwin amd64/arm64
  - ldflags version injection (version/commit/date)
affects: [02-foundation, 03-foundation, 04-foundation, all-subsequent-phases]

# Tech tracking
tech-stack:
  added:
    - "github.com/spf13/cobra v1.10.2 — CLI framework with shell completion support"
    - "github.com/spf13/pflag v1.0.9 — POSIX-compliant flag parsing (cobra transitive dep)"
  patterns:
    - "ldflags version injection: -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"
    - "SetVersionInfo() called from main() before Execute() — version vars live in main package"
    - "TDD: failing test commit before implementation commit"

key-files:
  created:
    - go.mod
    - go.sum
    - main.go
    - cmd/root.go
    - cmd/root_test.go
    - Makefile
    - .goreleaser.yaml
  modified: []

key-decisions:
  - "Single main.go entry point (not two cmd/ packages) — goreleaser produces both binary names"
  - "bin/hop created as symlink to bin/claudehopper (not a separate binary) in Makefile"
  - "No windows builds in goreleaser — renameio/v2 does not support atomic symlinks on Windows"
  - "stdlib testing only — no testify dependency"

patterns-established:
  - "cmd/ package: rootCmd is unexported var, SetVersionInfo and Execute are the public surface"
  - "Version string format: '{version} (commit {commit}, built {date})'"
  - "Makefile LDFLAGS pattern reused for all future build targets"

requirements-completed: [DIST-02, DIST-04]

# Metrics
duration: 2min
completed: 2026-03-14
---

# Phase 1 Plan 01: Go module scaffold with Cobra CLI and dual binary build infrastructure

**Cobra CLI skeleton at github.com/matthewfritsch/claudehopper with ldflags version injection, Makefile producing claudehopper+hop symlink, and goreleaser config for linux/darwin dual binary releases**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-03-14T17:40:01Z
- **Completed:** 2026-03-14T17:41:30Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Go module initialized at github.com/matthewfritsch/claudehopper with Cobra v1.10.2
- Cobra root command with `--help` and `--version` flags; version string injected via ldflags
- Makefile with build/install/test/clean targets producing bin/claudehopper + bin/hop symlink
- .goreleaser.yaml with two build stanzas for linux/darwin amd64/arm64 (no Windows per renameio constraint)
- TDD cycle: failing tests committed before implementation

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: failing tests** - `011a612` (test)
2. **Task 1 GREEN: Go module, main.go, Cobra root command** - `2606917` (feat)
3. **Task 2: Makefile and goreleaser config** - `dddd6cc` (feat)

_Note: Task 1 used TDD — test commit before implementation commit._

## Files Created/Modified

- `go.mod` — Module declaration: github.com/matthewfritsch/claudehopper
- `go.sum` — Pinned cobra v1.10.2 and transitive deps
- `main.go` — Entry point: ldflags vars (version/commit/date), calls SetVersionInfo then Execute
- `cmd/root.go` — Cobra rootCmd (Use: "claudehopper"), SetVersionInfo, Execute
- `cmd/root_test.go` — 4 tests covering version format, non-empty, no (devel), Use field
- `Makefile` — build/install/test/clean with ldflags injection; bin/hop as symlink
- `.goreleaser.yaml` — Dual stanza (claudehopper + hop) for linux/darwin amd64/arm64

## Decisions Made

- **Single entry point:** One main.go, two binary names via goreleaser build stanzas and Makefile symlink — no separate cmd/hop package needed
- **bin/hop as symlink:** `ln -sf claudehopper bin/hop` in Makefile; both names route to same binary
- **No Windows builds:** goreleaser only targets linux and darwin due to renameio/v2 atomic symlink limitation on Windows
- **stdlib testing:** No testify or other test dependencies added

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Compilable Go module is ready for all subsequent plans
- `cmd/` package pattern established: add subcommands by registering to rootCmd
- Build tooling ready: `make build` for local, goreleaser for releases
- Next plan (01-02) can start implementing atomic symlink engine in `internal/fs/`

---
*Phase: 01-foundation*
*Completed: 2026-03-14*
