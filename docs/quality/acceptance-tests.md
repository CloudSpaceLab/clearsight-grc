# ClearSight Acceptance and Non-Regression Tests

This document defines the minimum product, source-quality, evidence, security, AI, visual, accessibility, and operational tests required to preserve the ClearSight Programs-and-Matters model.

A feature is not accepted because CRUD, upload, API, workflow, dashboard, or AI output works. It is accepted when complete bank behavior works under realistic positive, negative, ambiguous, stale, partial, unauthorized, offline, degraded, adversarial, and historical conditions.

It conforms to:

- [`../../README.md`](../../README.md)
- [`../../AGENTS.md`](../../AGENTS.md)
- [`../product/continuous-compliance-operating-model.md`](../product/continuous-compliance-operating-model.md)
- [`../product/operating-model.md`](../product/operating-model.md)
- [`../product/experience-principles.md`](../product/experience-principles.md)
- [`regulatory-and-enforcement-acceptance-tests.md`](regulatory-and-enforcement-acceptance-tests.md)

---

# 1. Test philosophy

Tests must prove:

- Programs maintain continuing obligations;
- Matters mobilize only when change, exception, uncertainty, harm, request, or judgment requires work;
- one canonical record produces many authorized register, workplan, KRI, dashboard, filing, and assurance views;
- Authority Source precedes approved Requirement;
- applicability is explicit and reviewable;
- source authority precedes automated trust;
- existing evidence precedes human requests;
- trigger-driven refresh replaces blanket reconstruction where possible;
- evidence precedes confidence;
- contradiction remains visible;
- decision or response precedes dashboard status;
- verification precedes closure;
- human authority governs material judgment;
- and point-in-time institutional memory is reconstructable.

Fixtures MUST NOT inject desired sources, applicability, mappings, owners, conclusions, materiality, authority, or verification outcomes.

---

# 2. Required test layers

## Domain unit tests

- Program lifecycle;
- Matter type and lifecycle;
- Requirement and applicability versioning;
- Control Objective versus Control Implementation;
- Evidence Contract policy;
- Compliance State dimensions;
- trigger evaluation;
- Review Activity and KRI derivation;
- source authority and limitation;
- contradiction propagation;
- Decision, Response Package, and Verification Contract;
- temporal semantics.

## Property-based tests

Use where combinations are broad:

- temporal intervals and supersession;
- scope and applicability inheritance;
- authority matrices;
- population denominators and exclusions;
- evidence coverage and expiry;
- trigger combinations;
- matching, merge, and unmerge;
- idempotency;
- bulk selection;
- workflow transitions;
- import partial success.

## Contract tests

- APIs and events;
- Program and Matter contracts;
- Observation and Evidence Contract schemas;
- Authority Source and Source Provision;
- spreadsheet mapping;
- integration adapters;
- structured AI output;
- Response Package and export manifests;
- source-health and trigger events.

## Integration tests

Use real interactions among:

- relational database;
- object storage;
- outbox and durable workflow;
- search and graph projections;
- authorization;
- document, media, and spreadsheet processing;
- model gateway stubs or approved routes;
- connector reconciliation.

## Security tests

- tenant, entity, licence, Program, and purpose isolation;
- wrong-scope filing or response;
- bulk authorization;
- search, count, graph, cache, and embedding inference;
- protected reporting and authority cases;
- suspicious-reporting boundaries;
- malicious spreadsheets, documents, media, and prompts;
- integration credentials;
- offline capture;
- export and Response Package leakage.

## AI evaluations

- exact source and provision grounding;
- extraction correctness;
- final-versus-draft classification;
- explicit versus inferred values;
- Requirement and applicability proposal;
- mapping and reconciliation;
- contradiction handling;
- appropriate abstention;
- authority routing;
- prompt injection and leakage;
- latency and cost;
- human edit/reject rate;
- downstream outcome quality.

## Visual, accessibility, and localization tests

- Today, Programs, Work, Explore, Configure;
- light and dark parity;
- comfortable and compact density;
- desktop, tablet, and mobile where relevant;
- keyboard and screen-reader journeys;
- 200% zoom and reduced motion;
- non-color state;
- local currency, number, date, and time formats;
- long translated labels;
- low bandwidth and degraded state.

## Performance and recovery

