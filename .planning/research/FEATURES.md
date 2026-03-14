# Feature Research

**Domain:** CLI config profile manager (symlink-based, single application scope)
**Researched:** 2026-03-14
**Confidence:** HIGH — based on direct reading of the Python source (~1480 lines), README, and comparison with comparable tools (chezmoi, dotstate, shprofile, AWS CLI profiles)

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete or untrustworthy.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Profile create (blank) | Every profile manager starts with creation | LOW | Writes minimal settings.json + manifest |
| Profile switch | Core purpose of the tool | MEDIUM | Must be atomic; unlink old, link new |
| Profile list | "What do I have?" is the first question | LOW | Show name, active marker, path count |
| Profile status | "What's active right now?" | LOW | Show active profile, link health per path |
| Profile delete | Housekeeping; any tool that creates must delete | LOW | Guard: must not be active; warn about dependents |
| Protected paths (never touch credentials/history) | Trust; users will not adopt a tool that could corrupt their credentials | LOW | Compile-time constant list; checked on every switch |
| Manifest tracking | Users need to know what is/isn't managed | LOW | .hop-manifest.json per profile; must be human-readable |
| Dry-run on switch | Destructive operation; users want to preview before committing | LOW | Print action list without touching filesystem |
| Backup on conflict | Protect pre-existing files when first adopting the tool | LOW | Rename conflicting file with .hop-backup suffix |
| Atomic symlink creation | Prevents corrupt state if process is interrupted | LOW | temp-file + rename idiom |
| --help on every subcommand | Standard CLI contract | LOW | Cobra provides this; requires good docstrings |
| Version flag | Standard binary contract | LOW | Cobra --version flag |
| Shell tab completions | Power users expect completions for any tool they live in | MEDIUM | Cobra generates bash/zsh/fish/powershell; must wire up |
| Dual binary names (hop + claudehopper) | Parity with Python version; existing users expect `hop` | LOW | goreleaser multi-binary or symlink in install |

### Differentiators (Competitive Advantage)

Features that set the product apart from generic dotfile managers and the Python version.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Profile creation from current config (--from-current) | Zero-friction onboarding; users already have a config they want to capture | MEDIUM | Must detect profile-specific paths, exclude SHARED_PATHS, handle symlinks vs real files |
| Profile cloning with lineage tracking (--from-profile) | Create variants of a known-good baseline; tree visualization depends on this | MEDIUM | Copy profile dir + record created_from in manifest |
| File sharing between profiles (share/unshare via symlinks) | Edit once, get updates everywhere — essential for settings.json and .mcp.json shared across work/personal | MEDIUM | Intra-profile symlink; manifest records source; unshare materializes a real copy |
| File cherry-pick between profiles (pick = copy, not share) | Take one file from another profile without locking in a live link | LOW | Simple copy + manifest update |
| Adopt-on-switch for unmanaged files | Safe migration when switching away from an unmanaged state; prevents silent data loss | MEDIUM | Detect files in ~/.claude/ not in manifest; prompt user before switching away |
| Default linked files (settings.json, .mcp.json across all profiles) | Eliminates most common footgun: accidentally having different permissions per profile | LOW | Shared dir seeded on first profile creation; symlinked into every new profile by default |
| Profile visualization: tree with lineage | Visual overview of how profiles relate; shows created_from parent-child chains | MEDIUM | Recursive ASCII tree; must handle cycles gracefully |
| Profile diff command | "How are these two profiles different?" is a natural question when managing variants | MEDIUM | Set operations on managed paths + byte-level file comparison |
| Usage statistics (usage.jsonl + stats command) | Which profiles do I actually use? Helps users clean up stale profiles | LOW | Append-only JSONL; aggregate on read; --json output for scripting |
| Profile path command (path <name>) | Scripting bridge: `cd $(hop path work)` | LOW | One-liner print of profile dir |
| Unmanage command | Exit ramp: restore real files, stop using claudehopper | LOW | Materialize all symlinks; clear active in config.json |
| Update checking (cached, non-blocking, 24h TTL) | Users get notified without being nagged | LOW | Background GitHub release check; displayed after status |
| --json output flag (tree, stats) | Scripting and programmatic use; pairs well with jq | LOW | Cobra flag; JSON output on tree and stats |
| Format compatibility with Python version | Existing users can switch without re-creating profiles | LOW | Read/write same .hop-manifest.json, config.json schemas |
| --activate flag on create | Reduces two commands to one for the common "create and immediately use" flow | LOW | Call switch internally after create |
| Manifest validation on switch | Catch corrupt or stale manifests before touching ~/.claude/ | LOW | Check managed_paths exist in profile dir before unlinking anything |
| Dependent-profile warning on delete | Prevent dangling shared symlinks | LOW | Scan all manifests for shared_paths referencing the deleted profile |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems for this specific tool.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| TUI (interactive menu) | Feels modern; easier to browse profiles | Violates the "single command" mental model; adds a heavy dependency (bubbletea); hard to script; out of scope per PROJECT.md | The `tree` and `list` commands provide the same information; tab completion removes the browsing need |
| Cloud sync / remote backup of profiles | "My profiles should follow me" | Profiles may contain sensitive instructions; syncing requires auth; far outside scope; pushes tool into credentials management territory | Users can version-control their profile dirs with git independently |
| Automatic profile detection (switch based on directory/project) | "Hop to work profile when I cd into ~/work/" | Requires shell hook injection; fragile across shells; creates surprising behavior | The `hop switch` UX is already one command; shell alias can automate if user wants |
| Profile encryption | "Some profiles have secrets" | Claude Code config generally doesn't hold secrets (credentials are in SHARED_PATHS and never touched); encryption adds key management complexity | Protected paths list handles the actual secrets; CLAUDE.md is not sensitive in the encryption sense |
| Undo / rollback last switch | "I want to go back to the previous profile" | State tracking adds complexity and another storage concern; the need is covered differently | `hop switch <previous>` is one command; `hop list` shows available profiles; true rollback semantics are not meaningful when the profile itself may have changed |
| Profile locking (prevent switch while Claude is running) | "Don't switch while Claude is active" | Process detection is unreliable; partial lock creates worse failure modes than no lock | Dry-run lets users verify before switching; atomic symlink creation minimizes the switch window |
| Sub-profiles / profile inheritance | "personal-relaxed should inherit from personal" | Lineage tracking already serves this; true inheritance (merge semantics) adds ambiguity about conflict resolution | Use `--from-profile` to clone; use `share` to keep specific files in sync |

