# claudehopper Setup Guide (for Claude)

You are helping a user set up claudehopper, a CLI tool that manages multiple Claude Code configuration profiles via symlinks in `~/.claude/`.

## Step 1: Install

```bash
uv tool install claudehopper
```

Verify with `hop --version`. Both `claudehopper` and `hop` work as CLI commands.

## Step 2: Assess `~/.claude/`

Run `hop status` to see what's in `~/.claude/`. Look at the profile-specific items detected — common ones include:

- `settings.json` / `settings.local.json` — Claude Code settings
- `CLAUDE.md` — global instructions
- `commands/` — custom slash commands
- `.mcp.json` — MCP server configuration
- `plugins/` — installed plugins (e.g., oh-my-claudecode)

## Step 3: Create profiles

### Capture the current setup first

Choose a name based on what's installed (e.g., "omc" for oh-my-claudecode, "work" for a work setup). Ask if unclear.

```bash
hop create <name> --from-current --description "<description>" --activate
```

The `--activate` flag immediately symlinks everything back so nothing breaks.

### Create a vanilla baseline

```bash
hop create vanilla --description "Stock Claude Code"
```

### Optional: additional profiles

Clone an existing profile as a starting point:

```bash
hop create work --from-profile omc --description "Work configuration"
```

## Step 4: Verify

```bash
hop status    # active profile, all paths [linked]
hop list      # all profiles
hop tree      # visual overview
```

Verify that:
1. The active profile shows all paths as `[linked]`
2. `~/.claude/` still works normally (credentials, history intact)
3. Switching works: `hop switch vanilla && hop switch <original>`

## Step 5: Show key commands

```bash
hop switch <name>                        # switch profiles
hop status                               # what's active
hop share work .mcp.json --target personal  # share a file
hop pick work CLAUDE.md --target personal   # copy a file
hop diff work personal                   # compare profiles
hop tree                                 # visual overview
hop stats                                # usage stats
```

## Notes

- Permissions (`settings.json`, `settings.local.json`) and MCP config (`.mcp.json`) are shared across all profiles by default. Use `--no-shared-defaults` on `create` to opt out.
- Don't manually edit files in `~/.config/claudehopper/profiles/` while a profile is active — edit via `~/.claude/` instead (the symlinks).
- If something goes wrong, `hop unmanage` materializes all symlinks back to real files.
- Run `hop <command> --help` for detailed usage and examples.
