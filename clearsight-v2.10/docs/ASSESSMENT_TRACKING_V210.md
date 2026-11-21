# V2.10 — named assessment responses and independent review

## Implemented behavior

The assessment target remains the organization, location or vendor engagement. A separate named person owns each response. An external supplier contact remains the named respondent when an internal coordinator records their supplied response; the actual submitting user is recorded separately.

Program → **Assessments**, the assigned forms list, individual response workspaces and the response history page now show the named respondent, internal coordinator where applicable, actual submitter, submission time, source version, response cycle, assigned reviewer, decision and comments. Program pages have visible assignment and response-history controls. Every submitted cycle has a full, read-only source page with answers, technical observations, pinned Documents, review events and a comparison with its previous submitted answers.

Every new canonical assessment or assigned form submission enters an independent whole-response review. This includes forms without requirement mappings. The review appears in Work and in the assigned reviewer’s Inbox. Accepting a response does not mark any requirement Satisfied. Requirement conclusions remain separate decisions based on the current accepted response.

The acting user supplies the recorded submission and review identity. Submission no longer accepts a selectable substitute submitter. An internal respondent must submit their own assigned response. A named external respondent requires an assigned internal coordinator and an explicit source/capture reference. Designer previews are saved as previews and cannot create formal unassigned submissions.

## Review and history rules

| Operation | Validation and resulting behavior |
| --- | --- |
| Assign a response | The acting program owner or eligible independent risk officer chooses a named respondent, reviewer and due date. Separate people receive separate response stores even when the program, target and form version are identical. |
| Submit | The actual actor must be the named internal respondent or external-response coordinator. Required answers and source checks run before a snapshot is created. Assignment, respondent, actual submitter, date, source version, answer snapshot and checksums are pinned. |
| Accept response | Only the assigned reviewer may decide. They must currently be the program owner or an active, scoped independent risk reviewer, and must differ from both the named respondent and the actual submitter. Source integrity, current program basis and pinned document references are checked. |
| Request changes | A substantive comment creates a linked next draft. Prior answers, document pins, actor attribution, timestamps, comments and receipts remain intact. A missing source document may be returned for correction; it cannot be accepted. |
| Resubmit | The named person submits a new immutable cycle. It gets its own review Work item and actionable notification. The previous accepted response and program conclusions remain in history. |
| Start a new response | The respondent, coordinator or eligible program manager gives a reason. It opens a distinct next draft. Another pending draft is resumed rather than duplicated. |
| Reassign a pending review | The program owner or eligible risk officer chooses an eligible replacement and records a reason. The original reviewer remains in the submitted envelope; routing history identifies the replacement. The former reviewer’s alert resolves, and the replacement receives a new alert. |
| Record a requirement conclusion | The source must be the current accepted response with an unchanged program basis. The actual actor must be an eligible independent owner/risk reviewer. A Satisfied conclusion also checks objective coverage, source freshness and the actual pinned document bytes. Actor, location, program basis and response state are checked again after asynchronous file verification. |

Starting a later response draft or submitting an update does not overwrite the existing accepted requirement evaluation. The earlier response is available for inspection; a new conclusion waits for the current response to be accepted. Updating a program’s applicable requirements or rules invalidates response readiness until a response against that basis is reviewed.

## Legacy information

Legacy source data is imported as **Legacy record**, not retroactively accepted. A recorded legacy person name is retained with a provenance limitation; a department or unit name is not converted into a person. Missing submitter identities, dates, whole-response reviews and source versions remain “Not recorded.”

Legacy canonical runs did not pin historical document versions or exact question definitions. Their actual answer values are preserved, current question labels are identified as catalog labels, and current Documents are not backfilled as historical support. Existing historical program conclusions remain intact.

## Integration

Load `assets/assessment-tracking.js` and `.css` after `flow-polish.js`. It registers wrappers, routes, actions and APIs; it does not render during script evaluation. The shared startup/render path performs the lazy legacy import. `documents-unified.js`, `vendor-onboarding.js` and the separate assessment reporting adapter consume the public contract.

Routes:

- `assessment-history?id=PROGRAM&target=TARGET&assignment=ASSIGNMENT` — program/target or assignment history. Optional `person` and `state` filters affect both the displayed records and CSV export.
- `assessment-review?submission=SUBMISSION` — exact immutable submitted response, Documents and decision history.
- `assessment?id=PROGRAM&target=TARGET&assignment=ASSIGNMENT` — named canonical response. Its storage key includes the assignment ID.
- `form-response?binding=BINDING` — named assigned form cycle. Correction cycles use linked new binding IDs.

The new read model is stored in `S.assessmentTracking`:

```js
{
  version: 1,
  assignments: [],
  submissions: [],
  audit: []
}
```

An assignment retains its program, target, exact scope IDs, named respondent, coordinator, reviewer, deadline, current cycle, current submission, last accepted submission, source binding/storage key and assignment history. Canonical answers remain in `S.assessments` under an assignment-specific key; form answers remain under their exact form binding. There is no shared mutable answer store between named people.

A submission retains:

