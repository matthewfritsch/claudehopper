#!/usr/bin/env python3
"""claudehopper - Switch between Claude Code configuration profiles.

Profiles are stored in ~/.config/claudehopper/profiles/<name>/.
Profile-specific files in ~/.claude/ are symlinked to the active profile.
Shared files (credentials, history, projects, cache) are never touched.
"""

import argparse
import datetime
import json
import os
import shutil
import subprocess
import sys
import tempfile
import urllib.request
import urllib.error
from pathlib import Path

CLAUDE_DIR = Path.home() / ".claude"
HOPPER_DIR = Path.home() / ".config" / "claudehopper"
PROFILES_DIR = HOPPER_DIR / "profiles"
CONFIG_FILE = HOPPER_DIR / "config.json"

MANIFEST_NAME = ".hop-manifest.json"
BACKUP_SUFFIX = ".hop-backup"

SHARED_DIR = HOPPER_DIR / "shared"
USAGE_FILE_NAME = "usage.jsonl"
UPDATE_CHECK_FILE = "last-update-check.json"
GITHUB_REPO = "matthewfritsch/claudehopper"

# Paths linked across all profiles by default (permissions, MCP config).
# These live in ~/.config/claudehopper/shared/ and are symlinked into each profile.
DEFAULT_LINKED = {
    "settings.json",
    "settings.local.json",
    ".mcp.json",
}

# These belong to Claude Code itself — never profile-managed
SHARED_PATHS = {
    ".credentials.json",
    "history.jsonl",
    "projects",
    "cache",
    "downloads",
    "transcripts",
    "shell-snapshots",
    "file-history",
    "backups",
    "session-env",
    ".session-stats.json",
}


def die(msg: str):
    print(f"error: {msg}", file=sys.stderr)
    sys.exit(1)


def _prompt(msg: str) -> str:
    """Prompt the user for input. Extracted for testability."""
    try:
        return input(msg).strip().lower()
    except (EOFError, KeyboardInterrupt):
        print()
        return "n"


def ensure_shared_defaults():
    """Ensure ~/.config/claudehopper/shared/ exists with default linked files.

    On first run, if the shared dir doesn't exist, copies any matching files
    from the first profile or creates empty defaults.
    """
    if SHARED_DIR.exists():
        return
    SHARED_DIR.mkdir(parents=True, exist_ok=True)


def link_defaults_into_profile(pdir: Path, from_source: Path | None = None):
    """Link DEFAULT_LINKED files from shared dir into a profile.

    If the shared dir doesn't have a file yet but from_source does,
    moves the file into shared first (bootstrapping).
    """
    ensure_shared_defaults()
    shared_paths = get_shared_paths(pdir)
    managed = get_managed_paths(pdir)

    for filename in DEFAULT_LINKED:
        shared_file = SHARED_DIR / filename
        profile_file = pdir / filename

        # Bootstrap: if shared dir doesn't have it yet, seed from source
        if not shared_file.exists() and from_source:
            source_file = from_source / filename
            if source_file.exists() and not source_file.is_symlink():
                shutil.copy2(source_file, shared_file)
            elif source_file.is_symlink():
                # Follow the symlink and copy the real content
                resolved = source_file.resolve()
                if resolved.exists():
                    shutil.copy2(resolved, shared_file)

        if not shared_file.exists():
            continue

        # Replace the profile's copy with a symlink to shared
        if profile_file.exists() or profile_file.is_symlink():
            if profile_file.is_dir() and not profile_file.is_symlink():
                shutil.rmtree(profile_file)
            else:
                profile_file.unlink()

        profile_file.symlink_to(shared_file)
        shared_paths[filename] = "(shared)"
        if filename not in managed:
            managed.append(filename)

    save_manifest(pdir, managed, shared_paths=shared_paths)


def record_usage(profile: str, action: str):
    """Append a usage record. Never raises."""
    try:
        HOPPER_DIR.mkdir(parents=True, exist_ok=True)
        entry = {
            "profile": profile,
            "timestamp": datetime.datetime.now().isoformat(),
            "action": action,
        }
        with (HOPPER_DIR / USAGE_FILE_NAME).open("a") as f:
            f.write(json.dumps(entry) + "\n")
    except Exception as e:
        print(f"warning: could not record usage: {e}", file=sys.stderr)


def load_config() -> dict:
    if CONFIG_FILE.exists():
        return json.loads(CONFIG_FILE.read_text())
    return {"active": None}


def save_config(config: dict):
    HOPPER_DIR.mkdir(parents=True, exist_ok=True)
    CONFIG_FILE.write_text(json.dumps(config, indent=2) + "\n")


def profile_dir(name: str) -> Path:
    return PROFILES_DIR / name


def require_profile(name: str) -> Path:
    """Return profile dir or die if it doesn't exist."""
    pdir = profile_dir(name)
    if not pdir.exists():
        available = []
        if PROFILES_DIR.exists():
            available = sorted(d.name for d in PROFILES_DIR.iterdir() if d.is_dir())
        msg = f"profile '{name}' not found."
        if available:
            msg += f" Available: {', '.join(available)}"
        die(msg)
    return pdir


def load_manifest(pdir: Path) -> dict:
    """Load profile manifest."""
    manifest = pdir / MANIFEST_NAME
    if manifest.exists():
        return json.loads(manifest.read_text())
    return {"managed_paths": [], "shared_paths": {}, "description": ""}


def get_managed_paths(pdir: Path) -> list[str]:
    manifest = load_manifest(pdir)
    paths = manifest.get("managed_paths", [])
    if paths:
        return paths
    # Infer from directory contents
    return sorted(
        item.name for item in pdir.iterdir()
        if item.name != MANIFEST_NAME
    )


def get_shared_paths(pdir: Path) -> dict[str, str]:
    """Return {path: source_profile} for paths shared from other profiles."""
    manifest = load_manifest(pdir)
    return manifest.get("shared_paths", {})


