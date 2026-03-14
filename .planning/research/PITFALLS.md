# Pitfalls Research

**Domain:** Go CLI tool — symlink-based config profile manager (port from Python)
**Researched:** 2026-03-14
**Confidence:** HIGH (symlink/fs operations, Go stdlib behavior); MEDIUM (Cobra gotchas, goreleaser edge cases)

---

## Critical Pitfalls

### Pitfall 1: Non-Atomic Symlink Replacement

**What goes wrong:**
The naive approach to "update" a symlink is `os.Remove(path)` then `os.Symlink(target, path)`. Between those two calls there is a window where the symlink does not exist. If the process is interrupted (signal, power loss, Ctrl-C), the managed path is left dangling — or absent — and the user's Claude config is broken.

**Why it happens:**
Developers copy the mental model from scripting languages where "delete then recreate" is idiomatic. Go's `os.Symlink` returns an error if the destination already exists, which forces the delete-first pattern unless you know the `rename(2)`-based alternative.

**How to avoid:**
Create the symlink at a temporary path in the same directory (ensuring same filesystem/mount point), then call `os.Rename(tmp, dest)`. On POSIX, `rename(2)` is atomic and replaces the destination even if it already exists. The `github.com/google/renameio` package wraps this pattern cleanly with its `Symlink` function. Never use `os.Symlink` directly to update an existing managed path.

**Warning signs:**
- Any code path that calls `os.Remove` immediately before `os.Symlink` on a managed path
- Switch operations that lack interrupt handling (no `signal.Notify` cleanup)
- Tests that work fine but leave stale symlinks on SIGINT during development

**Phase to address:**
Foundation / symlink engine phase — this is the core primitive; get it right before building any profile switching logic on top of it.

---

### Pitfall 2: os.Rename Fails Across Filesystem Boundaries

**What goes wrong:**
The atomic rename trick only works when both the temp file/symlink and the destination are on the same filesystem (same mount point). If `$TMPDIR` is on a different partition (e.g., `/tmp` on a tmpfs while `~/.config` is on the main ext4), `os.Rename` returns `EXDEV: invalid cross-device link`.

**Why it happens:**
`rename(2)` is a kernel operation that cannot copy data — it only moves directory entries. Crossing filesystem boundaries requires a copy-then-delete, which loses atomicity. The `google/renameio` package explicitly documents this; many developers miss it.

**How to avoid:**
Always create temp files/symlinks in the same directory as the destination, not in `os.TempDir()`. For symlink replacement: `ioutil.TempFile(filepath.Dir(dest), ".hop-tmp-*")` ensures the temp artifact lives on the same filesystem as the target.

**Warning signs:**
- Code that passes `os.TempDir()` as the location for staging files before rename
- CI failures on systems where `/tmp` is a tmpfs (very common on Linux)
- `invalid cross-device link` errors only seen in certain environments

**Phase to address:**
Foundation / symlink engine phase — establish a `safeSymlink(target, dest string)` helper that enforces same-directory temp creation.

---

### Pitfall 3: os.Stat vs os.Lstat Confusion Silently Follows Symlinks

**What goes wrong:**
Using `os.Stat` to check whether a path is a symlink always follows the link and reports the target's type. Code that checks `fileInfo.Mode()&os.ModeSymlink != 0` after `os.Stat` will always return false, even on a symlink. Validation logic that relies on this to detect managed paths will silently treat symlinks as regular files, allowing corrupt state to pass undetected.

**Why it happens:**
`os.Stat` following symlinks is correct for most file operations, so developers reach for it first. The distinction between "what is at this path" and "what does this path point to" only matters in a symlink manager — and it matters critically.

**How to avoid:**
Use `os.Lstat` everywhere you are interrogating whether a path is a managed symlink (manifest validation, adopt-on-switch detection, status display). Only use `os.Stat` when you deliberately want the target's metadata (e.g., verifying the target file exists and is readable).

**Warning signs:**
- Any `os.Stat` call in manifest validation or profile inspection code
- `fileInfo.Mode()&os.ModeSymlink` check that never fires in tests
- Adopt-on-switch logic that incorrectly adopts existing managed symlinks

**Phase to address:**
Foundation / symlink engine — establish a convention and add a linter comment (or wrapper) so the distinction is explicit throughout the codebase.

---

### Pitfall 4: JSON Compatibility Break Between Python and Go Versions

