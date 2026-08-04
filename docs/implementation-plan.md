# ClearSight Implementation Plan

This plan turns the ClearSight situation-first product model into a bank-grade implementation without requiring perfect integrations, exposing internal architecture as UI, or recreating every mature GRC module before the core product works.

It conforms to:

- [`../README.md`](../README.md)
- [`product/operating-model.md`](product/operating-model.md)
- [`product/experience-principles.md`](product/experience-principles.md)
- [`product/differentiation.md`](product/differentiation.md)
- [`../AGENTS.md`](../AGENTS.md)

Checkboxes indicate planned work, not completed implementation.

---

# 1. Delivery thesis

The first product must prove a narrow but complete loop:

```text
Observation
→ bounded banking Risk Situation
→ precise Claims and Evidence Recipes
→ existing evidence search
→ focused capture of missing facts
→ evidence-backed Conclusion
→ authorized Decision
→ Action
→ Verification of defined outcome criteria
→ historical reconstruction
```

The first implementation must not attempt an institution-wide graph, all risk domains, all regulatory content, all deployment modes, or autonomous material judgment.

---

# 2. Delivery principles

## 2.1 Situations are the vertical slice

Each milestone must make at least one recognizable banking situation more understandable or actionable.

Avoid milestones that produce only schemas, generic forms, dashboards, or isolated AI demonstrations.

## 2.2 Source trust before broad intelligence

Materiality and executive compression are only credible when source authority, freshness, mapping, population, and contradiction are visible.

The Source Registry, Observation contract, import reconciliation, and source-health model are early product features—not integration plumbing deferred until later.

## 2.3 Progressive integration

Support:

- forms and controlled values;
- photos and scans;
- spreadsheets and CSV;
- managed recurring imports;
- APIs;
- and events.

All methods produce the same Observation model.

## 2.4 Correct interaction form

Use:

- cards for a small attention queue;
- tables and worklists for populations;
- step flows for capture and imports;
- comparison views for contradiction and reconciliation;
- paths for dependency;
- timelines for history;
- and charts for specific questions.

Do not use cards or node graphs as universal components.

## 2.5 Begin as a coherent modular core

Start with a modular monolith or similarly disciplined core with:

- explicit bounded contexts;
- transactional integrity;
- outbox and durable jobs;
- authoritative relational data;
- versioned object storage;
- rebuildable search and graph projections;
- and replaceable adapters.

Microservices and a dedicated graph engine require measured justification.

## 2.6 AI compiles; policy decides

AI may extract, normalize, suggest, compare, summarize, and draft.

Deterministic domain services enforce:

- source policy;
- evidence minimums;
- authority;
- authorization;
- state transitions;
- and closure.

## 2.7 Protected reporting is an isolated track

Protected reporting must not be casually mixed into ordinary graph, search, analytics, or AI routes.

The first release may support protected intake and sanitized escalation without becoming a full investigation platform.

---

# 3. Initial product wedge

The initial pilot should cover:

## Institution boundary

- one bank;
- one legal entity;
- selected regions or branches;
- one or two channels;
- selected authority roles.

## Primary situation A — ATM inventory and physical/operational confirmation

- known ATM population;
- unresolved device-location-owner relationships;
- switch heartbeat or operational state;
- targeted branch photo and structured confirmation;
- source contradiction;
- remediation and verification.

## Primary situation B — POS terminal identity or settlement reconciliation

Choose one based on pilot data availability:

- terminal-to-merchant identity and location reconciliation; or
- transaction-to-settlement population reconciliation.

## Common capabilities proved

- Source Registry;
- spreadsheet and CSV ingestion;
- at least one API or scheduled source;
- photo or structured human capture;
- population worklist;
- matching and contradiction;
- Risk Situation workspace;
- Claim and Evidence Recipe;
- Decision and authority;
- Action and Verification Contract;
- Today brief;
- point-in-time history.

Protected reporting may be developed in parallel but must not block this wedge.

---

# 4. Recommended repository topology

