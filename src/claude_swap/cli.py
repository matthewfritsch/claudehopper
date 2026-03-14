#!/usr/bin/env python3
"""ccswap - Switch between Claude Code configuration profiles.

Profiles are stored in ~/.claude-swap/profiles/<name>/.
Profile-specific files in ~/.claude/ are symlinked to the active profile.
Shared files (credentials, history, projects, cache) are never touched.
"""

import argparse
import json
import os
import shutil
import sys
from pathlib import Path

CLAUDE_DIR = Path.home() / ".claude"
SWAP_DIR = Path.home() / ".claude-swap"
PROFILES_DIR = SWAP_DIR / "profiles"
CONFIG_FILE = SWAP_DIR / "config.json"

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


def load_config() -> dict:
    if CONFIG_FILE.exists():
        return json.loads(CONFIG_FILE.read_text())
    return {"active": None}


def save_config(config: dict):
    SWAP_DIR.mkdir(parents=True, exist_ok=True)
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
    manifest = pdir / ".ccswap-manifest.json"
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
        if item.name != ".ccswap-manifest.json"
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
    (pdir / ".ccswap-manifest.json").write_text(json.dumps(data, indent=2) + "\n")


def detect_profile_paths() -> list[str]:
    """Find profile-specific paths in current ~/.claude/."""
    if not CLAUDE_DIR.exists():
        return []
    return sorted(
        item.name for item in CLAUDE_DIR.iterdir()
        if item.name not in SHARED_PATHS and not item.name.startswith(".ccswap")
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
    """Validate everything before a switch. Returns list of (action, detail) pairs.

    Raises SystemExit on fatal validation errors (unless force=True for recoverable ones).
    """
    pdir = require_profile(name)
    managed = get_managed_paths(pdir)
    actions = []
    errors = []

    config = load_config()
    current = config.get("active")

    # Validate: all sources exist in the target profile
    for p in managed:
        src = pdir / p
        if not src.exists() and not src.is_symlink():
            errors.append(f"  '{p}' listed in manifest but missing from profile dir")

    if errors:
        print("Validation errors in target profile:", file=sys.stderr)
        for e in errors:
            print(e, file=sys.stderr)
        die("fix the profile before switching")

    # Check what needs to happen in ~/.claude/
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
                        f"Run 'ccswap create <name> --from-current' first, or use --force.")
                actions.append(("backup", p))

    for p in managed:
        src = pdir / p
        if src.exists() or src.is_symlink():
            actions.append(("link", f"{p} -> {src.resolve() if not src.is_symlink() else os.readlink(src)}"))

    return actions


def backup_path(path: Path) -> Path:
    """Generate a backup path that doesn't collide."""
    backup = path.parent / (path.name + ".ccswap-backup")
    n = 1
    while backup.exists():
        backup = path.parent / (path.name + f".ccswap-backup.{n}")
        n += 1
    return backup


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
            print("\nRun 'ccswap create <name> --from-current' to import as a profile.")
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


def cmd_list(_args):
    config = load_config()
    active = config.get("active")

    if not PROFILES_DIR.exists():
        print("No profiles. Run 'ccswap create <name>' to create one.")
        return

    profiles = sorted(d.name for d in PROFILES_DIR.iterdir() if d.is_dir())
    if not profiles:
        print("No profiles. Run 'ccswap create <name>' to create one.")
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

    if args.from_current:
        paths = detect_profile_paths()

        # Pre-flight: validate all sources are readable
        errors = []
        for p in paths:
            src = CLAUDE_DIR / p
            if not src.exists() and not src.is_symlink():
                errors.append(f"  '{p}' detected but doesn't exist (race condition?)")
        if errors:
            shutil.rmtree(pdir)  # clean up the dir we just created
            print("Validation failed:", file=sys.stderr)
            for e in errors:
                print(e, file=sys.stderr)
            die("aborting profile creation")

        # All sources validated — now copy
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
        save_manifest(pdir, paths, args.description or "")
        print(f"Created '{name}' from current ~/.claude/ ({len(paths)} paths)")

        if args.activate:
            _do_switch(name, force=True)

    elif args.from_profile:
        src_dir = require_profile(args.from_profile)
        shutil.copytree(src_dir, pdir, dirs_exist_ok=True, symlinks=True,
                        ignore_dangling_symlinks=True)
        manifest = load_manifest(pdir)
        if args.description:
            save_manifest(pdir, manifest.get("managed_paths", []), args.description)
        print(f"Created '{name}' from profile '{args.from_profile}'")

    else:
        (pdir / "settings.json").write_text("{}\n")
        save_manifest(pdir, ["settings.json"], args.description or "blank profile")
        print(f"Created blank profile '{name}'")