**What goes wrong:**
Python's `json` module serializes `True`/`False` as `true`/`false`, preserves all keys including those with `None` values, and does not reorder keys. Go's `encoding/json` with `omitempty` silently drops zero-value fields (empty strings, `false` booleans, `0` ints, nil slices). A Go binary writing a manifest or config file that omits zero-value fields will produce JSON that the Python version reads back incorrectly — fields missing from JSON unmarshal to zero values, which can look like intentional settings.

**Why it happens:**
`omitempty` is idiomatic Go and is copied from examples without thinking through cross-version compatibility. The Python version wrote every field; the Go version omits them.

**How to avoid:**
Do not use `omitempty` on any field in `config.json` or `.hop-manifest.json` structs. Write every field explicitly on marshal. Add a round-trip test: write a known Python-generated fixture, unmarshal in Go, remarshal in Go, and diff the output. Also: Go's `encoding/json` unmarshals case-insensitively, which is a non-issue for reading Python output, but be aware the inverse is not true.

**Warning signs:**
- `omitempty` tags on any struct field that corresponds to a field in the on-disk format
- No fixture-based round-trip tests in the test suite
- Boolean fields like `is_default` or `linked` that could legitimately be `false`

**Phase to address:**
Config/manifest parsing phase — write fixture tests from real Python-generated files before implementing any serialization.

---

### Pitfall 5: Tilde and XDG_CONFIG_HOME Not Expanded by Go stdlib

**What goes wrong:**
Go's `filepath` and `os` packages do not expand `~` in paths. If any user-facing path (stored in config, passed as a flag, or hardcoded as a default) contains `~`, it will be used literally as a filesystem path and all file operations will fail with "no such file or directory". Similarly, `os.UserConfigDir()` on Linux correctly returns `$XDG_CONFIG_HOME` if set, but `os.UserHomeDir()` under `sudo` returns root's home, not the invoking user's home.

**Why it happens:**
In Python, `os.path.expanduser("~")` is the obvious idiom. Go developers either forget the expansion step or assume stdlib handles it. The `sudo` issue catches tools that are occasionally run with elevated privileges for other reasons.

**How to avoid:**
Never store or accept paths with `~`. At the single point where the config directory is resolved (application startup), use `os.UserConfigDir()` (which respects `XDG_CONFIG_HOME`) and resolve to an absolute path immediately. Do not read `$HOME` directly — it may be unset or wrong under sudo. If sudo support is needed, use `$SUDO_USER` to determine the original user and look up their home via `user.Lookup()`.

**Warning signs:**
- Hardcoded `~/.config/claudehopper` strings anywhere in source
- No test for `XDG_CONFIG_HOME` override
- Path construction that concatenates `os.Getenv("HOME")` manually

**Phase to address:**
Foundation / config resolution phase — write a single `configDir()` function used everywhere and test it with an overridden `XDG_CONFIG_HOME`.

---

### Pitfall 6: Protected Paths List Diverging from Python Version

**What goes wrong:**
The Python version has an explicit list of paths that must never be touched by profile switching (credentials, history, projects, cache). If the Go version's list is even slightly different — a typo, a missing entry, an extra entry — users who run both versions will be confused, and worse, the Go version may inadvertently touch a protected path or refuse to touch a path it should manage.

**Why it happens:**
The list is typically hardcoded as a literal slice/constant in Go. Without a canonical source or test against the Python version's list, drift happens during porting.

**How to avoid:**
Extract the Python version's protected paths list and default linked files list into a test fixture. Write a test that asserts the Go constants match exactly. Treat any change to these lists as a breaking change requiring explicit justification.

**Warning signs:**
- Protected paths defined only in a Go `var` with no corresponding test
- No reference to the Python source when the constants are written
- Comments like "should match Python" with no automated verification

