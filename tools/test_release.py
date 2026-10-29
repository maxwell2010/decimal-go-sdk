"""Release archive tests use disposable local Git repositories, never uploads."""
import hashlib
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest
import zipfile

import release


class ReleaseTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name) / "sdk"
        self.root.mkdir()
        self.git("init", "-q")
        self.git("config", "user.name", "Release Test")
        self.git("config", "user.email", "test@example.invalid")
        data = {n: "fixture\n" for n in release.REQUIRED}
        data.update({"VERSION": "v0.1.0-preview.1\n", "go.mod": "module " + release.MODULE + "\n", "cosmos/go.mod": "module " + release.MODULE + "/cosmos\n", ".gitignore": "/dist/\n.cache/\n.env\n", "docs/sources.json": "{}\n", "abi/Token.json": "[]\n", "sdk_test.go": "package decimal\n", "examples/offline/main.go": "package main\n"})
        for name, text in data.items():
            self.write(name, text)
        self.commit()

    def git(self, *args):
        return subprocess.run(["git", "-C", str(self.root), *args], check=True, capture_output=True).stdout

    def write(self, name, text):
        p = self.root / name
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text(text, encoding="utf-8")

    def commit(self):
        self.git("add", "-A")
        self.git("commit", "-qm", "fixture")

    def test_complete_reproducible_archives_and_checksums(self):
        paths = release.build(self.root)
        before = [p.read_bytes() for p in paths]
        lines = paths[2].read_text().splitlines()
        for line, path in zip(lines, paths[:2]):
            self.assertEqual(line, hashlib.sha256(path.read_bytes()).hexdigest() + "  " + path.name)
        prefix = "decimal-go-sdk-0.1.0-preview.1/"
        with zipfile.ZipFile(paths[0]) as archive:
            zip_content = {n.removeprefix(prefix): archive.read(n) for n in archive.namelist()}
        with tarfile.open(paths[1]) as archive:
            tar_content = {m.name.removeprefix(prefix): archive.extractfile(m).read() for m in archive if m.isfile()}
        self.assertEqual(zip_content, tar_content)
        for required in ("sdk_test.go", "docs/sources.json", "examples/offline/main.go", "cosmos/LICENSE", "abi/Token.json"):
            self.assertIn(required, zip_content)
        self.assertFalse(any(n.startswith("dist/") for n in zip_content))
        self.assertEqual(before, [p.read_bytes() for p in release.build(self.root)])

    def test_dirty_and_untracked_refused(self):
        self.write("README.md", "dirty")
        with self.assertRaisesRegex(ValueError, "clean"):
            release.build(self.root)
        self.git("checkout", "--", "README.md")
        self.write("untracked.txt", "unknown")
        with self.assertRaisesRegex(ValueError, "clean"):
            release.build(self.root)

    def test_ignored_credentials_and_cache_refused(self):
        for name in (".env", ".cache/token"):
            self.write(name, "private")
            with self.assertRaisesRegex(ValueError, "unsafe local"):
                release.build(self.root)
            (self.root / name).unlink()
            if name.startswith(".cache/"):
                (self.root / ".cache").rmdir()

    def test_tracked_binary_and_symlink_modes_refused(self):
        self.write("sdk.exe", "binary")
        self.commit()
        with self.assertRaisesRegex(ValueError, "unsafe"):
            release.build(self.root)
        (self.root / "sdk.exe").unlink()
        self.commit()
        self.write("linked", "README.md")
        self.commit()
        blob = self.git("rev-parse", "HEAD:linked").decode().strip()
        self.git("update-index", "--cacheinfo", "120000," + blob + ",linked")
        self.git("commit", "-qm", "unsafe link")
        # A symlink-capable Git checkout may report the regular worktree as dirty;
        # either the clean gate or mode gate must refuse this snapshot.
        with self.assertRaises(ValueError):
            release.build(self.root)

    def test_wrong_module_and_parent_checkout_refused(self):
        self.write("cosmos/go.mod", "module decimal.local/sdk/cosmos\n")
        self.commit()
        with self.assertRaisesRegex(ValueError, "public module"):
            release.build(self.root)
        with self.assertRaisesRegex(ValueError, "standalone"):
            release.build(self.root / "cosmos")


if __name__ == "__main__":
    unittest.main()
