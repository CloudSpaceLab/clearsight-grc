# Fidelity/Archer requirements — Control Tower acceptance map

Source: user-provided `Fidelity Bank_Archer (1).xlsx`, Sheet2, 31 numbered requirements. This file records a **paraphrased implementation/acceptance mapping**, not source workbook content or a claim that the features have been verified. All runtime acceptance remains **unverified** until each row has API/UI/provenance test evidence. Parent: #462.

| ID | Required operator outcome | Current capability owner to evaluate | Gate / likely integration |
| ---: | --- | --- | --- |
| 1 | Maintain an IT exception register with owner, rating, affected asset, remedy and closure | Risk + Matter + Source import | Native risk classifications, editable exceptions, exact source lineage |
| 2 | Remind 21 days before remediation due date and escalate past deadline | Workflow + notifications | Per-policy offset, escalation target, evidence of dispatch |
| 3 | Track multi-team subtasks; prevent parent closure until each completes | Matter + workflow | Parent/child acceptance and atomic closure guard |
| 4 | Manage annual IT risk workplans with assessors, cycles and 14-day reminders | Programs + assignments + reporting | Annual recurrence, owner visibility and due-date proof |
| 5 | Track PCI DSS/ISO 27001 assessment and recertification | Programs + published framework catalog | Verified edition, applicable controls, due reviews |
| 6 | Route, version, approve and publish policies; alert 6 months before expiry | Policies + workflow + reminders | Permissioned sign-off and configurable renewal offset |
| 7 | Let owners submit evidence for risk closure; risk team accepts or rejects | Risk/Matter + Evidence | Separate requester and reviewer and reopen path |
| 8 | Find historical recertification evidence quickly | Evidence + document browser | Permissioned search, version/source, expiry |
| 9 | Alert 5 months before PCI/ISMS certificate expiry; manage gaps | Framework Program + workflow | Dates, review plan, approved issue closure |
| 10 | Review associated assets in a single risk-assessment workspace | Asset references + Risk | One-pane scoped asset preview, missing asset state |
| 11 | Collect vendor onboarding details, assessment and required documents | Vendors + governed Forms + evidence | Required-document coverage and assessment review |
| 12 | Route BRD/process changes sequentially through participating teams | Workflows + documents | Explicit stage owner, parallel inputs, return/rework |
| 13 | Import multi-sheet IT risk data, including asset/impact/likelihood/risk matrix | Register migration + Risk | No row-number titles, exact workbook coordinates, native measures |
| 14 | Maintain a vendor risk register | Vendors + Risk / linked Matters | Service-level vendor relationships and exact risk links |
| 15 | Record losses by organization, date, event, currency and recovery | Native Loss Ledger | Exact money/currency, period and source identity |
| 16 | Run branch/unit RCSA cycles, assign actions, weekly reminders, escalate | Native RCSA + workflow | One canonical cycle, review and bounded recurrence |
| 17 | Let branches and HQ update KRIs and approved thresholds | Native Indicators + monitoring | Unit/threshold versions, organization attribution, no fake normal |
| 18 | Maintain BIA including RPO/RTO, multi-team completion and annual review | Continuity + Forms / workflow | Native register, timeline, 12-month stale review |
| 19 | Link recoveries to an originating loss and reconcile net amounts | Native Loss Ledger | Multiple recoveries, correct gross/recovery/net, exact currency |
| 20 | Link incidents to remediation, issue closure or crystallized loss | Incident + Risk/Loss | Audited transition/link, prevent silent duplicate loss |
| 21 | Require project-level privacy screening and DPO decision on full DPIA | Privacy/DPIA + project workflow | Decision with rationale and exact project link |
| 22 | Track DPIA risks, owners and closure before project go-live | Privacy + Risk/Matter | Authority gate and rejected/blocked outcome |
| 23 | Require DPO/NDPA review before vendor activation | Vendor onboarding + approval | Mandatory policy gate and conditional approval |
| 24 | Track privacy breaches including affected categories, subjects and NDPC/CBN notification | Privacy incidents + documents | Regulator-specific notice states/deadline/evidence |
| 25 | Store DPIA/vendor/privacy files and capture designated e-signatures | Evidence + signature and workflow | Signer identity, hash, refusal and audit trail |
| 26 | Generate and deliver DPO reports to authorized stakeholders | Reporting | Scope/currency/period and export authorization |
| 27 | Capture NDPA vendor obligations and essential DPA/NDA/SLA/certification documents | Vendor evidence + Forms | Requirements differ by jurisdiction/service; no auto-approval |
| 28 | Maintain ROPA records by department via review reminders | Native ROPA + Forms | Questionnaire-to-native-register mapping + refresh cycle |
| 29 | Provide auditable, permissioned ROPA repository with validation and dashboard | Native ROPA + Metrics/Reports | Stable process identity, exact drill and data quality |
| 30 | Manage customer consent across channels with integrations and audit history | Consent workflow + external adapters | Consent provenance, withdrawal and downstream propagation |
| 31 | Connect Vicarius patching, CTM360 threat intelligence, SIEM and OneIAM | Integration + source access | Approved adapters and bounded freshness/error evidence |

## Content readiness gate

A catalog starter is **not** an operational implementation until its submission maps to a native record or governed Program requirement, permitted roles can complete review/approval, and Oversight/Reports read the authoritative result. Before enabling a pack, verify:

1. Field contract and meaningful defaults (never favorable prefilled answers or blank as zero), stable categories, tenant/organization scope, reviewer, source/expiry and reporting semantics.
2. Full flow from starter selection to draft, assigned response, review, canonical ROPA/KRI/RCSA/vendor/Program source, metrics and report without creating an unrelated duplicate record.
3. Idempotent versioned pack installation, backward compatibility for existing responses, and explicit source/rights/reviewer lineage where normative framework text is involved.
4. Negative cases: stale, partial, missing, denied, duplicate, changed edition, mixed currency, cancelled and reverted workflow.

## Priority for initial curated data

- **First**: reuse active vendor due-diligence starter and verify its document/assessment path; then add only its missing approved fields as a new immutable release.
- **Next**: ROPA, KRI and RCSA starters **with mapped native-register outcomes**, not just library cards.
- **Then**: small cited, rights-cleared framework edition packs through #437 and tenant adoption; no “500+” headline until genuine source-backed publications are released.
- **Private OpsRisk**: the historical loss import retains eight canonical events, external manifests, explicit demo scope and protected backup/receipt; do not seed from the source Excel automatically.

A UI implementation remains blocked from claiming any row complete until a linked test, reviewed recording or operator acceptance proves it.
