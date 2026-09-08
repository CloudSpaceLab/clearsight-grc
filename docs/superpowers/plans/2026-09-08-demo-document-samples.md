# Demo document samples implementation plan

> **For agentic workers:** Use subagent-driven-development or executing-plans. Follow test-driven-development for application behavior, and render document artifacts before accepting them.

**Goal:** Show readable fictional submitted documents and an explicitly simulated scan treatment without claiming antivirus inspection or weakening genuine-upload controls.

**Architecture:** Ship a small immutable file pack with a digest/size/media-type manifest. A demo-only content capability applies only to those exact unscanned bytes after the existing occurrence authorization and complete object integrity check. Keep stored artifact status, inspection jobs and review acceptance unchanged. Seed through the current vendor assessment and respondent services, not synthetic rows in the document read model.

## Task 1: Readable immutable sample files

- [x] Author compact fictional Northstar service documents: security self-declaration, prior declaration, insurance schedule with a deliberately different vendor name, recovery plan, registered-office statement image and subprocessor register. Use dates and named fictional contact roles, no copied confidential data, fake certification seals or signatures. Every file states that it is sample data and cannot establish real compliance.
- [x] Include current, approaching-expiry, expired and contradiction inputs. Keep scenario dates explicit rather than silently refreshing a document's issue date. Missing evidence is an omitted optional answer, not an empty uploaded file.
- [x] Use the PDF/DOCX/spreadsheet artifact workflows to render and inspect every shipped page or sheet, with the Word limitation below. Commit the small final assets and reproducible authoring sources, not rendering intermediates.
- [x] Add a pure Go embedded manifest package. Tests require actual size, digest, correct media type, no missing/extra manifest entries and rejection of changed bytes or same-name substitutions. Manifest matching is not antivirus inspection.

The first pack uses PDF for the recovery plan. The Windows dependency bundle has no LibreOffice executable, and `render_docx.py` failed at executable discovery. The unverified Word draft was moved to local QA storage and is not shipped or allowlisted. Word fixture verification remains open; the existing Word filter and download support are unchanged.

Task 1 receipts: commits `99c34e27` and `ad306e0f`; six immutable PDF/PNG/XLSX assets, independently approved specification and quality reviews, fresh `go test ./internal/demodocuments -count=1` passed. The binary Git attributes preserve the reviewed content hashes across Windows/Linux checkouts. This foundation alone does not enable application previews or establish document legitimacy.

## Task 2: Protected demo preview and truthful presentation

- [ ] Add RED tests around the evidence/content boundary: a known sample remains STORED_UNSCANNED; default/non-demo open rejects it; configured demo opening verifies every stored byte; altered/truncated/appended bytes, wrong media type, quarantine/deletion and unknown files remain unavailable. Ordinary OpenArtifact behavior is unchanged.
- [ ] Add one explicit demo-sample opening method that reuses the existing private integrity routine; enable only through validated demo configuration. No general allow-unscanned option or alternate public file route.
- [ ] Decorate already-authorized DocumentOccurrence results with an optional demo-preview capability using the same manifest and demo flag. Ignore any incoming/client-provided capability; recompute it at the server content boundary. Preserve exact tenant/entity/principal/revision/membership checks and protected headers.
- [ ] Share preview eligibility between frontend preview/download components. Show “Demo check complete” and “No antivirus scan was performed” for eligible sample files. Keep unknown unscanned files pending and blocked. Do not add artificial timers or green antivirus claims. Preview does not change review acceptance.
- [ ] Cover production/demo refusal, scope revocation, content integrity, UI warning and no-fetch/no-download on blocked documents. Update product/architecture/acceptance documentation with the simulation limitation.

## Task 3: Connected persisted responses

Implementation boundaries from the service audit:

- Add an explicit document-sample-only invocation to the existing installer. It must not replay the unrelated reference projection maintainers or install new authority routes. Default deployment seeding remains unchanged; run document installation explicitly after the normal API/worker are ready.
- Resolve the existing sample principals and legal-entity membership, then use current effective command authority in enforce mode. Use the governed library form commands with separate maker and checker, not the legacy form helper that lacks this guard. Recheck the exact stored contract on a partial rerun; an operator pause or changed form stops continuation.
- Serialize sample installers with a scoped advisory lock, and bound exact artifact/revision reads to the expected fixture population plus one. Compare all stored answers and artifact membership before resuming. No direct material SQL writes, generalized seed framework, or silent overwrite of changed sample records.
- Poll only the exact sample assessment setup result. The normal worker performs setup; the installer must not claim or run unrelated maintenance jobs on the shared host. Invitation delivery stays disabled for the `.invalid` sample audience, and route/session secrets never enter receipts.

- [ ] Extend only the non-production sample installer. Require demo mode and a durable configured artifact store. Preserve existing scoring fixtures and all genuine records.
- [ ] Use a dedicated source-marked sample vendor and governed form so reruns cannot overwrite an operator's existing vendor assessment. Establish the canonical assessment/request origin via existing services and upload through the respondent session before submitting typed answers.
- [ ] Use exact indexed sample identifiers and existing idempotent workflow episode keys. On partial reruns inspect the exact request/workspace/revision, resume its existing route where allowed, and do not duplicate submissions or replace a person's edited answers. No SMTP delivery or logging of route/session secrets.
- [ ] Include a previous and replacement response through normal versioned response/reopen or clarification paths. Last-submitted values come from actual stored submissions; authored issue/expiry dates remain separate.
- [ ] Test repeat and interrupted installation, production refusal, unrelated-data preservation, and the same stored occurrence from response detail, Forms Documents and vendor Documents. If an existing workflow cannot support a scenario, document the exact missing capability instead of manufacturing its completion.

### Confirmed assessment-submission limitation

At baseline `60a6a606`, the workspace submit transaction in `internal/evidence/response_workspace_postgres.go` emits `FORM_DISTRIBUTION / FORM_RESPONSE_REVISION_SUBMITTED`. The assessment consumer in `internal/thirdparty/assessment_consumer.go` accepts only `EVIDENCE_REQUEST / EvidenceResponseSubmitted`, emitted by the legacy submission path. No translation was found. Consequently a workspace response can be stored and visible in Documents while its linked assessment remains collecting. In addition, the assessment resolver and submitted reaction currently require `COLLECTING`, so simply forwarding an event does not safely support a second response revision.

Track the workflow correction under #139/#80 separately from fictional file installation. Acceptance must cover the exact assessment/request/form link, transactional event delivery, inbox retry, out-of-order and duplicate revisions, current-versus-superseded submissions, and review/activation freshness. Do not rewrite assessment state in seed SQL, synthesize an approval, or claim the sample journey completed. The sample installer may demonstrate actual immutable responses and their documents while reporting this limitation explicitly.

## Task 4: Review and deployment proof

- [ ] Specification review followed by quality/security review. Run full Go/PostgreSQL and web gates, copy quality, runtime fixture isolation and browser evidence. Inspect desktop/narrow light/dark previews and blocked states.
- [ ] Merge only exact-head green CI and use normal demo deployment. Run installer only against the owned non-production app with no real recipients. Verify stored bytes, manifest digest, demo warning, vendor/Forms access and unchanged blocked genuine-upload behavior on the hosted revision.
- [ ] Record receipts in #147/#200 and keep live antivirus enablement, real document validation/acceptance and remaining vendor handoffs open until independently verified.
