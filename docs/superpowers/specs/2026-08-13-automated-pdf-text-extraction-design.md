# Automated PDF Text Extraction Design

## Outcome

Searchable PDF uploads are extracted automatically by the document-import worker and produce reviewable compliance proposals with exact page anchors. Image-only or otherwise non-extractable PDFs remain stored, but are reported explicitly as requiring OCR; the system never invents text.

## Root cause

The current extractor returns `UNSUPPORTED` for every `.pdf` file before examining its contents. The original artifact is stored correctly, but no text, page sections, or proposals can be produced.

## Considered approaches

1. **Poppler command-line tools in the worker (selected).** `pdfinfo` provides a cheap page-count gate and `pdftotext` provides mature UTF-8 extraction with form-feed page boundaries. The tools run as child processes without a shell, under the worker's non-root account, with context cancellation, a hard timeout, and bounded output.
2. **A pure-Go PDF parser.** This keeps the runtime image smaller, but the available parsers have materially weaker support for real-world font encodings and layout. That is a poor trade for a regulator-document demo.
3. **Python extraction or OCR.** This adds a Python environment and larger memory/runtime dependencies. Full OCR also needs page rasterization and Tesseract. It is deferred because searchable regulator PDFs are the critical path and OCR would increase server load substantially.

## Architecture

`ExtractWithPolicy` delegates PDFs to a focused Poppler adapter. The adapter writes the already-bounded upload bytes to a private temporary file, discovers `pdfinfo` and `pdftotext`, validates the page count, then executes `pdftotext -layout -enc UTF-8 -eol unix <file> -`. Poppler inserts form-feed page breaks by default; each non-empty page becomes one `Section` with `Page` set to its one-based source page.

The existing deterministic analyzer consumes those sections unchanged. Its existing anchor propagation then copies `Section.Page` to every proposal, so review records retain exact page provenance.

Only the worker runtime receives Poppler because durable PostgreSQL imports are processed there. The API continues to enqueue imports and remains on its minimal distroless image.

## Resource and security boundaries

- Keep the existing 20 MiB upload/read cap.
- Add a 500-page PDF cap before text conversion.
- Add a 30-second extraction deadline per PDF, bounded by any earlier caller deadline.
- Cap Poppler stdout slightly above the retained-text budget; exceeding it fails truthfully with `Extraction resource limit exceeded`.
- Cap diagnostic stderr so malformed input cannot create unbounded logs.
- Invoke binaries directly through `exec.CommandContext`; never interpolate a shell command.
- Use a private temporary directory/file and remove it after extraction.
- Run the worker and Poppler as an unprivileged user.

## Status and error semantics

- Searchable text found: `EXTRACTED`, method `POPPLER_TEXT_V1`.
- No machine-readable text: `UNSUPPORTED`, method `POPPLER_TEXT_V1`, with an explicit OCR-required limitation and no proposals.
- Poppler unavailable: `UNSUPPORTED`, method `NONE`, with an installation limitation. The production image and CI install it, so this is a truthful development fallback.
- Malformed, encrypted-without-password, timed-out, page-limit, or output-limit PDF: `FAILED`, method `POPPLER_TEXT_V1`, with a bounded diagnostic and no proposals.

## Test and demo acceptance

- Unit tests prove page boundaries become page-numbered sections and proposal anchors.
- Unit tests prove image-only output is explicit and proposal-free.
- Unit tests prove page and output budgets fail safely.
- A real Poppler smoke test exercises a generated two-page searchable PDF when the tools are installed; CI installs `poppler-utils`, so it must run there.
- Backend, PostgreSQL, deployment configuration, formatting, and vet checks remain green.
- After deployment, fresh uploads of the official Federal Reserve cybersecurity report and Bank of England operational-resilience statement must reach `EXTRACTED`; at least one proposal must carry a positive page anchor.

## Non-goals

- OCR for scanned/image-only PDFs.
- Table reconstruction, images, signatures, or PDF form extraction.
- Automatic acceptance of proposals or legal interpretation.
