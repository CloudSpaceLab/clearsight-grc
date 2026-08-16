# ClearSight implementation ledger

**Status date:** 2026-08-16
**Current execution issue:** #61 — AI governance gateway
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
| Enterprise identity/access EIA-0…5 | PR #59 |
| Reusable connected-source T0…T2 | issue #61 through PR #70 |
| Stateless AI gateway transport T3 | issue #61; `cmd/ai-gateway` and `internal/aigateway` |


## 2. AI governance gateway — T3 transport implemented

The repository now has an isolated stateless gateway process with OpenAI-compatible Chat Completions and Responses ingress, truthful SSE translation, OpenAI and Anthropic pilot adapters, SHA-256 workload authentication, model aliases, weighted routing, pre-output fallback, route circuit breaking, request/token/cost/concurrency budgets and content-free logs/metrics.

T3 deliberately owns no durable table and makes no governance decision. T4 remains the current next scope: governed AI workload registration, Automation Policy lifecycle, deterministic decisions/obligations, reusable Source Binding resolution and maker-checker-controlled shadow/enforcement activation. T5 remains durable receipts, response controls and approval/execution grants. See [`architecture/ai-gateway-transport.md`](architecture/ai-gateway-transport.md).

## 3. Enterprise identity/access — implemented on PR #59

The completed architecture is intentionally smaller than the earlier greenfield P2/P3 proposal:

```text
OIDC sign-in
→ local principal
→ current position and/or governed directory group-role binding
→ legal-entity-wide or exact-department capability eligibility
→ existing route registry / visibility / command authority

SCIM provisioning
→ source-owned User + direct Group facts
→ explicit ClearSight group → existing role binding
→ same local capability resolver

OVERDUE escalation
→ existing routing-policy sequence
→ next workflow timer only
→ current authority/delegation/grant/segregation/visibility
→ optional current source-role + target role/group guard
→ same Workflow Task escalation overlay
```

LDAP/AD/SAML/Kerberos/password/MFA infrastructure remains upstream in authentik, an existing bank Keycloak deployment, or another standards-compliant identity layer.

### EIA-0 — department and escalation contract

- hierarchical `org_positions.department_path` without a department subsystem;
- bounded `role_templates.capabilities`;
- strict multi-level escalation definitions inside existing routing-policy versions;
- duplicate triggers, trailing/unknown JSON, invalid responsibility, excessive depth and non-monotonic thresholds fail closed.

### EIA-1 — OIDC + server session

- Authorization Code + state + nonce + PKCE S256;
- immutable tenant-bound issuer + subject → principal correlation;
- unknown subjects fail closed;
- one identity maps to one principal rather than one legal entity;
- maintained SCS pgx server sessions;
- no OIDC token or cached authorization truth in browser session state;
- trusted-origin redirect/CORS/cross-origin protections;
- signed-gateway and development compatibility retained.

### EIA-2 — local capability evaluation

- each OIDC request re-resolves current local access state from PostgreSQL;
- IdP group/role/permission claims are not ClearSight authorization truth;
- empty department path = legal-entity-wide capability;
- non-empty path = exact-department grant;
- no inferred parent/child capability inheritance;
- current principal/position/source/group/binding/role changes take effect without IdP reauthentication;
- material commands still require current visibility + `commandauth.Guard` + authority/delegation/segregation.

### EIA-3 — SCIM Users + Groups

- exact-pinned OSS SCIM transport behind `internal/scimapi`;
- isolated `/scim/v2` machine edge with tenant-scoped bearer digests;
- bounded Users/Groups CRUD, PATCH, filtering and pagination;
- source-owned canonical `PERSON` principals;
- no mutable-email takeover of existing principals;
- direct same-source group membership only;
- SCIM cannot write group-role mappings;
- explicit ClearSight group → existing role bindings feed the same EIA-2 resolver.

### EIA-4 — executable multi-level OVERDUE escalation

- no new escalation table, scheduler or task model;
- existing timer → outbox → inbox-idempotent consumer path;
- current Matter/legal entity/materiality/decision type/visibility/authority is re-read when a level fires;
- exact department ancestry narrows already-eligible authority candidates;
- optional `source_roles` validates the current originating task principal;
- optional `targets.roles` / `targets.groups` further narrow already-authorized target candidates with OR semantics;
- directory-group guards use current direct membership and require an active SCIM source;
- target-role guards use current effective position-role or governed group-role state;
- guards never manufacture responsibility, authority or protected-record visibility;
- no-route/ambiguity/multi-candidate/hidden/source-role/target-guard mismatches fail closed;
- replay is idempotent;
- only the next level is scheduled;
- completion/cancellation cancels pending escalation;
- escalation assignment remains work routing, not material command authority.

