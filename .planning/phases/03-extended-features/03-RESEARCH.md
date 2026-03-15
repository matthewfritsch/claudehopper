# Phase 3: Extended Features - Research

**Researched:** 2026-03-14
**Domain:** Go CLI — file sharing between profiles, terminal visualization, JSONL usage tracking, unmanage exit ramp
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Enhanced tree: show shared file indicators and profile sizes alongside lineage (parent-child from created_from)
- `--json` output: Claude decides schema (richer is better — include managed_paths counts, shared files, etc.)
- Charmbracelet libraries (lipgloss for styling) are acceptable if Claude determines the dependency is worth it for tree/diff/stats output
- If Charm is too heavy or not justified, plain ASCII with color (fatih/color or similar) is fine
- Claude decides comparison scope for diff (set operations on paths and/or byte-level content)
- Claude decides display style for diff
- Claude decides which actions to log to usage.jsonl (at minimum: switch, create, delete)
- Optional: fingerprint data from .claude/ could enrich stats — not required
- `hop stats` display: Claude decides useful analytics (switch counts, last-used, profile breakdown are baseline)

### Claude's Discretion
- Whether to add charmbracelet/lipgloss as a dependency (evaluate effort vs visual payoff)
- Diff comparison depth (paths only vs content diff)
- Usage tracking action scope
- Stats display format and filtering options
- Share/pick/unshare implementation details (match Python behavior)
- Unmanage implementation (materialize symlinks, clean config)
- `hop path` implementation (trivial — print profile dir)

### Deferred Ideas (OUT OF SCOPE)
- None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| SHAR-01 | User can symlink files between profiles (`hop share`) | Python cmd_share verified; uses atomic_symlink + manifest.shared_paths update; re-links active profile after |
| SHAR-02 | User can copy files between profiles independently (`hop pick`) | Python cmd_pick verified; shutil.copy2 equivalent; adds to managed_paths; re-links active profile after |
| SHAR-03 | User can materialize shared symlinks back to independent copies (`hop unshare`) | Python cmd_unshare verified; resolves symlink target, copies real file, removes from shared_paths |
| VIZ-01 | User can view profile lineage tree (`hop tree`) with optional `--json` output | Python cmd_tree verified; recursive ASCII render using created_from; JSON schema documented below |
| VIZ-02 | User can compare two profiles side-by-side (`hop diff`) | Python cmd_diff verified; set operations + byte-level file comparison for common paths |
| VIZ-03 | User can view usage statistics (`hop stats`) with optional `--json` output | Python cmd_stats verified; reads usage.jsonl; --since and --profile filters; JSON schema documented |
| VIZ-04 | User can print a profile's directory path for scripting (`hop path <name>`) | Python cmd_path verified; trivial — resolve profile dir and print |
| OPS-01 | User can stop using claudehopper by materializing all symlinks (`hop unmanage`) | Python cmd_unmanage verified; materialize all symlinks in ~/.claude/, set active=null |
| OPS-03 | All profile actions are logged to `usage.jsonl` for statistics | Python record_usage verified; append-only JSONL; format: {profile, timestamp, action}; never raises |
</phase_requirements>

## Summary

Phase 3 builds seven new commands on top of the complete CRUD and switch infrastructure from Phase 2. All commands have direct Python equivalents in `cli.py` that define the expected behavior precisely — this is a port, not a design exercise for most features.

The three file-sharing commands (share, pick, unshare) operate on profile directories using existing `config.LoadManifest`/`config.SaveManifest` and the already-implemented `renameio.Symlink` pattern. They follow the same manifest-mutation + optional re-link-active pattern used by existing commands. No new infrastructure is needed for these three.

The visualization commands (tree, diff, stats, path) are purely read operations — they read manifests, usage.jsonl, and the active config, then format output. The sole design decision is whether to add lipgloss for styling. After evaluating the dependency, the recommendation below is to use `fatih/color` for ANSI color (already a transitive community dependency, zero extra architecture), avoid lipgloss (justified only if building box-drawing layouts, which this project does not need). The stats and tree `--json` output schemas are defined explicitly below based on Python parity plus richer fields.

