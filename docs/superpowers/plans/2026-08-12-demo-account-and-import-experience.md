# Demo Account and Document Import Experience Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make demo account selection and switching low-clutter, and prove the document import workflow end to end with official regulatory PDFs.

**Architecture:** Keep authentication and document-import APIs unchanged. Reduce frontend information density through progressive disclosure, retain full application unmount on role switch, and treat PDF storage as a precise terminal outcome until an approved extractor exists.

**Tech Stack:** React 19, TypeScript, Vitest, Testing Library, axe-core, Go 1.26, PostgreSQL 18, GitHub Actions, Docker Compose.

---

## File structure

- `web/src/components/DemoLoginPage.tsx`: compact role selection and one-action sign-in.
- `web/src/components/DemoAuthGate.tsx`: signed-in account menu and secure switch transition.
- `web/src/demo-login.css`: responsive chooser/menu presentation.
- `web/src/components/DocumentImportWorkspace.tsx`: progressive intake disclosure and precise import outcomes.
- `web/src/document-import.css`: calm review-first layout and responsive intake.
- React tests: interaction, state and accessibility regression proof.
- Existing Go document-import tests: durable, bounded, tenant-safe PDF storage contract.

### Task 1: Compact role selection and account switching

- [ ] Add failing React tests that require one visible role summary per account, no repeated password text, a `Continue as <role>` action, and an expandable `Viewing as` account control after login.
- [ ] Run `npm test -- Accessibility.test.tsx DemoAuthGate.test.tsx` and confirm failures are caused by the old credential grid and floating switch button.
- [ ] Change `DemoLoginPage` to track the selected account, call `loginDemo` with its supplied credentials, and render email only as secondary read-only context.
- [ ] Change `DemoAuthGate` to retain the authenticated actor label from runtime context, expose an accessible account menu, and keep logout-before-unmount behavior.
- [ ] Replace the old two-column credential and floating-pill CSS with responsive compact list, summary and popover styles using existing semantic tokens.
- [ ] Re-run focused tests and require them to pass.

### Task 2: Review-first import workspace

- [ ] Add failing tests requiring collapsed intake when imports exist, open intake for an empty list, visible supported-format/20 MiB help, retained inputs after an upload error, and explicit PDF stored-only copy.
- [ ] Run `npm test -- DocumentImportWorkspace.test.tsx` and verify the new assertions fail against the permanent intake form and current terminal wording.
- [ ] Add an intake disclosure state derived from the initial list result; do not auto-close when an upload fails.
- [ ] Keep file, purpose and source type explicit, disable the primary action while uploading, and focus/reveal intake when validation fails.
- [ ] Render PDF `UNSUPPORTED`/`UNAVAILABLE` as `Original stored · text review unavailable`, retaining all limitations and original details.
- [ ] Update CSS so the review list/detail is primary and intake is a bounded disclosure panel.
- [ ] Re-run focused tests and axe checks.

### Task 3: Backend and build verification

- [ ] Run `go test ./internal/documentimport ./internal/httpapi` and retain the existing PDF unsupported, maximum-size, digest, tenant and durable receipt assertions.
- [ ] Run all web tests, TypeScript checking and production build.
- [ ] Run `go test ./...`, `go test -tags postgres ./...`, deployment configuration tests and `git diff --check`.
- [ ] Inspect the built default and `?demo=0` presentations for unchanged feature gating.

### Task 4: Official-document live acceptance

- [ ] Download two manageable PDFs from official NDPC/CBN pages into a temporary workspace and validate the `%PDF-` signature, page count, extractability and SHA-256 locally.
- [ ] Authenticate as the demo GRC Administrator, submit each with `source_type=REGULATORY` and an explicitly non-legal-review purpose, and require HTTP `201`.
- [ ] Poll each returned ID until processing is terminal. Assert filename, byte size, SHA-256, `extraction_status=UNSUPPORTED`, `analysis_status=UNAVAILABLE`, zero proposals and explicit limitations.
- [ ] Assert actor-scoped list/detail access and an unauthorized-scope negative read.
- [ ] Record exact source URLs and live IDs in the delivery report.

### Task 5: Commit, deploy and post-deploy verification

- [ ] Commit the frontend, tests and documentation with focused messages and push `main`.
- [ ] Require the pushed SHA's CI and automatic deployment runs to succeed.
- [ ] Require live readiness, default page, `?demo=0`, demo login/account switch and Imports API checks to pass.
- [ ] Report the deployed SHA, workflow links, live document IDs, terminal states and remaining PDF/OCR production boundary.

