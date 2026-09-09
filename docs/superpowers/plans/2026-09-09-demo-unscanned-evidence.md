# Demo unscanned evidence implementation plan

**Goal:** Allow document use and governed acceptance without an antivirus scan in the demo, as explicitly requested by the user.

**Architecture:** A validated startup setting, `CLEARSIGHT_DEMO_ALLOW_UNSCANNED_ARTIFACTS`, applies only when demo mode is enabled outside production. Artifact status and scan receipts remain truthful. Server-derived eligibility is recalculated from the active setting on reads and at transactional writes. No browser or request-body flag grants permission.

**Decision:** Reuse the existing artifact/review workflow with a demo exception. Do not mark files scanned or create a second demo acceptance workflow. Default the setting on in demo mode, support explicit off, and reject attempts to enable it outside demo or in production. Quarantine, content integrity on reads, current-version, expiry, authorization and independent review remain enforced.

**Tech stack:** Go, PostgreSQL, React, Vitest, Docker Compose, existing release CI.

- [x] Add config tests for demo default/explicit off, non-demo rejection and production rejection, then configure API/worker consistently.
- [x] Add artifact policy and document eligibility tests, retain hash/size verification, and recalculate reconciliation eligibility after disabling the setting.
- [x] Cover third-party reuse, receipt review, document validation, activation and work-document gates, including transaction-time checks and PostgreSQL tests.
- [x] Show concise Unscanned demo status and enable existing preview/review controls only from server-derived eligibility. Test both demo-enabled and default-off states; render affected desktop/mobile light/dark views.
- [ ] Update README, architecture and deployment configuration; run affected tests, full CI and independent review.
- [ ] Merge and deploy through the existing pipeline; verify exact API/worker revision and hosted existing-document reuse/review with sample data.
- [x] Apply the user's document-explorer refinement: align file-type icons/labels and selected marker, retain mobile select replacement, and verify six light/dark responsive renders. See [navigation brief](../../design/2026-09-09-document-browser-navigation.md).

## Acceptance states

Available files behave as before. Unscanned files are usable only with the active demo exception. Quarantined/unavailable files remain unusable even if a stale eligibility flag exists. Disabling the setting removes usability without rewriting scan history or acceptance history. Reconciliation and evidence acceptance remain separate user actions.

The user authorized this behavior and the ongoing main deployment in the release task. No real vendor invitation or private workbook upload is required.

## Local verification

The full Go suite passed. PostgreSQL API/worker composition, protected content and configuration tests passed. Full third-party PostgreSQL (57.112s) and evidence PostgreSQL (68.896s) suites passed on a newly migrated isolated database; the owned cluster was stopped afterward. These cover default off, enabled, quarantine/deletion, revocation, stale versions, forged caller flags and truthful audit/scan state. Audit metadata was added without changing outbox payload contracts.

All 181 affected frontend tests, seven DocumentBrowser tests, TypeScript, the evidence build and 15 browser-harness node checks passed. Full release CI and hosted verification remain required after local rendered proof.

The [demo evidence manifest](../../evidence/2026-09-09-demo-unscanned/manifest.json) records 20 states and 40 screenshots covering allowed/blocked collection and document selection plus allowed receipt review, at 1440/390px in light/dark themes. There were no page errors, document/dialog overflow or automated accessibility violations; incomplete accessibility diagnostics remain documented. The final horizontal picker fix preserves an 8px label/marker gap, 44px controls and reachable horizontal scrolling. No file bytes or material commands were sent by the local render check.

A final cross-module status scan also identified review-specific document-open routes and issue-remediation artifact checks. They use the same current server-derived exception, with regression cases for disabled/enabled, exact source scope and quarantined/deleted artifacts. The full continuity suite passed after its correction.

The HTTP follow-up adds ten helper cases and fourteen real handler cases, covering direct/reused/work document access, fresh occurrence authorization, exact source selectors and independently revoked artifact access. The full HTTP suite passed after the correction. A local linker disk-space failure was resolved by clearing only two verified, old task-owned Go build caches; the default Go cache and source files were preserved.

Final independent review found an AVAILABLE-only SQL predicate behind vendor outstanding counts and reminder eligibility. It now uses the current repository setting for both reads and the transactional reminder recheck; the worker configures its separate reminder repository explicitly. New PostgreSQL regression coverage verifies default off, enabled, revoked, quarantined and expired states, ignores forged or stale stored allowance flags, and rechecks policy changes before scheduling. The focused collection/vendor-form PostgreSQL suite passed (5.738s), as did the worker suite (1.221s).

The full evidence PostgreSQL suite passed again after the SQL correction (31.936s) from clean fixtures in the owned isolated database. The owned cluster was stopped. Independent follow-up review confirmed the outstanding-count/reminder gap was resolved with no remaining P1/P2 findings in the correction.

The initial CI UI run exposed a harness comparison that included the new decorative checkmark in raw button text. The assertion now checks one selected button, its label and the visible accessible name independently of the aria-hidden marker. The canonical Forms runner passed all 73 captures using the CI-pinned Playwright 1.55 runtime; 51 node harness/contract checks passed. The updated commit must pass CI before merge.