```text
.
├── README.md
├── AGENTS.md
├── docs/
│   ├── product/
│   ├── architecture/
│   ├── quality/
│   ├── reviews/
│   └── decisions/
├── apps/
│   ├── web/
│   ├── external-portal/
│   └── capture-pwa/
├── services/
│   └── core/
├── packages/
│   ├── design-system/
│   ├── domain-contracts/
│   ├── event-contracts/
│   ├── authorization/
│   ├── model-gateway/
│   └── integration-sdk/
├── workers/
│   ├── ingestion/
│   ├── media-processing/
│   └── projections/
├── tests/
│   ├── e2e/
│   ├── evaluations/
│   ├── security/
│   ├── visual/
│   └── performance/
└── infrastructure/
```

The first implementation may keep modules and workers in one deployable unit while preserving boundaries in code.

---

# Phase 0 — Product semantics, pilot, architecture, and design foundation

## Objective

Establish the canonical operating model, first pilot situations, source inventory, design language, threat model, and implementation decisions before feature development.

## 0.1 Pilot definition

- [ ] Select pilot bank and legal entity.
- [ ] Select ATM and POS or settlement situation.
- [ ] Define participating regions, branches, merchants, or populations.
- [ ] Define pilot personas and authority.
- [ ] Inventory available spreadsheets, databases, APIs, documents, and human sources.
- [ ] Identify source owners and data-quality limitations.
- [ ] Define measurable pilot outcomes.

## 0.2 Canonical language

- [ ] Define Scope.
- [ ] Define Exposure Pattern.
- [ ] Define Risk Situation.
- [ ] Define Claim and Evidence Recipe.
- [ ] Define Observation.
- [ ] Define Conclusion.
- [ ] Define Decision and Verification Contract.
- [ ] Define prohibited semantic collapses.

Prohibited collapses include:

- upload is not evidence sufficiency;
- integration success is not data truth;
- observation is not verified fact;
- signal is not incident;
- action completion is not outcome;
- source age is not risk severity;
- AI confidence is not evidence strength.

## 0.3 First exposure and channel packs

- [ ] Define universal exposure families.
- [ ] Define ATM channel pack.
- [ ] Define selected POS or settlement channel pack.
- [ ] Define initial claims and evidence recipes.
- [ ] Define initial capture templates.
- [ ] Define initial verification patterns.

## 0.4 Architecture decision records

Create ADRs for:

- [ ] modular core and split criteria;
- [ ] backend and frontend stack;
- [ ] relational authoritative model;
- [ ] temporal/versioning strategy;
- [ ] outbox and durable processing;
- [ ] workflow runtime;
- [ ] object storage and evidence integrity;
- [ ] authorization engine;
- [ ] search and graph projections;
- [ ] model gateway;
- [ ] offline capture boundary;
- [ ] protected reporting isolation;
- [ ] initial deployment mode.

## 0.5 Threat and privacy model

- [ ] cross-tenant and cross-entity access;
- [ ] wrong-scope approval;
- [ ] evidence tampering;
- [ ] malicious media and spreadsheets;
- [ ] formula and file-content attacks;
- [ ] prompt injection;
- [ ] export leakage;
- [ ] insider misuse;
- [ ] protected identity exposure;
- [ ] mobile and offline evidence risk;
- [ ] integration compromise;
- [ ] graph, search, count, and timing inference.

## 0.6 Design foundation

- [ ] semantic tokens;
- [ ] typography and numeric styles;
- [ ] comfortable and compact density;
- [ ] light and dark parity;
- [ ] scope/context header;
- [ ] Today, Situation, Capture, Explore, Configure shell;
- [ ] situation card;
- [ ] population worklist;
- [ ] spreadsheet mapper;
- [ ] photo-capture concept;
- [ ] source profile;
- [ ] contradiction and reconciliation view;
- [ ] verification states;
- [ ] accessibility baseline.

## 0.7 Repository and CI

- [ ] scaffold repository;
- [ ] formatting, linting, type checking;
- [ ] unit, integration, E2E, accessibility, and visual test runners;
- [ ] dependency and secret scanning;
- [ ] SBOM and artifact-signing plan;
- [ ] preview environments;
- [ ] database migration verification;
- [ ] pull-request template enforcing AGENTS review.

## Phase 0 acceptance gate

Do not begin domain implementation until:

- the operating objects are distinct;
- two pilot situations are fully specified;
- source owners and limitations are known;
- first Evidence Recipes are testable;
- the task-oriented interface is prototyped;
- population and capture workflows are represented;
- protected reporting boundaries are approved;
- and core stack choices have ADRs.