def _do_switch(name: str, force: bool = False, dry_run: bool = False):
    """Core switch logic. Validates everything before touching any files."""
    pdir = require_profile(name)

    config = load_config()
    current = config.get("active")

    if current == name and not force:
        print(f"Already on '{name}'. Use --force to re-link.")
        return

    # ── Pre-flight validation ──
    actions = validate_switch_preflight(name, force)

    if dry_run:
        print(f"Dry run: switch to '{name}'")
        for action, detail in actions:
            print(f"  [{action}] {detail}")
        return

    # ── Execute (all validation passed) ──
    CLAUDE_DIR.mkdir(parents=True, exist_ok=True)

    # Unlink current profile
    if current:
        current_dir = profile_dir(current)
        if current_dir.exists():
            for p in get_managed_paths(current_dir):
                link = CLAUDE_DIR / p
                if link.is_symlink():
                    link.unlink()
    else:
        # First switch — back up conflicting real files
        for p in get_managed_paths(pdir):
            existing = CLAUDE_DIR / p
            if existing.exists() and not existing.is_symlink():
                bp = backup_path(existing)
                print(f"  Backing up {p} -> {bp.name}")
                existing.rename(bp)

    # Link new profile
    managed = get_managed_paths(pdir)
    linked = 0
    for p in managed:
        src = pdir / p
        link = CLAUDE_DIR / p
        if link.is_symlink():
            link.unlink()
        if src.exists() or src.is_symlink():
            link.symlink_to(src.resolve() if not src.is_symlink() else src)
            linked += 1

    config["active"] = name
    save_config(config)
    print(f"Switched to '{name}' ({linked} paths linked)")


def cmd_switch(args):
    _do_switch(args.name, force=args.force, dry_run=args.dry_run)


def cmd_pick(args):
    """Cherry-pick files from one profile into another (copies)."""
    src_name = args.source
    src_dir = require_profile(src_name)

    config = load_config()
    target_name = args.target or config.get("active")
    if not target_name:
        die("no target specified and no active profile")

    tgt_dir = require_profile(target_name)

    # Pre-flight: validate all source paths exist
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

    # Execute copies
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

    # Update manifest
    if copied:
        managed = get_managed_paths(tgt_dir)
        for p in copied:
            if p not in managed:
                managed.append(p)
        save_manifest(tgt_dir, managed)

        # Re-link if target is active
        if target_name == config.get("active"):
            _do_switch(target_name, force=True)


def cmd_share(args):
    """Share files from one profile into another via symlinks.

    Unlike 'pick' (which copies), 'share' creates symlinks so both profiles
    reference the same underlying files. Changes in one are reflected in the other.
    """
    src_name = args.source
    src_dir = require_profile(src_name)

    config = load_config()
    target_name = args.target or config.get("active")
    if not target_name:
        die("no target specified and no active profile")
    if target_name == src_name:
        die("cannot share a profile with itself")

    tgt_dir = require_profile(target_name)

    # Pre-flight: validate all source paths
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

    # Execute shares
    shared = []
    shared_paths = get_shared_paths(tgt_dir)

    for path in valid_paths:
        src = src_dir / path
        dst = tgt_dir / path

        # Remove existing dst
        if dst.exists() or dst.is_symlink():
            if dst.is_dir() and not dst.is_symlink():
                shutil.rmtree(dst)
            else:
                dst.unlink()

        dst.parent.mkdir(parents=True, exist_ok=True)

        # Create symlink to source profile's file
        target = src.resolve() if not src.is_symlink() else src
        dst.symlink_to(target)
        shared_paths[path] = src_name
        shared.append(path)
        print(f"  shared: {path}  ({src_name} -> {target_name})")

    # Update manifest
    if shared:
        managed = get_managed_paths(tgt_dir)
        for p in shared:
            if p not in managed:
                managed.append(p)
        save_manifest(tgt_dir, managed, shared_paths=shared_paths)

        # Re-link if target is active
        if target_name == config.get("active"):
            _do_switch(target_name, force=True)


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

    # Check if other profiles share from this one
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


