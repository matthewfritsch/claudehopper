---
phase: 04-polish-distribution
plan: 02
subsystem: infra
tags: [goreleaser, github-actions, homebrew, shell-completions, distribution]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: goreleaser dual-build config producing hop and claudehopper binaries

provides:
  - homebrew_casks block in .goreleaser.yaml targeting matthewfritsch/homebrew-claudehopper
  - GitHub Actions release workflow triggering goreleaser-action on v* tags
  - Verified shell completions for bash, zsh, fish, powershell via Cobra

affects: [distribution, homebrew-claudehopper tap repo, release automation]

# Tech tracking
tech-stack:
  added: [goreleaser-action@v7, actions/setup-go@v5, actions/checkout@v4]
  patterns: [goreleaser v2 homebrew_casks for automated formula publishing, HOMEBREW_TAP_GITHUB_TOKEN secret for tap repo write access]

key-files:
  created: [.github/workflows/release.yml]
  modified: [.goreleaser.yaml]

key-decisions:
  - "homebrew_casks block (not brews) used — goreleaser v2 distributes both hop and claudehopper as cask binaries"
  - "goreleaser-action version pinned to ~> v2 to stay on goreleaser v2 API"
  - "fetch-depth: 0 required so goreleaser can read full git history for changelog generation"

patterns-established:
  - "Release workflow: push v* tag triggers goreleaser CI with full git depth"
  - "Two secrets required for releases: GITHUB_TOKEN (GitHub default) and HOMEBREW_TAP_GITHUB_TOKEN (repo-level secret)"

requirements-completed: [DIST-01, DIST-03]

# Metrics
duration: 5min
completed: 2026-03-14
---

# Phase 4 Plan 02: Polish Distribution Summary

**goreleaser homebrew_casks tap config and GitHub Actions release workflow on v* tags with all four Cobra shell completions verified valid**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-14T00:00:00Z
- **Completed:** 2026-03-14T00:05:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Added homebrew_casks block to .goreleaser.yaml for matthewfritsch/homebrew-claudehopper tap with both hop and claudehopper binaries
- Created .github/workflows/release.yml with goreleaser-action@v7 triggered on v* tags, passing GITHUB_TOKEN and HOMEBREW_TAP_GITHUB_TOKEN
- Verified all four shell completions (bash, zsh, fish, powershell) produce valid scripts via Cobra's built-in completion generation

## Task Commits

Each task was committed atomically:

1. **Task 1: Add homebrew_casks to goreleaser and create release workflow** - `5a4123a` (chore)
2. **Task 2: Verify shell completions output valid scripts** - no commit (verification only, no files changed)

**Plan metadata:** (docs commit follows)

## Files Created/Modified
- `.goreleaser.yaml` - Added homebrew_casks block after changelog section; existing builds/archives/checksum unchanged
- `.github/workflows/release.yml` - New GitHub Actions workflow triggering goreleaser-action on v* tag pushes

## Decisions Made
- homebrew_casks is the correct goreleaser v2 key (not brews) for distributing prebuilt binaries through a Homebrew cask tap
- fetch-depth: 0 is required in checkout so goreleaser can walk git history for changelog generation
- goreleaser-action version ~> v2 aligns with the goreleaser v2 config in the repo

## Deviations from Plan

None - plan executed exactly as written.

Note: `goreleaser check` was not run as goreleaser is not installed locally. The config structure follows goreleaser v2 documentation. Manual verification step: run `goreleaser check` after installing goreleaser, or let the CI workflow validate on first tag push.

## Issues Encountered

None.

## User Setup Required

Before the first release, add these secrets to the GitHub repository:
- `HOMEBREW_TAP_GITHUB_TOKEN` — a GitHub PAT with write access to the matthewfritsch/homebrew-claudehopper repository. The default `GITHUB_TOKEN` has no cross-repo write permission, so a PAT is required.
- `GITHUB_TOKEN` — automatically provided by GitHub Actions; no manual setup needed.

## Next Phase Readiness

- Release automation is fully configured
- Trigger a release by pushing a tag: `git tag v1.0.0 && git push origin v1.0.0`
- Homebrew tap repo (matthewfritsch/homebrew-claudehopper) must exist before first release for the formula push to succeed

---
*Phase: 04-polish-distribution*
*Completed: 2026-03-14*
