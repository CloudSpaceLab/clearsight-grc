# ClearSight Implementation Plan

This plan delivers ClearSight as a source-led, correctly routed, AI-assisted, bank-grade continuous compliance and risk operating system.

Checkboxes indicate planned work, not completed implementation.

## Delivery thesis

Every vertical slice must prove:

```text
Authoritative source, schedule, event, or report
→ Program update or typed Matter
→ known bank context assembled
→ responsible, reviewer, challenger, and authorizer resolved
→ only missing facts requested through a safe channel
→ grounded recommendation or first draft
→ authorized decision, action, or response
→ verification or acknowledgement
→ Program, queue, report, and history updated
```

Do not build isolated CRUD, generic forms, dashboards, approval chains, or AI demonstrations.

## Cross-cutting gates

Every milestone must define:

- use-case IDs and maturity;
- customer/persona outcome;
- routine active-effort and comprehension target;
- responsibility, authority, delegation, conflict, and escalation behavior;
- sources, prefill, evidence, and degraded paths;
- data cardinality, query shape, consistency, latency, and recovery budget;
- secure invitation/capture behavior where relevant;
- AI contribution and deterministic fallback;
- end-to-end and non-regression tests.

A milestone is not complete because its APIs or screens exist.

# Phase 0 — Product contracts, pilot, architecture decisions, and design foundation

## Product and pilot

- [ ] Select pilot bank, legal entity, Programs, Matter types, source systems, and deployment mode.
- [ ] Confirm pilot personas and the four initial journeys.
- [ ] Mark every use case as Foundation, Pilot, Expansion, or Enterprise.
- [ ] Define success metrics for effort, comprehension, correctness, evidence, authority, and performance.
- [ ] Confirm data residency, retention, protected-case, and external-invitation constraints.

## Workflow contracts

For each pilot use case:

- [ ] trigger, scope, outcome, non-goal, and state machine;
- [ ] performer, owner, reviewer, challenger, authorizer, signatory, and escalation owner;
- [ ] happy, ambiguous, conflict, absence, overdue, degraded, prohibited, closure, and reopening paths;
- [ ] known fields, unresolved facts, and evidence contracts;
- [ ] first-use and repeat-use UX references;
- [ ] performance and data profile.

## ADRs

- [ ] modular core and split criteria;
- [ ] tenancy, deployment, and residency;
- [ ] identity, authorization, purpose, and inference resistance;
- [ ] role/authority/routing/escalation model;
- [ ] workflow runtime, timers, concurrency, and idempotency;
- [ ] invitation and external-session exchange;
- [ ] temporal data, evidence storage, retention, and legal hold;
- [ ] relational, object, search, graph, vector, and audit stores;
- [ ] AI/model gateway and external automation adapters;
- [ ] offline capture;
- [ ] observability, workload profiles, SLOs, backup, and recovery;
- [ ] design tokens, accessibility, responsiveness, and reports.

## Acceptance gate

Do not begin feature implementation until:

- pilot use cases are traceable end to end;
- authority and escalation can be expressed without hard-coded users;
- invitation threat model and protected-report boundary are approved;
- system workload and performance budgets are explicit;
- representative low-fidelity flows pass role-based usability review.

# Phase 1 — Tenant, identity, scope, authority, routing, workflow, and audit foundation

## Core identity and scope

- [ ] tenant, institution, legal entity, jurisdiction, licence, Program, service, branch, vendor, customer/account, asset, and population scope;
- [ ] enterprise identity and directory integration;
- [ ] principal, team, queue, committee, external party, and service identities;
- [ ] valid time, record time, versioning, and supersession;
- [ ] immutable material audit and security event separation.

## Responsibility and authority

- [ ] role templates and organizational positions;
- [ ] scoped responsibility assignments;
- [ ] authority grants and decision policies;
- [ ] routing and escalation policies;
- [ ] delegation, substitution, handoff, conflict, recusal, and segregation of duties;
- [ ] sequence, parallel, quorum, veto, and challenge patterns;
- [ ] versioned Configure experience with simulation, impact preview, maker-checker, activation, and rollback;
- [ ] materialized assignment index and runtime explanation.

## Workflow runtime

