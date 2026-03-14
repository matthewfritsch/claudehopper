# Project Research Summary

**Project:** claudehopper-go
**Domain:** Go CLI tool — symlink-based config profile manager for Claude Code
**Researched:** 2026-03-14
**Confidence:** HIGH

## Executive Summary

claudehopper-go is a port of a working Python CLI tool that manages Claude Code configuration profiles using filesystem symlinks. The core value proposition is instant, atomic profile switching: a single `hop switch <name>` command repoints `~/.claude/` symlinks from one profile directory to another. Research was conducted with HIGH confidence because the Python source (~1480 lines) was directly available as the authoritative spec, and all major Go libraries (Cobra, renameio, goreleaser) have stable, well-documented APIs. The recommended approach is a clean `cmd/` + `internal/` package separation using Cobra with the NewCommand() constructor pattern, with all filesystem mutation routed through a single `internal/fs` package that enforces atomic symlink replacement.

The highest-risk area is the symlink engine itself: three filesystem pitfalls (non-atomic replacement, cross-device rename, os.Stat vs os.Lstat confusion) must be solved in the foundation phase before any higher-level feature is built on top. A secondary risk is JSON format compatibility — Go's `omitempty` behavior will silently break round-trip compatibility with Python-generated manifests if not proactively prevented with fixture tests. Both risks are well-understood and have straightforward mitigations; neither should block the project.

The recommended phase structure follows the dependency graph from FEATURES.md and the build order from ARCHITECTURE.md: foundation first (fs package + config/manifest), then the core switch/create/list cycle, then the full v1 feature set, then polish and distribution. Shell completions are nearly free via Cobra and should be included in the core phase, not deferred. The tool has no significant performance, scalability, or architectural uncertainty — the main challenge is careful porting discipline.

## Key Findings

### Recommended Stack

The stack is narrow and well-validated. Go 1.25 is current stable. Cobra v1.10.2 is the industry-standard CLI framework and handles shell completions across all four target shells (bash/zsh/fish/powershell) automatically — this covers the shell completions requirement at near-zero implementation cost. `google/renameio/v2` is the correct library for atomic symlink replacement; it encapsulates the tmp+rename pattern and is the only actively maintained Go library built specifically for this problem. `creativeprojects/go-selfupdate v1.5.2` handles update checking from GitHub releases and is the actively maintained fork of the more well-known but stale rhysd version.

All other needs are met by the Go standard library: `encoding/json` for config/manifest serialization, `os`/`path/filepath` for all other filesystem operations. No CGO dependencies are needed or desirable — CGO breaks cross-compilation and `go install`. Avoid `spf13/viper` (over-engineering for a tool that owns its own config format) and any TUI library (non-interactive CLI, out of scope).

**Core technologies:**
- Go 1.25: Language runtime — current stable, `os.Root` type in 1.24+ is useful for path validation
- spf13/cobra v1.10.2: CLI framework + shell completions — industry standard, used by Kubernetes/Docker/GitHub CLI
- google/renameio/v2: Atomic symlink creation/replacement — only Go library purpose-built for this; 5,000+ dependents
- creativeprojects/go-selfupdate v1.5.2: Update checking from GitHub releases — actively maintained through Dec 2025
- goreleaser v2.14.3: Cross-platform binary distribution — supports dual binary names from one main package
- encoding/json + os + path/filepath (stdlib): All serialization and filesystem operations — stdlib is sufficient

### Expected Features

The feature set is well-specified by the Python version. The v1 scope is clear: get existing Python users migrated without losing profiles or workflow. The top priority is format compatibility (same `config.json` and `.hop-manifest.json` schemas) because breaking that would require existing users to rebuild their profiles from scratch.

**Must have (table stakes — v1):**
- Profile create (blank, --from-current, --from-profile) — cannot adopt the tool without onboarding
- Profile switch with atomic symlinks, dry-run, and backup-on-conflict — the core value proposition
- Protected paths enforcement — non-negotiable; credentials must never be touched
- Manifest read/write in Python-compatible format — required for migration compatibility
- Profile list and status — basic observability
- Profile delete with dependent-profile warning — housekeeping
- Default linked files (settings.json, .mcp.json shared across all profiles) — prevents the most common footgun
- Adopt-on-switch for unmanaged files — prevents silent data loss on first switch
- Shell completions (bash/zsh/fish/powershell via Cobra) — low cost, high value, explicit project goal
- Dual binary names (hop + claudehopper) — existing muscle memory

**Should have (v1.x — after core is validated):**
- Profile share/pick/unshare — file-level sharing between profiles
- Profile diff — compare two profiles' managed paths
- Profile tree with lineage visualization — ASCII parent-child relationship tree
- Usage tracking (usage.jsonl) + stats command
- Profile path command (scripting bridge: `cd $(hop path work)`)
- Unmanage command (exit ramp: materialize all symlinks, stop managing)
- Update checking (cached GitHub release check, 24h TTL, non-blocking)
- --json output flags on tree and stats

**Defer (v2+):**
- goreleaser config and prebuilt binaries for non-Go users (distribution infrastructure)
- Documentation artifacts (Go-version setup guide)

