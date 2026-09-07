# Form and vendor document management: evaluation and proposal

Date: 7 September 2026. Status: approved by the user; implementation tracked in `../superpowers/plans/2026-09-07-shared-document-management.md`.

Evaluated revision: `831b009d8eea743898b7c10ea3858edd25b35e2f`.

## Recommendation

Provide one shared document list and document review panel in Vendors and Forms, using existing Evidence/Capture artifacts, immutable submissions, authority routes and delivery workflows. The document should open with its source question, submission, respondent, reviewer and next valid action already in context.

The principal problem is fragmented access and incomplete lifecycle support. It is not a missing upload widget. A new standalone document-management product would add navigation, duplicate records and a second review process without resolving the underlying gaps.

Confirmed user requirement: signatures are required only where the form indicates a signature field, following that field's required/conditional rules. Document “sign off” means a bank reviewer accepts the exact document for its stated purpose; it adds no signature requirement. Neither action approves the vendor relationship, changes a Program conclusion, or closes an issue automatically.

## Repository and verification receipt

- Fetched origin and fast-forwarded the existing main worktree by 65 commits, from `dccff922` to `831b009d`. No merge conflicts or application edits.
- Preserved the original dirty vendor-management checkout and the unrelated modified presentation image in the main worktree.
- Public `/health/ready` returned PostgreSQL mode, `ready`, and the evaluated revision. This confirms deployment identity and readiness, not every browser workflow.
- Focused Go tests for artifact opening, documents, completed responses and evidence review passed in `internal/evidence`, `internal/thirdparty` and `internal/httpapi`.
- Frontend tests passed: 3 files / 52 tests across `ResponsesView`, `VendorDueDiligence` and `VendorWorkPanel`.
- Evaluation used source, relevant product/architecture/plan/acceptance documents, tests, current open-issue titles and public readiness. No real vendor records were changed or messages sent. This is not a fresh rendered-browser or live PostgreSQL acceptance run.

## Current capability and gaps

| Area | Observed implementation | Practical gap |
| --- | --- | --- |
| Upload integrity | `internal/evidence/service.go` creates request-bound artifact manifests with digest, size, media type and stored state; structural inspection exists. | New uploads enter `STORED_UNSCANNED`. Production scanner, storage and retention lifecycle are explicitly unfinished. |
| Document opening | `internal/evidence/artifact_open.go` opens an exact AVAILABLE artifact. Vendor routes stream protected bytes inline and hide storage keys. | No general Forms document inventory/opening contract was found in the runtime route registry. Native inline content is not a shared preview workspace. |
| Forms Responses | `web/src/components/forms/ResponsesView.tsx:217` renders score, completion, assurance and revision history. | The current review panel does not render submitted answers or attachment rows. The response DTO at `internal/httpapi/form_distribution_response_dtos.go:72` also lacks the answer/artifact detail required for that experience. |
| Vendor assessments | `VendorDueDiligence.tsx:657` renders document metadata, Open, Validate and Reject. `assessment_document.go:92` authorizes and persists review. | UI opening is limited to UNDER_REVIEW and disabled for rejected/expired records. The document review command requires structured `vendor_document` metadata, so ordinary file/photo attachments do not have equivalent review coverage. |
| Vendor request work | `VendorWorkPanel.tsx` opens available documents, accepts a whole response, and requests selected fields again. | Acceptance of a response is not a shared per-file acceptance record; the interaction differs from assessment review and Forms. |
| Historical evidence | Immutable submissions and response revisions exist. | Assessment open checks require the current request (`vendor_assessment_document_open.go:50`). Historical viewing needs an explicitly scoped read path. |
| Replacement | Assessment clarification and vendor-work change requests already issue focused successor requests. | The user must navigate a workflow and select fields; there is no consistent “Request new upload” action on a document in both interfaces. Generic Forms need a corresponding governed successor path. |
| Deletion | Artifact `DELETED` state and low-level storage deletion exist; storage cleanup on persistence failure exists. | These are not a governed submitted-document delete API. Retention, legal hold, affected-use checks and disposal execution are unfinished. |

