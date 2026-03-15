# Phase 4: Polish & Distribution - Research

**Researched:** 2026-03-14
**Domain:** Go self-update (creativeprojects/go-selfupdate), goreleaser v2 distribution, Cobra shell completions, Homebrew tap setup
**Confidence:** HIGH

## Summary

Phase 4 makes claudehopper releasable: it wires up update checking with a 24h file-based TTL cache, adds a `hop update` command, validates goreleaser config and adds a GitHub Actions release workflow, verifies Cobra shell completions work for all four shells, and creates the `homebrew-claudehopper` tap repository fed by goreleaser.

The codebase is already well-prepared. `.goreleaser.yaml` has the dual-build config, `cmd/root.go` wires ldflags version info, and Cobra auto-generates a `completion` subcommand. The three main deliverables are: (1) `internal/updater/` package with TTL-cached GitHub release check, (2) goreleaser config finishing touches + CI workflow, and (3) Homebrew tap repository creation.

**Primary recommendation:** Use `creativeprojects/go-selfupdate` with `ChecksumValidator{UniqueFilename: "checksums.txt"}` for integrity, store the last-check timestamp in the claudehopper config directory as a plain file to implement the 24h TTL, run the check in a goroutine so `hop status` is never slowed down, and display any available update after the status output.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Update notice appears only after `hop status` command — least intrusive placement
- `hop update` auto-installs the new version via `go install` (for source installs) or downloads binary (for binary installs)
- 24h cached TTL for GitHub release checks
- Non-blocking — never slow down normal commands
- Release targets: Linux amd64/arm64, macOS amd64/arm64 only (no Windows)
- GitHub Releases for binary distribution
- Homebrew tap (`homebrew-claudehopper`) for macOS/Linux package management
- No AUR package

### Claude's Discretion
- `go-selfupdate` library configuration details
- Shell completions verification approach (manual testing vs automated)
- Homebrew formula structure and tap repository setup
- Goreleaser archive format preferences (tar.gz vs zip per platform)
- CI/CD workflow if needed (GitHub Actions for goreleaser)

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| OPS-02 | Tool checks for updates from GitHub releases with 24h cached TTL | go-selfupdate DetectLatest + file-based TTL stamp in config dir |
| DIST-01 | Shell tab completions work for bash, zsh, fish, and powershell via Cobra | Cobra auto-generates `completion` subcommand; needs verification |
| DIST-03 | Tool distributable via `go install` and prebuilt binaries (goreleaser) | Existing .goreleaser.yaml needs `homebrew_casks` block + CI workflow |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| creativeprojects/go-selfupdate | v1.5.2 | GitHub release detection and binary self-update | Stable v1+ API, goreleaser checksums.txt integration, Context support, rollback on failure |
| goreleaser | v2.x | Cross-platform binary builds and GitHub release publishing | Already in project; dual-build config exists |
| spf13/cobra | v1.10.2 (existing) | Shell completions | Already wired; auto-generates `completion` subcommand |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| os (stdlib) | — | TTL stamp file read/write | Implement 24h cache without extra deps |
| context (stdlib) | — | go-selfupdate API requires Context | All DetectLatest calls |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| File-based TTL stamp | In-memory TTL cache library | File persists across restarts; no extra dep; correct for CLI tools |
| creativeprojects/go-selfupdate | rhysd/go-github-selfupdate | rhysd is older, less maintained; creativeprojects has active development, checksums.txt support |
| homebrew_casks (goreleaser) | Manually maintained Formula | Manual formula requires independent update process; goreleaser automates it |

**Installation:**
```bash
go get github.com/creativeprojects/go-selfupdate@v1.5.2
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── updater/           # NEW: update check and self-update logic
│   ├── updater.go     # CheckForUpdate, PerformUpdate, TTL cache
│   └── updater_test.go
cmd/
├── status.go          # MODIFY: call updater.CheckForUpdate after status display
├── update.go          # NEW: `hop update` command
.github/
└── workflows/
    └── release.yml    # NEW: goreleaser GitHub Actions workflow
```

