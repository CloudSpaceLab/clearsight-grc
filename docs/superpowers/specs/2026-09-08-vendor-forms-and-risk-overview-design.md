# Vendor form assessment and risk overview

Date: 2026-09-08
Status: Approved by user on 2026-09-08; implemented and locally verified; approved for merge
Baseline: freshly fetched origin/main at 60a6a6065b7ca6de7f52b66dd24c36fb19fc0904
Use cases: UC-TPRM-01, UC-TPRM-02, UC-EVID-02

## Decision

Extend existing Forms and Vendors so the bank can configure how each field is assessed, manage the associated rules and policies from the UI, request existing forms from one or several vendors, and identify outstanding responses, unfinished bank assessments and poor results from the vendor record.

Document-to-form creation already exists. All creation paths continue to produce the same ordinary form draft. This change must not introduce another importer, form builder, scoring engine, distribution service, policy engine or vendor-assessment lifecycle.

Recommended approach: field assessment controls in the current builder; usable policy configuration through the existing Forms Policies and Configure surfaces; vendor-scoped summaries and links into the current request/response workflows. A separate vendor-only engine duplicates shared capabilities; dashboard-only changes leave manual assessment and configuration usability unresolved.

## Corrected and synchronized baseline

The first draft mistakenly used the old codex/vendor-management checkout at 3c5b2e5e as the current product baseline. After the user's instruction, remote changes were fetched and this checkout was fast-forwarded to 60a6a606. HEAD was verified equal to origin/main, with no tracked working or staged changes. The previous 15 edited files are preserved in the named stash preserved-before-vendor-main-update-2026-09-08; untracked artifacts were retained.

Current code already includes:

| Existing capability | Reuse point |
| --- | --- |
| Uploaded document to reviewed form proposal and ordinary draft | FormProposalReview, FormsWorkspace and Imports handoff |
| Typed fields, immutable revisions, approval and reusable forms | FormBuilder and internal/formcontract |
| Risk/compliance scoring, contributions, typed AND/OR/NOT conditions, floors/caps, disqualification and server preview | AdvancedScoringEditor and advanced_scoring.go |
| Immutable response scores, coverage and score-filtered current/history queries | Forms Responses and evidence response revisions |
| Governed response policies, simulation, approval, shadow mode, activation, suspension, rollback and issue deduplication | FormPoliciesView, FormPolicyEditor and internal/formpolicy |
| Distribution, recipients, deadlines, expiry, secure access and communications | DistributionComposer, formsDistributionApi and Forms Sent forms/Communications |
| Vendor due diligence, requests, documents, findings, review and vendor-linked work | VendorsWorkspace, VendorDueDiligence, VendorWorkPanel and DocumentBrowser |
| Bank acceptance/rejection of proposed changes to held vendor values | VendorResponseReview |

The reviewed field contract does not define general manual bank scoring or immutable per-field scoring judgements. Acceptance of a proposed vendor address change is not an assessment of its risk. FormPolicyEditor still requests raw automation-policy IDs/revisions and comma-separated stored subject types. Vendor rows do not summarize all response-completion and risk work. These are the change targets; deployment acceptance is not inferred from the existence of code.

Before behavior changes, read the updated root documentation, relevant product specifications, implementation ledger and acceptance tests. Preserve current main behavior and the saved old work; do not reapply obsolete changes over the newer form system.

## 1. Field assessment in the existing builder

Every field exposes **How this field is assessed**:

| Mode | Result |
| --- | --- |
| Not scored | Collect information or evidence without adding points; existing required-response and evidence checks still apply. |
| Bank review | An eligible bank reviewer applies the approved rubric and records a judgement with rationale. |
| Automatic rules | Existing deterministic rules calculate the contribution. |
| Automatic rules, then bank review | Rules suggest a contribution; an eligible bank reviewer confirms it or records a permitted adjustment before assessment is final. |

All imported, manually authored, starter and AI-proposed drafts use the same controls. Legacy forms retain their existing behavior; changes create explicit new revisions.

Bank-review configuration includes a named rubric with outcomes mapped to points or a bounded numeric scale, weight, required/optional assessment, applicability and reviewer responsibility/route. Rationale is required for manual judgement or adjustment. Document fields can require an assessment of evidence sufficiency or validity; upload alone must not pass them.

Automatic scoring reuses existing typed operators, conditions, weights, bands, missing-input semantics and critical overrides. Use labelled field/operator/value selectors and the existing exact-saved-revision **Test rules** preview. Do not add scripts, expression JSON or a second calculator.

Combined assessment retains the original automatic result and reviewed contribution. Confirmation cannot bypass a critical override, floor or approved condition. Any allowed adjustment requires its configured authority and reason; the default is to flag a disputed result for review rather than grant unrestricted override.

Add a form-level **Assessment and scoring** summary listing each field's mode, rule/rubric, weight and reviewer, with a link back to the field. Show incomplete configuration before approval. Reuse simulation, impact preview, independent approval, effective dating, rollback and history.

