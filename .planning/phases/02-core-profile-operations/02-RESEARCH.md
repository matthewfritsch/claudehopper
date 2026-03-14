# Phase 2: Core Profile Operations - Research

**Researched:** 2026-03-14
**Domain:** Go CLI profile management — file operations, symlinks, Cobra commands, Python behavioral parity
**Confidence:** HIGH

## Summary

Phase 2 builds the full profile lifecycle on top of Phase 1's established primitives. Every piece of infrastructure needed is already in place: `AtomicSymlink`, `IsProtected`, `LoadManifest`/`SaveManifest`, `LoadConfig`/`SaveConfig`, and all path resolution helpers. The implementation work is wiring these into Cobra subcommands and faithfully replicating the Python logic described in `cli.py`.

The biggest risk is behavioral divergence from Python. The Python source at `~/Programming/claudehopper/src/claudehopper/cli.py` is the canonical specification — it must be read before implementing each command, not recalled from memory. Key subtleties: the `created_from` field is written to manifest as a raw JSON field that the current Go `Manifest` struct does not model; the `link_managed_path` function deliberately uses `src.resolve()` for non-symlinks but passes the symlink path directly for symlinks; and the default-linked bootstrapping from a source profile has nuanced copy-vs-symlink logic.

The manifest `created_from` field is the only structural gap between the Python format and the current Go `Manifest` struct. It must be added as an `omitempty` JSON field before any create command is implemented, or the Go version will silently drop lineage information written by the Python version.

**Primary recommendation:** Add `CreatedFrom string` to the Go `Manifest` struct first (with `json:"created_from,omitempty"`), then implement commands in dependency order: `create` → `list` → `status` → `switch` → `delete`. Test each against real Python-generated fixture data.

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **Create from-current behavior:** Match Python's detection logic exactly — capture everything non-protected from `~/.claude/`. Protected paths stay in `~/.claude/` and are shared across all profiles automatically. Existing symlinks in `~/.claude/` are preserved as shared_paths in the new profile's manifest (not resolved to copies). Output format matches Python — show captured files and summary.
- **Switch & dry-run UX:** Normal switch output: Claude decides clean format. Dry-run output matches Python's format (file-by-file action list showing what would change).
- **Adopt-on-switch flow:** When unmanaged files found during switch: Claude decides safest UX (likely prompt user with list of files, offer to adopt into departing profile). Non-interactive mode (stdin not a TTY): skip adoption silently, just switch — safest for scripts and automation.
- **Profile name rules:** Claude decides validation rules (alphanumeric + hyphens + underscores as directory-safe names). Case-insensitive — normalize to lowercase to avoid confusion on macOS (case-insensitive filesystem).

### Claude's Discretion