### Pattern 1: Non-Blocking Update Check with File-Based TTL
**What:** Check for updates in a goroutine, writing/reading a stamp file in the claudehopper config dir to enforce the 24h TTL. Post result back on a channel.
**When to use:** In `runStatus` after printing profile status.
**Example:**
```go
// Source: go-selfupdate pkg.go.dev docs + stdlib os
func CheckForUpdate(ctx context.Context, configDir, currentVersion string) (*selfupdate.Release, error) {
    stampPath := filepath.Join(configDir, "update-check.stamp")
    if info, err := os.Stat(stampPath); err == nil {
        if time.Since(info.ModTime()) < 24*time.Hour {
            return nil, nil // within TTL, skip check
        }
    }
    latest, found, err := selfupdate.DetectLatest(ctx, selfupdate.ParseSlug("matthewfritsch/claudehopper"))
    if err != nil {
        return nil, err
    }
    // Write stamp regardless of result — avoids hammering API on error loops
    _ = os.WriteFile(stampPath, []byte(time.Now().Format(time.RFC3339)), 0644)
    if !found || latest.LessOrEqual(currentVersion) {
        return nil, nil
    }
    return latest, nil
}
```

### Pattern 2: Non-Blocking Goroutine in cmd/status.go
**What:** Wrap CheckForUpdate in a goroutine with channel, then print notice if available.
**When to use:** At the end of runStatus, after profile status is already printed.
**Example:**
```go
// After fmt.Print(profile.FormatProfileStatus(info))
type updateResult struct {
    release *selfupdate.Release
    err     error
}
ch := make(chan updateResult, 1)
go func() {
    r, e := updater.CheckForUpdate(context.Background(), configDir, version)
    ch <- updateResult{r, e}
}()
select {
case res := <-ch:
    if res.err == nil && res.release != nil {
        fmt.Printf("\nUpdate available: %s → run `hop update` to install\n", res.release.Version())
    }
case <-time.After(3 * time.Second):
    // timeout: silently skip
}
```

### Pattern 3: go-selfupdate Updater with ChecksumValidator
**What:** Use `selfupdate.NewUpdater` with `ChecksumValidator` pointing to goreleaser's `checksums.txt`.
**When to use:** In `hop update` command — verifies binary integrity before replacing executable.
**Example:**
```go
// Source: go-selfupdate pkg.go.dev docs
updater, err := selfupdate.NewUpdater(selfupdate.Config{
    Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
})
if err != nil {
    return fmt.Errorf("create updater: %w", err)
}
latest, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug("matthewfritsch/claudehopper"))
// ...
exe, err := selfupdate.ExecutablePath()
if err != nil {
    return fmt.Errorf("locate executable: %w", err)
}
if err := updater.UpdateTo(ctx, latest, exe); err != nil {
    return fmt.Errorf("update binary: %w", err)
}
```

### Pattern 4: Install Detection for `hop update`
**What:** Detect whether binary came from `go install` (lives in GOPATH/bin) vs downloaded binary, then choose update strategy.
**When to use:** In the `hop update` command.
**Example:**
```go
exe, _ := selfupdate.ExecutablePath()
gobin := filepath.Join(build.Default.GOPATH, "bin")
if strings.HasPrefix(exe, gobin) {
    // source install: use go install
    return runGoInstall(latest.Version())
}
// binary install: use go-selfupdate UpdateTo
return updater.UpdateTo(ctx, latest, exe)
```

### Pattern 5: Goreleaser homebrew_casks Block
**What:** Adds a Homebrew Cask entry to a separate `homebrew-claudehopper` repo after each release.
**When to use:** In `.goreleaser.yaml` for macOS/Linux Homebrew distribution.
**Example:**
```yaml
# goreleaser v2.10+ required; homebrew_casks replaces deprecated brews
homebrew_casks:
  - repository:
      owner: matthewfritsch
      name: homebrew-claudehopper
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    homepage: "https://github.com/matthewfritsch/claudehopper"
    description: "Instant, safe profile switching for Claude Code configs"
    binaries:
      - hop
      - claudehopper
```

