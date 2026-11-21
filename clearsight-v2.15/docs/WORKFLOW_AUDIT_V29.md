# V2.9 — Completion of operational workflows

## Concrete gaps found and completed

| Existing gap | Completed interaction | Resulting behavior |
| --- | --- | --- |
| New controls had a free-text business unit but no organizational ID, so they could disappear from scoped registers. | Full control implementation editor with explicit hierarchy location, owner, operating objective/procedure, frequency, execution type, reference and requirement/risk links. | A saved control remains visible in its accountable scope. A new control begins as **Not assessed**. |
| Controls were largely read-only after creation, with no documented test or independent conclusion. | Saved control drafts; structured test capture; submission; independent approve/request-changes; corrected runs; pinned document version; test and configuration history. | Population, sample, exceptions, method and evidence remain explicit. Only accepted test conclusions update control effectiveness. Risk and requirement conclusions stay independent. |
| An edited implementation could retain an effectiveness conclusion for an older definition. | Definition digest on tests; stale-review rejection; operational-definition changes mark the current implementation **Not assessed**, retaining previous effectiveness and historical test snapshots. | A historical accepted test is not silently presented as assurance over changed implementation details. |
| First-line RCSA typing was lost on navigation and there was no save/resume affordance. | Autosaved draft by cycle + risk, explicit Save & exit, same-cycle resume and empty-evidence recovery links. | Draft changes do not affect effective risk ratings. |
| RCSA accepted mutable source relationships and weak reviewer/decision input validation. | Population and scope guards, known independent reviewer, fixed decision choices, pinned submission basis/evidence version and effective-rating comparison at decision time. | A changed source or concurrently approved rating cannot be overwritten by a stale new submission. Challenge/resubmit remains available. Shared decision receipts remain intact. |
| Vendor activation was a one-way modal checkbox without structured reviewer/accountability. | Full service review/lifecycle workspace with source snapshot, named preparer/reviewer, performance, issues, rationale, contract/reference, continuity/exit arrangements and next checkpoint. | Activation/resumption rechecks current assurance gates. Suspension and exit are independent recorded decisions. |
| Vendor records had no periodic review, suspension/resumption or verified exit journey. | Periodic review → independent decision; suspend/resume; exit with completed data disposition and evidence reference; correction/resubmission. | Source and decision history remain inspectable. Exited relationships stay available as history. |
| Suspended and exited relationships would appear in the old “anything not Active” onboarding filter. | Correct lifecycle tabs, review checkpoints and scoped supplier counts. | Onboarding means Onboarding/Due diligence. Suspended/exited services have their own view. |
| Risk context and accountability could not be edited from the risk workspace. | Saved risk-context editor for title, owner, taxonomy, cause/event/consequence and primary control. | Accepted ratings, appetite and location remain unchanged. Previous supporting control links are retained until explicitly removed. |
| Newly added review flows could have been isolated from the shared queue. | Pending control-test and vendor reviews project into `workItems()` and emit in-app review notifications. | Work opens the exact review page instead of a generic task or unrelated record. |

## Try the completed flows

### Control implementation and test

1. Open **Controls → Add control**.
2. Describe the implementation, choose the exact accountable location, map its requirements and save.
3. Open **Record a control test**. Capture the period, tester/reviewer, population, sample, exceptions, method, findings and working-paper reference. A document version can be pinned.
4. Submit. The control effectiveness remains unchanged while review is pending; the shared Work queue links to this test.
5. Independently approve the result or request changes. A correction creates a new run linked to the earlier immutable submission.

Editing the operational definition requires a fresh test. Editing does not approve the implementation or alter program compliance.

### RCSA draft and decision

Open a cycle and risk, begin the first-line assessment and navigate away. Reopening the same cycle/risk restores the draft. Submission locks the proposal and captures its basis. Approval validates evidence freshness/acceptance, reviewer independence, source identity and the effective rating that existed at submission. Source drift is handled through challenge and a new submission.

New submissions carry the enhanced provenance. Existing V2.8 seeded submissions retain their original history; historical provenance is not invented for them.

### Vendor periodic review or lifecycle decision

Open an engagement, then **Review service lifecycle**. Prepare a periodic review, activation, suspension, resumption or exit request as permitted by the current state. Save and resume when source evidence is incomplete. Submit to an independent reviewer, who can request changes or approve. Activation and resumption require all applicable linked assurance requirements to remain currently satisfied. Exit requires a recorded data-disposition outcome and verification reference.

### Risk context maintenance

Open a risk and choose **Edit risk context**. Update the causal statement, owner and primary control. The editor preserves the existing category taxonomy, prevents unrelated control scope, retains supporting links and records a change reason. Risk ratings continue through RCSA.

## Integration

Load `workflow-completion.css` and `workflow-completion.js` after the existing domain modules and before final bootstrap. No independent bootstrap is performed.

Additional standalone routes:

- `control-editor`
- `control-test`
- `vendor-review`
- `risk-edit`

The existing control workspace is replaced by the completed implementation/test view. Existing risk/vendor records gain clear entry points. New local state belongs to `S.workflowCompletion`; risk, control, engagement, RCSA and common decision records retain their established domain ownership.

Pending review and owner correction work project from `controlTests` and `vendorReviews`; they are not separately editable approval state. Notifications point to the exact submitted review. Returned reviews stay in the original preparer’s Work queue while a correction is drafted, and leave it only when a replacement review is submitted. Return notifications remain actionable on the same basis. Superseded returns link to their replacement instead of starting another correction.

## Validation

`node --check assets/workflow-completion.js`

`node tests/workflow_v29_checks.js`

Fourteen actual-source tests passed. They cover scoped control visibility, draft concurrency, sample validation, independent control-test approval, immutable corrections and returned-work queue/notification resolution, RCSA draft retention, source/rating drift, vendor review/activation/exit gates, lifecycle filtering, rating-preserving risk edits and registered workspace actions.

The test harness substitutes the browser/DOM shell only. No browser-layout, keyboard-navigation or screenshot verification is claimed; this environment still prevents Chromium from launching.

## Remaining implementation boundaries

This is a local interaction reference. Authenticated actor enforcement, server authorization, concurrent-request control, durable audit storage, evidence-byte verification, email delivery and scheduled review reminders belong in the production implementation. Source references may identify external working papers; they are not assertions that source bytes have been uploaded or independently checked.
