# Vendor evidence reconciliation acceptance

Date: 8 September 2026. Branch: `codex/vendor-evidence-reconciliation`. Before-state source: `11687656`.

## Delivered scope

- Prepare an assessment request before issuing access; review existing evidence before asking the vendor.
- Link an exact vendor-submitted source to a requirement with a reason, under current bank reviewer authority.
- Show Missing, Pending review, Accepted, Received and unresolved applicability separately. Received evidence never implies bank acceptance or vendor approval.
- Review each requested field independently, including when multiple requirements use one source artifact.
- Respect held evidence in capture, required-field validation, progress and reminder eligibility without inventing answers or submissions.
- Keep an existing vendor draft; avoid an empty response when every required document is already held.
- Preserve source provenance, old decisions and material versions; recheck current source eligibility when consuming a receipt.

## Interface evidence

The [23-state render manifest](../evidence/2026-09-08-vendor-evidence-reconciliation/ui/manifest.json) records routes, fixtures, viewports, themes, focus and layout metrics. It passed with `failure: null`. Screens were also inspected directly using the browser and local PNG viewer.

- [Desktop checklist](../evidence/2026-09-08-vendor-evidence-reconciliation/ui/vendor-collection-checklist-light-1440.png) and [dark checklist](../evidence/2026-09-08-vendor-evidence-reconciliation/ui/vendor-collection-checklist-dark-1440.png).
- [Document chooser](../evidence/2026-09-08-vendor-evidence-reconciliation/ui/vendor-collection-choose-light-1440.png) and [mobile confirmation](../evidence/2026-09-08-vendor-evidence-reconciliation/ui/vendor-collection-reason-light-390.png).
- [320px checklist](../evidence/2026-09-08-vendor-evidence-reconciliation/ui/vendor-collection-checklist-dark-320.png), long-content reflow, loading, empty scope, read-only authority, error/retry and conflict recovery.
- [Replaced source warning](../evidence/2026-09-08-vendor-evidence-reconciliation/ui/vendor-collection-replaced-light-1440.png), focused Accept/Reject, received vendor documents, retained draft recovery and [all-held vendor receipt](../evidence/2026-09-08-vendor-evidence-reconciliation/ui/vendor-collection-capture-all-held-light-390.png).

Repairs from inspection: forms history displaced the active review; it now follows due diligence. Reference facts are expandable. The review action opened below the viewport; it now uses the shared focused sheet. The sheet grid and three-column document browser squeezed filenames; selection now uses horizontal type filters and the full available list width. A filename-width regression check accompanies the normal overflow and scoped accessibility checks. No new palette, token, density or motion was introduced.

The fixtures use labelled sample data. They prove interaction and layout, not bank compliance. This new scope does not claim successful sample PDF byte preview, actual browser 200% zoom, assistive-technology user testing or hosted acceptance.

## Executable verification

- Node 24.19: 198 tests passed across 14 affected frontend files, including capture, vendor checklist, workspace, forms, document browser, activation and copy quality.
- TypeScript, production build and isolated evidence build passed.
- Runtime fixture-boundary and UI-contract checks: 12 passed.
- `go test ./internal/evidence ./internal/thirdparty ./internal/httpapi`: passed. PostgreSQL-tagged evidence, thirdparty, HTTP and API entry packages also compile.
- Real PostgreSQL 18.6: reconciliation/source provenance and atomic document/assessment/event/outbox tests passed. Capture consumption passed all four cases: quarantined source, changed request, unchanged held source and later replacement, including restored reminder eligibility. These integration tests require both `postgres` and `postgresintegration` tags; compilation alone is not their proof.
- New HTTP tests exercise actual preparation/resumed sending, masked checklist reads, unsigned identity, authority denial and forged actor/tenant/entity input. Schema ownership checks passed.

To regenerate this slice's renders, build the evidence entry, serve it locally and run `web/scripts/capture-ui-evidence.mjs` with `UI_EVIDENCE_SCOPE=vendor-collection`, `PAGE_URL` set to that server and `UI_EVIDENCE_DIR` set to the evidence directory above. The new scope also runs in the default evidence suite.

## Release limits

This is a local implementation with retained evidence, not a deployment. Migration 86 and the existing identity, authority, protected delivery and artifact scanning/storage services must be available in the target environment. No vendor invitations or external messages were sent during this work.

Automatic all-held transition/reminder suppression is limited to unconditional required document collections; conditional requirements still need their controlling answers. The broader spreadsheet interpretation, requirement deduplication, onboarding orchestration and expanded conditional-approval design remain in the [journey review](../reviews/2026-09-08-vendor-journeys-requirements-and-onboarding.md).
