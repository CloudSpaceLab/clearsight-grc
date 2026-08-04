# ClearSight Comprehensive Implementation Plan

This plan turns the ClearSight product thesis into an implementable bank-grade system without allowing the product to regress into a generic GRC portal.

It is intentionally detailed. Each phase contains objectives, tasks, subtasks, dependencies, deliverables, and acceptance gates.

Checkboxes indicate implementation work. They do not imply that any work is currently complete.

---

# 1. Delivery principles

## 1.1 Build the differentiators as the architecture

The following are not optional add-ons:

- Institutional Risk Graph
- Materiality Compiler
- Living Evidence Fabric
- Decision Ledger
- Verification Contracts
- Governed AI Operators
- Protected Reporting
- Calm Risk Command Surface

Commodity GRC features must be implemented around these mechanisms, not the other way around.

## 1.2 Begin as a coherent modular core

Start with a modular monolith or similarly disciplined core with explicit bounded contexts, transactional integrity, an outbox, durable workflows, and replaceable integrations.

Do not begin with many microservices. Split a module only when measured requirements justify independent scaling, isolation, ownership, deployment, or regulatory boundaries.

## 1.3 Preserve source, derivation, and decision history

The platform must separate:

- source facts;
- signals;
- claims;
- evidence;
- conclusions;
- decisions;
- actions;
- and outcomes.

Material history is versioned and reconstructable.

## 1.4 AI is optional to availability, mandatory to governance

The system must remain usable when AI services are unavailable. When AI is used, identity, purpose, scope, source lineage, structured output, policy, evaluation, and audit are mandatory.

## 1.5 Deliver vertical slices

Each milestone should demonstrate a complete path through:

> **Sense → Explain → Decide → Act → Prove**

Avoid delivering isolated data models or UI shells that do not exercise the real domain path.

---

# 2. Recommended repository topology

The exact technology stack requires ADR approval, but the repository should evolve toward a structure similar to:

```text
.
├── README.md
├── AGENTS.md
├── docs/
│   ├── architecture/
│   ├── product/
│   ├── quality/
│   └── decisions/
├── apps/
│   ├── web/                 # authenticated enterprise application
│   ├── external-portal/     # whistleblower, customer and vendor intake
│   └── mobile/              # optional native shell or PWA capture surface
├── services/
│   └── core/                # modular domain application
├── packages/
│   ├── design-system/
│   ├── domain-contracts/
│   ├── event-contracts/
│   ├── authorization/
│   ├── model-gateway/
│   └── integration-sdk/
├── workers/
│   ├── ingestion/
│   ├── evidence-processing/
│   └── projections/
├── tests/
│   ├── e2e/
│   ├── evaluations/
│   ├── security/
│   └── performance/
└── infrastructure/
```

The first implementation may keep workers and modules in one deployable unit while preserving boundaries in code.

---

# 3. Cross-cutting workstreams

Every phase must account for these workstreams.

## Product and domain

- canonical vocabulary;
- user and institutional outcomes;
- authority and workflow semantics;
- acceptance scenarios;
- and product telemetry.

## Experience and design

- information architecture;
- design system;
- light and dark modes;
- responsive behavior;
- accessibility;
- visual regression;
- and usability validation.

## Platform and data

- tenancy;
- temporal versioning;
- events;
- storage;
- search and graph projections;
- workflows;
- and observability.

## Security and privacy

- authorization;
- classification;
- encryption;
- retention;
- protected identity;
- threat modeling;
- audit;
- and deployment boundaries.

## AI and intelligence

- model gateway;
- operator registry;
- grounding;
- structured outputs;
- evaluations;
- monitoring;
- and human review.

## Integrations

- source mapping;
- trust and provenance;
- idempotency;
- replay;
- health;
- and reconciliation.

## Quality and operations

- automated tests;
- end-to-end golden journeys;
- performance budgets;
- reliability;
- deployment;
- rollback;
- and runbooks.

---

# Phase 0 — Product, architecture, and repository foundation

## Objective

Establish the non-negotiable product model, architecture decisions, design language, security baseline, and engineering workflow before feature development creates accidental constraints.

## Dependencies

None.

## 0.1 Product definition

- [ ] **P0.1.1 Define primary bank personas**
  - [ ] CRO
  - [ ] CCO
  - [ ] CISO
  - [ ] operational risk head
  - [ ] business/service owner
  - [ ] control owner and performer
  - [ ] compliance analyst
  - [ ] internal auditor
  - [ ] investigator
  - [ ] board or committee member
  - [ ] employee evidence respondent
  - [ ] vendor respondent
  - [ ] customer reporter
  - [ ] anonymous whistleblower
- [ ] **P0.1.2 Define authority-sensitive jobs to be done**
  - [ ] detect material exposure
  - [ ] challenge a conclusion
  - [ ] collect evidence
  - [ ] decide treatment
  - [ ] accept risk
  - [ ] verify remediation
  - [ ] prepare committee material
  - [ ] answer an examiner
  - [ ] report confidentially
- [ ] **P0.1.3 Define the first five golden journeys**
  - [ ] privileged-access evidence gap
  - [ ] payment resilience exposure
  - [ ] material risk acceptance
  - [ ] remediation verification failure
  - [ ] anonymous protected report
- [ ] **P0.1.4 Define product telemetry**
  - [ ] decision latency
  - [ ] human evidence effort
  - [ ] evidence debt
  - [ ] duplicate request avoidance
  - [ ] verification failure
  - [ ] executive noise suppression

## 0.2 Canonical domain language

- [ ] **P0.2.1 Create domain glossary**
  - [ ] fact
  - [ ] signal
  - [ ] event
  - [ ] incident
  - [ ] claim
  - [ ] evidence
  - [ ] assertion
  - [ ] conclusion
  - [ ] risk scenario
  - [ ] appetite
  - [ ] control objective
  - [ ] control implementation
  - [ ] finding
  - [ ] issue
  - [ ] decision
  - [ ] action
  - [ ] outcome
  - [ ] verification contract
- [ ] **P0.2.2 Define naming conventions for IDs, states, events, APIs, and versions**
- [ ] **P0.2.3 Define prohibited semantic collapses**
  - [ ] task completion is not outcome
  - [ ] attachment is not supporting evidence by default
  - [ ] signal is not incident
  - [ ] obligation is not source text
  - [ ] control objective is not implementation

## 0.3 Architecture decision records

Create and approve ADRs for:

