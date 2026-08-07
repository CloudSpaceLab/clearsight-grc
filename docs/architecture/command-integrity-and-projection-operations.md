# Command integrity, route access and projection operations

## Purpose

ClearSight must make it difficult to add an endpoint that accidentally bypasses identity, tenant scope or material authority. Material mutations must be attributable, authorized, atomic and recoverable. Derived Program status must remain fast to read without making a committed command depend on a second synchronous calculation.

This document describes the executable HTTP boundary introduced while addressing issue #26 and the remaining seams that gate issue #27 governed operator execution.

## 1. One executable HTTP route contract

`internal/httpapi/route_registry.go` is the canonical HTTP route inventory. A route is registered once with an explicit access class and, when material, its command-authorization policy.

Route classes are:

- **PUBLIC** — intentionally unauthenticated. Only health endpoints belong here.
- **AUTHENTICATED_READ** — requires verified actor context; tenant query scope is rebound from that actor.
- **AUTHENTICATED_OPERATION** — authenticated non-material operations such as authority resolve/simulate.
- **AUTHENTICATED_WRITE** — authenticated writes whose domain semantics do not require the material-command authority resolver.
- **MATERIAL_COMMAND** — authenticated write plus current authority resolution at execution time.
- **BOUNDED_CAPABILITY** — invitation/session capability routes whose short-lived bearer capability is the access boundary.
- **AUTHENTICATED_OR_CAPABILITY** — routes such as artifact upload that support either a signed internal actor or a valid bounded capture capability.

Registry validation rejects duplicate registrations, public mutation routes, and material routes without command policy.

This registry replaces separate route registration and command-policy inventories. Future #27 automation commands must extend this same contract rather than adding an agent-specific authorization layer.

## 2. Verified request actor boundary

Production signed identity provides:

- tenant;
- principal;
- legal entity;
- actor kind;
- authentication/assurance metadata;
- session;
- issue and expiry times.

The identity middleware verifies the envelope before placing the actor in request context.

For protected routes the registry then binds request scope:

- conflicting `tenant_id` query/body values are rejected;
- accepted requests use the verified tenant;
- actor fields written into domain DTOs come from the verified principal;
- legal-entity scope used for authorization comes from the verified actor;
- body `legal_entity_id` is injected only for DTOs that actually declare it.

Command-authorization mode does **not** change this identity rule. `off`, `audit` and `enforce` only change whether/how the authority resolver gates execution. They do not make client-supplied actor or tenant fields authoritative.

## 3. Strict DTO-safe actor binding

The HTTP JSON decoder rejects unknown fields. Therefore command identity binding is descriptor-driven rather than generic.

Examples:

- ordinary Program/Matter mutations use `actor_id`;
- applicability uses `approved_by`;
- evidence assessment uses `assessed_by`;
- Matter decision uses `authority_principal_id`;
- verification result uses `reviewer_principal_id`;
- projection reconcile/rebuild keep actor identity in request context and do not receive an unsupported JSON actor field.

This prevents a security wrapper from creating 400 responses by injecting fields that the target DTO does not define.

## 4. Material command authorization

Every `MATERIAL_COMMAND` route declares:

- command/decision name;
- object type;
- responsibility required now;
- minimum materiality;
- whether a service identity may execute it;
- DTO actor field;
- whether the DTO itself carries legal-entity scope.

At execution time the current authority route is resolved. The client cannot lower the descriptor's materiality floor. Matter transitions that enter decision/closure states may elevate the required responsibility/materiality.

In enforced mode execution fails closed when:

- verified identity is missing or mismatched;
- the current route cannot be resolved;
- the actor is not the selected principal;
- authority infrastructure is unavailable.

Operational errors remain explicit: sign-in required, wrong bank/legal-entity scope, authority unavailable, or current approval required.

## 5. Governance/configuration bootstrap boundary

Routing-policy and delegation lifecycle commands already enforce maker-checker and segregation rules, and their HTTP bodies are now actor/tenant-bound from verified identity.

They are intentionally **not yet classified as fully authority-routed material commands**. The current authority seed/configuration does not provide a safe bootstrap route for authorizing creation of the authority system itself.

Issue #26 requires a complete configuration authorization matrix with explicit bootstrap semantics, simulation, impact preview, effective dating and rollback. Do not solve this by hard-coding `CRO`, `GRC_ADMIN` or similar role names into HTTP handlers.

## 6. Bounded capture capabilities

External evidence capture is purpose-bound capability access, not a substitute staff identity.

Invitation/session routes remain capability-scoped. Evidence artifact upload supports:

- a valid bearer capture session; or
- a verified internal actor whose tenant/creator values are rebound server-side.

CORS explicitly allows the `Authorization` request header required by bearer capture sessions.

Protected internal routes must not be made public merely to support external capture.

## 7. Transaction truth

The intended material transaction boundary remains:

```text
command validation + identity/authority check
→ authoritative row changes
→ append-only domain/command event
→ transactional outbox event
→ required maintenance job
→ commit
```

A successful authoritative commit must not later be reported as a failed command because an asynchronous projection, publisher or reconciliation step failed.

Issue #26 still requires a route-by-route audit proving that compound commands meet this contract in PostgreSQL.

## 8. Program-status projection boundary

Program command version and calculated status/projection version remain distinct.

`continuity_projection_jobs` provides bounded leased maintenance, retry, failure state, health, reconciliation and manual rebuild. The UI may show stale known status with its assessed Program version, but it must not label that status current.

Projection reconcile/rebuild are material maintenance operations. Their verified actor is kept in request context while their existing strict DTO remains small.

## 9. Evidence reconciliation — next seam

Evidence/source transactions already have durable persistence and event/outbox primitives, but the complete cross-domain reconciliation bridge is still the next P0 item.

Required target:

```text
evidence/source transaction
→ transactional outbox
→ evidence-reconciliation work class
→ inbox dedupe
→ deterministic subject resolution
→ Program trigger and/or Matter consequence
→ projection maintenance
```

The bridge must be asynchronous, idempotent and tenant safe. A Signal/event is an observation; it cannot silently become a legal, risk or compliance conclusion.

## 10. Work-model separation

Do not merge these models while implementing reconciliation or future agent execution:

- **Matter Action** — canonical domain remediation/commitment and implementation state.
- **Workflow Task** — actor-facing routed/manual work needed to advance governed state.
- **Signal** — observed change/fact that may cause deterministic assessment.
- **Intervention Summary** — actor-facing read projection over canonical records.
- **Automation Policy** — permission boundary for eligible automation, not proof that automation ran.
- **Operator receipt** — future execution evidence written by the actual governed executor.

A generic task/event/agent abstraction that blurs these responsibilities is architectural bloat.

## 11. Remaining issue #26 gates before #27 executor work

In order:

1. evidence outbox/inbox reconciliation;
2. worker work-class isolation;
3. compound-command transaction audit/fixes;
4. route/query/OpenAPI/browser-client contract reconciliation;
5. complete governance/configuration authorization matrix;
6. only then governed #27 operator execution and persisted receipts.

This sequence is authoritative in `docs/implementation-plan.md`.

## 12. Validation requirements

The boundary requires adversarial tests, not only happy-path handler tests:

- public-route allowlist;
- protected route without identity;
- cross-tenant query/body spoofing;
- actor-field spoofing with authorization off/audit/enforce;
- strict DTO decoding after identity binding;
- capability route without staff identity;
- bearer CORS preflight;
- unresolved/mismatched authority;
- service-identity eligibility;
- PostgreSQL transaction/outbox/inbox replay and recovery.

No exact-head CI run means no CI-green claim.
