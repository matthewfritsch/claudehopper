# Stack Research

**Domain:** Go CLI tool — config profile manager with symlinks
**Researched:** 2026-03-14
**Confidence:** HIGH

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.25.x (current) | Language runtime | Current stable release as of August 2025; `os.Root` type added in 1.24 enables scoped filesystem operations which are useful for profile path validation |
| spf13/cobra | v1.10.2 | CLI framework, subcommand routing, shell completions | Industry standard for Go CLIs; used by Kubernetes, Docker, GitHub CLI; provides subcommand tree, persistent flags, automatic help generation, and built-in bash/zsh/fish/powershell completion generation — covers the shell completions requirement for free |
| google/renameio | v2 | Atomic symlink creation/replacement | The only Go library specifically designed for atomic symlink replacement; `os.Symlink` fails when target already exists, renameio wraps this correctly; used by 5,000+ dependents |
| creativeprojects/go-selfupdate | v1.5.2 | Update checking from GitHub releases | Actively maintained (Dec 2025 release); supports GitHub Releases API, version detection via git tags, cross-platform binary replacement; more complete than rhysd/go-github-selfupdate which is less actively maintained |
| goreleaser | v2.14.3 | Cross-platform binary distribution | De facto standard for Go binary releases; supports multiple build entries (build `hop` and `claudehopper` from the same main package); generates GitHub Release archives for Linux/macOS/Windows across amd64/arm64 |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| encoding/json (stdlib) | Go 1.25 stdlib | JSON config and manifest read/write | Always — stdlib is sufficient for this project's JSON needs (config.json, .hop-manifest.json, usage.jsonl); no performance requirements warrant a third-party library |
| os (stdlib) | Go 1.25 stdlib | Symlink creation, file copies, directory traversal | Always — `os.Symlink`, `os.Lstat`, `os.Readlink`, `os.MkdirAll` cover all filesystem primitives needed |
| path/filepath (stdlib) | Go 1.25 stdlib | Cross-platform path manipulation | Always — use `filepath.Join` and `filepath.Clean` for all path operations; never use string concatenation for paths |
| testing (stdlib) | Go 1.25 stdlib | Unit tests | Always — Go stdlib testing is sufficient for a CLI tool; no need for testify given the project's scope |
| stretchr/testify | v1.10.x | Assertion helpers for tests | Optional — use `assert.Equal` and `require.NoError` if test verbosity becomes a problem; skip if keeping zero test dependencies is a priority |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| golangci-lint | Static analysis, code quality | v2.x released in 2025; use `linters.default: standard` in config to get a well-curated baseline set without enabling everything; integrates with all major editors |
| goreleaser | Local build + release automation | Run `goreleaser build --snapshot --clean` for local multi-platform builds without publishing; `goreleaser release` for GitHub Releases |
| go test ./... | Test runner | Built-in; use `-race` flag for any concurrent code paths |

## Installation