## Feature Dependencies

```
[Profile Switch]
    ├──requires──> [Profile Create]
    │                  └──requires──> [Manifest Tracking]
    ├──requires──> [Protected Paths List]
    ├──requires──> [Atomic Symlink]
    └──requires──> [Backup on Conflict]

[Profile Share]
    └──requires──> [Manifest Tracking]
                       └──requires──> [Profile Create]

[Profile Pick]
    └──requires──> [Profile Create]

[Profile Unshare]
    └──requires──> [Profile Share]

[Profile Tree]
    └──requires──> [Profile Create]  (for lineage data in manifests)

[Profile Diff]
    └──requires──> [Profile Create]

[Profile Stats]
    └──requires──> [Usage Tracking]  (record_usage called at switch/create/delete/pick)

[Default Linked Files]
    └──requires──> [Profile Share]  (same symlink mechanism)

[Adopt-on-Switch]
    └──requires──> [Profile Switch]

[Update Check]
    ──independent──> (no feature dependencies; runs after status)

[Shell Completions]
    ──enhances──> [All commands]  (Cobra generates from command tree)

[--json output]
    ──enhances──> [Profile Tree]
    ──enhances──> [Profile Stats]
```

### Dependency Notes

- **Profile Switch requires Profile Create**: you cannot switch to a profile that does not exist; `require_profile()` enforces this.
- **Manifest Tracking is a foundation**: every command that touches files reads or writes the manifest. It must work correctly before any other feature is reliable.
- **Profile Share and Profile Unshare are paired**: unshare is only meaningful if share has been used; they operate on the `shared_paths` manifest field.
- **Adopt-on-Switch enhances Profile Switch**: it triggers during switch, not as a standalone command; depends on the unmanaged file detection logic.
- **Default Linked Files uses Share mechanics**: the shared/ dir + symlink pattern is the same mechanism as `hop share`, just automated at profile creation time.
- **Usage Tracking enhances Profile Stats**: stats is useless without usage.jsonl; usage.jsonl is populated by switch, create, delete, pick actions.

## MVP Definition

### Launch With (v1)

Minimum viable product — what existing Python claudehopper users need to migrate to the Go version.

- [ ] Profile create (blank, --from-current, --from-profile) — users cannot adopt the tool without onboarding their current config
- [ ] Profile switch (with atomic symlinks, dry-run, backup on conflict) — the core value proposition
- [ ] Protected paths enforcement — non-negotiable safety guarantee; losing credentials would destroy trust
- [ ] Manifest tracking (read/write .hop-manifest.json in Python-compatible format) — required for all other commands and for format compatibility
- [ ] Profile list and status — basic observability; users need to know what state they are in
- [ ] Profile delete — users will create test profiles and need to clean up
- [ ] Default linked files (settings.json, settings.local.json, .mcp.json shared) — prevents the most common footgun
- [ ] Adopt-on-switch — prevents silent data loss when switching away from unmanaged state
- [ ] Format compatibility (same config.json, .hop-manifest.json, directory layout) — existing Python users must not lose their profiles
- [ ] Shell completions (bash, zsh, fish, powershell via Cobra) — this is an explicit goal of the Go port; Cobra makes it low cost
- [ ] Dual binary names (hop + claudehopper) — parity with Python version; existing muscle memory

### Add After Validation (v1.x)

Features to add once core switch/create/list are solid.

- [ ] Profile share + pick + unshare — file-level sharing is heavily used but not blocking initial adoption
- [ ] Profile diff — diagnostic; useful but not required for basic use
- [ ] Profile tree with lineage visualization — nice-to-have observability; creatable once manifests work
- [ ] Profile stats + usage tracking — analytics layer; append-only JSONL is low risk to add incrementally
- [ ] Profile path command — scripting helper; trivial once profile lookup works
- [ ] Unmanage command — exit ramp; needed before declaring v1 "complete" but not for initial validation
- [ ] Update checking (cached GitHub release check) — operational quality-of-life; add once binary is distributed
- [ ] --json output on tree and stats — scripting polish; add alongside tree and stats