---

# Phase 1 — Trust, tenancy, identity, temporal history, audit, and storage

## Objective

Build the security and historical foundation required by every later capability.

## 1.1 Tenant and institutional scope

- [ ] tenant model;
- [ ] legal entities and jurisdictions;
- [ ] regions, business units, channels, services, branches, and populations;
- [ ] immutable context propagation;
- [ ] server-derived tenant and entity scope;
- [ ] scope-switch authorization;
- [ ] cross-store isolation tests.

## 1.2 Identity and access

- [ ] development identity provider;
- [ ] OIDC/SAML boundary;
- [ ] MFA and session hooks;
- [ ] SCIM or directory provisioning boundary;
- [ ] service identities;
- [ ] break-glass flow;
- [ ] delegated authority;
- [ ] authentication audit.

## 1.3 Authorization

- [ ] deny-by-default authorization service;
- [ ] RBAC plus attributes, relationships, purpose, and sensitivity;
- [ ] authority matrix;
- [ ] segregation of duties and conflict checks;
- [ ] field-, evidence-, source-, and relationship-level restrictions;
- [ ] authorization-aware search and projection contract;
- [ ] bulk-operation authorization;
- [ ] explainable administration decisions;
- [ ] exhaustive negative tests.

## 1.4 Temporal and immutable records

- [ ] valid time and record time metadata;
- [ ] version and supersession;
- [ ] optimistic concurrency;
- [ ] append-only material records;
- [ ] point-in-time query primitives;
- [ ] temporal consistency tests.

## 1.5 Audit ledger

- [ ] immutable audit event schema;
- [ ] operational-log separation;
- [ ] actor, purpose, scope, command, result, and correlation;
- [ ] sensitive-payload references;
- [ ] strict audit-query authorization;
- [ ] tamper-evidence strategy;
- [ ] action reconstruction.

## 1.6 Events and durable jobs

- [ ] transactional outbox;
- [ ] versioned event envelope;
- [ ] idempotent consumers;
- [ ] retry, dead letter, replay, and cancellation;
- [ ] correlation and causation;
- [ ] job progress and recovery.

## 1.7 Evidence storage

- [ ] versioned object-store abstraction;
- [ ] content hashes and integrity manifest;
- [ ] malware and content scanning;
- [ ] classification-aware encryption;
- [ ] retention, legal hold, and deletion hooks;
- [ ] resumable upload;
- [ ] protected-data boundary.

## Phase 1 acceptance gate

Pass:

- tenant and entity negative tests;
- wrong-scope action tests;
- point-in-time reconstruction;
- duplicate-event idempotency;
- unauthorized search, count, export, and bulk-action tests;
- evidence integrity;
- and service-identity audit.

---

# Phase 2 — Source Registry, Observation contract, and progressive ingestion

## Objective

Make source authority, mapping, freshness, and progressive data capture first-class before building broad risk intelligence.

## 2.1 Source Registry

- [ ] source profile aggregate;
- [ ] owner and custodian;
- [ ] source type and collection method;
- [ ] authoritative fields;
- [ ] explicit limitations;
- [ ] scope and identifiers;
- [ ] expected freshness;
- [ ] health state;
- [ ] mapping version;
- [ ] known limitations;
- [ ] purpose and access policy;
- [ ] affected-object and conclusion references.

## 2.2 Observation contract

- [ ] subject and fact schema;
- [ ] source and capture method;
- [ ] scope and population;
- [ ] effective and capture time;
- [ ] original artifact or source reference;
- [ ] transformation history;
- [ ] source authority and limitation;
- [ ] sensitivity;
- [ ] confidence and confirmation state;
- [ ] version and provenance.

## 2.3 Spreadsheet and CSV ingestion

- [ ] secure upload;
- [ ] sheet selection;
- [ ] column detection;
- [ ] reusable mapping templates;
- [ ] data-type and required-field validation;
- [ ] sample preview;
- [ ] formula and malicious-content controls;
- [ ] identifier normalization;
- [ ] duplicate detection;
- [ ] partial acceptance;
- [ ] row-level provenance;
- [ ] reconciliation report;
- [ ] rollback reference.

## 2.4 Structured capture