**Phase to address:**
Foundation phase — implement constants with a companion test against a fixture file checked in from the Python version.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| `os.Remove` + `os.Symlink` instead of atomic rename | Less code | Broken config on interrupt; corrupted state | Never for managed paths |
| `omitempty` on manifest struct fields | Cleaner JSON output | Python/Go cross-version compat break | Never for on-disk format structs |
| Hardcoded `~/.config/claudehopper` | Simple | Breaks `XDG_CONFIG_HOME`, root/sudo | Never |
| Single binary with alias via `ln -s` at install | Simpler goreleaser config | `go install` users get only one name | Acceptable if documented |
| Skipping round-trip JSON fixture tests | Faster initial ship | Silent compat regression | Never for shared format fields |
| Using `Run` instead of `RunE` in Cobra commands | Less boilerplate | Error handling requires `os.Exit`, untestable | Never — always use `RunE` |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Cobra shell completions | Using legacy `BashCompletionFunction` or `MarkFlagCustom` | Use `ValidArgsFunction` and `RegisterFlagCompletionFunc` — these are portable to bash, zsh, fish, powershell automatically |
| Cobra error handling | Using `Run` and calling `log.Fatal` / `os.Exit` inside | Use `RunE`, return errors; set `cmd.SilenceUsage = true` on root to prevent usage dump on runtime errors |
| goreleaser dual binary names | Single `builds` entry with two `binary` values | Two separate `builds` entries with different `id` and `binary`; two separate `archives` entries filtered by build ID to avoid archive binary count mismatch |
| goreleaser + `go install` | Version shows as `(devel)` | Inject version via `ldflags: ["-X main.version={{.Version}}"]` in goreleaser and set a `dev` fallback using `debug.ReadBuildInfo()` for `go install` users |
| GitHub releases update check | Polling GitHub API without rate-limit awareness | Use the unauthenticated `/releases/latest` endpoint, cache the result with a TTL, and handle 403/429 gracefully by silently skipping the check |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Walking `~/.config/claude` on every command | Noticeable latency on `hop status` | Only stat the specific files listed in the manifest; avoid recursive walks | When Claude config dir grows large (many plugins/commands) |
| Reading entire `usage.jsonl` to append one record | O(n) file reads | Open with `O_APPEND` flag; never read the file to append | After ~1000 switch operations (milliseconds but unnecessary) |
| Calling `os.Stat` on every protected path at switch time | Slight latency at switch | Build protected paths as a map for O(1) lookup; stat only once at startup | Non-issue at current scale; still bad practice |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Following symlinks when copying profile directories | Symlink escape: attacker-controlled symlink in a profile dir could cause `cp -r` equivalent to read/write outside the profile directory | Always use `os.Lstat` when traversing profile directories; refuse to follow symlinks encountered during profile copy/create operations |
| Writing backup files to a predictable path without checking for pre-existing symlinks | A symlink at the backup path target pointing to a sensitive file would cause the backup to overwrite it | Check that the backup destination does not exist and is not a symlink before creating it |
| Trusting `$PATH` for any subprocess calls | Malicious binary named `git` or `cp` could be invoked | This tool should not invoke subprocesses at all — pure Go fs operations only |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Printing Cobra's auto-generated usage block on every runtime error | Users see a wall of usage text when they mistype a profile name | Set `cmd.SilenceUsage = true` on root command; only show usage for flag/argument parsing errors |
| Dry-run mode that doesn't say what it would do | Users run `--dry-run` to preview but get no output | Dry-run must print every action it would take with a `[dry-run]` prefix; test that dry-run output is identical to normal output minus the actual fs mutations |
| Silent success on `hop switch` | Users don't know which profile is now active | Always print the active profile name after a successful switch, even without `--verbose` |
| Ambiguous error when a protected path is in the way | Users don't know why switch failed | Error message must name the specific protected path and explain it is intentionally unmanaged |
| `hop completions` requiring manual shell detection | Users have to know their shell | Auto-detect shell from `$SHELL` and print the right completion script by default, with explicit `--shell` flag as override |

---

## "Looks Done But Isn't" Checklist