- [ ] **P0.3.1 Application architecture** — modular monolith boundaries and split criteria
- [ ] **P0.3.2 Primary backend language and framework**
- [ ] **P0.3.3 Web framework and rendering strategy**
- [ ] **P0.3.4 Relational database and migration tooling**
- [ ] **P0.3.5 Temporal/versioning strategy**
- [ ] **P0.3.6 Event outbox and messaging strategy**
- [ ] **P0.3.7 Workflow engine or durable job strategy**
- [ ] **P0.3.8 Object storage and evidence integrity strategy**
- [ ] **P0.3.9 Authorization engine** — application-native, OPA, Cedar, or equivalent
- [ ] **P0.3.10 Search and vector projection strategy**
- [ ] **P0.3.11 Graph projection and dedicated graph-engine decision gate**
- [ ] **P0.3.12 Model gateway and provider-routing strategy**
- [ ] **P0.3.13 Deployment modes** — SaaS, dedicated, private, on-premises
- [ ] **P0.3.14 Observability and audit separation**

Each ADR must include:

- context;
- options;
- decision;
- consequences;
- security and operational impact;
- portability;
- and revisit triggers.

## 0.4 Threat model and privacy model

- [ ] **P0.4.1 Create system threat model**
  - [ ] cross-tenant access
  - [ ] graph inference
  - [ ] evidence tampering
  - [ ] malicious uploads
  - [ ] prompt injection
  - [ ] operator tool abuse
  - [ ] export leakage
  - [ ] insider misuse
  - [ ] protected identity exposure
  - [ ] integration compromise
- [ ] **P0.4.2 Define data classification scheme**
- [ ] **P0.4.3 Define purpose and retention policies**
- [ ] **P0.4.4 Define protected-report identity isolation model**
- [ ] **P0.4.5 Define cryptographic key hierarchy**
- [ ] **P0.4.6 Define legal hold and deletion behavior**

## 0.5 Design foundation

- [ ] **P0.5.1 Create visual mood and anti-reference board**
- [ ] **P0.5.2 Define semantic color roles**
- [ ] **P0.5.3 Define typography and numeric styles**
- [ ] **P0.5.4 Define spacing, radius, border, elevation, blur, and motion tokens**
- [ ] **P0.5.5 Define light and dark theme parity**
- [ ] **P0.5.6 Prototype the four-stage interaction grammar**
  - [ ] Brief
  - [ ] Explain
  - [ ] Act
  - [ ] Prove
- [ ] **P0.5.7 Prototype evidence micro-request on mobile and desktop**
- [ ] **P0.5.8 Test executive comprehension with representative scenarios**

## 0.6 Repository and delivery pipeline

- [ ] **P0.6.1 Scaffold repository**
- [ ] **P0.6.2 Configure formatting, linting, type checking, and commit conventions**
- [ ] **P0.6.3 Configure unit, integration, accessibility, and end-to-end test runners**
- [ ] **P0.6.4 Configure dependency and secret scanning**
- [ ] **P0.6.5 Configure SBOM and artifact signing plan**
- [ ] **P0.6.6 Configure preview environments**
- [ ] **P0.6.7 Configure database migration verification**
- [ ] **P0.6.8 Configure visual regression baseline**
- [ ] **P0.6.9 Add pull request template enforcing `AGENTS.md` review**

## Phase 0 deliverables

- Approved product glossary
- Five fully specified golden journeys
- Initial ADR set
- Threat and privacy model
- Design tokens and first prototypes
- Working repository scaffold and CI

## Phase 0 acceptance gate

Do not begin domain implementation until:

- facts, claims, conclusions, decisions, and outcomes are distinct;
- protected identity architecture is approved;
- the first vertical-slice scenarios are testable;
- the design direction demonstrates calm executive hierarchy in both themes;
- and core stack choices are documented through ADRs.

---

# Phase 1 — Trust, tenancy, identity, audit, and temporal foundation

## Objective

Build the security and history foundation that every later capability depends on.

## Dependencies

Phase 0 complete.

## 1.1 Tenant and institutional scope

- [ ] **P1.1.1 Implement tenant model**
- [ ] **P1.1.2 Implement legal entities, jurisdictions, business units, and environments**
- [ ] **P1.1.3 Create immutable tenant and entity context propagation**
- [ ] **P1.1.4 Prevent client-controlled tenant selection for authoritative writes**
- [ ] **P1.1.5 Add isolation tests across database, cache, search, queue, object store, and AI retrieval**

## 1.2 Identity and enterprise access

- [ ] **P1.2.1 Local development identity provider**
- [ ] **P1.2.2 OIDC/SAML integration boundary**
- [ ] **P1.2.3 MFA and session policy hooks**
- [ ] **P1.2.4 SCIM or directory provisioning boundary**
- [ ] **P1.2.5 Service identities for workers, integrations, and operators**
- [ ] **P1.2.6 Break-glass identity and approval flow**
- [ ] **P1.2.7 Authentication event audit**

## 1.3 Authorization

- [ ] **P1.3.1 Implement deny-by-default authorization service**
- [ ] **P1.3.2 Support RBAC, attributes, relationships, purpose, and sensitivity**
- [ ] **P1.3.3 Implement authority matrix for decisions**
- [ ] **P1.3.4 Implement segregation-of-duties and conflict checks**
- [ ] **P1.3.5 Implement field- and evidence-level restrictions**
- [ ] **P1.3.6 Implement authorization-aware search and graph query contract**
- [ ] **P1.3.7 Add explainable authorization decisions for administrators**
- [ ] **P1.3.8 Add exhaustive negative authorization tests**

## 1.4 Temporal records and versioning

- [ ] **P1.4.1 Implement common version metadata**
  - [ ] valid from/to
  - [ ] recorded at
  - [ ] supersedes
  - [ ] actor
  - [ ] reason
- [ ] **P1.4.2 Implement optimistic concurrency**
- [ ] **P1.4.3 Implement append-only material records**
- [ ] **P1.4.4 Implement point-in-time query primitives**
- [ ] **P1.4.5 Add temporal consistency tests**

## 1.5 Audit ledger

- [ ] **P1.5.1 Define audit event schema**
- [ ] **P1.5.2 Separate operational logs from immutable audit**
- [ ] **P1.5.3 Capture actor, purpose, scope, command, result, and correlation**
- [ ] **P1.5.4 Protect sensitive payloads through references and redaction**
- [ ] **P1.5.5 Implement audit query with strict authorization**
- [ ] **P1.5.6 Implement tamper-evidence strategy**
- [ ] **P1.5.7 Test point-in-time action reconstruction**

## 1.6 Events and durable processing

- [ ] **P1.6.1 Implement transactional outbox**
- [ ] **P1.6.2 Define event envelope and schema versioning**
- [ ] **P1.6.3 Implement idempotent consumers**
- [ ] **P1.6.4 Implement retry, dead-letter, and replay tooling**
- [ ] **P1.6.5 Implement correlation and causation IDs**
- [ ] **P1.6.6 Add event contract tests**

## 1.7 Evidence object storage foundation