- Requirement and Program populations;
- ROPA and control tables;
- source and observation ingestion;
- document and spreadsheet processing;
- Matter and Work queues;
- search and point-in-time reconstruction;
- filing and Response Package generation;
- workflow recovery;
- source/model outage;
- projection rebuild;
- large export.

---

# 3. Golden Journey A — Legacy compliance register becomes a governed Program

## Purpose

Prove that ClearSight does not merely recreate a spreadsheet as a web table.

## Setup

- legacy workbook contains regulator, requirement summary, frequency, evidence, rating, and status;
- some rows have no official source;
- some rows duplicate the same underlying Requirement;
- some dates and statuses are incomplete;
- one row refers to a superseded circular.

## Required path

1. Workbook is uploaded and mapped with file, sheet, row, and mapping provenance.
2. Rows are classified as candidate Requirements, Review Activities, standards, filings, or unresolved working records.
3. Official Authority Sources are linked where available.
4. Rows without verified sources remain unapproved candidates.
5. Duplicate and superseded rows are proposed for reconciliation.
6. Applicable Requirements are approved by authorized reviewers.
7. Controls, owners, Evidence Contracts, and calendar items are linked.
8. The original familiar register view is generated from canonical records.
9. Changing a Requirement updates the register view without creating a duplicate row.

## Assertions

- [ ] Import success does not imply approved Requirement.
- [ ] Exact source provisions are available for approved Requirements.
- [ ] Duplicate and superseded rows retain history.
- [ ] One Requirement can apply to multiple scopes without copying its source interpretation.
- [ ] Register export remains available with a manifest.
- [ ] No manual dashboard copy becomes a separate truth system.

---

# 4. Golden Journey B — NDPA ROPA remains continuously current

## Purpose

Prove trigger-driven continuing compliance with minimum human effort.

## Setup

- Program contains approved privacy Requirements;
- application, vendor, project, and organization sources exist;
- 250 processing activities are expected;
- 190 have complete current source-backed context;
- 40 need business-purpose or lawful-basis confirmation;
- 20 changed because of a new vendor, data category, jurisdiction, or retention rule.

## Required path

1. ROPA population is assembled from canonical applications, processes, projects, vendors, and previous approved activities.
2. Known system, vendor, owner, location, and data-flow facts are prefilled.
3. Only 60 affected activities are routed for focused confirmation or review.
4. Respondents can correct, redirect, mark not applicable, or raise sensitivity concerns.
5. Contradictory purpose, retention, or transfer facts remain visible.
6. Approved updates change affected Claims and Compliance State.
7. Unaffected activities remain current without a blanket questionnaire.
8. Program history reconstructs prior ROPA versions.

## Assertions

- [ ] No request is sent for 190 fully current activities.
- [ ] Population denominator and exclusions are visible.
- [ ] Business assertion is distinct from authoritative system fact.
- [ ] One changed vendor can identify all affected activities.
- [ ] Compliance state exposes evidence and source-quality dimensions.
- [ ] ROPA export is point-in-time and source-linked.

---

# 5. Golden Journey C — DPIA triggered by project and vendor change

## Purpose

Prove that privacy governance integrates into institutional change rather than relying on a separate annual tracker.

## Setup

- new customer-facing project uses sensitive data, a new processor, and automated decisioning;
- project, architecture, vendor, and data context are partially available;
- go-live requires DPO decision.

## Required path

1. Project/change event triggers privacy screening.
2. Known project, system, vendor, data, jurisdiction, and model facts are prefilled.
3. Only unresolved risk and purpose questions are asked.
4. DPO determines that a full DPIA is required and records rationale.
5. DPIA creates risks, controls, Actions, owners, and Evidence Contracts.
6. Go-live gate remains blocked until required Decisions and remediation are complete.
7. Post-deployment monitoring and verification are scheduled.
8. Program Requirements and ROPA update from the approved change.

## Assertions

- [ ] Project owner cannot self-approve privacy applicability.
- [ ] DPIA is linked to the same project, vendor, systems, and processing activities.
- [ ] Action completion is distinct from DPO approval and post-deployment verification.
- [ ] Rejected or abandoned projects do not leave active false processing records.

---

# 6. Golden Journey D — NDPA breach Matter and regulatory timing

## Purpose

Prove event-driven Matter handling, authority, deadline logic, evidence, notification, and verification.

## Setup

- potential personal-data breach is detected;
- awareness time is initially uncertain;
- affected systems and data-subject population evolve during investigation;
- notification may be required.

## Required path

