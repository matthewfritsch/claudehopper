"""Tests for claudehopper using isolated temp directories."""

import json
import os
import shutil
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from claudehopper import cli


class ClaudeHopperTestCase(unittest.TestCase):
    """Base class that redirects CLAUDE_DIR and HOPPER_DIR to temp dirs."""

    def setUp(self):
        self.tmpdir = Path(tempfile.mkdtemp(prefix="claudehopper-test-"))
        self.claude_dir = self.tmpdir / ".claude"
        self.hopper_dir = self.tmpdir / ".config" / "claudehopper"
        self.profiles_dir = self.hopper_dir / "profiles"
        self.legacy_dir = self.tmpdir / ".claude-swap"
        self.claude_dir.mkdir()

        # Patch module-level paths
        self._patchers = [
            mock.patch.object(cli, "CLAUDE_DIR", self.claude_dir),
            mock.patch.object(cli, "HOPPER_DIR", self.hopper_dir),
            mock.patch.object(cli, "PROFILES_DIR", self.profiles_dir),
            mock.patch.object(cli, "CONFIG_FILE", self.hopper_dir / "config.json"),
            mock.patch.object(cli, "LEGACY_DIR", self.legacy_dir),
        ]
        for p in self._patchers:
            p.start()

    def tearDown(self):
        for p in self._patchers:
            p.stop()
        shutil.rmtree(self.tmpdir)

    def _populate_claude_dir(self):
        """Create a fake ~/.claude/ with profile-specific files."""
        (self.claude_dir / "settings.json").write_text('{"hooks": {}}')
        (self.claude_dir / "CLAUDE.md").write_text("# Test CLAUDE.md")
        (self.claude_dir / "commands").mkdir()
        (self.claude_dir / "commands" / "test.md").write_text("# test command")
        (self.claude_dir / "agents").mkdir()
        # Shared file that should never be touched
        (self.claude_dir / ".credentials.json").write_text('{"secret": true}')
        (self.claude_dir / "projects").mkdir()


class TestStatus(ClaudeHopperTestCase):

    def test_status_unmanaged(self):
        self._populate_claude_dir()
        cli.cmd_status(None)  # should not raise

    def test_status_with_active_profile(self):
        self._populate_claude_dir()
        args = mock.Mock(from_current=True, from_profile=None,
                         description="test profile", activate=False)
        args.name = "test"
        cli.cmd_create(args)
        cli._do_switch("test", force=True)
        cli.cmd_status(None)  # should not raise


class TestValidation(ClaudeHopperTestCase):

    def test_invalid_profile_names(self):
        for bad_name in [".", "..", ".hidden", "-dash", "has/slash", ""]:
            with self.assertRaises(SystemExit, msg=f"should reject: {bad_name}"):
                cli.validate_profile_name(bad_name)

    def test_valid_profile_names(self):
        for good_name in ["omc", "vanilla", "my-profile", "test_123"]:
            cli.validate_profile_name(good_name)  # should not raise

    def test_require_profile_missing(self):
        with self.assertRaises(SystemExit):
            cli.require_profile("nonexistent")


