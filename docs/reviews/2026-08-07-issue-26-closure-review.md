# Issue #26 closure review after PR #25

**Reviewed:** 2026-08-07  
**Baseline:** `main@df98a7f66c28642637a45a10662abac042dcd144`  
**Purpose:** distinguish what PR #25 actually closed from what must still be implemented before issue #26 can close.

## Executive conclusion

PR #25 materially reduced the issue and fixed more than the earlier progress comment reflects. The remaining work is not a request to redesign the application or add another framework.

The closure blockers are three connected seams:

1. **P0.4 command/transaction truth** — the new verification bundle is atomic, but successful writes can still be reported as failures when a later full aggregate read/refresh fails;
2. **P0.5 transport-contract truth** — runtime routes and identity rules are stronger than OpenAPI/browser contracts, so security/path/schema drift can still survive CI;
3. **P0.6 authority truth** — the command guard exists, but the authority decision does not yet converge persisted assignments, grants, active delegations/substitutions and segregation into one bounded effective decision, and configuration writes do not yet have safe governed bootstrap semantics.

Before #26 closes, its still-valid lower-priority findings also need an explicit disposition into linked follow-up issues rather than disappearing with the P0 closure.

## What PR #25 closed

### Route and identity boundary

Verified complete at the runtime boundary:

- one typed route registry owns route class and command policy;
- only health routes are public;
- bounded evidence invitation/session routes are capability-scoped;
- protected reads/writes require verified identity;
- tenant/principal/actor fields are rebound from verified identity;
- legal-entity scope participates in material authorization;
- administrative permissions and route classes have adversarial tests.

### Capture consolidation/security

Verified complete for the P0 defects raised in the addendum:

- the parallel `internal/capture` service is gone from the current tree;
- migration `000013_capture_consolidation` removes the unused foundation `evidence_requests` and `invitation_grants` tables;
- request creation rejects past deadlines;
- internal/session submission and artifact upload require the request to remain open;
- maintenance expires requests;
- invitation redemption verifies `audience_hash`;
- invitation/session lifetime is bounded by request state/deadline;
- bearer capability CORS supports `Authorization`.

### Source-health reconciliation

Verified implemented through the existing outbox/inbox, autonomy Signal/Drift, Program trigger and projection path. This is no longer the “next seam” described in the prior architecture document.

### Worker isolation

Verified implemented as independent in-process work classes with bounded retry/terminal handling. A microservice split or generic executor is not required for #26.

### Today minimum production truth

Non-demo Today no longer returns an unconditional empty list. It projects active Workflow Tasks assigned to the verified principal. The broader event-driven intervention compiler remains #27 work rather than a reason to reopen a parallel task model in #26.

## P0.4 — exact remaining transaction/command work

### Already fixed

`RecordVerificationResult` now builds a narrow `VerificationResultBundle`.

For failed checks:

- REOPEN / ESCALATE: verification result and Matter transition share one transaction;
- CREATE_MATTER: verification result, follow-up Matter, optional Program link, events, outbox records and projection maintenance are committed together by the PostgreSQL bundle repository;
- `LinkedProgramIDs` errors are propagated instead of discarded.

### Still broken

The service method still calls `GetMatter` after the bundle commits. The one-event verification path does the same.

The same pattern is widespread:

- Program create/mutation methods call `refreshAndGetProgram` after their authoritative write;
- `refreshAndGetProgram` may perform derived refresh and then `GetProgram`;
- Matter action/verification/response mutations call `GetMatter` after `ApplyMatterEvent`;
- `createMatterWithInitialLink` correctly makes refresh best-effort, but still calls `GetMatter` after the atomic create/link commit;
- `applyTriggerBundle` commits the bundle and then calls `GetProgram`.

Therefore a read/projection failure after commit can still surface as a command failure even though the authoritative mutation succeeded.

### Required subtasks

- define one small authoritative command receipt/commit result;
- return committed aggregate ID/version/event identity without requiring a full event-history replay;
- keep mandatory outbox/projection jobs in the transaction;
- make optional post-commit detail refresh best-effort and never capable of reversing the reported commit outcome;
- audit every Program/Matter material mutator and compound path for this invariant;
- add fault injection before first write, between bundled writes, at commit, and after commit during refresh/detail read;
- prove rollback before commit and truthful success after commit in PostgreSQL integration tests.

The fix should reuse current repository and bundle patterns. Do not add a generic transaction/orchestration framework.

## P0.5 — exact remaining contract work

### Verified drift

