# AGENTS.md

This file defines mandatory rules for every human contributor, coding agent, design agent, reviewer, and automated change applied to ClearSight.

It exists to prevent the product from regressing into a conventional GRC portal, a generic AI chat interface, a dense dashboard, a graph demonstration, or a collection of digital registers that reproduce the bank’s existing spreadsheet fragmentation.

The words **MUST**, **MUST NOT**, **SHOULD**, and **SHOULD NOT** are normative.

---

# 1. Mission

ClearSight is a direct, AI-native continuous compliance and risk operating system built first for banks.

Every implementation decision must advance this outcome:

> **Help each stakeholder understand what the institution must do, how it is currently being satisfied, what evidence proves it, what changed or became uncertain, who must act, and whether the required outcome was achieved.**

ClearSight remains a comprehensive GRC platform, but users MUST NOT be required to operate its internal architecture or maintain duplicate truth across separate registers.

The product is optimized for:

- continuously maintained compliance Programs;
- direct, bounded Matters when something changes or requires action;
- familiar banking and regulatory language;
- less human evidence and reporting effort;
- explicit source authority and data quality;
- evidence-backed compliance and risk conclusions;
- accountable decisions and responses;
- verified outcomes;
- and durable institutional memory.

It is not optimized for the number of forms, modules, dashboards, records, alerts, graph nodes, AI messages, or configuration options it can display.

---

# 2. Required reading and precedence

Before changing product behavior, domain semantics, architecture, or interface structure, read:

1. [`README.md`](README.md)
2. [`docs/product/continuous-compliance-operating-model.md`](docs/product/continuous-compliance-operating-model.md)
3. [`docs/product/operating-model.md`](docs/product/operating-model.md)
4. [`docs/product/regulatory-and-enforcement-intelligence.md`](docs/product/regulatory-and-enforcement-intelligence.md) when external-authority work is affected
5. [`docs/product/experience-principles.md`](docs/product/experience-principles.md)
6. [`docs/product/differentiation.md`](docs/product/differentiation.md)
7. [`docs/architecture/product-semantics-mapping.md`](docs/architecture/product-semantics-mapping.md)
8. relevant deeper architecture documents
9. [`docs/implementation-plan.md`](docs/implementation-plan.md)
10. relevant acceptance tests

When documents conflict, use this order:

1. safety, confidentiality, legal boundaries, and tenant isolation;
2. README product intent;
3. continuous-compliance product semantics;
4. universal operating primitives;
5. specialized product specifications;
6. experience principles;
7. this normative file;
8. architecture documents;
9. implementation sequencing;
10. acceptance detail.

Internal architecture MUST NOT override the simpler product operating model without an explicit product decision and synchronized documentation change.

---

# 3. Priority order

When requirements conflict:

1. Safety, confidentiality, legal boundaries, and tenant isolation
2. Evidence integrity and decision auditability
3. Product semantics and invariants
4. User authority and segregation of duties
5. Functional correctness
6. Accessibility and usability
7. Reliability and recoverability
8. Performance
9. Visual polish
10. Implementation convenience

Visual polish may never conceal uncertainty, weaken accessibility, or replace missing domain correctness.

---

# 4. Canonical user-facing aggregates

## 4.1 Program

A **Program** is a stable, continuing body of requirements, controls, evidence obligations, scheduled reviews, exceptions, filings, and assurance.

Examples include NDPA, AML/CFT, CBN cybersecurity, PCI DSS, operational resilience, third-party assurance, RCSA, policy lifecycle, and regulatory returns.

A Program MUST be able to answer:

- what requirements apply;
- where they apply;
- how they are intended to be satisfied;
- who owns and reviews them;
- what evidence is required;
- whether evidence is current and sufficient;
- which gaps, exceptions, or filings require attention;
- and what changed since the last approved state.

A Program MUST NOT be implemented as one giant checklist, one unexplained compliance percentage, or a permanent campaign of broad questionnaires.

## 4.2 Matter

A **Matter** is a bounded occurrence requiring assessment, evidence, decision, action, response, or verification.

Matter types include:

- regulatory change;
- supervisory finding;
- authority or enforcement request;
- risk situation;
- control gap;
- audit finding;
- exception or waiver;
- incident or loss;
- data breach;
- vendor deficiency;
- customer or conduct concern;
- overdue obligation;
- failed verification;
- evidence contradiction;
- and KRI threshold breach.

A Matter MUST connect to affected Programs, requirements, controls, services, customers, accounts, assets, vendors, evidence, decisions, and actions without duplicating them.

Risk Situation remains a valid Matter subtype. It is not the only product aggregate.

---

# 5. Canonical shared primitives

Programs and Matters are assembled from shared primitives. These MUST remain distinct.

## 5.1 Scope

