# Enterprise identity and access

**Status:** EIA-0 through EIA-5 implemented on PR #59  
**Scope:** enterprise sign-in, provisioning, department-aware capability eligibility, governed escalation, and compact administration  
**Supersedes:** greenfield LDAP/SAML portions of the older enterprise-productization P2/P3 plan

## 1. Architecture decision

ClearSight does not become an IAM product.

```text
authentication      OIDC
provisioning        SCIM 2.0
session             server-side ClearSight session
coarse access       existing role templates + local capabilities
governed actions    existing responsibility / authority / delegation / segregation
legacy federation   upstream OSS IAM bridge
```

Reference bridge: **authentik**. Existing bank Keycloak, Entra, Okta, or another standards-compliant OIDC provider may connect directly.

ClearSight does **not** implement LDAP/Active Directory, SAML XML, Kerberos, passwords, MFA enrollment, passkeys, account recovery, or a generic directory console.

An ordinary authenticated read or command must make zero LDAP/SAML/SCIM/IdP network calls for authorization.

## 2. Foundations reused

Do not replace:

- `principals`;
- `org_positions`;
- `role_templates` and `position_role_bindings`;
- `responsibility_assignments`;
- `authority_grants`;
- `routing_policies` / `routing_policy_versions`;
- `effective_authority_routes`;
- delegations and segregation rules;
- `workflow_timers`, outbox/inbox, and the existing worker;
- Workflow Tasks;
- `governance_decisions`;
- `internal/identity`, `internal/authority`, and `internal/commandauth`.

Identity establishes a local principal and broad product eligibility. It never substitutes for current object visibility or material command authority.

## 3. Department model

Departments remain hierarchical stable codes, not a second organization subsystem:

```text
["BANK", "OPERATIONS", "PAYMENTS"]
```

Rules:

- empty path = legal-entity scope;
- non-empty path = exact department scope;
- path prefixes express hierarchy for governed escalation only;
- capability checks do not infer parent/child inheritance;
- client-supplied department scope never selects authorization scope.

## 4. Capability model

`role_templates.capabilities` owns bounded coarse capabilities. Position-role bindings and explicitly governed directory-group-role bindings are the only current role sources.

```text
empty department path role
→ actor.role_codes + actor.permission_codes
→ may satisfy legal-entity-wide route permissions

non-empty department path role
→ actor.department_grants[path]
→ exact department eligibility only
```

Current identity-administration capabilities are:

- `IDENTITY_READ` — inspect sign-in/provisioning, bounded people/groups, mappings and escalation health/preview;
- `IDENTITY_CONFIGURE` — create/rotate/revoke SCIM sources and create/retire group-role mappings.

Capability answers **which product function may this actor use?** Existing authority still answers **may this actor perform this governed action on this exact object now?**

## 5. Multi-level escalation

Escalation sequences remain inside versioned, maker-checker governed routing-policy definitions. No escalation-policy table or second scheduler exists.

Example:

```json
{
  "escalations": [
    {
      "id": "overdue-control-work",
      "trigger": "OVERDUE",
      "steps": [
        {"after": "0s", "responsibility": "ACCOUNTABLE_OWNER", "department_levels_up": 0},
        {"after": "4h", "responsibility": "ESCALATION_OWNER", "department_levels_up": 1},
        {"after": "24h", "responsibility": "AUTHORIZER"}
      ]
    }
  ]
}
```

Semantics:

- array order is escalation order;
- `department_levels_up: 0` = starting department;
- `1` = parent, `2` = grandparent, etc.;
- omitted level = legal-entity routing scope;
- each level selects responsibility + scope, not a person;
- the existing authority resolver chooses current candidates;
- only the next level is scheduled in `workflow_timers`;
- current authority/delegation/grant/segregation/visibility is re-read when a level fires;
- assignment to escalated work is not material command authority.

`OVERDUE` is executable. `NO_ROUTE`, `AUTHORITY_INSUFFICIENT`, `MATERIALITY_INCREASE`, `RECIPIENT_UNAVAILABLE`, and `CONFLICT` remain schema-valid until their canonical domain event source exists.

## 6. Open-source boundary

Native libraries:

- OIDC: `github.com/coreos/go-oidc/v3/oidc` + `golang.org/x/oauth2`;
- sessions: `github.com/alexedwards/scs/v2` + `github.com/alexedwards/scs/pgxstore`;
- SCIM transport: exact-pinned `github.com/elimity-com/scim` behind `internal/scimapi`.

Reference legacy bridge:

- `deploy/reference-identity/authentik/README.md` documents AD/LDAP/SAML → authentik → OIDC + SCIM → ClearSight.

Do not add Casbin, OPA, Cerbos, another policy DB, another workflow engine, another scheduler, or Redis just for this work.

## 7. Implemented tranches

### EIA-0 — department and escalation contract

- `org_positions.department_path`;
- `role_templates.capabilities`;
- strict bounded multi-level escalation parser;
- duplicate trigger sequences, trailing JSON, invalid responsibilities, non-monotonic thresholds and excessive hierarchy fail closed.

### EIA-1 — OIDC + server session

- Authorization Code + state + nonce + PKCE S256;
- immutable tenant-bound issuer + subject → local principal;
- unknown subjects fail closed;
- one identity maps to a principal, not a legal entity;
- SCS pgx server sessions;
- no access/refresh token or cached authorization truth in browser session state;
- trusted-origin redirect/CORS/cross-origin controls;
- signed-gateway and development compatibility modes retained.

### EIA-2 — local department-aware capability resolver

- every OIDC request re-resolves current local access state from PostgreSQL;
- IdP groups/roles/permissions are never ClearSight authorization truth;
- global and exact-department grants remain distinct;
- position/group/role/principal revocation changes access without IdP reauthentication;
- material commands remain behind current visibility + command guard + authority/delegation/segregation.