- [ ] forms generated from unresolved facts;
- [ ] controlled values;
- [ ] scoped searchable dropdowns;
- [ ] prefilled read-only known values;
- [ ] redirect and not-applicable state;
- [ ] partial answer;
- [ ] final assertion review.

## 2.5 Media capture

- [ ] mobile camera and document scan;
- [ ] framing and quality guidance;
- [ ] blur, glare, crop, and readability checks;
- [ ] metadata and location notice;
- [ ] redaction or retake where permitted;
- [ ] original preservation;
- [ ] extraction region and model lineage;
- [ ] user confirmation;
- [ ] bounded visible-attribute claims;
- [ ] low-bandwidth retry;
- [ ] offline queue decision gate.

## 2.6 Managed imports and APIs

- [ ] scheduled file ingestion;
- [ ] SFTP or approved managed transfer;
- [ ] database export ingestion;
- [ ] generic API connector boundary;
- [ ] event envelope boundary;
- [ ] cursors, versions, idempotency, deletion, and revocation;
- [ ] source freshness and health updates.

## Phase 2 acceptance gate

Demonstrate:

- one ATM spreadsheet import with partial and unresolved rows;
- one managed or API source;
- one structured human confirmation;
- one photo extraction with correction;
- complete source profiles;
- row- and artifact-level provenance;
- and stale-source impact without presenting stale data as current.

---

# Phase 3 — Scope model, Exposure Patterns, channel packs, populations, and Risk Situations

## Objective

Create the minimal connected institutional context needed to present real bank situations without attempting a universal institution graph.

## 3.1 Scope hierarchy

- [ ] institution, entity, jurisdiction, region, channel, service, branch, merchant group, vendor, and population;
- [ ] typed versioned relationships;
- [ ] ownership and accountable roles;
- [ ] scope aliases and external identifiers;
- [ ] point-in-time scope reconstruction;
- [ ] scope-aware authorization.

## 3.2 Exposure Pattern library

- [ ] canonical exposure families;
- [ ] causes and consequences;
- [ ] common indicators;
- [ ] common claims;
- [ ] likely sources;
- [ ] controls and obligations;
- [ ] decision thresholds;
- [ ] verification patterns;
- [ ] versions and institution overrides.

## 3.3 Channel packs

- [ ] ATM pack;
- [ ] selected POS or settlement pack;
- [ ] capture templates;
- [ ] initial source mappings;
- [ ] Evidence Recipes;
- [ ] situation presentation language;
- [ ] default visual summaries.

## 3.4 Population and matching

- [ ] population definition and snapshot;
- [ ] denominator and exclusions;
- [ ] exact and alias matching;
- [ ] provisional matches;
- [ ] unresolved and contradictory states;
- [ ] merge and unmerge;
- [ ] human reconciliation;
- [ ] downstream-impact view;
- [ ] evaluation dataset.

## 3.5 Risk Situation aggregate

- [ ] scope and period;
- [ ] exposure patterns;
- [ ] source observations;
- [ ] affected objects;
- [ ] change and why-now;
- [ ] materiality dimensions;
- [ ] claims and evidence state;
- [ ] required handling and authority;
- [ ] action and verification state;
- [ ] grouping and supersession;
- [ ] history.

## 3.6 Initial situation creation

- [ ] deterministic situation rules for ATM;
- [ ] deterministic situation rules for selected POS/settlement scenario;
- [ ] human create or escalate path;
- [ ] preserve source observations;
- [ ] avoid duplicate situations;
- [ ] dismiss, merge, split, and reopen with reason.

## Phase 3 acceptance gate

Demonstrate:

- ATM population with mapped, unresolved, stale, and contradictory records;
- one bounded ATM Risk Situation;
- one POS or settlement Risk Situation;
- reusable exposure patterns;
- visible source and scope context;
- no requirement for a dedicated graph database;
- and point-in-time relationship reconstruction.

---

# Phase 4 — Claims, Evidence Recipes, evidence requests, reconciliation, and conclusions

## Objective

Implement the complete minimum-question evidence loop on top of the source and situation foundation.

## 4.1 Claims and recipes

- [ ] versioned Claim;
- [ ] Evidence Recipe;
- [ ] required facts;
- [ ] acceptable source types;
- [ ] source-authority limits;
- [ ] freshness and coverage;
- [ ] independence and contradiction rules;
- [ ] materiality-sensitive policy;
- [ ] review and approval requirements.

