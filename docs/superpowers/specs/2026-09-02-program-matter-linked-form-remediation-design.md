# Program and Matter Linked-Form Remediation Design

**Date:** 2026-09-02
**Status:** Approved design
**Issue:** #140
**Use cases:** UC-PROG-01, UC-EVID-01, UC-EVID-02, UC-FIND-01, UC-HIST-01

## 1. Outcome

A Program owner can open an existing Program-linked issue, select an approved form revision, map the form to the exact information and outcome the issue requires, and send the normal focused request. The immutable submitted response becomes evidence for only the mapped requirements. It does not close the issue. A deterministic outcome check and current authority decide whether the issue is ready for explicit closure.

The resulting path is:

```text
Program
→ open linked issue
→ review existing requests and responses
→ select approved form revision
→ map exact missing items, Action and outcome check
→ normal distribution and capture workspace
→ immutable response revision and artifacts
→ mapped information supplied
→ deterministic outcome check
→ current authorized actor closes the issue when every gate passes
```

## 2. Problem being removed

The current issue workspace renders an ad-hoc **Add information** form for each missing-fact entry. That is useful for a small direct correction, but it becomes a duplicate collection workflow when the bank already has an approved form. Evidence Requests can already target a Matter, yet no current record binds an exact approved form revision, distribution, response revision, missing-item set, Action and verification contract together.

The remediation design removes that duplication. It does not turn every missing fact into a form request: a short authorized direct correction remains valid when no approved collection contract is required. When a linked form exists, the issue workspace presents the linked request or current review action instead of another evidence-entry form.

## 3. Canonical ownership

| Concern | Owner |
| --- | --- |
| continuing obligation and current status | Program |
| bounded gap, Action, decision, verification and closure | Matter |
| approved collection schema and scoring | Form template revision |
| recipient, deadline and access | Form distribution and Evidence Request |
| in-progress response | Response workspace |
| submitted answers and score | Immutable response revision |
| uploaded file | Existing artifact/object-storage boundary |
| evidence sufficiency | Existing evidence assessment/reviewer decision |
| outcome | Matter verification contract and result |
| material closure | Matter transition under current authority |

No parallel questionnaire, evidence-entry record, task queue, artifact store, approval table or closure mechanism is introduced.

## 4. Remediation binding

The new authoritative object is a versioned `MatterFormRemediationBinding`. It identifies:

- tenant and legal entity;
- Program and Matter;
- exact Matter version observed when the binding was proposed;
- approved form template and revision;
- subject type and subject identifier;
- exact missing-item keys mapped to form field identifiers;
- optional current Matter Action;
- one active Matter verification contract;
- purpose, audience class and requested responder class;
- creator, approval state, effective time and record version.

The binding stores identifiers and mapping semantics, not response answers or copied evidence. An immutable activation version is required before a request can be sent. Updating a mapping creates a new version; it does not reinterpret a response already submitted under an earlier version.

At most one active binding may own the same Matter missing-item key at an instant. A separate active binding may collect a different requirement when the scopes do not overlap.

## 5. Configuration and authority

An authorized Matter owner or Program owner may propose a binding. The service validates:

- verified actor tenant and legal-entity scope;
- current visibility of both Program and Matter;
- an active Program link to the Matter;
- the Matter is open and its expected version is current;
- the form revision is approved and active for the legal entity;
- every selected form field exists in that revision;
- every mapped missing-item key exists in the current Matter;
- the subject matches the Matter and request purpose;
- the verification contract is active and compatible with the requested outcome;
- the actor has the current responsibility required to propose the collection;
- maker/checker separation where the form or binding policy requires it.

Activation uses the existing material-command guard and rechecks current authority. Form approval does not approve the remediation binding, and Matter ownership does not grant evidence-review or closure authority.

## 6. Request creation and reuse

Before offering a new request, the service performs exact indexed reads for active bindings, distributions and submitted response revisions on the Matter. The UI shows:

- who is expected to respond;
- the exact form name and revision;
- request and access status;
- deadline;
- response/review state;
- mapped missing items;
- current next action.

**Send linked form** creates or reuses one distribution for the active binding and request episode. A stable idempotency key is based on binding version, recipient episode and collection period. Request creation reuses the standard distribution, invitation, capture, artifact and notification services. It never renders a Matter-specific form.

A current open request is reused. A replacement form revision requires explicit supersession preview and confirmation. A revoked, expired or superseded request remains in history and cannot satisfy the active binding.

## 7. Response application

