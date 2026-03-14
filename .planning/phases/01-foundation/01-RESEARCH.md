# Phase 1: Foundation - Research

**Researched:** 2026-03-14
**Domain:** Go module scaffold, Cobra CLI, atomic symlinks, config/path resolution, protected-path enforcement
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Primary binary name: `claudehopper` (this is what `go install` produces)
- Alias binary: `hop` — created via install mechanism (not a separate main package)
- Module path: `github.com/matthewfritsch/claudehopper`
- Single `main.go` entry point, not two `cmd/` packages
- goreleaser produces both names in release binaries
- For source installs, Claude decides alias mechanism (Makefile or similar)
- stdlib `testing` package only — no testify or other test dependencies
- Test fixtures in `testdata/` directories alongside test files (Go convention)
- Real JSON files from Python version stored as fixtures for format compatibility checks
- No automated cross-version fixture extraction — visual verification of Python constants is sufficient
- Claude decides test thoroughness per component based on risk level
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

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| SAFE-01 | Protected paths (credentials, history, projects, cache) are never touched during any operation | Python SHARED_PATHS constants verified; `internal/fs.IsProtected()` function design documented |
| SAFE-03 | Manifest and config.json formats are compatible with the Python claudehopper version | Live manifest and config.json files inspected; exact JSON schemas documented |
| DIST-02 | Tool installs as both `hop` and `claudehopper` binary names | goreleaser dual-build pattern documented; Makefile `ln -sf` pattern for source installs |
| DIST-04 | Every subcommand has `--help` and root has `--version` | Cobra version/help patterns fully documented; ldflags injection approach confirmed |
</phase_requirements>

## Summary

This phase scaffolds a greenfield Go module for `claudehopper`, a CLI tool that switches Claude Code configuration profiles. The Go rewrite must produce byte-for-byte compatible JSON manifests and config files with the existing Python version, enforce identical protected-path sets, and ship as two differently named binaries (`hop` and `claudehopper`).

The core technical challenges are: (1) atomic symlink replacement without ever leaving a broken state mid-operation, (2) identical protected-path constants with the Python source of truth, (3) `--version` that prints a real string (not `(devel)`) for both `go install` and release builds, and (4) dual binary distribution from a single `main.go`.

The stack is well-understood: Cobra v1.10.x for CLI (automatic `--help` on all commands, `--version` on root), `google/renameio/v2` for atomic symlinks (confirmed Windows limitation — Linux/macOS only), `fatih/color` v1.18 for TTY-aware output, and `os.UserConfigDir()` for XDG-compliant config path. All libraries are stable, widely used, and have HIGH confidence research coverage.

**Primary recommendation:** Structure the project as `main.go` + `cmd/root.go` + `internal/{fs,config}/`. Keep `internal/fs` focused on atomic symlinks and protected-path checks. Keep `internal/config` focused on path resolution and JSON serialization that matches the Python format exactly.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/spf13/cobra | v1.10.x | CLI command tree, --help, --version, shell completions | Used by kubectl, gh, Hugo; automatic help on all subcommands; Cobra's `rootCmd.Version` + ldflags is the canonical pattern |
| github.com/google/renameio/v2 | v2.x | Atomic symlink creation/replacement | Only correct solution on Linux/macOS; wraps `tmp symlink + rename(2)` atomically; what the STATE.md already decided |
| github.com/fatih/color | v1.18.0 | TTY-aware colored output | Auto-detects non-TTY (pipes) and disables color; respects NO_COLOR env var; widely used in Go CLIs |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| os.UserConfigDir() | stdlib | XDG_CONFIG_HOME / ~/.config resolution | Resolves `XDG_CONFIG_HOME` on Linux when set, falls back to `$HOME/.config`; avoids storing tilde strings |
| runtime/debug.ReadBuildInfo() | stdlib | Fallback version for `go install` | Returns `(devel)` when no ldflags set; use as fallback only |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| fatih/color | github.com/jwalton/gchalk | gchalk has better Windows support but adds complexity; fatih/color is sufficient for Linux/macOS target |
| os.UserConfigDir() | github.com/adrg/xdg | adrg/xdg is more spec-compliant but a dependency; stdlib is sufficient for Linux (where XDG_CONFIG_HOME works correctly) |
| renameio/v2 | hand-rolled tmp+rename | Never hand-roll — race conditions and cleanup edge cases are subtle; renameio handles all of them |

**Installation:**
```bash
go get github.com/spf13/cobra@latest
go get github.com/google/renameio/v2@latest
go get github.com/fatih/color@latest
```

