# Command integrity, route access and projection operations

**Status:** #26 P0 executable-integrity boundary implemented by PRs #25 and #30.

## Purpose

ClearSight must make it difficult to add a route or command that accidentally bypasses verified identity, tenant/entity scope, current authority, transaction truth or durable derived-state maintenance.

The architecture is deliberately a modular monolith. P0 strengthens existing identity, authority, continuity, outbox/inbox, worker and projection foundations instead of introducing parallel authorization, orchestration or event frameworks.

## 1. Executable route contract

`internal/httpapi/route_registry.go` is the canonical runtime route inventory. Every route declares exactly one access class:

- **PUBLIC** — intentionally unauthenticated; currently health only.
- **AUTHENTICATED_READ** — verified actor required.
- **AUTHENTICATED_OPERATION** — authenticated non-material operation.
- **AUTHENTICATED_WRITE** — authenticated write not classified as a material domain command.
- **MATERIAL_COMMAND** — verified actor plus current authority resolution at execution.
- **BOUNDED_CAPABILITY** — short-lived invitation/session capability.
- **AUTHENTICATED_OR_CAPABILITY** — either verified internal actor or bounded capability.

Material routes additionally declare object type, responsibility, materiality floor, service-identity eligibility and DTO actor-binding behavior.

`api/runtime.openapi.json` is the executable production HTTP contract. Tests require exact method/path, route-class and administrative-permission parity with the runtime registry. Detailed domain schemas may remain modular, but they cannot redefine production paths or access semantics.

## 2. Verified identity boundary

Production signed identity establishes tenant, legal entity, principal, actor kind, assurance/authentication metadata and session timing before protected handlers run.

For protected requests:

- conflicting tenant scope is rejected;
- authority-bearing tenant/principal fields are rebound from verified context;
- legal-entity authorization scope comes from the actor;
- actor binding is descriptor-driven so strict JSON DTO decoding remains valid;
- `off`, `audit` and `enforce` command-authorization modes never make caller-supplied identity authoritative.

External capture is intentionally different: invitation/session routes use bounded bearer capability rather than pretending an external respondent is a staff principal.

## 3. Capture boundary

There is one persisted Evidence Request capture domain. The former parallel `internal/capture` path and abandoned foundation request/invitation tables are removed.

The current boundary enforces:

- future request deadline on creation;
- expiry/open-state on internal submission, external session and artifact upload;
- invitation expiry bounded by request deadline;
- audience-hash verification at invitation redemption;
- session revocation/expiry;
- bearer `Authorization` CORS support.

A capture Submission remains evidence input; it is not automatically sufficient evidence.

## 4. Durable source reconciliation

Source-health state changes use existing transactional delivery primitives:

```text
Evidence Source transaction
→ transactional outbox
→ internal source-health consumer
→ inbox dedupe
→ exact Signal/Drift update or recovery
→ affected Program resolution
→ idempotent Program trigger
→ optional focused Matter
→ Program projection job
→ later external/log publication
```

Recovery is source-specific and reaches a Program only when all currently-required mapped sources are healthy. Generic Signal ingestion cannot forge source recovery.

This is the first canonical continuous-evidence reconciliation path; future signal classes should extend the same contract rather than add another event system.

## 5. Worker isolation

One deployable worker supervises independent bounded work classes for:

- evidence-source maintenance;
- Program projection;
- delegation lifecycle;
- workflow timers;
- outbox delivery.

Each class owns its interval, timeout, lease, batch, retry and backoff limits. Ordinary errors/panics degrade only that class. Exhausted timers become durable `FAILED`; exhausted outbox events become dead-lettered and neither is reclaimable.

The process may later be split only if measured scale/isolation requires it.

## 6. Material transaction truth

The authoritative command boundary is:

```text
verified identity/current authority
→ command validation
→ authoritative row change(s)
→ append-only domain event(s)
→ transactional outbox / required maintenance job
→ commit
→ optional detail/projection reconstruction
```