- [ ] typed state machines, timers, working calendars, and deadlines;
- [ ] parallel steps, joins, waits, cancellation, and supersession;
- [ ] save/resume and changed-since-last-view;
- [ ] outbox/inbox, idempotency, optimistic concurrency, retry, and compensation;
- [ ] routing-failure and break-glass workflows.

## Gate

- actor resolution meets performance budget;
- no self-approval or circular delegation;
- identity/role change safely reroutes in-flight work;
- material actions re-evaluate current authority;
- common workflows resume without reconstruction;
- point-in-time responsibility and authority are reconstructable.

# Phase 2 — Source Registry, evidence, imports, invitations, and capture

## Source and observation foundation

- [ ] Source Profile, authority, limitation, scope, freshness, health, mapping, and purpose;
- [ ] relational Observation and Evidence metadata;
- [ ] versioned object storage, hash, scan, chain of custody, retention, and legal hold;
- [ ] first pilot inventory adapters;
- [ ] source degradation and recovery workflow.

## Imports

- [ ] resumable upload and scanning;
- [ ] file/sheet selection, schema fingerprint, mapping, preview, validation, matching, and partial success;
- [ ] row provenance and reconciliation queue;
- [ ] repeat mapping reuse and exception-only review;
- [ ] chunking, backpressure, idempotent retry, and cancellation;
- [ ] 1-million-row reference benchmark.

## Requests and invitations

- [ ] request aggregate and minimum-question compiler;
- [ ] internal assignment and SSO flow;
- [ ] external recipient and organization model;
- [ ] opaque invitation, hashed token, audience, expiry, revocation, session exchange, and step-up authentication;
- [ ] wizard schema, drafts, resume, correction, follow-up, receipt, and lifecycle;
- [ ] notification minimization, wrong-recipient, forwarded-link, replay, and abuse handling;
- [ ] mobile capture and approved offline queue;
- [ ] protected-report identity/content separation and two-way mailbox foundation.

## Gate

- repeat import active effort under five minutes;
- ordinary request median under three minutes;
- request contains only unresolved facts;
- forwarded, revoked, expired, replayed, and wrong-recipient links fail safely;
- upload and submission remain distinct from evidence sufficiency;
- invitations and capture meet latency, resilience, and leakage tests.

# Phase 3 — Program engine and continuous compliance state

- [ ] Program aggregate and templates;
- [ ] Authority Sources, provisions, Requirements, and applicability;
- [ ] Control Objectives and scoped Implementations;
- [ ] Claims and Evidence Contracts;
- [ ] review, filing, certification, and testing schedules;
- [ ] trigger evaluation and incremental state invalidation;
- [ ] multidimensional Compliance State;
- [ ] Program overview, Requirement table, exception views, recent changes, and filing history;
- [ ] Program-to-Matter creation;
- [ ] AI-assisted extraction, mapping, evidence-gap, and change summary with deterministic fallback.

## Gate

- Program state derives from governed data rather than manual RAG;
- relevant state appears within performance budget without full synchronous recomputation;
- users move directly from gap to evidence, owner, Matter, or decision;
- review by exception exposes denominator, omitted items, source health, and full-review triggers;
- Program remains usable without AI or a live source.

# Phase 4 — Matter, decision, action, response, and verification engine

- [ ] typed Matter aggregate and subtype registry;
- [ ] subtype state and closure contracts;
- [ ] trigger/source, scope, affected objects, evidence needs, and contradictions;
- [ ] actor-specific workspace composition;
- [ ] assignment, redirect, delegate, conflict, escalate, merge, split, cancel, supersede, and reopen;
- [ ] Decision aggregate, options, challenge, approval, conditions, expiry, and invalidation;
- [ ] Action and external execution state;
- [ ] Response Package and signatory flow;
- [ ] Verification Contract, observation period, acknowledgement, and outcome;
- [ ] timeline and point-in-time reconstruction;
- [ ] AI summary, routing, option, action, response-index, and verification proposals.

## Gate

- one dominant next action is correct per actor and state;
- parallel work does not force serial module hopping;
- action completion cannot close a Matter;
- merge/split preserves scope, deadlines, evidence, authority, history, and protection boundaries;
- source or AI outage has safe manual operation;
- common Matter pages and commands meet SLOs.

