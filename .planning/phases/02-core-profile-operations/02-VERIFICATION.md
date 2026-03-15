---
phase: 02-core-profile-operations
verified: 2026-03-14T00:30:00Z
status: passed
score: 27/27 must-haves verified
re_verification: false
---

# Phase 2: Core Profile Operations Verification Report

**Phase Goal:** An existing Python claudehopper user can migrate and perform all daily profile management tasks with the Go binary
**Verified:** 2026-03-14T00:30:00Z
**Status:** PASSED
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

All must-haves are drawn from PLAN frontmatter across plans 01-04.

**Plan 01 truths:**

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Manifest round-trips created_from field without data loss between Python and Go versions | VERIFIED | `internal/config/manifest.go`: `CreatedFrom string \`json:"created_from,omitempty"\`` declared first in struct (matching Python key order). Fixture `hop-manifest-with-lineage.json` loads correctly. |
| 2 | User can create a blank profile with name and description that produces a valid manifest | VERIFIED | `CreateBlank` in `create.go` creates dir, writes `settings.json`, calls `LinkDefaultsIntoProfile`, saves manifest with `managed_paths=["settings.json"]`. |
| 3 | User can create a profile from current ~/.claude/ that captures all non-protected, non-hop files | VERIFIED | `CreateFromCurrent` in `create.go` uses triple filter: `fs.IsProtected`, `.hop-` prefix, `.ccswap` prefix. All three guards present. |
| 4 | User can clone an existing profile with lineage tracked via created_from | VERIFIED | `CreateFromProfile` in `create.go`: `srcM.CreatedFrom = sourceName` set before `SaveManifest`. |
| 5 | New profiles automatically get DEFAULT_LINKED symlinks to shared directory | VERIFIED | All three create functions call `LinkDefaultsIntoProfile`. `DefaultLinked = []string{"settings.json", "settings.local.json", ".mcp.json"}` matches Python constant. |
| 6 | Profile names are validated as directory-safe and normalized to lowercase | VERIFIED | `ValidateProfileName` uses `^[a-z0-9][a-z0-9_-]*$` regex. `NormalizeProfileName` does `strings.ToLower(strings.TrimSpace(name))`. All cmd entry points normalize before calling create. |

**Plan 02 truths:**

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 7 | User can list all profiles with name, active marker, and managed path count | VERIFIED | `ListProfiles` reads all dirs, loads manifests, marks active from `cfg.Active`. `FormatProfileList` outputs `name (active)  [N paths, M shared] - desc`. |
| 8 | User can view active profile status with per-path link health | VERIFIED | `GetProfileStatus` + `checkLinkHealth` classifies each path as linked/shared/conflict/not-linked/broken using `os.Lstat` + `os.Readlink`. |
| 9 | User can delete a profile and is warned if other profiles depend on it | VERIFIED | `FindDependents` scans `shared_paths` values and `created_from` fields. `DeleteProfile` returns `*DependentError` when dependents found. `cmd/delete.go` prompts or accepts `--yes`. |

**Plan 03 truths:**

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 10 | User can switch active profile and all managed symlinks update atomically | VERIFIED | `DoSwitch` calls `linkManagedPath` for each managed path; `linkManagedPath` calls `fs.AtomicSymlink` (uses `renameio.Symlink` — POSIX rename(2)). Config saved after all links. |
| 11 | User can preview a switch with --dry-run and see what would change without writing anything | VERIFIED | `DoSwitch` returns early with planned `actions` when `opts.DryRun == true`, before any `os.Remove`/`os.Rename`/`SaveConfig` call. `cmd/switch.go` prints `would link:`/`would backup:` format. |
| 12 | Conflicting real files are backed up with .hop-backup suffix before overwriting | VERIFIED | `linkManagedPath` detects real file/dir via `os.Lstat`, calls `os.Rename(claudePath, backupPath(claudePath))`. `backupPath` produces `.hop-backup`, `.hop-backup.1`, etc. with Lstat collision avoidance. |
| 13 | Manifest is validated before switch: all managed paths must exist in target profile dir | VERIFIED | `ValidatePreflight` checks `os.Lstat(filepath.Join(profileDir, name))` for every `ManagedPaths` entry. Returns error listing missing paths before any writes. |
| 14 | Unmanaged files in ~/.claude/ are detected on switch | VERIFIED | `DetectUnmanaged` filters: managed set, `fs.IsProtected`, `.hop-` prefix, `.ccswap` prefix, `.hop-backup` substring, symlinks to sharedDir. Called in `cmd/switch.go` before `DoSwitch`. |
| 15 | Non-interactive mode silently skips adoption prompt | VERIFIED | `cmd/switch.go` wraps adopt prompt in `if err == nil && isInteractive()`. `isInteractive()` uses `os.Stdin.Stat()` + `ModeCharDevice` check. |