class TestCreateProfile(ClaudeHopperTestCase):

    def test_create_blank(self):
        args = mock.Mock(from_current=False, from_profile=None,
                         description="blank", activate=False)
        args.name = "vanilla"
        cli.cmd_create(args)

        pdir = self.profiles_dir / "vanilla"
        self.assertTrue(pdir.exists())
        self.assertTrue((pdir / "settings.json").exists())
        self.assertEqual(json.loads((pdir / "settings.json").read_text()), {})

    def test_create_from_current(self):
        self._populate_claude_dir()
        args = mock.Mock(from_current=True, from_profile=None,
                         description="from current", activate=False)
        args.name = "omc"
        cli.cmd_create(args)

        pdir = self.profiles_dir / "omc"
        self.assertTrue((pdir / "settings.json").exists())
        self.assertTrue((pdir / "CLAUDE.md").exists())
        self.assertTrue((pdir / "commands" / "test.md").exists())
        # Shared files should NOT be copied
        self.assertFalse((pdir / ".credentials.json").exists())
        self.assertFalse((pdir / "projects").exists())

    def test_create_duplicate_fails(self):
        args = mock.Mock(from_current=False, from_profile=None,
                         description="", activate=False)
        args.name = "test"
        cli.cmd_create(args)
        with self.assertRaises(SystemExit):
            cli.cmd_create(args)

    def test_create_from_profile(self):
        # Create source profile
        args1 = mock.Mock(from_current=False, from_profile=None,
                          description="source", activate=False)
        args1.name = "src"
        cli.cmd_create(args1)

        # Clone it
        args2 = mock.Mock(from_current=False, from_profile="src",
                          description="cloned", activate=False)
        args2.name = "dst"
        cli.cmd_create(args2)

        self.assertTrue((self.profiles_dir / "dst" / "settings.json").exists())

    def test_create_from_profile_records_lineage(self):
        args1 = mock.Mock(from_current=False, from_profile=None,
                          description="source", activate=False)
        args1.name = "src"
        cli.cmd_create(args1)

        args2 = mock.Mock(from_current=False, from_profile="src",
                          description="cloned", activate=False)
        args2.name = "dst"
        cli.cmd_create(args2)

        manifest = cli.load_manifest(self.profiles_dir / "dst")
        self.assertEqual(manifest.get("created_from"), "src")


class TestSwitch(ClaudeHopperTestCase):

    def test_switch_basic(self):
        self._populate_claude_dir()
        args = mock.Mock(from_current=True, from_profile=None,
                         description="", activate=False)
        args.name = "p1"
        cli.cmd_create(args)

        cli._do_switch("p1", force=True)

        # settings.json should now be a symlink
        self.assertTrue((self.claude_dir / "settings.json").is_symlink())
        config = cli.load_config()
        self.assertEqual(config["active"], "p1")

    def test_switch_preserves_shared_files(self):
        self._populate_claude_dir()
        args = mock.Mock(from_current=True, from_profile=None,
                         description="", activate=False)
        args.name = "p1"
        cli.cmd_create(args)
        cli._do_switch("p1", force=True)

        # Credentials should be untouched real file
        self.assertFalse((self.claude_dir / ".credentials.json").is_symlink())
        self.assertEqual(
            json.loads((self.claude_dir / ".credentials.json").read_text()),
            {"secret": True}
        )

    def test_switch_between_profiles(self):
        self._populate_claude_dir()
        # Create two profiles
        for name in ["a", "b"]:
            args = mock.Mock(from_current=False, from_profile=None,
                             description="", activate=False)
            args.name = name
            cli.cmd_create(args)

        # Write distinct settings
        (self.profiles_dir / "a" / "settings.json").write_text('{"profile": "a"}')
        (self.profiles_dir / "b" / "settings.json").write_text('{"profile": "b"}')

        cli._do_switch("a", force=True)
        content = json.loads((self.claude_dir / "settings.json").read_text())
        self.assertEqual(content["profile"], "a")

        cli._do_switch("b")
        content = json.loads((self.claude_dir / "settings.json").read_text())
        self.assertEqual(content["profile"], "b")

    def test_switch_dry_run(self):
        self._populate_claude_dir()
        args = mock.Mock(from_current=True, from_profile=None,
                         description="", activate=False)
        args.name = "p1"
        cli.cmd_create(args)

        cli._do_switch("p1", force=True, dry_run=True)

        # Nothing should have changed
        self.assertFalse((self.claude_dir / "settings.json").is_symlink())
        config = cli.load_config()
        self.assertIsNone(config["active"])

    def test_switch_nonexistent_fails(self):
        with self.assertRaises(SystemExit):
            cli._do_switch("nope")

    def test_switch_validates_manifest(self):
        """Switch should fail if manifest lists a path that doesn't exist in profile."""
        pdir = self.profiles_dir / "broken"
        pdir.mkdir(parents=True)
        (pdir / "settings.json").write_text("{}")
        cli.save_manifest(pdir, ["settings.json", "nonexistent_file.json"])

        with self.assertRaises(SystemExit):
            cli._do_switch("broken", force=True)