## Architecture Patterns

### Recommended Project Structure
```
github.com/matthewfritsch/claudehopper/
├── main.go                    # Entry point; ldflags version vars; cobra Execute()
├── cmd/
│   └── root.go                # rootCmd definition, --version, PersistentPreRun
├── internal/
│   ├── fs/
│   │   ├── atomic.go          # AtomicSymlink() wrapping renameio
│   │   ├── atomic_test.go     # t.TempDir()-based tests
│   │   ├── protected.go       # IsProtected(), SHARED_PATHS constant set
│   │   └── protected_test.go  # fixture test against Python constants
│   └── config/
│       ├── paths.go           # ConfigDir(), ProfilesDir(), resolveConfigDir()
│       ├── paths_test.go      # XDG_CONFIG_HOME and default path tests
│       ├── config.go          # Load/Save config.json (active profile)
│       └── manifest.go        # Load/Save .hop-manifest.json (Phase 2 uses this)
├── Makefile                   # build, install, install-hop-alias targets
└── .goreleaser.yaml           # dual binary build
```

### Pattern 1: Cobra Root Command with ldflags Version
**What:** Single `main.go` declares `var version = "dev"` (overridden at build time); passes to Cobra's `rootCmd.Version`.
**When to use:** Always — this is the canonical Go CLI version pattern.
**Example:**
```go
// main.go — Source: https://goreleaser.com/cookbooks/using-main.version/
var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)

func main() {
    cmd.SetVersionInfo(version, commit, date)
    cmd.Execute()
}

// cmd/root.go
var rootCmd = &cobra.Command{
    Use:     "claudehopper",
    Short:   "Switch Claude Code configuration profiles",
    Version: "", // set via SetVersionInfo
}

func SetVersionInfo(version, commit, date string) {
    rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
}
```

Goreleaser ldflags:
```yaml
# .goreleaser.yaml
builds:
  - id: claudehopper
    main: .
    binary: claudehopper
    ldflags:
      - -s -w -X main.version={{.Version}} -X main.commit={{.ShortCommit}} -X main.date={{.Date}}
  - id: hop
    main: .
    binary: hop
    ldflags:
      - -s -w -X main.version={{.Version}} -X main.commit={{.ShortCommit}} -X main.date={{.Date}}
```

### Pattern 2: Atomic Symlink via renameio/v2
**What:** Wrap `renameio.Symlink` to create or replace a symlink atomically. Never use `os.Remove` + `os.Symlink`.
**When to use:** Every single symlink operation. No exceptions.
**Example:**
```go
// internal/fs/atomic.go — Source: https://pkg.go.dev/github.com/google/renameio/v2
import "github.com/google/renameio/v2"

// AtomicSymlink atomically creates or replaces a symlink at linkPath
// pointing to targetPath. Safe for concurrent use; never leaves a
// broken or missing symlink mid-operation.
// NOTE: Not supported on Windows (renameio/v2 does not export on Windows).
func AtomicSymlink(targetPath, linkPath string) error {
    return renameio.Symlink(targetPath, linkPath)
}
```

Test pattern using t.TempDir():
```go
// internal/fs/atomic_test.go
func TestAtomicSymlink_CreateNew(t *testing.T) {
    dir := t.TempDir()
    target := filepath.Join(dir, "target")
    link := filepath.Join(dir, "link")

    if err := os.WriteFile(target, []byte("data"), 0644); err != nil {
        t.Fatal(err)
    }
    if err := AtomicSymlink(target, link); err != nil {
        t.Fatalf("AtomicSymlink: %v", err)
    }
    dest, err := os.Readlink(link)
    if err != nil {
        t.Fatalf("Readlink: %v", err)
    }
    if dest != target {
        t.Errorf("got %q, want %q", dest, target)
    }
}

func TestAtomicSymlink_ReplaceExisting(t *testing.T) {
    dir := t.TempDir()
    target1 := filepath.Join(dir, "target1")
    target2 := filepath.Join(dir, "target2")
    link := filepath.Join(dir, "link")
    // ... create link pointing to target1, then replace with target2
    // verify no window of broken state (check link resolves throughout)
}
```