### EIA-3 — SCIM Users + Groups

- isolated `/scim/v2` machine edge;
- tenant-scoped bearer token digests;
- bounded Users/Groups CRUD, PATCH, filtering and pagination;
- SCIM users create source-owned canonical `PERSON` principals;
- no email-based takeover of existing manually governed principals;
- direct group membership only; nested-group closure is not materialized;
- SCIM cannot write group-role mappings;
- explicit ClearSight group → existing role bindings feed the EIA-2 resolver.

### EIA-4 — executable OVERDUE escalation

- no new durable escalation table;
- existing timer → outbox → inbox-idempotent consumer path;
- same Workflow Task carries the escalation overlay;
- exact department ancestry can narrow already-eligible authority candidates;
- no-route/ambiguity/multi-candidate/hidden states fail closed;
- replay is idempotent;
- completion/cancellation cancels the pending next level;
- timer stores lineage, not stale authorization truth.

EIA-5 review tightened the EIA-4 overlay invariant further: unchanged reconciliation may preserve an active escalation only when the canonical decision type, materiality, command, target status, allowed targets and sequence policy also remain unchanged. A material routing change therefore clears the old overlay.

### EIA-5 — compact Identity & Access administration

Implemented inside the existing Configure workspace, not as a new IAM shell.

**Sign-in & provisioning**

- displays current identity mode, OIDC issuer and assurance context;
- lists SCIM sources with active user/group counts and current source state;
- creates a source with a high-entropy reveal-once token;
- rotates a token, immediately invalidating the prior credential;
- revokes a source without deleting historical principals;
- plaintext SCIM tokens are never persisted.

**Group → role mappings**

- lists current effective mappings for the active legal entity;
- creates an effective-dated mapping to an existing role template;
- optional exact department path;
- retires rather than destroys mapping history;
- duplicate current mappings are prevented by migration `000028`;
- mappings never create responsibility or material authority.

**People & groups**

- bounded read-only inspection only;
- deliberately not a directory CRUD clone.

**Escalation runtime**

- current escalated task count;
- pending escalation timer count;
- unresolved events in the last 24h;
- failed escalation timer count;
- policy/sequence hierarchy preview using the same department ancestry function as runtime;
- preview never pretends to know the future actor: current authority is resolved only when a level fires.

**Administration history**

- source creation/token rotation/revocation and group-role binding/retirement reuse the existing append-only `governance_decisions` ledger;
- migration `000028` expands only its bounded object-type constraint and adds mapping uniqueness; it creates no table.

**Canonical route inventory**

Seven EIA-5 routes live in the existing `internal/httpapi/route_registry.go` and are projected into `api/runtime.openapi.json`. No parallel route registry exists.

## 8. Bugs closed during the final review

1. **Revoked SCIM source could retain group-derived access.** Local access resolution now requires the source to remain `ACTIVE` and the group to belong to the same source. Source revocation removes source-derived eligibility on the next request while preserving the principal record.
2. **Escalation overlay could survive a material routing change.** The preservation invariant now compares materiality/decision/command/target/sequence semantics as well as due date/work key/policy.
3. **Mutation error handling could submit a browser mutation twice.** The Configure client now uses a single request path for no-content mutations.
4. **Identity routes initially formed a parallel mini registry.** They were consolidated into the canonical route registry and runtime OpenAPI projection.
5. **Identity administration initially targeted removed generic `audit_events`.** It now reuses `governance_decisions`; the removed compatibility table remains absent.
6. **Shared escalation test fixture could leak into another integration test.** The overlay regression is transaction-local and rolls back.

## 9. Acceptance gates

### Identity and provisioning

- wrong issuer/audience/state/nonce/PKCE fails closed;
- unknown OIDC subject is not silently provisioned;
- disabled principal loses access on the next application request;
- source revocation disables provisioning authentication and group-derived access without deleting principal history;
- token rotation invalidates the old SCIM credential;
- tokens are reveal-once and stored only as digests;
- direct group membership cannot create material authority;
- duplicate active group-role mappings fail deterministically;
- `IDENTITY_READ` cannot mutate identity configuration;
- `IDENTITY_CONFIGURE` is required for source and mapping writes.

### Departments and capabilities

- global and exact-department grants remain distinct;
- no inferred parent/child capability inheritance;
- source/group/binding/role/principal revocation changes current access immediately from local state;
- department capability cannot satisfy a legal-entity-wide route unless explicitly granted at legal-entity scope.

### Escalation

- levels execute once and in order for `OVERDUE`;
- department traversal fails closed beyond available hierarchy;
- no-route/conflict cannot silently assign an administrator;
- current authority/delegation/grant/segregation/visibility is applied when the level fires;
- material source changes clear stale escalation overlays;
- completion/cancellation cancels the pending next level.

### UI

- Configure handles loading, restricted and unavailable identity states independently from automation policy state;
- token is visibly labeled reveal-once;
- source revocation warns that source-derived access will stop;
- group mappings state that material authority is unchanged;
- escalation preview distinguishes department, legal-entity and out-of-range scope;
- narrow viewport remains usable without creating another navigation hierarchy.

## 10. Remaining work outside EIA-0…5

The enterprise identity/access tranche is complete. Remaining productization should be driven by real customer/use-case evidence, especially:

- non-OVERDUE escalation trigger adapters when canonical domain events exist;
- broader Configure surfaces for existing decision-authority/delegation capabilities where required;
- representative bank-user usability and production resilience/security evidence.

Do not reopen LDAP/SAML/password/MFA/directory-platform work inside ClearSight unless a concrete deployment cannot use the standards boundary above.