class TestShare(ClaudeHopperTestCase):

    def test_share_between_profiles(self):
        # Create two profiles with different content
        for name in ["owner", "consumer"]:
            args = mock.Mock(from_current=False, from_profile=None,
                             description="", activate=False)
            args.name = name
            cli.cmd_create(args)

        # Add a commands dir to owner
        cmds = self.profiles_dir / "owner" / "commands"
        cmds.mkdir()
        (cmds / "tool.md").write_text("# shared tool")
        manifest = cli.load_manifest(self.profiles_dir / "owner")
        cli.save_manifest(self.profiles_dir / "owner",
                          manifest["managed_paths"] + ["commands"])

        # Share commands from owner to consumer
        args = mock.Mock(source="owner", paths=["commands"],
                         target="consumer", dry_run=False)
        cli.cmd_share(args)

        # Consumer should have a symlink
        consumer_cmds = self.profiles_dir / "consumer" / "commands"
        self.assertTrue(consumer_cmds.is_symlink())

        # Verify shared_paths recorded in manifest
        shared = cli.get_shared_paths(self.profiles_dir / "consumer")
        self.assertEqual(shared["commands"], "owner")

    def test_share_dry_run(self):
        for name in ["a", "b"]:
            args = mock.Mock(from_current=False, from_profile=None,
                             description="", activate=False)
            args.name = name
            cli.cmd_create(args)

        args = mock.Mock(source="a", paths=["settings.json"],
                         target="b", dry_run=True)
        cli.cmd_share(args)

        # Nothing should have changed
        shared = cli.get_shared_paths(self.profiles_dir / "b")
        self.assertEqual(shared, {})

    def test_share_self_fails(self):
        args = mock.Mock(from_current=False, from_profile=None,
                         description="", activate=False)
        args.name = "x"
        cli.cmd_create(args)

        with self.assertRaises(SystemExit):
            share_args = mock.Mock(source="x", paths=["settings.json"],
                                   target="x", dry_run=False)
            cli.cmd_share(share_args)


class TestUnshare(ClaudeHopperTestCase):

    def test_unshare_materializes(self):
        # Create owner with a file
        for name in ["owner", "consumer"]:
            args = mock.Mock(from_current=False, from_profile=None,
                             description="", activate=False)
            args.name = name
            cli.cmd_create(args)

        # Write distinct settings in owner
        (self.profiles_dir / "owner" / "settings.json").write_text('{"shared": true}')

        # Share it
        share_args = mock.Mock(source="owner", paths=["settings.json"],
                               target="consumer", dry_run=False)
        cli.cmd_share(share_args)

        # Verify it's a symlink
        consumer_settings = self.profiles_dir / "consumer" / "settings.json"
        self.assertTrue(consumer_settings.is_symlink())

        # Unshare it
        unshare_args = mock.Mock(paths=["settings.json"], profile="consumer", dry_run=False)
        cli.cmd_unshare(unshare_args)

        # Should now be a real file with same content
        self.assertFalse(consumer_settings.is_symlink())
        self.assertEqual(json.loads(consumer_settings.read_text()), {"shared": True})

        # shared_paths should be empty
        shared = cli.get_shared_paths(self.profiles_dir / "consumer")
        self.assertNotIn("settings.json", shared)


