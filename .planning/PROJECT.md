# claudehopper-go

## What This Is

A Go rewrite of claudehopper — a CLI tool for managing multiple Claude Code configuration profiles through symlink-based switching. Users maintain separate configs (CLAUDE.md, commands, plugins, etc.) for different contexts (work, personal, experimental) and switch between them instantly. The Go version ports all core features while improving code structure and adding shell completions.

## Core Value

Instant, safe profile switching for Claude Code configs — users can swap their entire Claude Code setup with a single command without risking credentials or shared data.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Profile creation (blank, from-current, from-profile with lineage tracking)
- [ ] Profile switching via symlinks with manifest validation
- [ ] Protected paths (credentials, history, projects, cache) never touched
- [ ] File sharing between profiles (share/pick/unshare)
- [ ] Default linked files (settings.json, settings.local.json, .mcp.json)
- [ ] Adopt-on-switch for unmanaged files
- [ ] Profile visualization (tree with lineage, diff, stats, status)
- [ ] Dry-run and force modes for switch
- [ ] Atomic symlink creation
- [ ] Backup system for conflicting files
- [ ] Update checking from GitHub releases
- [ ] Usage tracking (usage.jsonl)
- [ ] Shell tab completions (bash, zsh, fish, powershell) via Cobra
- [ ] Dual binary names: `hop` and `claudehopper`
- [ ] Distribution via `go install` and prebuilt binaries (goreleaser)

### Out of Scope

- GUI or TUI interface — this is a CLI tool
- Config file format changes — must remain compatible with existing claudehopper profiles
- Claude Code integration beyond config management — we manage files, not Claude itself

## Context

- Original Python version lives at ~/Programming/claudehopper (~1500 lines, single-file, zero dependencies)
- Go version should have cleaner multi-file structure (cmd/, internal/ packages)
- Uses Cobra for CLI framework, which gives us subcommand routing and shell completions for free
- Must maintain the same directory layout: `~/.config/claudehopper/profiles/<name>/`, `~/.config/claudehopper/config.json`, etc.
- Manifest format (`.hop-manifest.json`) must remain compatible
- Protected paths and default linked files lists must match the Python version exactly

## Constraints

- **Language**: Go — the whole point of this project
- **CLI Framework**: Cobra — industry standard for Go CLIs
- **Compatibility**: Must read/write the same config/manifest formats as the Python version so users can switch between versions
- **Distribution**: go install + goreleaser for GitHub Releases binaries

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Use Cobra for CLI | Industry standard, built-in completions, good subcommand support | — Pending |
| Multi-file structure | Python version is monolithic; Go idiom favors packages | — Pending |
| Maintain format compatibility | Users may have existing profiles from Python version | — Pending |
| Dual binary names (hop + claudehopper) | Matches Python version UX | — Pending |

---
*Last updated: 2026-03-14 after initialization*
