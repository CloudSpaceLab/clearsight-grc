# Submitted document browser — implementation decision and proof

## Decision

The user rejected the earlier concept and requested familiar macOS Finder-style file access, easy file-type filters and previews, and no more mockups. This change implements the actual application components. It does not introduce a file hierarchy, a separate document-management store, an external Office viewer or an AI provider.

The primary action is Preview. All/PDF/Images/Word/Spreadsheets/Other filters, filename search, one selected row and exact source details cover everyday access. Version history is a secondary filter. Both vendor and response actions launch the shared browser, preserving their exact relationship/revision scope. Quick Look is centered using the existing dialog; the browser launch uses the existing sheet. Signature requirements remain those of the source form.

## States and proof

- Before-state captures preserve Forms response review and Vendor due-diligence detail at light/dark desktop and light 390px.
- `forms-documents` is a labelled sample fixture with all five kinds, available/pending/quarantined states and unknown attribution. Existing `forms-response-history` and `forms-vendor-review-conflict` exercise actual launchers. `forms-documents-error` and other empty fixture populations exercise distinct failure and empty states.
- Unit tests cover exact encoded protected links, server file-type/vendor filters, MIME allowlists, keyboard row actions without nested-control interference, unsupported Office metadata/download, no-byte quarantine, failed-list recovery, image fetch, content-type/size mismatch, retry and object-URL cleanup.
- Browser evidence checks both Forms/Vendors at light/dark 1440px, light 720px CSS reflow, light 390px and dark 320px, scoped axe WCAG checks, no page overflow, type filtering, Space opening, Escape closing and returning focus to the selected row. All ten centered-dialog combinations passed. A separate check decoded a real PNG in the image element using labelled sample transport and confirmed that quarantine exposes no download action. Full Chromium rendered a generated one-page sample PDF in its native viewer; the screenshot was visually inspected. A browser with PDF viewing disabled showed an explicit download fallback without a blank frame. A live production document journey is not claimed by these sample-transport checks.
- Renders exposed overlong file rows; repeated form/question copy was removed from the row and retained in the inspector/preview. No important status or source detail was removed.

Evidence is generated from the real components with sample transport, not a standalone mockup. Local screenshots and browser script are under the task's temporary `clearsight-document-tests-5cd3b33810eb44b997c623fb31a7b720` directory; the final verification ledger identifies the executed checks.

## Release boundaries

### Hosted demo release verification — 7 September 2026

The deployment target is the existing persistent **non-production demo** described in `../engineering/demo-deployment.md`. Production scanner, versioned storage and capacity certification remain separate gates; they do not prohibit this demo release. Pending and quarantined files must still remain unavailable, and sample browser transport does not prove a live scanner integration.

The release review repeated all ten Forms/Vendors viewport/theme combinations successfully, with no page overflow, browser exceptions or scoped axe violations. It also added four document journeys to the existing recurring CI Forms evidence runner (both entry points, light desktop and dark 320px reflow). These check file-type filtering, Word preview/download availability, reachable dialog close, Space/Escape focus restoration and filename-search empty-state recovery. All four passed locally. No second browser framework or new UI variant was introduced.

Read-only checks of the pre-release hosted Forms workspace passed at light 1440px and dark 390px with no browser exceptions, horizontal overflow or axe WCAG A/AA violations. Before-state screenshots are retained in the task's `clearsight-release-a6539950fb0d468fb4038574130e7d3c` temporary directory. Deployed-revision verification is recorded in the release result, not inferred from these baseline checks.

Non-blocking polish candidates: keep document metadata values aligned consistently with their labels, and resolve stored actor IDs to authorized display names where such names are available. Unknown attribution must remain “Not recorded.” These are review observations, not completed changes or reasons to add another navigation layer.

Frontend verification: all 153 web test files / 978 tests and the production build passed on Node 24.19.0 on 7 September 2026, including the disabled-PDF fallback and absent review-timestamp contract. An awaiting-review record shows no invented decision date. Specification and quality reviews passed after the provenance, real-pointer selection and canonical review-label corrections, with independent approval of follow-up fixes. Both builds, copy-quality, runtime fixture isolation and UI contracts passed. Backend specification and security reviews passed after document-currency, exact work-form membership and scalar-answer authorization corrections; the browser does not independently infer these states. Full default Go, PostgreSQL composition, vet and relevant real-PostgreSQL integration suites passed. Production release remains open.

This tranche provides discovery and preview, not the full roadmap. Accept/reject, targeted new uploads, governed withdrawal/disposal and AI validation are not fabricated as frontend actions. The remaining tasks stay in `docs/superpowers/plans/2026-09-07-shared-document-management.md`. Production scanner, durable versioned storage, failed-job recovery, volume acceptance and retention/hold/provider acceptance remain release gates. DOC-16 separately tracks pre-existing restricted-work metadata exposure in the completed-response summary list. No external validation provider receives documents from this browser.