- [ ] **P1.7.1 Implement versioned object storage abstraction**
- [ ] **P1.7.2 Implement content hashing and integrity manifests**
- [ ] **P1.7.3 Implement malware scanning pipeline**
- [ ] **P1.7.4 Implement classification-aware encryption and access**
- [ ] **P1.7.5 Implement retention, legal hold, and deletion hooks**
- [ ] **P1.7.6 Implement resumable upload contract**

## 1.8 Observability and operations

- [ ] **P1.8.1 Structured application logs**
- [ ] **P1.8.2 Metrics and traces**
- [ ] **P1.8.3 Sensitive-data logging controls**
- [ ] **P1.8.4 Health and readiness endpoints**
- [ ] **P1.8.5 Background-job visibility**
- [ ] **P1.8.6 Initial SLOs and error budgets**

## Phase 1 deliverables

- Secure tenant and entity foundation
- Enterprise identity boundary
- Relationship- and purpose-aware authorization
- Temporal versioning
- Immutable audit
- Durable events
- Secure evidence storage
- Operational observability

## Phase 1 acceptance gate

Pass:

- cross-tenant negative tests;
- point-in-time record reconstruction;
- duplicate-event idempotency;
- unauthorized export and search tests;
- evidence integrity verification;
- and service-identity audit tests.

---

# Phase 2 — Institutional Risk Graph

## Objective

Create the connected, temporal semantic substrate used by every later capability.

## Dependencies

Phase 1.

## 2.1 Graph ontology

- [ ] **P2.1.1 Define canonical entity types**
- [ ] **P2.1.2 Define typed relationship catalog**
- [ ] **P2.1.3 Define required provenance by relationship type**
- [ ] **P2.1.4 Define valid-time behavior**
- [ ] **P2.1.5 Define sensitivity inheritance and relationship-level policy**
- [ ] **P2.1.6 Define entity aliases and external identifiers**

## 2.2 Authoritative graph model

- [ ] **P2.2.1 Implement versioned entity aggregate**
- [ ] **P2.2.2 Implement versioned typed relationship aggregate**
- [ ] **P2.2.3 Implement provenance references**
- [ ] **P2.2.4 Implement relationship confidence and review state**
- [ ] **P2.2.5 Implement supersession and correction**
- [ ] **P2.2.6 Implement graph domain events**

## 2.3 Entity resolution

- [ ] **P2.3.1 Exact identifier matching**
- [ ] **P2.3.2 Alias and normalized-name matching**
- [ ] **P2.3.3 Contextual candidate matching**
- [ ] **P2.3.4 Human review for ambiguous merges**
- [ ] **P2.3.5 Merge and unmerge with history**
- [ ] **P2.3.6 Evaluation dataset for entity resolution**

## 2.4 Graph projections and query

- [ ] **P2.4.1 Build relational traversal APIs**
- [ ] **P2.4.2 Build graph projection**
- [ ] **P2.4.3 Implement authorized neighborhood query**
- [ ] **P2.4.4 Implement shortest/most-relevant path query**
- [ ] **P2.4.5 Implement dependency and concentration query**
- [ ] **P2.4.6 Implement point-in-time graph reconstruction**
- [ ] **P2.4.7 Benchmark dedicated graph engine decision threshold**

## 2.5 Initial institutional import

- [ ] **P2.5.1 Define CSV/API import contracts**
- [ ] **P2.5.2 Import legal entities and business units**
- [ ] **P2.5.3 Import critical services and processes**
- [ ] **P2.5.4 Import systems and vendors**
- [ ] **P2.5.5 Import accountable roles and owners**
- [ ] **P2.5.6 Produce reconciliation and unresolved-match report**

## 2.6 Graph experience

- [ ] **P2.6.1 Implement relationship path component**
- [ ] **P2.6.2 Implement dependency map with progressive detail**
- [ ] **P2.6.3 Implement temporal comparison**
- [ ] **P2.6.4 Implement accessible textual path summary**
- [ ] **P2.6.5 Implement graph authorization-safe empty states**
- [ ] **P2.6.6 Add performance limits and virtualization**

## Phase 2 acceptance gate

Demonstrate:

- a payment service mapped to people, process, system, vendor, fourth party, and jurisdiction;
- complete provenance for each edge;
- authorized point-in-time reconstruction;
- no inaccessible-neighbor leakage;
- and merge/unmerge history.

---

# Phase 3 — Risk, appetite, signals, and Materiality Compiler

## Objective

Turn institutional relationships and incoming signals into explainable, decision-relevant risk movement.

## Dependencies

Phases 1–2.

## 3.1 Risk scenario model

- [ ] **P3.1.1 Implement risk taxonomy**
- [ ] **P3.1.2 Implement causal risk scenario structure**
  - [ ] cause
  - [ ] event
  - [ ] affected objective/service/asset
  - [ ] consequence
- [ ] **P3.1.3 Implement inherent, current, residual, target, and stressed views**
- [ ] **P3.1.4 Implement qualitative dimensions with explicit semantics**
- [ ] **P3.1.5 Implement quantitative range/distribution hooks**
- [ ] **P3.1.6 Implement uncertainty and confidence**
- [ ] **P3.1.7 Link controls, incidents, losses, issues, and evidence**

## 3.2 Risk appetite and authority

- [ ] **P3.2.1 Implement appetite statement versions**
- [ ] **P3.2.2 Implement limits, triggers, tolerances, and prohibited conditions**
- [ ] **P3.2.3 Implement scope by entity, service, product, and risk type**
- [ ] **P3.2.4 Implement time-bound exceptions**
- [ ] **P3.2.5 Implement escalation and authority rules**
- [ ] **P3.2.6 Implement appetite evaluation explanation**

## 3.3 Signal ingestion

- [ ] **P3.3.1 Define normalized signal envelope**
- [ ] **P3.3.2 Implement source trust and health state**
- [ ] **P3.3.3 Implement deduplication and replay**
- [ ] **P3.3.4 Implement entity resolution to graph**
- [ ] **P3.3.5 Implement preliminary classification**
- [ ] **P3.3.6 Implement signal retention and raw-source reference**

## 3.4 Materiality Compiler core

- [ ] **P3.4.1 Contextual enrichment**
  - [ ] affected critical services
  - [ ] customers
  - [ ] legal entities
  - [ ] systems and vendors
  - [ ] obligations and controls
- [ ] **P3.4.2 Causal grouping**
- [ ] **P3.4.3 Impact and velocity dimensions**
- [ ] **P3.4.4 Appetite comparison**
- [ ] **P3.4.5 Concentration and propagation analysis**
- [ ] **P3.4.6 Evidence-debt input placeholder**
- [ ] **P3.4.7 Decision-relevance classification**
- [ ] **P3.4.8 Structured explanation output**
- [ ] **P3.4.9 Versioned materiality assessment**
- [ ] **P3.4.10 Human override with reason**

## 3.5 Executive material item

