# Monitoring Setup and Risk Scoring Design

**Date:** 2026-08-17

**Status:** Approved
**Primary users:** CRO, CCO, CISO, Program Owner, GRC Administrator, Control Owner, Evidence Reviewer

## 1. Outcome

ClearSight must let an authorized bank user create and operate a governed monitoring Program without using an API or editing JSON. The user must be able to define what needs to be controlled, connect a reusable data source or build a reusable data-collection form, evaluate source values or form responses against deterministic rules, and route adverse or incomplete results for review.

The first acceptance scenario is a Mobile Banking Program with:

1. a connected endpoint that demonstrates whether live face verification is enabled and operating; and
2. a five-question password-reset form whose responses produce a reproducible risk result.

The feature is generic. Mobile banking is an acceptance fixture, not a hard-coded product type.

## 2. Product principles

- Programs remain the long-lived record for anything that must be controlled or monitored.
- Requirements describe what must be true. Controls describe how the bank achieves it. Monitoring Checks describe how ClearSight observes it.
- A source reading or form response is an observation. It is not automatically an approved evidence assessment or compliance conclusion.
- Risk results are deterministic, versioned, reconstructable and explainable from stored inputs and rules.
- Missing, stale, partial, ambiguous or invalid input never receives a favourable score.
- Material actions remain authority-checked. Configuration follows maker-checker approval.
- The complete routine workflow is available in the browser. Internal IDs, JSON definitions and adapter terminology are not required from business users.
- The experience remains usable without AI. AI may suggest labels or rules later, but is not part of scoring or execution.

## 3. Information architecture

### 3.1 Programs

The Programs workspace receives a primary **Create program** action. Program detail exposes five business sections:

- Overview
- Requirements and controls
- Monitoring
- Latest results
- Issues and actions

Creating a Program uses a full-page, resumable workspace rather than a modal. The steps are:

1. **Program** — name, purpose, owner, legal entity, jurisdiction, effective date and scope.
2. **Requirements** — one or more statements describing what must be true.
3. **Controls** — the operating safeguards and their owners.
4. **Monitoring** — connected-data checks, form checks, or both.
5. **Review** — responsibilities, evidence expectations and activation.

The scope builder provides familiar dimensions: channel, product, process, system, vendor, customer population, jurisdiction and legal entity. It serializes the selected values to the existing Program scope JSON without exposing JSON to the user.

### 3.2 Data sources

Configure gains a **Data sources** tab. Program setup can open the same source workflow contextually and return the selected approved data fields to the Monitoring step.

The source workflow is:

1. choose API, PostgreSQL, governed file or webhook;
2. name the source and assign an owner;
3. enter non-secret connection settings and select a deployment-managed credential when required;
4. test the connection;
5. select the records and stable identifier;
6. inspect and select fields;
7. preview bounded values;
8. submit for approval;
9. approve and activate through an independent authorized user.

The UI uses **Connection**, **Data set** and **Available fields**. Adapter, View and Binding remain specialist details and audit terms.

Raw credentials are not returned to the browser after submission and are never stored in source definitions. This tranche supports public endpoints and deployment-managed credential references. A browser-managed credential vault is a separate security product and is not simulated with plaintext database storage.

### 3.3 Forms

Monitoring setup includes a reusable form builder. Users can:

- name and describe a form;
- add, remove and reorder questions;
- use Yes/No, single choice, number, date, short text, long text and file fields;
- mark fields required;
- assign response scoring to bounded choice or numeric fields;
- mark a response as a critical failure;
- preview the respondent experience;
- save a draft, submit it for approval and activate it;
- start a collection by choosing a respondent, review owner, reporting period and deadline.

Starting a collection creates an immutable Evidence Request instance from the active form version. Reusing the form creates a new request; a prior request is never reopened or overwritten. Automatic weekly request generation is not part of this scope.

## 4. Domain model

### 4.1 FormTemplate

A `FormTemplate` is a tenant-scoped, versioned configuration record:

```text
id, tenant_id, code, name, purpose
fields[]
status: DRAFT | PENDING_APPROVAL | ACTIVE | REJECTED | PAUSED | RETIRED
is_current, effective_from, effective_until
created_by, submitted_by, approved_by, rejected_by
version, created_at, updated_at
```

Each field retains the existing Evidence Request field contract plus an optional scoring definition:

```text
field_id
weight: integer 1..100
answer_scores: map of allowed answer -> integer 0..100
critical_answers: set of allowed answers
```

Only `single_select` and `number` fields may contribute directly to a score in the first release. Text, date and file responses remain supporting evidence.

### 4.2 MonitoringCheck

A `MonitoringCheck` is a versioned Program configuration record:

```text
id, tenant_id, program_id
requirement_id, control_implementation_id, evidence_contract_id
code, name, claim
input_kind: FORM | SOURCE
form_template_id + form_template_version, or binding_id + binding_version
rules[]
thresholds
freshness_minutes, minimum_coverage
owner_principal_id, reviewer_principal_id
failure_action
status and lifecycle actors/dates
version, created_at, updated_at
```