Usage tracking (OPS-03) is a cross-cutting concern: a single `RecordUsage(profile, action string)` function in `internal/usage/` appended to from switch, create, delete, and pick commands. It must never propagate errors to callers.

**Primary recommendation:** Implement in order: (1) internal/usage package + wire into existing commands, (2) share/pick/unshare in internal/profile/, (3) tree/diff/stats/path in cmd/, (4) unmanage. Each step is independently testable.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| encoding/json (stdlib) | Go stdlib | usage.jsonl append, --json output | Already used throughout; JSONL is one encode-per-line |
| os / path/filepath (stdlib) | Go stdlib | Symlink resolution, file copy, dir traversal | All needed primitives are available |
| io (stdlib) | Go stdlib | copyFile already in internal/profile/shared.go | Reuse existing copyFile helper |
| strings (stdlib) | Go stdlib | ASCII tree box-drawing connectors | No external dependency needed |
| google/renameio/v2 | v2.0.2 (already in go.mod) | Atomic symlink for hop share | Same renameio.Symlink already used by DoSwitch |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| fatih/color | v1.18.0 | ANSI color for tree active marker, diff indicators | If adding color to terminal output; ~300KB binary overhead; pure Go |
| charmbracelet/lipgloss | v1.x | Styled terminal layout with padding/borders | Only if building multi-column aligned output; adds ~2MB binary; NOT recommended for this phase |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| fatih/color | plain ANSI escape sequences | Raw escapes work but fatih/color handles NO_COLOR env var and Windows fallback correctly |
| fatih/color | charmbracelet/lipgloss | lipgloss is justified for full-panel TUI layouts, not for single-line color annotations; oversized dependency here |
| stdlib io.Copy | existing copyFile in shared.go | Reuse internal/profile/shared.go copyFile — already handles file permissions correctly |

**Recommendation on Charm:** Do NOT add lipgloss. The tree and stats outputs are line-oriented with simple indentation — ASCII box-drawing characters (├──, └──, │) handle the tree rendering, and `fatih/color` is sufficient for the active-profile marker and diff indicators. lipgloss costs 2MB+ binary size and adds a layout engine for a use case that does not need it.

**Installation (if color added):**
```bash
go get github.com/fatih/color@v1.18.0
```

## Architecture Patterns

### Recommended Project Structure Additions
```
internal/
├── usage/
│   ├── usage.go        # RecordUsage(), ReadUsage(), UsageEntry struct
│   └── usage_test.go
│
profile/
├── share.go            # ShareFiles(), PickFiles(), UnshareFiles()
├── share_test.go
├── tree.go             # BuildTree(), TreeNode struct
├── tree_test.go
├── diff.go             # DiffProfiles(), DiffResult struct
├── diff_test.go
├── stats.go            # AggregateStats(), ProfileStats struct
└── unmanage.go         # UnmanageActive()

cmd/
├── share.go            # hop share <file>... --from <source> [--to <target>] [--dry-run]
├── pick.go             # hop pick <file>... --from <source> [--to <target>] [--dry-run]
├── unshare.go          # hop unshare [<file>...] [--profile <name>] [--dry-run]
├── tree.go             # hop tree [--json]
├── diff.go             # hop diff <profile-a> <profile-b>
├── stats.go            # hop stats [--json] [--since YYYY-MM-DD] [--profile <name>]
├── path.go             # hop path <name>
└── unmanage.go         # hop unmanage [--dry-run]
```

### Pattern 1: Share via Atomic Symlink (SHAR-01)

**What:** Create a symlink in the target profile directory pointing to the source profile's file. Record the source profile name in target's manifest `shared_paths`. If the target is the active profile, re-run DoSwitch with Force=true to re-link into ~/.claude/.

**When to use:** `hop share <file>... --from <source> [--to <target>]`

**Key detail from Python source (line 854-858):**
```python
# For symlink sources: resolve the actual target, not a symlink chain
target = src.resolve() if not src.is_symlink() else src
atomic_symlink(target, dst)
shared_paths[path] = src_name
```

