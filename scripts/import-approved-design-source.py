#!/usr/bin/env python3
"""Import the verified ClearSight reference-only design package.

Usage:
  python3 scripts/import-approved-design-source.py /path/to/clearsight-approved-design-source.zip

The input is the prepared package with MANIFEST.json, not the original nested ZIP.
This script never imports prototype files into the runtime web source tree.
"""
from __future__ import annotations

import hashlib
import json
import pathlib
import sys
import zipfile

ROOT = pathlib.Path(__file__).resolve().parents[1]
TARGET = ROOT / "docs/design/references/clearsight-2026-10/source"


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit("Expected one prepared source ZIP path")
    archive = pathlib.Path(sys.argv[1])
    with zipfile.ZipFile(archive) as source:
        manifest = json.loads(source.read("MANIFEST.json"))
        records = manifest["files"]
        expected = {record["path"]: record for record in records}
        names = {name for name in source.namelist() if not name.endswith("/")}
        if names != set(expected) | {"MANIFEST.json"}:
            raise SystemExit("Source entries differ from the manifest")
        validated: list[tuple[pathlib.Path, bytes]] = []
        for name, record in expected.items():
            relative = pathlib.PurePosixPath(name)
            if relative.is_absolute() or ".." in relative.parts or not relative.parts:
                raise SystemExit(f"Unsafe source path: {name}")
            data = source.read(name)
            if len(data) != record["bytes"] or hashlib.sha256(data).hexdigest() != record["sha256"]:
                raise SystemExit(f"Source checksum mismatch: {name}")
            validated.append((TARGET.joinpath(*relative.parts), data))
        for destination, data in validated:
            destination.parent.mkdir(parents=True, exist_ok=True)
            if destination.exists() and destination.read_bytes() != data:
                raise SystemExit(f"Existing source differs: {destination}")
            destination.write_bytes(data)
        (TARGET / "MANIFEST.json").write_bytes(source.read("MANIFEST.json"))
    print(f"Verified and imported {len(validated)} original reference files into {TARGET}")


if __name__ == "__main__":
    main()