A source rule compares an exact selected field using a bounded operator:

```text
EQUALS, NOT_EQUALS, GREATER_THAN, GREATER_OR_EQUAL,
LESS_THAN, LESS_OR_EQUAL, PRESENT, MAX_AGE_MINUTES
```

A form rule uses the scoring definition stored with the exact active Form Template version.

### 4.3 MonitoringResult

Every evaluation creates an immutable `MonitoringResult`:

```text
id, tenant_id, program_id, monitoring_check_id, monitoring_check_version
input_kind, input_reference_id, input_reference_version
score: 0..100 or null
band: LOW | MODERATE | HIGH | CRITICAL | NOT_ASSESSED
coverage: 0..1
critical_failures[]
rule_results[]
source_receipt or submission provenance
evaluated_at, evaluator_version, created_at
```

For forms, `input_reference_id` is the Evidence Request submission. For connected data, it is the canonical source-operation receipt identity.

Results are append-only. Re-evaluation creates another result and retains the rule and input versions used previously.

## 5. Scoring semantics

### 5.1 Form score

For answered scored fields:

```text
weighted_score = sum(answer_score * weight) / sum(weight)
coverage = answered_required_scored_fields / required_scored_fields
```

Rules:

- any configured critical answer sets the band to `CRITICAL`;
- missing required input sets the band to `NOT_ASSESSED` and score to null;
- invalid answers are rejected at submission and never coerced;
- unscored supporting fields do not affect the denominator;
- thresholds are inclusive and cannot overlap;
- the stored result includes every component used in the calculation.

Default thresholds offered by the UI are:

- Low: 0–24
- Moderate: 25–49
- High: 50–74
- Critical: 75–100

Users may change thresholds before activation. Changes create a new version and do not reinterpret historical results.

### 5.2 Source result

Each active rule returns pass, fail or indeterminate. All required rules passing yields score 0 and Low. Failed rules contribute configured risk points. A critical rule failure produces Critical. Stale, partial, unavailable, schema-drifted or ambiguous source resolution produces Not assessed and an explicit reason.

### 5.3 Governance effect

A Monitoring Result may:

- create a Program trigger;
- update the latest-results projection;
- create a reviewer task;
- recommend a Control Gap or Evidence Contradiction Matter.

It does not directly create an approved Evidence Assessment. Automatic Matter creation requires an active Automation Policy whose eligibility, blast radius, rollback and verification contract permit it. Without that policy, the reviewer receives the recommended action.

## 6. APIs

### 6.1 Program setup

The browser uses the existing authority-guarded Program commands:

```text
POST /api/v1/programs
POST /api/v1/programs/{id}/requirements
POST /api/v1/programs/{id}/applicability
POST /api/v1/programs/{id}/control-objectives
POST /api/v1/programs/{id}/control-implementations
POST /api/v1/programs/{id}/control-links
POST /api/v1/programs/{id}/evidence-contracts
POST /api/v1/programs/{id}/transition
```

The UI saves after each successful material command and resumes from the stored Program aggregate. A later-step failure does not erase earlier approved work.

### 6.2 Form templates

```text
GET  /api/v1/config/form-templates
POST /api/v1/config/form-templates
GET  /api/v1/config/form-templates/{id}
POST /api/v1/config/form-templates/{id}/submit
POST /api/v1/config/form-templates/{id}/approve
POST /api/v1/config/form-templates/{id}/reject
POST /api/v1/config/form-templates/{id}/pause
POST /api/v1/config/form-templates/{id}/retire
POST /api/v1/config/form-templates/{id}/collections
```

Collection creation validates that the requested version is active and then creates an Evidence Request with the exact field schema and form version reference.

### 6.3 Monitoring checks and results

```text
GET  /api/v1/programs/{id}/monitoring-checks
POST /api/v1/programs/{id}/monitoring-checks
GET  /api/v1/monitoring-checks/{id}
POST /api/v1/monitoring-checks/{id}/submit
POST /api/v1/monitoring-checks/{id}/approve
POST /api/v1/monitoring-checks/{id}/reject
POST /api/v1/monitoring-checks/{id}/evaluate
GET  /api/v1/monitoring-checks/{id}/results
```

Form submission invokes evaluation after the submission transaction commits through the existing outbox/worker boundary. A failed evaluation job does not turn a committed response into a submission failure and is retried idempotently.

### 6.4 Source lifecycle

The existing source configuration APIs gain lifecycle commands for Connection, View and Binding revisions:

```text
POST .../{id}/submit
POST .../{id}/approve
POST .../{id}/reject
POST .../{id}/pause
POST .../{id}/retire
```

Submitter and approver must differ. Approval revalidates the exact parent revisions, inspected schema, capabilities, source ownership, authority and current-version conflict before atomically activating the revision.

## 7. Authority and permissions