- Normal switch output format (clean, one summary line or brief)
- Adopt-on-switch interactive prompt design
- Profile name validation rules (within the directory-safe constraint)
- Internal package structure for `internal/profile/`
- How `--activate` flag on create calls switch internally

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| PROF-01 | User can create a blank profile with a name and optional description | Python `cmd_create` (blank path): creates `settings.json`, saves manifest with `["settings.json"]` |
| PROF-02 | User can create a profile from current `~/.claude/` config (`--from-current`) | Python `cmd_create` + `detect_profile_paths()` + file copy/symlink logic |
| PROF-03 | User can clone an existing profile (`--from-profile`) with lineage tracked | Python `cmd_create` clones via `shutil.copytree`, writes `created_from` to manifest |
| PROF-04 | User can list all profiles showing name, active marker, and managed path count | Python `cmd_list()` — format: `  name (active)  [N paths, M shared] - desc` |
| PROF-05 | User can view status of active profile with link health per managed path | Python `cmd_status()` — per-path `[linked]`, `[linked, shared from X]`, `[CONFLICT]`, `[not linked]` |
| PROF-06 | User can delete a profile with warning if other profiles depend on it | Python `cmd_delete()` — scan `shared_paths` in all other profiles for references |
| PROF-07 | User can create and immediately activate a profile (`--activate`) | Python calls `_do_switch(name, force=True)` after create |
| SWCH-01 | User can switch active profile via single command | Python `_do_switch()` — full logic |
| SWCH-02 | Switch uses atomic symlink replacement (tmp + rename, never remove + symlink) | Phase 1 `AtomicSymlink()` already handles this |
| SWCH-03 | User can preview switch with `--dry-run` before applying | Python dry-run: prints action list, returns without writing |
| SWCH-04 | Conflicting files backed up with `.hop-backup` suffix before overwriting | Python `backup_path()` — suffix collision handled with `.hop-backup.1`, `.hop-backup.2`, etc. |
| SWCH-05 | Manifest validated before switch (managed paths exist in profile dir) | Python `validate_switch_preflight()` — checks each path exists in profile dir |
| SWCH-06 | Unmanaged files in `~/.claude/` detected and offered for adoption on switch | Python `detect_unmanaged()` + `adopt_unmanaged()` — see detailed logic below |
| SAFE-02 | Each profile has a `.hop-manifest.json` tracking managed_paths, shared_paths, description, created_from | Go `Manifest` struct needs `CreatedFrom` field added |
| SHAR-04 | New profiles automatically share default linked files (settings.json, settings.local.json, .mcp.json) | Python `link_defaults_into_profile()` + `ensure_shared_defaults()` — shared dir is `~/.config/claudehopper/shared/` |
</phase_requirements>

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/spf13/cobra` | v1.10.2 | CLI subcommand routing | Already in go.mod; Phase 1 uses it |
| `github.com/google/renameio/v2` | v2.0.2 | Atomic symlink replacement | Already in go.mod; `AtomicSymlink` wraps it |
| stdlib `os`, `io/fs`, `path/filepath` | Go 1.26.1 | File operations, directory scanning | No extra deps needed |
| stdlib `bufio`, `os` | Go 1.26.1 | TTY detection (`os.ModeCharDevice`) | Needed for non-interactive adopt skip |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `io/fs` + `os.ReadDir` | stdlib | Directory listing for detect_profile_paths | Any directory scan |
| `strings` + `unicode` | stdlib | Profile name validation, lowercase normalization | `cmd_create` |
| `fmt`, `os.Stderr` | stdlib | Error output matching Python `die()` pattern | Error handling in all commands |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| stdlib testing | testify | testify assertions cleaner but adds dependency; project already uses stdlib only |
| `bufio.Scanner` for TTY | `golang.org/x/term` | x/term more portable but stdlib `os.ModeCharDevice` sufficient |

**Installation:** No new dependencies needed — all required packages are already in `go.mod`.

---

## Architecture Patterns

### Recommended Project Structure

```
cmd/
├── root.go           # existing — add subcommands here
├── create.go         # hop create [--from-current|--from-profile|--activate|--description]
├── list.go           # hop list
├── status.go         # hop status
├── switch.go         # hop switch [--dry-run|--force|--adopt]
└── delete.go         # hop delete [--yes]
internal/
├── config/           # existing — add CreatedFrom to Manifest
│   ├── manifest.go   # add CreatedFrom string field
│   └── ...
├── fs/               # existing
└── profile/          # NEW — pure business logic, no I/O side effects on public API
    ├── create.go     # CreateBlank, CreateFromCurrent, CreateFromProfile
    ├── switch.go     # ValidatePreflight, DoSwitch, DetectUnmanaged, AdoptUnmanaged
    ├── list.go       # ListProfiles, ProfileSummary
    ├── status.go     # ProfileStatus, ManagedPathHealth
    ├── delete.go     # DeleteProfile, FindDependents
    └── shared.go     # EnsureSharedDefaults, LinkDefaultsIntoProfile
```

The `internal/profile/` package holds all business logic as pure functions. Cobra command files in `cmd/` handle flag parsing, I/O, and call into `internal/profile/`. This mirrors the established pattern from Phase 1.

### Pattern 1: Command-to-Package Delegation

**What:** Cobra command files are thin — parse flags, call profile package, print result to stdout/stderr.
**When to use:** Every command in this phase.
**Example:**
```go
// cmd/list.go
var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List all profiles",
    RunE: func(cmd *cobra.Command, args []string) error {
        summaries, err := profile.ListProfiles()
        if err != nil {
            return err
        }
        profile.PrintList(summaries, os.Stdout)
        return nil
    },
}

