---
phase: 2
slug: core-profile-operations
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-14
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) |
| **Config file** | none — go test uses conventions |
| **Quick run command** | `go test ./...` |
| **Full suite command** | `go test -v -race ./...` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test -v -race ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | PROF-01 | unit | `go test ./internal/profile/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | PROF-02 | unit+integration | `go test ./internal/profile/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | PROF-03 | unit | `go test ./internal/profile/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | PROF-04 | unit | `go test ./cmd/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | PROF-05 | unit | `go test ./cmd/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | PROF-06 | unit | `go test ./internal/profile/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | PROF-07 | integration | `go test ./cmd/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SWCH-01 | integration | `go test ./internal/profile/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SWCH-02 | unit | `go test ./internal/profile/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SWCH-03 | unit | `go test ./internal/profile/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SWCH-04 | unit | `go test ./internal/profile/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SWCH-05 | unit | `go test ./internal/profile/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SWCH-06 | unit | `go test ./internal/profile/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SAFE-02 | unit | `go test ./internal/config/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SHAR-04 | integration | `go test ./internal/profile/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/profile/profile_test.go` — tests for create, switch, delete, adopt
- [ ] `internal/profile/testdata/` — Python-generated manifest and config fixtures
- [ ] `cmd/create_test.go` — tests for create command flags
- [ ] `cmd/switch_test.go` — tests for switch command with dry-run

*Existing Phase 1 infrastructure covers framework — Go test is built-in.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Adopt-on-switch interactive prompt | SWCH-06 | Requires TTY interaction | Run `hop switch <name>` with unmanaged files, verify prompt appears |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