def main():
    parser = argparse.ArgumentParser(
        prog="ccswap",
        description="Switch between Claude Code configuration profiles",
    )
    parser.add_argument(
        "--version", action="version",
        version=f"%(prog)s {__import__('claude_swap').__version__}"
    )
    sub = parser.add_subparsers(dest="command")

    # status
    sub.add_parser("status", help="Show current profile status")

    # list
    sub.add_parser("list", aliases=["ls"], help="List all profiles")

    # create
    p_create = sub.add_parser("create", help="Create a new profile")
    p_create.add_argument("name", help="Profile name")
    p_create.add_argument("--from-current", action="store_true", help="Import current ~/.claude/")
    p_create.add_argument("--from-profile", metavar="PROFILE", help="Clone existing profile")
    p_create.add_argument("--description", "-d", help="Short description")
    p_create.add_argument("--activate", action="store_true", help="Switch to it after creating")

    # switch
    p_switch = sub.add_parser("switch", aliases=["sw"], help="Switch active profile")
    p_switch.add_argument("name", help="Profile to activate")
    p_switch.add_argument("--force", "-f", action="store_true", help="Force, backing up conflicts")
    p_switch.add_argument("--dry-run", "-n", action="store_true", help="Show what would happen")

    # pick (copy between profiles)
    p_pick = sub.add_parser("pick", help="Copy files from one profile into another")
    p_pick.add_argument("source", help="Source profile")
    p_pick.add_argument("paths", nargs="+", help="Paths to copy (e.g. 'commands/gsd' 'agents')")
    p_pick.add_argument("--target", "-t", help="Target profile (default: active)")
    p_pick.add_argument("--dry-run", "-n", action="store_true", help="Show what would happen")

    # share (symlink between profiles)
    p_share = sub.add_parser("share", help="Share files from one profile into another via symlinks")
    p_share.add_argument("source", help="Source profile (owns the files)")
    p_share.add_argument("paths", nargs="+", help="Paths to share")
    p_share.add_argument("--target", "-t", help="Target profile (default: active)")
    p_share.add_argument("--dry-run", "-n", action="store_true", help="Show what would happen")

    # unshare (materialize shared symlinks)
    p_unshare = sub.add_parser("unshare", help="Convert shared files back to independent copies")
    p_unshare.add_argument("paths", nargs="*", help="Paths to unshare (default: all)")
    p_unshare.add_argument("--profile", "-p", help="Profile to unshare in (default: active)")
    p_unshare.add_argument("--dry-run", "-n", action="store_true", help="Show what would happen")

    # diff
    p_diff = sub.add_parser("diff", help="Compare two profiles")
    p_diff.add_argument("profile_a")
    p_diff.add_argument("profile_b")

    # delete
    p_del = sub.add_parser("delete", aliases=["rm"], help="Delete a profile")
    p_del.add_argument("name")
    p_del.add_argument("--yes", "-y", action="store_true", help="Skip confirmation")

    # unmanage
    p_unmanage = sub.add_parser("unmanage", help="Stop managing, materialize symlinks back to real files")
    p_unmanage.add_argument("--dry-run", "-n", action="store_true", help="Show what would happen")

    # path
    p_path = sub.add_parser("path", help="Print profile directory path")
    p_path.add_argument("name")

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
    }

    fn = commands.get(args.command, cmd_status)
    fn(args)


if __name__ == "__main__":
    main()