## 2. Assess submitted fields without changing vendor answers

Extend existing response detail with vendor answer/evidence, automatic rule result, approved rubric, bank judgement, explanation, reviewer and assessment time. Provide **Needs bank review**, **Poor results**, **Missing evidence** and **Reviewed** filters.

Bank scoring decisions are separate from respondent answers, document-validation decisions and held-record application decisions. Each versioned decision references the exact response revision, form revision, field, rubric/profile version and verified reviewer, with score/outcome, rationale, evidence references, timestamp and supersession. Corrections create successor decisions.

Extend the shared evaluator to combine applicable automatic contributions and approved bank decisions into a versioned assessed result. Preserve the original submission score and the raw/adverse direction distinction: high risk and low compliance can both mean concern. Never overwrite a respondent answer to encode a bank judgement.

Keep three states separate:

- Vendor response: awaiting response, in progress or submitted.
- Bank assessment: not required, awaiting review, in review or assessed.
- Risk/findings: result, coverage, freshness and open governed findings.

A submitted form can await bank review. Required unassessed fields keep the bank result provisional, not zero risk. Response coverage and bank-assessment coverage have distinct denominators; hidden conditional fields follow the exact form version.

Superseded answers/documents require current-revision assessment; prior judgements remain historical and cannot silently carry forward. Bank review exposes only authorized submitted data. Permitted draft progress summaries can show saved counts or missing question labels without exposing unsubmitted answer values or draft files. Unsaved browser input is unknown.

## 3. Manage configuration through existing UI surfaces

Forms Policies remains the entry point for response handling, linked from the form's assessment summary and from the policy receipt on a vendor response. Reuse the existing Configure, authority and Automation Policy records.

Replace raw identifier input with scoped named selectors for eligible automation policies/revisions, forms, subjects, responsibilities and supported outcome actions. Show purpose, current status, effective dates and where a policy is used. IDs remain specialist/audit details.

Where an underlying policy lacks a usable editor, add a contextual create/revise flow backed by its canonical lifecycle. A bank user must complete setup without SQL, API calls, JSON or copying IDs. Only offer actions supported by the executor; unavailable actions explain their limitation.

Keep ownership clear while providing connected UI:

- Assessment rules/rubrics/weights/bands belong to the form revision.
- Reviewer routes, response deadlines, renewal/reminder policies and clarification handling belong to existing workflow/configuration services.
- Response eligibility, result basis, supported issue handling, limits and outcome checks belong to existing formpolicy/Automation Policy records.

Expose the trigger, population, action, owner, approval state, effective/expiry dates, monitoring and execution history. Reuse simulation, impact preview, maker-checker approval, shadow runs, activation, suspension/kill switch, rollback/compensation and outcome checks.

Extend policy eligibility to distinguish automatic submission results from completed bank-assessment results. New policies depending on manual judgement default to the completed assessment and cannot act on provisional reviews. Existing automatic-only policies retain recorded behavior. Simulation and receipts explicitly identify the result basis.

Assessed-result events reuse current outbox/inbox, authority and adverse-episode deduplication. Automatic scoring and bank assessment of the same adverse response must not create duplicate issues. A later score cannot silently close a finding or verify remediation.

## 4. Request and manage existing forms from a vendor

Add a prominent **Forms and responses** section within vendor detail. **Request form** opens the existing distribution/request experience with vendor/service context preselected, an approved existing form chooser, permitted recipients and deadline. Reuse current bank-held facts/evidence. New form creation links to the existing authoring flow.

Preserve assessment-scoped and vendor-work origins, permissions and review transitions. Generic distribution cannot impersonate an assessment or advance its lifecycle. Present business purposes instead of asking users for internal subject codes.

List all authorized exact-linked vendor/service/workflow requests and submissions: form, purpose, recipient hint, deadline, response state, bank-assessment state, score/band, latest submission time and next action. Open existing response detail, documents, field assessment, clarification or governed issue directly. Current and historical responses remain distinguishable and separately paginated.

Allow multi-vendor selection from the register. Reuse the current distribution service with a separate correctly scoped subject target per vendor/service; multiple recipients for one subject are not multiple vendor subjects. Preview relationships, form revision, recipients, deadlines, existing equivalent requests and blocked targets. If multi-subject orchestration is missing, add only a bounded idempotent batch receipt over existing commands.

Preserve delivery, access, revocation, recipient changes, optimistic conflict, communication-template and retry behavior. Receipts distinguish creation, delivery and submission. Retry only failed/pending targets. No real vendor messages are sent during development without explicit recipient authorization.

## 5. Outstanding work and risk at a glance

Retain service context and support vendor grouping without name-based merging. Rows summarize:

- Outstanding and overdue forms.
- Submitted forms awaiting bank assessment and required fields left to review.
- Current assessed concern and coverage, separately labelled provisional automatic results.
- Open findings by severity and nearest outstanding deadline.
- Latest response/result time, stale state or unavailable data.