def save_manifest(pdir: Path, managed_paths: list[str], description: str = "",
                  shared_paths: dict[str, str] | None = None):
    existing = load_manifest(pdir)
    data = {
        "managed_paths": sorted(set(managed_paths)),
        "shared_paths": shared_paths if shared_paths is not None else existing.get("shared_paths", {}),
        "description": description or existing.get("description", ""),
    }
    (pdir / MANIFEST_NAME).write_text(json.dumps(data, indent=2) + "\n")


def detect_profile_paths() -> list[str]:
    """Find profile-specific paths in current ~/.claude/."""
    if not CLAUDE_DIR.exists():
        return []
    return sorted(
        item.name for item in CLAUDE_DIR.iterdir()
        if item.name not in SHARED_PATHS
        and not item.name.startswith(".hop-")
        and not item.name.startswith(".ccswap")
    )


def is_our_symlink(path: Path, pdir: Path) -> bool:
    """Check if path is a symlink pointing into the given profile dir."""
    if not path.is_symlink():
        return False
    try:
        return str(path.resolve()).startswith(str(pdir.resolve()))
    except OSError:
        return False


def validate_profile_name(name: str):
    """Ensure profile name is safe for use as a directory name."""
    if not name:
        die("profile name cannot be empty")
    if name.startswith(".") or name.startswith("-"):
        die(f"profile name cannot start with '.' or '-': {name}")
    if "/" in name or "\\" in name or "\0" in name:
        die(f"profile name contains invalid characters: {name}")
    if name in (".", ".."):
        die(f"invalid profile name: {name}")


def validate_switch_preflight(name: str, force: bool) -> list[tuple[str, str]]:
    """Validate everything before a switch. Returns list of (action, detail) pairs."""
    pdir = require_profile(name)
    managed = get_managed_paths(pdir)
    actions = []
    errors = []

    config = load_config()
    current = config.get("active")

    for p in managed:
        src = pdir / p
        if not src.exists() and not src.is_symlink():
            errors.append(f"  '{p}' listed in manifest but missing from profile dir")

    if errors:
        print("Validation errors in target profile:", file=sys.stderr)
        for e in errors:
            print(e, file=sys.stderr)
        die("fix the profile before switching")

    if current:
        current_dir = profile_dir(current)
        if current_dir.exists():
            for p in get_managed_paths(current_dir):
                link = CLAUDE_DIR / p
                if link.is_symlink():
                    actions.append(("unlink", f"{p} (from profile '{current}')"))
                elif link.exists():
                    actions.append(("orphan", f"{p} (real file, not managed by current profile)"))
    else:
        for p in managed:
            existing = CLAUDE_DIR / p
            if existing.exists() and not existing.is_symlink():
                if not force:
                    die(f"unmanaged '{p}' exists in ~/.claude/. "
                        f"Run 'claudehopper create <name> --from-current' first, or use --force.")
                actions.append(("backup", p))

    for p in managed:
        src = pdir / p
        if src.exists() or src.is_symlink():
            actions.append(("link", f"{p} -> {src}"))

    return actions


def atomic_symlink(target: Path, link: Path):
    """Create a symlink atomically using temp-file-then-rename.

    This avoids a window where the link doesn't exist between unlink and symlink_to.
    """
    tmp = link.parent / f".hop-tmp-{link.name}-{os.getpid()}"
    try:
        tmp.symlink_to(target)
        tmp.rename(link)
    except BaseException:
        if tmp.is_symlink():
            tmp.unlink()
        raise


def backup_path(path: Path) -> Path:
    """Generate a backup path that doesn't collide."""
    backup = path.parent / (path.name + BACKUP_SUFFIX)
    n = 1
    while backup.exists():
        backup = path.parent / (path.name + f"{BACKUP_SUFFIX}.{n}")
        n += 1
    return backup


def link_managed_path(pdir: Path, p: str):
    """Create a symlink in CLAUDE_DIR for a single managed path."""
    src = pdir / p
    link = CLAUDE_DIR / p
    if link.is_symlink():
        link.unlink()
    elif link.is_dir():
        # Real directory in the way — back it up before replacing
        bak = backup_path(link)
        link.rename(bak)
    elif link.exists():
        link.unlink()
    if src.exists() or src.is_symlink():
        target = src.resolve() if not src.is_symlink() else src
        atomic_symlink(target, link)
        return True
    return False


# ── Update Checking ───────────────────────────────────────────────────────


def _get_current_version() -> str:
    try:
        return __import__("claudehopper").__version__
    except Exception:
        return "dev"


def _fetch_latest_release() -> dict | None:
    """Fetch latest release info from GitHub. Returns None on any failure."""
    url = f"https://api.github.com/repos/{GITHUB_REPO}/releases/latest"
    try:
        req = urllib.request.Request(url, headers={"Accept": "application/vnd.github.v3+json"})
        with urllib.request.urlopen(req, timeout=5) as resp:
            return json.loads(resp.read().decode())
    except Exception:
        return None


def _parse_version(tag: str) -> tuple:
    """Parse 'v1.2.3' or '1.2.3' into (1, 2, 3) for comparison."""
    tag = tag.lstrip("v")
    try:
        return tuple(int(x) for x in tag.split("."))
    except (ValueError, AttributeError):
        return (0,)


def _check_update_cached() -> str | None:
    """Check for updates, caching results for 24 hours. Returns new version or None."""
    cache_file = HOPPER_DIR / UPDATE_CHECK_FILE
    now = datetime.datetime.now()

    # Check cache first
    if cache_file.exists():
        try:
            cache = json.loads(cache_file.read_text())
            checked_at = datetime.datetime.fromisoformat(cache["checked_at"])
            if (now - checked_at).total_seconds() < 86400:  # 24 hours
                latest = cache.get("latest_version")
                current = _get_current_version()
                if latest and _parse_version(latest) > _parse_version(current):
                    return latest
                return None
        except Exception:
            pass

    # Fetch fresh
    release = _fetch_latest_release()
    latest_version = None
    if release:
        latest_version = release.get("tag_name", "").lstrip("v")
        try:
            HOPPER_DIR.mkdir(parents=True, exist_ok=True)
            cache_file.write_text(json.dumps({
                "checked_at": now.isoformat(),
                "latest_version": latest_version,
            }) + "\n")
        except Exception:
            pass

    current = _get_current_version()
    if latest_version and _parse_version(latest_version) > _parse_version(current):
        return latest_version
    return None


