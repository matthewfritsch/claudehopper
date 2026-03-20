# claudehopper

Switch between Claude Code configuration profiles. Run Oh-my-claudecode one session, vanilla the next, GSD for a sprint — without blowing away your setup each time. Share what you want between profiles, but keep other files separate.

## The Problem

Claude Code stores everything in `~/.claude/`: your global instructions, slash commands, MCP server config, plugins, and more. When you work across different contexts (work, personal, experimental), you need different configurations — but swapping files manually is tedious and error-prone.

## Set Up

### Let Claude do it

Paste this into Claude Code and it will install claudehopper, capture your current setup, and create a clean baseline profile:

```
Fetch this link, follow it to set up claudehopper for me:
https://raw.githubusercontent.com/matthewfritsch/claudehopper/main/docs/claude-setup-guide.md
```

### Do it yourself

**From source (requires Go 1.25+):**

```bash
go install github.com/matthewfritsch/claudehopper@latest
ln -sf "$(go env GOPATH)/bin/claudehopper" "$(go env GOPATH)/bin/hop"
```

**From GitHub Releases:**

Download the binary for your platform from [Releases](https://github.com/matthewfritsch/claudehopper/releases), extract, and move both `claudehopper` and `hop` to your PATH.

Both `claudehopper` and `hop` work as the CLI command.

## Quick Start

```bash
# Capture your current setup as a profile
hop create work --from-current

# Create a clean profile for personal use
hop create personal

# Switch between them
hop switch personal
hop switch work

# See what you have
hop list
hop status
hop tree
```

## Usage

### Profile Management

```bash
# Create profiles
hop create <name>                          # blank profile
hop create <name> --from-current           # capture current ~/.claude/ setup
hop create <name> --from-profile <source>  # clone an existing profile
hop create <name> --activate               # create and switch in one step

# Switch profiles
hop switch <name>                          # switch active profile
hop switch <name> --dry-run                # preview what would change
hop switch <name> --force                  # backup conflicts automatically

# View profiles
hop list                                   # list all profiles
hop status                                 # active profile link health
hop tree                                   # lineage tree visualization
hop tree --json                            # machine-readable tree

# Compare and inspect
hop diff <profile_a> <profile_b>           # compare two profiles
hop path <name>                            # print profile directory (for scripting)
hop stats                                  # usage statistics
hop stats --since 2025-01-01               # filtered by date
```

### File Sharing

```bash
# Share files between profiles (symlinked — changes sync)
hop share CLAUDE.md --from work --to personal

# Copy files between profiles (independent copies)
hop pick commands/ --from work --to personal

# Stop sharing (materialize symlink to real file)
hop unshare CLAUDE.md --profile personal
```

By default, `settings.json`, `settings.local.json`, and `.mcp.json` are shared across all profiles automatically.

### Session Management

Browse, resume, and clean up Claude Code sessions across all your projects.

```bash
# List sessions grouped by project
hop sesh list                              # grouped by project
hop sesh list --flat                       # flat table view
hop sesh list -p myproject                 # filter by project name

# Inspect and resume
hop sesh info <id>                         # detailed session view
hop sesh resume <id>                       # print the resume command
hop sesh resume <id> -x                    # exec directly into the session

# AI-generated titles (cached, uses haiku)
hop sesh titles                            # generate titles for all sessions
hop sesh titles -p myproject               # only for one project
hop sesh titles clear                      # clear the title cache

# Pruning old sessions
hop sesh prune --older-than 30d            # remove sessions older than 30 days
hop sesh prune --older-than 2w --dry-run   # preview what would be deleted

# Overview
hop sesh stats                             # session count, disk usage, top projects
```

Session IDs support prefix matching — `hop sesh info fa4d` works if the prefix is unambiguous. Tab completion is available for session IDs after setting up shell completions.

### Maintenance

```bash
hop update                                 # update to latest version
hop delete <name>                          # delete a profile
hop unmanage                               # stop using claudehopper entirely
```

## How It Works

Profiles are stored in `~/.config/claudehopper/profiles/<name>/`. When you switch profiles, claudehopper creates symlinks from `~/.claude/` to the active profile's files. The switch is atomic — if interrupted, your config won't be left in a broken state.

Each profile has a `.hop-manifest.json` tracking which files are managed, which are shared between profiles, and where the profile was cloned from.

### What's Protected

These paths are **never touched** by claudehopper — they stay in `~/.claude/` and are shared across all profiles automatically:

- `.credentials.json` — your auth tokens
- `history.jsonl` — chat history
- `projects/` — project memory
- `cache/`, `downloads/`, `transcripts/`, `backups/`
- `shell-snapshots/`, `file-history/`, `session-env/`
- `.session-stats.json`

You never need to re-login when switching profiles.

### What's Shared by Default

These files are symlinked through a shared directory so changes propagate to all profiles:

- `settings.json` — editor settings
- `settings.local.json` — local overrides
- `.mcp.json` — MCP server configuration

Use `hop share` and `hop unshare` to control sharing for any file.

## Shell Completions

```bash
# Bash
hop completion bash > /etc/bash_completion.d/hop

# Zsh
hop completion zsh > "${fpath[1]}/_hop"

# Fish
hop completion fish > ~/.config/fish/completions/hop.fish

# PowerShell
hop completion powershell | Out-String | Invoke-Expression
```

## For AI Agents

If claudehopper is already installed and you're an AI agent helping the user:

- **Profiles**: Run `hop status` to see the current state. Key commands: `create`, `switch`, `share`, `pick`, `tree`.
- **Sessions**: Run `hop sesh list` to see recent sessions. Use `hop sesh info <id>` for details. Generate readable titles with `hop sesh titles`.
- Use `hop --help` and `hop <command> --help` for full usage.

## Safety

- **Atomic switching** — symlinks are replaced atomically (temp + rename), never removed then recreated
- **Backup on conflict** — files are renamed to `.hop-backup` before overwriting
- **Dry-run** — preview any destructive operation with `--dry-run`
- **Protected paths** — credentials and history are never touched
- **Exit ramp** — `hop unmanage` materializes all symlinks and leaves `~/.claude/` self-contained

## License

MIT — see [LICENSE](LICENSE)