## 4.2 Observation-to-claim evaluation

- [ ] supports;
- [ ] partially supports;
- [ ] contradicts;
- [ ] limits;
- [ ] duplicates;
- [ ] supersedes;
- [ ] irrelevant;
- [ ] unverifiable;
- [ ] pending review.

## 4.3 Sufficiency

- [ ] relevance;
- [ ] authenticity;
- [ ] coverage;
- [ ] freshness;
- [ ] independence;
- [ ] completeness;
- [ ] consistency;
- [ ] reliability;
- [ ] traceability;
- [ ] explainable policy result;
- [ ] override and challenge.

## 4.4 Contradiction and data-quality debt

- [ ] contradiction records;
- [ ] identity, temporal, scope, and factual conflict;
- [ ] effect on conclusions and situations;
- [ ] resolver workflow;
- [ ] evidence debt;
- [ ] data-quality debt;
- [ ] source-degradation propagation.

## 4.5 Best-source and minimum-question resolver

- [ ] search existing authorized observations;
- [ ] candidate-source registry;
- [ ] rank authority, directness, independence, freshness, coverage, burden, sensitivity, conflict, and response time;
- [ ] preserve rationale;
- [ ] human correction;
- [ ] generate minimum structured question;
- [ ] prefill context;
- [ ] select channel;
- [ ] estimate effort;
- [ ] deduplicate and cancel overlapping requests.

## 4.6 Evidence request lifecycle

- [ ] drafted;
- [ ] policy checked;
- [ ] delivered;
- [ ] viewed;
- [ ] partially answered;
- [ ] redirected;
- [ ] not applicable;
- [ ] validating;
- [ ] follow-up required;
- [ ] sufficient;
- [ ] declined;
- [ ] expired;
- [ ] cancelled.

## 4.7 Conclusion

- [ ] supported, partial, unsupported, contradicted, indeterminate, expired, not applicable;
- [ ] included and excluded evidence;
- [ ] assumptions;
- [ ] authority;
- [ ] valid period;
- [ ] supersession;
- [ ] invalidation and reopening.

## Phase 4 acceptance gate

Pass ATM and POS journeys where:

- existing evidence is searched first;
- only unresolved records are requested;
- photo, spreadsheet, API, and human observations remain distinct;
- source limitations are visible;
- contradiction blocks or qualifies conclusion;
- and no upload or response alone produces sufficiency.

---

# Phase 5 — Decisions, authority, external action, and verification

## Objective

Turn evidence-backed situations into authorized decisions, executable work, and observable outcomes.

## 5.1 Decision aggregate

- [ ] context and scope;
- [ ] conclusion and evidence;
- [ ] uncertainty and contradiction;
- [ ] options;
- [ ] expected effects and limitations;
- [ ] cost, dependencies, timing, reversibility, and customer impact;
- [ ] selection and rationale;
- [ ] authority and segregation of duties;
- [ ] dissent and override;
- [ ] conditions, expiry, and review triggers;
- [ ] supersession.

## 5.2 Approval and challenge

- [ ] sequential and parallel approval;
- [ ] challenge and request-more-evidence;
- [ ] conditional approval;
- [ ] rejection;
- [ ] delegation;
- [ ] emergency authority with later review;
- [ ] context-change invalidation;
- [ ] action-time authority re-evaluation.

## 5.3 Action plan

- [ ] owner and accountable role;
- [ ] tasks and dependencies;
- [ ] due dates and escalation;
- [ ] implementation evidence;
- [ ] generic external-task adapter;
- [ ] idempotent writes and reconciliation;
- [ ] partial failure and compensation;
- [ ] state separate from verification.

## 5.4 Verification Contract

- [ ] observable outcome;
- [ ] baseline;
- [ ] population and scope;
- [ ] measurement source;
- [ ] success and failure threshold;
- [ ] observation period;
- [ ] required evidence;
- [ ] acceptance authority;
- [ ] failure response.

## 5.5 Verification runtime

- [ ] begin after valid implementation evidence;
- [ ] collect outcome observations;
- [ ] evaluate contract;
- [ ] route ambiguous result;
- [ ] verified effective, ineffective, indeterminate, or incomplete;
- [ ] reopen or reclassify situation;
- [ ] preserve projected-versus-observed result;
- [ ] avoid causal overclaim.

