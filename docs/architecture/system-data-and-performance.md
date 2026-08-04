# System, Data, and Performance Architecture

This document defines the initial system shape, authoritative data boundaries, consistency model, workload profiles, performance budgets, scalability, resilience, and deployment requirements for ClearSight.

The goal is:

> **Deliver bank-grade correctness and isolation while keeping routine work fast, resumable, and operationally simple.**

The first implementation should be a coherent modular core. Microservices, a dedicated graph database, a vector database, or a large autonomous-agent platform are introduced only when measured workload or isolation requirements justify them.

## 1. Architectural principles

1. Relational authoritative state; rebuildable projections.
2. Versioned source and evidence artifacts in object storage.
3. Strong consistency for material commands; explicit eventual consistency for search, dashboards, graph, vector, and reports.
4. Durable workflows, timers, idempotency, and save/resume.
5. Authorization before retrieval, after relationship expansion, and again at material execution or export.
6. Deterministic context before AI; AI is never required to open, save, route, or submit core work.
7. Large imports, extraction, package generation, and verification are asynchronous and resumable.
8. Tenant, legal-entity, purpose, sensitivity, and policy version are part of every data-access boundary.
9. Point-in-time reconstruction is designed into the data model, not added through audit logs later.
10. Performance and capacity are defined per use case and data shape.

## 2. Reference workload

Every deployment must produce a measured workload profile. The initial large-bank reference profile is a design target, not a sales limit:

- up to 25,000 named users;
- up to 2,500 concurrent authenticated sessions during peak governance cycles;
- up to 10 million institution, relationship, requirement, control, scope, and workflow objects;
- up to 100 million new Observations and Evidence Assertions per year;
- up to 1 million evidence artifacts per year;
- imports up to 1 million rows per file;
- bursts of 5,000 request or workflow events per hour;
- thousands of active timers and external invitations;
- multi-year bitemporal and audit retention;
- large board, regulator, and examination packages.

Pilot benchmarks must run at the greater of:

- twice the forecast pilot peak; or
- the minimum reference profile defined for the feature class.

## 3. Initial logical architecture

```text
Edge and Delivery
├── Web application and responsive capture clients
├── API gateway / ingress
└── Notification and invitation delivery adapters

Core modular application
├── Identity, Tenant, Institution, and Scope
├── Responsibility, Authority, Routing, and Escalation
├── Programs, Requirements, Controls, and Compliance State
├── Matters, Decisions, Actions, Responses, and Verification
├── Requests, Invitations, Capture, and Evidence
├── Source Registry, Imports, and Integrations
├── Regulatory and Authority Intelligence
├── Reports, Exports, Search, and Projections
├── Governed AI and Automation Gateway
└── Audit, Retention, Legal Hold, and Reconstruction

Durable infrastructure
├── Relational authoritative database
├── Versioned object storage
├── Workflow/timer and job execution
├── Transactional outbox and idempotent inbox
├── Cache
├── Search projection
├── Optional graph and vector projections
└── Append-only security and audit stores
```

Modules may run in one deployable unit initially but must own explicit schemas, commands, events, and transaction boundaries.

## 4. Primary aggregates and transaction boundaries

### Program aggregate

Owns Program identity, scope, configuration, schedules, and references to Requirements, controls, evidence policy, state, assurance, and linked Matters.

Program state is derived and versioned. It should not require locking every related observation in one transaction.

### Matter aggregate

Owns type, workflow state, scope, owner, deadlines, evidence needs, decisions, actions, response, verification, and closure contract.

### Request aggregate

Owns purpose, recipient, invitation, schema, draft, submission, validation, follow-up, and completion state.

### Decision aggregate

Owns options, evidence set, authority policy, reviewers, votes, rationale, conditions, expiry, actions, and verification.

### Source and import aggregates

Own Source Profile, mapping version, file/schema fingerprint, import run, row provenance, reconciliation, and recovery.

### Authority-policy aggregate

Owns role templates, responsibility assignments, authority grants, routing/escalation policies, delegation, conflict, versions, and activation state.

Do not create distributed transactions across all aggregates. Use local transactions, outbox events, durable workflows, reconciliation, and compensating actions.

## 5. Relational data model

Use stable sortable identifiers such as UUIDv7 or ULID where operationally supported.