The Go equivalent resolves the file path before symlinking to avoid chained symlinks:
```go
// Source: internal/profile/share.go
func ShareFiles(profilesDir, srcName, tgtName string, paths []string, dryRun bool) ([]string, error) {
    srcDir := filepath.Join(profilesDir, srcName)
    tgtDir := filepath.Join(profilesDir, tgtName)
    tgtManifest, err := config.LoadManifest(filepath.Join(tgtDir, ".hop-manifest.json"))
    // ...
    for _, path := range paths {
        src := filepath.Join(srcDir, path)
        dst := filepath.Join(tgtDir, path)
        // Resolve real target if src is itself a symlink
        realTarget, _ := filepath.EvalSymlinks(src)
        if err := renameio.Symlink(realTarget, dst); err != nil {
            return shared, err
        }
        tgtManifest.SharedPaths[path] = srcName
        // Add to managed_paths if not already present
    }
    return shared, config.SaveManifest(...)
}
```

### Pattern 2: Pick via File Copy (SHAR-02)

**What:** Copy a file (or directory) from source profile to target profile. Add the path to target's `managed_paths`. Re-link active profile if target is active.

**Key detail from Python source (line 789-794):** When source is a symlink, preserve the symlink in the copy. When source is a directory, use recursive copy with symlinks=True. When source is a plain file, use a simple byte-for-byte copy.

```go
// Source: internal/profile/share.go
func PickFiles(profilesDir, srcName, tgtName string, paths []string, dryRun bool) ([]string, error) {
    // For each path: if symlink -> os.Symlink(os.Readlink(src), dst)
    //                if dir    -> copyDirRecursive(src, dst)
    //                if file   -> copyFile(src, dst)  [reuse existing helper]
    // Then add to tgtManifest.ManagedPaths, save manifest
}
```

### Pattern 3: Unshare via Materialize (SHAR-03)

**What:** For each path in shared_paths (or specified subset): resolve the symlink target, replace the symlink with a real file copy, remove the path from shared_paths. Re-link active profile if affected.

**Key detail:** Python handles the case where the shared target no longer exists (line 912) — just skip materializing but still remove from shared_paths. Go must match this.

### Pattern 4: ASCII Tree Rendering (VIZ-01)

**What:** Build a children_of map from created_from fields, then recursively render using box-drawing connectors. Visited set prevents cycles. Match Python output exactly.

**Tree connector logic (from Python, lines 1218-1233):**
- Root profiles: use `├── ` for non-last, `└── ` for last
- Child prefix: add `│   ` if parent used `├── `, else `    ` (four spaces)
- Files within a profile: same connector logic based on position
- Shared files get annotation: ` (shared from <source>)`
- Active profile gets annotation: ` (active)`

**JSON schema for `--json` output (richer than Python, as requested):**
```json
{
  "active": "work",
  "profiles": [
    {
      "name": "work",
      "active": true,
      "description": "Work context",
      "created_from": null,
      "managed_paths": ["CLAUDE.md", "commands/"],
      "managed_count": 2,
      "shared_paths": {"settings.json": "(shared)"},
      "shared_count": 1,
      "children": ["work-experiment"]
    }
  ]
}
```

### Pattern 5: Diff via Set Operations + Byte Comparison (VIZ-02)

**What:** Compare managed_paths of two profiles as sets: only_a, only_b, common. For common paths that are both plain files, compare bytes for identical/different. For common directories, count added/removed/shared files recursively. Match Python output format.

**Python diff format (lines 937-962):**
```
Only in 'work':
  CLAUDE.md
Only in 'personal':
  commands/
Shared:
  settings.json  [identical]
  .mcp.json      [different]
```

**Go recommendation:** Implement `DiffProfiles` as a pure function returning a `DiffResult` struct. The cmd layer formats it. This enables future `--json` output without restructuring.

### Pattern 6: Usage JSONL (OPS-03)

**What:** Append-only file at `~/.config/claudehopper/usage.jsonl`. One JSON object per line. Never propagates errors to callers.

**Usage entry schema (matches Python exactly):**
```json
{"profile": "work", "timestamp": "2026-03-14T10:23:45.123456", "action": "switch"}
```

**Actions to log:** switch, create, delete, pick, share (minimum: switch, create, delete per context)

