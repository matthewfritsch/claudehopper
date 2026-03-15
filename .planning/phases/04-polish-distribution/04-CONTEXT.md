# Phase 4: Polish & Distribution - Context

**Gathered:** 2026-03-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Update checking from GitHub releases with 24h cached TTL, goreleaser dual-binary release config for all target platforms, shell completions verification across all four shells, version injection via ldflags, and Homebrew tap setup. This phase makes the tool releasable.

</domain>

<decisions>
## Implementation Decisions

### Update checking UX
- Update notice appears only after `hop status` command — least intrusive placement
- `hop update` auto-installs the new version via `go install` (for source installs) or downloads binary (for binary installs)
- 24h cached TTL for GitHub release checks (already decided in requirements)
- Non-blocking — never slow down normal commands

### Release targets
- Linux amd64 and arm64
- macOS amd64 and arm64 (Intel + Apple Silicon)
- No Windows builds (renameio/v2 doesn't support atomic symlinks on Windows — decided in Phase 1)
- GitHub Releases for binary distribution
- Homebrew tap (`homebrew-claudehopper`) for macOS/Linux package management
- No AUR package for now

### Claude's Discretion
- `go-selfupdate` library configuration details
- Shell completions verification approach (manual testing vs automated)
- Homebrew formula structure and tap repository setup
- Goreleaser archive format preferences (tar.gz vs zip per platform)
- CI/CD workflow if needed (GitHub Actions for goreleaser)

</decisions>

<specifics>
## Specific Ideas

- Goreleaser already has dual build stanzas from Phase 1 — this phase refines and validates the config
- Cobra already generates completions — this phase verifies they work across bash/zsh/fish/powershell
- Version string already uses ldflags injection (`-X main.version=...`) — this phase ensures goreleaser sets it correctly

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `.goreleaser.yaml` — already has dual build config from Phase 1
- `cmd/root.go` — has `rootCmd.Version` wired to ldflags variable
- `Makefile` — has build targets and hop symlink
- `internal/usage/usage.go` — could record update check events
- `cmd/status.go` — where update notice will be shown

### Established Patterns
- Business logic in `internal/`, thin Cobra wrappers in `cmd/`
- `go-selfupdate` (creativeprojects/go-selfupdate v1.5.2) recommended by project research

### Integration Points
- Update notice hooks into `cmd/status.go` (run after status display)
- `internal/updater/` new package for update checking logic
- Goreleaser config needs validation with `goreleaser check`
- Homebrew tap is a separate repository (`homebrew-claudehopper`)

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 04-polish-distribution*
*Context gathered: 2026-03-15*