**Plan 04 truths:**

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 16 | hop create NAME creates a blank profile | VERIFIED | `cmd/create.go` default branch calls `profile.CreateBlank`. Registered via `rootCmd.AddCommand(createCmd)`. |
| 17 | hop create NAME --from-current captures current config | VERIFIED | `createFromCurrent` flag routes to `profile.CreateFromCurrent`. Outputs each captured file path then summary. |
| 18 | hop create NAME --from-profile=source clones a profile | VERIFIED | `createFromProfile` flag routes to `profile.CreateFromProfile`. |
| 19 | hop create NAME --activate creates and switches to profile | VERIFIED | After successful create, `createActivate` check calls `profile.DoSwitch` with `Force: true`. |
| 20 | hop list shows all profiles with active marker | VERIFIED | `cmd/list.go` calls `profile.ListProfiles` then `profile.FormatProfileList`. |
| 21 | hop status shows active profile link health | VERIFIED | `cmd/status.go` loads manifest, calls `profile.GetProfileStatus` + `profile.FormatProfileStatus`. |
| 22 | hop switch NAME switches profiles atomically | VERIFIED | `cmd/switch.go` calls `profile.DoSwitch`. |
| 23 | hop switch NAME --dry-run previews without writing | VERIFIED | `switchDryRun` flag passed as `opts.DryRun`. |
| 24 | hop delete NAME removes profile with dependent warning | VERIFIED | `cmd/delete.go` type-asserts `*profile.DependentError`, prompts or uses `--yes`. |

**Score:** 24/24 truths verified

---

## Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/config/manifest.go` | Manifest struct with CreatedFrom field | VERIFIED | `CreatedFrom string \`json:"created_from,omitempty"\`` present, first field in struct |
| `internal/profile/create.go` | CreateBlank, CreateFromCurrent, CreateFromProfile | VERIFIED | All three functions exported and substantive (not stubs) |
| `internal/profile/shared.go` | EnsureSharedDefaults, LinkDefaultsIntoProfile | VERIFIED | Both functions present with full implementation |
| `internal/profile/validate.go` | ValidateProfileName, NormalizeProfileName | VERIFIED | Both functions present |
| `internal/profile/list.go` | ProfileSummary, ListProfiles | VERIFIED | Struct and function exported, substantive |
| `internal/profile/status.go` | PathHealth, ProfileStatusInfo, GetProfileStatus | VERIFIED | All types and function present and substantive |
| `internal/profile/delete.go` | DeleteProfile, FindDependents | VERIFIED | Both functions present; DependentError type implemented |
| `internal/profile/switch.go` | DoSwitch, SwitchOptions, SwitchAction, ValidatePreflight, DetectUnmanaged | VERIFIED | All exports present, full implementation |
| `cmd/create.go` | Cobra create command with --from-current, --from-profile, --activate, --description | VERIFIED | All four flags registered; `rootCmd.AddCommand(createCmd)` in init() |
| `cmd/list.go` | Cobra list command | VERIFIED | `rootCmd.AddCommand(listCmd)` in init() |
| `cmd/status.go` | Cobra status command | VERIFIED | `rootCmd.AddCommand(statusCmd)` in init() |
| `cmd/switch.go` | Cobra switch command with --dry-run, --force | VERIFIED | Both flags registered; `rootCmd.AddCommand(switchCmd)` in init() |
| `cmd/delete.go` | Cobra delete command with --yes | VERIFIED | Flag registered; `rootCmd.AddCommand(deleteCmd)` in init() |
| `cmd/helpers.go` | isInteractive(), claudeDir() | VERIFIED | Both functions present with proper implementations |
| `internal/config/testdata/hop-manifest-with-lineage.json` | Python-format manifest fixture | VERIFIED | Present; correct JSON with `created_from` key |

---

## Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/profile/create.go` | `internal/config/manifest.go` | `config.SaveManifest` | WIRED | Called in CreateBlank, CreateFromCurrent, CreateFromProfile |
| `internal/profile/create.go` | `internal/profile/shared.go` | `LinkDefaultsIntoProfile` | WIRED | Called in all three create functions |
| `internal/profile/create.go` | `internal/fs/protected.go` | `fs.IsProtected` | WIRED | Line 97 of create.go: `if fs.IsProtected(entryName)` |
| `internal/profile/list.go` | `internal/config/manifest.go` | `config.LoadManifest` | WIRED | Line 49: `m, err := config.LoadManifest(manifestPath)` |
| `internal/profile/status.go` | `internal/fs/atomic.go` | `os.Lstat` for symlink health | WIRED | `checkLinkHealth` uses `os.Lstat` throughout |
| `internal/profile/delete.go` | `internal/config/manifest.go` | Scan shared_paths across all profiles | WIRED | Uses local `manifestForDependents` struct + `json.Unmarshal`; `FindDependents` scans all profiles |
| `internal/profile/switch.go` | `internal/fs/atomic.go` | `fs.AtomicSymlink` for every managed path | WIRED | Line 144: `if err := fs.AtomicSymlink(target, claudePath)` |
| `internal/profile/switch.go` | `internal/config/config.go` | `config.SaveConfig` to persist active profile | WIRED | Line 302: `if err := config.SaveConfig(configPath, cfg)` |
| `internal/profile/switch.go` | `internal/config/manifest.go` | `config.LoadManifest` for target profile | WIRED | Line 244: `config.LoadManifest(filepath.Join(targetProfileDir, ".hop-manifest.json"))` |
| `internal/profile/switch.go` | `internal/fs/protected.go` | `fs.IsProtected` in DetectUnmanaged | WIRED | Line 181: `if fs.IsProtected(name)` |
| `cmd/create.go` | `internal/profile/create.go` | `profile.CreateBlank/CreateFromCurrent/CreateFromProfile` | WIRED | Lines 70, 98, 104 respectively |
| `cmd/switch.go` | `internal/profile/switch.go` | `profile.DoSwitch` | WIRED | Line 96: `result, err := profile.DoSwitch(...)` |
| `cmd/list.go` | `internal/profile/list.go` | `profile.ListProfiles` | WIRED | Line 34: `summaries, err := profile.ListProfiles(...)` |
| `cmd/delete.go` | `internal/profile/delete.go` | `profile.DeleteProfile` | WIRED | Line 51: `err = profile.DeleteProfile(...)` |
| `cmd/status.go` | `internal/profile/status.go` | `profile.GetProfileStatus, FormatProfileStatus` | WIRED | Lines 57-58: both called in sequence |

---

## Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| PROF-01 | 02-01, 02-04 | User can create a blank profile with a name and optional description | SATISFIED | `CreateBlank` + `cmd/create.go` default branch |
| PROF-02 | 02-01, 02-04 | User can create a profile from their current `~/.claude/` config | SATISFIED | `CreateFromCurrent` + `--from-current` flag in `cmd/create.go` |
| PROF-03 | 02-01, 02-04 | User can clone an existing profile with lineage tracked in manifest | SATISFIED | `CreateFromProfile` sets `CreatedFrom`; `--from-profile` flag wired |
| PROF-04 | 02-02, 02-04 | User can list all profiles showing name, active marker, and managed path count | SATISFIED | `ListProfiles` + `FormatProfileList` + `cmd/list.go` |
| PROF-05 | 02-02, 02-04 | User can view status of active profile with link health per managed path | SATISFIED | `GetProfileStatus` + `FormatProfileStatus` + `cmd/status.go` |
| PROF-06 | 02-02, 02-04 | User can delete a profile with warning if other profiles depend on it | SATISFIED | `DeleteProfile` + `DependentError` + `cmd/delete.go` with prompt and `--yes` |
| PROF-07 | 02-01, 02-04 | User can create and immediately activate a profile (`--activate`) | SATISFIED | `--activate` flag in `cmd/create.go` calls `DoSwitch` after create |
| SWCH-01 | 02-03, 02-04 | User can switch active profile via single command | SATISFIED | `DoSwitch` + `cmd/switch.go` |
| SWCH-02 | 02-03, 02-04 | Switch uses atomic symlink replacement (tmp + rename, never remove + symlink) | SATISFIED | `linkManagedPath` calls `fs.AtomicSymlink` which uses `renameio.Symlink` (rename(2) atomic) |
| SWCH-03 | 02-03, 02-04 | User can preview switch with `--dry-run` before applying | SATISFIED | `DoSwitch` returns early with planned actions on `DryRun=true`; cmd prints `would link:` format |
| SWCH-04 | 02-03, 02-04 | Conflicting files are backed up with `.hop-backup` suffix before overwriting | SATISFIED | `backupPath` + `os.Rename` in `linkManagedPath`; collision avoidance via Lstat loop |
| SWCH-05 | 02-03, 02-04 | Manifest is validated before switch (managed paths exist in profile dir) | SATISFIED | `ValidatePreflight` called before any writes in `DoSwitch` |
| SWCH-06 | 02-03, 02-04 | Unmanaged files in `~/.claude/` are detected and offered for adoption on switch | SATISFIED | `DetectUnmanaged` + adopt prompt in `cmd/switch.go` gated by `isInteractive()` |
| SAFE-02 | 02-01, 02-04 | Each profile has a `.hop-manifest.json` tracking managed_paths, shared_paths, description, created_from | SATISFIED | All create modes call `config.SaveManifest` to write `.hop-manifest.json`; `Manifest` struct has all four fields |
| SHAR-04 | 02-01, 02-04 | New profiles automatically share default linked files (settings.json, settings.local.json, .mcp.json) | SATISFIED | `DefaultLinked` slice + `LinkDefaultsIntoProfile` called by all three create functions |