### Pattern 3: IsProtected and SHARED_PATHS
**What:** A string set of paths in `~/.claude/` that must never be touched. Checked before every profile operation.
**When to use:** Any code that would create, delete, or move a file under `~/.claude/`.
**Example:**
```go
// internal/fs/protected.go
// Copied verbatim from Python source SHARED_PATHS constant.
// See ~/Programming/claudehopper/src/claudehopper/cli.py:SHARED_PATHS
var sharedPaths = map[string]struct{}{
    ".credentials.json": {},
    "history.jsonl":     {},
    "projects":          {},
    "cache":             {},
    "downloads":         {},
    "transcripts":       {},
    "shell-snapshots":   {},
    "file-history":      {},
    "backups":           {},
    "session-env":       {},
    ".session-stats.json": {},
}

// IsProtected reports whether name (a bare filename, not a full path)
// is a path that claudehopper must never touch.
func IsProtected(name string) bool {
    _, ok := sharedPaths[name]
    return ok
}
```

Fixture test — compares Go constant against canonical Python source list:
```go
// internal/fs/protected_test.go
// testdata/python_shared_paths.txt — one path per line, copied from Python source
func TestIsProtected_MatchesPythonConstants(t *testing.T) {
    data, err := os.ReadFile("testdata/python_shared_paths.txt")
    // ... parse and verify every line in fixture is protected, and
    // set size matches (no extra Go entries, no missing entries)
}
```

### Pattern 4: Config Path Resolution (no tilde strings)
**What:** Use `os.UserConfigDir()` which respects `XDG_CONFIG_HOME` on Linux; never store `~` in any variable.
**When to use:** Every reference to the claudehopper config directory.
**Example:**
```go
// internal/config/paths.go
// Source: https://pkg.go.dev/os#UserConfigDir
func ConfigDir() (string, error) {
    base, err := os.UserConfigDir() // returns $XDG_CONFIG_HOME or $HOME/.config on Linux
    if err != nil {
        return "", fmt.Errorf("cannot determine config dir: %w", err)
    }
    return filepath.Join(base, "claudehopper"), nil
}

func ProfilesDir() (string, error) {
    cfg, err := ConfigDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(cfg, "profiles"), nil
}
```

Test pattern overriding XDG_CONFIG_HOME:
```go
func TestConfigDir_XDGOverride(t *testing.T) {
    tmp := t.TempDir()
    t.Setenv("XDG_CONFIG_HOME", tmp) // t.Setenv restores on cleanup
    got, err := ConfigDir()
    if err != nil {
        t.Fatal(err)
    }
    want := filepath.Join(tmp, "claudehopper")
    if got != want {
        t.Errorf("got %q, want %q", got, want)
    }
}
```

### Pattern 5: Dual Binary for Source Installs
**What:** `go install` only produces `claudehopper`. A Makefile `install` target also creates `hop` as a symlink in `$GOPATH/bin`.
**When to use:** For source-install users (not goreleaser release users).
**Example:**
```makefile
# Makefile
GOBIN ?= $(shell go env GOPATH)/bin

.PHONY: install
install:
	go install ./...
	ln -sf $(GOBIN)/claudehopper $(GOBIN)/hop

.PHONY: build
build:
	go build -ldflags="-X main.version=dev" -o bin/claudehopper .
	ln -sf claudehopper bin/hop
```

### Pattern 6: os.Lstat for Symlink Inspection
**What:** Use `os.Lstat` (not `os.Stat`) when inspecting managed symlinks. `os.Stat` follows symlinks to the target; `os.Lstat` returns info about the link itself.
**When to use:** Any operation checking if a managed path is a symlink, or getting info about a link.
**Example:**
```go
// Correct: checks the symlink itself
fi, err := os.Lstat(managedPath)
if err == nil && fi.Mode()&os.ModeSymlink != 0 {
    // path is a managed symlink
}

// Wrong: follows the link and would fail if target is missing
fi, err := os.Stat(managedPath) // DO NOT USE for managed symlinks
```

### Anti-Patterns to Avoid
- **os.Remove + os.Symlink:** Creates a window where the managed path does not exist. Use `renameio.Symlink` exclusively.
- **Storing tilde strings:** Never `~/.config/claudehopper` as a string. Always resolve via `os.UserConfigDir()`.
- **os.Stat on managed symlinks:** Will follow the link and return the target's info (or error if target missing). Use `os.Lstat`.
- **Hardcoding $HOME:** Use `os.UserHomeDir()` instead; avoids issues with sudo or non-standard environments.
- **Two separate main packages for two binary names:** Adds complexity with no benefit. Goreleaser handles dual names from one `main.go` via two build stanzas.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Atomic symlink replace | tmp + rename loop | renameio.Symlink | Edge cases: cleanup on failure, tmp name collision, EXDEV errors across filesystems |
| TTY detection + color | ANSI escape codes + isatty check | fatih/color | Handles NO_COLOR, piped output, CI, Cygwin; ~10 edge cases |
| --help on every command | Custom usage function | cobra.Command.Use + Short | Cobra auto-generates consistent help for every subcommand |
| --version printing | Custom flag | rootCmd.Version + ldflags | Cobra handles `-v`/`--version` automatically once Version is set |
| XDG config path | os.Getenv("XDG_CONFIG_HOME") + fallback | os.UserConfigDir() | Handles edge cases per OS; correct fallback chain already in stdlib |