**Anti-features (do not build):**
- TUI/interactive menus — violates single-command mental model, adds bubbletea dependency
- Cloud sync of profiles — out of scope, pushes into credentials management
- Automatic profile detection based on directory — requires shell hooks, fragile
- Profile locking during Claude Code session — process detection unreliable
- True profile inheritance/merge semantics — lineage tracking + share covers the use case

### Architecture Approach

The architecture is a clean three-layer Go CLI: `cmd/` for Cobra command wiring, `internal/` for all domain logic, and a storage layer of JSON files and symlinks under `~/.config/claudehopper/`. The critical structural rule is that `cmd/` files must not contain business logic — `RunE` only parses flags, validates arguments, calls `internal/`, and formats output. All state mutation lives in `internal/profile/`. All filesystem operations go through `internal/fs/`, which is the only package allowed to call `os.Symlink`. This boundary is what makes atomic replacement enforceable.

**Major components:**
1. `cmd/` (14 command files + root) — Cobra wiring; NewCommand() constructor pattern; no business logic
2. `internal/config/` — AppConfig struct, Load/Save, path constants, DEFAULT_LINKED and SHARED_PATHS
3. `internal/fs/` — AtomicSymlink(), BackupPath(), IsProtected(); the only code that touches symlinks
4. `internal/profile/` — Profile CRUD, manifest load/save, switch logic, adopt-on-switch, share/pick/unshare
5. `internal/updater/` — GitHub release check, version comparison, cached timestamp

The **build order** that the dependency graph mandates: `internal/config/` first (zero deps, establishes all constants), then `internal/fs/` (stdlib only, testable immediately with t.TempDir()), then `internal/profile/` data structures (Manifest + Profile structs), then `cmd/` skeleton (root + list + status), then `internal/profile/` mutations (create, switch, share), then remaining commands, then `internal/updater/`.

### Critical Pitfalls

1. **Non-atomic symlink replacement** — never use `os.Remove` + `os.Symlink` for managed paths; always use `fs.AtomicSymlink()` (tmp in same dir + `os.Rename`). Address in Phase 1; test with SIGINT mid-switch.

2. **JSON omitempty breaks Python compatibility** — do not use `omitempty` on any struct field that corresponds to an on-disk format field; write fixture tests using real Python-generated JSON before implementing serialization. Address in Phase 2.

3. **os.Stat vs os.Lstat confusion** — `os.Stat` follows symlinks; `os.Lstat` does not. Use `os.Lstat` everywhere the code interrogates whether a path is a managed symlink. Address in Phase 1; add grep check to "looks done but isn't" checklist.

4. **Tilde and XDG_CONFIG_HOME not expanded by Go stdlib** — never store or accept paths with `~`; always resolve via `os.UserConfigDir()` at startup to an absolute path. Address in Phase 1 config resolution function.

5. **Protected paths list divergence from Python version** — extract the Python constants into a test fixture; assert the Go constants match exactly; treat any change as a breaking change. Address in Phase 2.

## Implications for Roadmap

Based on the dependency graph in FEATURES.md and build order in ARCHITECTURE.md, four phases are suggested:

### Phase 1: Foundation
**Rationale:** Three critical pitfalls (non-atomic symlinks, Stat/Lstat confusion, path resolution) must be solved before any higher-level feature is reliable. The symlink engine, config loading, and Cobra scaffolding are zero-feature-value but load-bearing. Everything else depends on getting this right.
**Delivers:** Compilable binary with root command + list/status stubs; `internal/fs` with AtomicSymlink/BackupPath/IsProtected tested; `internal/config` with path resolution tested under XDG_CONFIG_HOME override; protected paths constants tested against Python fixture.
**Addresses from FEATURES.md:** Protected paths enforcement, atomic symlink creation, dual binary names (goreleaser config), shell completions scaffold
**Avoids:** Pitfall 1 (non-atomic symlink), Pitfall 3 (Stat/Lstat), Pitfall 5 (tilde/XDG), Pitfall 6 (protected paths divergence)

### Phase 2: Core Profile Operations
**Rationale:** Manifest read/write (format-compatible) is the next dependency because create, switch, list, status, and delete all depend on it. This phase delivers the minimum needed for an existing Python user to migrate. JSON fixture tests must be written before any serialization logic.
**Delivers:** `hop create`, `hop switch` (with dry-run, backup-on-conflict, adopt-on-switch), `hop list`, `hop status`, `hop delete`; manifest round-trip fixture tests passing; format compatibility with Python version verified.
**Uses:** `google/renameio/v2` for atomic symlinks in switch path; `encoding/json` stdlib with no omitempty on format fields
**Avoids:** Pitfall 2 (omitempty JSON compat), Pitfall 4 (protected paths list)
**Implements:** `internal/profile/` mutations (create, switch); `internal/profile/manifest.go`