- Creating and editing Program content uses current Program Owner routes.
- Applicability and Program activation use Program Authorizer routes.
- Form, Monitoring Check and source configuration require `CONFIG_WRITE` and a verified tenant identity.
- Activation requires the configured Authorizer or Reviewer responsibility and enforces maker-checker separation.
- Starting a collection requires authority over the linked Program or control and validates the recipient belongs to the tenant and can read the subject.
- Recording an Evidence Assessment remains a Reviewer command.
- Every lifecycle command records verified identity rather than accepting an actor from browser JSON.

## 8. User experience and copy

The UI uses bank operating language:

- Program
- Requirement
- Control
- Monitoring check
- Data source
- Form
- Collection
- Result
- Reviewer

It does not expose `Binding`, `View`, `JSON`, `adapter`, `projection` or internal status codes in routine screens.

The setup workspace keeps one primary action per step, supports Back without losing saved data, shows field-level validation, and provides a visible exit to Programs. It is not a blocking modal. Long technical details remain under an optional **Connection details** disclosure.

Every result shows:

- score and risk band;
- observation or response time;
- coverage and missing input;
- source or form version;
- failed rules;
- owner, reviewer and next action.

Colour is not the only state indicator. All controls have keyboard focus, programmatic labels and 44px minimum targets. The layout is verified at 375px, 768px and 1440px widths.

## 9. Sample organization naming

All customer-visible fixture names change to:

- tenant: **Clear Bank**
- legal entity: **Clear Bank Nigeria**

Durable IDs and slugs remain unchanged. Deployed fixture rows are renamed only when they still contain the prior ClearSight-owned sample names; customer-modified names are not overwritten. Sample disclosure uses **Sample data** or **Reference data** and does not make the institution name sound fictitious.

## 10. Failure and degraded states

- A failed Program setup command identifies the affected step and preserves prior saved work.
- A failed connection test stores no successful observation and cannot proceed to approval.
- Schema drift blocks source evaluation and identifies fields that changed.
- Source unavailability and stale data produce Not assessed, never Low risk.
- A form cannot activate with duplicate field IDs, invalid options, overlapping thresholds, missing scoring choices or no reviewer.
- A collection cannot start from a draft, paused or retired form version.
- A response submission remains accepted if downstream evaluation is temporarily unavailable; the result displays Pending evaluation until the worker succeeds.
- Unauthorized and cross-tenant reads and writes fail closed in the API, not only in the UI.

## 11. Delivery slices

### Slice A — Program setup and sample naming

Deliver the non-modal Program setup workflow over existing commands, scope builder, saved-step recovery, Program Monitoring section shell, and Clear Bank fixture rename.

### Slice B — Reusable forms and deterministic scoring

Deliver Form Template lifecycle, builder, collection creation, immutable template reference on Evidence Requests, submission-triggered evaluation, results and reviewer routing.

### Slice C — Governed source setup

Deliver source configuration UI, connection testing, schema/field selection, lifecycle commands, maker-checker activation and connected-data rule evaluation.

### Slice D — Integrated monitoring workflow

Deliver contextual source/form attachment from Program setup, Program latest-results projection, trigger/task generation, policy-governed Matter recommendation/creation, and the Mobile Banking acceptance fixture.

Each slice is deployable and testable. A partially delivered slice must not display enabled controls for unavailable actions.

## 12. Acceptance tests

### Mobile Banking Program

An authorized CRO can complete these actions without API knowledge:

1. create **Mobile Banking Channel Risk and Compliance**;
2. scope it to Clear Bank Nigeria and the Mobile Banking channel;
3. add a live-face-verification requirement and linked control;
4. add a secure-password-reset requirement and linked control;
5. connect and test an HTTPS face-verification status endpoint;
6. select `sdk_enabled`, `sdk_version`, `live_challenge_passed` and `observed_at`;
7. define the required values and maximum observation age;
8. build a five-question Yes/No password-reset form;
9. assign weights and critical No responses;
10. activate the configuration through independent approval;
11. send a form collection to an eligible respondent;
12. submit all five answers;
13. see the exact calculated score, band, coverage and failed questions;
14. see unavailable or incomplete source input as Not assessed;
15. route an adverse result to the correct reviewer;
16. see the Program result without an unsupported compliance claim.

### Required automated coverage

- tenant isolation and verified-actor binding;
- maker-checker separation and revoked authority;
- optimistic concurrency and idempotent evaluation;
- form schema and scoring validation;
- score boundaries, critical overrides and incomplete input;
- source freshness, partial results, drift and unavailability;
- exact version and provenance reconstruction;
- Program setup recovery after each command failure;
- keyboard, screen-reader and responsive UI behavior;
- customer-facing copy regression;
- deployed fixture rename without overwriting customer-modified names.

## 13. Out of scope

- automatic weekly form generation;
- an in-browser general-purpose credential vault;
- arbitrary user-authored code or expressions;
- AI-generated compliance conclusions;
- automatic approval of Evidence Assessments;
- arbitrary URL, header or query execution outside governed source definitions;
- replacing Programs with a generic custom-object platform.