`OVERDUE` is executable. Other schema-valid triggers remain deferred until their canonical domain event source exists.

### EIA-5 — compact Identity & Access Configure surface

Implemented inside the existing Configure workspace rather than creating a second IAM application.

- `IDENTITY_READ` and `IDENTITY_CONFIGURE` are explicit route capabilities;
- current sign-in mode/issuer/assurance is visible;
- SCIM source state and bounded user/group counts are visible;
- source create/rotate returns a reveal-once high-entropy token while storing only its digest;
- source revoke disables provisioning and source-derived group access on the next request without deleting principal history;
- bounded read-only People & Groups inspection;
- effective-dated group → existing role mappings with optional exact department path;
- duplicate current mappings are prevented;
- mapping retirement preserves history;
- escalation runtime health shows active escalations, pending levels, recent unresolved events and failed timers;
- hierarchy/guard preview uses the same runtime semantics and never predicts the future actor;
- operators can select one escalation level, optional originating role(s), and allowed target role(s)/directory group(s);
- escalation guard changes require both `IDENTITY_CONFIGURE` and governance `CONFIG_WRITE`;
- saving a guard creates the next **unapproved** `routing_policy_versions` row rather than mutating the active policy or creating an override table;
- `routing_policies.current_version` remains on the approved version while a proposal is pending, so current routing stays live;
- the proposal maker can preview/update their proposal but cannot approve it;
- another maker cannot overwrite the pending proposal;
- a different authorized checker supplies rationale and activates the latest revision;
- approval revalidates the whole policy, existing authority conflicts and current role/group references;
- stale/missing roles or groups block activation without changing the approved current version;
- activation effective-dates the old version out and the new version in atomically and refreshes the existing effective authority projection;
- source/token/mapping and guard activation history reuse existing governed decision/outbox infrastructure;
- all nine EIA-5 staff routes live in the canonical `route_registry.go` and `api/runtime.openapi.json` projection;
- `deploy/reference-identity/authentik/README.md` documents the optional AD/LDAP/SAML → authentik → OIDC/SCIM bridge without bundling or forking authentik.

### Demo stakeholder login — non-production fixture

Demo mode no longer silently injects one configured actor. When `CLEARSIGHT_DEMO_MODE=true` with development identity mode:

- the login page supplies CRO, CCO/Compliance Officer, CISO, GRC Administrator, System Administrator, Internal Auditor, Program Owner and Evidence Respondent accounts;
- all supplied accounts use the explicit fixture password `demo`;
- a successful selection creates an HttpOnly, SameSite=Lax, HMAC-signed eight-hour demo session;
- the process-local signing key means restart invalidates existing demo sessions;
- tampered and expired sessions fail closed;
- a bounded bearer/capture token is never interpreted as staff identity;
- `Switch demo role` logs out and unmounts the entire application before another role is selected so role-scoped UI state cannot leak between identities;
- the three `/api/v1/demo/*` routes use the same canonical route registry with `DEMO_ONLY` classification and are absent when demo mode is off;
- production OIDC/signed authentication is unchanged and production configuration rejects demo mode.

The fixture contract is documented in [`engineering/demo-role-login.md`](engineering/demo-role-login.md). It is not an enterprise password-authentication subsystem.

### Final review bugs closed

- revoked SCIM source previously could retain group-derived access;
- EIA-4 escalation overlay could survive a material routing-semantic change;
- an early EIA-5 browser error path could resend a mutation;
- identity-admin routes were initially placed in a parallel mini-registry before being consolidated into the one canonical route inventory;
- an early EIA-5 audit implementation targeted deliberately removed `audit_events`; governed history now reuses `governance_decisions`;
- the overlay regression fixture was made transaction-local so it cannot contaminate the global escalation maintainer;
- role-only/group-only candidate guards originally encoded the absent selector side as JSON `null`; evaluation now normalizes absent selectors to empty arrays before PostgreSQL filtering;
- demo mode previously auto-authenticated one actor without a real role-selection/login flow; the top-level demo auth gate now requires an explicit supplied role and resets the whole app on role switch;
- demo auth endpoints were briefly implemented as a mini route registry during review and were consolidated into the canonical route inventory before merge.

The exact final PR head must pass the normal release gates below before merge; do not rely on an older green head after documentation or demo-login changes.

## 4. Productization still required outside the identity tranche

### Capture / Imports

- provenance classes for materially sourced, prefilled and respondent-entered values;
- draft/resume and amendment/supersession where durable semantics exist;
- production malware/content inspection, quarantine and retry;
- governed multi-file collection only when the request contract requires it;
- recurring mapping/schema-change detection and governed canonical conversion;
- PDF/OCR provider isolation when introduced.