## Phase 5 acceptance gate

Pass scenarios where:

- an unauthorized owner cannot approve;
- context change expires a decision;
- an external ticket can complete while verification remains open;
- failed outcome evidence reopens the situation;
- green appears only after accepted verification;
- and the full decision is reconstructable.

---

# Phase 6 — Situation-first product experience

## Objective

Deliver the calm, direct interface for executives, specialists, operational users, and evidence respondents.

## 6.1 Design system

- [ ] tokens and themes;
- [ ] comfortable and compact density;
- [ ] typography and numeric formats;
- [ ] accessibility primitives;
- [ ] protected treatment;
- [ ] state system;
- [ ] Storybook or equivalent;
- [ ] visual regression.

## 6.2 Application shell

- [ ] Today;
- [ ] Situation;
- [ ] Capture;
- [ ] Explore;
- [ ] Configure;
- [ ] role simplification;
- [ ] scope/context header;
- [ ] keyboard navigation;
- [ ] global command surface without chat dominance.

## 6.3 Today

- [ ] role-aware situation ranking;
- [ ] three-to-seven default items;
- [ ] evidence, decision, action, verification, and deadline states;
- [ ] no-material-change state;
- [ ] expanded analyst mode;
- [ ] acknowledgement and delegation;
- [ ] notification grouping.

## 6.4 Situation workspace

- [ ] Summary;
- [ ] Evidence;
- [ ] Decision;
- [ ] Action;
- [ ] Outcome;
- [ ] History;
- [ ] relationship path;
- [ ] point-in-time control;
- [ ] accessible textual equivalents.

## 6.5 Operational work surfaces

- [ ] population worklist;
- [ ] reconciliation compare view;
- [ ] bulk action review;
- [ ] spreadsheet mapper;
- [ ] import summary;
- [ ] Source Profile;
- [ ] source degraded state;
- [ ] photo capture and extraction review;
- [ ] controlled-value form.

## 6.6 Responsive and degraded modes

- [ ] tablet executive and meeting use;
- [ ] mobile capture;
- [ ] low bandwidth;
- [ ] resumable upload;
- [ ] offline queue where approved;
- [ ] AI unavailable;
- [ ] source unavailable;
- [ ] 200% zoom;
- [ ] multilingual expansion.

## Phase 6 acceptance gate

Users must demonstrate:

- executive comprehension in under 60 seconds;
- branch evidence completion with no GRC-code knowledge;
- ATM and POS population reconciliation;
- safe bulk action;
- correct scope before approval;
- full keyboard operation;
- and clear implementation-versus-verification state.

---

# Phase 7 — Materiality, AI compiler, and natural-language inquiry

## Objective

Add broader intelligence only after source trust, situations, evidence, decision, and verification work deterministically.

## 7.1 Materiality service

- [ ] contextual enrichment;
- [ ] appetite and tolerance;
- [ ] affected customers and critical services;
- [ ] evidence and data-quality debt;
- [ ] velocity and time sensitivity;
- [ ] concentration and dependency;
- [ ] grouping with alternatives;
- [ ] decision relevance;
- [ ] structured explanation;
- [ ] human override;
- [ ] false-positive and missed-item feedback.

## 7.2 Model gateway

- [ ] provider adapters;
- [ ] model registry and allowlists;
- [ ] classification- and residency-aware routing;
- [ ] cost, latency, and context budgets;
- [ ] fallback and kill switch;
- [ ] invocation telemetry.

## 7.3 AI compiler capabilities

Implement in this order:

- [ ] spreadsheet and document extraction;
- [ ] photo and scan extraction;
- [ ] normalization and candidate mapping;
- [ ] duplicate and contradiction suggestion;
- [ ] focused request drafting;
- [ ] situation explanation;
- [ ] executive summarization;
- [ ] option and verification drafting;
- [ ] broader risk intelligence only after evaluation.

## 7.4 Operator governance

- [ ] operator identity and purpose;
- [ ] tool registry and action classes;
- [ ] structured output and validation;
- [ ] grounding and source versions;
- [ ] authorization and approval;
- [ ] prompt-injection defense;
- [ ] evaluation datasets;
- [ ] abstention;
- [ ] release gates and rollback.

## 7.5 Natural-language inquiry