**Implementation pattern:**
```go
// Source: internal/usage/usage.go
type UsageEntry struct {
    Profile   string `json:"profile"`
    Timestamp string `json:"timestamp"`
    Action    string `json:"action"`
}

// RecordUsage appends to usage.jsonl. Never returns an error to caller.
func RecordUsage(configDir, profile, action string) {
    entry := UsageEntry{
        Profile:   profile,
        Timestamp: time.Now().Format(time.RFC3339Nano),
        Action:    action,
    }
    data, _ := json.Marshal(entry)
    // Open for append, write data + "\n"
    // Swallow all errors — usage tracking must never fail a command
}
```

**Retroactive wiring:** After implementing `internal/usage`, add `usage.RecordUsage(...)` calls to:
- `cmd/switch.go` — after successful DoSwitch
- `cmd/create.go` — after successful create
- `cmd/delete.go` — after successful delete
- `cmd/share.go` (new) — after successful share operation (record "pick" for pick, "share" for share)

### Pattern 7: Stats Aggregation (VIZ-03)

**What:** Read all lines from usage.jsonl, parse each as UsageEntry, aggregate per profile. Support `--since` (ISO date filter) and `--profile` (name filter). Display sorted by switch count descending.

**JSON schema for `--json` output:**
```json
{
  "total_switches": 42,
  "profiles": [
    {
      "name": "work",
      "switches": 30,
      "last_used": "2026-03-14T10:23:45.123456",
      "actions": {"switch": 30, "create": 1}
    }
  ]
}
```

**Human-readable format (matching Python lines 1140-1146):**
```
Profile usage (all time):
  work        30 switches  (last: 2h ago)
  personal     8 switches  (last: 3d ago)
  experimental 4 switches  (last: 1w ago)

Total: 42 switches across 3 profiles
```

Relative time function: <60min → "Nm ago", <24h → "Nh ago", <7d → "Nd ago", else "Nw ago".

### Pattern 8: Unmanage (OPS-01)

**What:** For each path in the active profile's manifest, if the corresponding entry in `~/.claude/` is a symlink, resolve the target and replace the symlink with a real file/directory copy. Set `active: null` (empty string) in config.json.

**Edge cases from Python (lines 1023-1035):**
- Skip non-symlinks in `~/.claude/` (they are already real files)
- If symlink target is a directory: recursive copy
- If symlink target is a file: byte copy
- After materialization: save config with `active = ""`

**Dry-run mode:** List all symlinks that would be materialized without touching anything.

### Anti-Patterns to Avoid

- **Propagating RecordUsage errors:** usage tracking must swallow all errors — callers must never fail because of JSONL write issues
- **Calling re-link (DoSwitch) without Force=true:** share/pick/unshare call re-link to update ~/.claude/ symlinks; must use Force=true or the "already active" guard will no-op
- **String-concatenating paths for profile lookup:** always use `filepath.Join(profilesDir, name)` and validate with `os.Stat`
- **Mutating shared_paths map while ranging over it:** unshare iterates over a copy of keys, then removes from map — Go map mutation during range is safe but confusing; match Python's approach of building `paths_to_unshare` list first
- **Chained symlinks in share:** when the source file is itself a symlink (e.g., a default-linked file), resolve to the real target with `filepath.EvalSymlinks` before creating the new symlink — prevents chains of depth > 1

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Atomic symlink replacement | `os.Remove` then `os.Symlink` | `renameio.Symlink` (already in go.mod) | Two-step is not atomic; renameio is already present |
| File copy with permissions | Manual byte copy | `copyFile` in `internal/profile/shared.go` | Already implements correct permission preservation |
| Directory recursive copy | Manual `filepath.Walk` + copy | Extend existing `copyFile` pattern with `os.ReadDir` recursion | Simple enough to hand-write but must preserve symlinks |
| ANSI color | Raw `\033[32m` escape codes | `fatih/color` | Handles NO_COLOR env var, Windows terminal, isatty check |
| ISO timestamp parsing | Manual string parsing | `time.Time.Format(time.RFC3339Nano)` + `time.Parse` | stdlib handles this correctly |

**Key insight:** Every file operation in this phase is either symlink creation (renameio), file copy (extend existing copyFile), or manifest read/write (existing LoadManifest/SaveManifest). No new filesystem primitive infrastructure is needed.

## Common Pitfalls