Every material table should include as appropriate:

- tenant ID;
- legal-entity and scope references;
- object ID and version;
- valid-from/valid-to;
- recorded-at and superseded-at;
- state and state version;
- source or policy version;
- sensitivity and purpose classification;
- created/changed actor;
- correlation and causation IDs.

### Bitemporal strategy

- **Valid time** records when the fact, assignment, requirement, or relationship applied.
- **Record time** records when ClearSight learned, corrected, or superseded it.

Current-state tables or materialized views may optimize ordinary reads, but historical versions remain authoritative and immutable.

### High-volume tables

Observations, assertions, workflow events, notification events, audit references, and invitation/session events should be partitionable by tenant and time, with secondary partitioning or indexes based on dominant access patterns.

Avoid tenant-wide sequential scans. Every list, queue, report, and population query requires a documented index and cardinality model.

## 6. Object storage

Store original and derivative artifacts in versioned object storage:

- authority sources;
- evidence files;
- imports;
- media;
- response packages;
- frozen reports;
- integrity manifests.

Requirements:

- tenant and classification separation;
- encryption and key policy;
- immutable object version reference;
- content hash;
- malware and content-scanning state;
- retention and legal hold;
- lifecycle tiering;
- no raw artifact duplication in database logs, events, or analytics;
- pre-signed access only after current authorization and purpose checks.

Metadata remains authoritative in the relational store.

## 7. Workflow and timer runtime

The workflow runtime must support:

- typed state machines;
- parallel steps and joins;
- human tasks;
- durable timers and working calendars;
- save/resume;
- external waits;
- retries with backoff;
- idempotency and deduplication;
- optimistic concurrency;
- cancellation and supersession;
- compensation;
- changed-policy handling;
- replay and reconstruction.

At-least-once delivery is acceptable when handlers are idempotent. “Exactly once” should not be claimed across external systems.

Every side-effect command should use an idempotency key derived from tenant, workflow, action, target, and version.

## 8. Events and messaging

Use a transactional outbox for domain events and an idempotent inbox for consumers.

Events contain safe references and state versions, not raw evidence, invitation tokens, secrets, or protected identities.

Event consumers must:

- reject stale versions where required;
- tolerate duplicates and reordering;
- record processing position;
- support replay;
- expose lag and poison-message state;
- apply tenant and purpose boundaries.

Backpressure must protect the authoritative application when imports, AI, search indexing, or external integrations degrade.

## 9. Responsibility and authority resolution

Maintain authoritative policy records in the relational store and a materialized assignment index optimized for runtime resolution.

The index may include:

- tenant, entity, scope path, object type, relationship type;
- responsibility type and role template;
- principal or queue;
- authority limits;
- valid period;
- delegation, substitution, and conflict flags;
- source and policy version.

Resolution pipeline:

```text
requesting actor and purpose
→ object, scope, workflow state, and decision type
→ candidate assignments and authority grants
→ relationship and threshold evaluation
→ conflict and segregation checks
→ delegation/substitution and fallback
→ route, sequence, deadline, and explanation
```

Material action rechecks authority against current policy and identity state. Cached route results are hints, not final authority.

## 10. Invitation and external-session service

Invitation records must be separate from evidence content.

Store:

- request and recipient reference;
- audience and identity policy;
- token hash and issue generation;
- expiry, usage limit, revocation, and consumed state;
- delivery channel and delivery reference;
- session exchange and assurance level;
- suspicious use and security events.

Redemption flow:

```text
opaque token
→ constant-time lookup/verification
→ request and recipient state checks
→ audience and risk checks
→ optional identity step-up
→ short-lived bounded session
→ token removed from URL
```

Rate limiting, replay detection, device/session risk signals, and safe failure must not reveal whether a protected request exists.

## 11. Authorization architecture

Use a policy decision layer with domain-owned enforcement.

Effective access considers:

- tenant and deployment boundary;
- legal entity, jurisdiction, scope, and relationship;
- principal, role, assignment, and delegation;
- purpose and current workflow state;
- data classification, privilege, and protected-case membership;
- action type and materiality;
- current policy and object version.

Enforce authorization in:

- SQL/query construction and domain services;
- object-storage access;
- search and autocomplete;
- graph and vector expansion;
- counts and aggregations;
- caches;
- background jobs;
- reports and exports;
- AI retrieval and tools;
- invitation sessions.