The optional final reconstruction is not part of commit truth.

### Compound commands

Failed verification consequences use a narrow `VerificationResultBundle` so the result and required REOPEN/ESCALATE or CREATE_MATTER/link consequence commit atomically in PostgreSQL.

### Post-commit response failures

For existing Program/Matter aggregates, the material HTTP boundary probes the normalized authoritative version before and after a handler failure. If the version advanced, the command is reported as `COMMITTED` with a small degraded-response receipt instead of a false 5xx.

For create commands, API PostgreSQL composition uses a reliable repository wrapper that temporarily retains only the just-committed create result. A normal reconstruction clears the fallback; if that immediate read fails, the committed create result can be returned once. The fallback is never durable or authoritative state.

This avoids a second receipt/orchestration framework while preserving the core invariant: **a successful material commit is never reported as an uncommitted failure merely because later read work failed.**

## 7. Program projection boundary

Command version and calculated Program-state version remain distinct.

`continuity_projection_jobs` owns leased, retryable Program-state maintenance. A known stale projection may be shown with its assessed Program version, but must not be presented as current.

P1 strengthens the semantic correctness of the projection itself; P0 establishes its durability and separation from command commit truth.

## 8. Effective authority

Migration `000014_effective_authority_routes` compiles currently-effective approved routing-policy rules into an indexed execution read model.

Production resolution combines:

1. compiled routing rules;
2. current responsibility assignments;
3. active scoped delegation chains;
4. applicable authority grants/materiality limits;
5. active segregation constraints.

The resolution path is bounded by requested tenant/entity/object/responsibility/decision/materiality/time rather than iterating every active policy and querying each selector separately.

Ranking uses explicit priority plus specificity. Same-rank conflicting candidate sets fail closed.

ROLE/POSITION selectors expand to current eligible occupants. If several humans are eligible, the Resolution carries an explicit candidate set and the command guard accepts only a member of that effective set. TEAM/QUEUE/COMMITTEE semantics remain collective rather than being silently collapsed to an arbitrary `LIMIT 1` occupant.

Unresolved selectors are excluded from execution but remain visible as integrity findings.

Enterprise administration still has later work—directory-backed lifecycle, richer committee/quorum semantics, step-up assurance, policy editing/new-version UX, impact preview and scheduled rollback. Those are productization features, not reasons to retain the pre-P0 unsafe execution seam.

## 9. Work-model separation

Do not merge these concepts:

- **Matter Action** — accountable remediation/implementation domain state.
- **Workflow Task** — actor-facing routed step.
- **Signal** — observation.
- **Drift** — deterministic assessment of a condition requiring attention.
- **Intervention Summary** — actor-facing read projection.
- **Automation Policy** — permission boundary, not execution evidence.
- **Operator receipt** — future evidence written only by a real governed executor.

This separation is the primary guard against generic workflow/agent bloat.

## 10. Next semantic tranche

P0 closes executable seam integrity. P1 begins with Program-state truth:

- valid/effective-time filtering;
- deterministic multi-source currentness;
- bounded evidence-assessment validity;
- mandatory unknown dimensions cannot produce `CURRENT`;
- Program pause/resume preserves configured validity period;
- reads expose assessed Program version/projection freshness and complete reason counts.

Matter closure current-record truth follows as the next P1 tranche.

## 11. Validation contract

Material changes require the relevant exact-head gates:

- `gofmt`, race-enabled tests and `go vet`;
- PostgreSQL composition, migrations and serialized integration tests;
- route/OpenAPI access-contract parity;
- adversarial identity/tenant/entity/actor tests;
- delegated/expired/grant/segregation/ambiguity authority tests;
- compound commit/post-commit response tests;
- outbox/inbox duplicate/recovery and worker terminal-work tests;
- TypeScript strict checking, rendered/axe tests and production web build.

An older green commit is not evidence for a newer head.