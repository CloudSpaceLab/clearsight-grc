# V2.10 crossflow review and response reporting

## Review scope

This review follows four connected needs: named response tracking and history, independent review of every new program response, one Documents experience for files and their assessment use, and a complete vendor onboarding journey. It examines the existing entry points and domain commands as well as the visible screens. Browser execution remains unavailable; validation uses the actual application sources in the shared Node runtime harness.

## Concrete gaps found in the V2.9 baseline

| Area | Source-level gap | V2.10 owner / intended resolution |
| --- | --- | --- |
| Person-level assignments | `faAttach` deduplicated a form/program/target tuple without a respondent. Two people could not have independent assigned cycles. Canonical assessment answers were also keyed only by program and target. | Assessment tracking adds named assignments and assignment-specific canonical storage. |
| Attribution | Form and canonical submission dialogs accepted a selected known respondent name without requiring it to match the acting user. | Commands bind the capturing actor to `recActor`; the external respondent is a separate identity. |
| Review eligibility | Several response, correction and requirement commands accepted any known nonrespondent reviewer. | Program-owner or authorized independent risk-officer eligibility is checked by the tracking command layer. |
| Unmapped forms | A form without requirement mappings produced an informational notification and no review obligation. | Every assigned new response receives its own independent review; requirement evaluation remains a separate operation. |
| Legacy paths | Public `programRequirement`, `evidenceDetail`, `newVendor` and vendor setup/activation helpers could revive superseded dialogs or command paths. | Owners route or guard compatibility entry points through the current workflows. |
| Document terminology | Separate Evidence and Documents navigation, review links and quality panels described overlapping file workflows. | One Documents experience retains distinct document-version approval and contextual assessment-use review. |
| Vendor onboarding | Initial service creation, assessment assignment, document collection and lifecycle approval were separate journeys without a complete intake dossier. | Vendor onboarding links owner confirmation, independent intake review, actual program/response assignments, exact document requests and activation gates. |
| Accountability reporting | Program report and CSV rows retained requirement posture by target, without respondent, capturing actor, response cycle or independent review history. | The response-reporting adapter adds a separate accountability view and captured annex. Existing compliance denominators are unchanged. |

These findings describe the baseline reviewed before the new modules were integrated. The focused crossflow suite records which intended resolutions have been verified in the integrated V2.10 runtime.

## Response-accountability report

Open **Reports & insights → Response accountability**, or the `assessment-insights` route. The read model comes from `CLEAR_SIGHT_ASSESSMENTS`, including its authoritative current-readiness check. The reporting adapter owns no assignment, submission, acceptance, correction or permission state.

The view provides:

- One current row for each named assignment, with program, target, response kind, respondent, internal capturing actor, assigned reviewer, due date and exact cycle/source identifiers.
- Separate counts for current accepted responses, submissions awaiting review, changes requested, unsubmitted assignments, earlier acceptance needing update, and legacy provenance.
- A review-workload chart by assigned reviewer, with a table equivalent. The count does not imply individual performance, capacity or quality.
- Named follow-up owners and source links for outstanding responses.
- A submitted-cycle register that preserves previous cycles, source fingerprints and independent review identities.
- Filters for program, respondent, reviewer and current response state, within the organizational navigation scope. Invalid filters do not silently broaden the population.

The assignment population is deliberately separate from the requirement population. Several people can respond to one program target, and one person can have several assignments. Neither situation changes the applicable requirement denominator or automatically changes a compliance conclusion.

An earlier accepted response does not imply that the current assignment is ready. A newer draft, pending review, changed source or missing decision receipt can make current readiness false while retaining the earlier accepted record in history.

Legacy records without reliable attribution remain explicit. Reporting does not assign a guessed respondent, reviewer or retrospective acceptance. External respondents are distinguished from the internal person who captures their response.

## Exports and frozen board packs

**Accountability CSV** contains the current assignment population, respondent and capturing identities, reviewer, due date, current state/readiness, source version/hash, latest submission and prior accepted pointer.

**Submission-history CSV** contains the submitted cycles for that same filtered population, with source/version/hash, respondent, capturing actor, submission time, review identity, decision, rationale and previous-cycle reference.

Both exports use the existing CSV escaping helper. They export attribution and provenance metadata, not response answers or document content.

`reportData().assessmentResponses` captures the reporting population and its history. Packs containing the assurance section receive a **Response-accountability annex** based exclusively on that captured data. Reviewer names and other display labels are captured with it; later identity, source, approval or navigation changes cannot refresh the historical annex. Older packs lacking this snapshot field receive no live-data annex.

The local Copilot preview uses the displayed accountability population and keeps its denominator distinct from requirement posture. Published-pack explanations use only the captured annex.

## Integration

Load `assessment-reporting.js` after the tracking, board-report and loss-reporting modules, before the final bootstrap. The reporting adapter adds `reports?view=responses`, registers `assessment-insights`, composes `reportData`, and extends the frozen pack renderer. It also supplies the two CSV actions. Load `assessment-reporting.css` after the shared report styles.

The shared runtime harness must load the new tracking, vendor-onboarding, document-consolidation and reporting modules in the same order as the application before crossflow validation.

Local role and actor checks demonstrate workflow behavior. They do not establish authenticated production identities, external vendor authentication, tenant isolation or a server-enforced data-access boundary.

## Validation status

Validation uses the integrated V2.10 sources, not replacement implementations:

| Suite | Result | Coverage |
| --- | --- | --- |
| `tests/v210_crossflow_checks.js` | 16 / 16 pass | Independent named response stores; actual actor attribution; external capture; unmapped response review; revisions and program-basis drift; scoped report populations; attribution exports; explicit unattributed legacy migration; frozen report annexes; route/action registration; separate response and document review; compatibility document links; vendor intake assignments and activation gates. |
| `tests/loss_reports_v29_checks.js` | 18 / 18 pass | Existing currency/date-basis loss reporting and published snapshot behavior under the new integrated runtime. |
| `tests/board_reports_v28_model_checks.js` | 11 / 11 pass | Existing report read models, KRI comparability, data coverage, source drillthrough and frozen pack regressions. |

Syntax checks pass for `assets/assessment-reporting.js` and `tests/v210_crossflow_checks.js`. The detailed crossflow result is saved in `tests/v210-crossflow-results.json`.

The integrated checks also reproduce and guard the corrected document-review edge case where the assigned program owner is the sole eligible independent reviewer. Validation of the document's source context must not mistake the acting reviewer for a new document submitter and demand a second reviewer.

The Node harness exercises the actual model commands, rendering functions, registered actions and exports with a minimal browser shell. It does not establish browser layout, keyboard behavior, accessibility conformance, authenticated identity or production data-access security. The vendor bridge check proves that real assessment/document obligations are created and that activation is blocked while they are outstanding; it does not claim a live supplier delivery or external authentication service.