func init() {
    rootCmd.AddCommand(listCmd)
}
```

### Pattern 2: Manifest CreatedFrom Field (CRITICAL gap to fix first)

**What:** The Python manifest format includes a `created_from` field for lineage. The current Go `Manifest` struct omits it.
**Impact:** Reading a Python-written manifest with `created_from` will silently drop the field on save, breaking cross-version compatibility.
**Fix:** Add to `internal/config/manifest.go` before any create command:
```go
type Manifest struct {
    ManagedPaths []string          `json:"managed_paths"`
    SharedPaths  map[string]string `json:"shared_paths"`
    Description  string            `json:"description"`
    CreatedFrom  string            `json:"created_from,omitempty"`
}
```
The `omitempty` ensures blank profiles serialize cleanly without the field (matching Python behavior where `created_from` is absent for blank profiles).

### Pattern 3: Atomic Switch Sequence

**What:** Profile switch must be all-or-nothing from the user's perspective.
**When to use:** `hop switch` command.
**Sequence (matches Python `_do_switch`):**
1. Validate: check active profile name matches config, require_profile target exists
2. `validate_switch_preflight` — verify all managed paths exist in target profile dir
3. If dry-run: print action list and return (no writes)
4. If current profile exists: `detect_unmanaged`, offer adoption if TTY
5. Unlink current profile's managed paths from `~/.claude/`
6. For each path in target manifest: `link_managed_path` (backup conflicts, atomic symlink)
7. `SaveConfig` with new active name
8. Print summary

**Critical:** Never remove symlinks before you've validated the target. Validation happens in step 2, writes happen in steps 5-7.

### Pattern 4: TTY Detection for Non-Interactive Mode

**What:** Adopt-on-switch prompt must be skipped when stdin is not a terminal.
**Implementation:**
```go
// isInteractive reports whether stdin is a terminal (not a pipe or script).
func isInteractive() bool {
    fi, err := os.Stdin.Stat()
    if err != nil {
        return false
    }
    return fi.Mode()&os.ModeCharDevice != 0
}
```

### Pattern 5: Backup Collision Avoidance

**What:** Python generates `.hop-backup`, `.hop-backup.1`, `.hop-backup.2` etc. to avoid overwriting existing backups.
**Go implementation:**
```go
func backupPath(path string) string {
    candidate := path + ".hop-backup"
    if _, err := os.Lstat(candidate); os.IsNotExist(err) {
        return candidate
    }
    for n := 1; ; n++ {
        candidate = fmt.Sprintf("%s.hop-backup.%d", path, n)
        if _, err := os.Lstat(candidate); os.IsNotExist(err) {
            return candidate
        }
    }
}
```
Use `os.Lstat` (not `os.Stat`) so dangling symlinks are also detected as existing.

### Pattern 6: link_managed_path — Symlink Target Resolution

**What:** Python's `link_managed_path` uses `src.resolve()` for regular files/dirs but passes the symlink path directly for symlinks-in-profile-dir.
**Why:** A symlink in the profile dir (e.g., a DEFAULT_LINKED file pointing to the shared dir) should have its destination preserved as-is; a regular file should be linked by absolute path.
```go
// Go equivalent
func linkManagedPath(profileDir, claudeDir, name string) (bool, error) {
    src := filepath.Join(profileDir, name)
    link := filepath.Join(claudeDir, name)

    // Remove whatever is at the link location
    if info, err := os.Lstat(link); err == nil {
        if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
            // Real directory — back up before replacing
            bak := backupPath(link)
            if err := os.Rename(link, bak); err != nil {
                return false, err
            }
        } else {
            os.Remove(link)
        }
    }

    // Determine symlink target
    srcInfo, err := os.Lstat(src)
    if err != nil {
        return false, nil // src missing — skip
    }
    var target string
    if srcInfo.Mode()&os.ModeSymlink != 0 {
        // Preserve the symlink destination as-is
        target, err = os.Readlink(src)
        if err != nil {
            return false, err
        }
    } else {
        // Use absolute path for regular files/dirs
        target, err = filepath.Abs(src)
        if err != nil {
            return false, err
        }
    }

    return true, fs.AtomicSymlink(target, link)
}
```

### Pattern 7: Profile Name Validation and Normalization

**What:** Names must be directory-safe and case-normalized to lowercase.
**Python rules (for reference):** Cannot be empty, cannot start with `.` or `-`, cannot contain `/`, `\`, `\0`, cannot be `.` or `..`.
**Go extension (Claude's discretion):** Also reject names containing only whitespace; strip and lowercase before validation.
```go
var validProfileName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