```js
{
  id, assignmentId, kind, programId, target, sourceId,
  scopeIds, orgScopeIds, version, cycle,
  respondent: { id, name, kind, organization?, email? },
  coordinatorId, captureReference,
  submittedById, submittedBy, submittedAt,
  reviewerId, assignedReviewerId, routingRevision,
  assignmentSnapshot, sourceSnapshot, sourceHash,
  sourceChecksum, envelopeHash, programDigest,
  previousSubmissionId,
  status, events, requirementDecisions,
  review: {
    decision, status, reviewerId, reviewerName, at, note,
    eligibility: { kind, reason, scopeIds, trace },
    receiptId
  }
}
```

`reviewerId` is the original submitted assignment. `assignedReviewerId` is the current pending route. `sourceSnapshot` contains the program basis, definition, answers, observations, explanation, run IDs, exact document-version metadata, source bindings and external capture reference. Hashes detect changes to both source contents and the submitted identity/assignment envelope. Review events and requirement decisions append without rewriting the submitted envelope.

## Public API

`CLEAR_SIGHT_ASSESSMENTS` exposes:

```js
assign(config)
submit(assignmentId, { captureReference? })
review(submissionId, { decision, note })
startRevision(assignmentId, { reason })
reassignReview(submissionId, reviewerId, reason)
requirementDecision(submissionId, requirementId, {
  decision, note, coverage?
}) // asynchronous when verifying source Documents

assignments({ assignmentId?, programId?, target? })
submissions({ assignmentId?, programId?, target? })
status(assignmentId)
responseRoute(assignmentId)
reviewRoute(submissionId)
reviewerEligibility(programId, target, userId, submitterId?, respondentId?)
eligibleReviewers(programId, target, submitterId?, respondentId?)
```

Commands use the actual `recActor()`; they do not require navigation to a particular page. Reads apply the selected organizational scope and include every person in that scope, not only the acting user. Report adapters can therefore preserve organization-wide person accountability without changing program compliance denominators.

Example internal form assignment:

```js
const assignment = CLEAR_SIGHT_ASSESSMENTS.assign({
  kind: 'form',
  programId: 'PRG-104',
  target: 'Meridian Bank',
  formVersionId: 'FORM-VERSION-ID',
  respondentId: 'USR-02',
  reviewerId: 'USR-03',
  due: '2026-11-30',
  reason: 'Named response and independent review for this target.'
});
```

For an external respondent, pass `respondent: {id, name, kind:'external', organization, email?}` and a known `coordinatorId`. Vendor onboarding can pass a preserved `origin` object identifying its approved dossier, revision and plan line. It may also pass an existing `bindingId`, a `formId`/published `formVersionId`, and configured technical `connections`.

`status(id)` returns `state`, `ready`, `pending`, response/review routes, current and last accepted submission IDs, named respondent, assigned reviewer, deadline, scope, integrity and program-basis currency. `ready` is true only for the latest accepted response with a recorded review receipt, valid source and current program basis, without a newer draft or pending submission.

Reviewer eligibility is shared with contextual Document reviews through both the API and the `atReviewerEligibility` / `atEligibleReviewers` globals. Program ownership is explicit; active permission denies take precedence. Other reviewers need a scoped active approval role with no combined program preparation grant. The respondent and actual submitting person are excluded.

## Work, Inbox and Documents

Work projects named response preparation/correction, whole-response review, missing person assignment and accepted-source requirement decisions. These rows link directly to their authoritative source. They cannot be completed through ordinary remediation-task completion.

Inbox events carry `assessmentSubmissionId`, `assessmentAssignmentId` and `assessmentEvent: 'review' | 'return' | 'accepted'`. Review alerts resolve on a source decision or routing change. Return alerts stay actionable during draft preparation and resolve when the linked replacement is submitted. Read, archive, snooze and acknowledgement remain recipient settings, separate from source decisions.

Documents remain in the canonical Documents library. Responses pin document-version references; they do not copy file bytes or create another document module. Whole-response acceptance and contextual Document acceptance remain independently recorded decisions. Historical source pages use exact version links and the unified contextual Document review route.

## Verification

Commands:

```sh
node --check assets/assessment-tracking.js
node tests/assessment_v210_checks.js
node tests/flow_v29_checks.js
node tests/v210_crossflow_checks.js
```

The dedicated suite passes **22/22 actual-source checks**. It covers unknown legacy provenance, person-isolated answer stores, duplicate assignment reuse, current role eligibility, actor binding, unmapped reviews, owner review, authority revocation and rerouting, immutable source checks, correction/resubmission, canonical history, accepted-conclusion gates, external capture attribution, program drift, missing pinned Documents, asynchronous verification races, stale scope/actor dialogs, designer preview boundaries, navigation and scoped exports.

The current V2.10 flow regression file preserves **25/25** earlier workflow invariants under the new identity and review policy. The original V2.9 version is retained at `tests/history/flow_v29_checks.js`. The separate crossflow suite passes **16/16**, including vendor onboarding, Documents and frozen reporting integration.

Results are recorded in `tests/assessment-v210-results.json`, `tests/flow-v29-results.json` and `tests/v210-crossflow-results.json`. Tests run actual application commands and data models with a substituted DOM/browser shell. Browser layout, screen-reader behavior and browser file persistence were not exercised because browser execution was unavailable in this environment.

## Local demonstration boundary

Named actors, assignments, decisions and recipient state are local demonstration records. The acting-user selector does not authenticate a user. No external supplier message or colleague email is sent. Production deployment needs server-enforced identity/authorization, transactional concurrency, immutable audit storage, durable Documents and notification delivery using these shared source contracts.
