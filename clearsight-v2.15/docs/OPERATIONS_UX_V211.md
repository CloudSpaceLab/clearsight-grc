# Operational interaction improvements · V2.11

## Changes and their purpose

The adapter targets points where users had to search a long record, infer which field to fix, or mentally compare prepared information with the effective record. It uses the existing source models, validators and commands. It creates no parallel workflow store and makes no automatic review decision.

| Existing friction | V2.11 interaction |
| --- | --- |
| The 51-question KRI review expanded every observation, and validation links led only to a section. | Review starts with an attention register containing exceptions, unknown observations and questions with unresolved submission checks. Each row opens the exact question. All details, Exceptions and Unknown / unavailable remain available. |
| KRI completeness could be mistaken for submission readiness. | Recorded values, N/A, reasoned unavailable answers and unanswered fields are shown separately. The canonical validator determines submission blockers. A valid reason for unavailable data does not invent a measurement or imply complete coverage. Readiness updates as saved answers change. |
| Loss review errors required the user to infer the appropriate step. | Each check names the capture section and opens the exact field. The existing draft command saves the current step before navigation. Per-section checks update from the displayed fields; the final step shows all outstanding checks. |
| A loss correction repeated an entire proposed record, making changed facts hard to find. | Correction review shows accepted versus proposed changed fields first, with the full submitted correction in an expandable source section. Currency changes also display the amount in each original currency. |
| Receipt and reversal decisions were buried below the financial narrative. | A pending-decision queue appears before the long record, with existing independently guarded verification/reversal actions. Section shortcuts reach facts, recoveries, the action plan and history. Receipt/closure actions that are already blocked by pending source decisions are visibly disabled. |
| The RCSA comparison ignored the retained first-line draft until submission. | The comparison reads the canonical saved draft and shows its likelihood, impact, resulting score and relationship to appetite beside the effective rating. A submitted proposal with a changed risk/control/effective-rating basis is visibly flagged. |
| Control reviewers had to reconcile a test recommendation with the effective control record. | Test preparation/review displays the effective conclusion alongside the prepared or submitted recommendation. A changed implementation is flagged; historical accepted tests retain explicitly historical wording. |
| Risk and control editors had no concise view of prepared changes. | A conditional, expandable comparison appears only when draft facts differ from the saved record. User-entered text is escaped. Risk overview points to the existing assessment or review handoff while preserving the context editor. |
| Loss concentration, aging and recovery-prospect charts stopped at an aggregate. | Chart labels open the exact captured source-event subset, retaining date basis, date window, grouping and currency. A subset CSV is available. Invalid or no-longer-available groups do not silently broaden the selection. |
| A board-pack validation failure left the author on the preview step. | Each publication check opens the appropriate draft step and exact field. It retains the captured snapshot, commentary and agenda. The existing publication command still performs the decision and validation. |

## Workflow and calculation boundaries

KRI attention uses `k28Snapshot`, `ixBand` and `k28Issues`. Approved snapshots and source definitions are not rewritten. The attention view does not classify contextual questions as appetite tests or treat N/A as a numeric observation. Complete source detail remains accessible.

Loss preparation uses `l29Issues`, `l29Draft` and `l29SaveDraft`. The preparer and destination permissions are rechecked by the existing save command before a field jump. A submitted correction remains separate from the accepted record. Pending receipts and reversal requests remain outside the verified financial balance until the existing independent command records its decision. The new queue does not supply another approval path.

RCSA and control previews are read-only interpretations of the existing saved draft or submitted snapshot. A preview never changes residual ratings, control effectiveness, requirement conclusions or accepted evidence. Existing source-drift and independent-review guards remain in force.

Report drills intersect canonical concentration/age/prospect IDs with the records already captured by `lrpCapture`. They do not requery an unscoped event ledger, mix currencies, reinterpret dates, create estimates or change the full report denominator. The source selection affects its on-screen list and dedicated subset CSV. Board packs and the main cohort CSV retain the full selected report cohort; the interface states that distinction beside the drilldown.

Frozen loss annex rendering returns through the original renderer before live drill links or source panels are added. Existing published pack data and historical output remain unchanged.

## Integration

Load `assets/operations-interactions.js` after the existing domain/report modules, `interaction-core.js`, and the other interaction adapters. Load its CSS after the shared interaction and domain styles. It uses `ui211Route`, `ui211Focus`, `ui211Issues` and `ui211Next` from the shared core.

The adapter composes the existing route functions and small rendering helpers. It does not boot the application or wrap the global render loop. Its listeners update only the relevant comparison/check region after the original source listener has saved canonical draft data. Expanded check/comparison sections remain expanded while their content updates.

Owned deliverables:

- `assets/operations-interactions.js`
- `assets/operations-interactions.css`
- `tests/operations_interactions_v211_checks.js`
- `tests/operations-interactions-v211-results.json`
- This document.

One narrow existing-test update was authorized during integration: the risk monetary regression previously used the exact generic heading “Keep the risk context current” as evidence that the workflow wrapper remained reachable. It now verifies the retained `wf29-risk-edit` action and the contextual assessment handoff. Every monetary, currency, scope, filter and snapshot assertion is unchanged. The original test is retained at `tests/history/loss_reports_v29_checks.pre_v211.js`.

## Validation

| Actual-source suite | Result |
| --- | --- |
| `operations_interactions_v211_checks.js` | 16 / 16 passed |
| `board_reports_v28_model_checks.js` | 11 / 11 passed |
| `loss_reports_v29_checks.js` | 18 / 18 passed |

The operations suite verifies exact KRI destinations, unknown/unavailable semantics, accepted-snapshot preservation, save-before-field-jump behavior, independent loss/recovery guards, correction comparisons, saved RCSA previews, stale-source rejection, control history, escaped draft text, currency/date-preserving report drills, subset exports and field-specific board publication recovery. Route checks exercise the changed pages and their registered actions.

JavaScript syntax checks pass. A focused independent source review also confirmed the loss-navigation guards, pending-financial-action conditions, report-cohort intersection and early frozen-render return.

Validation uses the real application sources in the Node runtime harness with a minimal browser shell. Browser rendering remains unavailable, so this does not claim visual, keyboard or assistive-technology conformance. The local role checks demonstrate workflow behavior; authenticated identities, secure data retrieval and server-side authorization remain production integration responsibilities.