class TestPick(ClaudeHopperTestCase):

    def test_pick_copies_files(self):
        for name in ["src", "dst"]:
            args = mock.Mock(from_current=False, from_profile=None,
                             description="", activate=False)
            args.name = name
            cli.cmd_create(args)

        (self.profiles_dir / "src" / "settings.json").write_text('{"picked": true}')

        # Activate dst so pick defaults to it
        cli._do_switch("dst", force=True)

        pick_args = mock.Mock(source="src", paths=["settings.json"],
                              target=None, dry_run=False)
        cli.cmd_pick(pick_args)

        dst_settings = self.profiles_dir / "dst" / "settings.json"
        self.assertFalse(dst_settings.is_symlink())  # copy, not link
        self.assertEqual(json.loads(dst_settings.read_text()), {"picked": True})

    def test_pick_dry_run(self):
        for name in ["src", "dst"]:
            args = mock.Mock(from_current=False, from_profile=None,
                             description="", activate=False)
            args.name = name
            cli.cmd_create(args)

        original = (self.profiles_dir / "dst" / "settings.json").read_text()

        pick_args = mock.Mock(source="src", paths=["settings.json"],
                              target="dst", dry_run=True)
        cli.cmd_pick(pick_args)

        # dst should be unchanged
        self.assertEqual((self.profiles_dir / "dst" / "settings.json").read_text(), original)


class TestDelete(ClaudeHopperTestCase):

    def test_delete_active_fails(self):
        args = mock.Mock(from_current=False, from_profile=None,
                         description="", activate=False)
        args.name = "test"
        cli.cmd_create(args)
        cli._do_switch("test", force=True)

        with self.assertRaises(SystemExit):
            del_args = mock.Mock(yes=True)
            del_args.name = "test"
            cli.cmd_delete(del_args)

    def test_delete_warns_about_dependents(self):
        # Create owner and consumer with shared files
        for name in ["owner", "consumer"]:
            args = mock.Mock(from_current=False, from_profile=None,
                             description="", activate=False)
            args.name = name
            cli.cmd_create(args)

        share_args = mock.Mock(source="owner", paths=["settings.json"],
                               target="consumer", dry_run=False)
        cli.cmd_share(share_args)

        # Delete owner with --yes should still work
        del_args = mock.Mock(yes=True)
        del_args.name = "owner"
        cli.cmd_delete(del_args)
        self.assertFalse((self.profiles_dir / "owner").exists())


class TestDiff(ClaudeHopperTestCase):

    def test_diff_shows_differences(self):
        for name in ["a", "b"]:
            args = mock.Mock(from_current=False, from_profile=None,
                             description="", activate=False)
            args.name = name
            cli.cmd_create(args)

        (self.profiles_dir / "a" / "settings.json").write_text('{"a": true}')
        (self.profiles_dir / "b" / "settings.json").write_text('{"b": true}')

        diff_args = mock.Mock(profile_a="a", profile_b="b")
        cli.cmd_diff(diff_args)  # should not raise


class TestUnmanage(ClaudeHopperTestCase):

    def test_unmanage_materializes_symlinks(self):
        self._populate_claude_dir()
        args = mock.Mock(from_current=True, from_profile=None,
                         description="", activate=False)
        args.name = "p1"
        cli.cmd_create(args)
        cli._do_switch("p1", force=True)

        # Verify symlinks exist
        self.assertTrue((self.claude_dir / "settings.json").is_symlink())

        unmanage_args = mock.Mock(dry_run=False)
        cli.cmd_unmanage(unmanage_args)

        # Should be real files now
        self.assertFalse((self.claude_dir / "settings.json").is_symlink())
        self.assertTrue((self.claude_dir / "settings.json").exists())
        config = cli.load_config()
        self.assertIsNone(config["active"])