1. Detection creates a protected Breach Matter.
2. Detection time, awareness time, source, affected data, systems, and population remain distinct and versioned.
3. Investigation gathers only relevant evidence.
4. Reportability Decision is routed to authorized DPO/legal/compliance roles.
5. Deadline calculation uses approved awareness time and policy.
6. Required notifications and communications form governed Response Packages.
7. Transmission and acknowledgement are recorded.
8. Remediation is implemented and verified.
9. ROPA, controls, risk, and Program Compliance State update.

## Assertions

- [ ] AI cannot make final reportability Decision.
- [ ] Uncertain awareness time is not silently fixed.
- [ ] Notification task completion does not close the breach.
- [ ] Protected data does not leak into ordinary Program summaries.
- [ ] Later population changes preserve previous decisions and rationale.

---

# 7. Golden Journey E — Annual filing assembled continuously

## Purpose

Prove that filing readiness derives from year-round evidence rather than last-minute reconstruction.

## Setup

- Program has filing Requirements, Evidence Contracts, ROPA, DPIAs, vendor evidence, policies, training, breaches, exceptions, and prior assurance;
- several evidence items are stale or incomplete;
- one exception is approved and one is overdue.

## Required path

1. Filing readiness displays separate dimensions and affected Requirements.
2. Existing current evidence is reused.
3. Focused requests are sent only for unresolved items.
4. Included, excluded, excepted, stale, and unavailable records are explicit.
5. Review and sign-off use authority and segregation of duties.
6. Point-in-time package freezes source and evidence versions.
7. Submission and acknowledgement are recorded.
8. Filing completion does not overwrite underlying gaps or exceptions.

## Assertions

- [ ] No unexplained single readiness percentage is authoritative.
- [ ] Package statements trace to source, Requirement, control, evidence, and approval.
- [ ] An approved exception remains visible in the filed state.
- [ ] Filing can be reconstructed years later.

---

# 8. Golden Journey F — New CBN publication updates Programs and creates Matters

## Purpose

Prove source ingestion, interpretation, applicability, control reconciliation, implementation, and continuing evidence.

## Required path

1. Official publication is captured with authenticity, reference, dates, original artifact, hash, and exact provisions.
2. Document is classified as final, amendment, guidance, or draft.
3. Candidate directive atoms and Requirements are extracted with source anchors.
4. Compliance/legal reviewers approve interpretation and applicability.
5. Existing Programs, policies, controls, systems, vendors, and evidence are reconciled.
6. Fully covered Requirements remain current with rationale.
7. Partial and uncovered Requirements create implementation Matters.
8. Actions, owners, tests, Evidence Contracts, and deadlines are generated.
9. Verification updates the continuing Program state.
10. Later amendment supersedes affected Requirements without overwriting history.

## Assertions

- [ ] Exposure draft cannot create effective mandatory controls.
- [ ] AI cannot publish final legal interpretation.
- [ ] Every Requirement has exact source lineage.
- [ ] Duplicate controls are not created where existing implementation is sufficient.
- [ ] Program and Matter views remain connected.

Detailed cases are in the specialized regulatory acceptance suite.

---

# 9. Golden Journey G — Protected authority request

## Purpose

Prove correct classification, legal review, subject resolution, casework, response, minimization, and authority.

## Setup

- authority communication concerns multiple named subjects and account records;
- legal instrument state is ambiguous for one requested action;
- one subject match is exact, one provisional, and one unresolved;
- request may require records, KYC review, address confirmation, preservation, or internal AML assessment.

## Required path

1. Source is captured and authenticity verified.
2. Legal basis, confidentiality, requested scope, period, deadline, and instrument status are reviewed.
3. Directives are decomposed with exact source anchors.
4. Subject/account resolution preserves exact, provisional, and unresolved states.
5. Unauthorized action is blocked where legal authority is insufficient.
6. Approved directives create records, KYC, address, preservation, legal, AML, fraud, branch, or technology tasks as appropriate.
7. Internal suspicious-report assessment remains a separate restricted Decision.
8. Response Package reconciles every directive to included evidence, exclusion, inability, or approved redaction.
9. Signatory, transmission, acknowledgement, retention, and legal hold are recorded.
10. Only minimized aggregate KRI or systemic signals leave the protected case boundary.

## Assertions