- [ ] **P3.5.1 Implement material item aggregate**
- [ ] **P3.5.2 Link source signals and graph snapshot**
- [ ] **P3.5.3 Identify required authority**
- [ ] **P3.5.4 Track dismissed, delegated, escalated, and superseded states**
- [ ] **P3.5.5 Prevent duplicate executive items**
- [ ] **P3.5.6 Measure false-positive and missed-materiality feedback**

## 3.6 Initial scenario vertical slice

Implement the payment-resilience scenario end to end:

- [ ] ingest service latency signal;
- [ ] ingest vendor status signal;
- [ ] connect recent change and stale failover evidence placeholder;
- [ ] group into one material item;
- [ ] compare with impact tolerance;
- [ ] identify evidence gap;
- [ ] render concise executive explanation.

## Phase 3 acceptance gate

The system must:

- preserve raw signals;
- produce one material item from related signals;
- explain all materiality dimensions;
- separate exposure from evidence uncertainty;
- identify authority;
- and suppress non-material noise without deleting it.

---

# Phase 4 — Living Evidence Fabric

## Objective

Implement the complete claim-centric evidence lifecycle and dynamic micro-request system.

## Dependencies

Phases 1–3.

## 4.1 Claims and evidence domain

- [ ] **P4.1.1 Implement claim types and versioning**
- [ ] **P4.1.2 Implement immutable evidence versions**
- [ ] **P4.1.3 Implement evidence assertions**
- [ ] **P4.1.4 Implement claim-evidence evaluations**
- [ ] **P4.1.5 Implement conclusions and approval states**
- [ ] **P4.1.6 Implement evidence invalidation and supersession**
- [ ] **P4.1.7 Link evidence to graph entities and time scope**

## 4.2 Capture pipeline

- [ ] **P4.2.1 Secure file and document capture**
- [ ] **P4.2.2 Structured attestation capture**
- [ ] **P4.2.3 API/event evidence capture**
- [ ] **P4.2.4 Email and messaging adapter contract**
- [ ] **P4.2.5 Mobile image, scan, audio, and video capture**
- [ ] **P4.2.6 Original source preservation**
- [ ] **P4.2.7 Capture and integrity manifest**
- [ ] **P4.2.8 Validation errors and recovery**

## 4.3 Evidence processing

- [ ] **P4.3.1 Malware and integrity validation**
- [ ] **P4.3.2 Text and metadata extraction**
- [ ] **P4.3.3 OCR only where necessary**
- [ ] **P4.3.4 Audio/video transcription**
- [ ] **P4.3.5 Entity and date extraction**
- [ ] **P4.3.6 Duplicate and near-duplicate detection**
- [ ] **P4.3.7 Sensitivity and redaction suggestion**
- [ ] **P4.3.8 User confirmation of machine-inferred fields**

## 4.4 Sufficiency engine

- [ ] **P4.4.1 Implement dimension model**
  - [ ] relevance
  - [ ] authenticity
  - [ ] coverage
  - [ ] freshness
  - [ ] independence
  - [ ] completeness
  - [ ] consistency
  - [ ] reliability
  - [ ] traceability
- [ ] **P4.4.2 Implement claim-type evidence policies**
- [ ] **P4.4.3 Implement materiality-sensitive minimums**
- [ ] **P4.4.4 Implement explainable sufficiency result**
- [ ] **P4.4.5 Implement override and challenge**
- [ ] **P4.4.6 Calibrate any summary score**

## 4.5 Contradiction and evidence debt

- [ ] **P4.5.1 Implement contradiction records**
- [ ] **P4.5.2 Detect temporal, scope, identity, and factual conflicts**
- [ ] **P4.5.3 Propagate contradiction to conclusions and decisions**
- [ ] **P4.5.4 Implement resolver workflow**
- [ ] **P4.5.5 Implement evidence-debt dimensions**
- [ ] **P4.5.6 Feed material evidence debt into Materiality Compiler**

## 4.6 Best-placed-source resolver

- [ ] **P4.6.1 Build candidate-source registry**
- [ ] **P4.6.2 Rank authority, directness, independence, freshness, coverage, burden, sensitivity, and conflict**
- [ ] **P4.6.3 Preserve ranking explanation**
- [ ] **P4.6.4 Support human source correction**
- [ ] **P4.6.5 Learn source reliability from validated outcomes under policy**

## 4.7 Dynamic micro-request engine

- [ ] **P4.7.1 Determine unresolved facts**
- [ ] **P4.7.2 Search and reuse authorized existing evidence**
- [ ] **P4.7.3 Generate minimal structured question**
- [ ] **P4.7.4 Prefill known context**
- [ ] **P4.7.5 Estimate recipient effort**
- [ ] **P4.7.6 Select approved channel**
- [ ] **P4.7.7 Implement delivery, view, partial response, redirect, decline, follow-up, sufficient, expired, and cancel states**
- [ ] **P4.7.8 Deduplicate overlapping requests**
- [ ] **P4.7.9 Cancel reminders when evidence arrives elsewhere**
- [ ] **P4.7.10 Measure burden and rejection reasons**

## 4.8 Evidence experience

- [ ] **P4.8.1 Evidence request component**
- [ ] **P4.8.2 Mobile capture flow**
- [ ] **P4.8.3 Evidence sufficiency panel**
- [ ] **P4.8.4 Contradiction compare view**
- [ ] **P4.8.5 Source lineage viewer**
- [ ] **P4.8.6 Evidence debt indicator**
- [ ] **P4.8.7 Original-versus-extracted comparison**
- [ ] **P4.8.8 Light/dark visual regression**

## 4.9 Privileged-access golden journey

- [ ] import IAM population;
- [ ] import approvals;
- [ ] import HR status;
- [ ] identify four unresolved accounts;
- [ ] request only missing business-need evidence;
- [ ] detect one contradiction;
- [ ] prevent conclusion until resolved;
- [ ] issue approved conclusion with lineage.

## Phase 4 acceptance gate

No evidence capability is accepted unless:

- evidence remains immutable and versioned;
- claims are explicit;
- existing evidence is searched before a person is asked;
- the request contains only unresolved facts;
- machine and human evidence can contradict one another;
- sufficiency is multidimensional;
- and point-in-time lineage is reconstructable.

---

# Phase 5 — Decision Ledger, action, and verified remediation

## Objective

Turn material items and evidence-backed conclusions into authorized decisions, executable work, and verified outcomes.

## Dependencies

Phases 1–4.

## 5.1 Decision aggregate

- [ ] **P5.1.1 Implement decision types**
- [ ] **P5.1.2 Capture material context and graph references**
- [ ] **P5.1.3 Capture evidence included and excluded**
- [ ] **P5.1.4 Capture uncertainty and contradiction**
- [ ] **P5.1.5 Implement option model**
- [ ] **P5.1.6 Implement selected option and rationale**
- [ ] **P5.1.7 Implement conditions, expiry, and review triggers**
- [ ] **P5.1.8 Implement supersession without overwrite**

