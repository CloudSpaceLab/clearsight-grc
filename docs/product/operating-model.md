# ClearSight Bank Operating Model

This document defines the canonical product semantics beneath ClearSight’s Programs, Matters, evidence, decisions, responses, and verification.

It describes the smallest reusable set of concepts through which ClearSight can support regional, national, and multinational banks without exposing internal architecture or requiring perfect integrations.

It conforms to [`ease-of-use-standard.md`](ease-of-use-standard.md).

---

# 1. Product objective

ClearSight should help each stakeholder:

1. understand the continuing Program or bounded Matter relevant to them;
2. see what applies, what is known, and what is missing, stale, contradictory, or uncertain;
3. reuse approved bank inventories and evidence rather than re-entering information;
4. review grounded recommendations rather than start from blank pages;
5. provide or inspect only the minimum necessary information;
6. make or route an authorized decision or response;
7. coordinate action;
8. verify the defined outcome or authority response;
9. preserve complete temporal and audit history.

Routine work should normally require only a few steps and less than five minutes of active user effort.

---

# 2. Canonical operating loops

## 2.1 Program loop

```text
Maintain Requirements and scope
→ observe current implementation and evidence
→ compute compliance state
→ detect change, gap, expiry, or exception
→ create or update Matter
→ resolve and verify
→ refresh Program state
```

## 2.2 Matter loop

```text
Trigger or source received
→ resolve scope and affected objects
→ retrieve known context
→ identify missing or contradictory facts
→ recommend or request the next action
→ decide or respond within authority
→ act
→ verify outcome or acknowledgement
→ update Program and institutional memory
```

---

# 3. User-facing aggregates

## 3.1 Program

A long-lived body of continuing obligations, controls, evidence, review activities, exceptions, filings, and assurance.

Required attributes:

- Program type and version;
- owning function and accountable authority;
- institution, legal entity, jurisdiction, and business scope;
- applicable Requirements;
- scoped Control Implementations;
- Evidence Contracts;
- schedules and triggers;
- Compliance State;
- open Matters;
- assurance and filing history;
- configuration and policy versions.

Program examples:

- NDPA;
- AML/CFT;
- CBN cybersecurity;
- PCI DSS;
- ISO 27001;
- operational resilience;
- third-party assurance;
- RCSA;
- policy lifecycle;
- regulatory returns.

## 3.2 Matter

A bounded occurrence requiring assessment, evidence, decision, action, response, or verification.

Required attributes:

- Matter type;
- source or trigger;
- scope, period, and affected objects;
- linked Programs and Requirements;
- current state and priority;
- known facts, missing facts, and contradictions;
- accountable owner and required authority;
- evidence and decisions;
- actions, response, and dependencies;
- verification or acknowledgement;
- temporal history.

Matter types include regulatory change, supervisory finding, authority request, risk situation, control gap, exception, incident, loss, breach, vendor deficiency, complaint, KRI breach, evidence contradiction, and failed verification.

---

# 4. Universal primitives

## 4.1 Scope

The bounded part of the institution being governed.

Examples:

- institution or legal entity;
- jurisdiction or region;
- Program;
- channel or service;
- product;
- branch or operating unit;
- customer, account, merchant, or transaction population;
- vendor relationship;
- application, system, asset, or process;
- project or change;
- processing activity or data set.

Scopes may be nested and versioned.

Active scope and period must be explicit before material action, approval, export, bulk change, or evidence submission.

## 4.2 Authority Source

An immutable, versioned external or internal authoritative source:

- law, regulation, circular, guideline, standard, licence condition, contract, court instrument, supervisory communication, authority request, approved interpretation, policy, or standard.

A spreadsheet row is not automatically an Authority Source.

## 4.3 Requirement

A source-linked statement describing what an actor must, must not, may, or is expected to do.

Attributes include actor, modality, action, object, scope, condition, threshold, frequency, deadline, exception, evidence expectation, and source provision.

