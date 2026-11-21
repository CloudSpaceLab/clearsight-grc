# Assurance interactions — V2.11

## Purpose and scope

This change makes the existing assessment, program adoption, Documents and vendor onboarding workflows easier to complete. It directs users to the next source record or missing field while retaining the V2.10 named-person, independent-review and immutable-source contracts.

The implementation lives in `assets/assurance-interactions.js` and `assets/assurance-interactions.css`. It is a presentation adapter over the existing assessment tracking, Documents and vendor onboarding modules. It creates no additional response, document, approval, assignment or readiness store.

## Findings and implemented changes

| Observed friction | Interaction change | Canonical source retained |
| --- | --- | --- |
| Assignment rows offered “Continue response” to people who could only inspect another person’s draft, and a separate Review action competed with it. | Each row has one principal action derived from the current actor, named respondent, assigned reviewer and response cycle. History and assignment management remain secondary. | `atStatus`, `atCanRespond`, `atCanManage`, `atReviewerEligibility`, exact response/review routes. |
| Program overview showed a target-level Draft/Submitted summary that could disagree with the named response assignments. | Its existing Next assessment panel becomes Next response responsibility, with the actual responsible person, current state, due date and exact source action. | Named assignments and current source submissions. |
| Long responses produced unstructured submit errors without directing the user to a question. | The response checklist lists outstanding answers, current supporting documents and uncertain-check explanations. Selecting an issue focuses the actual control. The all-question navigator remains available. | `faErrors`, `faVisible`, canonical question policies, `qualifiedEvidence`, date and number validators. |
| Grid validation named a missing value but did not identify the field to edit. Document answers use picker buttons rather than text inputs. | Missing grid cells and matrix rows receive exact selectors. Scalar and grid Document fields focus their real picker buttons. Calculated-field errors point to missing human inputs where available. | The existing field renderer, Document picker action and source-validation rules. |
| Reviewer choices were collected across several target options and could include the chosen respondent. | The assignment dialog refreshes eligible choices for the selected program, target and respondent; external coordinators are excluded from their own reviews. | `atEligibleReviewers`; `atAssign` revalidates on save. |
| Reviewers had to scan the entire source page to locate the decision, documents or changes. | Compact section controls focus those locations. A revision disclosure summarizes changed answers, observations, document versions, document-review context, response explanation and external capture reference. Full prior and current snapshots remain available. | Immutable submission snapshots and existing source-history links. |
| Publishing a program landed on the generic overview before anyone had been named to respond. | Publishing hands off directly to the selected target’s named assignments. Adoption copy distinguishes participating teams from named respondents. Final-check errors open the relevant adoption step and field. | Existing publish command, program baseline and assignment command. No person is inferred from a team label. |
| The Documents queue mixed all responsibilities behind generic Open document links. | Pending reviews can be filtered to My review decisions or Submitted by me. Actions distinguish an actual review from inspecting another person’s submission. | `d210ReviewEligibility`, scoped contextual review records and exact version routes. |
| Leaving a Document context sent users to a generic assessment chooser, and reviewers lacked a direct next pending review. | Where the relationship is unambiguous, return links open the exact submitted response or editable named response and question. Otherwise they open the selected target’s assignment list. Reviewers can open their next pending Document. | Pinned submission-document links, named assignments, current scope and original contextual review IDs. |
| A blocked vendor activation always offered Review assignments, even when questionnaires were already accepted and the remaining work concerned documents. | The existing next-task area selects the current actor’s actual remaining responsibility, then due date. The dossier lists precise response, document-approval and requirement-conclusion blockers, each with an exact destination. | `voReadiness`, `voDocumentStatus`, `atStatus`, accepted-source selection and current requirement evaluations. |
| Vendor final checks listed problems without returning users to the appropriate intake step. | Each issue saves the existing intake draft, opens the relevant step and focuses the specific field. | `voIssues`, `voDraft` and `voSaveDraft`, including preparer and scope validation. |

## Interaction behavior

### Assessment preparation and review

“My next actions” means the current actor can perform the relevant response or review responsibility. A person viewing someone else’s draft sees View assigned draft. A pending review opens the exact immutable submission; the action is labelled Review response only for the assigned eligible reviewer. A returned response resumes its newly created correction binding and displays the recorded review comments.

The form checklist replaces the previous Assessment status panel. Canonical assessments reuse the existing Question navigator area. Required answers and documents are separate from technical-check results and requirement conclusions. A recorded failed source check may still be submitted for review; an unknown or missing check still requires the existing explanatory note. Conditional fields that do not apply are excluded using the source form’s visibility rule.

