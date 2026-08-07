# Command integrity, route access and projection operations

**Status:** reconciled against `main@df98a7f66c28642637a45a10662abac042dcd144` after PR #25.

## Purpose

ClearSight must make it difficult to add an endpoint that accidentally bypasses identity, tenant scope, material authority or durable command semantics. Material mutations must be attributable, authorized, atomic and recoverable. Derived state must remain fast to read without making a committed command depend on a later synchronous replay or projection refresh.

This document records the executable boundary now present and the remaining issue #26 gates before governed operator execution from #27.

## 1. One executable HTTP route contract

`internal/httpapi/route_registry.go` is the canonical runtime route inventory. A route is registered once with an explicit access class and, when material, its command-authorization descriptor.

Route classes are:

- **PUBLIC** — intentionally unauthenticated. Only health endpoints belong here.
- **AUTHENTICATED_READ** — requires verified actor context; tenant query scope is rebound from that actor.
- **AUTHENTICATED_OPERATION** — authenticated non-material operations such as authority resolution and simulation.
- **AUTHENTICATED_WRITE** — authenticated writes whose current semantics do not yet use material authority resolution.
- **MATERIAL_COMMAND** — authenticated write plus current authority resolution at execution time.
- **BOUNDED_CAPABILITY** — invitation/session capability routes whose short-lived bearer capability is the access boundary.
- **AUTHENTICATED_OR_CAPABILITY** — routes such as artifact upload that support either a verified internal actor or a valid bounded capture capability.

Registry validation rejects duplicate registrations, public mutation routes and material routes without command policy. Future #27 automation commands must extend this same contract rather than adding agent-specific authorization middleware.

## 2. Verified request actor boundary

Production signed identity provides tenant, principal, legal entity, actor kind, authentication/assurance metadata, session and issue/expiry times. The identity middleware verifies the envelope before placing the actor in request context.

For protected routes the registry then binds request scope:

- conflicting `tenant_id` query/body values are rejected;
- accepted requests use the verified tenant;
- actor fields written into domain DTOs come from the verified principal;
- legal-entity scope used for authorization comes from the verified actor;
- body `legal_entity_id` is injected only for DTOs that declare it.

Command-authorization mode does **not** change this identity rule. `off`, `audit` and `enforce` change authority enforcement only; they never make client-supplied actor or tenant fields authoritative.

## 3. Strict DTO-safe actor binding

The HTTP JSON decoder rejects unknown fields, so command identity binding is descriptor-driven rather than generic.

Examples:

- ordinary Program/Matter mutations use `actor_id`;
- applicability uses `approved_by`;
- evidence assessment uses `assessed_by`;
- Matter decision uses `authority_principal_id`;
- verification result uses `reviewer_principal_id`;
- projection reconcile/rebuild keep actor identity in request context and do not receive an unsupported JSON actor field.

This prevents the security boundary from creating false 400 responses by injecting fields that a target DTO does not define.

## 4. Material command authorization

Every `MATERIAL_COMMAND` route declares command/decision name, object type, responsibility, minimum materiality, service-identity eligibility, DTO actor field and whether the DTO carries legal-entity scope.

At execution time the current authority route is resolved. The client cannot lower the descriptor materiality floor. Matter transitions that enter decision or closure states may elevate responsibility and materiality.

In enforced mode execution fails closed when verified identity is missing/mismatched, no current route resolves, the actor is not the permitted principal, or authority infrastructure is unavailable.

This boundary is implemented, but the **authority model behind it is not yet complete**. See section 9.

## 5. Capture is one persisted domain

PR #25 removed the parallel demo `internal/capture` path and the unused foundation `evidence_requests` / `invitation_grants` tables. Demo and production capture now use the persisted Evidence Request domain.

The capture boundary now enforces the security defects identified in the #26 addendum:

- request deadlines must be future-dated;
- expired/closed requests cannot accept internal submissions, external sessions or artifacts;
- request maintenance can persist expiry;
- invitation redemption verifies the stored audience hash as well as token possession;
- invitation lifetime cannot exceed the request deadline;
- bearer-session CORS includes `Authorization`.

External capture remains purpose-bound capability access, not substitute staff identity.

## 6. Evidence reconciliation and worker isolation

PR #25 completed the first durable source-health reconciliation bridge:

```text
Evidence Source transaction
→ transactional outbox
→ internal source-health consumer
→ inbox dedupe
→ exact Signal/Drift degradation or recovery
→ dependent Program resolution
→ existing Program trigger / optional focused Matter
→ Program projection job
→ external/log publication
```

The worker remains one deployable process but now runs independent work classes for evidence-source maintenance, Program projection, delegation lifecycle, workflow timers and outbox delivery. Each class has bounded retries, timeout/lease semantics and independent health so one ordinary failure does not stop unrelated classes.

Do not reintroduce a second event bus, generic worker pool or agent executor for these responsibilities.

## 7. Transaction truth — partially complete

The intended material transaction boundary is:

```text
command validation + identity/authority check
→ authoritative row changes
→ append-only domain/command event
→ transactional outbox event
→ required maintenance job
→ commit
→ authoritative command receipt
```

The verification failure path now has a narrow atomic bundle. `RecordVerificationResult` can commit the result with its required REOPEN/ESCALATE transition or CREATE_MATTER follow-up/link in one PostgreSQL transaction, and linked-Program lookup errors are no longer discarded.

