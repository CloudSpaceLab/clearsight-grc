# ClearSight implementation ledger

**Status date:** 2026-08-10  
**Current execution issue:** #27  
**Umbrella pilot/GA catalogue:** #13

This file is the authoritative implementation ledger. Code, migrations and executable tests remain final capability truth. Completed detail belongs in focused architecture documents, PRs and tests rather than parallel planning frameworks.

## 1. Completed executable tranches — do not rebuild

| Capability | Completion |
| --- | --- |
| P0 route / identity / transaction / worker / authority integrity | PRs #25, #30 / #26 |
| P1 effective current-state, lifecycle, bounded reads and durable document imports | PRs #34–#39 / #32 |
| Intervention-first UI/UX foundation | PR #31 |
| Low-effort typed Capture + request-bound artifacts | PR #40 |
| P2 durable-schema ownership and dead compatibility removal | PRs #41, #42 / #33 |
| Today actor work queue + canonical Matter authority/materiality | PR #43 |
| Deterministic lifecycle work compiler | PR #45 |
| Governed lifecycle sequencing | PR #46 |
| Canonical Evidence Request recipient truth | PR #49 |
| Evidence Request recipient lifecycle + Today projection | PR #50 |
| Governed Program/Work mutation UX | PR #51 |
| Actor Program review baseline + review-by-exception | PR #53 |
| Protected Matter read parity + Program review explanation delta | PR #55 |

### PR #55 — protected Matter read parity and review-digest correction

- canonical `MatterVisibleTo` / `ParseMatterAccessPolicy` remains policy reference;
- production Matter list/search applies actor visibility before pagination/limit and fails closed on malformed protected access metadata;
- hidden Matters cannot consume page slots, alter cursors or leak existence;
- Program explanation-only changes produce explanation review deltas without manufacturing exception identity;
- no RLS layer, visibility table, policy engine, preference stack or new authorization framework was introduced;
- squash-merged as `e9e61cafa5d6715b3e94bd72454b58b3ead87ff4`.

## 2. Current execution — enterprise identity/access

PR #59 implements the focused OSS-first boundary in `docs/engineering/enterprise-identity-access.md`.

ClearSight owns OIDC, server sessions, local capabilities, SCIM provisioning and integration with existing authority/work routing. LDAP/AD/SAML/Kerberos/password/MFA infrastructure remains upstream in authentik, an existing bank Keycloak deployment, or another standards-compliant identity layer.

### EIA-0 — implemented on PR #59

- `org_positions.department_path` provides stable hierarchy without a department subsystem;
- `role_templates.capabilities` provides bounded coarse application eligibility;
- routing-policy definitions accept strict bounded multi-level escalation sequences selecting responsibility + department ancestry rather than people;
- malformed, duplicate-trigger, trailing, non-monotonic and over-deep escalation definitions fail closed;
- no escalation table, routing engine, worker stack or policy engine was introduced.

### EIA-1 — implemented on PR #59

- native OIDC Authorization Code + state + nonce + PKCE S256;
- immutable tenant-bound issuer + subject → local principal mapping;
- one federated identity maps to a principal, not a legal entity; legal-entity access is current ClearSight state;
- unknown subjects fail closed instead of privileged JIT provisioning;
- maintained SCS pgx server sessions; no OIDC token or cached role/capability truth in browser state;
- bounded trusted-origin callback/CORS/cross-origin protections;
- development and signed-gateway compatibility modes remain.

### EIA-2 — implemented on PR #59

- each OIDC application request re-resolves current principal/entity/access-source/role/capability state from PostgreSQL;
- IdP groups/roles/permissions are not ClearSight authorization truth;
- empty department roles are legal-entity-wide; non-empty roles remain exact department grants;
- client-supplied department scope cannot select authorization scope and no parent/child inheritance is inferred;
- principal/position/group/binding/role revocation takes effect without IdP reauthentication;
- material commands still pass object visibility + `commandauth.Guard` + current authority/delegation/segregation.

### EIA-3 — implemented on PR #59

