# Phase 3: Extended Features - Context

**Gathered:** 2026-03-15
**Status:** Ready for planning

<domain>
## Phase Boundary

File sharing between profiles (share/pick/unshare), profile visualization (tree with lineage, diff, stats, path), usage tracking (usage.jsonl), and unmanage exit ramp. All build on the working CRUD and switch from Phase 2.

</domain>

<decisions>
## Implementation Decisions

### Tree visualization
- Enhanced tree: show shared file indicators and profile sizes alongside lineage (parent-child from created_from)
- Charmbracelet libraries (lipgloss for styling) are acceptable if Claude determines the dependency is worth it for tree/diff/stats output — user is open to it
- If Charm is too heavy or not justified, plain ASCII with color (fatih/color or similar) is fine
- `--json` output: Claude decides schema (richer is better — include managed_paths counts, shared files, etc.)

### Diff output format
- Claude decides comparison scope (set operations on paths and/or byte-level content)
- Claude decides display style — can use Charm libraries if they add value
- No specific Python parity requirement here — user has no preference

### Usage tracking
- Claude decides which actions to log to usage.jsonl (at minimum: switch, create, delete)
- Optional: if fingerprint data from .claude/ could enrich stats (e.g., last active time per profile), Claude can incorporate it — but not required
- `hop stats` display: Claude decides useful analytics (switch counts, last-used, profile breakdown are baseline)

### Claude's Discretion
- Whether to add charmbracelet/lipgloss as a dependency (evaluate effort vs visual payoff)
- Diff comparison depth (paths only vs content diff)
- Usage tracking action scope
- Stats display format and filtering options
- Share/pick/unshare implementation details (match Python behavior)
- Unmanage implementation (materialize symlinks, clean config)
- `hop path` implementation (trivial — print profile dir)

</decisions>

<specifics>
## Specific Ideas

- User is open to Charm libraries for prettier terminal output (lipgloss, not bubbletea TUI)
- User mentioned fingerprint data from .claude/ as a possible stats enrichment — optional, not required
- Share/pick/unshare and unmanage should match Python behavior by default

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/profile/list.go`: `ListProfiles()` returns `[]ProfileSummary` — reuse for tree data source
- `internal/profile/status.go`: `GetProfileStatus()` returns link health — reuse for tree node annotations
- `internal/profile/shared.go`: `DefaultLinked`, `SharedDir()`, `LinkDefaultsIntoProfile()` — share/unshare build on this
- `internal/config/manifest.go`: `Manifest.SharedPaths`, `Manifest.CreatedFrom` — tree lineage source
- `internal/profile/delete.go`: `FindDependents()` — tree can reuse dependency scanning
- `cmd/helpers.go`: `isInteractive()`, `claudeDir()` — reuse in new commands

### Established Patterns
- Business logic in `internal/profile/`, thin Cobra wrappers in `cmd/`
- `os.Lstat` for symlink interrogation
- stdlib testing with `testdata/` fixtures
- Case-insensitive profile names

### Integration Points
- New commands register via `rootCmd.AddCommand()` in `cmd/`
- Usage tracking needs to be called from existing commands (switch, create, delete) retroactively
- Share/pick/unshare modify manifests — use existing `LoadManifest`/`SaveManifest`

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 03-extended-features*
*Context gathered: 2026-03-15*