- [ ] **Atomic symlink replacement:** Verify that `grep -r 'os.Remove' | grep -A1 'os.Symlink'` returns nothing — any paired remove+symlink is a bug.
- [ ] **Protected paths:** Verify a test asserts the Go constant list matches the extracted Python version list byte-for-byte.
- [ ] **JSON round-trip:** Verify fixture tests exist that unmarshal real Python-generated `config.json` and `.hop-manifest.json` and remarshal to identical output.
- [ ] **omitempty audit:** Verify no `omitempty` tag appears on any struct field used for on-disk serialization.
- [ ] **Dry-run coverage:** Verify every mutating operation (symlink create, file backup, manifest write) has a dry-run branch that logs but does not execute.
- [ ] **Shell completions portability:** Verify completions use `ValidArgsFunction` and `RegisterFlagCompletionFunc`, not bash scripting.
- [ ] **Dual binary distribution:** Verify `go install github.com/.../hop@latest` and `go install github.com/.../claudehopper@latest` both work (requires two `main` packages or build tag approach).
- [ ] **Version string:** Verify `hop --version` does not print `(devel)` from a goreleaser-built binary.
- [ ] **XDG_CONFIG_HOME:** Verify setting `XDG_CONFIG_HOME=/tmp/test` in tests redirects all config reads/writes correctly.
- [ ] **Interrupt safety:** Verify sending SIGINT during a `hop switch` operation leaves the filesystem in a consistent state (either old profile active or new profile active, never half-switched).

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Non-atomic symlink left dangling after interrupt | MEDIUM | Detect on next `hop status` run; re-run `hop switch <profile>` to restore; add "repair" subcommand to manifest validation |
| JSON compat break corrupts config.json | HIGH | Ship a `hop config repair` command that re-serializes from in-memory defaults; maintain a backup of config.json before any write |
| Protected paths list divergence causes data loss | HIGH | Requires manual recovery; add immediate regression test; pin list to a constant tested against Python fixture |
| Dual binary goreleaser misconfiguration | LOW | Fix `.goreleaser.yaml` and re-release; no user data affected |
| Version shows `(devel)` | LOW | Add `debug.ReadBuildInfo()` fallback in version command; re-release |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Non-atomic symlink replacement | Phase 1: Symlink engine | Test: send SIGINT mid-switch; verify no dangling symlinks |
| os.Rename cross-filesystem failure | Phase 1: Symlink engine | Test: mock different filesystem mount; verify EXDEV is handled |
| os.Stat vs os.Lstat confusion | Phase 1: Foundation | Code review checklist + `go vet` custom analyzer or grep |
| JSON compat break (omitempty) | Phase 2: Config/manifest parsing | Fixture test with Python-generated JSON files |
| Protected paths list divergence | Phase 2: Config/manifest parsing | Assertion test against extracted Python constants |
| Tilde / XDG_CONFIG_HOME not expanded | Phase 1: Foundation | Test with `XDG_CONFIG_HOME` env override |
| Cobra RunE / SilenceUsage | Phase 1: CLI scaffolding | Manual test: trigger a runtime error; verify no usage dump |
| Shell completions not portable | Phase 3: Shell completions | Test completions generated for fish; verify they reference no bash scripting |
| goreleaser dual binary misconfiguration | Phase 4: Distribution | CI: verify both binary names present in release archive |
| Version string `(devel)` | Phase 4: Distribution | CI: build with goreleaser and assert `--version` output |

---

## Sources

- [Atomic Symlinks — Tom Moertel's Blog](https://blog.moertel.com/posts/2005-08-22-how-to-change-symlinks-atomically.html)
- [github.com/google/renameio — Go Packages](https://pkg.go.dev/github.com/google/renameio)
- [The trouble with symbolic links — LWN.net](https://lwn.net/Articles/900334/)
- [os.Stat vs os.Lstat in Go — DEV Community](https://dev.to/moseeh_52/understanding-osstat-vs-oslstat-in-go-file-and-symlink-handling-3p5d)
- [Race condition when reading symlinks — gocryptfs issue #165](https://github.com/rfjakob/gocryptfs/issues/165)
- [GoReleaser: Archive has different binary count per platform](https://goreleaser.com/errors/multiple-binaries-archive/)
- [GoReleaser: Go builds customization](https://goreleaser.com/customization/builds/go/)
- [Cobra: Shell Completion documentation](https://cobra.dev/docs/how-to-guides/shell-completion/)
- [Error Handling in Cobra — JetBrains Guide](https://www.jetbrains.com/guide/go/tutorials/cli-apps-go-cobra/error_handling/)
- [os.Rename atomic behavior — golang-nuts group](https://groups.google.com/g/golang-nuts/c/ZjRWB8bMhv4)
- [Go os.Rename cross-device link error — GitHub Gist](https://gist.github.com/var23rav/23ae5d0d4d830aff886c3c970b8f6c6b)
- [Golang JSON Gotchas — okigiveup.net](https://okigiveup.net/blog/golang-json-gotchas-that-drove-me-crazy-but-i-have-learned-to-deal-with/)
- [github.com/mitchellh/go-homedir — Go Packages](https://pkg.go.dev/github.com/mitchellh/go-homedir)
- [os: UserHomeDir inconsistency issue #31070 — golang/go](https://github.com/golang/go/issues/31070)
- [Learning Go by porting a medium-sized web backend from Python — Ben Hoyt](https://benhoyt.com/writings/learning-go/)
- [Shipping completions with GoReleaser and Cobra — Carlos Becker](https://carlosbecker.com/posts/golang-completions-cobra/)
- [Using ldflags for version injection — DigitalOcean](https://www.digitalocean.com/community/tutorials/using-ldflags-to-set-version-information-for-go-applications)

---
*Pitfalls research for: Go CLI symlink-based config profile manager (claudehopper-go)*
*Researched: 2026-03-14*