The documented scan limitation is corroborated by `README.md:44` and `docs/architecture/source-evidence-and-secure-capture.md:104`. File admission is not a malware scan. Do not label admitted files safe or silently enable opening pending files to make the UI appear complete.

## Alternatives

1. **Shared contextual document management — recommended.** One reusable list, preview and action contract embedded in Vendors and Forms. Moderate backend work; consistent experience and no duplicate evidence store.
2. **Patch each existing screen independently.** Fastest first button delivery, but preserves divergent permissions, metadata, history and replacement behavior. Suitable only as an incremental rollout of option 1.
3. **Build a standalone document library/DMS.** Useful eventually for broad enterprise content management, but adds organization, search, permissions and navigation beyond this request. Defer folders, general editing and an independent document portal.

## Proposed experience

### Find and open

- **Vendor relationship → Documents:** include submitted documents from due diligence, vendor work and explicitly linked form distributions in the authorized relationship. Do not infer association from email address, filename or a shared vendor name. Cross-relationship/cross-entity visibility still requires authorization.
- **Forms → Documents:** provide a document-oriented sibling of Responses, scoped to the current legal entity, with filters for form, vendor/subject, submission period, review state and freshness. A form detail opens the same list filtered to that form.
- **Forms → Responses → selected response:** show Answers and Documents using the same source data and document panel. Preserve the existing score/history content.
- Program and issue links open the same selected document; they do not create another copy.
- Search by filename or document type. Provide quick filters for Needs review, Replacement requested and Expiring/expired, plus a history toggle. Keep access/security state separate from business review and currency.
- Draft uploads are visible only to the permitted draft participants. Unsubmitted respondent drafts do not silently enter the bank review queue. Forgotten draft objects belong to cleanup operations, not the submitted evidence population.

Rows show filename/type, vendor/service or other subject, source form/question, submitted by/time, review status and expiry when known. Reviewer and latest activity fit in the detail panel or optional columns. Counts must describe the authorized checked population. Unknown expiry displays “Expiry not recorded.”

Clicking the filename opens a wide shared sheet: document on the left, compact context and actions on the right. On narrow screens use a full-screen view with Preview, Details and History sections. Preserve filters and selected row when returning, and allow next/previous document without closing the review.

### View and preview

Preview AVAILABLE PDFs and images first. Office files initially show accurate metadata and an authorized download option; add rendered Office previews only when a bounded isolated renderer is available. Reuse import extraction only where its supported output meets this purpose; extraction text is not a faithful page preview.

Permission to view is distinct from permission to accept. Authorized users can inspect rejected, expired and historical documents if their bytes are available and access is still permitted. These labels warn about business use; they do not themselves block historical inspection. Pending, quarantined or disposed files show metadata and a concrete recovery action without returning bytes.

### Review and sign off

Use **Accept document**, **Request new upload**, and a secondary **Reject document** action. Show the currently responsible reviewer and permission reason. A decision records the exact artifact version/digest, submission and field occurrence, intended use, decision, actor, time, rationale and applicable expiry.

Display acceptance as “Accepted for [purpose] by [reviewer] on [date].” Viewing does not create acceptance. A new file version starts its own review; it cannot inherit acceptance simply because its filename matches. Previously accepted history remains inspectable with the new review requirement visible.

Document acceptance, whole-response acceptance, a signature captured by a form field, relationship approval and outcome verification remain separate. The UI can group their actions, but cannot manufacture one from another. A form without a signature field never requires a signature to accept its documents. Photo and ordinary file fields use the same review capability as structured vendor documents.

### Request new upload

From a selected document, preselect the source field, related request, current eligible recipient and bank reviewer. Ask only for a concise reason and due date; show recipient and delivery/access details for confirmation and permit an authorized change when needed.