func ValidateProfileName(name string) error {
    name = strings.ToLower(strings.TrimSpace(name))
    if name == "" {
        return fmt.Errorf("profile name cannot be empty")
    }
    if !validProfileName.MatchString(name) {
        return fmt.Errorf("profile name must contain only letters, digits, hyphens, and underscores, and must not start with a hyphen: %q", name)
    }
    return nil
}

func NormalizeProfileName(name string) string {
    return strings.ToLower(strings.TrimSpace(name))
}
```

### Anti-Patterns to Avoid

- **`os.Remove` + `os.Symlink` for switch:** Never — use `AtomicSymlink` for all symlink operations. The two-step sequence has a window where the link is absent.
- **`os.Stat` on symlinks:** Always use `os.Lstat` when checking existence of something that might be a symlink (managed paths, backup candidates).
- **Writing `Manifest` fields not in the struct:** Adding `created_from` via raw `json.RawMessage` manipulation is fragile. Add it to the struct with `omitempty`.
- **Calling profile business logic from `init()`:** Cobra `init()` runs at program startup; put logic in `RunE`, not `init()`.
- **Non-interactive detection via `os.Getenv("TERM")`:** Not reliable. Use `os.Stdin.Stat()` and check `ModeCharDevice`.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Atomic symlink replacement | custom tmp + rename loop | `fs.AtomicSymlink` (Phase 1) | Already tested, handles all edge cases |
| Protected path filtering | inline set membership check | `fs.IsProtected(name)` (Phase 1) | Drift-tested against Python SHARED_PATHS |
| Path resolution | string manipulation | `config.ProfileDir(name)`, `config.ConfigDir()` etc. | XDG-aware, already tested |
| Config read/write | `os.ReadFile` + `json.Marshal` inline | `config.LoadConfig`, `config.SaveConfig` | Format-compatible with Python, handles missing file |
| Manifest read/write | inline JSON | `config.LoadManifest`, `config.SaveManifest` | Sorts ManagedPaths, handles nil collections, round-trip tested |
| Directory copy for `--from-profile` | custom recursive copy | `os.CopyFS` (Go 1.23+) or `filepath.Walk` with `io.Copy` | Handles permissions, preserves symlinks when using `os.Lstat`; Go 1.26 has `os.CopyFS` |

**Key insight:** The existing Phase 1 infrastructure is specifically designed for this phase. Using it consistently keeps behavior correct and tests focused on business logic rather than primitives.

---

## Common Pitfalls

### Pitfall 1: Missing `created_from` in Manifest Struct

**What goes wrong:** Reading a Python manifest with `created_from` and then saving it drops the field silently. The `VIZ-01` tree command in Phase 3 depends on this field.
**Why it happens:** Go's `json.Unmarshal` ignores unknown fields by default; `json.MarshalIndent` only serializes struct fields.
**How to avoid:** Add `CreatedFrom string \`json:"created_from,omitempty"\`` to `Manifest` before implementing any create command.
**Warning signs:** A round-trip test that loads a Python manifest with `created_from` and saves it — the output should be byte-identical to the input.

### Pitfall 2: Symlink Target Resolution Mismatch