```bash
# Initialize module
go mod init github.com/<user>/claudehopper-go

# Core CLI framework
go get github.com/spf13/cobra@v1.10.2

# Atomic symlink operations
go get github.com/google/renameio/v2

# Update checking
go get github.com/creativeprojects/go-selfupdate@v1.5.2

# Dev: linting (install separately, not as a go module dependency)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Dev: release tooling (install separately)
go install github.com/goreleaser/goreleaser/v2@latest
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| spf13/cobra | urfave/cli | When you want a simpler API without the Command struct model; cobra is preferred here because the PROJECT.md already mandates it and it has better shell completion support |
| spf13/cobra | spf13/viper (paired with cobra) | Viper adds config file loading on top of cobra — skip it here because claudehopper manages its own JSON config format and viper would add complexity without benefit |
| google/renameio/v2 | manual os.Rename with temp files | Acceptable on Linux-only tools; renameio handles the edge cases (cross-device rename, Windows limitations) more robustly |
| creativeprojects/go-selfupdate | rhysd/go-github-selfupdate | rhysd's version is less actively maintained; creativeprojects is a fork with continued releases through Dec 2025 and better platform support |
| encoding/json (stdlib) | tidwall/gjson or go-json | Only warranted for high-throughput JSON at scale; claudehopper reads/writes small config files — stdlib is fast enough and has zero dependencies |
| stdlib testing | stretchr/testify | Testify is fine to add if assertions become verbose; defer the decision until tests are written |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| spf13/viper | Over-engineering for this project — viper adds environment variable layering, remote config, config file watching; claudehopper owns its config format entirely and none of viper's features are needed | encoding/json + hand-rolled config loading |
| urfave/cli v2 | Less shell completion support than cobra; PROJECT.md already specifies cobra | spf13/cobra |
| rhysd/go-github-selfupdate | Last meaningful updates pre-2024; creativeprojects fork has continued maintenance | creativeprojects/go-selfupdate |
| natefinch/atomic | Does not support Windows atomic writes (same limitation as renameio but without the v2 improvements) | google/renameio/v2 |
| promptui or bubbletea | This is a non-interactive CLI — no TUI or prompts are in scope | None needed |
| CGO | Any CGO dependency breaks cross-compilation and `go install`; all chosen dependencies are pure Go | Pure Go alternatives exist for all needs |

## Stack Patterns by Variant

**For the dual binary names (hop and claudehopper):**
- Use two `builds` entries in `.goreleaser.yaml`, both pointing to the same `main` package but with `binary: hop` and `binary: claudehopper` respectively
- At the Go source level, a single `main.go` entry point calling `cobra.Execute()` works for both names — the binary name difference is purely a build artifact
- Optionally: detect `os.Args[0]` at runtime if you ever want different default behavior per binary name (not needed for this project)

**For atomic symlink switching (profile switch):**
- Use `renameio.Symlink(target, linkname)` for each symlink being created/replaced
- This wraps the create-temp-then-rename pattern that makes symlink replacement atomic on Linux/macOS
- On Windows, renameio v2 does not export this function — document that Windows support for switching is best-effort

**For shell completions:**
- Cobra generates completion scripts automatically via `cobra.Command.GenBashCompletion`, `GenZshCompletion`, etc.
- Add a `completion` subcommand (cobra provides a scaffold via `cobra.Command.InitDefaultCompletionCmd`) — no extra library needed

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| spf13/cobra v1.10.2 | Go 1.21+ | v1.10.x requires Go 1.21 minimum per go.mod |
| google/renameio/v2 | Go 1.18+ | Pure Go, no CGO, works on Linux and macOS; Windows exports are intentionally excluded |
| creativeprojects/go-selfupdate v1.5.2 | Go 1.21+ | Uses generics internally from v1.4.x onward |
| goreleaser v2.14.3 | Go 1.23+ toolchain | goreleaser itself requires Go 1.23+ to build; it can target any Go version for your project builds |
| golangci-lint v2.x | Go 1.22+ | v2 requires Go 1.22 in the module being linted |

## Sources

- https://pkg.go.dev/github.com/spf13/cobra?tab=versions — cobra v1.10.2 confirmed as latest (Dec 3, 2025) — HIGH confidence
- https://github.com/goreleaser/goreleaser/releases/latest — goreleaser v2.14.3 confirmed (Mar 9, 2026) — HIGH confidence
- https://pkg.go.dev/github.com/creativeprojects/go-selfupdate?tab=versions — go-selfupdate v1.5.2 confirmed (Dec 19, 2025) — HIGH confidence
- https://github.com/google/renameio — renameio last commit Jan 9, 2025; v2 is current; Windows atomic writes explicitly unsupported — HIGH confidence
- https://go.dev/blog/go1.25 — Go 1.25 released August 2025; current stable — HIGH confidence
- https://goreleaser.com/customization/builds/go/ — multiple builds with distinct binary names confirmed supported — HIGH confidence
- https://golangci-lint.run/ — golangci-lint v2 released 2025 as standard — HIGH confidence
- WebSearch: Go project structure cmd/internal pattern — MEDIUM confidence (standard community consensus, multiple sources agree)
- WebSearch: stdlib testing vs testify — MEDIUM confidence (community preference, not authoritative)

---
*Stack research for: claudehopper-go — Go CLI config profile manager*
*Researched: 2026-03-14*
