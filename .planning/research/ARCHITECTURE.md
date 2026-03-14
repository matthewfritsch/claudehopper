# Architecture Research

**Domain:** Go CLI tool — config profile manager with symlink-based switching
**Researched:** 2026-03-14
**Confidence:** HIGH (Cobra and Go project layout are mature, well-documented; symlink atomicity from official Go/Linux sources)

## Standard Architecture

### System Overview

```
┌──────────────────────────────────────────────────────────────┐
│                    Entry Points (CLI Layer)                   │
│                                                              │
│  main.go         cmd/root.go                                 │
│  (calls          (PersistentPreRun: load config,             │
│  cmd.Execute())   version flag, completions setup)           │
├──────────────────────────────────────────────────────────────┤
│                   Command Layer  (cmd/)                       │
│                                                              │
│  create.go  switch.go  list.go  status.go  share.go          │
│  pick.go    unshare.go diff.go  delete.go  tree.go           │
│  stats.go   path.go    update.go                             │
│                                                              │
│  Each command: parse flags → validate → call internal pkg    │
├──────────────────────────────────────────────────────────────┤
│                  Internal Packages (internal/)                │
│                                                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────────┐  │
│  │ profile/ │ │  fs/     │ │ config/  │ │   updater/     │  │
│  │          │ │          │ │          │ │                │  │
│  │ CRUD ops │ │ symlinks │ │ load/    │ │ GitHub release │  │
│  │ manifest │ │ atomic   │ │ save     │ │ check + notify │  │
│  │ lineage  │ │ backup   │ │ JSON     │ │                │  │
│  └──────────┘ └──────────┘ └──────────┘ └────────────────┘  │
├──────────────────────────────────────────────────────────────┤
│                     Storage Layer                            │
│                                                              │
│  ~/.config/claudehopper/config.json   (active profile etc)  │
│  ~/.config/claudehopper/profiles/<name>/  (profile dirs)    │
│  ~/.config/claudehopper/profiles/<name>/.hop-manifest.json  │
│  ~/.config/claudehopper/shared/       (cross-profile files) │
│  ~/.config/claudehopper/usage.jsonl   (usage tracking)      │
│  ~/.claude/                           (live Claude dir)      │
└──────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Communicates With |
|-----------|----------------|-------------------|
| `main.go` | Binary entry point; calls `cmd.Execute()` | `cmd/` only |
| `cmd/root.go` | Root Cobra command; persistent flags (--dry-run, --force, --verbose); global config load in `PersistentPreRunE` | `internal/config/` |
| `cmd/<verb>.go` | One file per subcommand; parse args/flags, validate, delegate to `internal/` | `internal/profile/`, `internal/fs/`, `internal/config/` |
| `internal/profile/` | Profile CRUD, manifest load/save, lineage tracking, adoption logic, path validation | `internal/fs/`, `internal/config/` |
| `internal/fs/` | Atomic symlink creation/replacement, backup-on-conflict, protected path enforcement | OS only (stdlib `os`, `path/filepath`) |
| `internal/config/` | Load/save `config.json` and `.hop-manifest.json`; hold `DEFAULT_LINKED` and `SHARED_PATHS` constants | OS only |
| `internal/updater/` | Check GitHub releases API, cache last-check timestamp, format update notice | `internal/config/` (for cache path), `net/http` |

## Recommended Project Structure

```
claudehopper-go/
├── main.go                    # Entry: calls cmd.Execute(), sets version via ldflags
├── go.mod
├── go.sum
├── .goreleaser.yaml           # Builds both "hop" and "claudehopper" binaries
│
├── cmd/
│   ├── root.go                # rootCmd, PersistentPreRunE loads config
│   ├── create.go              # hop create <name> [--from-current | --from <profile>]
│   ├── switch.go              # hop switch <name> [--dry-run] [--force]
│   ├── list.go                # hop list
│   ├── status.go              # hop status
│   ├── share.go               # hop share <file>
│   ├── pick.go                # hop pick <file> [--from <profile>]
│   ├── unshare.go             # hop unshare <file>
│   ├── diff.go                # hop diff [<profile>]
│   ├── delete.go              # hop delete <name>
│   ├── tree.go                # hop tree
│   ├── stats.go               # hop stats [<profile>]
│   ├── path.go                # hop path [<profile>]
│   └── update.go              # hop update
│
├── internal/
│   ├── config/
│   │   ├── config.go          # AppConfig struct, Load(), Save(), dir resolution
│   │   ├── constants.go       # DEFAULT_LINKED, SHARED_PATHS, path constants
│   │   └── config_test.go
│   │
│   ├── profile/
│   │   ├── profile.go         # Profile struct, list, require, validate name
│   │   ├── manifest.go        # Manifest struct, load/save .hop-manifest.json
│   │   ├── create.go          # Create (blank / from-current / from-profile)
│   │   ├── switch.go          # _doSwitch: preflight, adopt, relink, record
│   │   ├── share.go           # Share/pick/unshare logic
│   │   ├── detect.go          # detect_profile_paths, detect_unmanaged
│   │   └── profile_test.go
│   │
│   ├── fs/
│   │   ├── symlink.go         # AtomicSymlink() using tmp+rename pattern
│   │   ├── backup.go          # BackupPath() → <file>.hop-backup
│   │   ├── protected.go       # IsProtected() checks path against SHARED_PATHS
│   │   └── fs_test.go
│   │
│   └── updater/
│       ├── updater.go         # CheckUpdate(), PrintUpdateNotice()
│       ├── version.go         # ParseVersion(), CompareVersions()
│       └── updater_test.go
│
└── testdata/                  # Fixture profiles/manifests for integration tests
```

### Structure Rationale

- **`cmd/`:** One file per subcommand. Each file contains a `NewXxxCmd()` constructor that returns a `*cobra.Command`. This is the NewCommand() pattern — preferred over global vars + `init()` because it makes dependency injection explicit and avoids init-order surprises.
- **`internal/profile/`:** Groups all domain logic about profiles together. Mirrors how the Python version's `cmd_*` functions cluster around profile state.
- **`internal/fs/`:** Isolates filesystem operations that need care (atomic symlinks, backups, protected-path enforcement). This boundary means the rest of the codebase never calls `os.Symlink` directly.
- **`internal/config/`:** Separates config/constants from profile logic so both `cmd/` and `internal/profile/` can import it without circular imports.
- **`internal/updater/`:** Self-contained; only needs config for the cache file path.
- **`main.go` is tiny:** Contains only `func main()` calling `cmd.Execute()`. Version string injected at build time via `-ldflags "-X main.version=..."`.

## Architectural Patterns

### Pattern 1: NewCommand() Constructor

**What:** Each `cmd/*.go` file exports a `NewXxxCmd() *cobra.Command` function instead of registering a global `var xxxCmd` in `init()`.

**When to use:** Any multi-file Cobra project. Eliminates implicit init-order dependencies and makes the wiring explicit in `root.go`.

**Trade-offs:** Slightly more boilerplate per file vs. global vars, but the explicit dependency graph is worth it.

**Example:**
```go
// cmd/switch.go
func NewSwitchCmd(cfg *config.AppConfig) *cobra.Command {
    var dryRun, force bool
    cmd := &cobra.Command{
        Use:   "switch <name>",
        Short: "Switch to a named profile",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return profile.DoSwitch(cfg, args[0], profile.SwitchOpts{
                DryRun: dryRun,
                Force:  force,
            })
        },
    }
    cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would change without doing it")
    cmd.Flags().BoolVar(&force, "force", false, "Overwrite non-managed files")
    return cmd
}

// cmd/root.go
func NewRootCmd(version string) *cobra.Command {
    var cfg config.AppConfig
    root := &cobra.Command{...}
    root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
        return config.Load(&cfg)
    }
    root.AddCommand(
        NewSwitchCmd(&cfg),
        NewCreateCmd(&cfg),
        // ...
    )
    return root
}
```

### Pattern 2: Atomic Symlink via tmp+rename

**What:** Never call `os.Symlink(target, linkName)` directly when `linkName` may already exist. Create in a temp path in the same directory, then `os.Rename(tmp, linkName)`.

**When to use:** Every symlink creation in the switch path. The Python version uses this — Go must too.

**Trade-offs:** Requires temp file cleanup on error, but guarantees no half-written state visible to other processes or a concurrent Claude Code session.

**Example:**
```go
// internal/fs/symlink.go
func AtomicSymlink(target, linkPath string) error {
    dir := filepath.Dir(linkPath)
    tmp, err := os.CreateTemp(dir, ".hop-tmp-*")
    if err != nil {
        return err
    }
    tmpName := tmp.Name()
    tmp.Close()
    os.Remove(tmpName) // remove the regular file; we want a symlink at this path

    if err := os.Symlink(target, tmpName); err != nil {
        return err
    }
    return os.Rename(tmpName, linkPath) // atomic on POSIX
}
```

### Pattern 3: Options Struct for Multi-Flag Commands

**What:** Group flags for complex operations into a struct (`SwitchOpts`, `CreateOpts`) rather than passing many positional booleans to internal functions.

**When to use:** Any internal function accepting more than 2 boolean flags.

**Trade-offs:** Minimal overhead; makes call sites readable and avoids parameter-order bugs.

## Data Flow

### Profile Switch Flow

```
User: hop switch work
        |
cmd/switch.go: NewSwitchCmd.RunE()
        | validate args/flags
        v
internal/profile/switch.go: DoSwitch(cfg, "work", opts)
        |
        +-- config.Load()        reads ~/.config/claudehopper/config.json
        +-- profile.Require()    verifies "work" profile directory exists
        +-- manifest.Load()      reads .hop-manifest.json from profile dir
        +-- ValidatePreflight()  checks for protected path conflicts
        |
        +-- [if unmanaged files in ~/.claude/]
        |       detect.DetectUnmanaged() → adopt into profile
        |
        +-- [for each managed path]
        |       fs.AtomicSymlink(profileDir/file, claudeDir/file)
        |
        +-- config.Save()        writes new active profile to config.json
        +-- record_usage()       appends to usage.jsonl
        |
        v
stdout: "Switched to profile 'work'"
```

### Config/Manifest Load Flow

```
On any command (PersistentPreRunE):
        AppConfig.Load()
            reads ~/.config/claudehopper/config.json
            sets: ActiveProfile, ProfilesDir, SharedDir
            creates dirs if absent (first-run bootstrap)

Per-command:
        manifest.Load(profileDir)
            reads profileDir/.hop-manifest.json
            returns: ManagedPaths, SharedPaths, Lineage, CreatedAt
```

### Key Data Flows

1. **Switch:** cmd layer reads flags → profile package validates + relinks → fs package does atomic symlinks → config package writes new active profile
2. **Create:** cmd layer reads name/source flags → profile package copies/links files, writes initial manifest → config is NOT updated (switch is required to activate)
3. **Share/Pick:** cmd layer reads filename → profile package resolves path, moves to shared dir, updates manifests in all affected profiles
4. **Update check:** Runs as a side effect in root PersistentPreRunE, non-blocking; checks cached timestamp before hitting GitHub API

## Scaling Considerations

This is a local CLI tool. "Scale" means binary size, startup latency, and maintenance over time — not distributed throughput.

| Concern | Reality | Approach |
|---------|---------|----------|
| Binary size | Cobra adds ~3 MB | Acceptable for a CLI. Use `go build -ldflags="-s -w"` + UPX in goreleaser if size matters |
| Startup time | Sub-50ms is trivial for a Go CLI | Avoid heavy init; defer network calls (update check) |
| Package sprawl | internal/ can grow | Keep package count low; merge small packages before splitting |
| Test coverage | Filesystem tests are slow | Use `t.TempDir()` for isolated test roots; mock atomic rename is unnecessary — use real fs |

## Anti-Patterns

### Anti-Pattern 1: Global Command Variables + init()

**What people do:** Declare `var switchCmd = &cobra.Command{...}` at package level and register it in `func init() { rootCmd.AddCommand(switchCmd) }`.

**Why it's wrong:** init() ordering is implicit, making it hard to inject dependencies (like a loaded config) into commands. Testing individual commands requires running the full init chain.

**Do this instead:** Use `NewXxxCmd(cfg *config.AppConfig) *cobra.Command` constructors, wired explicitly in `NewRootCmd`.

### Anti-Pattern 2: Calling os.Symlink Directly for Profile Switches

**What people do:** `os.Remove(dest); os.Symlink(src, dest)` — two non-atomic operations.

**Why it's wrong:** If the process dies between Remove and Symlink, the file is gone. Claude Code may run concurrently and see a missing file.

**Do this instead:** `fs.AtomicSymlink()` — create symlink at temp path in same directory, rename over destination (POSIX rename is atomic).

### Anti-Pattern 3: Putting Business Logic in cmd/

**What people do:** Write the full profile switch logic inside `RunE` in `cmd/switch.go`.

**Why it's wrong:** cmd/ files can't be tested without invoking Cobra's parsing machinery. Logic becomes coupled to CLI concerns.

**Do this instead:** `RunE` does only: parse flags → validate arguments → call `internal/profile` → format output. All state mutation lives in `internal/`.

### Anti-Pattern 4: Hard-Coding Path Separators or Home Directory

**What people do:** `path := "/home/" + user + "/.claude/" + file`.

**Why it's wrong:** Breaks on Windows (if ever needed) and in test environments.

**Do this instead:** `os.UserHomeDir()` or `os.UserConfigDir()` for base paths; `filepath.Join()` everywhere. Paths passed into `internal/` functions as `string` so tests can inject temp dirs.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| GitHub Releases API | HTTP GET to `api.github.com/repos/{owner}/{repo}/releases/latest` | Cache last-check time in `~/.config/claudehopper/last-update-check.json`. 24h TTL. Non-fatal on failure. |
| Shell completion systems (bash/zsh/fish/PowerShell) | Built into Cobra via `rootCmd.GenBashCompletion` etc. | Free from Cobra; exposed as `hop completion <shell>`. No extra code needed. |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| `cmd/` → `internal/profile/` | Direct function calls, options structs | cmd/ passes loaded `*config.AppConfig` pointer; profile package owns all mutation |
| `cmd/` → `internal/config/` | Load/Save only | Loaded once in PersistentPreRunE; pointer passed down to subcommands |
| `internal/profile/` → `internal/fs/` | Direct function calls | profile/ never calls `os.Symlink` directly; always through `fs.AtomicSymlink` |
| `internal/profile/` → `internal/config/` | Read constants, load/save manifest | profile/ imports config/ for `DEFAULT_LINKED`, `SHARED_PATHS`, and dir paths |
| `internal/updater/` → network | `net/http` GET, `encoding/json` decode | Isolated; no other internal package touches the network |

## Build Order for Implementation

The dependency graph drives a natural build order:

1. **`internal/config/`** — Zero dependencies; establishes path constants and config structs. Everything else imports this.
2. **`internal/fs/`** — Only depends on stdlib. AtomicSymlink, BackupPath, IsProtected. Testable in isolation with `t.TempDir()`.
3. **`internal/profile/`** (data structures) — Manifest + Profile structs, Load/Save. No switch logic yet. Testable immediately.
4. **`cmd/` skeleton** — root + list + status commands wired with `NewRootCmd` pattern. Validates that the Cobra wiring compiles and completions work.
5. **`internal/profile/`** (mutations) — Create, Switch, Share/Pick/Unshare. These are the core domain logic and should each have tests before moving on.
6. **Remaining `cmd/`** — Wire remaining commands to the completed internal packages.
7. **`internal/updater/`** — Self-contained; implement last.
8. **Goreleaser config** — Dual binary names (`hop`, `claudehopper`) + ldflags version injection.

## Sources

- [Official Go module layout — go.dev](https://go.dev/doc/modules/layout) (HIGH confidence — official)
- [Cobra repository and documentation — github.com/spf13/cobra](https://github.com/spf13/cobra) (HIGH confidence — official)
- [Structuring Go Code for CLI Applications — bytesizego.com](https://www.bytesizego.com/blog/structure-go-cli-app) (MEDIUM confidence — verified against official patterns)
- [Building CLI Apps in Go with Cobra & Viper — glukhov.org, 2025-11](https://www.glukhov.org/post/2025/11/go-cli-applications-with-cobra-and-viper) (MEDIUM confidence)
- [Go Project Structure: Practices & Patterns — glukhov.org, 2025-12](https://www.glukhov.org/post/2025/12/go-project-structure/) (MEDIUM confidence)
- [golang-standards/project-layout — github.com](https://github.com/golang-standards/project-layout) (MEDIUM confidence — community standard, not official)
- [google/renameio — atomic symlink package](https://github.com/google/renameio) (HIGH confidence — from Google, addresses exact POSIX atomicity requirement)
- [Atomically writing files in Go — Michael Stapelberg](https://michael.stapelberg.ch/posts/2017-01-28-golang_atomically_writing/) (MEDIUM confidence)
- [GoReleaser build customization](https://goreleaser.com/customization/builds/go/) (HIGH confidence — official)
- Python claudehopper source — `/home/matthew/Programming/claudehopper/src/claudehopper/cli.py` (PRIMARY reference for feature/domain logic)

---
*Architecture research for: claudehopper-go — Go CLI config profile manager*
*Researched: 2026-03-14*