## 4.4 Applicability

A versioned determination of where and when a Requirement applies.

Possible states:

- applicable;
- partially applicable;
- not applicable;
- potentially applicable—information required;
- applies later;
- superseded.

Material applicability requires authorized human approval.

## 4.5 Exposure Pattern

A reusable description of how an activity, service, population, dependency, control, or obligation can fail or cause harm.

Initial families:

- availability and resilience;
- asset and inventory integrity;
- identity and access;
- transaction integrity;
- reconciliation and settlement;
- fraud and abuse;
- data and privacy;
- customer and conduct harm;
- third-party and concentration dependency;
- change and configuration integrity;
- physical and environmental integrity;
- regulatory or contractual non-conformance;
- model or automated-decision failure;
- evidence and data-quality uncertainty.

## 4.6 Control Objective

The outcome that must be achieved.

## 4.7 Control Implementation

The actual policy, process, system rule, approval gate, monitoring mechanism, review, training, contractual clause, or operating practice used in a defined scope.

One Control Objective may have several scoped implementations.

## 4.8 Claim

A precise statement that can be supported, contradicted, qualified, or unresolved.

A Claim includes subject, scope, period, purpose, materiality, Evidence Contract, conclusion state, and version.

## 4.9 Evidence Contract

A policy defining:

- required facts;
- acceptable sources;
- source authority and limitations;
- population and period;
- coverage;
- freshness;
- independence;
- contradiction rules;
- approval;
- refresh schedule or triggers;
- failure handling.

## 4.10 Observation

A normalized, source-preserving record of something observed, submitted, imported, measured, extracted, or asserted.

Attributes:

- subject and property;
- value;
- source identity;
- capture method;
- effective and capture time;
- scope and population;
- original artifact or source reference;
- transformation history;
- authority and limitation;
- confidence or review state;
- sensitivity;
- version.

Sources include forms, dropdowns, photos, scans, spreadsheets, documents, APIs, database exports, telemetry, messages, attestations, vendor submissions, customer reports, protected reports, and external intelligence.

An Observation is not automatically a verified fact.

## 4.11 Conclusion and Compliance State

A Conclusion states what current evidence supports.

Possible states:

- supported;
- partially supported;
- unsupported;
- contradicted;
- indeterminate;
- expired;
- not applicable.

Program Compliance State combines multiple dimensions and must retain the underlying basis.

## 4.12 Decision

An authorized selection among options.

Includes context, evidence, uncertainty, options, effects, cost and dependencies where relevant, selected option, authority, rationale, dissent, conditions, expiry, review triggers, action plan, and verification.

## 4.13 Action

Work initiated because of a Requirement, Matter, decision, finding, or response obligation.

External completion is implementation evidence, not verified outcome.

## 4.14 Response Package

A governed set of records, evidence, explanations, redactions, approvals, transmission metadata, and acknowledgement prepared for an external authority or examiner.

## 4.15 Verification Contract

Defines expected observable outcome, baseline, scope, measurement source, threshold, observation period, evidence, authority, and failure response.

The system verifies whether defined criteria were met; it does not overstate causal proof.

---

# 5. Source Registry and inventory reuse

Every source has a Source Profile:

- source and owner;
- collection method;
- authoritative facts;
- limitations;
- scope;
- identifiers;
- mapping version;
- expected freshness;
- current health;
- known data-quality issues;
- access and purpose policy;
- dependent claims and Programs.

Approved inventories should supply workflow scope and controlled values:

- CMDB and application inventory;
- asset systems;
- branches and organization directories;
- HR and identity directories;
- vendor and contract systems;
- customer and account systems;
- acquiring and channel systems;
- ITSM and project systems;
- ROPA and BIA;
- policy and evidence repositories.

Users confirm or correct unresolved information rather than rebuild inventory.

---

# 6. Progressive integration

## Level 0

Controlled lists, forms, photos, spreadsheets, and documents.