The bounded institution, legal entity, jurisdiction, licence, channel, service, branch, merchant group, customer population, asset population, vendor relationship, system, project, or process being governed.

Active scope and effective period MUST be clear before material action, approval, export, bulk change, filing, response, or evidence submission.

## 5.2 Authority Source and Requirement

An Authority Source is an original law, circular, regulation, guideline, standard, licence condition, contract, court or enforcement instrument, or approved institutional interpretation.

A Requirement is a versioned statement of what an actor must, must not, may, or is expected to do for a defined scope and period.

A manually authored register row MUST NOT silently become an authoritative Requirement without source reconciliation and required approval.

## 5.3 Applicability

Applicability is a governed conclusion about where and when a Requirement applies.

AI MAY propose applicability. Material applicability MUST be approved by the appropriate compliance, legal, regulatory-affairs, or delegated authority.

## 5.4 Control Objective and Control Implementation

A Control Objective defines the required outcome.

A Control Implementation is the actual policy, process, system rule, approval, monitoring mechanism, review, or operating practice used in a specific scope.

Do not collapse a global objective and multiple legal-entity, channel, system, branch, or vendor implementations into one record.

## 5.5 Exposure Pattern and Risk Situation

An Exposure Pattern is a reusable description of how harm or non-conformance may occur.

A Risk Situation is a current Matter applying one or more patterns to a bounded context.

## 5.6 Claim and Evidence Contract

A Claim is a precise statement that can be supported, contradicted, qualified, or remain unresolved.

An Evidence Contract defines required facts, population, period, acceptable sources, source authority, freshness, coverage, independence, contradiction policy, reviewer authority, refresh triggers, and failure behavior.

The terms Evidence Contract and Evidence Recipe refer to the same underlying versioned policy. Use **Evidence Contract** when describing continuing Program proof and **Evidence Recipe** where a task-specific capture pattern is clearer.

## 5.7 Observation and Evidence

An Observation is a normalized, source-preserving record of something observed, submitted, imported, measured, extracted, or asserted.

Evidence is an Observation or source artifact evaluated for a defined Claim and purpose.

Forms, dropdowns, photos, spreadsheets, documents, APIs, telemetry, messages, attestations, customer reports, and protected reports MUST converge on the same governed observation contract.

## 5.8 Conclusion and Compliance State

A Conclusion is a versioned determination of what current evidence supports.

Compliance State is a governed projection across separate dimensions such as interpretation, applicability, control design, implementation, evidence sufficiency, operating effectiveness, exception, filing, assurance, and source quality.

Do not reduce these dimensions to one unexplained percentage or red/amber/green value.

## 5.9 Decision, Action, Verification, and Response Package

A Decision is an authorized selection among options with evidence, uncertainty, rationale, conditions, expiry, and side effects.

An Action is work initiated because of a Decision, Requirement, Matter, or scheduled Program activity.

A Verification Contract defines the observable outcome, source, baseline, population, threshold, period, authority, and failure response.

A Response Package is a governed set of records, evidence, approvals, exclusions, transmission metadata, and acknowledgement prepared for an authority or other external recipient.

---

# 6. Product invariants

## 6.1 Programs maintain; Matters mobilize

Stable obligations MUST remain in Programs. Change, gaps, failures, requests, exceptions, and time-bounded work MUST become Matters.

Do not create recurring Matters merely to reproduce every Program requirement as a ticket.

Do not hide a material gap inside a passive Program row when accountable action is required.

## 6.2 One truth, many views

Registers, workplans, calendars, KRIs, dashboards, committee packs, cases, and exports MUST be derived views over canonical objects.

Do not create separate module-specific copies of requirements, vendors, assets, findings, actions, or evidence.

## 6.3 Familiar language before GRC jargon

Primary language SHOULD begin with obligations, channels, services, branches, merchants, customers, accounts, assets, vendors, projects, filings, requests, and outcomes.

Control IDs, taxonomy codes, graph terminology, and operator names remain available for specialists but MUST NOT dominate ordinary tasks.

## 6.4 Existing evidence before human requests

The system MUST search authorized existing Observations and Evidence before contacting a person.

Requests MUST ask only for unresolved facts and stop when the evidence need is satisfied or no longer relevant.

## 6.5 Source authority before automated trust

Every source MUST expose owner, authoritative facts, limitations, scope, identifiers, freshness, health, mapping version, purpose, and known data-quality issues.

Successful ingestion MUST NOT be treated as truth, completeness, applicability, or evidence sufficiency.

## 6.6 Progressive integration

The product MUST remain useful with structured manual capture, managed imports, APIs, and events.

Regional banks MUST NOT require perfect enterprise APIs before obtaining value. Larger banks MUST be able to deepen automation without changing semantics.

## 6.7 Trigger-driven continuous compliance

Programs SHOULD react to calendar, change, event, and evidence triggers.

