# claudehopper-go

Go CLI for managing Claude Code configuration profiles via symlink switching.

## Architecture

- `main.go` — entry point, version injection via ldflags
- `cmd/` — thin Cobra command wrappers (no business logic)
- `internal/config/` — paths, config.json, manifest.go
- `internal/fs/` — atomic symlinks (renameio/v2), protected paths
- `internal/profile/` — all profile business logic (create, switch, share, tree, etc.)
- `internal/usage/` — usage.jsonl tracking
- `internal/updater/` — GitHub release checking with 24h TTL

## Conventions

- **stdlib testing only** — no testify. Fixtures in `testdata/` dirs.
- **os.Lstat** everywhere for symlink interrogation (never os.Stat).
- **AtomicSymlink** for all symlink operations (never os.Remove + os.Symlink).
- **Business logic in internal/, CLI wiring in cmd/** — cmd functions parse flags, call internal, format output.
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