- [ ] scope-aware query planning;
- [ ] authorized retrieval;
- [ ] source and period display;
- [ ] contradiction and missing-information display;
- [ ] transition to structured action;
- [ ] manual fallback.

## Phase 7 acceptance gate

No AI capability reaches production unless:

- structured output validates;
- explicit, extracted, inferred, confirmed, and approved values remain distinct;
- source lineage is complete;
- protected and cross-tenant leakage tests pass;
- abstention works;
- material action remains human-governed;
- and deterministic manual operation remains available.

---

# Phase 8 — Protected reporting and external signal intake

## Objective

Create isolated, secure, low-friction reporting without making the main product a full investigation platform.

## 8.1 External portal

- [ ] isolated deployment boundary;
- [ ] anonymous and identified modes;
- [ ] multilingual and accessible intake;
- [ ] low-bandwidth support;
- [ ] rate limiting and abuse controls;
- [ ] secure upload.

## 8.2 Protected case and identity

- [ ] protected case-content boundary;
- [ ] separate identity vault;
- [ ] pseudonymous case identity;
- [ ] reveal policy and workflow;
- [ ] conflict-aware access;
- [ ] restricted search, analytics, logs, backup, and export;
- [ ] immutable privileged access events.

## 8.3 Anonymous communication

- [ ] high-entropy case token;
- [ ] secure verifier;
- [ ] reporter inbox;
- [ ] investigator message;
- [ ] attachment exchange;
- [ ] anonymity-preserving notification options;
- [ ] token-loss guidance.

## 8.4 Sanitized risk escalation

- [ ] allegation-versus-fact model;
- [ ] investigator validation;
- [ ] minimized risk signal;
- [ ] no identity or identifying metadata;
- [ ] approved link to Risk Situation;
- [ ] chain of custody;
- [ ] AI protected route.

## Phase 8 acceptance gate

Prove that:

- identity and protected content cannot leak through ordinary search, counts, logs, export, AI, or graph traversal;
- a conflicted investigator cannot access the case;
- anonymous communication works;
- AI does not profile credibility;
- and only sanitized validated context enters ordinary Risk Situations.

---

# Phase 9 — Enterprise integration and execution fabric

## Objective

Expand connectors and execution engines without allowing external systems to become authoritative for ClearSight conclusions.

## 9.1 Integration SDK

- [ ] connector identity and secrets;
- [ ] source profile binding;
- [ ] external object mapping;
- [ ] cursor and version;
- [ ] idempotent ingest and write;
- [ ] retry, replay, partial failure, and reconciliation;
- [ ] source health and freshness;
- [ ] deletion and revocation;
- [ ] connector test harness.

## 9.2 Priority connectors

Choose by pilot need:

- [ ] IAM;
- [ ] HR;
- [ ] ITSM/change;
- [ ] switch or channel monitoring;
- [ ] asset or CMDB;
- [ ] document repository;
- [ ] vendor/procurement;
- [ ] complaints or CRM;
- [ ] incident platform;
- [ ] data warehouse.

## 9.3 Probo or execution-engine adapter

- [ ] server-controlled tenant mapping;
- [ ] selected control, task, evidence, finding, vendor, and audit mappings;
- [ ] source IDs and versions;
- [ ] scoped writes;
- [ ] no broad token exposure to models;
- [ ] completion reconciled as implementation state;
- [ ] final conclusion remains in ClearSight.

## Phase 9 acceptance gate

- duplicate events produce no duplicate objects;
- permission revocation propagates;
- partial failure remains visible;
- external completion cannot close a situation;
- source authority and version remain visible;
- and operators cannot exceed integration scope.

---

# Phase 10 — Domain and assurance expansion

## Objective

Expand through reusable channel packs, exposure patterns, claims, sources, decisions, and verification—not disconnected modules.

Candidate expansions:

- operational resilience;
- third-party and concentration risk;
- cyber and technology risk;
- regulatory change and compliance;
- incidents, losses, complaints, and near misses;
- policy and control assurance;
- model and AI risk;
- audit and examination;
- board and regulatory packages.

Each expansion MUST:

- reuse Scope, Exposure Pattern, Risk Situation, Claim, Evidence Recipe, Observation, Decision, and Verification;
- declare source authority;
- define population and matching;
- use the Situation workspace;
- include degraded and point-in-time behavior;
- and avoid creating a new top-level module unless user research proves it necessary.