def _print_update_notice():
    """Print a one-liner if an update is available. Never raises."""
    try:
        new_version = _check_update_cached()
        if new_version:
            current = _get_current_version()
            print(f"\nUpdate available: {current} → {new_version}"
                  f"  (run 'hop update' to install)\n")
    except Exception:
        pass


def cmd_update(args):
    """Check for and install updates from GitHub."""
    current = _get_current_version()

    if args.check:
        print(f"Current version: {current}")
        print(f"Checking GitHub ({GITHUB_REPO})...")
        release = _fetch_latest_release()
        if not release:
            print("Could not reach GitHub. Check your connection.")
            return
        latest = release.get("tag_name", "").lstrip("v")
        if _parse_version(latest) > _parse_version(current):
            print(f"New version available: {latest}")
            print(f"  Release: {release.get('html_url', '')}")
            print(f"\nRun 'hop update' to install.")
        else:
            print(f"You're up to date (latest: {latest}).")
        return

    # Install update
    print(f"Current version: {current}")
    print(f"Checking GitHub ({GITHUB_REPO})...")
    release = _fetch_latest_release()
    if not release:
        print("Could not reach GitHub. Check your connection.")
        return

    latest = release.get("tag_name", "").lstrip("v")
    if _parse_version(latest) <= _parse_version(current):
        print(f"Already up to date (v{current}).")
        return

    print(f"Updating to v{latest}...")
    repo_url = f"git+https://github.com/{GITHUB_REPO}"
    try:
        result = subprocess.run(
            ["uv", "tool", "install", "--reinstall", repo_url],
            capture_output=True, text=True,
        )
        if result.returncode == 0:
            print(f"Updated to v{latest}.")
            # Clear cache so next check picks up the new version
            cache_file = HOPPER_DIR / UPDATE_CHECK_FILE
            if cache_file.exists():
                cache_file.unlink()
        else:
            print("Update failed. Try manually:")
            print(f"  uv tool install --reinstall {repo_url}")
            if result.stderr:
                print(f"\n{result.stderr.strip()}")
    except FileNotFoundError:
        print("'uv' not found. Install manually:")
        print(f"  pip install --upgrade git+https://github.com/{GITHUB_REPO}")


# ── Commands ──────────────────────────────────────────────────────────────


def cmd_status(_args):
    config = load_config()
    active = config.get("active")

    if not active:
        print("No active profile. ~/.claude/ is unmanaged.")
        paths = detect_profile_paths()
        if paths:
            print(f"\n{len(paths)} profile-specific items detected:")
            for p in paths:
                item = CLAUDE_DIR / p
                if item.is_symlink():
                    print(f"  {p}  ->  {os.readlink(item)}")
                elif item.is_dir():
                    print(f"  {p}/")
                else:
                    print(f"  {p}")
            print("\nRun 'claudehopper create <name> --from-current' to import as a profile.")
        return

    pdir = profile_dir(active)
    managed = get_managed_paths(pdir)
    shared = get_shared_paths(pdir)
    manifest = load_manifest(pdir)

    print(f"Profile:  {active}")
    if manifest.get("description"):
        print(f"Desc:     {manifest['description']}")
    print(f"Location: {pdir}")
    print(f"Managed:  {len(managed)} paths")

    for p in sorted(managed):
        link = CLAUDE_DIR / p
        if is_our_symlink(link, pdir):
            label = "linked"
            if p in shared:
                label += f", shared from '{shared[p]}'"
            status = label
        elif link.exists():
            status = "CONFLICT (real file exists)"
        else:
            src = pdir / p
            status = "not linked" + ("" if src.exists() else " (missing in profile)")
        print(f"  {p}  [{status}]")

    _print_update_notice()


def cmd_list(_args):
    config = load_config()
    active = config.get("active")

    if not PROFILES_DIR.exists():
        print("No profiles. Run 'claudehopper create <name>' to create one.")
        return

    profiles = sorted(d.name for d in PROFILES_DIR.iterdir() if d.is_dir())
    if not profiles:
        print("No profiles. Run 'claudehopper create <name>' to create one.")
        return

    for name in profiles:
        marker = " (active)" if name == active else ""
        pdir = profile_dir(name)
        managed = get_managed_paths(pdir)
        shared = get_shared_paths(pdir)
        manifest = load_manifest(pdir)
        desc = f" - {manifest['description']}" if manifest.get("description") else ""
        shared_note = f", {len(shared)} shared" if shared else ""
        print(f"  {name}{marker}  [{len(managed)} paths{shared_note}]{desc}")


def cmd_create(args):
    name = args.name
    validate_profile_name(name)
    pdir = profile_dir(name)

    if pdir.exists():
        die(f"profile '{name}' already exists")

    pdir.mkdir(parents=True)
    created_from = None

    if args.from_current:
        paths = detect_profile_paths()

        errors = []
        for p in paths:
            src = CLAUDE_DIR / p
            if not src.exists() and not src.is_symlink():
                errors.append(f"  '{p}' detected but doesn't exist (race condition?)")
        if errors:
            shutil.rmtree(pdir)
            print("Validation failed:", file=sys.stderr)
            for e in errors:
                print(e, file=sys.stderr)
            die("aborting profile creation")

        for p in paths:
            src = CLAUDE_DIR / p
            dst = pdir / p
            if src.is_symlink():
                os.symlink(os.readlink(src), dst)
            elif src.is_dir():
                shutil.copytree(src, dst, symlinks=True, ignore_dangling_symlinks=True,
                                copy_function=shutil.copy2)
            elif src.is_file():
                shutil.copy2(src, dst)
        created_from = "(current ~/.claude/)"
        save_manifest(pdir, paths, args.description or "")
        print(f"Created '{name}' from current ~/.claude/ ({len(paths)} paths)")

        if args.activate:
            _do_switch(name, force=True)

    elif args.from_profile:
        src_dir = require_profile(args.from_profile)
        shutil.copytree(src_dir, pdir, dirs_exist_ok=True, symlinks=True,
                        ignore_dangling_symlinks=True)
        created_from = args.from_profile
        manifest = load_manifest(pdir)
        if args.description:
            save_manifest(pdir, manifest.get("managed_paths", []), args.description)
        print(f"Created '{name}' from profile '{args.from_profile}'")

    else:
        (pdir / "settings.json").write_text("{}\n")
        save_manifest(pdir, ["settings.json"], args.description or "blank profile")
        print(f"Created blank profile '{name}'")

    # Record lineage for tree visualization
    if created_from:
        manifest = load_manifest(pdir)
        manifest["created_from"] = created_from
        (pdir / MANIFEST_NAME).write_text(json.dumps(manifest, indent=2) + "\n")

    # Link default shared files (permissions, MCP) unless opted out
    if not args.no_shared_defaults:
        source = pdir if args.from_current or args.from_profile else None
        link_defaults_into_profile(pdir, from_source=source)

    record_usage(name, "create")