- `api/openapi.yaml` has no explicit production authentication/security scheme despite the runtime now being authenticated-by-default.
- Runtime governance routes are concrete paths such as `/governance/policies/{id}/submit`, `/approve`, `/reject`, `/retire`, while OpenAPI exposes a generic `/governance/policies/{id}/{action}` path that is not the registered runtime route.
- OpenAPI schemas still require authority-bearing `tenant_id`, `actor_id`, `maker_id`, `approved_by`, etc. in several client request bodies even though the server now derives/rebinds those values from verified identity.
- `web/src/api.ts` still builds tenant query/body scope from `/context` for ordinary protected calls.
- the runtime registry, three OpenAPI documents, Go DTOs and handwritten TypeScript transport types are not mechanically tied together.

### Required subtasks

- choose one canonical transport contract and define how split specs relate to it;
- express PUBLIC/authenticated/bounded-capability security in OpenAPI;
- make documented paths match exact runtime methods/paths;
- remove server-owned identity/scope fields from client-required request contracts;
- reconcile query, request, response and error shapes;
- stop the browser from manufacturing authority-bearing tenant/actor scope;
- generate TS transport types/client code from the contract or enforce equivalent drift checks without adding unnecessary tooling;
- fail CI for registry↔OpenAPI↔browser path/schema mismatch.

## P0.6 — exact remaining authority/configuration work

### Verified runtime gap

`commandauth.Guard` correctly requires a verified actor and invokes `authority.Service.Resolve`, but authorization succeeds only when the returned single principal ID equals the actor principal ID.

The PostgreSQL authority service currently:

1. loads all active routing policy JSON for the tenant;
2. decodes all rules;
3. resolves each rule selector separately;
4. sorts the resulting in-memory rules;
5. chooses the first eligible principal.

This preserves the N+1/cardinality concern from the audit.

The decision does not converge the persisted responsibility assignments, authority grants, active delegations/substitutions and segregation constraints into the material command decision. A valid persisted delegation therefore does not itself make the delegate executable authority.

Persisted routing-policy validation also permits only PRINCIPAL/POSITION/ROLE selectors, while queue/team/committee/candidate semantics expected by the authority model are not represented as first-class executable candidates.

### Configuration gap

Governance policy/delegation routes now require verified identity and config permissions, and maker/checker fields are rebound. However they are still ordinary authenticated writes. There is no complete configuration authorization/bootstrap model defining who may create/change the authority system itself without hard-coding executive role names or creating circular authority.

The lifecycle is also incomplete for real version management: create produces version 1, but there is no supported draft-definition edit/new-version/scheduled activation/supersession/rollback path.

### Required subtasks

- define one effective-authority decision spanning routing policies, assignments, grants, active delegations/substitutions and segregation/conflict rules;
- materialize/compile or otherwise query that model with bounded indexed lookups rather than per-policy/rule selector fan-out;
- explicitly model candidate sets/queue/team/committee/substitution semantics;
- define deterministic specificity/ambiguity for overlapping wildcard and exact routes;
- evaluate delegation scope/expiry, authority limits/materiality and segregation at command execution;
- define configuration bootstrap/break-glass semantics plus capability/responsibility/assurance requirements for every governance/config mutation;
- implement draft edit/new policy version, simulation/impact preview, scheduled activation, supersession and rollback;
- add end-to-end delegate allow/deny and conflict/segregation tests;
- add route-authorization coverage CI;
- add query-count and representative ~100k rule/assignment performance evidence.

## Lower-priority findings and issue closure

The original #26 body/addendum also contains P1/P2 findings covering Program temporal/state validity, Matter closure currency, workflow/action ownership, document-import resource behavior, full-history read cost, live write UX, schema ownership and other hardening.

The post-PR #25 comments intentionally narrowed the immediate implementation sequence to P0.4–P0.6. That narrowing is reasonable **only if the remaining valid P1/P2 findings are explicitly moved to linked follow-up issues or marked superseded with evidence before #26 closes**.

Do not mark the historical audit findings “done” merely because the P0 issue closes.

## Closure acceptance

Issue #26 can close when all of the following are true:

- [ ] P0.4 command receipts and fault-injected transaction truth are complete;
- [ ] P0.5 runtime/OpenAPI/browser parity is enforced by CI;
- [ ] P0.6 effective authority/configuration authorization is executable and bounded;
- [ ] remaining valid P1/P2 findings have linked follow-up ownership or evidence-backed supersession;
- [ ] the implementation ledger and architecture docs reflect current behavior;
- [ ] exact-head GitHub CI is green and required PostgreSQL integration/benchmark jobs actually ran.