**What goes wrong:** Using `filepath.EvalSymlinks(src)` (resolves all components) instead of `os.Readlink(src)` (reads one link level) for symlinks in the profile dir. This causes DEFAULT_LINKED symlinks (which point to `~/.config/claudehopper/shared/`) to be re-linked as the resolved absolute target rather than preserving the original symlink destination.
**Why it happens:** `filepath.EvalSymlinks` is the natural "get real path" function but it dereferences transitively.
**How to avoid:** Branch on `os.Lstat` result: if symlink, use `os.Readlink`; if not, use `filepath.Abs`.

### Pitfall 3: Unmanaged Detection Skipping `.hop-*` and `.hop-backup` Entries

**What goes wrong:** `detect_unmanaged` counting `.hop-manifest.json`, `.hop-backup` files, or `.ccswap` files as unmanaged. This pollutes the adopt prompt with internal tooling files.
**Why it happens:** Iterating `CLAUDE_DIR` without the same filter logic Python uses.
**How to avoid:** Replicate Python's exact filter: skip `SHARED_PATHS` members, skip names starting with `.hop-`, skip names ending with `.hop-backup` or matching `.hop-backup.\d+`, skip names starting with `.ccswap`. Also skip symlinks pointing into the shared dir.

### Pitfall 4: Backup Suffix Collision Using `os.Stat`

**What goes wrong:** `os.Stat` on a dangling symlink returns `ErrNotExist` (because Stat follows the link), so dangling `.hop-backup` symlinks appear non-existent and get overwritten.
**Why it happens:** Reflexive use of `os.Stat` instead of `os.Lstat`.
**How to avoid:** Always use `os.Lstat` in `backupPath` to check backup candidate existence.

### Pitfall 5: Switch When Already on Same Profile

**What goes wrong:** `hop switch work` when already on `work` either does nothing silently or re-links unnecessarily.
**Python behavior:** Print `Already on 'name'. Use --force to re-link.` and return.
**How to avoid:** Check `config.Active == name && !force` at the top of `DoSwitch`, print the message, return nil.

### Pitfall 6: `from-current` with Existing Symlinks in `~/.claude/`

**What goes wrong:** Copying a symlink from `~/.claude/` into the profile dir with `io.Copy` dereferences it, creating a file copy instead of a symlink.
**Python behavior:** `os.symlink(os.readlink(src), dst)` preserves the symlink as-is.
**How to avoid:** Check `os.Lstat` first; if symlink, use `os.Symlink(os.Readlink(src), dst)` not any copy function.

### Pitfall 7: Default Linked Files Bootstrap Logic

**What goes wrong:** On first profile creation, the shared dir may not exist or may not have `settings.json` yet. Skipping the bootstrap means DEFAULT_LINKED files are never shared.
**Python behavior:** `ensure_shared_defaults` creates the shared dir; `link_defaults_into_profile` seeds the shared file from `from_source` if it exists there and the shared copy is missing.
**How to avoid:** Implement `EnsureSharedDefaults` (create dir) and `LinkDefaultsIntoProfile` (bootstrap + symlink) faithfully. For `--from-current`, `from_source = pdir` (the newly-created profile dir). For `--from-profile`, `from_source = pdir` as well. For blank create, `from_source = nil`.

---

## Code Examples

Verified patterns from Python source (behavioral specification):

### detect_profile_paths — what to capture from `~/.claude/`

```python
# Source: cli.py:207-216
def detect_profile_paths() -> list[str]:
    return sorted(
        item.name for item in CLAUDE_DIR.iterdir()
        if item.name not in SHARED_PATHS
        and not item.name.startswith(".hop-")
        and not item.name.startswith(".ccswap")
    )
```
Go equivalent: `os.ReadDir(claudeDir)`, filter with `fs.IsProtected(name)`, `strings.HasPrefix(name, ".hop-")`, `strings.HasPrefix(name, ".ccswap")`.

### validate_switch_preflight — pre-switch validation

```python
# Source: cli.py:241-285
# Returns list of (action, detail) tuples
# Errors if any managed path missing from profile dir
# Actions include: "unlink", "orphan", "backup", "link"
```

### _do_switch — full switch sequence

```python
# Source: cli.py:673-739
# Guard: current == name && !force → print "Already on..."
# validate_switch_preflight (may die on errors)
# dry_run → print actions, return
# detect_unmanaged (if current profile) → offer adoption if TTY
# unlink current profile's managed paths
# link target profile's managed paths (backup conflicts)
# save_config with new active
# print "Switched to 'name' (N paths linked)"
```

