---
phase: 1
slug: foundation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-14
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) |
| **Config file** | none — go test uses conventions |
| **Quick run command** | `go test ./...` |
| **Full suite command** | `go test -v -race ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test -v -race ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | SAFE-01 | unit | `go test ./internal/fs/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SAFE-03 | unit | `go test ./internal/config/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | DIST-02 | integration | `go build -o /dev/null ./...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | DIST-04 | integration | `go build && ./claudehopper --help` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/fs/fs_test.go` — tests for AtomicSymlink, IsProtected, BackupPath
- [ ] `internal/config/config_test.go` — tests for path resolution, XDG override
- [ ] `testdata/` fixtures — Python-generated manifest and config JSON files

*Existing infrastructure covers framework — Go test is built-in.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `hop` alias works | DIST-02 | Depends on install mechanism (Makefile) | Run `make install && hop --help` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