Submission records an immutable response revision under the normal capture transaction. A bounded worker consumes the existing response event and performs an exact binding lookup. It verifies:

- distribution, request, workspace and response revision identities agree;
- the binding version is still active for application;
- form and subject revisions match;
- the response is final and current;
- required mapped fields are answered;
- required artifacts are available and have passed the applicable scan state;
- the response was not already applied.

The application transaction records a `MatterFormResponseApplied` fact containing safe identifiers and versions. It links the response revision and artifacts to the Matter evidence boundary, marks only the mapped missing items as supplied, and schedules the configured outcome check. It does not rewrite historical `missing_facts`, approve evidence, implement unrelated Actions, pass verification, close the Matter or mark the Program current.

Partial or poor responses remain linked and visible. They produce a review/correction action rather than pretending the request failed to exist.

## 8. Verification and closure

The binding names one active verification contract. After a response is applied, the system runs or schedules the contract's deterministic method. The result remains separate from the response score:

- a good score can still fail a source or outcome check;
- a poor score may create a review action but cannot manufacture a legal conclusion;
- unavailable inputs produce `UNKNOWN` or a configured failure result, never `PASS`;
- an Action can be `IMPLEMENTED` while verification remains pending or failed.

Automatic issue closure is intentionally not a background status flip. When all deterministic closure gates pass, the workflow compiles a current **Close issue** operation for the actor resolved by the current authority route. If the approved Matter subtype and current policy explicitly permit a deterministic auto-close action, that action must be governed by an Automation Policy with blast radius, kill switch, expiry, rollback/compensation and outcome contract. The initial V1 path uses explicit authorized closure.

Closure rechecks current Matter version, Action state, verification result, contradictions, open required decisions, response state and authority. A stale route or changed policy keeps the issue open.

## 9. UI behavior

The issue workspace keeps one dominant next action:

| State | Dominant action |
| --- | --- |
| no linked collection and approved form available | Send linked form |
| request open | Review request status |
| response received | Open response |
| response incomplete or poor | Request correction |
| evidence awaits review | Review evidence |
| outcome check ready | Check outcome |
| outcome passed and closure gates satisfied | Close issue |
| authority unavailable | Resolve assignment or route |

The missing-information section groups direct corrections separately from governed form collection. Mapped items show their current request/response state and no duplicate **Add information** control.

Desktop keeps issue context beside the focused sheet. Tablet moves context below the task. Mobile replaces the workspace with a full-screen focused flow. Selects and sheets use overlays that do not resize or scroll the left outline. Light/dark, comfortable/compact, keyboard, focus, reduced motion, 200% reflow and axe evidence are required.

## 10. Failure and recovery

- A stale Matter, form, binding, distribution, response or authority version fails closed.
- A submission already committed remains submitted if later application or projection fails.
- Worker retries are inbox-idempotent and cannot duplicate evidence links or outcome jobs.
- Notification failure does not revoke a valid request; delivery recovery is explicit.
- Revoked access cannot submit, and revoking one recipient does not erase other valid contributions.
- A superseded form does not reinterpret prior answers.
- Subject or legal-entity mismatch is rejected before any Matter mutation.
- Tokens, recipient addresses, raw answers and artifact content are excluded from event/outbox payloads and logs.
- The workflow remains operable without AI or email by using Today and the exact request workspace.

## 11. Data and performance

Known records use exact identifiers; no broad Program/Matter/distribution population replay locates a binding or response. Lists use bounded keyset queries. New authoritative rows, event/outbox facts and required maintenance jobs share the material transaction. High-volume joins are indexed by tenant, legal entity, Matter, active binding and response revision.

Point-in-time reconstruction must show the exact Program/Matter/form/binding/distribution/response/verification/authority versions used for collection, review and closure.

## 12. Acceptance

The design is complete when executable and hosted evidence proves:

1. an authorized Program owner opens an existing linked issue and sees prior requests;
2. they select an approved form and map exact missing items and an outcome check;
3. one distribution is created and the intended respondent completes the normal capture flow;
4. the immutable response revision is linked and only mapped items become supplied;
5. partial, poor, stale, replayed, revoked and mismatched responses keep the issue open safely;
6. a passing deterministic outcome produces the current close operation;
7. only a currently authorized actor can close;
8. failure or authority change leaves one clear next action;
9. Program state and oversight update from canonical Matter/evidence truth;
10. backend, PostgreSQL, route/OpenAPI, TypeScript, copy, rendered-state and accessibility gates pass on the exact deployed revision.