def detect_unmanaged(current_profile: str) -> list[str]:
    """Find files in ~/.claude/ not managed by the current profile and not shared."""
    current_dir = profile_dir(current_profile)
    managed = set(get_managed_paths(current_dir))
    unmanaged = []
    if not CLAUDE_DIR.exists():
        return unmanaged
    for item in sorted(CLAUDE_DIR.iterdir()):
        name = item.name
        if name in SHARED_PATHS:
            continue
        if name.startswith(".hop-") or name.endswith(BACKUP_SUFFIX):
            continue
        if name.startswith(".ccswap"):
            continue
        if name in managed:
            continue
        # Skip symlinks pointing into shared dir
        if item.is_symlink():
            try:
                if str(item.resolve()).startswith(str(SHARED_DIR.resolve())):
                    continue
            except OSError:
                pass
        unmanaged.append(name)
    return unmanaged


def adopt_unmanaged(profile_name: str, paths: list[str]):
    """Move unmanaged files from ~/.claude/ into the given profile."""
    pdir = profile_dir(profile_name)
    managed = get_managed_paths(pdir)
    shared_paths = get_shared_paths(pdir)

    for p in paths:
        src = CLAUDE_DIR / p
        dst = pdir / p
        if src.is_symlink():
            # Preserve symlink as-is
            if dst.exists() or dst.is_symlink():
                if dst.is_dir() and not dst.is_symlink():
                    shutil.rmtree(dst)
                else:
                    dst.unlink()
            os.symlink(os.readlink(src), dst)
        elif src.is_dir():
            if dst.exists():
                shutil.rmtree(dst)
            shutil.copytree(src, dst, symlinks=True, copy_function=shutil.copy2)
        elif src.is_file():
            shutil.copy2(src, dst)
        if p not in managed:
            managed.append(p)

    save_manifest(pdir, managed, shared_paths=shared_paths)


def _do_switch(name: str, force: bool = False, dry_run: bool = False, adopt: bool = False):
    """Core switch logic. Validates everything before touching any files."""
    pdir = require_profile(name)

    config = load_config()
    current = config.get("active")

    if current == name and not force:
        print(f"Already on '{name}'. Use --force to re-link.")
        return

    actions = validate_switch_preflight(name, force)

    if dry_run:
        print(f"Dry run: switch to '{name}'")
        for action, detail in actions:
            print(f"  [{action}] {detail}")
        return

    CLAUDE_DIR.mkdir(parents=True, exist_ok=True)

    # Detect unmanaged files before switching away
    if current:
        current_dir = profile_dir(current)
        if current_dir.exists():
            unmanaged = detect_unmanaged(current)
            if unmanaged:
                print(f"Unmanaged files detected in ~/.claude/ (not tracked by '{current}'):")
                for p in unmanaged:
                    item = CLAUDE_DIR / p
                    kind = "dir" if item.is_dir() and not item.is_symlink() else "file"
                    print(f"  {p} ({kind})")
                print()
                if adopt:
                    answer = "y"
                else:
                    answer = _prompt(f"Adopt these into '{current}' before switching? [Y/n/skip] ")
                if answer in ("", "y", "yes"):
                    adopt_unmanaged(current, unmanaged)
                    print(f"Adopted {len(unmanaged)} paths into '{current}'")
                elif answer in ("skip", "s"):
                    print("Skipping — files will remain in ~/.claude/")
                else:
                    print("Skipping — files will remain in ~/.claude/")

            for p in get_managed_paths(current_dir):
                link = CLAUDE_DIR / p
                if link.is_symlink():
                    link.unlink()
    else:
        for p in get_managed_paths(pdir):
            existing = CLAUDE_DIR / p
            if existing.exists() and not existing.is_symlink():
                bp = backup_path(existing)
                print(f"  Backing up {p} -> {bp.name}")
                existing.rename(bp)

    managed = get_managed_paths(pdir)
    linked = 0
    for p in managed:
        if link_managed_path(pdir, p):
            linked += 1

    config["active"] = name
    save_config(config)
    print(f"Switched to '{name}' ({linked} paths linked)")
    record_usage(name, "switch")


def cmd_switch(args):
    _do_switch(args.name, force=args.force, dry_run=args.dry_run, adopt=args.adopt)


def cmd_pick(args):
    """Cherry-pick files from one profile into another (copies)."""
    src_name = args.source
    src_dir = require_profile(src_name)

    config = load_config()
    target_name = args.target or config.get("active")
    if not target_name:
        die("no target specified and no active profile")

    tgt_dir = require_profile(target_name)

    valid_paths = []
    for path in args.paths:
        src = src_dir / path
        if not src.exists() and not src.is_symlink():
            print(f"  skip: '{path}' not found in '{src_name}'")
        else:
            valid_paths.append(path)

    if not valid_paths:
        die("no valid paths to copy")

    if args.dry_run:
        print(f"Dry run: pick from '{src_name}' to '{target_name}'")
        for path in valid_paths:
            dst = tgt_dir / path
            exists = "overwrite" if dst.exists() else "new"
            print(f"  [copy:{exists}] {path}")
        return

    copied = []
    for path in valid_paths:
        src = src_dir / path
        dst = tgt_dir / path

        dst.parent.mkdir(parents=True, exist_ok=True)
        if dst.exists() or dst.is_symlink():
            if dst.is_dir() and not dst.is_symlink():
                shutil.rmtree(dst)
            else:
                dst.unlink()

        if src.is_symlink():
            os.symlink(os.readlink(src), dst)
        elif src.is_dir():
            shutil.copytree(src, dst, symlinks=True, ignore_dangling_symlinks=True)
        else:
            shutil.copy2(src, dst)
        copied.append(path)
        print(f"  copied: {path}  ({src_name} -> {target_name})")

    if copied:
        managed = get_managed_paths(tgt_dir)
        for p in copied:
            if p not in managed:
                managed.append(p)
        save_manifest(tgt_dir, managed)

        if target_name == config.get("active"):
            _do_switch(target_name, force=True)

        record_usage(args.source, "pick")