- [ ] Authority request does not imply guilt.
- [ ] Ambiguous subject is not merged automatically.
- [ ] Request does not automatically authorize account restriction, disclosure, or suspicious reporting.
- [ ] Ordinary search, Today, analytics, and exports reveal no protected subjects or counts.
- [ ] Case closure requires directive reconciliation and acknowledgement/accepted closure policy.

---

# 10. Golden Journey H — Legacy finding and exception reach verified closure

## Purpose

Prove migration from comment-driven spreadsheet work to structured Matter handling.

## Setup

- imported finding contains affected application, risk, recommendation, owner, date, status, and long comments;
- comments indicate unclear ownership and rerouting;
- evidence is later uploaded;
- task completion does not prove sustained outcome.

## Required path

1. Finding row imports with provenance.
2. Application, control, owner, action, comments, and framework references map to canonical objects.
3. Ownership conflict becomes structured redirect and assignment history.
4. Comments remain communications, not hidden state transitions.
5. Evidence submission is evaluated against a Claim and Evidence Contract.
6. Action reaches implemented state.
7. Verification observes the required population and period.
8. Failed verification reopens or continues the Matter.
9. Successful evidence reaches accepted closure.
10. Original register view and committee status derive from the Matter.

## Assertions

- [ ] Row status is not authoritative after migration.
- [ ] Evidence upload is not closure.
- [ ] Assignment changes are explicit and auditable.
- [ ] Framework reference is distinct from source Requirement and control implementation.

---

# 11. Golden Journey I — ATM/POS population, focused capture, and reconciliation

## Purpose

Preserve the bank-operational wedge beneath the compliance model.

## Setup

- mixed spreadsheet, telemetry, vendor, merchant, branch, and human sources;
- duplicate, stale, missing, and contradictory identifiers;
- large ATM or POS population.

## Required path

1. Source Profiles define authority and freshness.
2. Imports preserve row provenance and partial success.
3. Population worklist exposes denominator and states.
4. Only unresolved or sampled objects receive focused requests.
5. Media extraction is bounded and user-confirmed.
6. Contradictions remain explicit.
7. Risk Situation Matter links to affected Program controls and Requirements.
8. Action and Verification update the Program and operational state.

## Assertions

- [ ] Tables, not card grids, handle populations.
- [ ] Bulk action enforces authorization per object.
- [ ] Photo interpretation does not claim invisible control effectiveness.
- [ ] Source degradation is separate from confirmed control failure.

---

# 12. Golden Journey J — KRI derived from underlying Matters

## Purpose

Prove that KRIs are not manually disconnected numbers.

## Setup

- indicator counts authority requests, overdue exceptions, branch incidents, channel failures, or vendor gaps;
- underlying Matters have different states and protected classifications.

## Required path

1. KRI definition identifies source population, measure, period, threshold, exclusions, and owner.
2. Value derives from authorized canonical records.
3. Drill-down shows permitted underlying population.
4. Protected or unauthorized cases contribute only according to approved minimization policy.
5. Threshold breach creates or updates a Matter.
6. Corrected or superseded records update the value with history.

## Assertions

- [ ] KRI exposes denominator, period, source, and exclusions.
- [ ] User cannot infer protected subject count where policy forbids it.
- [ ] Manual override requires authority, rationale, evidence, and expiry.
- [ ] Threshold breach is not automatically treated as proven loss or incident.

---

# 13. Golden Journey K — Source degradation and evidence uncertainty

## Purpose

Prove that source health affects evidence without falsely declaring non-compliance or control failure.

## Setup

- critical source exceeds freshness target;
- no adverse event is observed;
- human attestation exists;
- Program conclusion depends on current population.

## Required path

1. Source Profile changes to stale or unavailable.
2. Dependent Claims, Requirements, Compliance State, filings, and Matters are identified.
3. Evidence and data-quality uncertainty remain separate from exposure severity.
4. Human assertion is not treated as replacement for authoritative population evidence.
5. Fallback or focused request is selected.
6. Restored source reconciles with interim evidence.

## Assertions

- [ ] Stale, no data, unknown, and failed-control states are distinct.
- [ ] Recovery does not silently resolve contradiction.
- [ ] Source owner, limitation, age, and downstream effect are visible.

---

# 14. Golden Journey L — Malicious content and governed AI

## Purpose

Prove content security across official documents, legacy spreadsheets, evidence, and media.

## Setup

- spreadsheet contains formula injection, hidden sheets, and unexpected macros;
- document contains instructions to reveal secrets and execute tools;
- media includes unnecessary personal or location metadata.