**Key insight:** The three most dangerous hand-rolls in this phase are: (1) atomic symlinks — subtle race conditions, (2) TTY/color detection — many edge cases across platforms and CI, (3) config path — XDG spec compliance is non-trivial.

## Common Pitfalls

### Pitfall 1: version Prints "(devel)"
**What goes wrong:** `hop --version` or `claudehopper --version` prints `v0.0.0-devel` or `(devel)`.
**Why it happens:** Go embeds `(devel)` as the module version when built with `go build ./...` without ldflags override, or when using `runtime/debug.ReadBuildInfo()` on a local build.
**How to avoid:** Declare `var version = "dev"` in `main.go` and set `rootCmd.Version = version`. The goreleaser ldflags `-X main.version={{.Version}}` overwrites it for releases. Local `go build` shows "dev" which is acceptable; "dev" is not "(devel)".
**Warning signs:** Running `go build . && ./claudehopper --version` outputs something containing "(devel)".

### Pitfall 2: Broken Symlink Window During Switch
**What goes wrong:** A profile switch leaves `~/.claude/settings.json` missing for a brief moment if using `os.Remove` then `os.Symlink`.
**Why it happens:** Two separate syscalls with no atomicity guarantee; any signal or crash between them leaves broken state.
**How to avoid:** Use `renameio.Symlink` exclusively. It creates a tmp symlink and renames it atomically. Test with concurrent goroutines reading the path while the swap happens.
**Warning signs:** Test that removes and re-symlinks manually — any test not using `renameio` package.

### Pitfall 3: SHARED_PATHS Drift from Python
**What goes wrong:** Python version adds a new protected path (e.g., `session-env`) but Go version doesn't include it, so the Go version modifies a path the Python version considers protected.
**Why it happens:** Constants copied manually without a fixture test.
**How to avoid:** Write `testdata/python_shared_paths.txt` fixture from the Python source at implementation time. Write a test that compares Go's `sharedPaths` map keys against the fixture line by line. The test fails if either side drifts.
**Warning signs:** Any change to Python SHARED_PATHS not reflected in a test failure.

### Pitfall 4: Tilde String in Config Path
**What goes wrong:** Config path stored as `"~/.config/claudehopper"` string; tilde is not expanded by Go's file APIs.
**Why it happens:** Developers copy path from documentation or shell output without resolving it.
**How to avoid:** Always use `os.UserConfigDir()` which returns an absolute path. Test that `ConfigDir()` return value starts with `/` (not `~`).
**Warning signs:** Any string literal containing `~` in config path code.

### Pitfall 5: os.Stat Returns Error on Dangling Symlink
**What goes wrong:** Code uses `os.Stat(managedPath)` to check if a managed path exists; returns error for dangling symlinks (target deleted), causing false "path not found" logic.
**Why it happens:** `os.Stat` follows symlinks; if target is missing, it returns `os.ErrNotExist` even though the symlink file itself exists.
**How to avoid:** Use `os.Lstat` to interrogate managed symlinks. Only follow with `os.Stat` when you specifically need target info.
**Warning signs:** `if _, err := os.Stat(p); os.IsNotExist(err)` in any code that handles symlinked profile paths.

### Pitfall 6: renameio/v2 Not Available on Windows
**What goes wrong:** Cross-compilation for Windows fails, or Windows users get an empty binary.
**Why it happens:** `renameio/v2` does not export any functions on Windows — the package is a no-op there by design.
**How to avoid:** Document in Phase 1 that Windows is unsupported. Add a build comment or early runtime check. This is a known, accepted limitation.
**Warning signs:** Any CI/build target including `GOOS=windows`.

## Code Examples

Verified patterns from official sources:

### go.mod Initial Setup
```
// Source: go mod init documentation
module github.com/matthewfritsch/claudehopper

go 1.21
```