# Phase 5 — Initial business verticals

## Continuous NDPA Program

- [ ] checklist and ROPA import/reconciliation;
- [ ] DPO governance and evidence contracts;
- [ ] targeted ROPA updates through internal requests;
- [ ] DPIA screening and full DPIA Matter;
- [ ] privacy breach Matter and timing;
- [ ] vendor/processor review;
- [ ] annual filing package and acknowledgement.

## Regulatory Change Matter

- [ ] source intake/authenticity/status;
- [ ] provision segmentation and exact lineage;
- [ ] candidate Requirements and applicability;
- [ ] affected Program/control/system/vendor/owner mapping;
- [ ] implementation Matters, evidence, tests, and amendment propagation.

## Protected Authority Request Matter

- [ ] protected intake and legal-instrument review;
- [ ] subject/period/directive resolution;
- [ ] focused internal and external requests;
- [ ] KYC/address/records/AML/fraud/branch/legal tasks;
- [ ] response reconciliation, approval, signatory, transmission, acknowledgement, hold, and minimized outputs.

## Legacy finding or exception

- [ ] import with source provenance;
- [ ] canonical object matching;
- [ ] explicit owner/reviewer/authorizer route;
- [ ] action, evidence review, and verification;
- [ ] derived register/workplan/dashboard views.

## Gate

All four journeys pass traceability, timing, authority, invitation, source, AI, performance, recovery, protection, and point-in-time tests.

# Phase 6 — Expanded bank use cases

Deliver in vertical slices, not as generic modules:

- [ ] risk situation and material decision;
- [ ] exception/waiver and risk acceptance;
- [ ] incident and operational loss linkage;
- [ ] third-party onboarding, reassessment, deficiency, and exit;
- [ ] BIA and resilience testing;
- [ ] RCSA and KRI cycles;
- [ ] audit and assurance;
- [ ] policy lifecycle;
- [ ] complaint/conduct concern;
- [ ] data-subject request;
- [ ] protected reporting and investigator workflow;
- [ ] executive/committee pack, decision, dissent, and follow-up.

Each slice must complete its use-case contract and acceptance mapping before implementation.

# Phase 7 — Governed automation and external execution

- [ ] operator/capability registry and action classes;
- [ ] model and tool gateway;
- [ ] structured output, policy, authorization, and audit pipeline;
- [ ] prompt-injection and protected-data defenses;
- [ ] automation eligibility, simulation, blast-radius preview, canary, approval, activation, monitoring, suspension, rollback, expiry, and verification;
- [ ] constrained Probo, ITSM, IAM, email, messaging, and other adapters;
- [ ] evaluation harness and outcome-linked monitoring;
- [ ] provider and tool degraded mode.

## Gate

Automation ships only when it reduces effort without reducing correctness or authority, has bounded side effects, and can be suspended, recovered, and verified.

# Phase 8 — Enterprise scale, multi-entity, deployment, and GA

- [ ] multi-entity and jurisdiction operation;
- [ ] group/local responsibility and authority;
- [ ] residency-aware evidence and model routing;
- [ ] dedicated, VPC, on-premises, and sovereign deployment profiles;
- [ ] large examination/evidence rooms and board packages;
- [ ] workload isolation, partitioning, archive/tiering, and capacity automation;
- [ ] backup, restore, failover, RPO/RTO, and disaster-recovery exercises;
- [ ] tenant support, break-glass, migration, offboarding, deletion, and legal hold;
- [ ] independent security, privacy, accessibility, performance, and usability validation;
- [ ] operational runbooks, SLOs, cost models, and support readiness.

# Definition of done

A milestone is complete only when:

- its use cases and maturity are documented;
- actor routing, authority, escalation, conflict, and delegation work end to end;
- invitations and external capture are narrow and safe where used;
- source authority, provenance, contradiction, and evidence sufficiency are enforced;
- routine/checkpoint effort and comprehension targets are met without quality regression;
- data volume, query, latency, availability, and recovery targets pass;
- AI and integrations have safe fallbacks;
- task completion remains distinct from verified outcome;
- point-in-time reconstruction is possible;
- documentation, ADRs, implementation, and acceptance evidence are synchronized.