## 5.2 Option and counterfactual analysis

- [ ] **P5.2.1 Model projected risk movement**
- [ ] **P5.2.2 Model cost and resource needs**
- [ ] **P5.2.3 Model implementation time and time-to-effect**
- [ ] **P5.2.4 Model dependencies**
- [ ] **P5.2.5 Model reversibility and operational/customer impact**
- [ ] **P5.2.6 Distinguish estimates from observations**
- [ ] **P5.2.7 Show confidence ranges and assumptions**

## 5.3 Approval and challenge

- [ ] **P5.3.1 Evaluate authority matrix**
- [ ] **P5.3.2 Enforce segregation of duties**
- [ ] **P5.3.3 Detect conflicts**
- [ ] **P5.3.4 Support sequential and parallel approval**
- [ ] **P5.3.5 Support challenge, dissent, conditional approval, reject, and request-more-evidence**
- [ ] **P5.3.6 Implement emergency authority with mandatory later review**
- [ ] **P5.3.7 Audit all edits and approval context**

## 5.4 Actions and external execution

- [ ] **P5.4.1 Implement action plan and dependencies**
- [ ] **P5.4.2 Implement action states distinct from verification states**
- [ ] **P5.4.3 Implement domain task interface**
- [ ] **P5.4.4 Implement generic external-task adapter**
- [ ] **P5.4.5 Implement idempotent write and reconciliation**
- [ ] **P5.4.6 Preserve external object ID, version, and source**
- [ ] **P5.4.7 Handle partial failure and compensation**

## 5.5 Verification contracts

- [ ] **P5.5.1 Implement expected outcome**
- [ ] **P5.5.2 Implement baseline and measure**
- [ ] **P5.5.3 Implement population or scope**
- [ ] **P5.5.4 Implement success/failure thresholds**
- [ ] **P5.5.5 Implement observation period**
- [ ] **P5.5.6 Implement required evidence**
- [ ] **P5.5.7 Implement acceptance authority**
- [ ] **P5.5.8 Implement failure response**

## 5.6 Verification runtime

- [ ] **P5.6.1 Start verification after implementation evidence**
- [ ] **P5.6.2 Collect outcome evidence**
- [ ] **P5.6.3 Evaluate contract**
- [ ] **P5.6.4 Route ambiguous outcomes to review**
- [ ] **P5.6.5 Mark verified effective or ineffective**
- [ ] **P5.6.6 Reopen issue on failure**
- [ ] **P5.6.7 Update risk only after accepted evidence**
- [ ] **P5.6.8 Preserve projected-versus-observed comparison**

## 5.7 Decision and remediation experience

- [ ] **P5.7.1 Decision card**
- [ ] **P5.7.2 Option comparison**
- [ ] **P5.7.3 Approval review with evidence**
- [ ] **P5.7.4 Action dependency view**
- [ ] **P5.7.5 Verification contract panel**
- [ ] **P5.7.6 Implemented-versus-verified state treatment**
- [ ] **P5.7.7 Failed verification state**

## Phase 5 acceptance gate

Pass scenarios where:

- a material acceptance cannot be approved by an unauthorized owner;
- an approval can expire after context changes;
- an external ticket can complete while the issue remains open;
- failed outcome evidence reopens remediation;
- and the full decision is reconstructable years later.

---

# Phase 6 — Risk Command Surface and executive experience

## Objective

Deliver the calm, AI-first interface that makes the connected system usable with minimal executive effort.

## Dependencies

Phases 2–5. Design-system work began in Phase 0.

## 6.1 Design system implementation

- [ ] **P6.1.1 Implement semantic tokens**
- [ ] **P6.1.2 Implement typography and numeric components**
- [ ] **P6.1.3 Implement surfaces, focus, motion, and accessibility primitives**
- [ ] **P6.1.4 Implement core components from experience principles**
- [ ] **P6.1.5 Build Storybook or equivalent component catalog**
- [ ] **P6.1.6 Add automated accessibility tests**
- [ ] **P6.1.7 Add light/dark visual regression**

## 6.2 Application shell and navigation

- [ ] **P6.2.1 Today**
- [ ] **P6.2.2 Explore**
- [ ] **P6.2.3 Act**
- [ ] **P6.2.4 Prove**
- [ ] **P6.2.5 Govern**
- [ ] **P6.2.6 Role-specific simplification**
- [ ] **P6.2.7 Keyboard command navigation**

## 6.3 Today brief

- [ ] **P6.3.1 Role-aware material item ranking**
- [ ] **P6.3.2 Default limit of three to seven items**
- [ ] **P6.3.3 Material-change card**
- [ ] **P6.3.4 Evidence state and reason**
- [ ] **P6.3.5 Required authority and due time**
- [ ] **P6.3.6 Safe automation summary**
- [ ] **P6.3.7 No-material-change state**
- [ ] **P6.3.8 Expanded monitoring mode**

## 6.4 Explain workspace

- [ ] **P6.4.1 Concise explanation**
- [ ] **P6.4.2 Graph relationship path**
- [ ] **P6.4.3 Evidence and contradiction**
- [ ] **P6.4.4 Risk/appetite dimensions**
- [ ] **P6.4.5 Historical decisions and outcomes**
- [ ] **P6.4.6 Point-in-time toggle**
- [ ] **P6.4.7 Accessible textual equivalents**

## 6.5 Natural-language investigation

- [ ] **P6.5.1 Global command surface**
- [ ] **P6.5.2 Authorized query planning**
- [ ] **P6.5.3 Source and time-scope display**
- [ ] **P6.5.4 Assumption and contradiction display**
- [ ] **P6.5.5 Transition from answer to structured action**
- [ ] **P6.5.6 Conversation history with governed object references**
- [ ] **P6.5.7 Manual fallback**

## 6.6 Board and committee output

- [ ] **P6.6.1 Live committee workspace**
- [ ] **P6.6.2 Point-in-time pack freeze**
- [ ] **P6.6.3 Statement-to-source lineage**
- [ ] **P6.6.4 Commentary and approval**
- [ ] **P6.6.5 Version comparison**
- [ ] **P6.6.6 Accessible PDF/export manifest**

## 6.7 UX validation

- [ ] executive comprehension test in under 60 seconds;
- [ ] evidence respondent completion test;
- [ ] keyboard-only journey;
- [ ] screen-reader journey;
- [ ] 125–200% zoom review;
- [ ] low-bandwidth and AI-degraded tests;
- [ ] enterprise laptop/GPU performance review.

## Phase 6 acceptance gate

A CRO must be able to:

- understand the payment-resilience item in seconds;
- inspect why it is material;
- see evidence weakness;
- compare options;
- approve within authority;
- and see the verification method without navigating multiple modules.