### Cobra Root Command with Version
```go
// Source: https://www.jvt.me/posts/2023/02/27/go-cobra-goreleaser-version/
// cmd/root.go
package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "claudehopper",
    Short: "Switch Claude Code configuration profiles",
    Long:  `claudehopper manages and switches between Claude Code configuration profiles.`,
}

func SetVersionInfo(version, commit, date string) {
    rootCmd.Version = fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)
}

func Execute() error {
    return rootCmd.Execute()
}
```

### config.json Format (Python-compatible)
```json
{
  "active": "gsd"
}
```

### .hop-manifest.json Format (Python-compatible)
```json
{
  "managed_paths": [
    "agents",
    "settings.json"
  ],
  "shared_paths": {},
  "description": "get-shit-done spec-driven development"
}
```

Note: `managed_paths` is a sorted JSON array of strings. `shared_paths` is an object mapping filename to source profile name. `description` is a plain string. No additional fields in Phase 1.

### Python SHARED_PATHS Constants (source of truth)
The following paths are verified from `/home/matthew/Programming/claudehopper/src/claudehopper/cli.py` line 43-55:
```
.credentials.json
history.jsonl
projects
cache
downloads
transcripts
shell-snapshots
file-history
backups
session-env
.session-stats.json
```

### Python DEFAULT_LINKED Constants (source of truth)
Verified from Python source lines 36-39:
```
settings.json
settings.local.json
.mcp.json
```

These are files symlinked across all profiles by default (Phase 2 concern, but the constants belong in Phase 1).

### fatih/color TTY-Aware Output
```go
// Source: https://pkg.go.dev/github.com/fatih/color
import "github.com/fatih/color"

// Automatically disabled when piped, respects NO_COLOR env var
var errStyle = color.New(color.FgRed, color.Bold)
var okStyle  = color.New(color.FgGreen)

func Die(msg string) {
    errStyle.Fprintf(os.Stderr, "error: %s\n", msg)
    os.Exit(1)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hand-rolled `os.Remove` + `os.Symlink` | `renameio.Symlink` (atomic via rename(2)) | Available since renameio v0.1 | Eliminates broken-state window |
| Hardcoding `$HOME/.config` | `os.UserConfigDir()` (stdlib) | Go 1.13+ | Correct XDG_CONFIG_HOME support on Linux |
| Manual `--version` flag | `rootCmd.Version` + ldflags `-X` | Cobra v1.x + goreleaser pattern | Consistent version output across all subcommands |
| Separate binaries from separate main packages | Same main, goreleaser two build stanzas | GoReleaser capability | One codebase, two binary names in release |

**Deprecated/outdated:**
- `go-isatty` direct usage: fatih/color wraps it; no need to import go-isatty directly
- `runtime/debug.ReadBuildInfo()` for user-visible version: returns `(devel)` for local builds; use ldflags var + "dev" default instead

## Open Questions

1. **Makefile vs `go install` UX for `hop` alias**
   - What we know: goreleaser handles both names in release builds via two build stanzas; `go install` only produces `claudehopper`
   - What's unclear: Whether to use `ln -sf` in Makefile or a separate `cmd/hop/main.go` that imports and calls `cmd.Execute()` (allowed since user decided "not two `cmd/` packages" but a thin wrapper would be different)
   - Recommendation: Use `Makefile install-hop-alias` target with `ln -sf $GOPATH/bin/claudehopper $GOPATH/bin/hop`. Simplest, no duplicate binary, no second compilation.

2. **Windows support documentation**
   - What we know: `renameio/v2` exports nothing on Windows; atomic symlinks impossible
   - What's unclear: Should the README note Windows is unsupported, or should there be a compile-time build tag?
   - Recommendation: Add a `// +build !windows` compile-tag guard on `internal/fs/atomic.go` that provides a clear error at compile time, plus a README note. This is Phase 1 scope per STATE.md blockers.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (go test) |