A Program MUST NOT depend solely on periodic manual reconstruction or annual blanket questionnaires where targeted refresh is possible.

## 6.8 Evidence before confidence

AI confidence MUST NOT substitute for source authority, applicability, evidence sufficiency, or assurance.

Original source material and versions MUST remain available where policy permits. Contradictory evidence MUST remain visible.

## 6.9 Decisions before dashboards

Every material state MUST identify the evidence, assessment, decision, action, response, or verification required.

“View details” alone is not a handling path.

## 6.10 Verification before closure

Implementation, evidence submission, response transmission, and task completion MUST remain separate from verified effectiveness or accepted closure.

Later contradiction or failed outcome evidence MUST reopen or supersede conclusions without deleting history.

## 6.11 Human authority for material judgment

Material applicability, legal interpretation, risk acceptance, reportability, account restriction, suspicious-report filing, protected identity disclosure, regulatory representation, and other restricted decisions remain human-governed.

AI MUST NOT silently execute them.

## 6.12 Institutional memory

Material sources, requirements, applicability, controls, evidence, conclusions, decisions, actions, responses, and assurance MUST support point-in-time reconstruction and supersession rather than overwrite.

---

# 7. Continuous-compliance rules

## 7.1 Program setup

A Program MUST define:

- governing sources and versions;
- requirements and applicability;
- scoped controls and owners;
- evidence contracts;
- review and filing calendars;
- trigger subscriptions;
- exception and escalation policy;
- independent assurance roles;
- and approved summary states.

## 7.2 Trigger processing

Calendar, institutional-change, operational-event, regulatory-change, source-health, evidence-expiry, contradiction, and failed-verification triggers MUST be evaluated against affected Program scope.

Only affected requirements, controls, evidence, or populations should be refreshed where the scope can be determined.

## 7.3 Legacy register import

Legacy spreadsheet rows MUST preserve file, sheet, row, mapping version, uploader or managed source, import time, and validation state.

The system MUST distinguish imported working records from approved sources, requirements, controls, conclusions, and evidence.

## 7.4 Filing and certification

A filing or certification package MUST be assembled from governed current evidence, approved conclusions, included and excluded records, reviewers, signatories, and a point-in-time manifest.

A filing task being completed MUST NOT imply that all underlying obligations are satisfied.

---

# 8. Regulatory, supervisory, and enforcement rules

The product MUST distinguish:

1. normative regulatory change;
2. supervisory or examination work;
3. investigative or enforcement casework.

An exposure draft MUST NOT be treated as final regulation. A supervisory finding MUST NOT be collapsed into a general obligation. An authority request MUST NOT automatically imply wrongdoing or authorize unrelated account, disclosure, or reporting action.

Every external-authority workflow MUST preserve:

- source authenticity and version;
- exact source provisions or directives;
- legal and confidentiality status;
- applicability or authority review;
- affected entities or subjects;
- deadlines;
- required decisions;
- evidence and exclusions;
- response or implementation status;
- acknowledgement;
- and retention or legal hold.

---

# 9. Evidence, capture, population, and reconciliation rules

## 9.1 Claim-centric evidence

Evidence exists to support or contradict a precise Claim for a defined purpose, population, scope, and period.

Never reduce evidence to a generic file field, checklist attachment, unversioned URL, context-free attestation, or binary present/missing flag.

## 9.2 Spreadsheet and media ingestion

Spreadsheet import MUST distinguish uploaded, parsed, mapped, accepted as Observation, reconciled, and sufficient for a Claim.

Photo, scan, audio, and video interpretation MUST preserve originals, extraction coordinates or offsets where feasible, model/version lineage, explicit versus inferred values, correction, and human confirmation.

Do not claim a photo proves invisible security or continuous control operation.

## 9.3 Population integrity

Any percentage or completion state MUST expose its denominator and exclusions.

Population views SHOULD distinguish resolved, unresolved, stale, contradictory, not applicable, excluded, and unauthorized records.

## 9.4 Matching and bulk action

Entity resolution MUST support provisional, unresolved, merge, unmerge, provenance, review, and history.

Bulk operations MUST enforce server-side authorization per object, show exact criteria and counts, expose exclusions and failures, remain idempotent where applicable, and preserve a reconstructable manifest.

---

# 10. AI implementation rules

AI is a governed compiler from messy sources into proposed structured Observations, Requirements, mappings, Claims, questions, summaries, options, or domain commands.

A model is not an operator, an operator is not an authority, and model output is not evidence.

Material AI output MUST include exact source references and versions, scope, period, assumptions, uncertainty, and unresolved contradiction.

General model knowledge MUST NOT establish a material institutional fact or regulatory requirement.

AI output used in workflows MUST be structured, schema-validated, domain-validated, authorization-checked, policy-checked, and approval-gated before authoritative mutation.