def cmd_share(args):
    """Share files from one profile into another via symlinks."""
    src_name = args.source
    src_dir = require_profile(src_name)

    config = load_config()
    target_name = args.target or config.get("active")
    if not target_name:
        die("no target specified and no active profile")
    if target_name == src_name:
        die("cannot share a profile with itself")

    tgt_dir = require_profile(target_name)

    valid_paths = []
    for path in args.paths:
        src = src_dir / path
        if not src.exists() and not src.is_symlink():
            print(f"  skip: '{path}' not found in '{src_name}'")
        else:
            valid_paths.append(path)

    if not valid_paths:
        die("no valid paths to share")

    if args.dry_run:
        print(f"Dry run: share from '{src_name}' to '{target_name}'")
        for path in valid_paths:
            dst = tgt_dir / path
            src = src_dir / path
            exists = "replace" if (dst.exists() or dst.is_symlink()) else "new"
            print(f"  [link:{exists}] {path} -> {src}")
        return

    shared = []
    shared_paths = get_shared_paths(tgt_dir)

    for path in valid_paths:
        src = src_dir / path
        dst = tgt_dir / path

        dst.parent.mkdir(parents=True, exist_ok=True)

        # Atomic replacement: write symlink to temp, then rename over dst
        target = src.resolve() if not src.is_symlink() else src
        atomic_symlink(target, dst)
        shared_paths[path] = src_name
        shared.append(path)
        print(f"  shared: {path}  ({src_name} -> {target_name})")

    if shared:
        managed = get_managed_paths(tgt_dir)
        for p in shared:
            if p not in managed:
                managed.append(p)
        save_manifest(tgt_dir, managed, shared_paths=shared_paths)

        if target_name == config.get("active"):
            _do_switch(target_name, force=True)

        record_usage(args.source, "share")


def cmd_unshare(args):
    """Convert shared (symlinked) files back to independent copies."""
    config = load_config()
    target_name = args.profile or config.get("active")
    if not target_name:
        die("no profile specified and no active profile")

    tgt_dir = require_profile(target_name)
    shared_paths = get_shared_paths(tgt_dir)

    if not shared_paths:
        print(f"No shared paths in profile '{target_name}'.")
        return

    paths_to_unshare = args.paths if args.paths else list(shared_paths.keys())

    if args.dry_run:
        print(f"Dry run: unshare in '{target_name}'")
        for path in paths_to_unshare:
            if path in shared_paths:
                print(f"  [materialize] {path} (shared from '{shared_paths[path]}')")
        return

    materialized = []
    for path in paths_to_unshare:
        if path not in shared_paths:
            print(f"  skip: '{path}' is not shared")
            continue

        dst = tgt_dir / path
        if dst.is_symlink():
            target = dst.resolve()
            dst.unlink()
            if target.is_dir():
                shutil.copytree(target, dst, symlinks=True, ignore_dangling_symlinks=True)
            elif target.exists():
                shutil.copy2(target, dst)
            else:
                print(f"  warning: shared target for '{path}' no longer exists")
                continue
        del shared_paths[path]
        materialized.append(path)
        print(f"  materialized: {path}")

    if materialized:
        save_manifest(tgt_dir, get_managed_paths(tgt_dir), shared_paths=shared_paths)

        if target_name == config.get("active"):
            _do_switch(target_name, force=True)


def cmd_diff(args):
    """Compare two profiles."""
    a_dir = require_profile(args.profile_a)
    b_dir = require_profile(args.profile_b)

    a_paths = set(get_managed_paths(a_dir))
    b_paths = set(get_managed_paths(b_dir))

    only_a = a_paths - b_paths
    only_b = b_paths - a_paths
    common = a_paths & b_paths

    if only_a:
        print(f"Only in '{args.profile_a}':")
        for p in sorted(only_a):
            print(f"  {p}")

    if only_b:
        print(f"Only in '{args.profile_b}':")
        for p in sorted(only_b):
            print(f"  {p}")

    if common:
        print(f"Shared:")
        for p in sorted(common):
            af, bf = a_dir / p, b_dir / p
            if af.is_file() and bf.is_file():
                same = af.read_bytes() == bf.read_bytes()
                print(f"  {p}  [{'identical' if same else 'different'}]")
            elif af.is_dir() and bf.is_dir():
                a_items = {str(x.relative_to(af)) for x in af.rglob("*") if x.is_file()}
                b_items = {str(x.relative_to(bf)) for x in bf.rglob("*") if x.is_file()}
                added = len(b_items - a_items)
                removed = len(a_items - b_items)
                shared = len(a_items & b_items)
                print(f"  {p}/  [{shared} shared, +{added}, -{removed}]")
            else:
                print(f"  {p}  [type mismatch]")

    if not only_a and not only_b and not common:
        print("Both profiles are empty.")


def cmd_delete(args):
    config = load_config()
    if args.name == config.get("active"):
        die(f"cannot delete active profile '{args.name}'. Switch first.")

    pdir = require_profile(args.name)

    if PROFILES_DIR.exists():
        dependents = []
        for d in PROFILES_DIR.iterdir():
            if d.is_dir() and d.name != args.name:
                shared = get_shared_paths(d)
                refs = [p for p, src in shared.items() if src == args.name]
                if refs:
                    dependents.append((d.name, refs))
        if dependents:
            print(f"Warning: other profiles share files from '{args.name}':", file=sys.stderr)
            for dep_name, refs in dependents:
                print(f"  {dep_name}: {', '.join(refs)}", file=sys.stderr)
            if not args.yes:
                resp = input("Delete anyway? Shared links will break. [y/N] ")
                if resp.lower() != "y":
                    print("Cancelled.")
                    return

    if not args.yes:
        resp = input(f"Delete profile '{args.name}'? [y/N] ")
        if resp.lower() != "y":
            print("Cancelled.")
            return

    shutil.rmtree(pdir)
    print(f"Deleted profile '{args.name}'")
    record_usage(args.name, "delete")


