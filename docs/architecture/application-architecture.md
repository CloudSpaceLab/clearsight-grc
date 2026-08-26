# ClearSight Application Architecture

This is the canonical implementation architecture for the current application.

## Decision

Build a **modular monolith with separate API and worker processes** over one authoritative PostgreSQL database and versioned object storage. Preserve explicit domain interfaces so selected workloads can split later without changing product semantics.

## Technology baseline

| Layer | Initial choice | Reason |
|---|---|---|
| Backend | Go standard HTTP stack | predictable latency, low memory and simple deployment |
| PostgreSQL access | pgx v5 behind `postgres` build tag | efficient PostgreSQL-native pooling and explicit production composition |
| Web | React, TypeScript, Vite and Tailwind | typed direct UI control and bounded bundle surface |
| Authoritative data | PostgreSQL 18 | transactions, temporal history, JSONB where bounded and mature indexing |
| Artifacts | S3-compatible versioned object storage | immutable evidence and source objects |
| Async work | PostgreSQL jobs/outbox initially | transactional consistency without premature broker operations |
| Projections | PostgreSQL/read models first | one operational surface until measured need |

## Processes

### API

Handles bounded deterministic reads, material command acknowledgement, authority resolution, request save/submit, workflow transitions, onboarding state and signal ingestion.

It must not synchronously perform document/media extraction, large imports/exports, model inference outside a strict latency budget, external writes, full Program recomputation or long verification observations.

### Worker

Claims durable jobs for:

- outbox publication;
- timers, reminders and escalation;
- routing-integrity scans and re-routing;
- evidence aging and expiry;
- source-health evaluation;
- signal normalization and drift assessment;
- readiness snapshot generation;
- ingestion, extraction, matching and reconciliation;
- bounded vendor website-icon discovery and orphaned upload-reservation cleanup;
- AI recommendations and report/package generation;
- external execution and outcome verification.

Workers use leases, bounded batches, retry policy, dead-letter review, idempotency and backpressure.

### Web client

Renders deterministic context immediately, then progressively adds recommendations and long-running results. It includes surface-aware Today and Vendors guidance, premium illustration primitives, empty states, Today, Vendors, Configure, readiness, authority explanation and focused capture. Guidance is optional and non-blocking. It is never the authorization boundary.

## Modules

```text
Institution and Scope
Identity and Organization
Authority Routing and Integrity
Programs and Requirements
Matters and Durable Workflow
Evidence and Capture
Third-Party Relationships and External Work
Signals, Drift and Readiness
Decisions and Actions
Verification and Assurance
Onboarding and Guided Adoption
Regulatory and Authority Intelligence
Integrations and Governed AI
Projections and Reporting
Audit and Temporal Reconstruction
```

## Build modes

- default build: deterministic in-memory repositories for fast unit tests and UI development;
- `postgres` build tag: pgx-backed authority, workflow, onboarding and autonomy repositories.

Both modes implement the same domain interfaces. Production deployments use the PostgreSQL mode.

## Command path

```text
HTTP request
→ authenticate and resolve active context
→ authorize purpose, scope and command
→ validate expected aggregate version
→ execute domain command in transaction
→ persist authoritative state and outbox event
→ commit
→ return durable acknowledgement
→ workers update projections or perform side effects
```

Vendor due diligence and vendor-completed Program or Matter work follow this command path. The third-party module owns the relationship association, request purpose, bank owner, reviewer, deadline, delivery recovery and review outcome. Evidence and Capture own the exact form snapshot, invitation, session, draft, artifact and submission. Program and Matter modules retain their existing ownership, authority and closure rules. A vendor response cannot directly complete a Matter action, pass an outcome check or change a Program state.

Vendor legal identity is addressed separately from a service relationship. `/api/v1/vendor-identities/{vendor_id}` reads or changes the shared vendor identity, while `/api/v1/vendors/{relationship_id}` remains the legal-entity-scoped relationship resource. Identity and approved-logo commands use verified actor context, current authority and their own optimistic versions; they do not mutate relationship versions.

The protected `/api/v1/vendor-identities/{vendor_id}/brand` route returns only validated stored PNG bytes. A version token identifies an immutable historical asset; the unversioned read resolves the current approved override, then the current discovered icon. Remote URLs, storage keys, digests and job identifiers are not browser contracts. Upload reservation is durable before object write, final metadata/event/outbox/receipt state is transactional, and the worker reclaims uncommitted stored objects. Removing an override retains history and restores a matching discovered icon when available.

Outbound discovery uses an isolated no-proxy HTTPS client with address and redirect validation, bounded responses and safe raster conversion. It is enabled by default only for development. Production must opt in with `CLEARSIGHT_VENDOR_BRAND_DISCOVERY_ENABLED`; vendor records continue to use monograms when the worker is disabled or retrieval fails.

## Continuous-autonomy path

```text
Source/event/schedule
→ idempotent Signal
→ deterministic drift assessment
→ affected-object and source lookup
→ readiness dimension update
→ focused Matter/request/task where intervention is required
→ actor and authority resolution
→ action or decision
→ verification
```

Signal ingestion cannot directly approve applicability, declare compliance, accept risk or close a Matter.

## Authority path

Active routing policy versions contain ordered rules and target selectors. A selector may resolve a principal, organizational position, role-bound position, team, queue or committee. Resolution applies scope, materiality, decision class, validity, delegation, conflict and current state. The result contains principal, rule, policy version and explanation.

Routing integrity continuously detects unresolved selectors, missing authorizers, equal-priority ambiguity, empty positions, expired delegation and in-flight ownership gaps.

## Consistency

Strong consistency is required for:

- authority used by material commands;
- workflow transitions;
- invitation redemption and revocation;
- decisions, signatory state and protected identity access;
- evidence submission receipt;
- legal hold.

Search, dashboards, graph/vector projections, readiness summaries and generated reports may be eventually consistent but must expose freshness.

## Evolution triggers

Split a module only when measured independent scaling, confidentiality/residency isolation, availability, deployment cadence, workload engine or team ownership requirements justify the operational cost.
