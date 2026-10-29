"""Build source release archives from a clean, standalone Git HEAD (stdlib only)."""
from __future__ import annotations

import argparse
import gzip
import hashlib
import io
import os
from pathlib import Path, PurePosixPath
import re
import subprocess
import tarfile
import tempfile
import zipfile

ROOT = Path(__file__).resolve().parents[1]
MODULE = "github.com/maxwell2010/decimal-go-sdk"
REQUIRED = {"VERSION", "LICENSE", "README.md", "README.ru.md", "go.mod", "go.sum", "cosmos/LICENSE", "cosmos/go.mod", "cosmos/go.sum"}


def git(root: Path, *args: str) -> bytes:
    return subprocess.run(["git", "-C", str(root), *args], check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE).stdout


def unsafe(name: str) -> bool:
    p = PurePosixPath(name)
    parts = [s.lower() for s in p.parts]
    leaf = parts[-1] if parts else ""
    return (p.is_absolute() or ".." in parts or "\\" in name or not parts
            or any(s in {".tools", "__pycache__", ".pytest_cache", ".cache", "node_modules", ".venv", "venv", "dist", "private", "secrets", "keystore", "wallets"} for s in parts)
            or any(s.startswith(".env") for s in parts)
            or leaf in {"id_rsa", "id_ed25519", "credentials", "credentials.json", "wallet.json", "mnemonic.txt", "private_key", "private-key"}
            or p.suffix.lower() in {".pem", ".key", ".p12", ".pfx", ".db", ".sqlite", ".sqlite3", ".exe", ".dll", ".so", ".dylib", ".o", ".a", ".pyc"})


def build(root: Path, output: Path | None = None) -> list[Path]:
    root = root.resolve()
    output = (output or root / "dist").resolve()
    top = Path(git(root, "rev-parse", "--show-toplevel").decode().strip()).resolve()
    if top != root:
        raise ValueError("release root must be a standalone Git repository root")
    if output.is_relative_to(root) and output != root / "dist":
        raise ValueError("output inside repository must be root/dist")
    if git(root, "status", "--porcelain=v1", "--untracked-files=all"):
        raise ValueError("release repository must be clean, including untracked files")
    # Ignored credentials/caches are also refused, even though archives use HEAD.
    for base, dirs, files in os.walk(root, followlinks=False):
        dirs[:] = [d for d in dirs if not (Path(base) == root and d in {".git", "dist"})]
        for name in dirs + files:
            path = Path(base) / name
            relative = path.relative_to(root).as_posix()
            if path.is_symlink() or unsafe(relative):
                raise ValueError(f"unsafe local release path: {relative}")
    revision = git(root, "rev-parse", "HEAD").decode().strip()
    entries = {}
    for row in git(root, "ls-tree", "-rz", revision).split(b"\0"):
        if not row:
            continue
        meta, raw_name = row.split(b"\t", 1)
        mode, kind, _ = meta.split()
        name = raw_name.decode("utf-8")
        if mode not in {b"100644", b"100755"} or kind != b"blob" or unsafe(name):
            raise ValueError(f"unsafe tracked release path: {name}")
        entries[name] = int(mode, 8) & 0o777
    if not REQUIRED.issubset(entries):
        raise ValueError("required release files missing: " + ", ".join(sorted(REQUIRED - entries.keys())))
    for prefix in ("docs/", "examples/", "abi/"):
        if not any(n.startswith(prefix) for n in entries):
            raise ValueError(f"release must include {prefix}")
    if not any(n.endswith("_test.go") for n in entries):
        raise ValueError("release must include tests")
    # Git archive reads committed blobs, never mutable worktree contents.
    with tarfile.open(fileobj=io.BytesIO(git(root, "archive", "--format=tar", revision)), mode="r:") as source:
        content = {m.name: source.extractfile(m).read() for m in source if m.isfile()}
    if content.keys() != entries.keys():
        raise ValueError("Git archive excludes or changes the tracked release file set")
    version = content["VERSION"].decode().strip()
    if not re.fullmatch(r"v\d+\.\d+\.\d+(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?", version):
        raise ValueError("invalid release VERSION")
    for path, expected in (("go.mod", MODULE), ("cosmos/go.mod", MODULE + "/cosmos")):
        if not re.search(r"(?m)^module\s+" + re.escape(expected) + r"\s*$", content[path].decode()):
            raise ValueError(f"unexpected public module in {path}")
    epoch = int(git(root, "show", "-s", "--format=%ct", revision).strip())
    if git(root, "rev-parse", "HEAD").decode().strip() != revision or git(root, "status", "--porcelain=v1", "--untracked-files=all"):
        raise ValueError("release repository changed during snapshot")
    stem = "decimal-go-sdk-" + version.removeprefix("v")
    output.mkdir(parents=True, exist_ok=True)
    names = [stem + ".zip", stem + ".tar.gz"]
    with tempfile.TemporaryDirectory(dir=output, prefix=".release-") as tmp:
        stage = Path(tmp)
        with zipfile.ZipFile(stage / names[0], "w", compression=zipfile.ZIP_DEFLATED) as archive:
            for name in sorted(content):
                item = zipfile.ZipInfo(stem + "/" + name, (1980, 1, 1, 0, 0, 0))
                item.create_system = 3
                item.external_attr = (0o100000 | entries[name]) << 16
                item.compress_type = zipfile.ZIP_DEFLATED
                archive.writestr(item, content[name])
        with (stage / names[1]).open("wb") as raw:
            with gzip.GzipFile(fileobj=raw, mode="wb", filename="", mtime=epoch) as compressed:
                with tarfile.open(fileobj=compressed, mode="w", format=tarfile.PAX_FORMAT) as archive:
                    for name in sorted(content):
                        item = tarfile.TarInfo(stem + "/" + name)
                        item.mode, item.mtime, item.size = entries[name], epoch, len(content[name])
                        archive.addfile(item, io.BytesIO(content[name]))
        (stage / "SHA256SUMS").write_text("".join(f"{hashlib.sha256((stage / name).read_bytes()).hexdigest()}  {name}\n" for name in names), encoding="utf-8")
        for name in names + ["SHA256SUMS"]:
            os.replace(stage / name, output / name)
    return [output / name for name in names + ["SHA256SUMS"]]


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=ROOT)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    try:
        for path in build(args.root, args.output):
            print(path)
    except (ValueError, subprocess.CalledProcessError, OSError) as exc:
        parser.exit(1, f"release refused: {exc}\n")


if __name__ == "__main__":
    main()