def cmd_unmanage(args):
    """Stop managing — replace symlinks with real files."""
    config = load_config()
    active = config.get("active")
    if not active:
        print("No active profile to unmanage.")
        return

    pdir = profile_dir(active)
    managed = get_managed_paths(pdir)

    if args.dry_run:
        print(f"Dry run: unmanage profile '{active}'")
        for p in managed:
            link = CLAUDE_DIR / p
            if link.is_symlink():
                print(f"  [materialize] {p}")
        return

    for p in managed:
        link = CLAUDE_DIR / p
        if link.is_symlink():
            target = link.resolve()
            link.unlink()
            if target.is_dir():
                shutil.copytree(target, link, symlinks=True, ignore_dangling_symlinks=True)
            elif target.exists():
                shutil.copy2(target, link)
            print(f"  Materialized {p}")

    config["active"] = None
    save_config(config)
    print(f"Unmanaged. ~/.claude/ now has real files (was profile '{active}').")


def cmd_path(args):
    """Print the path to a profile directory."""
    pdir = require_profile(args.name)
    print(pdir)


def _relative_time(ts: str) -> str:
    """Return a human-readable relative time string for an ISO timestamp."""
    try:
        then = datetime.datetime.fromisoformat(ts)
        now = datetime.datetime.now()
        delta = now - then
        seconds = int(delta.total_seconds())
        if seconds < 3600:
            return f"{max(1, seconds // 60)}m ago"
        elif seconds < 86400:
            return f"{seconds // 3600}h ago"
        elif seconds < 7 * 86400:
            return f"{seconds // 86400}d ago"
        else:
            return f"{seconds // (7 * 86400)}w ago"
    except Exception:
        return ts


def cmd_stats(args):
    """Show profile usage statistics."""
    usage_file = HOPPER_DIR / USAGE_FILE_NAME
    if not usage_file.exists():
        print("No usage data yet.")
        return

    lines = usage_file.read_text().splitlines()
    entries = []
    for line in lines:
        line = line.strip()
        if not line:
            continue
        try:
            entries.append(json.loads(line))
        except json.JSONDecodeError:
            continue

    # Apply --since filter
    if args.since:
        raw = args.since
        if "T" not in raw:
            raw += "T00:00:00"
        try:
            since_dt = datetime.datetime.fromisoformat(raw)
        except ValueError:
            die(f"invalid --since date: {args.since!r} (expected YYYY-MM-DD or ISO format)")
        entries = [
            e for e in entries
            if "timestamp" in e
            and datetime.datetime.fromisoformat(e["timestamp"]) >= since_dt
        ]

    # Apply --profile filter
    if args.profile:
        entries = [e for e in entries if e["profile"] == args.profile]

    if not entries:
        print("No usage data yet.")
        return

    # Aggregate per profile
    profile_data: dict[str, dict] = {}
    for entry in entries:
        pname = entry["profile"]
        action = entry["action"]
        ts = entry["timestamp"]
        if pname not in profile_data:
            profile_data[pname] = {"switches": 0, "last_used": ts, "actions": {}}
        pd = profile_data[pname]
        pd["actions"][action] = pd["actions"].get(action, 0) + 1
        if action == "switch":
            pd["switches"] += 1
        if ts > pd["last_used"]:
            pd["last_used"] = ts

    total_switches = sum(pd["switches"] for pd in profile_data.values())

    if getattr(args, "json", False):
        result = {
            "profiles": [
                {
                    "name": name,
                    "switches": pd["switches"],
                    "last_used": pd["last_used"],
                    "actions": pd["actions"],
                }
                for name, pd in sorted(profile_data.items(), key=lambda x: -x[1]["switches"])
            ],
            "total": total_switches,
        }
        print(json.dumps(result))
        return

    # Human-readable output
    print("Profile usage (all time):")
    max_len = max(len(name) for name in profile_data)
    for name, pd in sorted(profile_data.items(), key=lambda x: -x[1]["switches"]):
        last = _relative_time(pd["last_used"])
        switches = pd["switches"]
        print(f"  {name:<{max_len}}  {switches:>3} switches  (last: {last})")

    print(f"\nTotal: {total_switches} switches across {len(profile_data)} profiles")


