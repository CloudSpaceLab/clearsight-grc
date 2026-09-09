# UI audit corrections — acceptance receipt

Date: 9 September 2026. Scope: the approved [copy, contrast and component audit](../reviews/2026-09-09-copy-contrast-component-audit.md), implemented on the existing `codex/vendor-evidence-reconciliation` working tree. This is a local, uncommitted correction; no deployment or external communication occurred.

## Delivered behavior

| Audit findings | Correction | Evidence |
| --- | --- | --- |
| COPY-01–03 | Missing/stale Program state and unknown schedules are explicit. Historical reviewer attribution never falls back to today's assignee. Requirement authoring collects the actual obligated party, action, object and obligation strength. | ProgramCurrentPosition, TodayInterventions, MatterOutcomePanel, ProgramRequirementAuthoring and continuityCommands regressions; 72 truth captures. |
| COPY-04–06,09–11 | Shared review copy is neutral and concise across authoring, requests, assessment, documents and external receipts. Submitted is distinct from completed work. Received, accepted and approved remain separate. New starter copy is neutral; saved material is unchanged. | Recursive copy-quality regression and affected Forms/vendor/capture tests. |
| COPY-07–08, API-01–06 | Configure copy describes tasks and actual limits. Refresh success follows a successful read. Network recovery preserves uncertainty and domain errors. Backend scope/assessment/guide/notification messages are neutral. Historical display names resolve exact scoped IDs and remain optional. | HTTP, import/proposal conflict tests, Go validation and identity tests; [backend receipt](../reviews/2026-09-09-api-copy-corrections.md) and [display-name contract](../../api/assessment-display-labels.md). |
| Component C01–C04 | One selected-vendor request entry; one response assessment/result/document composition; checklist coverage filters documents per field; Requests and Linked vendor work begin collapsed. Errors and reload actions remain outside the Requests disclosure. | Mixed-coverage/shared-artifact tests, duplicate-action tests, response review and vendor normal/error renders. |
| Component C05–C09 | Shared controls cover vendor relationship forms, due diligence and work recovery. Empty states and overlay lifecycle have one implementation. Retired editor/quality/readiness branches and dead CSS are removed. Filters, review states and score presentation are shared. | Enforced VendorDueDiligence/VendorWorkPanel migration entries, specialized radio exception, UI contracts, nested-overlay tests; reusable-section tests ported to the active picker. |
| Component C10–C11 | Imports opens directly, including old section URLs. Builder overview shows counts and errors rather than every rubric. Priority remains visible; advanced response filters are collapsed with active filters and reset visible. | Forms location tests, inspector tests and closed/open filter renders. |
| Contrast C-01–06 | Opaque placeholder colors, essential field borders and composited status pairs work across themes. Local overrides no longer weaken shared badges or explanatory text. The runner records unresolved readings instead of treating them as passes. | Preserved before/after measurements, browser captures and compositor regression tests. |

Existing documents remain reusable with a reason and source context. A linked ISO certificate appears as received/awaiting review until accepted; it is not listed as an outstanding vendor upload merely because it needs internal review. Conditional approval still follows the existing governed policy, conditions, owners and deadlines. No approval or legal-compliance semantics were weakened.

Independent review found and corrected a deduplication regression: assessment-load failure had hidden the already-loaded submission score and document access. Those remain accessible for the exact response while the assessment can be retried. A regression test reproduced the failure before the fix. Rendering also caught invalid definition-list roles and incomplete sample endpoint coverage; both were corrected and rechecked.

## Verification

- TypeScript strict check, production build and isolated evidence build passed on the final frozen source.
- All **28/28 browser accessibility scenarios passed** after correcting the 34px Imports mobile action to the existing 44px target. **168 contrast capture states** have zero detected violations or runtime errors; incomplete readings remain recorded. See the [contrast receipt](../reviews/2026-09-09-contrast-corrections.md).
- All 12 runtime-fixture-boundary and UI contract checks passed after the migration manifest updates.
- Full `go test ./...` passed; `go vet` passed for HTTP API, evidence, formcontract and onboarding packages.
- Full frontend suite: **1,154/1,154 tests passed**, zero failed suites. The final shared-select label correction was then verified by **211/211** shared-control and affected-workflow tests, including a new red/green case for concatenated descriptions. [Full results](../evidence/2026-09-09-ui-audit-corrections/verification/frontend-tests.json).
- Final affected frontend batch: **202/202**; recovery/fixture/quality/copy batch: **54/54**. Earlier failures were corrected and rerun; reports above represent the passing versions.

## Rendered proof

The [original 104 captures](../evidence/2026-09-09-ui-audit/contrast/) remain unchanged. The pre-edit source snapshot is `.codex-tmp/ui-audit-before-20260909.zip`.

- [Truth-state manifest](../evidence/2026-09-09-ui-audit-corrections/truth/manifest.json): 72 captures, both themes, desktop/390/320 and a labelled 720px reflow approximation; no page errors or document overflow.
- [Contrast after-state](../evidence/2026-09-09-ui-corrections/contrast/summary.json): full audit matrix plus targeted correction captures and measurements. Separate narrow/reflow/filter/recovery results preserve their own summaries.
- [Vendor panel matrix](../evidence/2026-09-09-ui-audit/after-vendor-panels/manifest.json): 48 panel captures, both themes at 1440/390/320, plus disclosure checks.
- [Final vendor state proof](../evidence/2026-09-09-ui-audit/after-vendor-final/manifest.json) and [error-state proof](../evidence/2026-09-09-ui-audit/after-vendor-errors/manifest.json): 18 normal/error captures with current shared controls. Narrow action centers were checked after scrolling to confirm they were not covered by navigation.

Manual inspection covered the checklist's Missing/Awaiting review/Received/Accepted states and reuse source, response history/assessment, shared review controls, missing/reassigned reviewer, missing Program calculation, requirement forms, collapsed work and narrow errors. Before-state screenshots are separate from after-state repairs.

## Limits

These are local sample-fixture and automated regression results, not an authenticated production journey, a full screen-reader session or product-wide WCAG certification. Automated incomplete/unsupported contrast and accessibility readings remain recorded for review, including gradient-dependent colors. A 720px reflow approximation is not browser zoom certification. Specialized vendor identity/row/radio controls remain explicitly scoped; this is not a claim that all workspaces use shared components.

The race-enabled Go command could not run because this host has CGO disabled and no available C compiler; normal tests and vet passed. No database migration was introduced by this audit correction. The pre-existing evidence-reconciliation changes and their separate acceptance requirements were preserved.

Automatic approval review blocked deletion of the temporary `web/evidence/dist-truth-evidence` build directory with “blocked by policy.” Its preview was stopped; the folder remains and is not a deliverable. No deletion bypass was attempted.
