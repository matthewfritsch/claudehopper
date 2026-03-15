---
phase: 3
slug: extended-features
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-15
---

# Phase 3 — Validation Strategy

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
| TBD | TBD | TBD | SHAR-01 | unit | `go test ./internal/profile/... -run TestShare` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SHAR-02 | unit | `go test ./internal/profile/... -run TestPick` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SHAR-03 | unit | `go test ./internal/profile/... -run TestUnshare` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | VIZ-01 | unit | `go test ./internal/profile/... -run TestTree` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | VIZ-02 | unit | `go test ./internal/profile/... -run TestDiff` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | VIZ-03 | unit | `go test ./internal/usage/... -run TestStats` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | VIZ-04 | unit | `go test ./cmd/... -run TestPath` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | OPS-01 | unit | `go test ./internal/profile/... -run TestUnmanage` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | OPS-03 | unit | `go test ./internal/usage/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/profile/share_test.go` — tests for share/pick/unshare
- [ ] `internal/profile/tree_test.go` — tests for tree generation
- [ ] `internal/profile/diff_test.go` — tests for profile comparison
- [ ] `internal/usage/usage_test.go` — tests for usage recording and stats
- [ ] `internal/profile/unmanage_test.go` — tests for unmanage operation

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| None | — | — | — |

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