### cmd_delete — dependent warning

```python
# Source: cli.py:968-1001
# Refuse to delete active profile
# Scan all other profiles' shared_paths for references to this profile name
# If dependents found: warn, optionally prompt (--yes skips)
# shutil.rmtree(pdir)
```

### save_manifest — Python reference format (already matched by Go)

```python
# Source: cli.py:196-204
# data = {
#     "managed_paths": sorted(set(managed_paths)),
#     "shared_paths": shared_paths or existing["shared_paths"],
#     "description": description or existing["description"],
# }
# (pdir / MANIFEST_NAME).write_text(json.dumps(data, indent=2) + "\n")
```
Note: `created_from` is written separately (line 605-606) by directly mutating the manifest dict. The Go version should include it in the struct and let `SaveManifest` handle it.

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `os.Remove` + `os.Symlink` | `renameio.Symlink` (atomic) | Phase 1 decision | No window where link is missing |
| Direct `json.Marshal` on struct | `SaveManifest` with sorting | Phase 1 | Sorted output, Python format parity |
| Monolithic command file | `cmd/` + `internal/profile/` split | Phase 2 design | Business logic testable without I/O |

**Deprecated/outdated:**
- Python's manual `tmp.symlink_to(target); tmp.rename(link)` pattern: replaced by `renameio.Symlink` in Go — do not replicate the manual pattern.

---

## Open Questions

