# Agents

## executor

When working on this codebase:

- Run `python -m pytest tests/ -v` after any change to `cli.py`
- This is a single-module CLI — all logic lives in `src/claude_swap/cli.py`
- The test base class `CCSwapTestCase` patches `CLAUDE_DIR`, `SWAP_DIR`, `PROFILES_DIR`, and `CONFIG_FILE` to temp dirs. Always use it.
- Never import or run ccswap against the real `~/.claude/` directory during development or testing

## code-reviewer

When reviewing this codebase, pay attention to:

- **Safety:** Every operation that touches the filesystem must validate paths before mutating. Check for TOCTOU races (time-of-check-time-of-use) between validation and execution.
- **Symlink handling:** Operations must handle symlinks, broken symlinks, and dangling symlinks without crashing. Use `is_symlink()` checks before `resolve()`.
- **Shared path protection:** The `SHARED_PATHS` set must never be included in profile operations. Credentials, history, and project memory must never be copied, moved, or symlinked.
- **Backup safety:** When backing up conflicting files, `backup_path()` must never collide with existing files.

## test-engineer

When adding tests:

- Extend `CCSwapTestCase` — it handles temp dir setup/teardown and path patching
- Test both the happy path and error cases (missing profiles, invalid names, conflicts)
- Test `--dry-run` variants to ensure they don't mutate state
- Test symlink behavior: share creates symlinks, pick creates copies, unshare materializes
- Verify `SHARED_PATHS` are never included in profile operations