- exact-pinned OSS `elimity-com/scim` is contained behind `internal/scimapi`;
- `/scim/v2` is an isolated machine edge with tenant-scoped bearer credentials stored only as SHA-256 digests;
- bounded SCIM discovery, Users/Groups CRUD, PATCH, filtering and pagination;
- SCIM User lifecycle creates and owns canonical `PERSON` principals without email-based takeover of unrelated principals;
- explicit source configuration may correlate `externalId`/`userName` to the EIA-1 OIDC issuer/subject mapping;
- direct group membership only; nested groups are rejected and no transitive closure table exists;
- SCIM cannot write group→role bindings;
- ClearSight-owned effective-dated group→existing-role bindings use the same EIA-2 capability evaluator;
- directory state cannot manufacture responsibility, authority grants, delegation, signatory authority, segregation bypasses or protected-record visibility.

EIA-3 final head `3a8d54057551b435bf7cd10ffa5d421d3a899eff` passed CI #840 and UI evidence #406.

### EIA-4 — executable OVERDUE multi-level escalation implemented on PR #59

EIA-4 reuses the existing runtime and authority topology rather than introducing a separate escalation stack:

```text
overdue Matter Workflow task
→ pinned routing-policy escalation sequence
→ one next workflow_timers row
→ existing timer claim/fire
→ WorkflowTimerFired outbox event
→ idempotent escalation consumer / existing inbox receipt
→ current Matter + task authorization facts
→ current authority + delegation + grants + segregation + visibility
→ exact department boundary where configured
→ same Workflow task ESCALATED overlay
→ next level only
```

Implemented invariants:

- **no new durable table**; migration `000027` only adds a small projection-preservation trigger/function to `workflow_tasks`;
- only `OVERDUE` is executable now; `NO_ROUTE`, `AUTHORITY_INSUFFICIENT`, `MATERIALITY_INCREASE`, `RECIPIENT_UNAVAILABLE`, and `CONFLICT` remain schema-valid until real domain event adapters exist;
- one escalation sequence per trigger removes runtime ambiguity;
- the timer stores lineage only: task/workflow, policy version, sequence/level and baseline due date;
- each fired level re-reads current Matter ID, legal entity, decision type, materiality, visibility and authority state rather than trusting stale timer authorization facts;
- the triggering task's current department is resolved when level 0 fires and pinned as the ancestry base for that escalation lineage;
- existing authority resolution runs first, including delegation/grant/segregation; exact department filtering then narrows its already-eligible candidate set;
- automatic escalation requires exactly one visible candidate; no-route/ambiguity/multi-candidate states emit `WORK_ESCALATION_UNRESOLVED` and never fall back to an administrator;
- successful levels emit `WORK_ESCALATED` and update the existing Workflow task rather than creating a second task model;
- outbox replay is idempotent through the existing inbox ledger and per-level attempt key;
- a due-date/policy/source race fails closed and cannot enqueue a stale next level;
- ordinary lifecycle reconciliation cannot erase an active escalation overlay when the canonical requirement, due date and policy are unchanged; genuine source changes can replace it;
- completion/cancellation cancels the pending next escalation timer;
- escalation assignment is work-routing projection only; executing a material command still requires normal current command authority.

The hardened executable head `ab52d798615376d0d5c50377b6608270b8aaa086` passed merge-context CI #855 including race-enabled tests, PostgreSQL composition, migration `000027` apply/rollback/reapply, adversarial child-department → parent-department → CRO escalation, replay idempotency, projection-overlay preservation, terminal cancellation and `go vet`.

### Next

1. **EIA-5:** compact Identity & Access Configure surface, source-token/bootstrap/rotation administration, group-role mapping administration, escalation simulation/status, and reference authentik deployment guidance.
2. Add non-OVERDUE trigger adapters only when their canonical domain event source is defined; do not fabricate generic trigger polling.

## 3. Later productization still required

### Capture / Imports

- provenance classes for materially sourced, prefilled and respondent-entered values;
- draft/resume and amendment/supersession where durable semantics exist;
- production malware/content inspection, quarantine and retry;
- governed multi-file collection only when the request contract requires it;
- recurring mapping/schema-change detection and governed canonical conversion;
- PDF/OCR provider isolation when introduced.

### Configure / enterprise administration

- EIA-5 compact identity/access administration and escalation simulation;
- responsibility and decision-authority matrices where current backend configuration is not operable from UI;
- governed delegation/substitution/absence;
- maker-checker, effective dating, impact preview and rollback;
- security/session/notification/integration policy surfaces tied to real backend capability.

### Enterprise shell / acceptance