If an editable response has outstanding issues, the Submit action focuses the first one and keeps the draft open. A valid response still goes through the original confirmation dialog and `atSubmit`. This adapter does not submit as a selected name, relax reviewer independence, modify an accepted response, or accept a source on the user’s behalf.

### Documents

The pending queue retains one row per contextual submission rather than merging different assessment uses of the same file. My review decisions includes only pending records for which the current actor passes the existing eligibility check. A submitter cannot appear as their own eligible reviewer.

Opening the next pending Document does not decide the current one. After a decision, the source command removes it from the pending population and its original receipt remains available. The exact file version and contextual review ID remain in the route. File-byte verification, source availability, expiry and post-await actor/version rechecks remain in `d210Review`.

### Vendor onboarding

The blocker list is a navigation read model, not a second readiness engine. It consumes the original readiness results. A missing document leads the named provider to Choose current document; a pending approval leads to the exact approval request and its current reviewer. An accepted questionnaire with an unresolved requirement leads to the source requirement review. A conclusion already recorded against that source is labelled for inspection and follow-up rather than presented as another undecided review.

The original activation entry, lifecycle submission and lifecycle approval gates remain unchanged. Selecting a blocker cannot make the service active, mark a document approved or alter a requirement conclusion.

## Integration

Load the adapter after `assessment-reporting.js` and `interaction-core.js`, with its CSS after the existing domain styles. It does not render during module initialization.

The adapter uses the shared `ui211Issues`, `ui211Focus`, `ui211Next` and `ui211Route` helpers. Source commands remain in their original modules. Existing response and vendor journey registrations are refreshed where those registrations had captured an earlier function reference.

The optional read-model API is `window.CLEAR_SIGHT_ASSURANCE_UX`, version `2.11`:

| Method | Purpose |
| --- | --- |
| `assignmentNext(assignmentId)` | Current source destination, responsible person and actor-appropriate action. |
| `responseIssues(assignmentId)` | Current draft’s field, document and explanation issues with focus selectors. |
| `reviewerChoices(programId, target, respondentId, coordinatorId?)` | Existing eligible reviewer population for the selected response responsibility. |
| `revisionChanges(submission)` | Differences from the prior immutable response cycle; scope is rechecked. |
| `documentQueue(mode)` | Scoped pending contextual reviews: `all`, `mine` or `prepared`. |
| `vendorBlockers(dossierId)` | Precise incomplete source responsibilities from the existing readiness model. |
| `adoptionIssues()` | Current adoption validation issues with their configuration step and field. |
| `intakeIssues(draft)` | Existing intake validation issues with their preparation step and field. |

The assignment `responsibility` filter and Document `queue` filter use the route query, preserving the program, organizational target and existing view context. The adapter adds no alternate filter store. General search, dialog dismissal and asynchronous modal submission remain the responsibility of `interaction-core.js`.

## Verification

`tests/assurance_interactions_v211_checks.js` runs the actual integrated application modules in `tests/runtime_harness.js`. Its final result is **17 passed, 0 failed**, recorded in `tests/assurance-interactions-v211-results.json`.

The checks cover:

- Actor-correct assignment actions and query-context retention.
- Assigned reviewer eligibility, including revoked authority.
- Correction binding selection and immutable prior source history.
- Canonical answer/document blockers and Submit focus without a source mutation.
- Exact grid-cell, matrix-row and real Document-picker focus destinations.
- Conditional-field exclusion and unknown-versus-failed technical observation behavior.
- Reviewer options that exclude the respondent or external coordinator.
- Revision comparison against immutable submissions.
- Scoped pending Document populations and asynchronous source-owned acceptance.
- Exact Document-to-response return routes without guessing among people.
- Vendor blocker destinations after all questionnaires are accepted.
- Exact pending document approval routing with the approval still pending.
- Adoption issue navigation and the publish-to-named-assignment handoff.
- Intake error navigation, retained draft facts and preparer restrictions.
- Registration of rendered source actions and destinations.
- Program overview’s actual named review responsibility.

The inherited assessment, Documents, vendor onboarding and crossflow suites also passed during integration: **22, 12, 15 and 16 checks**, respectively. Root additionally verified the complete inherited suite against the integrated V2.11 build.

These are source-command, rendered-markup and substituted-DOM interaction checks. They do not claim a live-browser layout or accessibility audit. Browser access remained unavailable and was not retried. The CSS follows the existing responsive breakpoints; real pointer, keyboard and visual review remains a deployment handoff check.

## Completion

The JavaScript and CSS are frozen after the 17-check run. Only new assurance interaction assets, this document, and the dedicated test/results files were changed by this work. No original source record schema, financial calculation, review receipt, identity command or activation gate was modified.
