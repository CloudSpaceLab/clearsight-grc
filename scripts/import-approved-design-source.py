#!/usr/bin/env python3
"""Verify and import original, reference-only ClearSight design source bytes."""
from __future__ import annotations

import hashlib
import io
import json
import pathlib
import sys
import zipfile

ROOT = pathlib.Path(__file__).resolve().parents[1]
TARGET = ROOT / "docs/design/references/clearsight-2026-10/source"
ARCHIVE_SHA256 = "65dcc0e3f26e9d41e38efd1ae1d571f5db0f0582a2d561f8c942b25817a5f6e5"
SPEC_SHA256 = "13e5531114779672c7920e939206db1e383be664af67718b617251efaf496c17"
SUFFIXES = {".html", ".css", ".js", ".md", ".txt"}
EXPECTED_FILES, EXPECTED_PACKAGES, EXPECTED_BYTES = 96, 18, 3_768_146


def safe_path(path: str) -> pathlib.PurePosixPath:
    result = pathlib.PurePosixPath(path)
    if (not path or path.startswith("/") or "\\" in path
            or any(part in (".", "..", "") for part in path.split("/"))):
        raise ValueError(f"Unsafe source path: {path!r}")
    return result


def validated_entries(data: bytes) -> tuple[list[tuple[str, bytes]], dict]:
    if hashlib.sha256(data).hexdigest() != ARCHIVE_SHA256:
        raise ValueError("Original source archive SHA-256 mismatch")
    records: list[dict[str, str | int]] = []
    entries: list[tuple[str, bytes]] = []
    with zipfile.ZipFile(io.BytesIO(data)) as outer:
        packages = [entry for entry in outer.infolist() if not entry.is_dir()]
        if len(packages) != EXPECTED_PACKAGES or any(not entry.filename.endswith(".zip") for entry in packages):
            raise ValueError("Unexpected nested design package inventory")
        for package in packages:
            root = safe_path(package.filename).stem
            with zipfile.ZipFile(io.BytesIO(outer.read(package))) as inner:
                for item in inner.infolist():
                    if item.is_dir() or pathlib.PurePosixPath(item.filename).suffix.lower() not in SUFFIXES:
                        continue
                    relative = str(safe_path(root) / safe_path(item.filename))
                    raw = inner.read(item)
                    raw.decode("utf-8")
                    entries.append((relative, raw))
                    records.append({"path": relative, "bytes": len(raw), "sha256": hashlib.sha256(raw).hexdigest()})
    if len(entries) != EXPECTED_FILES or sum(len(data) for _, data in entries) != EXPECTED_BYTES:
        raise ValueError("Extracted source count or byte total mismatch")
    if len({path for path, _ in entries}) != len(entries):
        raise ValueError("Duplicate source path")
    canonical = next((record for record in records if record["path"] ==
        "clearsight-unified-design-system-v1/clearsight-design-system/clearsight-design-system-spec.md"), None)
    if not canonical or canonical["sha256"] != SPEC_SHA256:
        raise ValueError("Canonical design specification checksum mismatch")
    return entries, {"source_archive": "complete designs.zip", "source_archive_sha256": ARCHIVE_SHA256, "files": records}


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit("Usage: import-approved-design-source.py <original complete designs.zip>")
    entries, manifest = validated_entries(pathlib.Path(sys.argv[1]).read_bytes())
    # Verify all collisions before the first write.
    for relative, raw in entries:
        path = TARGET / relative
        if path.exists() and path.read_bytes() != raw:
            raise SystemExit(f"Existing reference source differs: {relative}")
    manifest_bytes = (json.dumps(manifest, ensure_ascii=False, indent=2) + "\n").encode("utf-8")
    if (TARGET / "MANIFEST.json").exists() and (TARGET / "MANIFEST.json").read_bytes() != manifest_bytes:
        raise SystemExit("Existing source manifest differs")
    for relative, raw in entries:
        path = TARGET / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(raw)
    (TARGET / "MANIFEST.json").write_bytes(manifest_bytes)
    print(f"Verified {len(entries)} reference files across {EXPECTED_PACKAGES} packages; {EXPECTED_BYTES} original bytes")


if __name__ == "__main__":
    main()
