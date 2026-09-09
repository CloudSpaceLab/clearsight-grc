# Hosted form draft acceptance

## Decision brief

Hosted verification of `45c4971c` confirmed that the vendor starter upgraded to active version 6 with ten questions and that the synthetic three-row spreadsheet proposal rendered. Creating the draft then returned a conflict repeatedly, even after reloading the proposal. The imported source and proposal were unchanged.

Proposal reads return the tenant slug, while verified request identity supplies its UUID. Atomic acceptance compared these strings directly. Resolve both references against the same tenant row within the acceptance transaction. Keep legal-entity equality, source version/hash validation and atomic draft/event/outbox writes intact.

The API also grouped source, state and version conflicts under one code, while the browser discarded codes from the API's flat error envelope. Preserve flat and nested error codes, distinguish source/state recovery from a stale version, and use the shared error notice. Keep selected questions while displaying the recovery action.

The final copy pass removes bank-specific approval wording from the demo document notice and identity/version jargon from vendor onboarding. The unscanned sample limitation and initial accountable owner remain explicit.

## Required states and verification

- PostgreSQL: UUID and slug identity accept the same unchanged source; different/unknown tenants, another legal entity and a changed source remain rejected. Forced outbox failure rolls back the draft and acceptance. Repeated acceptance is idempotent. The clean-fixture monitoring integration suite passed.
- HTTP/browser: source and state conflicts retain their distinct recovery messages, selected fields remain selected, and unrelated version reloads are not triggered. The API's actual flat error envelope and the existing nested envelope are covered.
- Thirty focused form builder, proposal, HTTP and copy tests passed; TypeScript and the evidence build passed. Ninety earlier document-preview, vendor-workspace and copy tests also passed.
- [Eight recovery renders](../evidence/2026-09-09-form-draft-recovery/manifest.json) passed across light/dark themes and 1440/390 widths, with no page errors or horizontal overflow. The mobile error notice was inspected and changed to the shared component before recapture.
- The keyboard browser check now waits for listbox focus before sending keys and verifies focus progression after dismissal. The existing dismissal assertion remains. A delayed-focus regression reproduced the missing readiness check; the pinned browser scenario passed locally. Full release gates are required before merge.

The hosted document picker locates previously submitted sample files. Those samples remain preview-only because the demo has no completed antivirus scan for them; they cannot support evidence acceptance. No real vendor invitation or private workbook was sent. Historical user-authored and approved form wording remains versioned and is not rewritten by this release.