All 15 requirement IDs from the phase plans are accounted for. No orphaned requirements found for Phase 2.

---

## Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `cmd/create.go` | 18 | "placeholder settings.json" | Info | In docstring only — `{}` written is intentional empty JSON, not a code stub |
| `internal/profile/create.go` | 34 | "placeholder" comment | Info | Comment describes intentional empty `{}` sentinel, not a code stub |
| `internal/profile/create.go` | 260-278 | `copyFileWithPerm`, `copyFileIO` unused aliases | Warning | Dead code — two functions are aliases/duplicates of `copyFile` that are not called. Not blocking but bloats the file. |

No blocker anti-patterns found. The two dead code functions in `create.go` are cosmetic.

---

## Human Verification Required

### 1. Migration smoke test from real Python claudehopper installation

**Test:** On a machine with an existing Python claudehopper profile directory (e.g., `~/.config/claudehopper/profiles/`), run `hop list` and verify the profiles appear correctly, then `hop switch <existing-profile>` and verify symlinks are created in `~/.claude/`.
**Expected:** Existing Python-format manifests (with `created_from` key) load without error; symlinks created correctly.
**Why human:** Requires a live Python claudehopper installation to test true migration path.

### 2. TTY adopt-on-switch prompt

**Test:** Run `hop switch other-profile` in an interactive terminal where `~/.claude/` has unmanaged files.
**Expected:** A list of unmanaged files is printed to stderr, followed by `Adopt these N file(s) into profile 'X'? [y/N]`. Answering `y` moves files and updates the manifest.
**Why human:** `isInteractive()` returns false in any test harness — the prompt path cannot be reached programmatically.

### 3. Dry-run output format matches Python

**Test:** Run `hop switch other-profile --dry-run` and compare output line-by-line against the Python `hop switch other-profile --dry-run` output.
**Expected:** Format `would link: settings.json` / `would backup: foo.txt` matches Python style exactly.
**Why human:** Requires a real Python installation for side-by-side comparison.

---

## Build and Test Health

- `go test ./... -race`: ALL PASS (4 packages: cmd, internal/config, internal/fs, internal/profile)
- `go build ./...`: CLEAN
- `go vet ./...`: CLEAN
- Binary `--help`: All 5 commands (create, delete, list, status, switch) registered and visible

---

## Summary

Phase 2 goal is achieved. All 15 required requirement IDs are satisfied with substantive implementations wired from CLI through business logic to filesystem operations. The test suite passes with `-race` across all packages with no failures.

Two minor notes that do not block the goal:
1. `copyFileWithPerm` and `copyFileIO` in `create.go` are dead code — unused aliases of `copyFile`. Harmless but could be cleaned up in a future polish pass.
2. Three human-verification items require a live environment to confirm UX flows (migration from Python, TTY adopt prompt, dry-run format parity).

---

_Verified: 2026-03-14T00:30:00Z_
_Verifier: Claude (gsd-verifier)_
