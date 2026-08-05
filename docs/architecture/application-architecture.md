# ClearSight Application Architecture

This is the canonical implementation architecture for the initial ClearSight application.

## Decision

Build a **modular monolith with separate API and worker processes** over one authoritative PostgreSQL database and versioned object storage. Keep domain boundaries explicit so selected workloads can split later without changing product semantics.

## Technology baseline

| Layer | Initial choice | Reason |
|---|---|---|
| Backend | Go, standard HTTP stack | predictable latency, low memory, simple deployment, strong concurrency |
| Web | React 19, TypeScript, Vite 8, Tailwind 4 | fast build/runtime, typed UI, direct component control |
| Authoritative data | PostgreSQL 18 | transactions, temporal records, JSONB where bounded, mature indexing and partitioning |
| Artifacts | S3-compatible versioned object storage | immutable large evidence and source objects |
| Async work | PostgreSQL durable jobs/outbox initially | transactional consistency without premature broker operations |
| Projections | PostgreSQL/read models first; replaceable search/graph/vector adapters | one operational surface until measured need |
| Observability | structured logs, metrics, traces, safe audit events | performance and reconstruction without sensitive payloads |

Dependency versions are pinned in the scaffold and updated through tested dependency PRs.

## Processes

### API

Handles authentication context, bounded deterministic reads, material commands, request saves/submissions, invitation redemption, and command acknowledgement.

The API must not synchronously perform model inference beyond a strict low-latency budget, document/media extraction, large imports/exports, external writes, full Program recomputation, or long verification observations.

### Worker

Claims durable jobs for outbox publication, timers, reminders, escalation, routing refresh, ingestion, extraction, matching, reconciliation, projection updates, AI recommendations, package generation, external execution, evidence expiry, and verification.

Workers use leases, bounded batches, retry policy, dead-letter review, idempotency, and backpressure.

### Web client

Renders deterministic context immediately and progressively adds recommendations or long-running results. It never becomes the only authorization boundary.

## Module boundaries

```text
Institution and Scope
Identity and Authorization
Authority Routing
Programs and Requirements
Matters and Workflow
Evidence and Capture
Decisions and Actions
Verification and Assurance
Regulatory and Authority Intelligence
Integrations and AI
Projections and Reporting
Audit and Temporal Reconstruction
```

Each module owns its invariants and persistence contract. Cross-module writes occur through explicit application services in one transaction where consistency requires it, then emit outbox events.

## Code layout

```text
cmd/api                 process composition
cmd/worker              background processing composition
internal/<domain>       domain and application behavior
internal/httpapi        HTTP transport, middleware and DTO translation
internal/platform       narrow technical utilities
migrations              authoritative schema evolution
api/openapi.yaml        public HTTP contract
web                     browser application
```

No domain package may depend on HTTP, React, a model provider, or a specific external automation engine.

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
→ workers update projections or external side effects
```

Material command responses include the new version and a correlation identifier. Clients that submit stale versions receive a conflict and changed-context summary.

## Query path

Queries authenticate and authorize scope, select a current/read projection, apply tenant/purpose/object filtering in the query, and return a bounded page with freshness. Never load a broad population then filter authorization in application memory.

## Authority resolution path

Resolve tenant, legal entity, object, responsibility, materiality, decision class, and time; read the active policy version and precomputed candidates; apply limits, delegation, substitution, conflict, segregation of duties, availability, and workflow state; return principal(s), rule IDs, explanation, and policy version; re-evaluate before material execution.

## Invitation path

```text
Issue high-entropy token
→ store only token hash and bounded audience/request metadata
→ deliver through approved channel
→ redeem once
→ exchange for short server-side session
→ remove token from URL/history
→ authorize each request operation against session
→ revoke on completion, expiry, recipient change or cancellation
```

The same session cannot browse a Program, Matter, directory, or unrelated request.

## Consistency

Strong consistency is required for authority used by a material command, workflow transitions, invitation redemption/revocation, decisions, response approval/signatory state, evidence submission receipt, legal hold, and protected-identity access.

Explicit eventual consistency is acceptable for search, dashboards, graph/vector projections, summary counts, and generated reports. Every material projection exposes freshness.

## Evolution triggers

Split a module only when measurement proves independent scaling, confidentiality/residency isolation, availability, deployment cadence, workload-engine, or ownership requirements justify the operational cost. A split must preserve commands, event contracts, idempotency, history, and user semantics.

## Scaffold scope

The initial code proves bounded HTTP behavior, Today, deterministic authority resolution, focused capture, one-time invitation exchange, PostgreSQL foundation, React shell, and performance smoke. It does not yet claim production authentication, PostgreSQL repositories, object storage, durable workflow execution, or bank connectors.
