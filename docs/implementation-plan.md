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

### EIA-1 — implemented on PR #59

- native `oidc` identity mode using Authorization Code + state + nonce + PKCE S256;
- immutable tenant-bound issuer + subject mapping through `principal_identities`;
- one federated identity maps to one local principal, not one legal entity; active legal-entity access is revalidated from current organization state;
- unknown OIDC subjects are denied rather than silently privileged through JIT provisioning;
- server-side SCS sessions use the maintained pgx store and `web_sessions` infrastructure ledger;
- session token is renewed at login and roles/permissions are not stored as browser or OIDC token truth;
- callback returns only to the configured trusted application origin + bounded local path;
- tenant, legal-entity, return-path and assurance values entering session state are bounded;
- credentialed browser requests are limited to the trusted origin and unsafe cross-origin requests use Go's cross-origin protection;
- development and signed-gateway compatibility modes remain available;
- the three `/auth/...` endpoints are a narrow protocol edge; `/api/v1` route/access truth remains the existing typed registry.

### EIA-2 — implemented on PR #59

- each OIDC-authenticated application request re-resolves the active principal, active legal entity, current access sources, role templates and capabilities from PostgreSQL;
- native OIDC never trusts IdP group/role/permission claims as ClearSight authorization truth;
- empty department path roles become legal-entity-wide actor roles/capabilities and can satisfy existing unscoped route permissions;
- non-empty department roles/capabilities remain exact-scope `department_grants` and cannot satisfy existing global route permissions;
- no client-supplied department path can select authorization scope;
- parent/child department capability inheritance is not inferred;
- principal deactivation, position removal, group membership/binding removal or current role removal changes access without re-authenticating at the IdP;
- the same principal may operate across multiple legal entities only where current organization/group-role state makes each entity eligible;
- material commands continue through existing object visibility, lifecycle-specific `commandauth.Guard`, authority resolution, delegation and segregation.

### EIA-3 — implemented on PR #59

- exact-pinned open-source `elimity-com/scim` is contained behind `internal/scimapi`; library types do not become ClearSight domain types;
- `/scim/v2` is a separate machine-to-machine edge authenticated by tenant-scoped bearer credentials stored only as SHA-256 digests;
- SCIM discovery, Users and Groups support bounded create/read/list/replace/delete/PATCH, filtering and pagination;
- SCIM User lifecycle creates and owns a canonical `PERSON` principal; it does not email-match and seize unrelated manually governed principals;
- an explicitly configured source may correlate `externalId` or `userName` to the same OIDC issuer/subject mapping used by EIA-1;
- deactivation/deletion inactivates the source-owned principal and removes/revokes current source eligibility without an ordinary-request directory lookup;
- directory groups and direct User membership are stored as source facts; nested groups are rejected and no transitive closure table was added;
- SCIM cannot write `directory_group_role_bindings`;
- an explicit ClearSight group→existing-role binding is effective-dated, legal-entity scoped and optionally exact-department scoped;
- group-derived role/capability eligibility is evaluated by the same EIA-2 resolver as position-derived eligibility;
- stale role templates, deleted groups, deactivated users and removed memberships/bindings fail closed on the next request;
- directory membership still cannot create responsibility, authority grants, delegation, segregation bypasses, signatory authority or protected-record visibility because the existing authority engine is unchanged;
- a dedicated HTTP test proves the SCIM machine edge bypasses browser identity middleware while human application traffic continues through the normal OIDC/session/permission/authority stack.

The executable EIA-3 code head `8d0f95303af6fd6a348e24e93b20ea8ed099a195` passed CI #836 and UI-evidence #402, including race-enabled tests, PostgreSQL composition, migration `000026` apply/rollback/reapply, serialized PostgreSQL integration, `go vet`, strict web checks and deterministic UI evidence. Final documentation/test-contract cleanup must also pass on the exact final head before merge.

### Next after EIA-3

1. **EIA-4:** multi-level escalation execution over existing routing policy, authority resolver and `workflow_timers`;
2. **EIA-5:** compact Identity & access Configure surface, source-token/bootstrap administration, group-role mapping administration and reference authentik deployment guidance.

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

- EIA-4/EIA-5 escalation runtime and compact identity administration;
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
- Department-scoped role/capability ≠ legal-entity-wide role/capability.
- Client-supplied department scope ≠ authorization scope.
- Directory group membership ≠ role mapping, responsibility or material authority.
- Directory group → role binding ≠ responsibility or material authority.
- SCIM User/Group lifecycle ≠ IAM administration inside ClearSight.
- OIDC identity assertion ≠ ClearSight role/capability authority.
- OIDC identity correlation ≠ legal-entity authorization; current ClearSight organization/group-role state determines eligible entity context.
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
→ selected legal-entity context
→ current ClearSight position or explicitly governed directory group-role binding proves entity eligibility
→ server-side session
→ current local roles/capabilities resolved on each request
→ existing identity middleware
→ existing /api/v1 route permission / object visibility / command authority

enterprise provisioning
SCIM bearer source
→ source-owned User / canonical principal
→ direct directory Group membership
→ explicit ClearSight group → existing role binding
→ legal-entity / exact-department capability eligibility
→ normal OIDC/session request path

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

Presentation/projection/acknowledgement/session/provisioning state never substitutes for canonical domain or material authority truth.

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