---

# Phase 7 — Confidential reporting, customer signal, and external portals

## Objective

Create secure, low-friction channels for staff, customers, vendors, and anonymous reporters while preserving protected identity and connecting reports to institutional risk.

## Dependencies

Phases 1, 2, 4, and 5.

## 7.1 External portal foundation

- [ ] **P7.1.1 Isolated portal application and deployment boundary**
- [ ] **P7.1.2 Anonymous and authenticated modes**
- [ ] **P7.1.3 Rate limiting and abuse controls**
- [ ] **P7.1.4 Secure upload and resumability**
- [ ] **P7.1.5 Multilingual content and accessibility**
- [ ] **P7.1.6 Low-bandwidth support**

## 7.2 Protected identity vault

- [ ] **P7.2.1 Separate identity store and encryption boundary**
- [ ] **P7.2.2 Pseudonymous case identity**
- [ ] **P7.2.3 Identity reveal policy and workflow**
- [ ] **P7.2.4 Conflict-aware access**
- [ ] **P7.2.5 Restricted backup, search, analytics, and logging**
- [ ] **P7.2.6 Immutable identity access events**
- [ ] **P7.2.7 Negative leakage tests**

## 7.3 Anonymous two-way communication

- [ ] **P7.3.1 Generate high-entropy case token**
- [ ] **P7.3.2 Store token verifier securely**
- [ ] **P7.3.3 Reporter inbox and investigator messages**
- [ ] **P7.3.4 Attachment exchange**
- [ ] **P7.3.5 Token-loss guidance without identity recovery promise**
- [ ] **P7.3.6 Notification options that preserve anonymity**

## 7.4 Intake design

- [ ] **P7.4.1 Explain anonymity and confidentiality boundaries**
- [ ] **P7.4.2 Guided, non-leading report structure**
- [ ] **P7.4.3 Ongoing and immediate-impact indicators**
- [ ] **P7.4.4 Voice and attachment support**
- [ ] **P7.4.5 Reporter review before submit**
- [ ] **P7.4.6 Original-language preservation and translation**

## 7.5 Triage and investigation

- [ ] **P7.5.1 Allegation versus verified fact model**
- [ ] **P7.5.2 Sensitivity and urgency routing**
- [ ] **P7.5.3 Conflict-of-interest routing**
- [ ] **P7.5.4 Duplicate and related-case detection**
- [ ] **P7.5.5 Chain of custody**
- [ ] **P7.5.6 Legal privilege and legal hold**
- [ ] **P7.5.7 Investigation actions and approvals**
- [ ] **P7.5.8 Anti-retaliation checkpoints**

## 7.6 Customer and vendor signal intake

- [ ] **P7.6.1 Customer-impact report type**
- [ ] **P7.6.2 Vendor concern/evidence type**
- [ ] **P7.6.3 Link to products, services, branches, vendors, controls, incidents, and risks**
- [ ] **P7.6.4 Pattern and concentration analysis**
- [ ] **P7.6.5 Integration with specialist complaint/case systems**
- [ ] **P7.6.6 Recurrence measurement after remediation**

## Phase 7 acceptance gate

Prove that:

- an anonymous reporter can communicate without identity;
- identity cannot leak through search, logs, summaries, exports, or graph traversal;
- an investigator with a conflict cannot access the case;
- AI summaries distinguish allegation from fact;
- and a validated report can create a material risk signal without exposing protected data.

---

# Phase 8 — Governed AI operator platform

## Objective

Provide a reusable, evaluated, policy-controlled operator runtime across evidence, risk, regulatory, remediation, assurance, and executive use cases.

## Dependencies

Phases 1–5. Some deterministic and limited AI processing may be prototyped earlier behind feature flags.

## 8.1 Model gateway

- [ ] **P8.1.1 Provider adapters**
- [ ] **P8.1.2 Model registry and allowlists**
- [ ] **P8.1.3 Classification- and residency-aware routing**
- [ ] **P8.1.4 Cost, latency, and context budgets**
- [ ] **P8.1.5 Fallback and kill switches**
- [ ] **P8.1.6 Invocation telemetry and retention policy**

## 8.2 Operator registry

- [ ] **P8.2.1 Versioned operator definition**
- [ ] **P8.2.2 Owner and approval lifecycle**
- [ ] **P8.2.3 Capability and action-class registry**
- [ ] **P8.2.4 Data classification limits**
- [ ] **P8.2.5 Evaluation-suite linkage**
- [ ] **P8.2.6 Deployment and rollback**

## 8.3 Tool registry and execution

- [ ] **P8.3.1 Versioned tool schemas**
- [ ] **P8.3.2 Domain-command tools only**
- [ ] **P8.3.3 External adapter tools**
- [ ] **P8.3.4 Idempotency and side-effect verification**
- [ ] **P8.3.5 Timeout, cancellation, retry, and compensation**
- [ ] **P8.3.6 Tool allowlist by operator and purpose**

## 8.4 Grounding and retrieval

- [ ] **P8.4.1 Source hierarchy**
- [ ] **P8.4.2 Authorization-aware search and graph retrieval**
- [ ] **P8.4.3 Source IDs, versions, excerpts, and time scope**
- [ ] **P8.4.4 Contradiction preservation**
- [ ] **P8.4.5 Protected-data exclusion and specialized routes**

## 8.5 Structured output and policy pipeline

- [ ] **P8.5.1 Versioned output schemas**
- [ ] **P8.5.2 Schema validation and fail-closed behavior**
- [ ] **P8.5.3 Domain validation**
- [ ] **P8.5.4 Confidence and abstention**
- [ ] **P8.5.5 Authorization and authority gates**
- [ ] **P8.5.6 Human review state**
- [ ] **P8.5.7 Immutable operator audit event**

## 8.6 Security

- [ ] **P8.6.1 Prompt-injection defenses**
- [ ] **P8.6.2 Secret isolation**
- [ ] **P8.6.3 No raw credentials in model context**
- [ ] **P8.6.4 Cross-tenant retrieval tests**
- [ ] **P8.6.5 Protected identity leakage tests**
- [ ] **P8.6.6 Malicious document test suite**
- [ ] **P8.6.7 Rate and budget abuse controls**

## 8.7 Evaluation harness

- [ ] **P8.7.1 Versioned datasets**
- [ ] **P8.7.2 Grounding and citation metrics**
- [ ] **P8.7.3 Domain accuracy metrics**
- [ ] **P8.7.4 Abstention metrics**
- [ ] **P8.7.5 Authorization and tool-use tests**
- [ ] **P8.7.6 Security adversarial tests**
- [ ] **P8.7.7 Latency and cost regression**
- [ ] **P8.7.8 Model/prompt comparison reports**
- [ ] **P8.7.9 Release gates and rollback**

