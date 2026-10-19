#!/usr/bin/env python3
"""Verify the original ClearSight screenshot assets and optionally extract them.

The 96 source text files are already imported. This handles the distinct binary
screenshots without silently embedding large files in ordinary code tranches.
"""
from __future__ import annotations

import argparse
import hashlib
import io
import json
from pathlib import Path, PurePosixPath
import zipfile

ARCHIVE_SHA256 = "65dcc0e3f26e9d41e38efd1ae1d571f5db0f0582a2d561f8c942b25817a5f6e5"
EXPECTED_PACKAGES = 18
EXPECTED_IMAGES = 143
EXPECTED_BYTES = 81_005_321
SUFFIXES = {".png", ".jpg", ".jpeg", ".webp", ".svg"}


def safe_path(raw: str) -> PurePosixPath:
    path = PurePosixPath(raw)
    if not raw or raw.startswith("/") or "\\" in raw or any(part in {"", ".", ".."} for part in raw.split("/")):
        raise ValueError(f"Unsafe design asset path: {raw!r}")
    return path


def index_assets(archive: bytes, destination: Path | None) -> list[dict[str, str | int]]:
    if hashlib.sha256(archive).hexdigest() != ARCHIVE_SHA256:
        raise ValueError("Original approved design archive hash mismatch")
    records: list[dict[str, str | int]] = []
    seen: set[str] = set()
    with zipfile.ZipFile(io.BytesIO(archive)) as outer:
        packages = [item for item in outer.infolist() if not item.is_dir()]
        if len(packages) != EXPECTED_PACKAGES or any(not item.filename.endswith(".zip") for item in packages):
            raise ValueError("Original nested ZIP inventory mismatch")
        for package in packages:
            package_root = safe_path(package.filename).stem
            with zipfile.ZipFile(io.BytesIO(outer.read(package))) as inner:
                for item in inner.infolist():
                    if item.is_dir() or PurePosixPath(item.filename).suffix.lower() not in SUFFIXES:
                        continue
                    relative = str(PurePosixPath(package_root) / safe_path(item.filename))
                    if relative in seen:
                        raise ValueError(f"Duplicate visual asset: {relative}")
                    seen.add(relative)
                    raw = inner.read(item)
                    records.append({
                        "path": relative,
                        "bytes": len(raw),
                        "sha256": hashlib.sha256(raw).hexdigest(),
                    })
                    if destination is not None:
                        target = destination / relative
                        if target.exists() and target.read_bytes() != raw:
                            raise ValueError(f"Existing visual source differs: {relative}")
                        target.parent.mkdir(parents=True, exist_ok=True)
                        target.write_bytes(raw)
    if len(records) != EXPECTED_IMAGES or sum(int(x["bytes"]) for x in records) != EXPECTED_BYTES:
        raise ValueError("Visual source image count or byte total mismatch")
    return sorted(records, key=lambda item: str(item["path"]))


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("archive", type=Path, help="Original 18-package complete designs.zip")
    parser.add_argument("--manifest", type=Path, help="Write the checksum inventory as JSON")
    parser.add_argument("--extract-dir", type=Path, help="Extract original screenshot files for source-side review")
    args = parser.parse_args()
    # Verify the complete ZIP and inventory before extracting files.
    raw = args.archive.read_bytes()
    if hashlib.sha256(raw).hexdigest() != ARCHIVE_SHA256:
        raise SystemExit("Source ZIP failed integrity check")
    records = index_assets(raw, None)
    if args.extract_dir is not None:
        index_assets(raw, args.extract_dir)
    if args.manifest is not None:
        args.manifest.parent.mkdir(parents=True, exist_ok=True)
        args.manifest.write_text(json.dumps({
            "source_archive_sha256": ARCHIVE_SHA256,
            "source_images": records,
        }, indent=2) + "\n", encoding="utf-8")
    print(f"Verified {len(records)} original screenshots ({sum(int(x['bytes']) for x in records):,} bytes) across {EXPECTED_PACKAGES} packages")


if __name__ == "__main__":
    main()