### Pitfall 1: Re-linking Active Profile After Manifest Mutation
**What goes wrong:** share/pick/unshare mutate the profile manifest but if the target profile is currently active, the symlinks in `~/.claude/` are stale.
**Why it happens:** Manifest files and live symlinks are separate; mutating one does not update the other.
**How to avoid:** After any successful share/pick/unshare on the active profile, call `profile.DoSwitch(..., Force: true)` to re-link. Check `config.Active == targetName` before deciding to re-link.
**Warning signs:** Test that modifies a shared profile and checks `~/.claude/` symlink targets — they will be wrong if re-link is skipped.

### Pitfall 2: Double-Adding to managed_paths
**What goes wrong:** share/pick both add a path to `managed_paths` — but if the path is already present, it creates a duplicate in the sorted slice.
**Why it happens:** `append` does not deduplicate.
**How to avoid:** Check if path is already in ManagedPaths before appending. Python does this at lines 799-803 and 862-866 explicitly.

### Pitfall 3: usage.jsonl Written Before Profile Dir Exists
**What goes wrong:** RecordUsage called before ConfigDir exists (first-run) causes file write failure.
**Why it happens:** ConfigDir is only created on first profile operation; usage file is in ConfigDir.
**How to avoid:** RecordUsage must call `os.MkdirAll(configDir, 0755)` before opening the file — matching Python line 129. Since RecordUsage swallows errors, a missing dir silently drops the record, which is acceptable.

### Pitfall 4: Tree Cycle Detection
**What goes wrong:** If `created_from` is set to a value that creates a cycle (A created_from B, B created_from A), the recursive render loops indefinitely.
**Why it happens:** created_from is user-controlled data in the manifest.
**How to avoid:** Use a `visited` set in `render_profile` — skip if already visited. Python does this at lines 1208-1213. The Go implementation must do the same.

### Pitfall 5: stats --since Filter Timezone Ambiguity
**What goes wrong:** Usage entries stored with local time, --since filter parses as local time, but comparison may fail across DST boundaries or when migrating machines.
**Why it happens:** Python uses `datetime.datetime.now().isoformat()` which produces local time without timezone info.
**How to avoid:** Match Python behavior: store timestamps with `time.Now().Format(time.RFC3339Nano)` (local time, no UTC normalization). The --since filter compares lexicographically as strings after normalizing both to the same format (append T00:00:00 for date-only input, matching Python line 1085-1086).

### Pitfall 6: Unmanage Handles Shared-Path Symlinks Differently
**What goes wrong:** When a profile has paths in `shared_paths` (pointing to the shared dir), unmanage must materialize those too — the shared dir file, not the symlink.
**Why it happens:** Symlinks in ~/.claude/ can point to either the profile dir or the shared dir; materialize must follow to the real content.
**How to avoid:** `link.Resolve()` / `filepath.EvalSymlinks(link)` follows all symlink levels to the real file. Python uses `link.resolve()` which has the same behavior. Use `os.Stat(link)` (follows symlinks) rather than `os.Lstat(link)` when checking if target exists.

## Code Examples

Verified patterns from project codebase:

### Existing copyFile helper (reuse for pick and unshare)
```go
// Source: internal/profile/shared.go (lines 92-112) — reuse this, don't duplicate
func copyFile(src, dst string) error {
    in, err := os.Open(src)
    // ...preserves permissions...
}
```

### Atomic symlink for share (already used in switch.go)
```go
// Source: internal/profile/switch.go — same pattern applies in share
import "github.com/google/renameio/v2"
if err := renameio.Symlink(realTarget, dst); err != nil {
    return err
}
```

### Manifest mutation pattern (established in Phase 2)
```go
// Load → mutate → save — the standard pattern throughout the codebase
m, err := config.LoadManifest(filepath.Join(profileDir, ".hop-manifest.json"))
if err != nil { return err }
m.SharedPaths[path] = srcName
if err := config.SaveManifest(filepath.Join(profileDir, ".hop-manifest.json"), m); err != nil {
    return err
}
```

### XDG_CONFIG_HOME isolation in tests (established pattern)
```go
// Source: internal/profile/shared_test.go — use for new tests
tmp := t.TempDir()
t.Setenv("XDG_CONFIG_HOME", tmp)
```