Post-filtering an overbroad result is not sufficient for protected data.

## 12. Caching

Cache only when the key includes all material isolation dimensions:

- tenant;
- legal entity or scope;
- purpose;
- principal or authorization fingerprint;
- policy version;
- object/state version;
- locale where output differs.

Do not cache protected-case results in shared general caches. Use short TTLs and event-driven invalidation for authority, identity, source health, and material state.

The application must remain correct on cache miss or cache loss.

## 13. Search, graph, vector, and work-queue projections

These are rebuildable projections, not truth systems.

### Search

Authorization-aware indexing should use safe document segmentation, classification, tenant/scope tags, version references, and deletion/legal-hold propagation.

### Graph

Use relational relationships first. Add a dedicated graph engine only when traversal benchmarks or algorithms justify it.

### Vector

Use only for evaluated use cases. Store embeddings with source version, model/version, tenant, purpose, classification, and authorization tags. Protected content requires equivalent isolation or exclusion.

### Work queues

Precompute actor- and role-oriented queue projections where helpful, but recheck access and authority before rendering sensitive detail or executing action.

Projection lag must be observable and surfaced when it affects a decision.

## 14. Program and compliance-state computation

Do not recompute an entire Program synchronously on every page load.

Use incremental invalidation and versioned state computation:

1. an authoritative change emits affected object and claim references;
2. dependency resolution identifies impacted conclusions and Program dimensions;
3. jobs recompute bounded state slices;
4. current projections update atomically by version;
5. stale or pending recomputation is visible;
6. material actions may require fresh synchronous validation of the relevant slice.

## 15. Import and ingestion architecture

Large imports must use streaming or chunked processing:

```text
upload and durable receipt
→ scan
→ schema/header detection
→ preview
→ mapping and validation
→ chunked parsing
→ identifier matching and deduplication
→ accepted Observations
→ reconciliation queue
→ projection/state update
```

Requirements:

- resumable upload;
- bounded memory;
- row-level provenance;
- partial success;
- idempotent rerun;
- mapping and schema version;
- tenant quotas and backpressure;
- cancellation and supersession;
- progress based on completed durable stages, not false percentages.

## 16. AI and automation execution

AI and external automation run behind gateways.

AI requests include approved, minimized context and return structured proposals. Long analysis is asynchronous. The page shows deterministic source and workflow context while AI is pending.

External tools use constrained adapters with:

- scoped integration identity;
- allowlisted commands;
- idempotency;
- timeout and retry policy;
- versioned external object references;
- result validation;
- compensation or manual recovery;
- implementation evidence and later outcome verification.

Neither model providers nor external automation engines become authoritative stores.

## 17. Performance budgets

Initial service objectives under the reference workload:

| Interaction | Target |
|---|---:|
| Deterministic first meaningful content for common Today, Program, Matter, and request pages | p95 ≤ 1.5 s; p99 ≤ 3 s |
| Common command acknowledgement after durable commit | p95 ≤ 500 ms |
| Ordinary request step save | p95 ≤ 500 ms |
| Final request submission acknowledgement, excluding uploads | p95 ≤ 750 ms |
| Common filtered work queue or population page | p95 ≤ 1.5 s |
| Scoped search first page | p95 ≤ 1.5 s |
| Actor routing resolution | p95 ≤ 100 ms uncached; 25 ms cached |
| Material authorization decision | p95 ≤ 150 ms excluding external identity dependency |
| Invitation redemption/session exchange | p95 ≤ 500 ms |
| Point-in-time view for a bounded Matter or Decision | p95 ≤ 3 s |
| Async job state acknowledgement | ≤ 250 ms after durable enqueue |

A slower specialized operation must become an asynchronous, resumable workflow rather than block the primary interaction.

Performance budgets exclude user think time and external waiting but include authorization and tenant isolation.

## 18. Capacity and query discipline

Every feature design must state:

- expected rows/objects per tenant and per scope;
- growth rate and retention;
- common filters, sorts, joins, and aggregations;
- pagination strategy;
- index and partition strategy;
- materialization or precomputation;
- worst-case authorization expansion;
- export size and async threshold;
- benchmark fixture representing skew and hotspots.

Rules:

