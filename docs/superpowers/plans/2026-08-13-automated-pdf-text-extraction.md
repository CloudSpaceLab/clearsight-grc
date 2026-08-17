# Automated PDF Text Extraction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract searchable PDF uploads automatically into page-anchored sections and review proposals in the demo.

**Architecture:** A focused Poppler adapter executes `pdfinfo` and `pdftotext` without a shell, inside strict page, time, stdout, stderr, and existing upload budgets. The durable worker image supplies `poppler-utils`; existing section collection and deterministic analysis remain the only proposal path.

**Tech Stack:** Go 1.26, Poppler `pdfinfo`/`pdftotext`, Debian 12 worker runtime, GitHub Actions, Docker Compose deployment.

---

### Task 1: Specify PDF extraction behavior with failing tests

**Files:**
- Create: `internal/documentimport/pdf_extractor_test.go`
- Modify: `internal/documentimport/resource_limits_test.go`

- [ ] Add a fake command runner test proving `pdfinfo` page count and form-feed-separated `pdftotext` output become `Page 1` and `Page 2` sections.
- [ ] Add a test proving a page-anchored section produces a proposal whose `Anchor.Page` matches.
- [ ] Add tests for zero extracted text, unavailable tools, excess pages, and excess command output.
- [ ] Run `go test ./internal/documentimport -run 'TestPDF'` and confirm failures identify the missing PDF adapter.

### Task 2: Implement the bounded Poppler adapter

**Files:**
- Create: `internal/documentimport/pdf_extractor.go`
- Modify: `internal/documentimport/extractor.go`
- Modify: `internal/documentimport/extraction_policy.go`

- [ ] Add `MaxPDFPages` and `PDFExtractionTimeout` to the normalized policy with defaults of 500 pages and 30 seconds.
- [ ] Add a bounded command runner, tool discovery, private temporary file lifecycle, English `pdfinfo` parsing, and bounded diagnostics.
- [ ] Split default Poppler form-feed output into page-numbered sections and report image-only PDFs as OCR-required.
- [ ] Route `.pdf` through the adapter and preserve existing cancellation/resource-error semantics.
- [ ] Run the focused tests until green, then run `go test ./internal/documentimport`.

### Task 3: Exercise the real binary and package the runtime

**Files:**
- Modify: `internal/documentimport/pdf_extractor_test.go`
- Modify: `Dockerfile.worker`
- Modify: `.github/workflows/ci.yml`
- Modify: `deploy/tests/deployment_config_test.py`

- [ ] Generate a minimal two-page searchable PDF in the Go test and exercise real Poppler when present.
- [ ] Install `poppler-utils` in CI before backend tests so the smoke test cannot skip there.
- [ ] Change only the worker runtime to a non-root Debian slim image with `poppler-utils` installed.
- [ ] Add deployment configuration assertions for the worker's Poppler dependency and non-root execution.
- [ ] Run the focused Go and deployment tests.

### Task 4: Update operator documentation

**Files:**
- Modify: `README.md`
- Modify: `docs/engineering/governed-document-imports.md`
- Modify: `docs/architecture/document-import-resource-and-durability-boundary.md`

- [ ] Document automated searchable-PDF extraction, page anchors, budgets, worker dependency, and the explicit OCR gap.
- [ ] Remove statements that all PDF extraction is unimplemented while retaining the accurate malware-scanning and OCR limitations.

### Task 5: Verify, publish, deploy, and prove the demo

**Files:**
- Modify only if verification finds a tested defect.

- [ ] Run `gofmt` and `go test ./...`.
- [ ] Run `go test -tags postgres ./...` and `go test -p 1 -tags 'postgres postgresintegration' ./internal/...` against PostgreSQL 18.
- [ ] Run `go vet ./...`, the deployment configuration tests, and `git diff --check`.
- [ ] Commit the implementation, push `main`, and wait for CI and auto-deploy success.
- [ ] Upload fresh copies of the official Federal Reserve and Bank of England PDFs, poll processing, and verify `EXTRACTED`, page sections, `REVIEW_REQUIRED`, and positive proposal page anchors through the live API.