### Configure / enterprise administration

- responsibility and decision-authority matrices where existing backend configuration is not yet operable from the UI;
- governed delegation/substitution/absence where the current product surface is insufficient;
- security/session/notification/integration policy surfaces tied to real backend capability.

Do not reopen generic IAM/directory scope merely because adjacent enterprise administration remains.

### Enterprise shell / acceptance

- production scoped Explore/reconstruction without restricted-existence leakage;
- actor-scoped notification centre with exact-action launch;
- representative bank-user timed usability;
- real browser 200% zoom/reflow and assistive-technology review;
- production-scale resilience/security/backup/restore evidence;
- pilot CRO/CCO/CISO, owner, reviewer, signatory and evidence-respondent validation.

### Escalation adapters

Add `NO_ROUTE`, `AUTHORITY_INSUFFICIENT`, `MATERIALITY_INCREASE`, `RECIPIENT_UNAVAILABLE`, or `CONFLICT` runtime adapters only when the corresponding canonical domain event/timestamp exists. Do not fabricate generic trigger polling.

## 5. Canonical invariants

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
- Escalation sequence selects responsibility + scope; optional role/group guards only narrow current candidates.
- Escalation candidate guard ≠ authority grant.
- Target role/group guard = OR filter over candidates already admitted by authority/visibility/department constraints.
- Pending routing-policy revision ≠ active routing truth.
- Escalation assignment ≠ material command authority.
- Escalation timer lineage ≠ authorization truth.
- Department scope narrows eligibility; it never broadens tenant/legal-entity/object visibility.
- Department-scoped role/capability ≠ legal-entity-wide role/capability.
- Client-supplied department scope ≠ authorization scope.
- Directory group membership ≠ role mapping, responsibility or material authority.
- Directory group → role binding ≠ responsibility or material authority.
- SCIM source ACTIVE state is part of source-derived eligibility.
- SCIM lifecycle ≠ generic IAM administration inside ClearSight.
- OIDC assertion ≠ ClearSight role/capability authority.
- OIDC correlation ≠ legal-entity authorization.
- Demo role credential ≠ production authentication mechanism.
- Identity configuration history belongs to governed decision history, not a resurrected generic audit table.
- Evidence Request recipient is canonical request state; Workflow work is rebuildable actor projection.
- Workflow command packet is an executable projection; mutations are revalidated by domain service.
- Program review checkpoint = actor acknowledgement, not Program state or approval.
- Protected Matter visibility must fail closed before actor-facing search/pagination/limit.
- Saved Work view ≠ assignment or authorization truth.
- Schema/spec existence ≠ capability.

Do not add parallel authorization, task, workflow, event, worker, receipt, review, preference, document, directory or dashboard stacks that duplicate existing foundations.

## 6. Current executable flow truth

```text
enterprise sign-in
OIDC issuer + subject
→ provisioned local principal
→ selected legal entity
→ current position or ACTIVE-source governed group-role mapping proves eligibility
→ server session
→ current local roles/capabilities resolved each request
→ identity middleware
→ canonical route permission / object visibility / command authority

enterprise provisioning
SCIM bearer source
→ source-owned User / principal
→ direct Group membership
→ explicit ClearSight group → existing role binding
→ legal-entity / exact-department capability eligibility
→ normal OIDC/session application path

identity administration
IDENTITY_READ / IDENTITY_CONFIGURE
→ canonical route registry
→ source/token or group-role mutation
→ existing durable identity/access tables
→ governance_decisions history
→ current access resolver on next request

governed escalation guard change
IDENTITY_CONFIGURE + CONFIG_WRITE
→ select active policy / sequence / level
→ optional source roles + target roles/groups
→ create unapproved routing_policy_versions revision
→ approved current_version stays live
→ independent checker + rationale
→ revalidate policy / authority conflicts / current references
→ atomically activate new current_version

stakeholder demo login
DEMO_ONLY supplied role catalogue
→ explicit non-production credential
→ signed HttpOnly demo session
→ normal identity middleware / role-scoped demo work
→ switch role logs out and remounts application

OVERDUE Matter work
canonical due date + pinned escalation lineage
→ next workflow timer only
→ timer fire / outbox / inbox-idempotent consumer
→ current source-role qualification
→ current legal entity + materiality + decision + visibility
→ current authority/delegation/grant/segregation
→ exact department ancestry boundary where configured
→ optional current target role/group guard
→ same Workflow task escalation overlay or unresolved receipt
→ next timer only
→ material action still requires current command authority
```

Presentation/projection/session/provisioning/escalation/admin/demo state never substitutes for canonical domain or material authority truth.

## 7. Release gates

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