class TestMigration(ClaudeHopperTestCase):

    def _setup_legacy_dir(self):
        """Create a fake ~/.claude-swap/ with a profile."""
        self.legacy_dir.mkdir(parents=True)
        legacy_profiles = self.legacy_dir / "profiles"
        legacy_profiles.mkdir()

        # Create a legacy profile
        pdir = legacy_profiles / "myprofile"
        pdir.mkdir()
        (pdir / "settings.json").write_text('{"legacy": true}')
        # Write a legacy manifest
        manifest = {
            "managed_paths": ["settings.json"],
            "shared_paths": {},
            "description": "legacy profile",
        }
        (pdir / ".ccswap-manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")

        # Write a legacy config
        config = {"active": "myprofile"}
        (self.legacy_dir / "config.json").write_text(json.dumps(config, indent=2) + "\n")

        return pdir

    def test_check_migration_detects_legacy(self):
        """check_migration returns True when legacy dir exists and hopper dir does not."""
        self.legacy_dir.mkdir(parents=True)
        self.assertTrue(cli.check_migration())

    def test_check_migration_no_legacy(self):
        """check_migration returns False when no legacy dir exists."""
        self.assertFalse(cli.check_migration())

    def test_check_migration_both_exist(self):
        """check_migration returns False when both dirs exist (already migrated)."""
        self.legacy_dir.mkdir(parents=True)
        self.hopper_dir.mkdir(parents=True)
        self.assertFalse(cli.check_migration())

    def test_full_migration(self):
        """Migration copies profiles and renames manifest files."""
        self._setup_legacy_dir()

        args = mock.Mock(dry_run=False)
        cli.cmd_migrate(args)

        # Legacy dir should be gone
        self.assertFalse(self.legacy_dir.exists())

        # New profile dir should exist with migrated profile
        new_pdir = self.profiles_dir / "myprofile"
        self.assertTrue(new_pdir.exists())
        self.assertTrue((new_pdir / "settings.json").exists())

        # Manifest should use the new name
        self.assertTrue((new_pdir / ".hop-manifest.json").exists())
        self.assertFalse((new_pdir / ".ccswap-manifest.json").exists())

        # Config should be migrated
        config = cli.load_config()
        self.assertEqual(config.get("active"), "myprofile")

    def test_migration_dry_run(self):
        """Dry-run migration does not touch any files."""
        self._setup_legacy_dir()

        args = mock.Mock(dry_run=True)
        cli.cmd_migrate(args)

        # Legacy dir should be untouched
        self.assertTrue(self.legacy_dir.exists())
        # New dir should not have been created
        self.assertFalse(self.hopper_dir.exists())

    def test_migration_no_legacy_dir(self):
        """migrate command exits cleanly when no legacy dir exists."""
        args = mock.Mock(dry_run=False)
        cli.cmd_migrate(args)  # should not raise

    def test_migration_idempotent(self):
        """Migrating when hopper dir already has profiles raises an error."""
        self._setup_legacy_dir()

        # Pre-create the hopper profiles dir with content
        new_pdir = self.profiles_dir / "existing"
        new_pdir.mkdir(parents=True)

        args = mock.Mock(dry_run=False)
        with self.assertRaises(SystemExit):
            cli.cmd_migrate(args)


class TestAtomicSymlink(ClaudeHopperTestCase):

    def test_atomic_symlink_creates_symlink(self):
        target = self.tmpdir / "target.txt"
        target.write_text("hello")
        link = self.tmpdir / "link.txt"

        cli.atomic_symlink(target, link)

        self.assertTrue(link.is_symlink())
        self.assertEqual(link.read_text(), "hello")

    def test_atomic_symlink_replaces_existing(self):
        target_a = self.tmpdir / "a.txt"
        target_b = self.tmpdir / "b.txt"
        target_a.write_text("aaa")
        target_b.write_text("bbb")
        link = self.tmpdir / "link.txt"

        cli.atomic_symlink(target_a, link)
        self.assertEqual(link.read_text(), "aaa")

        cli.atomic_symlink(target_b, link)
        self.assertEqual(link.read_text(), "bbb")


class TestPathCommand(ClaudeHopperTestCase):

    def test_path_prints_profile_dir(self, capsys=None):
        args = mock.Mock(from_current=False, from_profile=None,
                         description="", activate=False)
        args.name = "myprof"
        cli.cmd_create(args)

        path_args = mock.Mock()
        path_args.name = "myprof"
        # Should not raise
        cli.cmd_path(path_args)

    def test_path_nonexistent_fails(self):
        path_args = mock.Mock()
        path_args.name = "doesnotexist"
        with self.assertRaises(SystemExit):
            cli.cmd_path(path_args)


if __name__ == "__main__":
    unittest.main()