Filters: **Awaiting vendor**, **Awaiting bank review**, **Submitted with risks**, **High or critical concern**, **Open findings**, **Overdue**, **Not assessed**, and a specific form. Groups can overlap.

For multiple forms/services, label the roll-up **Highest assessed concern**, identify the assessed population and use the existing adverse-score semantics. Do not average incompatible profiles or imply an approved overall vendor rating. An outstanding request may coexist with a previous risk result.

Each count/band opens its contributing requests, missing items, unreviewed fields, poorly scoring answers or findings. Findings show description, implication, severity, owner, deadline, state and evidence/outcome-check links. Completing a response or every bank review does not imply the vendor has no remaining findings.

## Workbook acceptance example

Use the existing uploaded-document-to-form flow for Sample Third-Party Risk Register (1).xlsx. Read-only inspection found two anonymized service entries and five open Medium findings in worksheet 2026 Register, rows 2–6, with merged context and no numeric scoring formulas. Preserve exact anchors and unresolved mappings through the existing proposal review.

Repeated provider name xxxxx must not become an automatic identity match. Imported severities remain attributed source statements. Bank assessor identity, recommendations, ratings and closure are not vendor-editable merely because they were workbook columns.

After conversion, configure a mixed example using the same builder: certification evidence, a vulnerability-test document requiring bank review against an approved rubric, and a structured control answer evaluated automatically. Source findings provide test context, not an automatically approved scoring policy or legal conclusion.

## Architecture, authority and performance

Extend shared form contracts and evidence response assessments rather than adding a vendor-only answer/score store. Thirdparty retains vendor/relationship/workflow associations; canonical Matters own findings and outcome handling.

Commands bind verified actor, tenant and legal entity and reevaluate current authority. Required assessment state, append-only events, outbox and maintenance work commit together. Post-commit calculation/projection failure returns a committed receipt with recovery state.

Scope reads before counts, filtering and pagination, including restricted Program/issue-linked responses. Vendor ownership alone cannot grant response access. Use exact workflow links, indexed current-response queries and keyset pagination; avoid browser aggregation of entire populations and per-vendor fan-out.

Add only needed response-revision/field assessment and vendor/service link indexes. Expose count availability, observation time and form/response/assessment versions. Preserve history and existing retention. Extend existing load fixtures and performance targets with grouped vendor counts and assessment detail, recording cardinalities, query plans, concurrency and latency. Do not introduce partitioning or destructive retention without measured need.

## Acceptance and UI proof

Preserve before-state renders from current main. Verify:

1. Existing document-generated and manually created forms share assessment controls and retain source/version/approval data.
2. Mixed automatic, manual, combined and unscored fields round-trip through save/reopen/approval; unchanged legacy forms score identically.
3. A vendor's fully submitted response remains Awaiting bank review until required bank decisions exist, with truthful provisional risk and separate coverage.
4. Field decisions retain evidence, rationale and identity; conflicts, revocation and supersession cannot silently change current scores.
5. Rule/policy setup, simulation, approval, activation, suspension and history work entirely from the UI without copied IDs.
6. Automatic and bank-assessed policy results respect their configured basis and reuse the same adverse episode.
7. Existing due-diligence, vendor-work and ordinary form requests/responses appear in the right vendor context with current permissions.
8. Two vendor requests retain isolated subjects and per-target delivery receipts; retries do not duplicate successful work.
9. Incomplete response, submitted poor result, awaiting bank review and completed assessment with open findings are distinguishable and open to precise contributing items.
10. Missing/unavailable/stale data, conditional fields, historical responses, restricted linked work and invalid documents never imply zero risk or completion.

Run affected Go/API/unit tests, configured PostgreSQL integration, UI tests, copy-quality regression, typecheck, build and UI contracts. Do not claim skipped database/deployment checks passed.

Render builder assessment controls, policy setup, vendor overview, request preview and response review in light/dark themes at desktop 1440px, tablet 1024px, mobile 390px/320px and 200% zoom. Mobile replaces dense rows with labelled summaries and stacks answer/evidence/assessment in reading order. Check keyboard/focus, axe, reduced motion, long content, loading, empty, denied, stale and failure states. Fix and recapture the highest-impact defect; notices/footers must not obstruct primary actions.

Synchronize current README, DESIGN where patterns change, governed-forms specification, implementation ledger, API/schema ownership and acceptance/rendered evidence. Scope remains assessment configuration and connected vendor operations, using existing product capabilities.

## Delivery order

1. Review this corrected current-main scope.
2. Add shared field assessment and bank decision/assessed-result contracts with meaningful failing tests first.
3. Connect usable rule/policy configuration and assessed-result eligibility.
4. Integrate vendor requests, response lists, multi-vendor selection, summaries and exact drill-down.
5. Verify the workbook/mixed assessment journey, authority, recovery, copy, responsive behavior and performance.

Repository synchronization preceded implementation. The user approved this design on 8 September 2026; the local implementation and verification receipts are recorded in [vendor-form-assessment acceptance](../../acceptance/vendor-form-assessment.md). No live vendor requests have been sent by this delivery.