def cmd_tree(args):
    """Show all profiles in a tree structure."""
    use_json = getattr(args, "json", False)

    config = load_config()
    active = config.get("active")

    if not PROFILES_DIR.exists():
        if use_json:
            print(json.dumps({"profiles": []}, indent=2))
        else:
            print("No profiles. Run 'claudehopper create <name>' to create one.")
        return

    profile_names = sorted(d.name for d in PROFILES_DIR.iterdir() if d.is_dir())
    if not profile_names:
        if use_json:
            print(json.dumps({"profiles": []}, indent=2))
        else:
            print("No profiles. Run 'claudehopper create <name>' to create one.")
        return

    # Load all profile data
    profiles_data = {}
    for name in profile_names:
        pdir = profile_dir(name)
        manifest = load_manifest(pdir)
        profiles_data[name] = {
            "name": name,
            "active": name == active,
            "managed_paths": manifest.get("managed_paths", []),
            "shared_paths": manifest.get("shared_paths", {}),
            "created_from": manifest.get("created_from"),
            "description": manifest.get("description", ""),
        }

    if use_json:
        print(json.dumps({"profiles": list(profiles_data.values())}, indent=2))
        return

    # Build lineage tree: map parent -> list of children
    # Only treat created_from as parent if it names a known profile
    children_of = {name: [] for name in profile_names}
    root_profiles = []
    for name in profile_names:
        parent = profiles_data[name]["created_from"]
        if parent and parent in profiles_data:
            children_of[parent].append(name)
        else:
            root_profiles.append(name)

    # Collect all shared links across profiles for summary section
    all_shared_links = []
    for name in profile_names:
        for path, src in profiles_data[name]["shared_paths"].items():
            all_shared_links.append((name, path, src))

    print("claudehopper profiles")

    visited = set()

    def render_profile(name, prefix, connector):
        """Render a profile and its children recursively."""
        if name in visited:
            return
        visited.add(name)
        data = profiles_data[name]
        active_marker = " (active)" if data["active"] else ""
        print(f"{prefix}{connector}{name}{active_marker}")

        child_prefix = prefix + ("│   " if connector == "├── " else "    ")
        managed = sorted(data["managed_paths"])
        shared = data["shared_paths"]
        children = children_of[name]

        for i, path in enumerate(managed):
            is_last_item = (i == len(managed) - 1) and not children
            item_connector = "└── " if is_last_item else "├── "
            shared_note = f" (shared from {shared[path]})" if path in shared else ""
            print(f"{child_prefix}{item_connector}{path}{shared_note}")

        for i, child in enumerate(children):
            is_last = i == len(children) - 1
            child_connector = "└── " if is_last else "├── "
            render_profile(child, child_prefix, child_connector)

    for i, name in enumerate(root_profiles):
        is_last = i == len(root_profiles) - 1
        connector = "└── " if is_last else "├── "
        render_profile(name, "", connector)

    if all_shared_links:
        print()
        print("Shared links:")
        for profile_name, path, src in all_shared_links:
            print(f"  {profile_name}/{path} → {src}/{path}")


