# Approved screenshot assets

The October source archive is [complete designs.zip](https://drive.google.com/file/d/1y1Ij8zyfZKUKfcxp68T9FmJHabyWREZ_/view?usp=drivesdk), verified by SHA-256 `65dcc0e3f26e9d41e38efd1ae1d571f5db0f0582a2d561f8c942b25817a5f6e5`.

The initial importer preserved **96 text files**. It did **not** import the original **143 image references (81,005,321 uncompressed bytes)** distributed through 18 nested ZIP packages. These include desktop/mobile and open-state screenshots for Assessment, Program, Frameworks, Vendor, Risk & Losses, Reports, Work, Privacy, Setup, Search, Notifications, Audit, Controls, People, Automation and Copilot.

To reproduce and check the original image inventory on a developer machine:

```bash
python3 scripts/index-approved-visual-assets.py "complete designs.zip" \
  --manifest docs/design/references/clearsight-2026-10/visual-source-manifest.json \
  --extract-dir docs/design/references/clearsight-2026-10/visual-source
```

The command checks the outer archive checksum, nested package count, path safety, visual count, original byte count and individual file SHA-256. It refuses to overwrite different bytes. Extracted images are reference-only, not UI assets.

**Do not commit 81 MB of unreviewed binary content into a single large PR.** Import approved review states in small asset-only tranches or use an organizational LFS/artifact store, retaining the manifest and exact paths. The checked archive remains the authority until every required reference is accessible.

No screenshot produced by functional CI constitutes source-to-runtime parity. The approved [visual reset](VISUAL-PARITY-RESET-2026-10-19.md) remains at **0/19 accepted screens**.
