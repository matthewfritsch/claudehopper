---
phase: 4
slug: polish-distribution
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-15
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) |
| **Config file** | none |
| **Quick run command** | `go test ./...` |
| **Full suite command** | `go test -v -race ./...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test -v -race ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | OPS-02 | unit | `go test ./internal/updater/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | DIST-01 | manual | `hop completion bash \| head -5` | N/A | ⬜ pending |
| TBD | TBD | TBD | DIST-03 | manual | `goreleaser check` | N/A | ⬜ pending |

---

## Wave 0 Requirements

- [ ] `internal/updater/updater_test.go` — tests for update checking with 24h TTL

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Shell completions work | DIST-01 | Requires actual shell environment | Run `hop completion bash/zsh/fish/powershell` and verify output |
| goreleaser produces both binaries | DIST-03 | Requires goreleaser build | Run `goreleaser check` and `goreleaser build --snapshot` |
| Homebrew tap formula | DIST-03 | Requires separate repo | Verify formula installs correctly |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
