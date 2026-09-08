# Demo document samples implementation plan

> **For agentic workers:** Use subagent-driven-development or executing-plans. Follow test-driven-development for application behavior, and render document artifacts before accepting them.

**Goal:** Show readable fictional submitted documents and an explicitly simulated scan treatment without claiming antivirus inspection or weakening genuine-upload controls.

**Architecture:** Ship a small immutable file pack with a digest/size/media-type manifest. A demo-only content capability applies only to those exact unscanned bytes after the existing occurrence authorization and complete object integrity check. Keep stored artifact status, inspection jobs and review acceptance unchanged. Seed through the current vendor assessment and respondent services, not synthetic rows in the document read model.

## Task 1: Readable immutable sample files

- [ ] Author compact fictional Northstar service documents: security self-declaration, prior declaration, insurance schedule with a deliberately different vendor name, recovery plan, registered-office statement image and subprocessor register. Use dates and named fictional contact roles, no copied confidential data, fake certification seals or signatures. Every file states that it is sample data and cannot establish real compliance.
- [ ] Include current, approaching-expiry, expired and contradiction inputs. Keep scenario dates explicit rather than silently refreshing a document's issue date. Missing evidence is an omitted optional answer, not an empty uploaded file.
- [ ] Use the PDF/DOCX/spreadsheet artifact workflows to render and inspect every authored page or sheet. Commit the small final assets and reproducible authoring sources, not rendering intermediates.
- [ ] Add a pure Go embedded manifest package. Tests require actual size, digest, correct media type, no missing/extra manifest entries and rejection of changed bytes or same-name substitutions. Manifest matching is not antivirus inspection.

The first pack uses PDF for the recovery plan. The Windows dependency bundle has no LibreOffice executable, and `render_docx.py` failed at executable discovery. The unverified Word draft was moved to local QA storage and is not shipped or allowlisted. Word fixture verification remains open; the existing Word filter and download support are unchanged.

## Task 2: Protected demo preview and truthful presentation

- [ ] Add RED tests around the evidence/content boundary: a known sample remains STORED_UNSCANNED; default/non-demo open rejects it; configured demo opening verifies every stored byte; altered/truncated/appended bytes, wrong media type, quarantine/deletion and unknown files remain unavailable. Ordinary OpenArtifact behavior is unchanged.
- [ ] Add one explicit demo-sample opening method that reuses the existing private integrity routine; enable only through validated demo configuration. No general allow-unscanned option or alternate public file route.
- [ ] Decorate already-authorized DocumentOccurrence results with an optional demo-preview capability using the same manifest and demo flag. Ignore any incoming/client-provided capability; recompute it at the server content boundary. Preserve exact tenant/entity/principal/revision/membership checks and protected headers.
- [ ] Share preview eligibility between frontend preview/download components. Show “Demo check complete” and “No antivirus scan was performed” for eligible sample files. Keep unknown unscanned files pending and blocked. Do not add artificial timers or green antivirus claims. Preview does not change review acceptance.
- [ ] Cover production/demo refusal, scope revocation, content integrity, UI warning and no-fetch/no-download on blocked documents. Update product/architecture/acceptance documentation with the simulation limitation.

## Task 3: Connected persisted responses

- [ ] Extend only the non-production sample installer. Require demo mode and a durable configured artifact store. Preserve existing scoring fixtures and all genuine records.
- [ ] Use a dedicated source-marked sample vendor and governed form so reruns cannot overwrite an operator's existing vendor assessment. Establish the canonical assessment/request origin via existing services and upload through the respondent session before submitting typed answers.
- [ ] Use exact indexed sample identifiers and existing idempotent workflow episode keys. On partial reruns inspect the exact request/workspace/revision, resume its existing route where allowed, and do not duplicate submissions or replace a person's edited answers. No SMTP delivery or logging of route/session secrets.
- [ ] Include a previous and replacement response through normal versioned response/reopen or clarification paths. Last-submitted values come from actual stored submissions; authored issue/expiry dates remain separate.
- [ ] Test repeat and interrupted installation, production refusal, unrelated-data preservation, and the same stored occurrence from response detail, Forms Documents and vendor Documents. If an existing workflow cannot support a scenario, document the exact missing capability instead of manufacturing its completion.

## Task 4: Review and deployment proof

- [ ] Specification review followed by quality/security review. Run full Go/PostgreSQL and web gates, copy quality, runtime fixture isolation and browser evidence. Inspect desktop/narrow light/dark previews and blocked states.
- [ ] Merge only exact-head green CI and use normal demo deployment. Run installer only against the owned non-production app with no real recipients. Verify stored bytes, manifest digest, demo warning, vendor/Forms access and unchanged blocked genuine-upload behavior on the hosted revision.
- [ ] Record receipts in #147/#200 and keep live antivirus enablement, real document validation/acceptance and remaining vendor handoffs open until independently verified.
