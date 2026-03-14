# Phase 1: Foundation - Context

**Gathered:** 2026-03-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Go module scaffold, Cobra CLI skeleton, atomic symlink engine, config/path resolution, and protected-paths enforcement. This phase produces a compilable, testable binary with infrastructure that all profile operations depend on. No user-facing profile features — those come in Phase 2.

</domain>

<decisions>
## Implementation Decisions

### Dual binary strategy
- Primary binary name: `claudehopper` (this is what `go install` produces)
- Alias binary: `hop` — created via install mechanism (not a separate main package)
- Module path: `github.com/matthewfritsch/claudehopper`
- Single `main.go` entry point, not two `cmd/` packages
- goreleaser produces both names in release binaries
- For source installs, Claude decides alias mechanism (Makefile or similar)

### Testing approach
- stdlib `testing` package only — no testify or other test dependencies
- Test fixtures in `testdata/` directories alongside test files (Go convention)
- Real JSON files from Python version stored as fixtures for format compatibility checks
- No automated cross-version fixture extraction — visual verification of Python constants is sufficient
- Claude decides test thoroughness per component based on risk level

### Error & output style
- Colored output with automatic TTY detection (colors when terminal, plain when piped)
- Claude decides color library (fatih/color or similar) and exact error format
- No --verbose or --quiet flags for now — keep it simple
- Clean, consistent error messages to stderr with appropriate exit codes

### Claude's Discretion
- Package boundaries within internal/ (fs/, config/, profile/, updater/ split)
- Color library choice
- Error message formatting style
- Test thoroughness per component (unit vs integration based on risk)
- Alias creation mechanism for `hop` from source builds

</decisions>

<specifics>
## Specific Ideas

- User wants "cleaner code" than the Python monolith — multi-file structure is a key motivation for the Go rewrite
- Shell completions via Cobra are an explicit goal of the rewrite (nearly free with Cobra)
- Format compatibility with Python version is a hard constraint — same directory layout, same manifest schema

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- None — greenfield Go project

### Established Patterns
- Python source at ~/Programming/claudehopper serves as the behavioral specification
- Python version's constants (SHARED_PATHS, DEFAULT_LINKED) are the source of truth for protected paths and default linked files

### Integration Points
- `~/.config/claudehopper/` directory structure must match Python version exactly
- `.hop-manifest.json` format must be round-trip compatible with Python version

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 01-foundation*
*Context gathered: 2026-03-14*
