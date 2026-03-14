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


def load_config() -> dict:
    if CONFIG_FILE.exists():
        return json.loads(CONFIG_FILE.read_text())
    return {"active": None}


def save_config(config: dict):
    SWAP_DIR.mkdir(parents=True, exist_ok=True)
    CONFIG_FILE.write_text(json.dumps(config, indent=2) + "\n")


def profile_dir(name: str) -> Path:
    return PROFILES_DIR / name


def load_manifest(pdir: Path) -> dict:
    manifest = pdir / ".ccswap-manifest.json"
    if manifest.exists():
        return json.loads(manifest.read_text())
    return {"managed_paths": [], "description": ""}


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


def save_manifest(pdir: Path, managed_paths: list[str], description: str = ""):
    existing = load_manifest(pdir)
    data = {
        "managed_paths": sorted(set(managed_paths)),
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


def unlink_profile(name: str):
    """Remove symlinks for a profile's managed paths."""
    pdir = profile_dir(name)
    if not pdir.exists():
        return
    for p in get_managed_paths(pdir):
        link = CLAUDE_DIR / p
        if link.is_symlink():
            link.unlink()


def link_profile(name: str):
    """Create symlinks for a profile's managed paths."""
    pdir = profile_dir(name)
    for p in get_managed_paths(pdir):
        src = pdir / p
        link = CLAUDE_DIR / p
        if link.is_symlink():
            link.unlink()
        if src.exists():
            link.symlink_to(src.resolve())


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
    manifest = load_manifest(pdir)

    print(f"Profile:  {active}")
    if manifest.get("description"):
        print(f"Desc:     {manifest['description']}")
    print(f"Location: {pdir}")
    print(f"Managed:  {len(managed)} paths")

    for p in sorted(managed):
        link = CLAUDE_DIR / p
        if is_our_symlink(link, pdir):
            status = "linked"
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
        manifest = load_manifest(pdir)
        desc = f" - {manifest['description']}" if manifest.get("description") else ""
        print(f"  {name}{marker}  [{len(managed)} paths]{desc}")


def cmd_create(args):
    name = args.name
    pdir = profile_dir(name)

    if pdir.exists():
        print(f"Profile '{name}' already exists.", file=sys.stderr)
        sys.exit(1)

    pdir.mkdir(parents=True)

    if args.from_current:
        paths = detect_profile_paths()
        for p in paths:
            src = CLAUDE_DIR / p
            dst = pdir / p
            # Preserve symlinks within dirs, ignore broken ones
            if src.is_symlink():
                # Top-level symlink: copy the link itself
                os.symlink(os.readlink(src), dst)
            elif src.is_dir():
                def _on_error(fn, path, exc_info):
                    pass  # skip broken symlinks / permission errors
                shutil.copytree(src, dst, symlinks=True, ignore_dangling_symlinks=True,
                                copy_function=shutil.copy2)
            elif src.is_file():
                shutil.copy2(src, dst)
        save_manifest(pdir, paths, args.description or "")
        print(f"Created '{name}' from current ~/.claude/ ({len(paths)} paths)")

        if args.activate:
            _do_switch(name, force=True)

    elif args.from_profile:
        src_dir = profile_dir(args.from_profile)
        if not src_dir.exists():
            print(f"Source profile '{args.from_profile}' not found.", file=sys.stderr)
            sys.exit(1)
        shutil.copytree(src_dir, pdir, dirs_exist_ok=True)
        manifest = load_manifest(pdir)
        if args.description:
            save_manifest(pdir, manifest.get("managed_paths", []), args.description)
        print(f"Created '{name}' from profile '{args.from_profile}'")

    else:
        # Blank profile — minimal settings.json
        (pdir / "settings.json").write_text("{}\n")
        save_manifest(pdir, ["settings.json"], args.description or "blank profile")
        print(f"Created blank profile '{name}'")


def _do_switch(name: str, force: bool = False):
    """Core switch logic used by cmd_switch and other commands."""
    pdir = profile_dir(name)

    if not pdir.exists():
        print(f"Profile '{name}' not found.", file=sys.stderr)
        if PROFILES_DIR.exists():
            available = sorted(d.name for d in PROFILES_DIR.iterdir() if d.is_dir())
            if available:
                print(f"Available: {', '.join(available)}")
        sys.exit(1)

    config = load_config()
    current = config.get("active")

    if current == name and not force:
        print(f"Already on '{name}'. Use --force to re-link.")
        return

    CLAUDE_DIR.mkdir(parents=True, exist_ok=True)

    # Unlink current profile
    if current:
        unlink_profile(current)
    else:
        # First switch — back up any conflicting real files
        for p in get_managed_paths(pdir):
            existing = CLAUDE_DIR / p
            if existing.exists() and not existing.is_symlink():
                if not force:
                    print(f"Unmanaged '{p}' exists. Run 'ccswap create <name> --from-current' first, or use --force.")
                    sys.exit(1)
                bp = backup_path(existing)
                print(f"  Backing up {p} -> {bp.name}")
                existing.rename(bp)

    link_profile(name)

    config["active"] = name
    save_config(config)
    print(f"Switched to '{name}' ({len(get_managed_paths(pdir))} paths linked)")


def cmd_switch(args):
    _do_switch(args.name, force=args.force)


def cmd_pick(args):
    """Cherry-pick files from one profile into another."""
    src_name = args.source
    src_dir = profile_dir(src_name)

    if not src_dir.exists():
        print(f"Source profile '{src_name}' not found.", file=sys.stderr)
        sys.exit(1)

    config = load_config()
    target_name = args.target or config.get("active")
    if not target_name:
        print("No target specified and no active profile.", file=sys.stderr)
        sys.exit(1)

    tgt_dir = profile_dir(target_name)
    if not tgt_dir.exists():
        print(f"Target profile '{target_name}' not found.", file=sys.stderr)
        sys.exit(1)

    copied = []
    for path in args.paths:
        src = src_dir / path
        dst = tgt_dir / path

        if not src.exists():
            print(f"  skip: {path} not found in '{src_name}'")
            continue

        dst.parent.mkdir(parents=True, exist_ok=True)
        if dst.exists():
            if dst.is_dir() and not dst.is_symlink():
                shutil.rmtree(dst)
            else:
                dst.unlink()

        if src.is_dir():
            shutil.copytree(src, dst)
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
            link_profile(target_name)
            print(f"Re-linked active profile '{target_name}'")


def cmd_diff(args):
    """Compare two profiles."""
    a_dir = profile_dir(args.profile_a)
    b_dir = profile_dir(args.profile_b)

    for name, d in [(args.profile_a, a_dir), (args.profile_b, b_dir)]:
        if not d.exists():
            print(f"Profile '{name}' not found.", file=sys.stderr)
            sys.exit(1)

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
        print(f"Cannot delete active profile '{args.name}'. Switch first.", file=sys.stderr)
        sys.exit(1)

    pdir = profile_dir(args.name)
    if not pdir.exists():
        print(f"Profile '{args.name}' not found.", file=sys.stderr)
        sys.exit(1)

    if not args.yes:
        resp = input(f"Delete profile '{args.name}'? [y/N] ")
        if resp.lower() != "y":
            print("Cancelled.")
            return

    shutil.rmtree(pdir)
    print(f"Deleted profile '{args.name}'")


def cmd_unmanage(_args):
    """Stop managing — replace symlinks with real files."""
    config = load_config()
    active = config.get("active")
    if not active:
        print("No active profile to unmanage.")
        return

    pdir = profile_dir(active)
    for p in get_managed_paths(pdir):
        link = CLAUDE_DIR / p
        if link.is_symlink():
            target = link.resolve()
            link.unlink()
            if target.is_dir():
                shutil.copytree(target, link)
            elif target.exists():
                shutil.copy2(target, link)
            print(f"  Materialized {p}")

    config["active"] = None
    save_config(config)
    print(f"Unmanaged. ~/.claude/ now has real files (was profile '{active}').")


def cmd_path(args):
    """Print the path to a profile directory."""
    pdir = profile_dir(args.name)
    if not pdir.exists():
        print(f"Profile '{args.name}' not found.", file=sys.stderr)
        sys.exit(1)
    print(pdir)


def main():
    parser = argparse.ArgumentParser(
        prog="ccswap",
        description="Switch between Claude Code configuration profiles",
    )
    parser.add_argument(
        "--version", action="version", version=f"%(prog)s {__import__('claude_swap').__version__}"
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

    # pick
    p_pick = sub.add_parser("pick", help="Cherry-pick files between profiles")
    p_pick.add_argument("source", help="Source profile")
    p_pick.add_argument("paths", nargs="+", help="Paths to copy (e.g. 'commands/gsd' 'agents')")
    p_pick.add_argument("--target", "-t", help="Target profile (default: active)")

    # diff
    p_diff = sub.add_parser("diff", help="Compare two profiles")
    p_diff.add_argument("profile_a")
    p_diff.add_argument("profile_b")

    # delete
    p_del = sub.add_parser("delete", aliases=["rm"], help="Delete a profile")
    p_del.add_argument("name")
    p_del.add_argument("--yes", "-y", action="store_true", help="Skip confirmation")

    # unmanage
    sub.add_parser("unmanage", help="Stop managing, materialize symlinks back to real files")

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
