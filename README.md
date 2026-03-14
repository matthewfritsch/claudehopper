# claude-swap

Switch between Claude Code configuration profiles. Run OMC one session, vanilla the next, GSD for a sprint — without blowing away your setup each time.

## The problem

Tools like [oh-my-claudecode](https://github.com/Yeachan-Heo/oh-my-claudecode), [GSD](https://github.com/gsd-build/get-shit-done), [Gastown](https://github.com/steveyegge/gastown), and [Claude Forge](https://github.com/sangrokjung/claude-forge) all want to own `~/.claude/`. They modify `settings.json`, install hooks, add agents and commands, and inject into `CLAUDE.md`. Running more than one at a time is a recipe for conflicts.

`ccswap` solves this by treating each configuration as a **profile** — a named set of files that gets symlinked into `~/.claude/` when active. Shared files (credentials, history, project memory) are never touched.

## Install

```bash
cd ~/Programming/claude-swap
pip install -e .
```

## Quick start

```bash
# Import your current setup as a profile
ccswap create omc --from-current --activate

# Create a blank profile for experimenting
ccswap create vanilla -d "stock Claude Code"

# Switch between them
ccswap switch vanilla
ccswap switch omc

# See what's active
ccswap
```

## Commands

| Command | Description |
|---|---|
| `ccswap` / `ccswap status` | Show active profile and link status |
| `ccswap list` | List all profiles |
| `ccswap create <name>` | Create a profile (`--from-current`, `--from-profile`, or blank) |
| `ccswap switch <name>` | Activate a profile (symlinks managed paths) |
| `ccswap pick <source> <path...>` | Cherry-pick files from one profile into another |
| `ccswap diff <a> <b>` | Compare what two profiles contain |
| `ccswap delete <name>` | Remove a profile |
| `ccswap unmanage` | Materialize symlinks back to real files, stop managing |
| `ccswap path <name>` | Print the profile directory path |

## How it works

Profiles live in `~/.claude-swap/profiles/<name>/`. When you switch:

1. Symlinks for the **old** profile's managed paths are removed from `~/.claude/`
2. Symlinks for the **new** profile's managed paths are created
3. The active profile is recorded in `~/.claude-swap/config.json`

**Shared files** (credentials, history, project memory, cache) stay as real files in `~/.claude/` and are never touched. **Profile-specific files** (settings.json, CLAUDE.md, commands/, agents/, hooks/, plugins/, etc.) are what get swapped.

## What each tool touches

| File/Dir | OMC | GSD | Gastown | Claude Forge |
|---|---|---|---|---|
| `settings.json` | Injects hooks, plugins, statusLine | Injects hooks + statusLine | Untouched (uses `--settings` flag) | Full replacement |
| `CLAUDE.md` | Overwrites (sentinel markers) | No | Own workspace only | No |
| `commands/` | Via plugin | ~25 files (`gsd/` ns) | 3 (in workspace) | ~36 (flat) |
| `agents/` | Via plugin | 12 | Role-based | 11 (symlinked) |
| `hooks/` | Via plugin | 4 scripts | 7 (workspace) | 15 (symlinked) |
| `plugins/` | OMC plugin | None | None | None |

## Cherry-picking between profiles

Want GSD's commands in your OMC setup? Pick them:

```bash
ccswap pick gsd commands/gsd --target omc
```

Want to try a profile's agents without fully switching:

```bash
ccswap pick forge agents --target omc
```

## Naming candidates

We considered a few names for this project — keeping them here in case we revisit:

- **`ccswap`** — short, obvious, what we went with
- **`claude-closet`** — you pick an outfit for Claude
- **`cc-rig`** — "rig" as in equipment loadout (Gastown uses this term for projects though)
- **`claude-wardrobe`** — same metaphor, more whimsical
- **`claude-face`** — different faces of Claude
- **`cprof`** — terse, "claude profiles"
- **`cc-deck`** — swap loadout decks

## License

MIT