1. **`os.CopyFS` availability for `--from-profile`**
   - What we know: Go 1.23 added `os.CopyFS`; this project uses Go 1.26.1
   - What's unclear: `os.CopyFS` copies from an `fs.FS` — it may not preserve symlinks as symlinks
   - Recommendation: Use `filepath.Walk` + `os.Lstat` + manual copy to ensure symlinks are preserved as symlinks (matching Python's `shutil.copytree(symlinks=True)`)

2. **`--activate` calling switch with `force=true`**
   - What we know: Python calls `_do_switch(name, force=True)` after create
   - What's unclear: This bypasses the "already active" guard but also bypasses unmanaged detection (newly created profile, so no current profile yet, or user just switched away)
   - Recommendation: The `--activate` flag calls `DoSwitch(name, SwitchOptions{Force: true})` after create succeeds. The `force=true` is correct because there is no "current profile" in the typical create-and-activate flow.

3. **SharedPaths value for DEFAULT_LINKED**
   - What we know: Python writes `shared_paths[filename] = "(shared)"` for DEFAULT_LINKED files, using the literal string `"(shared)"` rather than a source profile name
   - What's unclear: The CONTEXT.md says "Existing symlinks in `~/.claude/` are preserved as shared_paths in the new profile's manifest" — unclear if this uses the same `"(shared)"` sentinel or a profile name
   - Recommendation: Use `"(shared)"` as the source value for DEFAULT_LINKED files (matches Python exactly). For symlinks captured from `~/.claude/` during `--from-current`, preserve them as shared_paths with `"(shared)"` as the source since we can't determine the original profile name.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` |
| Config file | none — `go test ./...` is sufficient |
| Quick run command | `cd /home/matthew/Programming/claudehopper-go && go test ./cmd/... ./internal/...` |
| Full suite command | `cd /home/matthew/Programming/claudehopper-go && go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PROF-01 | CreateBlank makes dir + manifest + settings.json | unit | `go test ./internal/profile/... -run TestCreateBlank` | Wave 0 |
| PROF-02 | CreateFromCurrent captures non-protected paths | unit | `go test ./internal/profile/... -run TestCreateFromCurrent` | Wave 0 |
| PROF-03 | CreateFromProfile copies dir + preserves symlinks + writes created_from | unit | `go test ./internal/profile/... -run TestCreateFromProfile` | Wave 0 |
| PROF-04 | ListProfiles returns correct count + active marker | unit | `go test ./internal/profile/... -run TestListProfiles` | Wave 0 |
| PROF-05 | ProfileStatus shows link health correctly | unit | `go test ./internal/profile/... -run TestProfileStatus` | Wave 0 |
| PROF-06 | DeleteProfile warns on dependents | unit | `go test ./internal/profile/... -run TestDeleteProfile` | Wave 0 |
| PROF-07 | --activate calls switch after create | unit | `go test ./cmd/... -run TestCreateActivate` | Wave 0 |
| SWCH-01 | DoSwitch updates symlinks and config | unit | `go test ./internal/profile/... -run TestDoSwitch` | Wave 0 |
| SWCH-02 | Symlink replacement is atomic (no TOCTOU window) | unit | `go test ./internal/fs/... -run TestAtomicSymlink` | ✅ |
| SWCH-03 | DryRun returns action list without writing | unit | `go test ./internal/profile/... -run TestDryRun` | Wave 0 |
| SWCH-04 | Conflicting real file gets .hop-backup suffix | unit | `go test ./internal/profile/... -run TestBackupConflict` | Wave 0 |
| SWCH-05 | Switch fails if manifest lists missing path | unit | `go test ./internal/profile/... -run TestValidatePreflight` | Wave 0 |
| SWCH-06 | Unmanaged files detected; non-TTY skips prompt | unit | `go test ./internal/profile/... -run TestDetectUnmanaged` | Wave 0 |
| SAFE-02 | Manifest round-trips created_from field | unit | `go test ./internal/config/... -run TestManifest_CreatedFrom` | Wave 0 |
| SHAR-04 | New profile gets DEFAULT_LINKED symlinks | unit | `go test ./internal/profile/... -run TestLinkDefaults` | Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./internal/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/profile/create_test.go` — covers PROF-01, PROF-02, PROF-03, PROF-07
- [ ] `internal/profile/switch_test.go` — covers SWCH-01, SWCH-03, SWCH-04, SWCH-05, SWCH-06
- [ ] `internal/profile/list_test.go` — covers PROF-04
- [ ] `internal/profile/status_test.go` — covers PROF-05
- [ ] `internal/profile/delete_test.go` — covers PROF-06
- [ ] `internal/profile/shared_test.go` — covers SHAR-04
- [ ] `internal/config/manifest_created_from_test.go` — covers SAFE-02
- [ ] `internal/profile/testdata/` — Python-generated manifest fixtures with `created_from` field

---

## Sources

### Primary (HIGH confidence)

- `/home/matthew/Programming/claudehopper/src/claudehopper/cli.py` — canonical Python implementation; all behavioral specs derived from here
- `/home/matthew/Programming/claudehopper-go/internal/config/manifest.go` — Go Manifest struct definition
- `/home/matthew/Programming/claudehopper-go/internal/fs/atomic.go` — AtomicSymlink implementation
- `/home/matthew/Programming/claudehopper-go/internal/fs/protected.go` — IsProtected + sharedPaths set
- `/home/matthew/Programming/claudehopper-go/internal/config/paths.go` — path resolution helpers
- `/home/matthew/Programming/claudehopper-go/go.mod` — confirmed Go 1.26.1, cobra v1.10.2, renameio v2.0.2
- `.planning/phases/02-core-profile-operations/02-CONTEXT.md` — locked user decisions

### Secondary (MEDIUM confidence)

- Go stdlib `os` docs (Lstat vs Stat distinction for symlinks) — standard Go idiom
- Go stdlib `os.ModeCharDevice` for TTY detection — documented behavior

### Tertiary (LOW confidence)

- `os.CopyFS` symlink behavior in Go 1.23+ — not verified against docs; recommendation is conservative (use filepath.Walk instead)

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all dependencies already in go.mod, Phase 1 established
- Architecture: HIGH — Python source is the specification, Go primitives are ready
- Pitfalls: HIGH — identified from direct Python source reading, not guesswork
- `created_from` manifest gap: HIGH — confirmed by inspecting both Go struct and Python save logic
- `os.CopyFS` symlink behavior: LOW — not verified against Go 1.26 docs

**Research date:** 2026-03-14
**Valid until:** 2026-04-14 (stable domain — Go stdlib + project-specific code)
