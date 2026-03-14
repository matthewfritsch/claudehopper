# Phase 2: Core Profile Operations - Context

**Gathered:** 2026-03-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Full profile CRUD (create blank/from-current/from-profile with lineage), profile switching with atomic symlinks and safety features (dry-run, backup, manifest validation, adopt-on-switch), list, status, delete with dependent warning. Default linked files auto-shared on create. Format compatibility with Python version. This phase delivers enough for existing Python users to migrate.

</domain>

<decisions>
## Implementation Decisions

### Create from-current behavior
- Match Python's detection logic exactly — capture everything non-protected from `~/.claude/`
- Protected paths stay in `~/.claude/` and are shared across all profiles automatically (credentials, history, etc. are never moved — no re-login needed)
- Existing symlinks in `~/.claude/` are preserved as shared_paths in the new profile's manifest (not resolved to copies)
- Output format matches Python — show captured files and summary

### Switch & dry-run UX
- Normal switch output: Claude decides clean format
- Dry-run output matches Python's format (file-by-file action list showing what would change)

### Adopt-on-switch flow
- When unmanaged files found during switch: Claude decides safest UX (likely prompt user with list of files, offer to adopt into departing profile)
- Non-interactive mode (stdin not a TTY): skip adoption silently, just switch — safest for scripts and automation

### Profile name rules
- Claude decides validation rules (alphanumeric + hyphens + underscores as directory-safe names)
- Case-insensitive — normalize to lowercase to avoid confusion on macOS (case-insensitive filesystem)

### Claude's Discretion
- Normal switch output format (clean, one summary line or brief)
- Adopt-on-switch interactive prompt design
- Profile name validation rules (within the directory-safe constraint)
- Internal package structure for `internal/profile/`
- How `--activate` flag on create calls switch internally

</decisions>

<specifics>
## Specific Ideas

- "Match Python" was the recurring theme — user wants behavioral parity for migration comfort
- Protected paths clarification: they're "protected" by being left alone, not by being copied. Credentials persist across all profiles because they live in `~/.claude/` directly.
- Case-insensitive profile names to avoid macOS filesystem issues

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/fs/atomic.go`: `AtomicSymlink()` — use for all profile switch symlink operations
- `internal/fs/protected.go`: `IsProtected()`, `ProtectedPaths` — check before any file operation
- `internal/config/paths.go`: `ConfigDir()`, `ProfilesDir()`, `ProfileDir()` — all path resolution
- `internal/config/config.go`: `LoadConfig()`, `SaveConfig()` — active profile tracking
- `internal/config/manifest.go`: `LoadManifest()`, `SaveManifest()` — manifest CRUD with sorted ManagedPaths
- `cmd/root.go`: Cobra root command — add subcommands here

### Established Patterns
- stdlib testing with `testdata/` fixtures
- `os.Lstat` for symlink interrogation (not `os.Stat`)
- `SaveManifest` sorts `ManagedPaths` on write — callers can append freely
- `LoadConfig` returns zero Config for missing file (first-run safe)
- Colored output with TTY auto-detect (decided but not yet implemented)

### Integration Points
- New commands wire into `cmd/root.go` via `rootCmd.AddCommand()`
- Profile operations use `internal/config` for paths and `internal/fs` for safe symlink operations
- Python source at `~/Programming/claudehopper` is the behavioral specification for matching output

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 02-core-profile-operations*
*Context gathered: 2026-03-14*
