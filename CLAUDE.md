# claudehopper

CLI tool for switching between Claude Code configuration profiles.

## Build & Test

```bash
uv pip install -e .          # install in dev mode
python -m pytest tests/ -v   # run tests
```

## Architecture

Single-module Python CLI (`src/claudehopper/cli.py`) using only stdlib (argparse, json, shutil, pathlib). No external dependencies.

**Key paths:**
- `~/.claude/` — Claude Code config dir (symlinks managed by claudehopper)
- `~/.config/claudehopper/profiles/<name>/` — profile storage
- `~/.config/claudehopper/config.json` — tracks active profile
- `.hop-manifest.json` — per-profile manifest (managed_paths, shared_paths, description)

**Core mechanism:** Profile-specific files in `~/.claude/` are symlinked to the active profile dir. Shared files (credentials, history, projects, cache) are never touched.

## Rules

- **Never touch real `~/.claude/` in tests.** All tests must use `ClaudeHopperTestCase` which patches paths to temp dirs.
- **Validate before mutating.** Every command that moves/copies/links must verify src and dst exist before touching anything.
- **No external dependencies.** This is a stdlib-only tool. Keep it that way.
- **Symlinks, not copies** for profile switching. Copies only for `pick` and `create --from-current`.
- **Atomic symlinks.** Use `atomic_symlink()` for all symlink creation to avoid race conditions.