### Phase 3: Extended Feature Set
**Rationale:** share/pick/unshare, diff, tree, stats, path, and unmanage all depend on working create/switch/manifest from Phase 2. These features follow clearly from the Python source and have no new architectural uncertainty. Usage tracking (usage.jsonl) is append-only and low risk.
**Delivers:** `hop share`, `hop pick`, `hop unshare`, `hop diff`, `hop tree`, `hop stats`, `hop path`, `hop unmanage`; `--json` output flags; usage.jsonl append-only tracking
**Uses:** All `internal/profile/` mutation packages from Phase 2 as foundation

### Phase 4: Polish and Distribution
**Rationale:** Update checking, dual binary goreleaser config, version string injection via ldflags, and completions wiring are all self-contained and do not block any other phase. The goreleaser dual-binary pitfall (two `builds` entries required, not one entry with two binary values) is documented and straightforward to avoid.
**Delivers:** `hop update` command; `internal/updater/` with 24h TTL cache and graceful 403/429 handling; goreleaser config producing both `hop` and `claudehopper` binaries for Linux/macOS/Windows amd64/arm64; `hop --version` printing real version from ldflags; shell completions verified across all four shells
**Avoids:** goreleaser dual binary misconfiguration pitfall; version string `(devel)` pitfall

### Phase Ordering Rationale

- Phase 1 before Phase 2 because three critical pitfalls are filesystem-level and would corrupt all higher-level features if deferred
- Phase 2 before Phase 3 because every extended feature depends on manifest read/write and profile CRUD
- Phase 4 last because update checking and distribution are independent of all domain logic
- Shell completions are wired during Phase 1 (Cobra scaffold) and verified in Phase 4 — Cobra generates them for free so this does not add a separate phase
- adopt-on-switch belongs in Phase 2 (not Phase 3) because it fires during switch and silent data loss on first adoption would be a trust-destroying bug

### Research Flags

Phases with standard patterns (skip research-phase):
- **Phase 1:** Foundation patterns (Go project layout, Cobra scaffolding, atomic symlink) are mature with high-confidence official sources
- **Phase 2:** Profile CRUD and manifest format are fully specified by the Python source; JSON serialization is stdlib
- **Phase 3:** All features follow from the Python source; no new dependencies
- **Phase 4:** goreleaser and Cobra completions are well-documented with official sources

No phase requires additional research — the Python source provides a complete behavioral specification and all chosen libraries have official documentation. The primary research need during implementation is careful reading of the Python source for exact field names, schema structures, and edge-case behavior.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All library versions confirmed from official sources; goreleaser, Cobra, renameio all have stable APIs |
| Features | HIGH | Python source read directly (~1480 lines); feature scope is authoritative |
| Architecture | HIGH | Cobra and Go project layout are mature; symlink atomicity from official Go/Linux sources |
| Pitfalls | HIGH (fs/symlink), MEDIUM (Cobra/goreleaser edge cases) | Filesystem behavior from kernel docs and official Go source; Cobra gotchas from community sources |

**Overall confidence:** HIGH

### Gaps to Address

- **Dual binary `go install` UX:** `go install` works by import path, and two binary names from one `main` package requires either two separate `cmd/hop` and `cmd/claudehopper` entry points (each calling the same root) or a build-tag approach. PITFALLS.md flags this as needing verification. Resolve during Phase 4 goreleaser config.
- **Windows support for atomic symlinks:** `google/renameio/v2` explicitly does not export `Symlink` on Windows. The tool may have best-effort Windows support for switch operations. Document the limitation during Phase 1 rather than discovering it in production.
- **testify vs stdlib testing:** The stack research defers this decision to when tests are written. Either is acceptable; decide in Phase 1 when the first tests are authored and apply consistently.

## Sources

### Primary (HIGH confidence)
- Python claudehopper source `/home/matthew/Programming/claudehopper/src/claudehopper/cli.py` — complete behavioral specification
- https://pkg.go.dev/github.com/spf13/cobra — Cobra v1.10.2 API and completions
- https://github.com/goreleaser/goreleaser/releases/latest — goreleaser v2.14.3
- https://pkg.go.dev/github.com/creativeprojects/go-selfupdate — go-selfupdate v1.5.2
- https://github.com/google/renameio — renameio v2 atomic symlink behavior
- https://go.dev/blog/go1.25 — Go 1.25 current stable
- https://goreleaser.com/customization/builds/go/ — dual binary build configuration
- https://go.dev/doc/modules/layout — official Go module layout

### Secondary (MEDIUM confidence)
- https://www.bytesizego.com/blog/structure-go-cli-app — Go CLI structure patterns
- https://github.com/golang-standards/project-layout — community project layout standard
- https://clig.dev/ — CLI UX best practices (dry-run, --help conventions)
- https://cobra.dev/docs/how-to-guides/shell-completion/ — completions portability
- https://goreleaser.com/errors/multiple-binaries-archive/ — dual binary goreleaser pitfall

### Tertiary (supporting)
- https://blog.moertel.com/posts/2005-08-22-how-to-change-symlinks-atomically.html — atomic symlink theory
- https://lwn.net/Articles/900334/ — symbolic link security considerations
- https://benhoyt.com/writings/learning-go/ — Python-to-Go porting patterns

---
*Research completed: 2026-03-14*
*Ready for roadmap: yes*
