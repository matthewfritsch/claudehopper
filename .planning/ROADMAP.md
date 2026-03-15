# Roadmap: claudehopper-go

## Overview

claudehopper-go ports a working Python CLI tool to Go, delivering instant symlink-based Claude Code config profile switching. The build order is dictated by dependency: a safe filesystem foundation must exist before profile operations can be built on it, extended features build on a proven CRUD core, and distribution polish comes last. Four phases take a blank Go module to a releasable binary with full feature parity and goreleaser distribution.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation** - Go module scaffold, Cobra CLI skeleton, atomic symlink engine, config/path resolution, and protected-paths enforcement (completed 2026-03-14)
- [ ] **Phase 2: Core Profile Operations** - Full profile CRUD, switch with atomic replacement and safety features, manifest format compatibility with Python version
- [ ] **Phase 3: Extended Features** - File sharing, profile visualization, usage tracking, and the unmanage escape hatch
- [ ] **Phase 4: Polish & Distribution** - Update checking, goreleaser dual-binary release, shell completions verification, and version injection

## Phase Details

### Phase 1: Foundation
**Goal**: A compilable, testable Go module exists with the load-bearing infrastructure that all profile operations depend on
**Depends on**: Nothing (first phase)
**Requirements**: SAFE-01, SAFE-03, DIST-02, DIST-04
**Plans:** 3/3 plans complete

Plans:
- [ ] 01-01-PLAN.md — Go module scaffold, Cobra CLI skeleton, and dual-binary build tooling
- [ ] 01-02-PLAN.md — Atomic symlink engine and protected-paths enforcement
- [ ] 01-03-PLAN.md — Config path resolution and Python-compatible JSON serialization

**Success Criteria** (what must be TRUE):
  1. `hop --help` and `claudehopper --help` both print usage from a single compiled binary
  2. `hop --version` prints a version string (not `(devel)`)
  3. `internal/fs.AtomicSymlink()` creates and replaces symlinks without ever leaving a broken state mid-operation, verified by tests using t.TempDir()
  4. Protected paths (credentials, history, projects, cache) are enforced by `internal/fs.IsProtected()` and match the Python version's constants exactly, verified by a fixture test
  5. Config path resolves correctly under both default `~/.config/claudehopper/` and `XDG_CONFIG_HOME` override, with no tilde strings stored anywhere

### Phase 2: Core Profile Operations
**Goal**: An existing Python claudehopper user can migrate and perform all daily profile management tasks with the Go binary
**Depends on**: Phase 1
**Requirements**: PROF-01, PROF-02, PROF-03, PROF-04, PROF-05, PROF-06, PROF-07, SWCH-01, SWCH-02, SWCH-03, SWCH-04, SWCH-05, SWCH-06, SAFE-02, SHAR-04
**Plans:** 2/4 plans executed

Plans:
- [ ] 02-01-PLAN.md — Manifest CreatedFrom fix, profile name validation, and create logic (blank/from-current/from-profile) with shared defaults
- [ ] 02-02-PLAN.md — List profiles, profile status with link health, and delete with dependent warnings
- [ ] 02-03-PLAN.md — Switch engine: preflight validation, backup, atomic link, dry-run, and unmanaged file detection
- [ ] 02-04-PLAN.md — Cobra command wiring for create, list, status, switch, delete with human verification

**Success Criteria** (what must be TRUE):
  1. User can create a blank profile, a profile copied from current `~/.claude/`, and a profile cloned from an existing profile — all with lineage recorded in the manifest
  2. User can switch profiles with a single command; symlinks update atomically and conflicting files are backed up with `.hop-backup` suffix before being overwritten
  3. User can preview a switch with `--dry-run` and see exactly what would change before anything is written
  4. Unmanaged files in `~/.claude/` are detected on switch and offered for adoption rather than silently abandoned
  5. A manifest written by the Python version can be read by the Go version and vice versa with no data loss (fixture tests pass)

### Phase 3: Extended Features
**Goal**: Users have full file-sharing between profiles, rich visualization commands, usage tracking, and a clean exit ramp from the tool
**Depends on**: Phase 2
**Requirements**: SHAR-01, SHAR-02, SHAR-03, VIZ-01, VIZ-02, VIZ-03, VIZ-04, OPS-01, OPS-03
**Success Criteria** (what must be TRUE):
  1. User can symlink a file from one profile into another (`hop share`), copy it independently (`hop pick`), and materialize shared symlinks back to independent files (`hop unshare`)
  2. User can view a profile lineage tree (`hop tree`) with parent-child relationships, compare two profiles side-by-side (`hop diff`), and print a profile's directory path for scripting (`hop path <name>`)
  3. `hop stats` shows accurate usage data drawn from `usage.jsonl` entries that are appended on every profile action
  4. User can stop using the tool entirely with `hop unmanage`, which materializes all symlinks to real files and leaves `~/.claude/` in a self-contained state
**Plans**: TBD

### Phase 4: Polish & Distribution
**Goal**: The tool is releasable: versioned binaries for all platforms, shell completions verified, and update checking working
**Depends on**: Phase 3
**Requirements**: OPS-02, DIST-01, DIST-03
**Success Criteria** (what must be TRUE):
  1. Shell tab completions work in bash, zsh, fish, and powershell via `hop completion <shell>`
  2. goreleaser produces both `hop` and `claudehopper` binaries for Linux/macOS/Windows on amd64 and arm64 from a single release workflow
  3. `hop update` checks GitHub releases for a newer version with a 24-hour cached TTL and prints a non-blocking upgrade notice if one is available
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation | 3/3 | Complete    | 2026-03-14 |
| 2. Core Profile Operations | 2/4 | In Progress|  |
| 3. Extended Features | 0/TBD | Not started | - |
| 4. Polish & Distribution | 0/TBD | Not started | - |