---

# Phase 11 — Scale, resilience, certification, migration, and general availability

## 11.1 Performance

- signal and observation ingestion;
- spreadsheet and media processing;
- population and reconciliation queries;
- Today and Situation latency;
- authorized search and projections;
- workflow backlog recovery;
- audit and export scale;
- model cost and latency;
- cost per tenant or deployment.

## 11.2 Reliability

- database backup and restore;
- object-store recovery;
- message replay;
- workflow recovery;
- source outage;
- model outage;
- regional failure;
- offline-capture recovery;
- projection rebuild.

## 11.3 Security and privacy

- penetration test;
- tenant and entity isolation;
- protected-report assessment;
- mobile and offline assessment;
- supply-chain security;
- secret and key rotation;
- incident-response runbooks;
- retention, deletion, and legal-hold tests;
- residency and model-provider controls.

## 11.4 Deployment

Productize deployment modes only as demand requires:

1. dedicated cloud tenancy first;
2. multi-tenant SaaS where appropriate;
3. private cloud or on-premises after operational evidence;
4. hybrid evidence plane and customer-managed keys where required.

## 11.5 Migration

- source inventory;
- source profiles;
- mapping templates;
- duplicate and conflict handling;
- historical evidence import;
- situation reconstruction;
- authority import;
- parallel run;
- reconciliation, acceptance, and rollback.

## General availability gate

Requires:

- selected golden journeys passing;
- no unresolved critical isolation or protected-data defect;
- tested backup and recovery;
- AI and source degraded-mode operation;
- approved SLOs and runbooks;
- accessible and localized critical journeys;
- and pilot evidence that ClearSight reduces effort while strengthening data quality, evidence, decisions, and verified outcomes.

---

# 5. Program metrics

## Human effort

- active time per accepted evidence item;
- questions per request;
- duplicate requests avoided;
- redirected requests;
- manual reconciliation time;
- spreadsheet mapping reuse;
- and branch completion rate.

## Source and data quality

- source freshness;
- unresolved mappings;
- duplicate identifiers;
- partially accepted imports;
- source-health incidents;
- stale conclusions;
- and correction rate.

## Situation quality

- material situations later dismissed as noise;
- material situations detected late;
- duplicate situations;
- average executive situation count;
- time from observation to accountable handling;
- and context-switch errors.

## Evidence

- material claims with sufficient evidence;
- unresolved contradiction;
- evidence debt;
- unnecessary human requests;
- conclusion reversal;
- and source reuse.

## Decision and outcome

- time to decision;
- decisions returned for evidence;
- expired decisions;
- verification failure;
- reopened situations;
- and projected-versus-observed effect.

## AI trust

- source-lineage completeness;
- unsupported assertion rate;
- extraction correction rate;
- abstention quality;
- unauthorized-action attempt;
- protected-data leakage tests;
- latency;
- and cost.

## Experience

- executive comprehension time;
- capture completion;
- import resolution time;
- keyboard and screen-reader success;
- visual-regression defects;
- zoom and localization defects;
- and performance-budget adherence.

---

# 6. Work that must not be pulled forward prematurely

Avoid early expansion into:

- institution-wide ontology completeness;
- dozens of frameworks;
- broad report catalogues;
- generic form builders;
- dashboard marketplaces;
- extensive user-configurable layouts;
- dedicated graph infrastructure without benchmarks;
- many microservices;
- autonomous material decisions;
- all deployment modes;
- or full replacement of specialist systems.

These can consume delivery capacity while source trust, capture, situations, decisions, and verification remain incomplete.

---

# 7. Completion standard

A milestone is complete only when:

- a real bank situation is exercised;
- scope and source authority are explicit;
- observations remain traceable;
- unresolved and contradictory data are visible;
- the correct interaction form is used;
- authority and privacy are enforced;
- failure and degraded modes work;
- AI behavior is evaluated where used;
- accessibility and visual standards pass;
- and the outcome can be evaluated against a defined verification contract.

The core product is not finished until ClearSight can:

> receive imperfect bank data through realistic channels, show what can and cannot be trusted, create one understandable risk situation, ask only for missing facts, route the right decision, coordinate action, verify the defined outcome, and reconstruct the full history later.