## 8.8 Initial operators

Implement in this order:

- [ ] **P8.8.1 Evidence Operator**
- [ ] **P8.8.2 Executive Briefing Operator**
- [ ] **P8.8.3 Remediation Operator**
- [ ] **P8.8.4 Risk Intelligence Operator**
- [ ] **P8.8.5 Regulatory Operator**
- [ ] **P8.8.6 Assurance Operator**
- [ ] **P8.8.7 Resilience Operator**
- [ ] **P8.8.8 Third-Party Operator**

## Phase 8 acceptance gate

No operator reaches production unless:

- it has a distinct identity and purpose;
- tool access is allowlisted;
- output is structured and validated;
- material actions require appropriate human authority;
- evaluation thresholds pass;
- malicious-content tests pass;
- and the full invocation can be reconstructed from audit records.

---

# Phase 9 — Probo and enterprise integration fabric

## Objective

Integrate commodity compliance automation and institutional source systems without giving external tools authority over material ClearSight conclusions.

## Dependencies

Phases 1, 2, 4, 5, and 8 for AI-driven orchestration.

## 9.1 Integration SDK

- [ ] **P9.1.1 Connector identity and secrets contract**
- [ ] **P9.1.2 External object mapping**
- [ ] **P9.1.3 Cursor/version state**
- [ ] **P9.1.4 Idempotent ingest and write**
- [ ] **P9.1.5 Retry, replay, partial failure, and reconciliation**
- [ ] **P9.1.6 Source health and freshness**
- [ ] **P9.1.7 Data classification and purpose policy**
- [ ] **P9.1.8 Connector test harness**

## 9.2 Probo adapter

- [ ] **P9.2.1 Map organization and tenant safely**
- [ ] **P9.2.2 Map frameworks and controls**
- [ ] **P9.2.3 Map measures to control implementations**
- [ ] **P9.2.4 Map risks without making Probo authoritative for ClearSight materiality**
- [ ] **P9.2.5 Map vendors, assets, evidence, tasks, findings, documents, audits, obligations, and snapshots**
- [ ] **P9.2.6 Preserve source IDs and versions**
- [ ] **P9.2.7 Implement read synchronization**
- [ ] **P9.2.8 Implement approved task/evidence writes**
- [ ] **P9.2.9 Reconcile external completion with ClearSight verification**
- [ ] **P9.2.10 Handle deletion, revocation, and permission change**

## 9.3 MCP safety boundary

- [ ] **P9.3.1 Do not expose broad Probo bearer tokens to models**
- [ ] **P9.3.2 Proxy only approved tool schemas**
- [ ] **P9.3.3 Bind organization server-side**
- [ ] **P9.3.4 Restrict actions by operator and purpose**
- [ ] **P9.3.5 Audit every call and side effect**
- [ ] **P9.3.6 Verify returned object and state**

## 9.4 Priority institutional connectors

Implement based on first pilot bank:

- [ ] identity/IAM
- [ ] HR directory
- [ ] ITSM/change management
- [ ] collaboration/email
- [ ] document repository
- [ ] CMDB/service catalog
- [ ] vulnerability/security platform
- [ ] cloud inventory
- [ ] vendor/procurement system
- [ ] complaints or CRM
- [ ] incident platform
- [ ] data warehouse

## Phase 9 acceptance gate

Prove that:

- duplicate external events do not duplicate objects;
- external permission revocation propagates;
- a Probo task completion does not close ClearSight remediation;
- source provenance remains visible;
- and an operator cannot exceed its scoped integration authority.

---

# Phase 10 — Bank risk-domain expansion

## Objective

Expand from the core operating loop into bank-specific domain capabilities while reusing the same graph, evidence, decision, and verification mechanisms.

## Dependencies

Phases 2–9.

## 10.1 Operational resilience

- [ ] critical-operation inventory
- [ ] impact tolerances
- [ ] people/process/system/data/facility/vendor dependency mapping
- [ ] scenario library
- [ ] exercise planning
- [ ] test evidence capture
- [ ] recovery measurement
- [ ] concentration and single-point-of-failure analysis
- [ ] resilience decision and investment comparison
- [ ] board reporting

## 10.2 Third-party and concentration risk

- [ ] third- and fourth-party catalog
- [ ] service and data dependency mapping
- [ ] due diligence orchestration
- [ ] reusable vendor evidence
- [ ] contract and SLA obligations
- [ ] continuous internal/external signals
- [ ] concentration and substitutability
- [ ] stressed exit plans
- [ ] review based on risk change rather than calendar only

## 10.3 Cyber and technology risk

- [ ] asset/identity/data/threat/vulnerability relationships
- [ ] continuous control evidence
- [ ] exception and compensating-control workflows
- [ ] change and cloud risk
- [ ] material cyber-incident decision workflow
- [ ] investment option comparison
- [ ] CISO and board brief

## 10.4 Regulatory change and compliance

- [ ] authoritative source ingestion
- [ ] change detection
- [ ] obligation extraction and normalization
- [ ] applicability profile
- [ ] source-to-obligation lineage
- [ ] obligation-to-policy/control/service/entity mapping
- [ ] evidence-backed gap analysis
- [ ] implementation actions
- [ ] examiner-ready lineage

## 10.5 Incidents, losses, complaints, and near misses

- [ ] intake and classification
- [ ] timeline reconstruction
- [ ] root cause and contributing factors
- [ ] customer and financial impact
- [ ] control and risk linkage
- [ ] reportability decision
- [ ] remediation and verification
- [ ] recurrence analysis
- [ ] scenario recalibration

## 10.6 Model and AI risk

- [ ] model and AI-system inventory
- [ ] use case and deployment context
- [ ] data and vendor lineage
- [ ] criticality classification
- [ ] validation and approval
- [ ] bias, privacy, explainability, security, robustness, and misuse risk
- [ ] human oversight
- [ ] prompt/model/version tracking
- [ ] monitoring, incidents, and exceptions
- [ ] operator and external-agent governance

## Phase 10 acceptance gate

Each domain module must:

- use shared canonical entities;
- produce and consume graph relationships;
- use Living Evidence Fabric rather than isolated uploads;
- use Decision Ledger for material choices;
- use verification contracts;
- and preserve domain-specific authority.

---

# Phase 11 — Assurance, audit, examination, and reporting

## Objective

Provide independent assurance and examiner-grade reconstruction without creating a separate truth system.

## Dependencies

Core platform complete; selected domain modules available.

## 11.1 Control and assurance testing

- [ ] test-plan model
- [ ] population and sample definition
- [ ] sample-selection provenance
- [ ] design and operating-effectiveness conclusions
- [ ] independent reviewer role
- [ ] evidence sufficiency challenge
- [ ] exception and finding creation
- [ ] continuous assurance scheduling

## 11.2 Audit

