#!/usr/bin/env python3
"""Portable launcher regressions; downloads use a fixture curl, never the network."""
import hashlib
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
SHELL = os.environ.get("ROCKDEV_TEST_SHELL") or shutil.which("sh")
BASE_PATH = subprocess.check_output([SHELL, "-c", 'printf "%s" "$PATH"'], text=True)


def shell_path(path):
    value = Path(path).resolve().as_posix()
    if os.name == "nt":
        return "/" + value[0].lower() + value[2:]
    return value


class LauncherTests(unittest.TestCase):
    def setUp(self):
        self.assertTrue((ROOT / "rockdev").is_file(), "root ./rockdev launcher is missing")
        self.tmp = tempfile.TemporaryDirectory(prefix="launcher-", dir=ROOT / ".cache")
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.repo = self.root / "repo with spaces"
        self.repo.mkdir()
        (self.repo / "release").mkdir()
        self.assets = self.root / "assets"
        self.assets.mkdir()
        self.bin = self.root / "bin"
        self.bin.mkdir()
        self.cache = self.root / "cache"
        self.cache.mkdir()
        self.os_file = self.root / "os-release"
        self.os_file.write_text('ID=rocky\nVERSION_ID="8.10"\n')
        source = (ROOT / "rockdev").read_text()
        self.assertIn("OS_RELEASE_FILE=/etc/os-release", source)
        self.assertIn("cache_home=/var/cache", source)
        source = source.replace("OS_RELEASE_FILE=/etc/os-release", "OS_RELEASE_FILE='" + shell_path(self.os_file) + "'")
        source = source.replace("cache_home=/var/cache", "cache_home='" + shell_path(self.cache) + "'")
        self.launcher = self.repo / "rockdev"
        self.launcher.write_text(source, newline="\n")
        (self.repo / "release/VERSION").write_text("1.0.1\n")
        sums = []
        for arch in ["amd64", "arm64"]:
            name = "rockdev-linux-" + arch
            payload = '#!/bin/sh\nprintf "%s\\n" "PAYLOAD-' + arch + '" "$@"\nexit "${TEST_PAYLOAD_EXIT:-0}"\n'
            (self.assets / name).write_text(payload, newline="\n")
            sums.append(hashlib.sha256(payload.encode()).hexdigest() + "  " + name)
        (self.repo / "release/SHA256SUMS").write_text("\n".join(sums) + "\n")
        self.stub("uname", 'case "$1" in -s) echo "${TEST_OS:-Linux}";; -m) echo "${TEST_ARCH:-x86_64}";; *) exit 1;; esac\n')
        self.stub("curl", '''out=; url=
while [ "$#" -gt 0 ]; do
 case "$1" in --output) out=$2; shift 2;; *) url=$1; shift;; esac
done
printf '%s\\n' "$url" >> "$TEST_DOWNLOAD_LOG"
[ "${TEST_NETWORK_FAIL:-0}" = 0 ] || exit 22
cp "$TEST_ASSETS/${url##*/}" "$out"
''')
        if os.name == 'nt':
            self.stub('stat', '''for target do :; done
case "$*" in *%a*)
 if [ "$target" = "${TEST_MODE_PATH:-}" ]; then echo "$TEST_MODE_VALUE"; exit; fi
 case "$target" in "$TEST_ROOT"|"$TEST_ROOT"/*) ;; *) echo 755; exit;; esac
esac
exec /usr/bin/stat "$@"
''')
        self.log = self.root / "downloads.log"
        self.env = dict(os.environ, XDG_CACHE_HOME=shell_path(self.cache),
                        TEST_ASSETS=shell_path(self.assets), TEST_DOWNLOAD_LOG=shell_path(self.log))
        self.env["TEST_PATH"] = shell_path(self.bin) + ":" + BASE_PATH
        self.env['TEST_ROOT'] = shell_path(self.root)
        for key in ["TEST_OS", "TEST_ARCH", "TEST_NETWORK_FAIL", "TEST_PAYLOAD_EXIT"]:
            self.env.pop(key, None)

    def stub(self, name, body):
        p = self.bin / name
        p.write_text("#!/bin/sh\n" + body, newline="\n")
        p.chmod(0o755)

    def run_cli(self, *args, **env):
        return subprocess.run(self.command(*args), env=dict(self.env, **env),
                              text=True, capture_output=True, timeout=30)

    def command(self, *args):
        # Git Bash prepends its own PATH on startup; set the fixture PATH inside
        # the shell before sourcing the otherwise unchanged launcher.
        return [SHELL, "-c", 'PATH="$TEST_PATH"; export PATH; . "$0"', shell_path(self.launcher), *args]

    def downloads(self):
        return self.log.read_text().splitlines() if self.log.exists() else []

    def cached(self, arch="amd64"):
        return self.cache / "rockdev" / "1.0.1" / arch / ("rockdev-linux-" + arch)

    def move_cache(self, path):
        old = self.cache
        old.rename(path)
        self.cache = path
        self.env['XDG_CACHE_HOME'] = shell_path(path)
        source = self.launcher.read_text().replace("cache_home='" + shell_path(old) + "'",
                                                  "cache_home='" + shell_path(path) + "'")
        self.launcher.write_text(source)

    def test_writable_ancestor_without_sticky_is_rejected(self):
        shared = self.root / 'shared'
        shared.mkdir()
        shared.chmod(0o777)
        self.move_cache(shared / 'owned-cache')
        r = self.run_cli(TEST_MODE_PATH=shell_path(shared), TEST_MODE_VALUE='777')
        self.assertNotEqual(r.returncode, 0)
        self.assertIn('writable ancestor', r.stderr)
        self.assertEqual(self.downloads(), [])

    def test_owned_child_under_trusted_sticky_ancestor(self):
        shared = self.root / 'sticky'
        shared.mkdir()
        shared.chmod(0o1777)
        self.move_cache(shared / 'owned-cache')
        r = self.run_cli(TEST_MODE_PATH=shell_path(shared), TEST_MODE_VALUE='1777')
        self.assertEqual(r.returncode, 0, r.stderr)

    def test_first_run_and_argument_forwarding(self):
        r = self.run_cli("--lang", "ja", "profile", "validate", "a file.json")
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertEqual(r.stdout.splitlines(), ["PAYLOAD-amd64", "--lang", "ja", "profile", "validate", "a file.json"])
        self.assertEqual(self.downloads(), ["https://github.com/YWQ211/Rocky_Linux_deployment_tool/releases/download/v1.0.1/rockdev-linux-amd64"])

    def test_cached_run_needs_no_network(self):
        self.assertEqual(self.run_cli("version").returncode, 0)
        r = self.run_cli("version", TEST_NETWORK_FAIL="1")
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertEqual(len(self.downloads()), 1)

    def test_arm64_selection(self):
        r = self.run_cli("version", TEST_ARCH="aarch64")
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertIn("PAYLOAD-arm64", r.stdout)
        self.assertTrue(self.cached("arm64").is_file())

    def test_reject_other_os_before_download(self):
        r = self.run_cli(TEST_OS="Darwin")
        self.assertNotEqual(r.returncode, 0)
        self.assertIn('System deployment requires Rocky Linux', r.stderr)
        self.assertEqual(self.downloads(), [])

    def test_reject_other_distribution(self):
        self.os_file.write_text('ID=ubuntu\nVERSION_ID="24.04"\n')
        r = self.run_cli()
        self.assertNotEqual(r.returncode, 0)
        self.assertIn('Only Rocky Linux', r.stderr)
        self.assertEqual(self.downloads(), [])

    def test_reject_other_rocky_version(self):
        self.os_file.write_text('ID=rocky\nVERSION_ID="7.9"\n')
        self.assertNotEqual(self.run_cli().returncode, 0)
        self.assertEqual(self.downloads(), [])

    def test_reject_other_architecture(self):
        self.assertNotEqual(self.run_cli(TEST_ARCH="armv7l").returncode, 0)
        self.assertEqual(self.downloads(), [])

    def test_network_failure_never_executes(self):
        r = self.run_cli(TEST_NETWORK_FAIL="1")
        self.assertNotEqual(r.returncode, 0)
        self.assertNotIn("PAYLOAD", r.stdout)
        self.assertFalse(self.cached().exists())
        self.assertEqual(len(self.downloads()), 1)

    def test_wrong_download_hash_never_executes(self):
        (self.assets / "rockdev-linux-amd64").write_text("#!/bin/sh\necho BAD-PAYLOAD\n")
        r = self.run_cli()
        self.assertNotEqual(r.returncode, 0)
        self.assertNotIn("BAD-PAYLOAD", r.stdout)
        self.assertFalse(self.cached().exists())
        self.assertIn('SHA256 verification failed', r.stderr)

    def test_corrupt_cache_is_verified_and_replaced(self):
        self.assertEqual(self.run_cli().returncode, 0)
        self.cached().write_text("#!/bin/sh\necho BAD-CACHE\n")
        r = self.run_cli()
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertNotIn("BAD-CACHE", r.stdout)
        self.assertEqual(len(self.downloads()), 2)

    def test_corrupt_cache_offline_is_not_executed(self):
        self.assertEqual(self.run_cli().returncode, 0)
        self.cached().write_text("#!/bin/sh\necho BAD-CACHE\n")
        r = self.run_cli(TEST_NETWORK_FAIL="1")
        self.assertNotEqual(r.returncode, 0)
        self.assertNotIn("BAD-CACHE", r.stdout)

    def test_invalid_version_is_not_a_download_url(self):
        (self.repo / "release/VERSION").write_text("../../evil\n")
        self.assertNotEqual(self.run_cli().returncode, 0)
        self.assertEqual(self.downloads(), [])

    def test_duplicate_checksum_is_rejected(self):
        p = self.repo / "release/SHA256SUMS"
        p.write_text(p.read_text() * 2)
        self.assertNotEqual(self.run_cli().returncode, 0)
        self.assertEqual(self.downloads(), [])

    def test_missing_checksum_is_rejected(self):
        (self.repo / "release/SHA256SUMS").write_text("")
        self.assertNotEqual(self.run_cli().returncode, 0)
        self.assertEqual(self.downloads(), [])

    def test_payload_exit_code_is_preserved(self):
        self.assertEqual(self.run_cli(TEST_PAYLOAD_EXIT="19").returncode, 19)

    def test_cached_symlink_is_rejected(self):
        self.assertEqual(self.run_cli().returncode, 0)
        self.cached().unlink()
        try:
            self.cached().symlink_to(self.assets / 'rockdev-linux-amd64')
        except OSError as exc:
            self.skipTest(str(exc))
        r = self.run_cli()
        self.assertNotEqual(r.returncode, 0)
        self.assertIn('symbolic link', r.stderr)
        self.assertNotIn('PAYLOAD', r.stdout)

    def test_cache_directory_symlink_is_rejected(self):
        other = self.root / 'other'
        other.mkdir()
        self.cache.rmdir()
        try:
            self.cache.symlink_to(other, target_is_directory=True)
        except OSError as exc:
            self.skipTest(str(exc))
        r = self.run_cli()
        self.assertNotEqual(r.returncode, 0)
        self.assertIn('symbolic link', r.stderr)
        self.assertEqual(self.downloads(), [])

    def test_concurrent_first_runs(self):
        args = self.command("version")
        a = subprocess.Popen(args, env=self.env, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        b = subprocess.Popen(args, env=self.env, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        for process in [a, b]:
            out, err = process.communicate(timeout=30)
            self.assertEqual(process.returncode, 0, err)
            self.assertIn("PAYLOAD-amd64", out)


if __name__ == "__main__":
    (ROOT / ".cache").mkdir(exist_ok=True)
    unittest.main()