Send a focused successor request through existing invitation, distribution, outbox and reminder mechanisms. The recipient sees what needs replacement, why, and when it is due. Require fresh bytes for the selected upload; keep unrelated submitted answers available as attributed context without presenting inherited signatures as newly signed. Include a fresh signature only when the source form defines an applicable required signature field covering the changed answers; show its original signing context and never add a signature field to a form that has none.

Record the chain from requested replacement to actual delivery, new submission and review. A retry reuses the same request. Another open request for the same document should be shown instead of sending duplicates. Sending a request does not by itself supersede the accepted file; the new response creates the successor relationship and explicitly marks review/currency implications. For closed assessments, route a new reassessment/follow-up under existing lifecycle rules rather than mutating the completed episode.

### Remove and delete

- **Draft attachment:** Remove attachment detaches it from the unsubmitted answer; eligible unreferenced bytes are reclaimed through bounded cleanup. Explain if the upload bytes remain temporarily stored.
- **Submitted document:** Withdraw from use retains the original submitted answer and review history, requires a reason, and identifies affected uses. Do not describe this as permanent deletion.
- **Permanent deletion:** a separate authorized action, with retention and legal-hold checks, referenced-use impact and a durable deletion receipt. Block execution if those controls are unavailable. Dispose of original bytes and applicable derived previews/caches through monitored jobs, while retaining the permitted audit tombstone. State backup/version-retention limits honestly.

No ordinary reviewer can silently erase a signed or relied-upon record. Conversely, legitimate removal remains an explicit, discoverable operation rather than a hidden administrator workaround.

## Minimal integration design

Evidence owns file bytes/manifests and exact submission membership. Third-party services continue owning assessment and vendor-work decisions. Forms owns distribution/workspace navigation. Add a narrow shared document facade to resolve visible rows and delegate commands to the appropriate existing owner; avoid separate vendor and Forms review truth.

Use exact revision → submission → answer field → artifact membership to construct historical reads. A reused artifact may appear in more than one response: `response_workspace_postgres_write.go:69` keeps the first `submission_id` with COALESCE. Therefore `artifact.submission_id` alone is not a complete revision-membership index, and “latest submission” is not a historical resolver. Multiple contributors also require distinguishing the uploader from the person who finally submitted the shared response. Unknown historical attribution stays unknown.

Add only missing durable contracts: shared document-use/review decisions where existing domain records cannot represent them, exact replacement linkage, and disposition records/jobs. Reuse existing vendor decisions through adapters, preserving identifiers and history. Confirm table shape after query and ownership design; do not precommit to a new generalized document hierarchy.

All list/detail/open commands resolve verified tenant, legal entity, purpose and subject visibility before limits or file delivery. The facade returns allowed actions with explanations, and mutations recheck current authority and expected versions. Material records, event, outbox and required jobs commit together. File writes and deletion use recoverable storage orchestration with durable receipts.

Start with indexed, bounded SQL joins. Add a rebuildable document summary projection only if representative measured queries justify it; expose freshness. Versioned object storage, scanning, previews and disposal use the existing API/worker architecture rather than new services by default. No new search cluster, queue product, AI approval engine or DMS integration is necessary for the initial workflow.

## Gap tracker and delivery sequence

All entries below are proposed work, not new GitHub issues or completed features.

