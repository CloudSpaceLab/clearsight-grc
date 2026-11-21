# ClearSight V2.10 verification report

20 November 2026 · Synthetic scenario as of 13 November 2026

## Verified scope

Actual application JavaScript executes in Node through `tests/runtime_harness.js`; only browser/DOM APIs are substituted. Tests exercise model, command, identity, state, money, source-version, routing and registered-action behavior. Real-browser layout, storage, downloads and print are not verified.

| Suite | Script | Passed |
| --- | --- | ---: |
| Branch KRI source model | `kri_v28_checks.js` | 11 |
| KRI reviews and corrected actions | `kri_review_v28_checks.js` | 7 |
| Roles, access and escalation | `governance_v28_checks.js` | 14 |
| Detailed Inbox | `inbox_v28_model_checks.js` | 17 |
| Existing board reports | `board_reports_v28_model_checks.js` | 11 |
| Loss capture and workbook import | `loss_v29_checks.js` | 12 |
| Independent loss edge review | `loss_edge_review_v29_checks.js` | 10 |
| Controls, RCSA, vendor lifecycle and risk editing | `workflow_v29_checks.js` | 14 |
| Forms, Documents, privacy, Work and source handoffs | `flow_v29_checks.js` | 25 |
| Loss reporting and captured context | `loss_reports_v29_checks.js` | 18 |
| Named assessment identity, history and review | `assessment_v210_checks.js` | 22 |
| Unified Documents and contextual review | `documents_v210_checks.js` | 12 |
| Vendor intake through reviewed activation | `vendor_onboarding_v210_checks.js` | 15 |
| Independent V2.10 crossflow and reporting review | `v210_crossflow_checks.js` | 16 |
| **Model/workflow total** | | **204/204** |
| Routes/actions: 73 entry routes + 10 report perspectives | `routes_v210_checks.js` | **83/83** |

## Material V2.10 scenarios

The assessment tests cover independent same-target person stores, actual-actor submission, external respondent versus capturing coordinator, all assigned submissions requiring review, current owner/role eligibility, revocation, explicit denies, self-review rejection, reassignment, legacy unknown provenance and immutable corrections. Accepted prior conclusions remain while replacements are pending; stale program/form bases fail readiness.

Document tests and the adapted flow regression exercise one file/version store, canonical context routes, submitter/uploader distinction, assigned eligible reviews, actual-byte verification before acceptance, invalid-source no-partial-mutation, return/resubmission and accepted supersession history. The crossflow suite verifies that a sole eligible assigned program owner can review and that whole-response, document and requirement decisions remain distinct.

Vendor tests complete intake, owner confirmation, independent risk approval, generated program and named assignments, returned-intake revision, exact-version document approvals, source/validity gates and reviewed activation. They also cover actual identity, scoped creation authority, explicit-deny precedence, legacy routes and no activation from attachments or submitted questionnaires alone.

The reporting review checks named/source attribution, scope, two history/current exports, unchanged compliance denominators and immutable old/new board packs. Inherited KRI, governance, Inbox, control/RCSA, privacy, financial loss and report contracts are rerun or checked under the current runtime. The final crossflow rerun includes the final vendor authority changes.

## Build and packaging checks

- 26 JavaScript source modules and both standalone inline script blocks compile.
- 73 entry points resolve all 3,358 shared asset references.
- The standalone starts at Programs; no external JavaScript or stylesheet dependencies.
- Release manifest records file bytes and SHA-256 hashes; `verify_reference.py` verifies an untouched package.
- ZIP integrity and exact standalone-copy equality are checked before saving.

## Running and retained fixtures

Run `python3 build.py`, then `node tests/<script>` for each suite above. Names retaining `v28`/`v29` intentionally exercise inherited contracts against current sources. The original flow and vendor lifecycle fixtures changed for the new actual-actor and mandatory-review policy are archived under `tests/history/`; the updated versions keep the prior scope, version, independent-decision, correction and lifecycle invariants.

Detailed assertions are in the results JSON files; `tests/validation-summary.json` is the current authoritative summary. Older screenshots, browser scripts and results remain historical.

## Browser and production limits

The environment blocked Chromium process/socket creation and rejected local-file cloud-browser navigation. Those restrictions were not bypassed. Source/CSS review and substituted-DOM route checks do not establish rendered layout, keyboard/screen-reader usability, mobile overflow, real browser persistence, file interaction or print behavior. Those checks remain a production handoff gate.

This is an interactive design package with synthetic records and local identities. Tests do not certify authenticated server authorization, concurrent writes, secure file hosting, real supplier invitations, notification delivery, scheduling or enterprise integrations.

Run `python3 verify_reference.py` on an untouched release. Rebuilding or rerunning tests can change manifest hashes; regenerate intentionally for a later release.