The remaining defect is **post-commit response truth**. Several Program and Matter mutators still commit and then call `GetProgram`, `GetMatter` or synchronous refresh/reconstruction. A later read/projection failure can therefore make an already committed command appear to fail.

Issue #26 P0.4 requires:

1. a small authoritative command receipt/current-version result from the commit path;
2. no material command returning failure solely because optional post-commit rehydration or derived refresh failed;
3. PostgreSQL fault-injection tests at pre-commit, commit and post-commit seams;
4. audit of all compound paths, including Program mutation/refresh, Matter create-with-link, trigger bundles and verification bundles.

Do not add a general transaction coordinator. Extend the existing repository/bundle patterns only where multiple authoritative records must share one transaction.

## 8. API/OpenAPI/browser contract — still open

The runtime route registry is now stronger than the published contract. `api/openapi.yaml` still lacks an explicit production authentication/security model, still presents some runtime concrete action routes as generic `{action}` paths, and still describes verified tenant/actor fields as client-supplied request data. The browser client also continues to send tenant scope from `/context` even though the server now derives authority-bearing scope from verified identity.

Issue #26 P0.5 requires one executable contract boundary:

- every runtime route must map to a canonical OpenAPI operation;
- the spec must describe authenticated, public and bounded-capability security correctly;
- client-facing schemas must not require caller-supplied tenant/actor identity that the server owns;
- split/additional specs must be generated from or mechanically checked against the canonical contract;
- TypeScript request/response types and URLs must be generated or checked in CI against that contract;
- route/schema/client drift must fail CI.

## 9. Effective authority and configuration bootstrap — still open

Governance policy/delegation writes are actor-bound and permission-gated, but they are still `AUTHENTICATED_WRITE` rather than a complete governed configuration-command matrix.

More importantly, PostgreSQL material command authorization still resolves only persisted routing-policy JSON into one selected principal. The current path loads active policy definitions, resolves selectors separately, and then the command guard compares the selected principal ID with the actor. Persisted responsibility assignments, authority grants, active delegations/substitutions and segregation constraints are not yet one executable authority decision.

This produces correctness and scale gaps:

- a valid approved delegation is not sufficient to execute the delegated responsibility;
- grant/materiality limits and segregation constraints are not evaluated as one decision;
- route resolution cost grows with tenant-wide active rules and selector lookups;
- TEAM/QUEUE/COMMITTEE/candidate-set semantics are not represented by the persisted policy validator;
- overlapping wildcard/specific rules do not have a complete deterministic specificity model;
- policy lifecycle still lacks supported draft editing/new-version, scheduled activation, supersession and rollback.

Issue #26 P0.6 requires **convergence, not another RBAC engine**:

1. compile or query routing rules, assignments, grants, active delegations/substitutions and segregation constraints through one indexed effective-authority read model;
2. resolve with bounded queries keyed by tenant, legal entity, object/scope, responsibility, decision class, materiality and effective time;
3. represent candidate sets/queues/committees explicitly and define deterministic specificity/ambiguity rules;
4. re-evaluate delegation scope, grant limits, conflicts and segregation at material execution;
5. define governed configuration bootstrap semantics and map every governance/config mutation to required capability/responsibility/assurance without hard-coded executive role names;
6. support simulation, impact preview, real version creation/editing, effective dating, supersession and rollback;
7. prove delegation allow/deny behavior, ambiguity rejection and bounded query count at enterprise-scale rule/assignment cardinality.

## 10. Work-model separation

Do not merge these models while closing #26 or implementing #27:

- **Matter Action** — canonical remediation/commitment and implementation state.
- **Workflow Task** — actor-facing routed/manual work needed to advance governed state.
- **Signal** — observed change/fact that may cause deterministic assessment.
- **Intervention Summary** — actor-facing read projection over canonical records.
- **Automation Policy** — permission boundary for eligible automation, not proof that automation ran.
- **Operator receipt** — future execution evidence written by the actual governed executor.

A generic task/event/agent abstraction that blurs these responsibilities is architectural bloat.

## 11. Issue #26 closure gates

PR #25 completed the route/identity boundary, persisted capture consolidation/security fixes, source-health reconciliation and worker failure isolation. Issue #26 remains open for:

1. **P0.4** — authoritative transaction/command-response truth across material mutations;
2. **P0.5** — runtime route ↔ OpenAPI ↔ browser-client contract reconciliation;
3. **P0.6** — effective authority convergence and governed configuration authorization/bootstrap;
4. disposition of still-valid lower-priority audit findings into linked follow-up issues rather than silently treating them as fixed;
5. exact-head CI and PostgreSQL integration evidence for the completed seams.

Only after P0.4–P0.6 may #27 add governed operator execution and persisted execution receipts.

## 12. Validation requirements

The boundary requires adversarial tests, not only happy-path handler tests:

- public-route allowlist and missing identity;
- cross-tenant/entity query/body/form spoofing;
- actor-field spoofing with authorization off/audit/enforce;
- strict DTO decoding after identity binding;
- capability route without staff identity and bearer CORS preflight;
- unresolved/mismatched/delegated/expired authority;
- ambiguity, materiality/grant and segregation checks;
- route/OpenAPI/client parity;
- PostgreSQL compound transaction rollback and post-commit response truth;
- outbox/inbox replay, duplicate delivery and worker degraded-path recovery;
- authority query-count/cardinality benchmark.

No exact-head CI run means no CI-green claim.
