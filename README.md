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

### Option A: Let Claude set it up for you

Paste this into Claude Code:

```
Set up claudehopper for me. Read the setup guide at docs/claude-setup-guide.md
in the claudehopper repo, then follow it to create profiles from my current
~/.claude/ configuration.
```

Or, if you haven't cloned the repo, give Claude this prompt:

```
Install claudehopper (uv tool install claudehopper), then:
1. Run `claudehopper status` to see my current ~/.claude/ contents
2. Create a profile from my current setup with a descriptive name and --activate
3. Create a "vanilla" profile for clean Claude Code without plugins
4. Run `claudehopper tree` to show me what was set up
```

Claude will analyze your `~/.claude/` directory, name profiles based on what's installed (e.g., "omc" if you have oh-my-claudecode), and set everything up automatically.

### Option B: Manual setup

```bash
# See what's currently active
claudehopper status

# Create a profile from your current config
claudehopper create work --from-current --description "Work projects setup" --activate

# Create a fresh profile
claudehopper create vanilla --description "Clean Claude Code"

# Switch to a profile
hop switch work

# List all profiles
hop list
```

## Examples

### Separate work and personal configs

```bash
# Capture your current setup as "work"
claudehopper create work --from-current --description "Work - strict mode" --activate

# Clone it for personal use, then customize
claudehopper create personal --from-profile work --description "Personal experiments"
hop switch personal
# Edit ~/.claude/CLAUDE.md, ~/.claude/settings.json to taste
hop switch work   # back to work mode
```

### Share MCP servers across profiles

```bash
# Your "work" profile has MCP servers you want everywhere
hop share work .mcp.json --target personal

# Now both profiles use the same .mcp.json — edit once, applies to both
```

### Keep a clean baseline

```bash
# Create a vanilla profile with no plugins or customizations
claudehopper create vanilla --description "Stock Claude Code"

# When you want clean Claude behavior:
hop switch vanilla

# Back to your full setup:
hop switch work
```

### See what you use most

```bash
# Usage stats across all profiles
hop stats

# Just one profile
hop stats --profile work

# Machine-readable
hop stats --json
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

### `tree`

Show all profiles as a visual tree with lineage and shared file relationships.

```bash
claudehopper tree
claudehopper tree --json
```

### `stats`

Show profile usage statistics — switch counts, last used times, and more.

```bash
claudehopper stats
claudehopper stats --profile work
claudehopper stats --since 2025-01-01
claudehopper stats --json
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
