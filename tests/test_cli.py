"""Tests for claudehopper using isolated temp directories."""

import argparse
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
        self.claude_dir.mkdir()

        # Patch module-level paths
        self.shared_dir = self.hopper_dir / "shared"
        self._patchers = [
            mock.patch.object(cli, "CLAUDE_DIR", self.claude_dir),
            mock.patch.object(cli, "HOPPER_DIR", self.hopper_dir),
            mock.patch.object(cli, "PROFILES_DIR", self.profiles_dir),
            mock.patch.object(cli, "CONFIG_FILE", self.hopper_dir / "config.json"),
            mock.patch.object(cli, "SHARED_DIR", self.shared_dir),
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

    @mock.patch.object(cli, "_prompt", return_value="n")
    def test_switch_between_profiles(self, mock_prompt):
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


class TestAdoptOnSwitch(ClaudeHopperTestCase):
    """Tests for detecting and adopting unmanaged files on switch."""

    def _setup_profile_with_extra_files(self):
        """Create a profile, switch to it, then add unmanaged files."""
        args = argparse.Namespace(
            name="main", from_current=False, from_profile=None,
            description="", activate=False, no_shared_defaults=True,
        )
        cli.cmd_create(args)
        cli._do_switch("main", force=True)

        # Simulate an installer creating new files in ~/.claude/
        (self.claude_dir / "agents").mkdir()
        (self.claude_dir / "agents" / "helper.md").write_text("# helper")
        (self.claude_dir / "new-config.json").write_text('{"installed": true}')

        args2 = argparse.Namespace(
            name="other", from_current=False, from_profile=None,
            description="", activate=False, no_shared_defaults=True,
        )
        cli.cmd_create(args2)

    @mock.patch.object(cli, "_prompt", return_value="y")
    def test_adopt_yes_moves_files_into_profile(self, mock_prompt):
        self._setup_profile_with_extra_files()
        cli._do_switch("other")

        pdir = self.profiles_dir / "main"
        manifest = cli.load_manifest(pdir)
        managed = manifest["managed_paths"]
        self.assertIn("agents", managed)
        self.assertIn("new-config.json", managed)
        self.assertTrue((pdir / "agents" / "helper.md").exists())
        self.assertTrue((pdir / "new-config.json").exists())

    @mock.patch.object(cli, "_prompt", return_value="n")
    def test_adopt_no_leaves_files(self, mock_prompt):
        self._setup_profile_with_extra_files()
        cli._do_switch("other")

        # Files should NOT be in the profile
        pdir = self.profiles_dir / "main"
        manifest = cli.load_manifest(pdir)
        self.assertNotIn("agents", manifest["managed_paths"])
        self.assertNotIn("new-config.json", manifest["managed_paths"])

    @mock.patch.object(cli, "_prompt", return_value="skip")
    def test_adopt_skip_leaves_files(self, mock_prompt):
        self._setup_profile_with_extra_files()
        cli._do_switch("other")

        pdir = self.profiles_dir / "main"
        manifest = cli.load_manifest(pdir)
        self.assertNotIn("agents", manifest["managed_paths"])

    def test_no_prompt_when_no_unmanaged(self):
        """No prompt if there are no unmanaged files."""
        args = argparse.Namespace(
            name="a", from_current=False, from_profile=None,
            description="", activate=False, no_shared_defaults=True,
        )
        cli.cmd_create(args)
        args2 = argparse.Namespace(
            name="b", from_current=False, from_profile=None,
            description="", activate=False, no_shared_defaults=True,
        )
        cli.cmd_create(args2)

        cli._do_switch("a", force=True)
        # Switch with no extra files — should not prompt
        with mock.patch.object(cli, "_prompt") as mock_prompt:
            cli._do_switch("b")
            mock_prompt.assert_not_called()

    def test_adopt_flag_skips_prompt(self):
        """--adopt flag auto-adopts without prompting."""
        self._setup_profile_with_extra_files()
        with mock.patch.object(cli, "_prompt") as mock_prompt:
            cli._do_switch("other", adopt=True)
            mock_prompt.assert_not_called()

        pdir = self.profiles_dir / "main"
        manifest = cli.load_manifest(pdir)
        self.assertIn("agents", manifest["managed_paths"])
        self.assertIn("new-config.json", manifest["managed_paths"])

    def test_detect_unmanaged_ignores_shared_paths(self):
        """Shared paths like .credentials.json should not be flagged."""
        args = argparse.Namespace(
            name="main", from_current=False, from_profile=None,
            description="", activate=False, no_shared_defaults=True,
        )
        cli.cmd_create(args)
        cli._do_switch("main", force=True)

        # These are in SHARED_PATHS and should be ignored
        (self.claude_dir / ".credentials.json").write_text("{}")
        (self.claude_dir / "projects").mkdir(exist_ok=True)

        unmanaged = cli.detect_unmanaged("main")
        self.assertNotIn(".credentials.json", unmanaged)
        self.assertNotIn("projects", unmanaged)


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


class TestUsageRecording(ClaudeHopperTestCase):

    def test_record_usage_creates_file(self):
        self.hopper_dir.mkdir(parents=True, exist_ok=True)
        cli.record_usage("omc", "switch")
        usage_file = self.hopper_dir / "usage.jsonl"
        self.assertTrue(usage_file.exists())
        lines = usage_file.read_text().splitlines()
        self.assertEqual(len(lines), 1)
        entry = json.loads(lines[0])
        self.assertEqual(entry["profile"], "omc")
        self.assertEqual(entry["action"], "switch")
        self.assertIn("timestamp", entry)

    def test_record_usage_appends(self):
        self.hopper_dir.mkdir(parents=True, exist_ok=True)
        cli.record_usage("omc", "switch")
        cli.record_usage("gsd", "create")
        cli.record_usage("omc", "switch")
        usage_file = self.hopper_dir / "usage.jsonl"
        lines = usage_file.read_text().splitlines()
        self.assertEqual(len(lines), 3)
        self.assertEqual(json.loads(lines[0])["profile"], "omc")
        self.assertEqual(json.loads(lines[1])["profile"], "gsd")
        self.assertEqual(json.loads(lines[2])["action"], "switch")

    def test_record_usage_on_switch(self):
        self._populate_claude_dir()
        args = mock.Mock(from_current=True, from_profile=None,
                         description="", activate=False)
        args.name = "p1"
        cli.cmd_create(args)
        cli._do_switch("p1", force=True)

        usage_file = self.hopper_dir / "usage.jsonl"
        self.assertTrue(usage_file.exists())
        entries = [json.loads(l) for l in usage_file.read_text().splitlines()]
        switch_entries = [e for e in entries if e["action"] == "switch"]
        self.assertTrue(len(switch_entries) >= 1)
        self.assertEqual(switch_entries[-1]["profile"], "p1")

    def test_record_usage_on_create(self):
        args = mock.Mock(from_current=False, from_profile=None,
                         description="", activate=False)
        args.name = "myprof"
        cli.cmd_create(args)

        usage_file = self.hopper_dir / "usage.jsonl"
        self.assertTrue(usage_file.exists())
        entries = [json.loads(l) for l in usage_file.read_text().splitlines()]
        create_entries = [e for e in entries if e["action"] == "create"]
        self.assertEqual(len(create_entries), 1)
        self.assertEqual(create_entries[0]["profile"], "myprof")

    def test_record_usage_not_on_dry_run(self):
        self._populate_claude_dir()
        args = mock.Mock(from_current=True, from_profile=None,
                         description="", activate=False)
        args.name = "p1"
        cli.cmd_create(args)

        # Clear any usage recorded during create
        usage_file = self.hopper_dir / "usage.jsonl"
        usage_file.write_text("")

        cli._do_switch("p1", force=True, dry_run=True)

        switch_entries = [
            json.loads(l) for l in usage_file.read_text().splitlines() if l.strip()
            if json.loads(l)["action"] == "switch"
        ]
        self.assertEqual(len(switch_entries), 0)


class TestStats(ClaudeHopperTestCase):

    def setUp(self):
        super().setUp()
        self.usage_file = self.hopper_dir / "usage.jsonl"

    def _write_entries(self, entries):
        self.hopper_dir.mkdir(parents=True, exist_ok=True)
        with self.usage_file.open("w") as f:
            for entry in entries:
                f.write(json.dumps(entry) + "\n")

    def test_stats_empty(self):
        import io
        from contextlib import redirect_stdout
        buf = io.StringIO()
        with redirect_stdout(buf):
            stats_args = mock.Mock(profile=None, since=None)
            stats_args.json = False
            cli.cmd_stats(stats_args)
        self.assertIn("No usage data yet", buf.getvalue())

    def test_stats_with_data(self):
        import io
        from contextlib import redirect_stdout
        self._write_entries([
            {"profile": "omc", "timestamp": "2026-03-01T10:00:00", "action": "switch"},
            {"profile": "omc", "timestamp": "2026-03-02T10:00:00", "action": "switch"},
            {"profile": "gsd", "timestamp": "2026-03-03T10:00:00", "action": "switch"},
        ])
        buf = io.StringIO()
        with redirect_stdout(buf):
            stats_args = mock.Mock(profile=None, since=None)
            stats_args.json = False
            cli.cmd_stats(stats_args)
        output = buf.getvalue()
        self.assertIn("omc", output)
        self.assertIn("gsd", output)
        self.assertIn("3 switches", output)  # total

    def test_stats_profile_filter(self):
        import io
        from contextlib import redirect_stdout
        self._write_entries([
            {"profile": "omc", "timestamp": "2026-03-01T10:00:00", "action": "switch"},
            {"profile": "omc", "timestamp": "2026-03-02T10:00:00", "action": "switch"},
            {"profile": "gsd", "timestamp": "2026-03-03T10:00:00", "action": "switch"},
        ])
        buf = io.StringIO()
        with redirect_stdout(buf):
            stats_args = mock.Mock(profile="omc", since=None)
            stats_args.json = False
            cli.cmd_stats(stats_args)
        output = buf.getvalue()
        self.assertIn("omc", output)
        self.assertNotIn("gsd", output)

    def test_stats_json_output(self):
        self._write_entries([
            {"profile": "omc", "timestamp": "2026-03-01T10:00:00", "action": "switch"},
            {"profile": "omc", "timestamp": "2026-03-02T10:00:00", "action": "switch"},
            {"profile": "gsd", "timestamp": "2026-03-03T10:00:00", "action": "switch"},
        ])
        import io
        from contextlib import redirect_stdout
        buf = io.StringIO()
        with redirect_stdout(buf):
            stats_args = mock.Mock(profile=None, since=None)
            stats_args.json = True
            cli.cmd_stats(stats_args)
        result = json.loads(buf.getvalue())
        self.assertIn("profiles", result)
        self.assertIn("total", result)
        self.assertEqual(result["total"], 3)
        names = [p["name"] for p in result["profiles"]]
        self.assertIn("omc", names)
        self.assertIn("gsd", names)
        omc = next(p for p in result["profiles"] if p["name"] == "omc")
        self.assertEqual(omc["switches"], 2)



class TestTree(ClaudeHopperTestCase):

    def _capture_tree(self, json_flag=False):
        import io
        from contextlib import redirect_stdout
        buf = io.StringIO()
        args = mock.Mock()
        args.json = json_flag
        with redirect_stdout(buf):
            cli.cmd_tree(args)
        return buf.getvalue()

    def _make_profile(self, name, description=""):
        args = mock.Mock(from_current=False, from_profile=None,
                         description=description, activate=False)
        args.name = name
        cli.cmd_create(args)

    def test_tree_no_profiles(self):
        output = self._capture_tree()
        self.assertIn("No profiles", output)
        self.assertIn("claudehopper create", output)

    def test_tree_single_profile(self):
        self._make_profile("vanilla")
        output = self._capture_tree()
        self.assertIn("claudehopper profiles", output)
        self.assertIn("vanilla", output)
        self.assertIn("settings.json", output)

    def test_tree_multiple_profiles(self):
        self._make_profile("alpha")
        self._make_profile("beta")
        cli._do_switch("alpha", force=True)
        output = self._capture_tree()
        self.assertIn("alpha (active)", output)
        self.assertIn("beta", output)
        self.assertNotIn("beta (active)", output)

    def test_tree_with_shared_paths(self):
        self._make_profile("owner")
        self._make_profile("consumer")
        share_args = mock.Mock(source="owner", paths=["settings.json"],
                               target="consumer", dry_run=False)
        cli.cmd_share(share_args)
        output = self._capture_tree()
        self.assertIn("shared from owner", output)
        self.assertIn("Shared links:", output)
        self.assertIn("consumer/settings.json", output)

    def test_tree_with_lineage(self):
        self._make_profile("parent")
        clone_args = mock.Mock(from_current=False, from_profile="parent",
                               description="child profile", activate=False)
        clone_args.name = "child"
        cli.cmd_create(clone_args)
        output = self._capture_tree()
        # child should appear indented under parent
        lines = output.splitlines()
        parent_idx = next(i for i, l in enumerate(lines) if "parent" in l)
        child_idx = next(i for i, l in enumerate(lines) if "child" in l)
        self.assertGreater(child_idx, parent_idx)
        child_line = lines[child_idx]
        # child line should be indented (has leading spaces or box chars before name)
        self.assertTrue(child_line.startswith(" ") or child_line.startswith("│"))

    def test_tree_json_output(self):
        import io
        from contextlib import redirect_stdout
        self._make_profile("omc", description="my omc profile")
        self._make_profile("vanilla")
        cli._do_switch("omc", force=True)
        output = self._capture_tree(json_flag=True)
        result = json.loads(output)
        self.assertIn("profiles", result)
        names = [p["name"] for p in result["profiles"]]
        self.assertIn("omc", names)
        self.assertIn("vanilla", names)
        omc = next(p for p in result["profiles"] if p["name"] == "omc")
        self.assertTrue(omc["active"])
        self.assertEqual(omc["description"], "my omc profile")
        self.assertIn("managed_paths", omc)
        self.assertIn("shared_paths", omc)
        self.assertIn("created_from", omc)
        vanilla = next(p for p in result["profiles"] if p["name"] == "vanilla")
        self.assertFalse(vanilla["active"])

    def test_tree_no_profiles_json(self):
        output = self._capture_tree(json_flag=True)
        result = json.loads(output)
        self.assertEqual(result, {"profiles": []})

class TestSharedDefaults(ClaudeHopperTestCase):
    """Tests for DEFAULT_LINKED shared file behavior."""

    def test_create_from_current_seeds_shared_dir(self):
        """First profile created from current ~/.claude/ seeds the shared dir."""
        self._populate_claude_dir()
        # Add settings.local.json and .mcp.json to claude_dir
        (self.claude_dir / "settings.local.json").write_text('{"local": true}')
        (self.claude_dir / ".mcp.json").write_text('{"mcpServers": {}}')

        args = argparse.Namespace(
            name="first", from_current=True, from_profile=None,
            description="first profile", activate=False, no_shared_defaults=False,
        )
        cli.cmd_create(args)

        pdir = self.profiles_dir / "first"
        # Shared dir should now exist and contain the files
        self.assertTrue(self.shared_dir.exists())
        self.assertTrue((self.shared_dir / "settings.json").exists())
        self.assertTrue((self.shared_dir / "settings.local.json").exists())
        self.assertTrue((self.shared_dir / ".mcp.json").exists())

        # Profile files should be symlinks to shared dir
        self.assertTrue((pdir / "settings.json").is_symlink())
        self.assertTrue((pdir / "settings.local.json").is_symlink())
        self.assertTrue((pdir / ".mcp.json").is_symlink())
        self.assertEqual((pdir / "settings.json").resolve(), (self.shared_dir / "settings.json").resolve())

    def test_create_blank_links_existing_shared(self):
        """Blank profile gets symlinks if shared dir already has files."""
        # Pre-populate shared dir
        self.shared_dir.mkdir(parents=True)
        (self.shared_dir / "settings.json").write_text('{"shared": true}')
        (self.shared_dir / ".mcp.json").write_text('{"mcpServers": {}}')

        args = argparse.Namespace(
            name="blank", from_current=False, from_profile=None,
            description="blank", activate=False, no_shared_defaults=False,
        )
        cli.cmd_create(args)

        pdir = self.profiles_dir / "blank"
        # settings.json should be a symlink to shared
        self.assertTrue((pdir / "settings.json").is_symlink())
        self.assertEqual((pdir / "settings.json").resolve(), (self.shared_dir / "settings.json").resolve())
        # .mcp.json should also be linked
        self.assertTrue((pdir / ".mcp.json").is_symlink())

    def test_no_shared_defaults_flag(self):
        """--no-shared-defaults prevents linking."""
        self.shared_dir.mkdir(parents=True)
        (self.shared_dir / "settings.json").write_text('{"shared": true}')

        args = argparse.Namespace(
            name="isolated", from_current=False, from_profile=None,
            description="isolated", activate=False, no_shared_defaults=True,
        )
        cli.cmd_create(args)

        pdir = self.profiles_dir / "isolated"
        # settings.json should be a regular file, not a symlink
        self.assertTrue((pdir / "settings.json").exists())
        self.assertFalse((pdir / "settings.json").is_symlink())

    def test_shared_content_is_preserved(self):
        """Shared files contain the original content after linking."""
        self._populate_claude_dir()
        (self.claude_dir / "settings.json").write_text('{"hooks": {"pre": "test"}}')

        args = argparse.Namespace(
            name="check", from_current=True, from_profile=None,
            description="", activate=False, no_shared_defaults=False,
        )
        cli.cmd_create(args)

        # Shared file should have the original content
        content = json.loads((self.shared_dir / "settings.json").read_text())
        self.assertEqual(content, {"hooks": {"pre": "test"}})

        # Reading through the profile symlink should give the same content
        pdir = self.profiles_dir / "check"
        content_via_link = json.loads((pdir / "settings.json").read_text())
        self.assertEqual(content_via_link, {"hooks": {"pre": "test"}})

    def test_manifest_records_shared_paths(self):
        """Manifest should record shared paths after linking."""
        self._populate_claude_dir()

        args = argparse.Namespace(
            name="manifest", from_current=True, from_profile=None,
            description="", activate=False, no_shared_defaults=False,
        )
        cli.cmd_create(args)

        pdir = self.profiles_dir / "manifest"
        manifest = cli.load_manifest(pdir)
        shared = manifest.get("shared_paths", {})
        self.assertEqual(shared.get("settings.json"), "(shared)")

    def test_second_profile_links_to_same_shared(self):
        """Two profiles both link to the same shared files."""
        self._populate_claude_dir()
        (self.claude_dir / ".mcp.json").write_text('{"mcpServers": {"fs": {}}}')

        args1 = argparse.Namespace(
            name="first", from_current=True, from_profile=None,
            description="", activate=False, no_shared_defaults=False,
        )
        cli.cmd_create(args1)

        args2 = argparse.Namespace(
            name="second", from_current=False, from_profile=None,
            description="", activate=False, no_shared_defaults=False,
        )
        cli.cmd_create(args2)

        p1 = self.profiles_dir / "first"
        p2 = self.profiles_dir / "second"

        # Both should point to the same shared file
        self.assertEqual(
            (p1 / ".mcp.json").resolve(),
            (p2 / ".mcp.json").resolve(),
        )

    def test_missing_default_linked_files_skipped(self):
        """Files not present in source or shared dir are silently skipped."""
        # Only create settings.json, not settings.local.json or .mcp.json
        self._populate_claude_dir()

        args = argparse.Namespace(
            name="partial", from_current=True, from_profile=None,
            description="", activate=False, no_shared_defaults=False,
        )
        cli.cmd_create(args)

        pdir = self.profiles_dir / "partial"
        # settings.json should be linked (it existed)
        self.assertTrue((pdir / "settings.json").is_symlink())
        # settings.local.json and .mcp.json should not exist (weren't in source)
        self.assertFalse((pdir / "settings.local.json").exists())
        self.assertFalse((pdir / ".mcp.json").exists())

    def test_link_defaults_replaces_existing_file(self):
        """link_defaults_into_profile replaces a real file with a symlink."""
        self.shared_dir.mkdir(parents=True)
        (self.shared_dir / "settings.json").write_text('{"shared": true}')

        pdir = self.profiles_dir / "replace"
        pdir.mkdir(parents=True)
        (pdir / "settings.json").write_text('{"old": true}')
        cli.save_manifest(pdir, ["settings.json"])

        cli.link_defaults_into_profile(pdir)

        self.assertTrue((pdir / "settings.json").is_symlink())
        content = json.loads((pdir / "settings.json").read_text())
        self.assertEqual(content, {"shared": True})


if __name__ == "__main__":
    unittest.main()