All documents, spreadsheets, media, messages, and retrieved content are untrusted. Tool permission and material-action authority MUST remain outside prompts.

AI MUST be able to abstain. Core Program and Matter workflows MUST remain usable when AI is unavailable.

---

# 11. Security, privacy, and authorization

All access is deny-by-default and enforced server-side across reads, counts, search, graph traversal, exports, caches, embeddings, AI context, workers, bulk operations, and writes.

Unauthorized users MUST NOT infer protected existence, count, identity, title, snippet, relationship, or timing information.

Protected reports, authority cases, legal privilege, customer records, suspicious-reporting work, and protected identities require stronger purpose and access boundaries.

Exports and Response Packages MUST re-evaluate authorization and record requester, purpose, scope, included versions, exclusions, classification, and manifest.

Logs MUST NOT contain secrets, unnecessary personal data, raw restricted evidence, protected identity, or unrestricted model context.

---

# 12. Visual and interaction rules

Primary surfaces are:

- **Today** — role-specific attention;
- **Programs** — continuing compliance and assurance;
- **Work** — Matters, actions, evidence requests, reviews, and approvals;
- **Explore** — authorized institutional inquiry;
- **Configure** — restricted Program, source, scope, evidence, authority, and policy administration.

Capture and Respond are contextual lightweight experiences, not mandatory permanent navigation for every user.

Do not reintroduce Graph, Evidence Fabric, Decision Ledger, Assurance, AI Operator, or internal bounded-context names as mandatory top-level navigation.

ClearSight MUST feel calm, direct, precise, relatable, institutional, accessible, and trustworthy.

Use cards for small attention queues, tables for populations, timelines for history, comparisons for contradiction, split views for source interpretation, paths for dependencies, and charts only for a defined question.

Green means a sufficiently evidenced and accepted state, never merely uploaded, submitted, assigned, or implemented.

---

# 13. Architecture rules

Begin with a coherent modular core, authoritative relational model, versioned object storage, durable workflows and outbox, rebuildable projections, and replaceable adapters.

Programs and Matters are product aggregates, not necessarily single database tables or services.

A module MUST NOT bypass another module’s invariants by directly mutating its storage.

Search, graph, vector, analytics, and reporting are projections, not authoritative stores.

A dedicated graph engine, autonomous agent platform, or large microservice estate requires measured justification.

---

# 14. Testing requirements

Tests must prove complete Program and Matter behavior under positive, negative, ambiguous, stale, partial, unauthorized, degraded, offline, historical, and adversarial conditions.

Required journeys include:

- continuing NDPA Program with ROPA and DPIA triggers;
- new regulatory publication to approved implementation and evidence;
- authority request to protected response package;
- legacy register import and reconciliation;
- ATM/POS population evidence and verification;
- source degradation;
- control exception and failed verification;
- filing and point-in-time reconstruction;
- protected reporting;
- and malicious document or spreadsheet content.

Fixtures MUST NOT inject desired sources, applicability, mappings, owners, conclusions, or verification outcomes.

---

# 15. Change protocol

For every meaningful change:

1. identify whether it affects a Program, Matter, or shared primitive;
2. identify user and institutional outcome;
3. identify authoritative sources and scope;
4. define authority and privacy boundaries;
5. distinguish observations, conclusions, decisions, actions, and outcomes;
6. define trigger, failure, retry, cancellation, and recovery behavior;
7. define evidence, response, and audit requirements;
8. add or update acceptance tests;
9. review visual and accessibility regressions;
10. synchronize canonical documentation and ADRs.

---

# 16. Prohibited shortcuts

Do not:

- rebuild each spreadsheet as a separate module;
- use one generic record or status for unrelated lifecycles;
- treat a register row as authoritative regulation without source lineage;
- create a Matter for every passive recurring requirement;
- hide a material Program gap because no ticket exists;
- close remediation or authority work at task completion;
- allow AI to publish final legal interpretation or restricted action;
- use embeddings or frontend hiding as authorization;
- expose protected case data to ordinary analytics;
- flood users with broad questionnaires or reminders when targeted evidence or automation is possible;
- copy competitor screens;
- rely on decorative glass, glow, metric walls, or chat as the product shell;
- or document planned behavior as implemented.

---

# 17. Definition of done

Work is complete only when:

- Program or Matter semantics are correct;
- canonical sources and relationships are preserved;
- authorization is enforced server-side;
- evidence and audit lineage are complete;
- failure and recovery paths work;
- tests cover meaningful negative cases;
- accessibility and visual regression pass;
- performance is within budget;
- migrations and rollback are safe;
- documentation is synchronized;
- and no planned capability is misrepresented as implemented.

For a material workflow, completion also requires a passing end-to-end path from source or trigger through evidence, decision or response, action, verification, and historical reconstruction.