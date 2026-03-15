# claudehopper Setup Guide (for Claude)

You are helping a user set up claudehopper, a CLI tool that manages multiple Claude Code configuration profiles via symlinks in `~/.claude/`.

## Step 1: Install

Detect the user's platform and install accordingly.

**If Go is available (check with `go version`):**

```bash
go install github.com/matthewfritsch/claudehopper@latest
# Create the hop alias
ln -sf "$(go env GOPATH)/bin/claudehopper" "$(go env GOPATH)/bin/hop"
```

**If Go is not available, download the binary:**

```bash
# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
esac

# Get latest release
LATEST=$(curl -s https://api.github.com/repos/matthewfritsch/claudehopper/releases/latest | grep -o '"tag_name": "[^"]*' | cut -d'"' -f4)

# Download, extract, install
curl -sL "https://github.com/matthewfritsch/claudehopper/releases/download/${LATEST}/claudehopper_${LATEST#v}_${OS}_${ARCH}.tar.gz" | tar xz -C /tmp
sudo mv /tmp/claudehopper /tmp/hop /usr/local/bin/
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

Choose a name based on what's installed (e.g., "omc" for oh-my-claudecode, "work" for a work setup, "gsd" for get-shit-done). Ask if unclear.

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
hop create work --from-profile <name> --description "Work configuration"
```

## Step 4: Verify

```bash
hop status    # active profile, all paths linked
hop list      # all profiles
hop tree      # visual overview
```

Verify that:
1. The active profile shows all paths as linked
2. `~/.claude/` still works normally (credentials, history intact)
3. Switching works: `hop switch vanilla && hop switch <original>`

## Step 5: Show key commands

```bash
hop switch <name>                           # switch profiles
hop status                                  # what's active
hop share CLAUDE.md --from work --to personal  # share a file
hop pick commands/ --from work --to personal   # copy a file
hop diff work personal                      # compare profiles
hop tree                                    # visual overview
hop stats                                   # usage stats
```

## Notes

- Permissions (`settings.json`, `settings.local.json`) and MCP config (`.mcp.json`) are shared across all profiles by default.
- Don't manually edit files in `~/.config/claudehopper/profiles/` while a profile is active — edit via `~/.claude/` instead (the symlinks).
- If something goes wrong, `hop unmanage` materializes all symlinks back to real files.
- Run `hop <command> --help` for detailed usage and examples.