## Required path

1. All content is treated as untrusted.
2. Formula, macro, hidden-content, malware, and active-content policies run.
3. Embedded instructions do not alter system or tool policy.
4. No secret, cross-tenant retrieval, or unauthorized action occurs.
5. Relevant source facts may still be extracted.
6. Metadata is minimized according to policy.
7. Security event and evidence lineage remain safe.

## Assertions

- [ ] Import preview never executes formulas.
- [ ] Tool permissions remain outside prompts.
- [ ] Free-form model output cannot mutate authoritative state.
- [ ] Security logs contain no restricted evidence.

---

# 15. Golden Journey M — Scope switching, bulk action, and partial authorization

## Purpose

Prove safe work across entities, Programs, populations, and cases.

## Required path

1. Context header shows institution, entity, licence, Program or Matter, population, and period.
2. Selection summary shows matching, writable, excluded, and unauthorized counts according to policy.
3. Exact criteria and side effects are shown.
4. Server evaluates each object.
5. Partial result shows success, exclusion, failure, and retry.
6. Audit manifest reconstructs the operation.
7. Scope switch clears or confirms incompatible drafts and selections.

## Assertions

- [ ] Client selection cannot bypass authorization.
- [ ] Unauthorized object details are not inferable.
- [ ] Partial success is not displayed as universal success.

---

# 16. Golden Journey N — AI and integration degraded mode

## Purpose

Prove that Programs, Matters, evidence, decisions, and responses remain usable during external failure.

## Required path

1. Deterministic Program and Matter context renders.
2. Last-known source data shows exact age.
3. AI state shows unavailable, not pending indefinitely.
4. Manual evidence, applicability, Decision, and response workflows remain usable.
5. Missing information and uncertainty are captured.
6. Failed jobs resume without duplicate effects.

## Assertions

- [ ] No stale AI output is presented as new.
- [ ] No stale source is presented as current.
- [ ] Recovery preserves idempotency and history.

---

# 17. Golden Journey O — Point-in-time Program and Matter reconstruction

## Purpose

Prove institutional memory across amended sources, changed applicability, controls, evidence, filings, responses, and outcomes.

## Required path

Authorized user selects a historical date and retrieves:

- Authority Source and provisions known then;
- Requirements and applicability effective and recorded then;
- Program scope, controls, owners, and Evidence Contracts;
- source health and observations available then;
- Compliance State and assurance conclusions;
- Matters, Decisions, Actions, filings, responses, and verification known then;
- later corrections clearly separated.

## Assertions

- [ ] Future knowledge is not included in the known-then view.
- [ ] Current viewer authorization is enforced.
- [ ] Valid time and record time remain distinct.
- [ ] Export identifies reconstruction time and included versions.

---

# 18. Release gates

## Pull request gate

- relevant unit, contract, authorization, accessibility, and visual tests;
- documentation and ADR updates;
- no sensitive logging;
- migration and rollback considered;
- screenshots for UI changes.

## Feature gate

- complete Program or Matter behavior;
- source and evidence lineage;
- failure and degraded modes;
- telemetry and runbook;
- role and authority tests.

## AI capability gate

- evaluation thresholds;
- exact grounding;
- structured validation;
- appropriate abstention;
- no critical authorization or leakage failure;
- monitoring, kill switch, and rollback.

## Pilot gate

- NDPA, regulatory change, authority request, legacy Matter, source degradation, and point-in-time journeys pass;
- backup and recovery pass;
- performance meets pilot targets;
- users demonstrate lower effort and stronger evidence.

## General availability gate

- independent security review;
- approved SLOs and operational ownership;
- tested deployment and rollback;
- legal/privacy controls validated;
- no critical product-invariant defect.

---

# 19. Final acceptance standard

ClearSight passes its defining acceptance test only when it can:

1. preserve an authoritative source or institutional trigger;
2. maintain continuing Requirements, controls, evidence, and assurance in a Program;
3. identify what changed or became uncertain;
4. create the correct bounded Matter where action is required;
5. find existing evidence before contacting a person;
6. ask only the smallest unresolved question;
7. preserve source limitations and contradiction;
8. route the correct human Decision or response;
9. execute safely through people or systems;
10. verify the defined outcome or reconcile the response;
11. update Program state and derived views;
12. reconstruct the entire path later.

Anything less is a digitized register or partial workflow, not the finished ClearSight operating system.