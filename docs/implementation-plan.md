# ClearSight implementation ledger

**Status date:** 2026-08-07  
**P0 closure PR:** #30  
**Pre-P0 baseline:** `main@df98a7f66c28642637a45a10662abac042dcd144` (PR #25)

This is the authoritative execution ledger. Product, design, architecture and enterprise-productization documents define requirements, but this file controls the current implementation order and capability truth.

## 1. Current sequence

### #26 P0 executable integrity — complete in PRs #25 and #30

- [x] P0.1 typed route/access registry and verified actor/tenant binding.
- [x] persisted capture consolidation and bounded invitation/session security.
- [x] P0.2 durable source-health reconciliation through outbox/inbox → Signal/Drift → affected Program trigger/Matter → projection.
- [x] P0.3 independent worker work classes with bounded retry/dead-letter behavior.
- [x] P0.4 compound-command and post-commit response truth.
- [x] P0.5 executable route/OpenAPI contract parity.
- [x] P0.6 bounded effective-authority convergence across routing rules, assignments, grants, active delegations and segregation constraints.
- [x] lower-priority audit findings are moved to linked P1/P2 follow-up issues before #26 closes.

### Next: P1 semantic/current-state correctness

P1 is intentionally correctness-first rather than feature-first.

1. **P1.1 Program-state truth — first tranche**
   - valid-time/effective-time selection;
   - deterministic multi-source currentness;
   - bounded evidence-assessment validity;
   - mandatory unknown dimensions cannot become `CURRENT`;
   - preserve configured Program period across pause/resume;
   - expose assessed Program version/projection freshness and complete reason counts.
2. **P1.2 Matter closure current-record truth**
   - current Decision/Response selection and supersession;
   - expiry/conditions at closure;
   - verification independence and observation-period enforcement.
3. **P1.3 lifecycle-specific command responsibility**
   - proposer/reviewer/challenger/authorizer/signatory/transmitter/acknowledgement responsibilities vary by requested transition rather than static command name.
4. **P1.4 bounded ordinary reads and explicit work-model projection contracts.**
5. **P1.5 document-import resource, durability and paging hardening.**

Only after semantic/current-state correctness should wider #27 governed operator execution be treated as trustworthy execution rather than presentation.

## 2. Canonical domain invariants

These distinctions are mandatory:

- **Program** = ongoing obligation/compliance continuity.
- **Matter** = bounded change, exception, finding, decision, action, response or verification case.
- **Matter Action ≠ Workflow Task.** Action is accountable domain work; Task is an actor-facing routed step.
- **Signal ≠ conclusion.** A Signal is an observation that deterministic assessment may convert into drift or attention.
- **Submission ≠ sufficient evidence.** Evidence Contract assessment determines sufficiency.
- **Implementation ≠ verified outcome.** Completion alone cannot close material work.
- **Recommendation ≠ approval.** Current authority remains explicit.
- **Automation Policy ≠ execution receipt.** Permission is not evidence that an action ran.
- **Intervention Summary ≠ authoritative state.** It is a read projection over canonical records.

Do not add a parallel authorization, task, event, worker, receipt or generic workflow stack that duplicates these foundations.

## 3. #26 P0 closure truth

### P0.1 — route and identity boundary

`internal/httpapi/route_registry.go` is the canonical executable route inventory.

- [x] every production route has one explicit access class;
- [x] only health routes are truly public;
- [x] bounded capture routes use capability access;
- [x] other protected routes require verified actor context;
- [x] tenant/legal-entity and actor-bearing fields are rebound server-side where authoritative;
- [x] material commands resolve current authority at execution;
- [x] registry tests fail on invalid/unclassified route definitions.

Direct OIDC/SAML, directory synchronization, RLS/ABAC defense in depth and step-up assurance remain enterprise-release work, not #26 P0 blockers.

### Capture consolidation/security

- [x] one persisted Evidence Request/Invitation/Session/Submission/Artifact domain remains;
- [x] parallel `internal/capture` implementation is removed;
- [x] migration `000013_capture_consolidation` removes abandoned foundation request/invitation tables;
- [x] invitation audience hash is verified at redemption;
- [x] request deadlines/open-state and artifact upload state are enforced;
- [x] request expiry is maintained;
- [x] bearer capture CORS is supported.

### P0.2 — durable source-health reconciliation

- [x] `SourceHealthChanged` is internally consumed before publication completion;
- [x] inbox receipts provide idempotent internal delivery;
- [x] degradation/recovery updates the exact source-quality drift;
- [x] recovery cannot be forged through generic Signal ingestion;
- [x] active Evidence Contracts and currently-effective approved Requirement source mappings resolve affected Programs;
- [x] one unhealthy source cannot be cleared by recovery of a different required source;
- [x] Program trigger/Matter/projection consequences reuse existing transactional continuity paths;
- [x] PostgreSQL replay tests exercise active Program degradation, duplicate delivery and recovery.

### P0.3 — worker failure isolation

- [x] evidence-source maintenance, Program projection, delegation lifecycle, workflow timers and outbox delivery run independently in one deployable worker;
- [x] each class owns its interval, timeout, lease, batch, retry and backoff budget;
- [x] ordinary class failures/panics do not terminate unrelated work;
- [x] exhausted timers become `FAILED` and exhausted outbox events become dead-lettered;
- [x] terminal work is not reclaimable;
- [x] queue/class health exposes actionable lag/failure state;
- [x] shared PostgreSQL integration fixtures are deterministic under serialized package execution.

### P0.4 — transaction and command-response truth

Failed verification consequences already use `VerificationResultBundle` to commit the verification result together with required REOPEN/ESCALATE or CREATE_MATTER/link effects.

PR #30 closes the remaining response seam:

- [x] normalized Program/Matter version probes determine authoritative commit outcome without replaying event history;
- [x] a material update that commits but later fails response reconstruction returns a small `COMMITTED` degraded-response receipt rather than a false 5xx;
- [x] genuine pre-commit failures remain failures;
- [x] API PostgreSQL composition uses a reliable repository wrapper that preserves only a just-committed create result as a short-lived response fallback if the immediate reconstruction fails;
- [x] this fallback is never authoritative state and is deleted after normal reconstruction/fallback use;
- [x] no second durable receipt or transaction-orchestration framework was introduced.

Canonical transaction rule:

```text
identity/current-authority check
→ authoritative row change
→ append-only domain event
→ transactional outbox / required maintenance job
→ commit
→ optional detail/projection read
```

The final optional read may degrade the response detail; it may not reverse or misreport the commit.

### P0.5 — executable API contract parity

`api/runtime.openapi.json` is the mechanically verified production route/access contract. Detailed domain schemas may remain modular, but they cannot redefine runtime paths or access classes.

- [x] exact method/path inventory matches `route_registry.go`;
- [x] access class and administrative permission match mechanically;
- [x] signed identity and bounded capture security schemes are explicit;
- [x] capability and authenticated-or-capability routes are distinguishable;
- [x] CI fails when a production route is added/removed/reclassified without contract update;
- [x] CI uses `npm ci` against `web/package-lock.json`.

Generated browser transport types/client consolidation is a later custom-code deletion tranche; P0 does not add a dependency solely to claim generation.

### P0.6 — effective authority convergence

Migration `000014_effective_authority_routes` materializes current approved routing-policy rules into an indexed execution read model.

The production authority path now:

1. resolves matching compiled routes **and** current responsibility assignments in one bounded ranked query;
2. applies deterministic priority + specificity and fails closed on same-rank ambiguity;
3. expands active scoped delegation chains with cycle/depth protection;
4. enforces current authority-grant materiality limits when grants govern that decision class/entity;
5. applies active segregation constraints;
6. returns an explicit candidate set when several humans are currently eligible.

Additional guarantees:

- [x] execution no longer performs selector N+1 work over every active policy JSON document;
- [x] ROLE/POSITION selectors expand to current eligible occupants rather than arbitrary `LIMIT 1` selection;
- [x] unresolved selectors are excluded from execution but remain visible as integrity findings;
- [x] delegated actors may execute only when delegation scope/time/responsibility remains effective;
- [x] expired delegation, grant limit and segregation paths fail closed in PostgreSQL integration tests;
- [x] collective TEAM/QUEUE/COMMITTEE routes remain explicit collective/candidate semantics rather than pretending one arbitrary person is authoritative.

Full enterprise configuration administration—directory-backed principals, advanced quorum/committee workflow, step-up assurance, governed policy editing/version creation/impact preview/scheduled rollback—remains later enterprise work. P0 closes the unsafe execution seam without pretending those productization features already exist.

## 4. Current Today and automation truth

### Today

Non-demo Today currently projects active Workflow Tasks explicitly assigned to the verified principal. Completed/cancelled work is excluded. Team/unassigned work remains in its routing queue until resolved.

Demo/reference journeys may seed stakeholder presentation data, but are not production Today truth.

Today is not yet the complete event-driven intervention compiler envisioned by #27.

### Automation policy

`automation_policies` currently exposes governed eligibility/configuration boundaries. A visible policy does **not** prove that an automated action ran, succeeded or was independently verified.

Those claims require a real governed executor and persisted execution/verification evidence.

## 5. Enterprise work after semantic P1

Detailed enterprise requirements remain in `docs/engineering/enterprise-productization-implementation-plan.md` and product/design specifications.

Major later gates include:

- OIDC/SAML/SCIM/LDAP or equivalent controlled identity synchronization;
- source-backed organization/position lifecycle;
- complete configuration administration and rollback UX;
- production notifications and privacy-minimized templates;
- WebAuthn/TOTP/session management and policy-bound step-up;
- production object storage, malware scanning, retention and legal hold;
- PDF/OCR provider isolation;
- representative capacity evidence and retained query plans;
- backup/restore/provider-outage exercises;
- pilot-bank legal/configuration approval and governed go-live.

## 6. Release and validation rules

Checkboxes describe repository capability, not deployment readiness.

A tranche is not complete until the relevant executable gates pass on its exact head:

- `gofmt` and `go vet`;
- race-enabled Go tests;
- PostgreSQL composition, migrations and integration tests;
- TypeScript strict checking;
- Vitest/axe rendered-state tests;
- production Vite build;
- adversarial identity, tenant, authority, replay and degraded-path tests;
- representative performance/recovery evidence when cardinality or durability changes.

Never claim a branch or PR is CI-green based on an older commit.