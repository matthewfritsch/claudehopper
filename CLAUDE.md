# claudehopper-go

Go CLI for managing Claude Code configuration profiles and sessions. Switch between different `~/.claude/` setups via symlinks; list, inspect, resume, and prune Claude Code sessions across projects.

## Architecture

- `main.go` — entry point, version injection via ldflags
- `cmd/` — thin Cobra command wrappers (no business logic)
- `internal/config/` — paths, config.json, manifest.go
- `internal/fs/` — atomic symlinks (renameio/v2), protected paths
- `internal/profile/` — all profile business logic (create, switch, share, tree, etc.)
- `internal/session/` — Claude Code session scanning, pruning, and AI title generation
- `internal/usage/` — usage.jsonl tracking
- `internal/updater/` — GitHub release checking with 24h TTL

## Key commands

### Profiles
- `hop create <name>` — create profile (blank, `--from-current`, `--from-profile`)
- `hop switch <name>` — atomic symlink switch (`--dry-run`, `--force`)
- `hop list` / `hop status` / `hop tree` — view profiles
- `hop share` / `hop pick` / `hop unshare` — share or copy files between profiles
- `hop diff <a> <b>` — compare profiles

### Sessions (`hop sessions` / `hop sesh`)
- `hop sesh list` — sessions grouped by project, with topic, age, size, git branch
- `hop sesh info <id>` — detailed session view (supports ID prefix matching)
- `hop sesh resume <id>` — print resume command (`-x` to exec directly)
- `hop sesh titles` — generate short AI titles via `claude -p --model haiku` (cached)
- `hop sesh prune --older-than 30d` — remove old sessions (`--dry-run` to preview)
- `hop sesh stats` — aggregate overview

Session data is read from `~/.claude/projects/{encoded-path}/{sessionId}.jsonl`. Title cache stored at `{CLAUDEHOPPER_HOME}/title-cache.json`.

## Conventions

- **stdlib testing only** — no testify. Fixtures in `testdata/` dirs.
- **os.Lstat** everywhere for symlink interrogation (never os.Stat).
- **AtomicSymlink** for all symlink operations (never os.Remove + os.Symlink).
- **Business logic in internal/, CLI wiring in cmd/** — cmd functions parse flags, call internal, format output.
- **Explicit path parameters** — internal packages accept directory paths as function parameters. They never read env vars directly. Env var overrides (`CLAUDE_DIR`, `CLAUDEHOPPER_HOME`) are resolved in the CLI layer only (`cmd/helpers.go`, `internal/config/paths.go`).
- **Protected paths** — see `internal/fs/protected.go`.
- **Format compatibility** — config.json and .hop-manifest.json use stable JSON formats (2-space indent, trailing newline, sorted keys).
- **Case-insensitive profile names** — normalized to lowercase.

## Testing

```bash
go test ./...              # quick
go test -v -race ./...     # full suite
```

### Sandbox testing (safe manual testing)

claudehopper supports two environment variables that redirect all file operations away from your real `~/.claude/` and `~/.config/claudehopper/`:

| Variable | Default | Purpose |
|----------|---------|---------|
| `CLAUDE_DIR` | `~/.claude` | Where claudehopper creates/reads symlinks |
| `CLAUDEHOPPER_HOME` | `~/.config/claudehopper` | Where profiles, config, and usage data live |

**Interactive sandbox** — drops you into a shell with temp directories, builds a fresh binary, and cleans up on exit:

```bash
./scripts/sandbox.sh
```

Inside the sandbox, `hop` is on your PATH and points at isolated temp dirs. Your real config is never touched.

**Single command in sandbox:**

```bash
./scripts/sandbox.sh hop create test --from-current
./scripts/sandbox.sh hop list
```

Note: each single-command invocation gets its own fresh sandbox. Use the interactive shell if you need state to persist across commands.

**Preserve sandbox after exit** (for inspection):

```bash
KEEP_SANDBOX=1 ./scripts/sandbox.sh
```

**Manual env var usage** (without the script):

```bash
export CLAUDE_DIR=/tmp/my-test/claude
export CLAUDEHOPPER_HOME=/tmp/my-test/hopper
mkdir -p "$CLAUDE_DIR" "$CLAUDEHOPPER_HOME"
go run . create test --from-current
go run . list
```

### How internal tests work

The `internal/` packages accept explicit directory parameters (e.g., `CreateBlank(profilesDir, sharedDir, name, desc)`) rather than reading global paths. Tests use `t.TempDir()` for full isolation — no env vars needed, no filesystem side effects. The env var overrides only affect the CLI layer (`cmd/` package).

## Building

```bash
make build                 # produces bin/claudehopper + bin/hop
make install               # installs to $GOPATH/bin
```
