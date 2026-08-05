# AGENTS.md

Mandatory rules for every human contributor, coding agent, design agent, reviewer, and automated change applied to ClearSight. **MUST**, **MUST NOT**, **SHOULD**, and **SHOULD NOT** are normative.

## Mission

Every change must help bank stakeholders understand what must be done, what proves it, what changed, who performs, reviews or authorizes, and whether the required outcome was achieved—with the minimum reasonable effort.

## Read first

1. [`README.md`](README.md)
2. [`docs/README.md`](docs/README.md)
3. relevant product specification and use-case ID
4. [`docs/architecture/application-architecture.md`](docs/architecture/application-architecture.md)
5. relevant ADR, implementation phase, and release gate

## Product invariants

- Programs maintain continuing obligations; Matters handle bounded change and exception.
- A task, upload, API success, or implementation is not a verified outcome.
- Performer, owner, reviewer, challenger, authorizer, signatory, and escalation owner remain distinct.
- One dominant next action is per actor and workflow state; other actors may work in parallel.
- Prefill before asking; search existing authorized evidence before requesting more.
- Review by exception must expose denominator, omitted items, source health, sampling, and full-review triggers.
- Invitation access is narrow, purpose-bound, short-lived, revocable, and exchanged for a bounded server-side session.
- AI proposes; policy and authorized humans decide material matters.
- Material history is versioned and reconstructable.

## Implementation architecture

The initial system is a modular monolith:

```text
cmd/api and cmd/worker
→ internal domain/application packages
→ PostgreSQL authoritative state and durable workflow
→ versioned object storage
→ rebuildable projections
→ governed external adapters
```

Do not add a microservice, broker, graph database, vector database, cache cluster, or workflow product without an ADR and measured need.

### Package boundaries

- `internal/<domain>` owns domain types, invariants, and application behavior.
- `internal/httpapi` translates HTTP only; it must not own business rules.
- persistence implementations depend on domain interfaces, not the reverse.
- cross-domain mutations use explicit application services and durable events.
- packages must not import frontend-generated code or infrastructure clients into domain logic.

### Backend rules

- Use the Go standard library where it remains clear and sufficient.
- Every HTTP server must set finite read, write, header, idle, and body-size limits.
- Use `context.Context` for request, database, and external-call cancellation.
- Material writes require optimistic concurrency or an equivalent explicit conflict rule.
- External writes require idempotency keys and result reconciliation.
- Never log invitation tokens, secrets, raw restricted evidence, protected identities, or full request bodies.
- Return stable machine error codes and safe human messages.
- Tests must cover invariants, unauthorized paths, concurrency, replay, expiry, and degraded operation.

### Database rules

- PostgreSQL is authoritative for material structured state.
- Every high-volume table requires tenant-leading indexes, cardinality estimates, retention, and a partition decision.
- Use time-ordered UUIDs for write-heavy identifiers where supported.
- Store large artifacts in object storage; relational rows keep versioned metadata and hashes.
- Material rows are superseded rather than overwritten when historical truth matters.
- Migrations are forward-only in production; down files are local-development aids, not a rollback strategy for data-bearing releases.
- Queries must be authorization-aware and bounded; no unpaginated population endpoints.

### Frontend rules

- Deterministic context renders before AI or long-running enrichment.
- Use semantic HTML, keyboard operation, visible focus, non-color status, reduced motion, and 200% zoom support.
- Large populations use virtualized/paginated tables, not card walls.
- Do not store invitation tokens, protected data, or durable authority in browser storage.
- All material actions re-confirm current scope, side effects, authority, and version.
- The web client may optimize presentation but cannot enforce the only authorization check.

## Authority and routing

Routing must be resolved from versioned policy, scope, object relationship, materiality, authority limits, delegation, substitution, conflict, segregation of duties, availability, and current workflow state.

Do not reduce routing to one `assignee_id`, hard-code executives, or silently transfer accountability during escalation.

Routing configuration requires simulation, impact preview, maker-checker approval, effective dates, rollback, and point-in-time reconstruction.

## Respond and Capture

Requests derive from an exact Claim, Evidence Contract, case directive, or governed purpose. Show known facts, the smallest unresolved question, acceptable response types, sensitivity, deadline, and redirect/wrong-recipient/concern routes.

Invitation tokens must be opaque, high-entropy, hashed at rest, audience/request/expiry bound, one-time where appropriate, removed from URLs after redemption, and excluded from logs and analytics. Error responses must not reveal whether a token was unknown, used, expired, or revoked.

Protected anonymous reporting uses a separate identity-isolated mailbox and reveal workflow.

## Performance and efficiency

Every feature must define:

- expected cardinality and growth;
- dominant read/write/query path;
- latency, availability, and consistency target;
- index, partition, caching, and retention strategy;
- authorization cost;
- backpressure, retry, and recovery behavior;
- observability without sensitive content.

Critical targets:

- common deterministic page p95 ≤ 1.5 s;
- common command acknowledgement p95 ≤ 500 ms;
- authority resolution p95 ≤ 100 ms uncached;
- request-step save p95 ≤ 500 ms;
- invitation redemption p95 ≤ 500 ms.

Do not place AI, document extraction, large export generation, or remote execution on the synchronous save/submit path.

## Security and privacy

Enforce tenant, legal entity, relationship, purpose, sensitivity, workflow state, and authority server-side for reads, counts, search, graph/vector retrieval, caches, exports, bulk work, AI context, jobs, invitations, and writes.

Prevent inference through labels, counts, snippets, suggestions, timing, cache keys, and manifests. Re-evaluate authority for material command execution and export download. Break-glass access must be explicit, time-limited, notified, and reviewed.

## Definition of done

A change is complete only when:

- it maps to a documented use case and maturity;
- domain and authority invariants are tested;
- known context is reused;
- the UX meets effort and comprehension gates without quality regression;
- data and performance assumptions are measured;
- replay, conflict, partial failure, retry, and recovery are safe;
- AI and integrations have deterministic fallbacks;
- history and audit are reconstructable;
- docs, ADRs, migrations, API contract, implementation plan, and tests agree.

If a feature is possible but cumbersome, unbounded, unsafe under delegation, dependent on hidden configuration, or unproven at expected scale, it is not finished.