### Anti-Patterns to Avoid
- **Blocking update check:** Never call `DetectLatest` synchronously in a command path — wrap in goroutine with timeout.
- **Unconditional API call:** Always read the TTL stamp before calling GitHub API — avoids rate limiting and slowdowns.
- **Separate checksums per binary:** goreleaser produces a single `checksums.txt` covering all release assets; use `ChecksumValidator{UniqueFilename: "checksums.txt"}` not per-asset SHA files.
- **Using deprecated `brews` section in goreleaser:** Deprecated in v2.10, scheduled for removal in v3. Use `homebrew_casks`.
- **Skipping `fetch-depth: 0` in GitHub Actions:** GoReleaser needs full git history to generate changelog and determine version from tags; omitting this causes release failures.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| GitHub release detection | Custom HTTP + JSON parser for GitHub API | `selfupdate.DetectLatest` | Rate limits, pagination, pre-release filtering, semantic version comparison |
| Binary replacement | Direct file copy over running binary | `updater.UpdateTo` | Needs atomic replacement with rollback; os.Rename semantics vary by platform |
| Archive extraction | Custom tar.gz/zip extractor | Built into go-selfupdate | Multiple archive formats, platform-specific executable detection |
| Homebrew formula generation | Manually maintained .rb file | goreleaser `homebrew_casks` | Formula URL/hash must update on every release; automation prevents drift |
| Shell completion scripts | Hand-written completion functions | Cobra built-in | Cobra generates correct completion for all four shells; custom scripts diverge from actual CLI |

**Key insight:** The update-check TTL IS hand-rolled (file stamp), but that's correct — it's a trivial file read/write that avoids adding a cache library dependency.

## Common Pitfalls

### Pitfall 1: Update Check Slows Down `hop status`
**What goes wrong:** `DetectLatest` does an HTTP request; if GitHub is slow, status takes several seconds.
**Why it happens:** Direct synchronous call in command handler.
**How to avoid:** Always run in a goroutine with a 3-second timeout select. If timeout fires, silently skip the notice.
**Warning signs:** `hop status` taking >1s.

### Pitfall 2: go-selfupdate Binary Naming Mismatch
**What goes wrong:** `UpdateTo` cannot find the matching asset in the GitHub release because the archive name doesn't match the expected pattern.
**Why it happens:** go-selfupdate expects `{cmd}_{goos}_{goarch}` format. The current goreleaser archive name template is `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}` — the version is between project name and OS, which matches goreleaser conventions and go-selfupdate's pattern matching.
**How to avoid:** Test with a snapshot release first using `goreleaser release --snapshot`. Verify asset names match `claudehopper_*_linux_amd64.tar.gz` pattern.
**Warning signs:** `ErrAssetNotFound` error from go-selfupdate.

### Pitfall 3: `hop update` Replaces `claudehopper` but Not `hop` (or vice versa)
**What goes wrong:** Binary install replaces whichever binary the user ran, but the Makefile/goreleaser ships them as separate executables in the same archive. User expects both to update.
**Why it happens:** go-selfupdate's `UpdateTo` replaces the single `exe` path from `ExecutablePath()`.
**How to avoid:** After updating the primary binary, check if the sibling binary exists in the same directory and update it too, or document that both binaries update independently.
**Warning signs:** Version mismatch between `hop --version` and `claudehopper --version` after update.

### Pitfall 4: HOMEBREW_TAP_GITHUB_TOKEN Missing in CI
**What goes wrong:** Goreleaser fails to push to the tap repository during release because `GITHUB_TOKEN` only has write access to the source repo, not the separate `homebrew-claudehopper` repo.
**Why it happens:** GitHub Actions `GITHUB_TOKEN` is scoped per-repository.
**How to avoid:** Create a Personal Access Token with `repo` scope, store it as `HOMEBREW_TAP_GITHUB_TOKEN` secret in the source repo, reference it in the goreleaser `homebrew_casks.repository.token` field.
**Warning signs:** Goreleaser succeeds for binary release but fails at the homebrew step.

### Pitfall 5: Cobra Completion Command Auto-Generated but Not Advertised
**What goes wrong:** Cobra adds `completion` subcommand automatically, but users don't know it exists because it's not in the README.
**Why it happens:** The completion command appears in `--help` output but isn't tested.
**How to avoid:** Verify the `completion` subcommand outputs valid scripts. Test by actually sourcing the output in a shell. Document install instructions per shell in README (out of scope here, but note the verification step in tasks).
**Warning signs:** `hop completion bash | bash -s` errors out.