- [ ] audit universe
- [ ] risk-based planning
- [ ] engagement scope
- [ ] information requests
- [ ] workpapers or integration boundary
- [ ] findings and management response
- [ ] issue linkage
- [ ] audit independence controls

## 11.3 Evidence rooms

- [ ] scoped collection
- [ ] point-in-time freeze
- [ ] access requests
- [ ] NDA or approval boundary where needed
- [ ] watermarking
- [ ] download manifest
- [ ] revocation
- [ ] examiner Q&A lineage

## 11.4 Regulatory and board packages

- [ ] statement-to-source mapping
- [ ] versioned narrative
- [ ] approval and sign-off
- [ ] included/excluded evidence manifest
- [ ] AI involvement record
- [ ] accessible export
- [ ] cryptographic package manifest where required

## Phase 11 acceptance gate

An authorized examiner must be able to trace a statement from report to decision, conclusion, claim, evidence version, obligation, control, and original source without exposing unauthorized adjacent data.

---

# Phase 12 — Scale, resilience, security certification, and general availability

## Objective

Validate the product under enterprise load, deployment constraints, attack scenarios, and operational failure.

## Dependencies

Pilot-complete product.

## 12.1 Performance and scale

- [ ] define pilot and target workload models
- [ ] signal-ingestion load tests
- [ ] graph-query benchmarks
- [ ] executive brief latency tests
- [ ] evidence upload and processing throughput
- [ ] search and vector retrieval benchmarks
- [ ] event backlog recovery
- [ ] workflow concurrency
- [ ] export scale
- [ ] cost-per-tenant analysis

## 12.2 Reliability

- [ ] service SLOs
- [ ] database backup and point-in-time restore
- [ ] object-store recovery
- [ ] message replay
- [ ] workflow recovery
- [ ] AI provider outage
- [ ] integration outage
- [ ] regional failure plan
- [ ] disaster recovery exercise

## 12.3 Security

- [ ] independent penetration test
- [ ] tenant-isolation assessment
- [ ] protected identity assessment
- [ ] supply-chain security
- [ ] secret and key rotation
- [ ] threat-model review
- [ ] secure configuration baselines
- [ ] incident response runbooks
- [ ] vulnerability-management process

## 12.4 Privacy and governance

- [ ] retention and deletion tests
- [ ] legal-hold tests
- [ ] data residency tests
- [ ] data-subject or equivalent privacy workflows where applicable
- [ ] model-provider contract controls
- [ ] AI inventory and impact assessment
- [ ] operator governance committee process

## 12.5 Deployment modes

- [ ] multi-tenant SaaS
- [ ] dedicated tenant
- [ ] private cloud
- [ ] on-premises packaging
- [ ] hybrid evidence plane
- [ ] customer-managed keys
- [ ] region-specific model routes

## 12.6 Migration and onboarding

- [ ] source inventory
- [ ] mapping templates
- [ ] data quality and reconciliation
- [ ] historical evidence import
- [ ] relationship reconstruction
- [ ] user and authority import
- [ ] parallel-run strategy
- [ ] acceptance and rollback

## 12.7 Operational readiness

- [ ] runbooks
- [ ] support escalation
- [ ] tenant admin tooling
- [ ] audit support
- [ ] status communication
- [ ] release and rollback process
- [ ] feature flags
- [ ] product analytics and privacy controls

## Phase 12 acceptance gate

General availability requires:

- all critical golden journeys passing;
- no unresolved critical tenant-isolation or protected-identity defect;
- tested backup and recovery;
- model and integration degraded-mode operation;
- approved SLOs and runbooks;
- and pilot evidence that the product reduces effort while strengthening evidence and decisions.

---

# 4. Suggested first vertical release

A practical first release should not attempt every domain.

## Scope

### Institutional context

- one legal entity;
- one critical payment service;
- supporting processes, systems, vendor, and owners;
- selected appetite and impact tolerance.

### Signals

- service health;
- vendor state;
- recent change;
- evidence expiry;
- and one audit finding.

### Living Evidence Fabric

- IAM and approval evidence;
- targeted manager micro-request;
- sufficiency dimensions;
- contradiction;
- and evidence debt.

### Decision Ledger

- one treatment decision;
- authority check;
- option comparison;
- action plan;
- and verification contract.

### Execution

- one external task adapter;
- implementation evidence;
- outcome evidence;
- failed and successful verification paths.

### Experience

- Today brief;
- Explain workspace;
- decision card;
- evidence request;
- verification view;
- and point-in-time history.

### Protected reporting

- anonymous report;
- secure token;
- identity-isolated case;
- and risk-signal escalation.

## Why this slice

It proves the product’s moat:

- connected institutional context;
- materiality;
- dynamic evidence;
- minimum human effort;
- authority;
- and verified risk handling.

---

# 5. Work that must not be pulled forward prematurely

Avoid early expansion into:

- dozens of frameworks;
- broad report catalogs;
- generic form builders;
- extensive dashboard customization;
- a marketplace;
- dedicated graph infrastructure without benchmarks;
- many microservices;
- autonomous material actions;
- or full replacement of specialist systems.

These can consume delivery capacity while leaving the distinctive operating loop incomplete.

---

# 6. Program-level acceptance metrics

The implementation program should track:

## Human effort

- median active time per accepted evidence request;
- number of questions per request;
- duplicate requests avoided;
- manual evidence handling eliminated;
- and redirected requests.

## Materiality

- material items later dismissed as noise;
- material items detected late;
- average number of executive items;
- and time from signal to authority.

## Evidence

- material claims with sufficient evidence;
- unresolved contradictions;
- evidence debt;
- stale evidence;
- and conclusion reversals.

## Decision and outcome

- time to decision;
- decisions returned for evidence;
- expired decisions;
- verification failure;
- reopened issues;
- and projected-versus-observed treatment effect.

## AI trust

- source-lineage completeness;
- unsupported assertion rate;
- abstention quality;
- human edit/reject rate;
- unauthorized-action attempts;
- protected-data leakage tests;
- latency;
- and cost.

## Experience

- executive comprehension time;
- task completion rate;
- accessibility defects;
- visual-regression defects;
- and performance-budget adherence.

---

# 7. Completion standard

The product is not complete because every planned screen or module exists.

A milestone is complete only when:

- the real domain path is exercised;
- product invariants hold;
- authority and privacy are enforced;
- evidence and decision history are reconstructable;
- AI behavior is evaluated and governed;
- failure and degraded modes work;
- visual and accessibility standards pass;
- and the resulting action can be verified against an institutional outcome.

The final test is straightforward:

> Can ClearSight identify a material bank risk, explain why it matters, gather only the missing evidence, route the right human decision, execute safely, and prove whether the response worked—without forcing users to operate a conventional GRC system?

Until the answer is yes, the core product is not finished.