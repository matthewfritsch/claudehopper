# Requirements: claudehopper-go

**Defined:** 2026-03-14
**Core Value:** Instant, safe profile switching for Claude Code configs — users can swap their entire Claude Code setup with a single command without risking credentials or shared data.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Profile Management

- [ ] **PROF-01**: User can create a blank profile with a name and optional description
- [ ] **PROF-02**: User can create a profile from their current `~/.claude/` config (`--from-current`)
- [ ] **PROF-03**: User can clone an existing profile (`--from-profile`) with lineage tracked in manifest
- [ ] **PROF-04**: User can list all profiles showing name, active marker, and managed path count
- [ ] **PROF-05**: User can view status of active profile with link health per managed path
- [ ] **PROF-06**: User can delete a profile with warning if other profiles depend on it
- [ ] **PROF-07**: User can create and immediately activate a profile (`--activate`)

### Profile Switching

- [ ] **SWCH-01**: User can switch active profile via single command
- [ ] **SWCH-02**: Switch uses atomic symlink replacement (tmp + rename, never remove + symlink)
- [ ] **SWCH-03**: User can preview switch with `--dry-run` before applying
- [ ] **SWCH-04**: Conflicting files are backed up with `.hop-backup` suffix before overwriting
- [ ] **SWCH-05**: Manifest is validated before switch (managed paths exist in profile dir)
- [ ] **SWCH-06**: Unmanaged files in `~/.claude/` are detected and offered for adoption on switch

### Safety & Compatibility

- [ ] **SAFE-01**: Protected paths (credentials, history, projects, cache) are never touched during any operation
- [ ] **SAFE-02**: Each profile has a `.hop-manifest.json` tracking managed_paths, shared_paths, description, created_from
- [ ] **SAFE-03**: Manifest and config.json formats are compatible with the Python claudehopper version

### File Sharing

- [ ] **SHAR-01**: User can symlink files between profiles (`hop share`)
- [ ] **SHAR-02**: User can copy files between profiles independently (`hop pick`)
- [ ] **SHAR-03**: User can materialize shared symlinks back to independent copies (`hop unshare`)
- [ ] **SHAR-04**: New profiles automatically share default linked files (settings.json, settings.local.json, .mcp.json)

### Visualization & Analysis

- [ ] **VIZ-01**: User can view profile lineage tree (`hop tree`) with optional `--json` output
- [ ] **VIZ-02**: User can compare two profiles side-by-side (`hop diff`)
- [ ] **VIZ-03**: User can view usage statistics (`hop stats`) with optional `--json` output
- [ ] **VIZ-04**: User can print a profile's directory path for scripting (`hop path <name>`)

### Operations

- [ ] **OPS-01**: User can stop using claudehopper by materializing all symlinks (`hop unmanage`)
- [ ] **OPS-02**: Tool checks for updates from GitHub releases with 24h cached TTL
- [ ] **OPS-03**: All profile actions are logged to `usage.jsonl` for statistics

### Distribution & UX

- [ ] **DIST-01**: Shell tab completions work for bash, zsh, fish, and powershell via Cobra
- [ ] **DIST-02**: Tool installs as both `hop` and `claudehopper` binary names
- [ ] **DIST-03**: Tool distributable via `go install` and prebuilt binaries (goreleaser)
- [ ] **DIST-04**: Every subcommand has `--help` and root has `--version`

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
| PROF-01 | Phase 2 | Pending |
| PROF-02 | Phase 2 | Pending |
| PROF-03 | Phase 2 | Pending |
| PROF-04 | Phase 2 | Pending |
| PROF-05 | Phase 2 | Pending |
| PROF-06 | Phase 2 | Pending |
| PROF-07 | Phase 2 | Pending |
| SWCH-01 | Phase 2 | Pending |
| SWCH-02 | Phase 2 | Pending |
| SWCH-03 | Phase 2 | Pending |
| SWCH-04 | Phase 2 | Pending |
| SWCH-05 | Phase 2 | Pending |
| SWCH-06 | Phase 2 | Pending |
| SAFE-01 | Phase 1 | Pending |
| SAFE-02 | Phase 2 | Pending |
| SAFE-03 | Phase 1 | Pending |
| SHAR-01 | Phase 3 | Pending |
| SHAR-02 | Phase 3 | Pending |
| SHAR-03 | Phase 3 | Pending |
| SHAR-04 | Phase 2 | Pending |
| VIZ-01 | Phase 3 | Pending |
| VIZ-02 | Phase 3 | Pending |
| VIZ-03 | Phase 3 | Pending |
| VIZ-04 | Phase 3 | Pending |
| OPS-01 | Phase 3 | Pending |
| OPS-02 | Phase 4 | Pending |
| OPS-03 | Phase 3 | Pending |
| DIST-01 | Phase 4 | Pending |
| DIST-02 | Phase 1 | Pending |
| DIST-03 | Phase 4 | Pending |
| DIST-04 | Phase 1 | Pending |

**Coverage:**
- v1 requirements: 31 total
- Mapped to phases: 31
- Unmapped: 0

---
*Requirements defined: 2026-03-14*
*Last updated: 2026-03-14 after roadmap creation*