### Pitfall 6: goreleaser `check` Warnings vs Errors
**What goes wrong:** `goreleaser check` warns about deprecated fields but exits 0, so CI doesn't catch config problems until release day.
**Why it happens:** Deprecation warnings are soft; only invalid config is hard-error.
**How to avoid:** Run `goreleaser check` locally before merging. Treat any "WARN" about deprecated fields as a blocker.
**Warning signs:** `goreleaser check` output shows "WARN" lines.

## Code Examples

Verified patterns from official sources:

### go-selfupdate: Detect Latest Release
```go
// Source: pkg.go.dev/github.com/creativeprojects/go-selfupdate
latest, found, err := selfupdate.DetectLatest(
    context.Background(),
    selfupdate.ParseSlug("matthewfritsch/claudehopper"),
)
if err != nil {
    return fmt.Errorf("error detecting version: %w", err)
}
if !found || latest.LessOrEqual(currentVersion) {
    return nil // already up to date
}
fmt.Printf("Update available: %s\n", latest.Version())
```

### go-selfupdate: Perform Binary Update with Checksum Validation
```go
// Source: pkg.go.dev/github.com/creativeprojects/go-selfupdate
updater, err := selfupdate.NewUpdater(selfupdate.Config{
    Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
})
if err != nil {
    return err
}
exe, err := selfupdate.ExecutablePath()
if err != nil {
    return fmt.Errorf("could not locate executable path: %w", err)
}
if err := updater.UpdateTo(ctx, latest, exe); err != nil {
    return fmt.Errorf("error updating binary: %w", err)
}
fmt.Printf("Updated to %s\n", latest.Version())
```

### goreleaser: Complete .goreleaser.yaml with homebrew_casks
```yaml
# version: 2 already set
homebrew_casks:
  - repository:
      owner: matthewfritsch
      name: homebrew-claudehopper
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    homepage: "https://github.com/matthewfritsch/claudehopper"
    description: "Instant, safe profile switching for Claude Code configs"
    license: MIT
    binaries:
      - hop
      - claudehopper
```

### GitHub Actions: Release Workflow
```yaml
# Source: goreleaser.com/ci/actions/
name: goreleaser
on:
  push:
    tags:
      - "v*"
permissions:
  contents: write
jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: stable
      - uses: goreleaser/goreleaser-action@v7
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

### Cobra: Shell Completions (auto-generated)
```bash
# Cobra automatically adds `completion` subcommand — no code required.
# Verification commands:
hop completion bash   # outputs bash completion script
hop completion zsh    # outputs zsh completion script
hop completion fish   # outputs fish completion script
hop completion powershell  # outputs PowerShell completion script

# Install for current user (bash):
hop completion bash > ~/.bash_completion.d/hop

# Install for current user (zsh):
hop completion zsh > "${fpath[1]}/_hop"

# Install for fish:
hop completion fish > ~/.config/fish/completions/hop.fish
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `brews` section in goreleaser | `homebrew_casks` section | goreleaser v2.10 (2025) | Must migrate; `brews` deprecated, removed in v3 |
| `homebrew_formulas` section | `homebrew_casks` section | goreleaser v2.10 (2025) | Same migration |
| Manual Homebrew formula | goreleaser-managed cask | goreleaser v2.10+ | Fully automated tap updates |

**Deprecated/outdated:**
- `brews:` goreleaser config key: deprecated as of v2.10; replaced by `homebrew_casks`. Still works but emits warnings.
- `homebrew_formulas:` goreleaser config key: same deprecation.

## Open Questions

1. **`hop update` for `go install` users — detecting vs installing**
   - What we know: The current Makefile `install` target uses `go install` + `ln -sf` for the `hop` symlink. On a source install, the installed path is `$GOPATH/bin/claudehopper` with a symlink `$GOPATH/bin/hop`.
   - What's unclear: Should `hop update` re-run `go install github.com/matthewfritsch/claudehopper@latest` (requires internet + Go toolchain) or always use binary download? The CONTEXT.md says "go install for source installs."
   - Recommendation: Detect by checking if `$GOPATH/bin/claudehopper` matches `ExecutablePath()`. If yes, run `go install`; otherwise use go-selfupdate binary download. Implement `go install` branch as `exec.Command("go", "install", "github.com/matthewfritsch/claudehopper@latest")`.