def main():
    fmt = argparse.RawDescriptionHelpFormatter
    parser = argparse.ArgumentParser(
        prog="claudehopper",
        description="Switch between Claude Code configuration profiles.\n\n"
                    "Profiles are stored in ~/.config/claudehopper/profiles/<name>/.\n"
                    "Profile-specific files in ~/.claude/ are symlinked to the active profile.\n"
                    "Shared files (credentials, history, projects) are never touched.",
        formatter_class=fmt,
    )
    parser.add_argument(
        "--version", action="version",
        version=f"%(prog)s {__import__('claudehopper').__version__}"
    )
    sub = parser.add_subparsers(dest="command")

    # status
    sub.add_parser("status", help="Show current profile status",
                   formatter_class=fmt,
                   description="Show which profile is active and the link status of each managed path.\n"
                               "If no profile is active, shows profile-specific items in ~/.claude/.",
                   epilog="Examples:\n"
                          "  hop status")

    # list
    sub.add_parser("list", aliases=["ls"], help="List all profiles",
                   formatter_class=fmt,
                   description="List all profiles with their managed path counts, shared paths, and descriptions.",
                   epilog="Examples:\n"
                          "  hop list\n"
                          "  hop ls")

    # create
    p_create = sub.add_parser("create", help="Create a new profile",
                              formatter_class=fmt,
                              description="Create a new profile. By default creates a blank profile with just\n"
                                          "settings.json. Use --from-current to capture your current ~/.claude/\n"
                                          "setup, or --from-profile to clone an existing profile.",
                              epilog="Examples:\n"
                                     "  hop create work --from-current -d 'Work setup'\n"
                                     "  hop create personal --from-profile work -d 'Personal'\n"
                                     "  hop create vanilla -d 'Clean Claude Code'\n"
                                     "  hop create omc --from-current --activate")
    p_create.add_argument("name", help="Profile name")
    p_create.add_argument("--from-current", action="store_true",
                          help="Import profile-specific files from current ~/.claude/")
    p_create.add_argument("--from-profile", metavar="PROFILE",
                          help="Clone all files from an existing profile")
    p_create.add_argument("--description", "-d", help="Short description for this profile")
    p_create.add_argument("--activate", action="store_true",
                          help="Switch to the new profile immediately after creating it")
    p_create.add_argument("--no-shared-defaults", action="store_true",
                          help="Don't link shared files (permissions, MCP) into this profile")

    # switch
    p_switch = sub.add_parser("switch", aliases=["sw"], help="Switch active profile",
                              formatter_class=fmt,
                              description="Switch to a different profile. Removes symlinks for the current\n"
                                          "profile and creates new symlinks pointing to the target profile.\n"
                                          "Credentials, history, and other shared files are never touched.",
                              epilog="Examples:\n"
                                     "  hop switch work\n"
                                     "  hop sw personal\n"
                                     "  hop switch omc --dry-run\n"
                                     "  hop switch work --force")
    p_switch.add_argument("name", help="Profile to activate")
    p_switch.add_argument("--force", "-f", action="store_true",
                          help="Force switch, backing up any conflicting unmanaged files")
    p_switch.add_argument("--dry-run", "-n", action="store_true",
                          help="Show what would change without touching anything")
    p_switch.add_argument("--adopt", "-a", action="store_true",
                          help="Automatically adopt unmanaged files into the current profile before switching")

    # pick
    p_pick = sub.add_parser("pick", help="Copy files from one profile into another",
                            formatter_class=fmt,
                            description="Copy specific files from one profile into another as independent\n"
                                        "copies. Unlike 'share', changes to the copy won't affect the original.\n"
                                        "If no --target is given, copies into the active profile.",
                            epilog="Examples:\n"
                                   "  hop pick work CLAUDE.md settings.json\n"
                                   "  hop pick work commands/ --target personal\n"
                                   "  hop pick work CLAUDE.md --dry-run")
    p_pick.add_argument("source", help="Source profile to copy from")
    p_pick.add_argument("paths", nargs="+", help="File or directory names to copy")
    p_pick.add_argument("--target", "-t", help="Target profile (default: active profile)")
    p_pick.add_argument("--dry-run", "-n", action="store_true", help="Show what would happen")

    # share
    p_share = sub.add_parser("share", help="Share files between profiles via symlinks",
                             formatter_class=fmt,
                             description="Share files from one profile into another using symlinks.\n"
                                         "Both profiles will point to the same underlying file, so edits\n"
                                         "in one are immediately visible in the other. The source profile\n"
                                         "owns the real file.",
                             epilog="Examples:\n"
                                    "  hop share work commands/ --target personal\n"
                                    "  hop share omc .mcp.json\n"
                                    "  hop share work settings.json --dry-run")
    p_share.add_argument("source", help="Source profile that owns the files")
    p_share.add_argument("paths", nargs="+", help="File or directory names to share")
    p_share.add_argument("--target", "-t", help="Target profile (default: active profile)")
    p_share.add_argument("--dry-run", "-n", action="store_true", help="Show what would happen")

    # unshare
    p_unshare = sub.add_parser("unshare", help="Convert shared files back to independent copies",
                               formatter_class=fmt,
                               description="Materialize shared (symlinked) files back into independent copies.\n"
                                           "After unsharing, changes in one profile won't affect the other.\n"
                                           "If no paths are specified, unshares all shared files.",
                               epilog="Examples:\n"
                                      "  hop unshare commands/\n"
                                      "  hop unshare                    # unshare everything\n"
                                      "  hop unshare -p work settings.json")
    p_unshare.add_argument("paths", nargs="*", help="Paths to unshare (default: all shared paths)")
    p_unshare.add_argument("--profile", "-p", help="Profile to unshare in (default: active)")
    p_unshare.add_argument("--dry-run", "-n", action="store_true", help="Show what would happen")

    # diff
    p_diff = sub.add_parser("diff", help="Compare two profiles",
                            formatter_class=fmt,
                            description="Compare the contents of two profiles side by side.\n"
                                        "Shows files unique to each profile and whether shared files\n"
                                        "are identical or different.",
                            epilog="Examples:\n"
                                   "  hop diff work personal\n"
                                   "  hop diff omc vanilla")
    p_diff.add_argument("profile_a", help="First profile to compare")
    p_diff.add_argument("profile_b", help="Second profile to compare")

    # delete
    p_del = sub.add_parser("delete", aliases=["rm"], help="Delete a profile",
                           formatter_class=fmt,
                           description="Delete a profile and all its files. Cannot delete the active\n"
                                       "profile — switch to a different one first. Warns if other\n"
                                       "profiles share files from the one being deleted.",
                           epilog="Examples:\n"
                                  "  hop delete old-profile\n"
                                  "  hop rm temp --yes")
    p_del.add_argument("name", help="Profile to delete")
    p_del.add_argument("--yes", "-y", action="store_true", help="Skip confirmation prompt")

    # unmanage
    p_unmanage = sub.add_parser("unmanage", help="Stop managing ~/.claude/, restore real files",
                                formatter_class=fmt,
                                description="Stop using claudehopper for the active profile. Replaces all\n"
                                            "symlinks in ~/.claude/ with real copies of the files, so your\n"
                                            "config works without claudehopper installed. Does not delete\n"
                                            "the profile — you can re-activate it later.",
                                epilog="Examples:\n"
                                       "  hop unmanage\n"
                                       "  hop unmanage --dry-run")
    p_unmanage.add_argument("--dry-run", "-n", action="store_true", help="Show what would happen")

    # tree
    p_tree = sub.add_parser("tree", help="Visualize profile relationships",
                            formatter_class=fmt,
                            description="Show all profiles as a visual tree with their managed files,\n"
                                        "shared file relationships, and profile lineage (which profiles\n"
                                        "were cloned from others).",
                            epilog="Examples:\n"
                                   "  hop tree\n"
                                   "  hop tree --json")
    p_tree.add_argument("--json", action="store_true", help="Output as JSON instead of a tree")

    # path
    p_path = sub.add_parser("path", help="Print a profile's directory path",
                            formatter_class=fmt,
                            description="Print the full filesystem path to a profile's directory.\n"
                                        "Useful for scripting or opening the profile in a file manager.",
                            epilog="Examples:\n"
                                   "  hop path work\n"
                                   "  # Output: /home/you/.config/claudehopper/profiles/work\n\n"
                                   "  # Open in file manager:\n"
                                   "  xdg-open $(hop path work)\n\n"
                                   "  # Edit a profile's CLAUDE.md directly:\n"
                                   "  vim $(hop path work)/CLAUDE.md")
    p_path.add_argument("name", help="Profile name")

    # stats
    p_stats = sub.add_parser("stats", help="Show profile usage statistics",
                             formatter_class=fmt,
                             description="Show how often each profile has been used, with switch counts\n"
                                         "and last-used times. Usage is recorded automatically on switch,\n"
                                         "create, delete, pick, and share actions.",
                             epilog="Examples:\n"
                                    "  hop stats\n"
                                    "  hop stats --profile work\n"
                                    "  hop stats --since 2025-01-01\n"
                                    "  hop stats --json")
    p_stats.add_argument("--profile", "-p", help="Filter to a specific profile")
    p_stats.add_argument("--since", help="Only show usage after this date (YYYY-MM-DD)")
    p_stats.add_argument("--json", action="store_true", help="Output as JSON")

    # update
    p_update = sub.add_parser("update", help="Check for and install updates",
                              formatter_class=fmt,
                              description="Check GitHub for a newer release and optionally install it.\n"
                                          "Uses 'uv tool install --reinstall' to update in place.",
                              epilog="Examples:\n"
                                     "  hop update              # install latest from GitHub\n"
                                     "  hop update --check      # just check, don't install")
    p_update.add_argument("--check", "-c", action="store_true",
                          help="Only check for updates, don't install")

    args = parser.parse_args()

    commands = {
        None: cmd_status,
        "status": cmd_status,
        "list": cmd_list,
        "ls": cmd_list,
        "create": cmd_create,
        "switch": cmd_switch,
        "sw": cmd_switch,
        "pick": cmd_pick,
        "share": cmd_share,
        "unshare": cmd_unshare,
        "diff": cmd_diff,
        "delete": cmd_delete,
        "rm": cmd_delete,
        "unmanage": cmd_unmanage,
        "path": cmd_path,
        "tree": cmd_tree,
        "stats": cmd_stats,
        "update": cmd_update,
    }

    fn = commands.get(args.command, cmd_status)
    fn(args)


if __name__ == "__main__":
    main()
