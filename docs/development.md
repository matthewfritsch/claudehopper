# Development Guide

## Sandbox Testing

claudehopper manages real files in `~/.claude/` and `~/.config/claudehopper/`. When developing or testing, you should **never** operate on your real configuration. Two environment variables redirect all file operations to safe locations:

| Variable | Default | Purpose |
|----------|---------|---------|
| `CLAUDE_DIR` | `~/.claude` | Where claudehopper creates/reads symlinks |
| `CLAUDEHOPPER_HOME` | `~/.config/claudehopper` | Where profiles, config, and usage data live |

### Using the sandbox script

The easiest way to test is `scripts/sandbox.sh`. It creates temp directories, builds a fresh binary, seeds a minimal fake `~/.claude/`, and drops you into an isolated shell:

```bash
./scripts/sandbox.sh
```

Inside the sandbox, `hop` is on your PATH and everything points at `/tmp/hop-sandbox-XXXX/`. Your real config is never touched. Type `exit` to leave — the sandbox is cleaned up automatically.

**Run a single command:**

```bash
./scripts/sandbox.sh hop create test --from-current
./scripts/sandbox.sh hop list
```

Each single-command invocation gets its own fresh sandbox. Use the interactive shell if you need state to persist across commands.

**Preserve the sandbox after exit** (for inspection):

```bash
KEEP_SANDBOX=1 ./scripts/sandbox.sh
```

### Manual env var usage

If you prefer not to use the script:

```bash
export CLAUDE_DIR=/tmp/my-test/claude
export CLAUDEHOPPER_HOME=/tmp/my-test/hopper
mkdir -p "$CLAUDE_DIR" "$CLAUDEHOPPER_HOME"

# Seed some files
echo '{}' > "$CLAUDE_DIR/settings.json"
echo '# Test' > "$CLAUDE_DIR/CLAUDE.md"

# Run directly
go run . create test --from-current
go run . list
go run . switch test
go run . status
```

## Unit Tests

```bash
go test ./...              # quick
go test -v -race ./...     # full suite with race detector
```

### How internal tests work

The `internal/` packages accept explicit directory parameters (e.g., `CreateBlank(profilesDir, sharedDir, name, desc)`) rather than reading global paths. Tests use `t.TempDir()` for full isolation — no env vars needed, no filesystem side effects.

The env var overrides (`CLAUDE_DIR`, `CLAUDEHOPPER_HOME`) only affect the CLI layer (`cmd/` package and `internal/config/paths.go`).

### Test fixtures

Python-format JSON files are stored in `testdata/` directories alongside the test files that use them. These ensure round-trip compatibility with the Python version of claudehopper.

## Building

```bash
make build                 # produces bin/claudehopper + bin/hop
make install               # installs to $GOPATH/bin with hop alias
```

Version injection happens via ldflags. Local builds show `dev`; goreleaser sets the real version from the git tag.

## For AI Agents

When developing or testing claudehopper, **always use the sandbox** or set env vars. Never run `hop create`, `hop switch`, or any mutating command against the user's real `~/.claude/` during development.

```bash
# Safe: sandbox
./scripts/sandbox.sh hop create test --from-current

# Safe: env vars
CLAUDE_DIR=/tmp/test CLAUDEHOPPER_HOME=/tmp/test-hopper hop list

# UNSAFE: touches real config
hop create test --from-current  # DO NOT do this during development
```