- production scoped Explore/reconstruction without restricted-existence leakage;
- actor-scoped notification centre with exact-action launch;
- representative bank-user timed usability;
- real browser 200% zoom/reflow and assistive-technology review;
- production-scale resilience/security/backup/restore evidence;
- pilot CRO/CCO/CISO, owner, reviewer, signatory and evidence-respondent validation.

## 4. Canonical invariants

- Program = ongoing obligation/compliance continuity.
- Matter = bounded change, exception, finding, decision, action, response or verification case.
- Matter Action ≠ Workflow Task.
- Signal ≠ conclusion.
- Submission ≠ sufficient evidence.
- Implementation ≠ verified outcome.
- Recommendation ≠ approval.
- Automation Policy ≠ execution receipt.
- WorkRequirement ≠ authoritative state.
- WorkAmbiguity ≠ actor assignment.
- Lifecycle sequence policy selects responsibility, not outcome or actor.
- Lifecycle sequence rule ≠ authority route.
- Escalation sequence selects responsibility + scope, not person or bypass route.
- Escalation assignment ≠ material command authority.
- Escalation timer lineage ≠ authorization truth; current facts are re-read when a level fires.
- Department scope narrows eligibility; it never broadens tenant/legal-entity/object visibility.
- Department-scoped role/capability ≠ legal-entity-wide role/capability.
- Client-supplied department scope ≠ authorization scope.
- Directory group membership ≠ role mapping, responsibility or material authority.
- Directory group → role binding ≠ responsibility or material authority.
- SCIM lifecycle ≠ IAM administration inside ClearSight.
- OIDC assertion ≠ ClearSight role/capability authority.
- OIDC correlation ≠ legal-entity authorization; current ClearSight state decides eligible context.
- Evidence Request recipient is canonical request state; Workflow work is rebuildable actor projection.
- Workflow command packet is an executable projection; mutations are revalidated by domain service.
- Program review checkpoint = actor acknowledgement of exact canonical versions, not Program state or approval.
- Protected Matter visibility must fail closed before actor-facing search/pagination/limit.
- Saved Work view ≠ assignment or authorization truth.
- Schema/spec existence ≠ capability.

Do not add parallel authorization, task, workflow, event, worker, receipt, review, preference, document or dashboard stacks that duplicate existing foundations.

## 5. Current executable flow truth

```text
enterprise sign-in
OIDC issuer + subject
→ provisioned local principal
→ selected legal entity
→ current position or governed directory group-role binding proves eligibility
→ server session
→ current local roles/capabilities resolved each request
→ identity middleware
→ /api/v1 route permission / object visibility / command authority

enterprise provisioning
SCIM bearer source
→ source-owned User / principal
→ direct Group membership
→ explicit ClearSight group → existing role binding
→ legal-entity / exact-department capability eligibility
→ normal OIDC/session application path

OVERDUE Matter work
canonical task due date + pinned policy version
→ next existing workflow timer only
→ timer fire / outbox / inbox-idempotent escalation consumer
→ current Matter legal entity + materiality + decision type + visibility
→ current authority/delegation/grant/segregation
→ exact department ancestry boundary where configured
→ same Workflow task escalation overlay
→ next timer only
→ material action still requires current command authority

Matter
canonical state + visibility
→ deterministic work compiler / Action
→ current authority/accountable owner
→ existing Workflow projection
→ Today / Work
→ governed domain command
→ authoritative Matter aggregate
→ projection converges

Program
canonical current state
→ actor review checkpoint
→ exact baseline + bounded post-baseline history
→ review-by-exception digest
→ actor acknowledges exact current versions

Evidence Request
canonical recipient + subject visibility
→ recipient lifecycle
→ rebuildable Workflow projection
→ Today / Capture
```

Presentation/projection/session/provisioning/escalation state never substitutes for canonical domain or material authority truth.

## 6. Release gates

A tranche is not complete until relevant gates pass on its **exact final head**:

- `gofmt` and `go vet`;
- race-enabled Go tests;
- PostgreSQL composition, migrations and latest rollback/reapply;
- serialized PostgreSQL integration/adversarial tests;
- TypeScript strict checking;
- rendered-state/axe tests;
- production Vite build;
- deterministic UI evidence when a visual/user-flow surface changes;
- identity/tenant/authority/degraded/replay tests where relevant;
- representative query/performance/recovery proof when cardinality or durability changes.

Never claim a branch or PR is green from an older commit.