## Level 1

Scheduled files, SFTP, database exports, and recurring imports.

## Level 2

APIs.

## Level 3

Events and telemetry.

Every level produces the same Observation model and uses the same Program and Matter semantics.

---

# 7. Ease-of-use semantics

## 7.1 Prefill before asking

Before creating a request or editable field, search authorized sources for an existing value.

## 7.2 Minimum-question generation

Determine exact unresolved facts and ask only those.

## 7.3 Recommendation-first workflow

Where approved, AI should propose Requirements, mappings, evidence requests, actions, verification, summaries, response indexes, and assignments with source lineage.

## 7.4 Review by exception

Human reviewers focus on changed, low-confidence, contradictory, unsupported, material, or high-impact items.

## 7.5 Save and resume

Every interruptible Matter preserves completed work, changes, blockers, and next action.

## 7.6 Active-effort budget

Routine workflows target less than five minutes. Complex workflows reach a safe saved next state in that time.

## 7.7 Stable interaction across source maturity

Replacing a spreadsheet with an API should remove effort without changing product semantics or user mental model.

---

# 8. Matching, populations, and reconciliation

Population definitions preserve denominator, scope, time, inclusion logic, and exclusions.

Matching states:

- matched;
- provisionally matched;
- unresolved;
- contradictory;
- duplicate;
- rejected;
- superseded.

AI may propose matches. Material merges require policy or review and must support unmerge with history.

Bulk operations enforce authorization per object and produce reconstructable manifests.

---

# 9. Flexible configuration

Different banks use one product through:

- base banking model;
- Programs;
- Matter types;
- channel packs;
- jurisdiction packs;
- institution profile;
- source profiles;
- evidence contracts;
- authority matrices;
- workflow and trigger policies;
- role-specific views.

Avoid arbitrary schemas, per-customer forks, uncontrolled scripts, and large generic form builders as the primary model.

---

# 10. AI role

AI acts as a governed compiler between messy inputs and structured objects.

AI may extract, normalize, classify, compare, map, summarize, recommend, and draft.

AI does not become:

- the source of institutional truth;
- the evidence itself;
- the authority;
- the sole interface;
- a route around policy;
- a reason to add user effort.

Every material AI output preserves source, version, scope, assumptions, uncertainty, structured output, validation, and review state.

---

# 11. User experience boundary

Primary surfaces:

- Today;
- Programs;
- Work;
- Explore;
- Configure.

Focused Respond and Capture experiences expose only the necessary subset of context.

Graph, evidence engine, decision ledger, workflow runtime, and AI operator platform are internal capabilities, not mandatory navigation concepts.

---

# 12. Product invariants

1. Programs maintain continuity; Matters handle change and exception.
2. Routine work targets under five minutes of active effort.
3. Complex work reaches a clear saved next state within five minutes.
4. Prefill before asking.
5. Existing evidence before requests.
6. Approved inventories before manual re-entry.
7. AI-grounded first drafts before blank-page work.
8. One clear next action.
9. Scope before action.
10. Banking language before GRC jargon.
11. Source authority before automated trust.
12. Evidence before confidence.
13. Contradiction before false certainty.
14. Decisions before dashboards.
15. Verification before closure.
16. Human authority for material judgment.
17. Progressive integration without semantic change.
18. Institutional memory over periodic reconstruction.

---

# 13. Definition of success

The operating model succeeds when:

- regional banks can begin with spreadsheets and controlled capture;
- national and multinational banks can add APIs, events, entities, and jurisdictions without changing semantics;
- approved inventories eliminate repetitive entry;
- Programs remain continuously current;
- Matters contain all context required for action;
- routine users complete work in a few steps;
- AI recommendations reduce assembly without becoming authority;
- source weakness and contradiction remain visible;
- one underlying record can drive register, KRI, dashboard, audit, filing, and committee views;
- closure requires evidence and verification;
- every material state can be reconstructed later.