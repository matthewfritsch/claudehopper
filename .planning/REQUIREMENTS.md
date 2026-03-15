# Requirements: claudehopper-go

**Defined:** 2026-03-14
**Core Value:** Instant, safe profile switching for Claude Code configs — users can swap their entire Claude Code setup with a single command without risking credentials or shared data.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Profile Management

- [x] **PROF-01**: User can create a blank profile with a name and optional description
- [x] **PROF-02**: User can create a profile from their current `~/.claude/` config (`--from-current`)
- [x] **PROF-03**: User can clone an existing profile (`--from-profile`) with lineage tracked in manifest
- [x] **PROF-04**: User can list all profiles showing name, active marker, and managed path count
- [x] **PROF-05**: User can view status of active profile with link health per managed path
- [x] **PROF-06**: User can delete a profile with warning if other profiles depend on it
- [x] **PROF-07**: User can create and immediately activate a profile (`--activate`)

### Profile Switching

- [x] **SWCH-01**: User can switch active profile via single command
- [x] **SWCH-02**: Switch uses atomic symlink replacement (tmp + rename, never remove + symlink)
- [x] **SWCH-03**: User can preview switch with `--dry-run` before applying
- [x] **SWCH-04**: Conflicting files are backed up with `.hop-backup` suffix before overwriting
- [x] **SWCH-05**: Manifest is validated before switch (managed paths exist in profile dir)
- [x] **SWCH-06**: Unmanaged files in `~/.claude/` are detected and offered for adoption on switch

### Safety & Compatibility

- [x] **SAFE-01**: Protected paths (credentials, history, projects, cache) are never touched during any operation
- [x] **SAFE-02**: Each profile has a `.hop-manifest.json` tracking managed_paths, shared_paths, description, created_from
- [x] **SAFE-03**: Manifest and config.json formats are compatible with the Python claudehopper version

### File Sharing

- [x] **SHAR-01**: User can symlink files between profiles (`hop share`)
- [x] **SHAR-02**: User can copy files between profiles independently (`hop pick`)
- [x] **SHAR-03**: User can materialize shared symlinks back to independent copies (`hop unshare`)
- [x] **SHAR-04**: New profiles automatically share default linked files (settings.json, settings.local.json, .mcp.json)

### Visualization & Analysis

- [x] **VIZ-01**: User can view profile lineage tree (`hop tree`) with optional `--json` output
- [x] **VIZ-02**: User can compare two profiles side-by-side (`hop diff`)
- [x] **VIZ-03**: User can view usage statistics (`hop stats`) with optional `--json` output
- [x] **VIZ-04**: User can print a profile's directory path for scripting (`hop path <name>`)

### Operations

- [x] **OPS-01**: User can stop using claudehopper by materializing all symlinks (`hop unmanage`)
- [x] **OPS-02**: Tool checks for updates from GitHub releases with 24h cached TTL
- [x] **OPS-03**: All profile actions are logged to `usage.jsonl` for statistics

### Distribution & UX

- [x] **DIST-01**: Shell tab completions work for bash, zsh, fish, and powershell via Cobra
- [x] **DIST-02**: Tool installs as both `hop` and `claudehopper` binary names
- [x] **DIST-03**: Tool distributable via `go install` and prebuilt binaries (goreleaser)
- [x] **DIST-04**: Every subcommand has `--help` and root has `--version`

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Distribution

- **DIST-05**: AI agent setup guide documentation (Go-version equivalent of docs/claude-setup-guide.md)

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| TUI / interactive menu | Violates single-command mental model; adds bubbletea dependency |
| Cloud sync of profiles | Out of scope; pushes into credentials management territory |
| Automatic profile detection by directory | Requires shell hooks; fragile across shells |
| Profile locking during Claude Code session | Process detection unreliable; partial lock worse than no lock |
| Profile inheritance / merge semantics | Lineage tracking + share already covers the use case |
| Profile encryption | Claude Code config generally not sensitive; protected paths handles credentials |
| Config file format changes | Must remain compatible with existing Python version profiles |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| PROF-01 | Phase 2 | Complete |
| PROF-02 | Phase 2 | Complete |
| PROF-03 | Phase 2 | Complete |
| PROF-04 | Phase 2 | Complete |
| PROF-05 | Phase 2 | Complete |
| PROF-06 | Phase 2 | Complete |
| PROF-07 | Phase 2 | Complete |
| SWCH-01 | Phase 2 | Complete |
| SWCH-02 | Phase 2 | Complete |
| SWCH-03 | Phase 2 | Complete |
| SWCH-04 | Phase 2 | Complete |
| SWCH-05 | Phase 2 | Complete |
| SWCH-06 | Phase 2 | Complete |
| SAFE-01 | Phase 1 | Complete |
| SAFE-02 | Phase 2 | Complete |
| SAFE-03 | Phase 1 | Complete |
| SHAR-01 | Phase 3 | Complete |
| SHAR-02 | Phase 3 | Complete |
| SHAR-03 | Phase 3 | Complete |
| SHAR-04 | Phase 2 | Complete |
| VIZ-01 | Phase 3 | Complete |
| VIZ-02 | Phase 3 | Complete |
| VIZ-03 | Phase 3 | Complete |
| VIZ-04 | Phase 3 | Complete |
| OPS-01 | Phase 3 | Complete |
| OPS-02 | Phase 4 | Complete |
| OPS-03 | Phase 3 | Complete |
| DIST-01 | Phase 4 | Complete |
| DIST-02 | Phase 1 | Complete |
| DIST-03 | Phase 4 | Complete |
| DIST-04 | Phase 1 | Complete |

**Coverage:**
- v1 requirements: 31 total
- Mapped to phases: 31
- Unmapped: 0

---
*Requirements defined: 2026-03-14*
*Last updated: 2026-03-14 after roadmap creation*