- cursor pagination for mutable large lists;
- no offset scans for deep populations;
- no N+1 source, directory, or policy calls;
- bulk commands operate through bounded manifests and chunks;
- dashboards query projections, not raw evidence joins;
- exports and large reports are asynchronous with frozen manifests;
- tenant-specific workload guards prevent noisy-neighbor failure.

## 19. Tenancy and deployment modes

Support a common logical model across:

- managed multi-tenant SaaS;
- dedicated tenant deployment;
- bank VPC/private cloud;
- on-premises or sovereign deployment where required.

Deployment policy determines database, encryption-key, object-store, network, model-provider, region, and residency isolation.

Application code must not depend on one physical tenancy strategy. Tenant and legal-entity isolation remain explicit in the authoritative model even in a dedicated deployment.

## 20. Availability, recovery, and degraded mode

Initial baseline, subject to deployment tier:

- core application monthly availability target: 99.9%;
- RPO: 15 minutes or better;
- RTO: 4 hours or better;
- tested restoration of database, object metadata, workflow state, and encryption configuration;
- multi-zone durability where supported;
- no single model or integration provider required for core operation.

Degraded behavior must cover:

- database replica or cache loss;
- job-worker restart;
- source unavailable or stale;
- model unavailable;
- notification provider failure;
- object scan/extraction delay;
- search or projection lag;
- partial external execution;
- authorization source degradation;
- invitation delivery delay;
- offline synchronization conflict.

Unsafe actions are blocked; drafts and durable workflow state remain available.

## 21. Retention, legal hold, and deletion

Retention is policy-driven by tenant, data class, purpose, jurisdiction, Program/Matter type, case state, and legal hold.

Requirements:

- retention state on metadata and object versions;
- legal hold prevents deletion and records authority;
- deletion propagates to search, vector, graph, caches, derivatives, and exports where legally required;
- protected identity and content may have separate policies;
- cryptographic erasure considered for isolated keys;
- offboarding produces a verified export and deletion/hold manifest;
- audit records preserve lawful proof without retaining unnecessary raw content.

## 22. Observability

Measure:

- latency by use case, tenant profile, and authorization complexity;
- queue depth, lag, retries, poison messages, and timer drift;
- import throughput and reconciliation backlog;
- route-resolution failure and stale identity data;
- invitation issuance, redemption, failure, revocation, and abuse signals;
- source health and affected conclusions;
- AI latency, cost, schema failure, abstention, and reviewer outcome;
- cache hit/miss without exposing protected keys;
- storage growth, partition health, and query plans;
- recovery objectives and restore tests.

Logs and metrics must not contain raw evidence, invitation tokens, customer data, protected identities, secrets, or unrestricted query text.

## 23. Performance release process

Before release:

1. define the feature workload profile;
2. benchmark deterministic and degraded paths;
3. test tenant skew, authorization complexity, and realistic history;
4. test concurrency, retries, and partial failures;
5. verify no correctness loss under caching or projection lag;
6. load-test invitations, timers, queues, imports, and exports;
7. record capacity, cost, and scaling trigger;
8. add production SLOs and alerts;
9. prove rollback or recovery.

## 24. Split criteria

A module may become an independent service when at least one is demonstrated:

- materially different scaling or latency profile;
- strong security or residency isolation;
- independent release and failure boundary;
- specialized runtime or storage need;
- sustained team ownership boundary;
- benchmarked contention in the modular core.

A service split must not weaken transaction semantics, authorization, temporal history, or operational simplicity.

## 25. Prohibited shortcuts

Do not:

- place raw evidence in events or logs;
- use search, graph, vector, or dashboard projections as authoritative state;
- make AI synchronous on the critical save/submit path;
- perform tenant-wide joins without scope and index design;
- trust cached authority for material execution;
- use permanent invitation tokens;
- rely on distributed transactions across external tools;
- claim exactly-once external side effects;
- let one slow integration block core workflow;
- build an unbounded custom-schema platform;
- introduce microservices without measured need;
- defer workload and recovery design until GA.

## 26. Definition of success

The architecture succeeds when ClearSight can maintain defensible Programs and Matters, resolve the correct actors, collect evidence safely, preserve complete history, and remain fast under realistic bank populations—while integrations, AI, search, imports, and external execution may degrade independently without corrupting authority or losing work.