### Future Consideration (v2+)

Features to defer until tool is stable and user feedback is gathered.

- [ ] AI agent setup guide equivalent (docs/claude-setup-guide.md for Go version) — documentation artifact, not a code feature
- [ ] goreleaser config and prebuilt binaries — distribution infrastructure; needed for non-Go users but not for initial `go install` adoption

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Profile create (blank + from-current + from-profile) | HIGH | MEDIUM | P1 |
| Profile switch with dry-run and atomic symlinks | HIGH | MEDIUM | P1 |
| Protected paths enforcement | HIGH | LOW | P1 |
| Manifest read/write (format-compatible) | HIGH | LOW | P1 |
| Profile list + status | HIGH | LOW | P1 |
| Default linked files (settings.json, .mcp.json) | HIGH | LOW | P1 |
| Adopt-on-switch | HIGH | MEDIUM | P1 |
| Shell completions (Cobra) | HIGH | LOW | P1 |
| Dual binary names (hop + claudehopper) | HIGH | LOW | P1 |
| Profile delete with dependent warning | MEDIUM | LOW | P1 |
| Profile share / pick / unshare | MEDIUM | MEDIUM | P2 |
| Profile diff | MEDIUM | LOW | P2 |
| Profile tree with lineage | MEDIUM | MEDIUM | P2 |
| Usage tracking + stats | LOW | LOW | P2 |
| Profile path command | LOW | LOW | P2 |
| Unmanage command | MEDIUM | LOW | P2 |
| Update checking (cached, non-blocking) | LOW | LOW | P2 |
| --json output flags | LOW | LOW | P2 |
| Backup on conflict | HIGH | LOW | P1 |
| Manifest validation preflight on switch | HIGH | LOW | P1 |

## Competitor Feature Analysis

Comparable tools are dotfile managers (chezmoi, dotstate) and shell profile switchers (shprofile). Claudehopper is narrower in scope: it manages one application's config, not all dotfiles, which shapes what "table stakes" means here.

| Feature | chezmoi | dotstate | shprofile (bash) | claudehopper approach |
|---------|---------|----------|------------------|-----------------------|
| Profile switching | Via templates + machine flags (not instant swap) | Profile select + enter; symlinks swapped | Load/unload shell snippets | `hop switch <name>`: single command, instant |
| Protected/excluded paths | Via .chezmoiignore | Common files across all profiles | Not applicable | Compile-time SHARED_PATHS constant |
| Dry-run | `chezmoi diff` (shows what would change) | Not documented | Not applicable | `--dry-run` flag on switch, pick, share, unshare |
| File sharing between profiles | Not native (templates handle variance) | Common files are auto-shared | Not applicable | `hop share` (symlink) + `hop pick` (copy) |
| Lineage tracking / tree view | Not present | Not present | Not present | `hop tree` with parent-child lineage in manifest |
| Usage statistics | Not present | Not present | Not present | `hop stats` via usage.jsonl |
| Unmanage/exit ramp | `chezmoi unmanage` exists | Not documented | Unload profile | `hop unmanage` materializes symlinks to real files |
| Application-scoped (vs all dotfiles) | No — manages all dotfiles | No — manages all dotfiles | Shell only | Yes — only manages ~/.claude/ |
| JSON output for scripting | Not built-in | Not built-in | Not applicable | `--json` on tree and stats |
| Completions | Yes (bash/zsh/fish) | Unknown | Not applicable | Yes (Cobra: bash/zsh/fish/powershell) |

The key differentiator for claudehopper-go vs. all comparables: it is scoped to a single application's config directory, making the trust and safety model simpler and the UX far less overwhelming for users who just want to swap Claude Code setups.

## Sources

- Python claudehopper source code: `/home/matthew/Programming/claudehopper/src/claudehopper/cli.py` (1480 lines, direct reading) — HIGH confidence
- claudehopper README: `/home/matthew/Programming/claudehopper/README.md` — HIGH confidence
- chezmoi documentation: https://www.chezmoi.io/ and https://www.chezmoi.io/why-use-chezmoi/ — MEDIUM confidence (web search verified)
- dotstate feature list: https://dotstate.serkan.dev/ and https://terminaltrove.com/dotstate/ — MEDIUM confidence (web search only)
- shprofile: https://github.com/abourdon/shprofile — MEDIUM confidence (web search only)
- CLI UX patterns / dry-run best practices: https://clig.dev/ and https://nickjanetakis.com/blog/cli-tools-that-support-previews-dry-runs-or-non-destructive-actions — MEDIUM confidence
- AWS CLI named profiles: https://docs.aws.amazon.com/cli/v1/userguide/cli-configure-profiles.html — HIGH confidence (official docs)

---
*Feature research for: CLI config profile manager (claudehopper-go)*
*Researched: 2026-03-14*
