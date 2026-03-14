# claudehopper

Switch between Claude Code configuration profiles instantly. Keep separate setups for work, personal projects, or different AI behaviors — and hop between them with a single command.

## The Problem

Claude Code stores all its configuration in `~/.claude/`: your `CLAUDE.md` instructions, settings, MCP servers, and more. If you want different configurations for different contexts — a strict work setup vs. a more experimental personal one — you're stuck manually swapping files.

claudehopper manages this by symlinking profile-specific files into `~/.claude/` when you switch profiles. Shared files (credentials, history, projects, cache) are never touched.

## Install

```bash
uv tool install claudehopper
```

Both `claudehopper` and `hop` invoke the same CLI.

## Quick Start

```bash
# See what's currently active
claudehopper status

# Create a profile from your current config
claudehopper create work --from-current --description "Work projects setup"

# Create a fresh profile
claudehopper create personal --description "Personal experiments"

# Switch to a profile
hop switch work

# List all profiles
hop list
```

## Command Reference

### `status`

Show the currently active profile and which files are managed.

```bash
claudehopper status
```

### `list` / `ls`

List all available profiles.

```bash
claudehopper list
claudehopper ls
```

### `create <name>`

Create a new profile.

```bash
claudehopper create <name> [options]

Options:
  --from-current          Copy managed files from ~/.claude/ into the new profile
  --from-profile <name>   Copy managed files from an existing profile
  --description <text>    Short description for the profile
```

### `switch <name>` / `sw`

Switch to a profile. Replaces managed symlinks in `~/.claude/` with links to the target profile.

```bash
claudehopper switch <name>
hop sw <name>

Options:
  --force     Replace conflicting files without prompting
  --dry-run   Show what would change without touching anything
```

### `pick <source> <paths...>`

Copy specific files from one profile into another (or the active profile).

```bash
claudehopper pick work CLAUDE.md settings.json
```

### `share <source> <paths...>`

Share files between profiles via symlinks. Both profiles point to the same file.

```bash
claudehopper share work .mcp.json
```

### `unshare [paths...]`

Convert shared (symlinked) files back to independent copies in each profile.

```bash
claudehopper unshare .mcp.json
claudehopper unshare          # unshare all shared files
```

### `diff <a> <b>`

Compare two profiles side-by-side.

```bash
claudehopper diff work personal
```

### `delete <name>` / `rm`

Delete a profile. Prompts for confirmation unless `--force` is passed.

```bash
claudehopper delete personal
hop rm personal --force
```

### `unmanage`

Stop using claudehopper. Materializes all symlinks in `~/.claude/` back to real files and removes claudehopper's configuration.

```bash
claudehopper unmanage
```

### `migrate`

Migrate an existing `~/.claude-swap/` setup (legacy tool) to claudehopper profiles.

```bash
claudehopper migrate
```

### `path <name>`

Print the filesystem path to a profile's directory. Useful for scripting.

```bash
claudehopper path work
# /home/you/.config/claudehopper/profiles/work
```

## How It Works

**Profile storage:** Each profile lives in `~/.config/claudehopper/profiles/<name>/`. Files in that directory mirror what you want in `~/.claude/`.

**Switching:** When you run `hop switch <name>`, claudehopper replaces the managed files in `~/.claude/` with symlinks pointing to the target profile's directory. Claude Code sees normal files and never knows they're symlinks.

**Shared files:** The following paths in `~/.claude/` are never touched by claudehopper, regardless of profile:

- `credentials.json` — API keys and auth tokens
- `.credentials.json`
- `history/` — command history
- `projects/` — project-specific memory
- `todos/` — task storage
- `.cache/` — cached data

**Manifest:** Each profile contains a `.hop-manifest.json` that records which paths are managed, which are shared with other profiles, and the profile description.

**Active profile:** `~/.config/claudehopper/config.json` tracks which profile is currently active.

## Safety

- Credentials and history are never modified, copied, or symlinked.
- `--dry-run` is available on `switch` to preview changes before applying them.
- `unmanage` always materializes symlinks to real files before removing configuration, so you never lose data.
- Conflicting unmanaged files in `~/.claude/` are backed up before being replaced on switch.

## Migration from claude-swap

If you previously used `~/.claude-swap/`, run:

```bash
claudehopper migrate
```

This imports your existing swap configurations as named profiles.