2. **Homebrew tap: brews vs homebrew_casks on Linux**
   - What we know: `homebrew_casks` is the correct new approach. Homebrew runs on Linux (Linuxbrew), but "Casks" are traditionally macOS-only in Homebrew's mental model.
   - What's unclear: Whether `homebrew_casks` in goreleaser generates a formula that installs on Linux via `brew install` or is macOS-only.
   - Recommendation: Since no AUR is planned and GitHub Releases + `go install` cover Linux, treat Homebrew tap as primarily macOS. Linux users use `go install` or download from GitHub Releases directly.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + `go test -race` |
| Config file | none (stdlib testing, no config file) |
| Quick run command | `go test ./internal/updater/... -v -race` |
| Full suite command | `go test -v -race ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| OPS-02 | TTL check skips API when stamp < 24h old | unit | `go test ./internal/updater/... -run TestCheckForUpdate_SkipsWithinTTL -v` | ❌ Wave 0 |
| OPS-02 | TTL check hits API when stamp > 24h old | unit | `go test ./internal/updater/... -run TestCheckForUpdate_CallsAPIAfterTTL -v` | ❌ Wave 0 |
| OPS-02 | Returns nil when already at latest version | unit | `go test ./internal/updater/... -run TestCheckForUpdate_AlreadyLatest -v` | ❌ Wave 0 |
| DIST-01 | `hop completion bash` outputs valid bash script | smoke | `hop completion bash \| bash -n` | ❌ Wave 0 |
| DIST-01 | `hop completion zsh` outputs valid zsh script | manual | manual shell test | manual-only |
| DIST-01 | `hop completion fish` outputs valid fish script | manual | manual shell test | manual-only |
| DIST-03 | goreleaser config is valid | smoke | `goreleaser check` | ❌ Wave 0 (config addition) |
| DIST-03 | goreleaser snapshot build succeeds | integration | `goreleaser release --snapshot --clean` | manual-only |

### Sampling Rate
- **Per task commit:** `go test ./internal/updater/... -v -race`
- **Per wave merge:** `go test -v -race ./...`
- **Phase gate:** Full suite green + `goreleaser check` exits 0 before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/updater/updater.go` — new package; must exist before tests
- [ ] `internal/updater/updater_test.go` — covers OPS-02 TTL and version comparison
- [ ] `cmd/update.go` — new command file
- [ ] `.goreleaser.yaml` additions — `homebrew_casks` block (needed for `goreleaser check`)
- [ ] `.github/workflows/release.yml` — new file (smoke tested with `goreleaser check`)

## Sources

### Primary (HIGH confidence)
- `pkg.go.dev/github.com/creativeprojects/go-selfupdate` — DetectLatest, UpdateTo, ChecksumValidator, ExecutablePath APIs
- `goreleaser.com/ci/actions/` — GitHub Actions workflow YAML
- `goreleaser.com/customization/homebrew_casks/` — homebrew_casks config structure

### Secondary (MEDIUM confidence)
- `goreleaser.com/blog/goreleaser-v2.10/` — confirms homebrew_casks introduced, brews deprecated
- `goreleaser.com/deprecations/` — confirms brews/homebrew_formulas deprecation timeline
- `cobra.dev/docs/how-to-guides/shell-completion/` — completion subcommand auto-generation

### Tertiary (LOW confidence)
- WebSearch result: homebrew_casks Linux support scope — unverified whether Linux install works via `brew install` with casks

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — go-selfupdate API verified from pkg.go.dev; goreleaser from official docs
- Architecture: HIGH — patterns derived directly from library APIs and existing project code
- Pitfalls: MEDIUM — most from direct API inspection; binary naming pitfall is deduced from go-selfupdate's documented naming rules
- Homebrew tap: MEDIUM — homebrew_casks config verified from goreleaser docs; Linux cask support LOW

**Research date:** 2026-03-14
**Valid until:** 2026-04-14 (goreleaser moves fast; re-verify homebrew_casks config if > 30 days)
