# API copy correction receipt

Implements API-01–06 from [the API copy audit](2026-09-09-api-copy-audit.md). This receipt concerns the audited backend producers and the separately assigned vendor shared-control migration, not every dynamic server error or every application surface.

## Backend results

- Shared assessment and scope messages now use assessment and organization language. Verified actors, exact legal-entity boundaries, error codes, authority evaluation and transactions are unchanged.
- Import conversion conflicts explain reload/review recovery. Failure identifies the Program requirement or control that could not be created and retains the consequence that no approval was recorded.
- Score preview rejects invalid revision selection, more than 500 answers, and each existing value-size limit with a distinct accurate message. Size bounds still use bytes where the existing validation uses Go string length. No limit was relaxed.
- Oversight messages distinguish unavailable data from a legal entity whose oversight has not yet been calculated. Upload scope recovery directs the user to reopen the evidence request.
- Field-assessment validation uses a typed reviewed message without exposing joined internal error prefixes. Maker-checker, conflict and inactive form states retain separate recovery messages and unchanged codes. Unrelated dynamic errors were not suppressed wholesale.
- Shared guides use concise neutral descriptions with their optional, resumable lifecycle intact. Notification text and HTML use **Next action**. Universal communication previews use **Sample organization** while saved `bank_name` placeholders remain compatible. No notifications were sent.
- Historical reviewer and response-application names use optional HTTP read DTO enrichment. The [display-label contract](../../api/assessment-display-labels.md) defines exact scoped resolution and omission behavior. Current pending responsibility is not inferred from a historical decision or owner.

## Verification

Focused validation and display-name tests were observed failing before implementation and passing afterward. Cases cover status/code preservation, each score-preview bound, reviewed domain errors, scope mismatch, unavailable identities, duplicate historical IDs, missing names and pending decisions. Full `go test ./...` passed after the backend changes. This is local automated verification; no new database deployment or authenticated browser failure-injection run is claimed.

## Vendor controls

`VendorDueDiligence` and `VendorWorkPanel` now use shared buttons and fields across start, send, reissue, document review, conclusion, clarification, finding, cancellation, secure-link recovery and work actions. Native radio grouping and specialized capture behavior remain intact. Tests use accessible shared-select options and preserve command payload and state assertions. The combined VendorDueDiligence, VendorWorkPanel and VendorsWorkspace suite passed **126/126** after integration, with TypeScript passing.

The initial panel render probe exposed invalid definition-list grouping in submitted answers. Removing the overriding group role preserves the native term/definition relationship. A focused axe regression demonstrated the failure and subsequent pass. Final panel rendering is recorded separately by the reproducible `web/scripts/capture-vendor-review-panels.mjs` manifest; incomplete automated accessibility checks require inspection and are not counted as passes.

## Final render and follow-up checks

- [Eight-panel matrix](../evidence/2026-09-09-ui-audit/after-vendor-panels/manifest.json): 48 panel states across light/dark and 1440/390/320px, with zero detected axe violations, JavaScript errors or horizontal overflow. These captures preceded the later sample vendor request routing correction and final register controls; the panel controls themselves were unchanged afterward.
- [Final selected vendor and conclusion](../evidence/2026-09-09-ui-audit/after-vendor-final/manifest.json): 12 refreshed panel states plus selected-vendor collapsed/expanded screenshots. Requests show one scoped sample request; linked work remains a separate population. Both disclosures start collapsed and expand. Narrow viewport captures confirm that the final action is not covered after scrolling into view.
- [Explicit unavailable vendor fixture](../evidence/2026-09-09-ui-audit/after-vendor-errors/manifest.json): six theme/width states with expected Forms/activation unavailability, usable start-review controls and no detected axe violations, JavaScript errors or horizontal overflow.
- Reviewed representative start dark320, conclusion light390, document dark320, clarification desktop, normal selected-vendor mobile and expanded dark mobile images. Tall element/full-page screenshots include fixed shell bars at the screenshot scroll position; the additional actual-viewport images demonstrate the action location without interpreting those stitched images as a viewport.
- Each capture retains 1–2 incomplete axe categories. The matrix includes incomplete color-contrast analysis and generic-container ARIA labeling checks; they are not a passing accessibility certification. No screen-reader session, production authorization simulation, request-volume timing benchmark or full 25-request visual matrix is claimed here.

The Forms proposal conflict workflow also repeated a premature “loaded” claim. Its copy now states reload recovery while the read is pending or failed and reports a loaded version only after success. Tests exercise all three conditions. Form quality messages now identify the record destination to correct and state the actual score-weight range of 1–100 separately from points of 0–100. The focused and affected proposal/quality/static-fixture/copy suite passed **54/54**, with TypeScript passing.

Sample fixtures remain behind the existing static-demo plus UI-evidence build flags. The history fixture serves exact revision IDs and saved automatic scores of 72/86, without inventing manual decisions. Normal vendor Forms and activation endpoints now match exact sample relationship IDs instead of falling into the relationship-detail route. The request uses the existing sample distribution; activation remains blocked on a modeled unsatisfied gate. `vendor-requests-error` separately exercises service failure. None of these fixture handlers implement production access or authorization.

## Independent audit cross-check

Source review of the final working tree confirms the following implementation paths. Root integration evidence and tests remain the source for the broader product acceptance; this cross-check does not convert a source finding into a browser claim.

| Audit | Final source finding |
| --- | --- |
| COPY-04/05 | Shared review/configuration/capture/checklist labels are neutral; stable enums and persisted template keys remain compatible. |
| COPY-06 | Responses uses submission wording, including its accessible table name and review-sheet fallback. Internal completed-response API/type names remain stable. |
| COPY-07 | Audited Configure narration was removed; operator-relevant provider, transport and export limits remain. |
| COPY-08 | Network errors are normalized; import and proposal conflicts no longer claim refresh success before completion. |
| COPY-09 | New shared starter content is neutral; this code does not rewrite saved template revisions. |
| COPY-10 | Today and current-handoff title/prose duplication was reduced; unavailable action does not imply an unassigned issue. |
| COPY-11 | The copy scan now includes recursive component sources and focused narration patterns. A passing scan remains a regression aid, not complete copy approval. |
| C01–03 | Duplicate selected-vendor generic request removed; response result/document composition consolidated; vendor residual documents filtered by field, preserving shared-artifact independent review. |
| C04/05 | Requests and linked work are secondary disclosures; vendor panels and register action/field groups use shared controls. Specialized row selection and radio semantics remain intentional. |
| C06/07 | Retired editor/readiness/lifecycle branches removed or migrated to production fixtures; empty states adapt the shared contract; overlay lifecycle is shared with nested-lock handling. |
| C08/09 | Vendor filters and score facts share explicit presenters; domain-specific states remain separate; stylesheet cleanup is recorded by the contrast reviewer. |
| C10/11 | Imports navigates directly through the existing dirty-editor handling; builder assessment overview shows counts/errors; response secondary filters collapse while priority, active filters and reset remain accessible. |

This cross-check found the residual register controls, response accessibility copy, collapsed priority control, proposal refresh claim and form-quality narration. They were returned to the owning agents and corrected before the final build. Larger acceptance populations, zoom/reflow proof and nested-overlay coverage should be read from the coordinating acceptance receipt rather than inferred from this vendor panel matrix.
