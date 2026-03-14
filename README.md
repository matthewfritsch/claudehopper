# claudehopper

Switch between Claude Code configuration profiles. Run Oh-my-claudecode one session, vanilla the next, GSD for a sprint — without blowing away your setup each time. Share what you want between profiles, but keep other files separate.

## The Problem

Claude Code stores all its configuration in `~/.claude/`: your `CLAUDE.md` instructions, settings, MCP servers, and more. If you want different configurations for different contexts — a strict work setup vs. a more experimental personal one — you're stuck manually swapping files.

claudehopper manages this by symlinking profile-specific files into `~/.claude/` when you switch profiles. Credentials, history, and project memory are never touched.

## Set Up

### Let Claude do it

Paste this into Claude Code and it will install claudehopper, capture your current setup, and create a clean baseline profile:

```
Fetch https://raw.githubusercontent.com/matthewfritsch/claudehopper/main/docs/claude-setup-guide.md and follow it to set up claudehopper for me.
```

### Do it yourself

```bash
uv tool install claudehopper
hop create main --from-current -d "My current setup" --activate
hop create vanilla -d "Stock Claude Code"
hop tree
```

Both `claudehopper` and `hop` work as the CLI command.

## Usage

```bash
hop switch vanilla        # clean Claude Code
hop switch main           # back to your setup
hop status                # what's active?
hop list                  # all profiles
hop tree                  # visual overview
hop stats                 # which profiles you use most
```

### Sharing between profiles

By default, permissions (`settings.json`, `settings.local.json`) and MCP config (`.mcp.json`) are shared across all profiles — edit once, applies everywhere. Use `--no-shared-defaults` on `create` to opt out.

For other files:

```bash
hop share work CLAUDE.md --target personal    # symlink (changes sync)
hop pick work commands/ --target personal     # copy (independent)
hop unshare CLAUDE.md                         # back to independent copy
hop diff work personal                        # compare two profiles
```

### All commands

| Command | What it does |
|---|---|
| `hop status` | Show active profile and link status |
| `hop list` | List all profiles |
| `hop create <name>` | Create a profile (`--from-current`, `--from-profile`, `--no-shared-defaults`) |
| `hop switch <name>` | Switch to a profile (`--force`, `--dry-run`) |
| `hop pick <src> <paths>` | Copy files between profiles |
| `hop share <src> <paths>` | Share files via symlinks |
| `hop unshare [paths]` | Convert shared files back to copies |
| `hop diff <a> <b>` | Compare two profiles |
| `hop delete <name>` | Delete a profile |
| `hop tree` | Visual tree with lineage and shared files (`--json`) |
| `hop stats` | Usage statistics (`--profile`, `--since`, `--json`) |
| `hop path <name>` | Print profile directory path |
| `hop update` | Check for / install updates (`--check`) |
| `hop unmanage` | Stop using claudehopper, restore real files |

Run `hop <command> --help` for detailed usage and examples.

## How It Works

Each profile lives in `~/.config/claudehopper/profiles/<name>/`. When you `hop switch`, claudehopper replaces managed files in `~/.claude/` with symlinks to the target profile. Claude Code sees normal files and never knows the difference.

Files that belong to Claude Code itself — credentials, history, project memory, cache — are never touched. A `.hop-manifest.json` in each profile tracks what's managed, what's shared, and where it came from.

## For AI Agents

If claudehopper is already installed and you're an AI agent helping the user manage profiles, run `hop status` to see the current state, then use `hop --help` and `hop <command> --help` for full usage. The key commands are `create`, `switch`, `share`, `pick`, and `tree`.

## Safety

- Credentials and history are never modified, copied, or symlinked
- `--dry-run` on `switch` previews changes before applying
- `unmanage` materializes all symlinks back to real files — you never lose data
- Conflicting files in `~/.claude/` are backed up before being replaced
