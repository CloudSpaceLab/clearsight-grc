# V2.9 workflow completion audit

## Scope and integration

`assets/flow-polish.js` completes existing forms, assessments, evidence, approval preparation, privacy cases, ordinary work actions and notification connections. It uses existing records, routes, dialogs, storage and audit helpers. `assets/flow-polish.css` provides a responsive return/context bar within the existing interface.

Load the script after the V2.8 modules, `loss-register.js` and `workflow-completion.js`. It registers wrappers and actions only; it does not call the startup render. The separate loss and control/vendor/RCSA modules retain their domain decisions.

## Findings and completed flows

| Observed gap | Completed behavior | Record protection |
| --- | --- | --- |
| Form assignments listed every location, and new assignments could target out-of-scope programs. | The assignment queue, target choices and commit validation now use the current location. Assigning the same pinned version opens the current response cycle. | Actor or location changes invalidate an open assignment/source-configuration dialog. |
| A submitted form had no correction loop; connection settings could still change. | A reviewer can request a correction, opening a linked response cycle with the same form version. The respondent can save, return to the program, resume and submit again. | The original definition, submitted answers, observations, attachments and checksum stay unchanged. Human answers and allowed pinned attachments carry forward; technical observations must be collected again. Submitted connection settings are frozen. |
| Submitted custom forms were missing from the shared review queue. | Applicable mapped requirements create exact requirement-review work items. Returned cycles create respondent correction work. Submission and return events appear in Inbox. | A form with no applicable mappings creates an informational event, not an approval that cannot be completed. The historical returned submission is excluded from current requirement sources. Existing program conclusions remain until a new independent decision. |
| Canonical program assessments stayed locked after submission. | Request correction archives the previous run and opens the next draft. Prior submitted answers can be inspected; save/return and resubmit are available. | Previous submitted runs and previous requirement conclusions remain in history. Returning an assessment does not reset effective compliance evaluations. |
| Add evidence exposed all program targets and ended without opening the exact review. | Evidence intake offers only eligible targets, applicable questions and matching source documents. It opens the submitted observation directly. Uploading instead retains the chosen program/question context. | Commit rechecks the original actor/location and current document/target. Missing original-file references cannot be selected as refreshed source files. |
| Expired or returned evidence had no targeted renewal path. | Submit refreshed evidence pins the original program, target and question, creates a new observation and links both observations. The review page shows the correction reason and navigation to each version. | The earlier observation and its pinned file remain unchanged during review. Only independent acceptance supersedes the prior observation chain. A pending replacement is resumed instead of creating competing branches. |
| Approval preparation disappeared on navigation or reload and could be overwritten by starting another request. | Reviewer routing and message persist as a local draft, with save/return and resume from Documents. Starting another request first resumes the saved draft. Cancellation clears it. | No approval request is created until the existing send command succeeds. The command retains source-version, author/requester separation, original-file and reviewer checks. |
| Privacy intake accepted invalid calendar dates, unknown locations or a shared owner/reviewer; closed cases could not be reopened. | Intake validates exact dates and known independent people. Review commands validate scope, reviewer and supported decisions. A closed case can reopen with new information and an explained target date. | The closed episode, decision and previous history are retained. Reopening clears the old outcome/evidence-reference fields and requires another review. |
| Work completion relied on HTML-only text length, had stale-context gaps and could not reopen completed actions. | Completion validates substantive evidence, actor, location and the typed source. Source links open exact records. Reopen records the new owner, target and reason while keeping the completion episode. | Action completion never changes risk ratings, accepted KRI values, loss amounts or source approvals. Missing/mismatched source references block completion. |
| KRI and loss remediation kinds were read-only in the original Work dialog. | Both use the source-aware completion/reopen flow and require a known verifier independent of the action owner. | Loss actions must match their exact `lossEvent`; KRI actions must match their exact return/observation. Verification, source revision and previous completion are retained. |
| New control/vendor review routes were not understood by Inbox, and source decisions could leave stale action alerts. | Inbox recognizes exact control-test, vendor-review and form-response sources. Pending approval alerts resolve when their source review leaves Submitted. Return alerts remain actionable until the linked corrected review is submitted. | Review IDs must belong to the stated source record. Read/archive/acknowledge receipts remain separate from source completion. |

## Notification integration

The module extends the existing event adapter without replacing its emission, data or unread-count APIs:

```js
CS_V28_EVENTS.resolve(eventId, {
  sourceRevision: 3, // optional guard against resolving a different revision
  reason: 'The source review was completed.'
});
```

The function returns `false` for a missing event or a revision mismatch. Otherwise it records the resolution time, acting user and reason, saves state, updates the notification count and leaves recipient read/archive/snooze receipts intact. Use the exact event ID from the domain workflow; this is an internal source-command adapter, not a generic Inbox close button.

Control/vendor review events should supply `sourceReviewId`, `sourceWorkflow` and the exact owning record ID. Return notifications additionally use `type: 'program'`, `sourceState: 'Changes requested'`, and `actionable: true`. Their submitted replacement must retain `corrects: originalReviewId`.

`CLEAR_SIGHT_FLOWS` exposes the current scoped assignments, persisted approval preparation and the resolution adapter for inspection.

## Verification

Run:

```sh
node --check assets/flow-polish.js
node tests/flow_v29_checks.js
```

The suite reports **25 passing actual-source integration checks**. Results are saved in `tests/flow-v29-results.json`. The shared harness loads all V2.9 application modules and substitutes only the DOM/browser shell; commands, records, calculations, route registration and validation execute from the actual source.

The checks cover scoped assignment visibility and stale-context rejection; duplicate assignment reuse; submitted-source freezing; correction/resubmission and immutable checksums; independent review; canonical assessment history; evidence refresh/acceptance and intake routing; approval draft save/resume/cancel; privacy intake and reopening; exact notification resolution; ordinary action completion/reopening; control and engagement references; actual loss-generated remediation; KRI exception resolution; returned vendor-review resolution; and rendered action/route registration.

These checks do not claim browser layout, browser file-persistence or screen-reader validation. Browser execution was unavailable in this environment. Responsive styles and semantic controls retain the existing application conventions.

## Simulation boundary

The package remains a local enterprise workflow demonstration. Named actors, reviewer choices, audit entries and recipient state are persisted locally. These flows send no email or colleague messages and do not establish authenticated server authorization. Production authorization, concurrent transaction handling, notification delivery and durable file storage require the shared service contracts described elsewhere in the package.
