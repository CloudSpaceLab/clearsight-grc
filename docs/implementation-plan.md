# ClearSight implementation ledger

**Status date:** 2026-08-10  
**Current execution issue:** #27  
**Umbrella pilot/GA catalogue:** #13

This file is the authoritative implementation ledger. Code, migrations and executable tests remain the final capability truth. Completed detail should live in focused architecture documents, PRs and tests rather than being copied into new planning frameworks.

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

Selected from a fresh merged-code audit rather than backlog order.

- canonical `MatterVisibleTo` / `ParseMatterAccessPolicy` remains the policy reference;
- PostgreSQL Matter summary/search reads now fail closed on malformed access metadata before cursor/limit;
- `RESTRICTED` requires a string-only, non-empty allow-list and exact current principal membership;
- verified actor tenant must match the tenant being listed when actor context is present;
- production `CurrentPostgresRepository.ListMatters` applies actor visibility before `LIMIT` rather than relying on post-limit HTTP filtering;
- in-memory Matter lists apply the same canonical visibility before sorting/limit;
- internal worker/reconciliation reads without actor context keep their existing tenant-scoped behavior;
- hidden/malformed Matters therefore cannot consume page slots, alter cursors or appear in search for an unauthorized actor;
- a Program status-reason wording change now produces an `EXPLANATION` review delta without manufacturing a new/resolved exception identity;
- no RLS layer, visibility table, policy engine, preference system, task model or new authorization framework was introduced;
- exact final head `e245754bebea013475499a7fbdb0f6da0db62032` passed CI #717;
- squash-merged as `e9e61cafa5d6715b3e94bd72454b58b3ead87ff4`.

## 2. Current execution — enterprise identity/access

PR #59 implements the focused OSS-first identity/access boundary from `docs/engineering/enterprise-identity-access.md`.

The architectural rule remains narrow: ClearSight owns OIDC, server sessions, local capabilities, SCIM provisioning and integration with existing authority. LDAP/AD/SAML/Kerberos/password/MFA infrastructure remains upstream in authentik, an existing bank Keycloak deployment, or another standards-compliant identity layer.

### EIA-0 — implemented on PR #59

- `org_positions.department_path` provides stable hierarchical department scope without a department subsystem;
- `role_templates.capabilities` provides bounded coarse application eligibility;
- routing-policy definitions accept strict bounded multi-level escalation sequences that select responsibility + department ancestry rather than hard-coded people;
- policy validation rejects malformed, unknown, trailing, non-monotonic and over-deep escalation definitions;
- no new escalation table, routing engine, worker or UI surface was introduced.

### EIA-1 — implemented on PR #59, exact-head validation pending

- native `oidc` identity mode using Authorization Code + state + nonce + PKCE S256;
- immutable tenant-bound issuer + subject mapping through `principal_identities`;
- unknown OIDC subjects are denied rather than silently privileged through JIT provisioning;
- server-side SCS sessions use the maintained pgx store and `web_sessions` infrastructure ledger;
- session token is renewed at login and roles/permissions are not stored as browser or OIDC token truth;
- callback returns only to the configured trusted application origin + bounded local path;
- credentialed browser requests are limited to the trusted origin and unsafe cross-origin requests use Go's cross-origin protection;
- development and signed-gateway compatibility modes remain available;
- the three `/auth/...` endpoints are a narrow protocol edge; `/api/v1` route/access truth remains the existing typed registry.

### EIA-2 — implemented on PR #59, exact-head validation pending

- each OIDC-authenticated application request re-resolves the active principal, legal entity, positions, role templates and capabilities from PostgreSQL;
- native OIDC never trusts IdP group/role/permission claims as ClearSight authorization truth;
- empty department path roles become global actor roles/capabilities and can satisfy existing tenant/legal-entity-wide route permissions;
- non-empty department roles/capabilities remain exact-scope `department_grants` and cannot satisfy existing global route permissions;
- no client-supplied department path can select authorization scope;
- parent/child department capability inheritance is not inferred;
- principal deactivation or current role removal changes access without re-authenticating at the IdP;
- material commands continue through existing object visibility, lifecycle-specific `commandauth.Guard`, authority resolution, delegation and segregation.

Do not mark EIA-1/EIA-2 complete until the current exact PR head passes the normal release gates.

### Next after EIA-1/EIA-2

1. **EIA-3:** minimal SCIM Users/Groups + explicit approved group-to-role mapping;
2. **EIA-4:** escalation execution over existing routing policy, authority resolver and `workflow_timers`;
3. **EIA-5:** compact Identity & access Configure surface + reference authentik deployment guidance.

Do not represent policy-schema support as executable escalation until EIA-4 exists.

## 3. Later productization still required

### Capture / Imports

- provenance classes for materially sourced, prefilled and respondent-entered values;
- draft/resume and amendment/supersession where durable semantics exist;
- production malware/content inspection, quarantine and retry;
- governed multi-file collection only when the request contract requires it;
- recurring mapping/schema-change detection and governed canonical conversion;
- PDF/OCR provider isolation when introduced.

### Configure / enterprise administration

- EIA-3–EIA-5 provisioning, group mapping, escalation runtime and compact administration;
- responsibility and decision-authority matrices where current backend configuration is not yet operable from the UI;
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
- Escalation sequence selects responsibility + scope, not a person or bypass route.
- Department scope narrows eligibility; it never broadens tenant/legal-entity/object visibility.
- Department-scoped role/capability ≠ tenant-wide role/capability.
- Client-supplied department scope ≠ authorization scope.
- Directory group membership ≠ responsibility or material authority.
- OIDC identity assertion ≠ ClearSight role/capability authority.
- Evidence Request recipient is canonical request state; Workflow work is a rebuildable actor projection.
- Workflow command packet is an executable projection; every mutation is revalidated by the domain service.
- Program UI lifecycle choices are affordances only; server lifecycle/authority/version checks remain final.
- Program review checkpoint = actor acknowledgement of exact canonical versions; it is not Program state or approval.
- Program review digest = bounded presentation; full continuity history remains authoritative.
- Protected Matter visibility must fail closed before actor-facing search/pagination/limit.
- Saved Work view ≠ assignment or authorization truth.
- Schema/spec existence ≠ capability.

Do not add parallel authorization, task, workflow, event, worker, receipt, review, preference, document or dashboard stacks that duplicate existing foundations.

## 5. Current executable flow truth

```text
enterprise sign-in
OIDC issuer + subject
→ provisioned local principal identity
→ server-side session
→ current local roles/capabilities resolved on each request
→ existing identity middleware
→ existing /api/v1 route permission / object visibility / command authority

Matter
canonical state + canonical visibility
→ deterministic work compiler / canonical Action
→ current authority or accountable owner
→ existing Workflow projection
→ Today / Work
→ governed domain command
→ authoritative Matter aggregate
→ projection converges

Program
canonical current state
→ actor review checkpoint
→ exact baseline projection + bounded post-baseline history
→ review-by-exception digest
→ actor acknowledges exact current versions

Evidence Request
canonical recipient + subject visibility
→ recipient lifecycle
→ rebuildable Workflow projection
→ Today / Capture
```

Presentation/projection/acknowledgement/session state never substitutes for canonical domain or authority truth.

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
