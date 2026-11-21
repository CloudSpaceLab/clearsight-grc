# Documents sample preview — 21 November 2026

Reviewer: Codex agent `/root`. The evidence fixture now draws a labelled 600 × 760 PNG containing sample access-review details; it replaces the previous one-pixel image. The browser check asserts the image has decoded at that size before capture, and checks horizontal overflow at 1440, 1280, 1024, 768 and 390 pixels. Inspected the 1440 and 390 pixel captures. The preview is legible at desktop size; small text is difficult to read at 390 pixels and a larger-view action remains an improvement to investigate.

This synthetic PNG exercises the existing image preview only. It does not verify PDF or other file rendering, real storage, authorization, dark mode, keyboard access or full v2.15 acceptance. The earlier one-pixel baseline remains in `../2026-11-21-documents-domain/`.