| ID | Priority | Work and owner boundary | Acceptance |
| --- | --- | --- | --- |
| DOC-01 | P0 | Evidence: operational storage/inspection path | Real upload reaches AVAILABLE or QUARANTINED with a receipt; outage/retry is visible and recoverable. |
| DOC-02 | P0 | Evidence/API: shared authorized document reads | Every submitted file/photo/vendor-document occurrence can be resolved from Forms and its authorized vendor relationship, including historical revisions and multiple contributors. |
| DOC-03 | P0 | Web: shared list and preview sheet | Same rows and view behavior from both entry points; completed/rejected/expired AVAILABLE files remain viewable; pending/blocked states explain recovery. |
| DOC-04 | P1 | Evidence + third-party authority: review integration | Exact-version acceptance/rejection works for generic and structured file fields without duplicate decisions or unintended parent approval. |
| DOC-05 | P1 | Existing request owners: targeted replacement | One document action produces one field-focused request, delivery receipt, replacement submission and renewed review; retry does not send twice. |
| DOC-06 | P1 | Evidence/storage governance: remove/dispose | Draft removal, submitted withdrawal and authorized permanent deletion have distinct, working outcomes; holds and retained references prevent invalid disposal. |
| DOC-07 | P1 | QA/product: end-to-end acceptance | Real vendor journey, alternate authorized reviewer, both entry points, history, failure recovery and mobile/keyboard operation pass. |

Deliver in four slices: (1) inspectable storage plus accurate document inventory; (2) shared viewer and generic document review; (3) focused replacement with delivery/history; (4) governed disposal and full release acceptance. Ship each slice with explicit supported capabilities. Permanent deletion remains unavailable until its own retention/hold checks are implemented. Treat production artifact readiness as a prerequisite, not an optional polish item.

Keep this work linked to the existing third-party lifecycle (#80), hosted vendor journeys (#139), and release acceptance (#147) so their acceptance evidence is reused. Open issues are planning evidence, not proof that all underlying functionality is absent.

## Acceptance scenarios

1. A vendor submits a certification PDF and an ordinary multi-file attachment. Both appear under the correct relationship and exact Forms response, with the same review decisions.
2. A reviewer opens an inspected file from either surface, accepts it with a reason, and sees actor/time/purpose in both places without approving the parent relationship.
3. A reviewer requests one replacement. The correct recipient receives one working scoped request; unaffected answers stay available, fresh bytes are required, reminders stop on submission, and the prior version remains inspectable.
4. A completed assessment, rejected document and expired document can be read by permitted users; all material changes still obey the relevant lifecycle.
5. Multiple contributors and amended/shared responses preserve exact file occurrence and attribution. A filename match or identical digest cannot leak another relationship's document or transfer acceptance.
6. An unauthorized user cannot list, preview, download or mutate the document by guessing IDs. Permissions are rechecked after assignment revocation, delegation changes or response replacement.
7. Scan failure, quarantine, unsupported preview and missing bytes produce distinct recovery. No pending document is relabelled inspected to enable a button.
8. Withdrawal lists affected uses; a held record cannot be permanently deleted. Disposal retries safely and records what remains under retention.
9. Render desktop and narrow-screen behavior, keyboard focus, screen-reader labels, 200% zoom, loading, empty, unavailable and stale/conflict states. Proposed usability targets: open a visible file in one click; request a replacement in one focused sheet without manually selecting its form or vendor again.
10. Exercise bounded pagination and query plans with realistic document counts across multiple vendors/entities; test storage/worker recovery and include an actual emailed replacement journey before claiming completion.
11. A form without a signature field completes document review and replacement without requesting a signature. A form with a signature field follows its required/conditional rules, and a replacement does not reuse an old signature as a new one.

## Competitive framing

Document review and replacement fit GRC/third-party risk management. Microsoft describes SIEM around security telemetry, threat detection, investigation and response: [Microsoft Sentinel overview](https://learn.microsoft.com/en-us/azure/sentinel/overview). ServiceNow documents questionnaires, document requests and follow-up assessments in TPRM: [Assess third-party risk](https://www.servicenow.com/docs/r/governance-risk-compliance/third-party-risk-management/tprm-assessing-tpr.html).

This is a high-value usability gap, but the review does not support claiming it is ClearSight's only competitive gap. Production storage/scanning/retention, proven real recipient delivery and recovery, and representative user acceptance are directly relevant remaining concerns. If SIEM is the intended category, integrate selected security observations into the existing source/evidence/issue workflow; telemetry correlation and threat investigation are a separate product scope, not capabilities gained through document management.