### ASCII tree box-drawing (match Python output)
```go
// Connectors
const (
    connectorMid  = "├── "
    connectorLast = "└── "
    prefixMid     = "│   "
    prefixLast    = "    "
)
```

### RecordUsage implementation pattern
```go
// internal/usage/usage.go
func RecordUsage(configDir, profileName, action string) {
    // Must never return error — swallow everything
    _ = os.MkdirAll(configDir, 0755)
    entry := UsageEntry{
        Profile:   profileName,
        Timestamp: time.Now().Format(time.RFC3339Nano),
        Action:    action,
    }
    data, _ := json.Marshal(entry)
    f, err := os.OpenFile(filepath.Join(configDir, "usage.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil { return }
    defer f.Close()
    _, _ = f.Write(append(data, '\n'))
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Global cmd vars + init() | Global vars + init() (existing Phase 2 pattern) | Established in Phase 2 | Phase 3 follows the same pattern; no architecture change needed |
| Python `shutil.copy2` | Go `copyFile` from `internal/profile/shared.go` | Phase 1 | Already implemented; extend to handle directory recursion |
| Python `datetime.datetime.now().isoformat()` | Go `time.Now().Format(time.RFC3339Nano)` | Phase 3 (new) | RFC3339Nano includes sub-second precision; compatible with Python's isoformat() format |

**No deprecated approaches in this phase** — all patterns follow what was established in Phases 1-2.

## Open Questions

1. **Directory copy in pick/unshare**
   - What we know: Python uses `shutil.copytree(src, dst, symlinks=True, ignore_dangling_symlinks=True)` for directory sources
   - What's unclear: Go stdlib has no direct `copytree` equivalent; must implement with `os.ReadDir` + recursive copyFile
   - Recommendation: Implement `copyDirRecursive(src, dst string) error` in `internal/profile/` that preserves symlinks via `os.Readlink` + `os.Symlink` for symlink entries, and uses `copyFile` for regular files. This is straightforward and well within stdlib capabilities.

2. **fatih/color dependency decision**
   - What we know: The project currently has zero color output; Python version has no colors; adding color would be a UX enhancement
   - What's unclear: Whether the user wants color at all given Python parity is a goal
   - Recommendation: Skip fatih/color entirely. Use plain ASCII output matching Python. The active marker `(active)` and `(shared from X)` annotations are sufficient without color. This keeps binary size down and maintains Python output parity. If color is added later it is a non-breaking enhancement.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (no external framework) |
| Config file | none — standard `go test` |
| Quick run command | `go test ./internal/... ./cmd/... -run TestShare\|TestPick\|TestUnshare\|TestTree\|TestDiff\|TestStats\|TestUnmanage\|TestRecordUsage -timeout 30s` |
| Full suite command | `go test ./... -timeout 60s` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SHAR-01 | ShareFiles creates symlink in target profile dir pointing to source file | unit | `go test ./internal/profile/ -run TestShareFiles -v` | ❌ Wave 0 |
| SHAR-01 | ShareFiles updates target manifest shared_paths | unit | `go test ./internal/profile/ -run TestShareFiles_ManifestUpdated -v` | ❌ Wave 0 |
| SHAR-01 | ShareFiles dry-run prints without modifying filesystem | unit | `go test ./internal/profile/ -run TestShareFiles_DryRun -v` | ❌ Wave 0 |
| SHAR-02 | PickFiles copies file content to target profile dir | unit | `go test ./internal/profile/ -run TestPickFiles -v` | ❌ Wave 0 |
| SHAR-02 | PickFiles adds path to target manifest managed_paths | unit | `go test ./internal/profile/ -run TestPickFiles_ManifestUpdated -v` | ❌ Wave 0 |
| SHAR-03 | UnshareFiles replaces symlink with real file copy | unit | `go test ./internal/profile/ -run TestUnshareFiles -v` | ❌ Wave 0 |
| SHAR-03 | UnshareFiles removes path from manifest shared_paths | unit | `go test ./internal/profile/ -run TestUnshareFiles_ManifestCleaned -v` | ❌ Wave 0 |
| VIZ-01 | BuildTree returns correct parent-child structure | unit | `go test ./internal/profile/ -run TestBuildTree -v` | ❌ Wave 0 |
| VIZ-01 | Tree cycle detection (visited set) prevents infinite loop | unit | `go test ./internal/profile/ -run TestBuildTree_Cycle -v` | ❌ Wave 0 |
| VIZ-02 | DiffProfiles returns correct only_a, only_b, common sets | unit | `go test ./internal/profile/ -run TestDiffProfiles -v` | ❌ Wave 0 |
| VIZ-02 | DiffProfiles detects identical vs different files in common | unit | `go test ./internal/profile/ -run TestDiffProfiles_FileComparison -v` | ❌ Wave 0 |
| VIZ-03 | AggregateStats counts switches per profile correctly | unit | `go test ./internal/usage/ -run TestAggregateStats -v` | ❌ Wave 0 |
| VIZ-03 | AggregateStats --since filter excludes older entries | unit | `go test ./internal/usage/ -run TestAggregateStats_Since -v` | ❌ Wave 0 |
| VIZ-04 | hop path prints profile directory path | unit | `go test ./cmd/ -run TestPathCmd -v` | ❌ Wave 0 |
| OPS-01 | UnmanageActive materializes symlinks to real files in claudeDir | unit | `go test ./internal/profile/ -run TestUnmanageActive -v` | ❌ Wave 0 |
| OPS-01 | UnmanageActive sets active to empty string in config | unit | `go test ./internal/profile/ -run TestUnmanageActive_ClearsConfig -v` | ❌ Wave 0 |
| OPS-03 | RecordUsage appends correct JSON line to usage.jsonl | unit | `go test ./internal/usage/ -run TestRecordUsage -v` | ❌ Wave 0 |
| OPS-03 | RecordUsage creates configDir if missing (first-run) | unit | `go test ./internal/usage/ -run TestRecordUsage_CreatesDir -v` | ❌ Wave 0 |
| OPS-03 | RecordUsage never panics on write error (read-only fs) | unit | `go test ./internal/usage/ -run TestRecordUsage_NoError -v` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/profile/ ./internal/usage/ -timeout 30s`
- **Per wave merge:** `go test ./... -timeout 60s`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/usage/usage_test.go` — covers OPS-03 requirements
- [ ] `internal/profile/share_test.go` — covers SHAR-01, SHAR-02, SHAR-03
- [ ] `internal/profile/tree_test.go` — covers VIZ-01
- [ ] `internal/profile/diff_test.go` — covers VIZ-02
- [ ] `internal/profile/unmanage_test.go` — covers OPS-01
- [ ] `internal/usage/` directory — new package, needs creation
- [ ] Framework install: none needed — Go stdlib testing already in place

## Sources

### Primary (HIGH confidence)
- `/home/matthew/Programming/claudehopper/src/claudehopper/cli.py` — Direct reading of Python reference implementation; all command behavior verified from source lines cited above
- `internal/profile/shared.go` — Existing `copyFile`, `LinkDefaultsIntoProfile`, `SharedDir` — verified reusable
- `internal/config/manifest.go` — `LoadManifest`, `SaveManifest` — verified mutation pattern
- `internal/profile/switch.go` — `DoSwitch` with `Force: true` — verified re-link pattern
- `go.mod` — Confirmed `google/renameio/v2 v2.0.2` already present; no new symlink library needed

### Secondary (MEDIUM confidence)
- `internal/profile/shared_test.go` — Verified `t.Setenv("XDG_CONFIG_HOME", tmp)` test isolation pattern
- `cmd/create.go` — Verified `init()` + global var + `rootCmd.AddCommand()` is the established cmd registration pattern for this codebase (not the NewCommand constructor pattern from architecture research)

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries already in go.mod or stdlib; no new dependencies required for core functionality
- Architecture: HIGH — all command behaviors verified against Python source with line citations; established patterns confirmed from existing Phase 2 code
- Pitfalls: HIGH — most pitfalls derived directly from Python source behavior + existing codebase patterns; cycle detection and re-link-active are explicitly present in Python
- Validation: HIGH — test patterns match existing test files in project

**Research date:** 2026-03-14
**Valid until:** 2026-06-14 (stable Go stdlib; Python reference unlikely to change)
