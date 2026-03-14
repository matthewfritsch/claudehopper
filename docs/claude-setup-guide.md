# claudehopper Setup Guide (for Claude)

You are helping a user set up claudehopper, a CLI tool that manages multiple Claude Code configuration profiles via symlinks in `~/.claude/`.

## Overview

claudehopper lets users maintain separate Claude Code configurations (settings, CLAUDE.md, commands, MCP servers, plugins, etc.) and switch between them instantly. Shared files like credentials, history, and project memory are never touched.

## Step 1: Install

```bash
uv tool install claudehopper
```

Verify installation:

```bash
claudehopper --version
```

Both `claudehopper` and `hop` are available as CLI commands.

## Step 2: Assess the user's current `~/.claude/`

Run `claudehopper status` to see what's currently in `~/.claude/`. This shows:
- Whether any profile is active
- Which files are profile-specific vs shared

Look at the profile-specific items detected. Common ones include:
- `settings.json` — Claude Code settings (hooks, permissions, model preferences)
- `settings.local.json` — local overrides
- `CLAUDE.md` — global instructions for Claude
- `commands/` — custom slash commands
- `plugins/` — installed plugins (e.g., oh-my-claudecode)
- `.omc/` — oh-my-claudecode state directory
- `.omc-config.json` — oh-my-claudecode configuration
- `hud/` — HUD display configuration
- `sessions/` — session data
- `tasks/` — task storage
- `teams/` — team configurations
- `telemetry/` — telemetry data

## Step 3: Create profiles

### Current profile (always do this first)

Capture the user's current `~/.claude/` as a named profile. Choose a descriptive name based on what they're using:

- If they have oh-my-claudecode installed: name it `omc`
- If it's a work setup: name it `work`
- If it's generic: name it `default` or `main`
- Ask the user what they'd like to call it if unclear

```bash
claudehopper create <name> --from-current --description "<description>" --activate
```

The `--activate` flag immediately symlinks everything back, so nothing breaks.

### Base/vanilla profile

Create a minimal "vanilla" profile for clean Claude Code usage without plugins or customizations:

```bash
claudehopper create vanilla --description "Clean Claude Code - no plugins or customizations"
```

This creates a profile with just an empty `settings.json`. The user can switch to this when they want stock Claude Code behavior.

### Additional profiles (optional)

Based on the user's needs, suggest profiles like:

- **work** — stricter settings, work-specific CLAUDE.md instructions
- **personal** — experimental settings, personal projects
- **minimal** — stripped-down config for fast startup
- **shared-project** — specific MCP servers and commands for a team project

To clone an existing profile as a starting point:

```bash
claudehopper create work --from-profile omc --description "Work configuration"
```

Then the user can customize each profile independently.

## Step 4: Verify

```bash
# Check the active profile
claudehopper status

# List all profiles
claudehopper list

# See the profile tree
claudehopper tree
```

Verify that:
1. The active profile shows all paths as `[linked]`
2. `~/.claude/` still works normally (credentials, history intact)
3. The user can switch profiles: `claudehopper switch vanilla && claudehopper switch <original>`

## Step 5: Teach common workflows

Show the user these key commands:

```bash
# Switch profiles
hop switch work
hop switch personal

# See what's active
hop status

# Share a file between profiles (e.g., same MCP config everywhere)
hop share work .mcp.json --target personal

# Copy a file from one profile to another (independent copy)
hop pick work CLAUDE.md --target personal

# Compare two profiles
hop diff work personal

# See usage stats
hop stats

# Visualize profile relationships
hop tree
```

## Important Notes

- **Never** manually edit files in `~/.config/claudehopper/profiles/` while a profile is active. Use `claudehopper` commands or edit via `~/.claude/` (the symlinks).
- Switching profiles is instant — it just swaps symlinks.
- If something goes wrong, `claudehopper unmanage` materializes all symlinks back to real files, restoring a normal `~/.claude/` directory.
- Backup files (`.hop-backup`) are created in `~/.claude/` when real files are replaced during the first switch. These are safe to delete once you've verified everything works.

## Troubleshooting

**"unmanaged file exists in ~/.claude/"**
Run `claudehopper create <name> --from-current` first to capture existing files, or use `--force` on switch to back them up automatically.

**Broken symlinks after switching**
Run `claudehopper switch <profile> --force` to re-link everything.

**Want to stop using claudehopper**
Run `claudehopper unmanage` to replace all symlinks with real files and go back to a normal `~/.claude/` directory.