| Config file | none — `go test ./...` works out of the box |
| Quick run command | `go test ./internal/...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SAFE-01 | `IsProtected()` returns true for all 11 protected paths, false for non-protected | unit | `go test ./internal/fs/ -run TestIsProtected -v` | Wave 0 |
| SAFE-01 | `IsProtected()` constants match Python SHARED_PATHS fixture exactly | fixture/unit | `go test ./internal/fs/ -run TestIsProtected_MatchesPythonConstants -v` | Wave 0 |
| SAFE-03 | Config JSON round-trips with expected keys (`active`) | unit | `go test ./internal/config/ -run TestConfigJSON -v` | Wave 0 |
| SAFE-03 | Manifest JSON round-trips with expected keys (`managed_paths`, `shared_paths`, `description`) | unit | `go test ./internal/config/ -run TestManifestJSON -v` | Wave 0 |
| DIST-02 | `hop` and `claudehopper` both exist and point to same binary after `make install` | smoke (manual) | `make install && hop --help && claudehopper --help` | manual only — requires $GOPATH/bin |
| DIST-04 | `claudehopper --help` prints usage | smoke | `go run . --help` | Wave 0 (via cobra) |
| DIST-04 | `claudehopper --version` prints non-devel string | unit | `go test ./cmd/ -run TestVersion -v` | Wave 0 |
| SAFE-03 | `AtomicSymlink` creates symlink, verified with `os.Readlink` | unit | `go test ./internal/fs/ -run TestAtomicSymlink -v` | Wave 0 |
| SAFE-03 | `AtomicSymlink` replaces existing symlink atomically | unit | `go test ./internal/fs/ -run TestAtomicSymlink_Replace -v` | Wave 0 |
| SAFE-03 | `ConfigDir()` returns path under `XDG_CONFIG_HOME` when set | unit | `go test ./internal/config/ -run TestConfigDir_XDG -v` | Wave 0 |
| SAFE-03 | `ConfigDir()` returns path under `$HOME/.config` by default | unit | `go test ./internal/config/ -run TestConfigDir_Default -v` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/fs/atomic_test.go` — covers AtomicSymlink create and replace (SAFE-01, SAFE-03)
- [ ] `internal/fs/protected_test.go` — covers IsProtected and Python fixture match (SAFE-01)
- [ ] `internal/fs/testdata/python_shared_paths.txt` — fixture listing Python SHARED_PATHS constants
- [ ] `internal/config/paths_test.go` — covers ConfigDir XDG and default (SAFE-03)
- [ ] `internal/config/config_test.go` — covers config.json JSON round-trip (SAFE-03)
- [ ] `internal/config/manifest_test.go` — covers .hop-manifest.json round-trip (SAFE-03)
- [ ] `cmd/version_test.go` — covers rootCmd.Version non-devel (DIST-04)
- [ ] Go module: `go mod init github.com/matthewfritsch/claudehopper` (Wave 0 task)

*(Framework: stdlib `testing` — no install needed. Run `go mod tidy` after all imports are added.)*

## Sources

### Primary (HIGH confidence)
- https://pkg.go.dev/github.com/google/renameio/v2 — Symlink function signature, platform support, Windows limitation
- https://pkg.go.dev/github.com/spf13/cobra — Version v1.10.2, rootCmd.Version, shell completions API
- https://pkg.go.dev/os#UserConfigDir — XDG_CONFIG_HOME behavior on Linux, $HOME/.config fallback
- https://pkg.go.dev/github.com/fatih/color — v1.18.0, NoColor, TTY detection via go-isatty, NO_COLOR support
- /home/matthew/Programming/claudehopper/src/claudehopper/cli.py — Python source of truth: SHARED_PATHS (lines 43-55), DEFAULT_LINKED (lines 36-39), atomic_symlink pattern (lines 288-300), JSON manifest schema (lines 196-203)
- ~/.config/claudehopper/config.json — Live config.json format: `{"active": "gsd"}`
- ~/.config/claudehopper/profiles/gsd/.hop-manifest.json — Live manifest format with managed_paths, shared_paths, description

### Secondary (MEDIUM confidence)
- https://goreleaser.com/customization/builds/go/ — Dual build stanza pattern for two binary names from one main package
- https://www.jvt.me/posts/2023/02/27/go-cobra-goreleaser-version/ — Cobra + goreleaser version ldflags pattern
- https://goreleaser.com/cookbooks/using-main.version/ — ldflags -X main.version={{.Version}} template

### Tertiary (LOW confidence)
- WebSearch: Makefile `ln -sf` alias pattern for source installs — verified pattern is reasonable but no canonical Go ecosystem source

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries verified via pkg.go.dev; Python source read directly
- Architecture: HIGH — patterns derived from official docs and verified Python source constants
- Pitfalls: HIGH for symlink/version/stat pitfalls (verified against official docs); MEDIUM for drift detection (relies on maintained test fixture)
- JSON schema: HIGH — read from live files on this machine

**Research date:** 2026-03-14
**Valid until:** 2026-09-14 (stable libraries; cobra/renameio/fatih/color are all mature)
