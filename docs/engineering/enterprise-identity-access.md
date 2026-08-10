# Enterprise identity and access

**Status:** EIA-0 implemented; EIA-1/EIA-2 implemented on PR #59 pending exact-head validation  
**Scope:** enterprise sign-in, provisioning, department-aware role eligibility, and governed escalation  
**Supersedes:** the greenfield LDAP/SAML portions of the older enterprise-productization P2/P3 plan

## 1. Decision

ClearSight will not become an IAM product.

The native enterprise boundary is deliberately small:

```text
authentication      OIDC
provisioning        SCIM 2.0
session             server-side ClearSight session
coarse access       existing role templates + local capabilities
governed actions    existing responsibility / authority / delegation / segregation
legacy federation   upstream open-source IAM bridge
```

Reference open-source bridge: **authentik**. Existing bank Keycloak, Entra, Okta, or another standards-compliant OIDC provider connects directly. LDAP/Active Directory, SAML, Kerberos, passwords, MFA, passkeys, and recovery remain upstream identity-provider concerns.

ClearSight must never require an LDAP, SAML, SCIM, or IdP network call on an ordinary authenticated read or command path.

## 2. Existing foundations to reuse

Do not replace:

- `principals`;
- `org_positions` and `parent_position_id`;
- `role_templates` and `position_role_bindings`;
- `responsibility_assignments`;
- `authority_grants`;
- `routing_policies` / `routing_policy_versions`;
- `effective_authority_routes`;
- `delegations` and segregation checks;
- `workflow_timers`, outbox/inbox, and current worker infrastructure;
- `internal/identity`, `internal/authority`, and `internal/commandauth` trust boundaries.

Enterprise identity establishes the local principal and broad capability eligibility. It does not replace current object visibility or material command authority.

## 3. Department model without another organization subsystem

A separate department CRUD/domain model is not required for the pilot.

Departments are represented as a **stable hierarchical path of source-backed codes** on organizational positions and, when later needed, group role bindings:

```text
["BANK", "OPERATIONS", "PAYMENTS"]
```

Rules:

- path values are stable codes, not display names;
- an empty path means tenant/legal-entity scope rather than an invented department;
- a position may belong to one current department path;
- department hierarchy comes from path prefixes;
- directory synchronization may update paths, but ClearSight preserves historical command attribution through existing principal/event history;
- department capabilities require an exact department path unless a governed server-side policy explicitly defines inheritance.

This supports department-level access and escalation without adding `departments`, department-membership, or department-history tables before they are proven necessary.

## 4. Capability model

`role_templates` owns a bounded set of coarse capabilities. Initial families should remain small:

- `PROGRAM_READ`, `PROGRAM_CONFIGURE`;
- `MATTER_READ`, `MATTER_CREATE`;
- `EVIDENCE_READ`, `EVIDENCE_REQUEST`, `EVIDENCE_RESPOND`, `EVIDENCE_REVIEW`;
- `IMPORT_READ`, `IMPORT_CREATE`, `IMPORT_REVIEW`;
- `AUTHORITY_READ`, `AUTHORITY_CONFIGURE`;
- `IDENTITY_READ`, `IDENTITY_CONFIGURE`;
- `PLATFORM_OPERATIONS_READ`, `PLATFORM_OPERATIONS_WRITE`.

The first executable source is the existing position → role binding. Direct principal bindings are still deferred because there is no proven case that cannot be represented through current positions; approved directory-group bindings belong to EIA-3.

Effective access is deliberately split:

```text
empty department path role
→ actor.role_codes + actor.permission_codes
→ may satisfy existing tenant/legal-entity-wide route permissions

non-empty department path role
→ actor.department_grants[path]
→ may satisfy only a future route whose department scope is derived server-side
```

A client-supplied department path never selects authorization scope. Department scope narrows role eligibility; it never broadens tenant, legal-entity, object, or protected-record visibility.

A role/capability answers **which class of function may this actor use?** The existing authority resolver still answers **may this actor perform this governed action on this exact object now?**

## 5. Multi-level escalation

Do not add an escalation-policy table or a second routing engine.

Escalation sequences live inside the already versioned and maker-checker governed `routing_policy_versions.definition` alongside routing rules.

Canonical shape:

```json
{
  "rules": [
    {
      "id": "department-owner",
      "responsibility": "ACCOUNTABLE_OWNER",
      "selector": {"kind": "ROLE", "ref": "DEPARTMENT_OWNER"}
    },
    {
      "id": "risk-escalation",
      "responsibility": "ESCALATION_OWNER",
      "selector": {"kind": "ROLE", "ref": "RISK_MANAGER"}
    }
  ],
  "escalations": [
    {
      "id": "overdue-control-work",
      "trigger": "OVERDUE",
      "steps": [
        {"after": "0s", "responsibility": "ACCOUNTABLE_OWNER", "department_levels_up": 0},
        {"after": "4h", "responsibility": "ESCALATION_OWNER", "department_levels_up": 1},
        {"after": "24h", "responsibility": "ESCALATION_OWNER"}
      ]
    }
  ]
}
```

Semantics:

- array order is the escalation order; no redundant `level` field;
- `department_levels_up: 0` = current department;
- `1` = parent department; `2` = grandparent, and so on;
- omitted `department_levels_up` = legal-entity/tenant routing scope;
- each step selects a **responsibility and scope**, not a person;
- the existing authority/routing rules resolve the actor for that responsibility;
- `after` values are cumulative elapsed thresholds and must be monotonic;
- sequence execution reuses `workflow_timers`; it must not create another scheduler;
- reminder-only behavior stays separate from an ownership/authority escalation.

Initial triggers supported by the policy contract:

- `OVERDUE`;
- `NO_ROUTE`;
- `AUTHORITY_INSUFFICIENT`;
- `MATERIALITY_INCREASE`;
- `RECIPIENT_UNAVAILABLE`;
- `CONFLICT`.

Runtime support can be introduced trigger-by-trigger. Policy-schema support is not executable escalation until EIA-4 compiles it into existing timer/workflow infrastructure.

## 6. Open-source component boundary

### Native libraries

- OIDC relying party: `github.com/coreos/go-oidc/v3/oidc` + `golang.org/x/oauth2`;
- server-side sessions: `github.com/alexedwards/scs/v2` with `github.com/alexedwards/scs/pgxstore`;
- SCIM protocol adapter in EIA-3: `github.com/elimity-com/scim`, contained behind a ClearSight service adapter.

### External open-source IAM

**authentik** is the reference compatibility bridge for LDAP/AD/SAML to ClearSight OIDC + SCIM. **Keycloak** is a first-class direct OIDC compatibility target when already deployed by the bank. ClearSight does not depend on either vendor's internal APIs for authorization.

Do not add Casbin, OPA, Cerbos, another workflow engine, another event bus, or another policy database for this work.

## 7. Implemented tranches

### EIA-0 — shared access and escalation contract

Implemented on PR #59:

- `org_positions.department_path`;
- `role_templates.capabilities`;
- strict bounded parser/validation for multi-level `escalations` inside existing routing policy definitions;
- no department table, escalation table, authorization engine, worker, or UI surface.

### EIA-1 — OIDC + server session

Implemented on PR #59 pending exact-head validation:

- `CLEARSIGHT_IDENTITY_MODE=oidc` while retaining `development` and the signed-gateway compatibility mode;
- one operationally configured OIDC connection for the first tranche rather than premature connection-management CRUD;
- Authorization Code flow with state, nonce, and PKCE S256;
- exact OIDC issuer + subject correlation through tenant-bound `principal_identities`;
- no just-in-time privileged account creation: an unknown subject is denied until provisioned;
- SCS server-side sessions stored through the maintained pgx store;
- session fixation protection through token renewal at login;
- session cookie is `HttpOnly`, `SameSite=Lax`, and required `Secure` in production;
- the browser stores no OIDC access/refresh token and no authorization truth;
- existing signed identity and development modes remain intact;
- OIDC transport routes are the narrow `/auth/...` protocol edge; all application authorization continues through the existing identity middleware and `/api/v1` route registry;
- credentialed CORS is limited to the configured application origin, with Go standard-library cross-origin protection on unsafe requests;
- callback return paths are local paths joined only to that configured trusted application origin.

### EIA-2 — local department-aware capability resolver

Implemented on PR #59 pending exact-head validation:

- OIDC session state stores only canonical tenant/principal/legal-entity/session/assurance facts;
- each authenticated request re-resolves the principal and current roles/capabilities from PostgreSQL;
- deactivated/expired principals therefore lose current application access without an IdP/LDAP/SCIM call;
- current capabilities derive from existing `org_positions → position_role_bindings → role_templates` only;
- empty-department roles become global actor roles/capabilities;
- non-empty department roles/capabilities remain exact-scope `department_grants` and cannot satisfy existing global route permissions;
- parent/child department access is not inferred;
- native OIDC does not consume IdP role/group/permission claims as ClearSight authorization truth;
- existing material commands still additionally pass `commandauth.Guard`, current responsibility/authority resolution, delegation, segregation, visibility, and lifecycle checks.

## 8. Next sequence

### EIA-3 — SCIM Users + Groups

Expose the minimal SCIM 2.0 service-provider surface required for enterprise provisioning. Map Users into existing principals/`principal_identities`; map Groups into a narrow directory-group projection; support explicit approved group-to-role bindings.

A directory group may grant role eligibility/capabilities. It must never directly grant responsibility, decision authority, signatory authority, or protected-record visibility.

### EIA-4 — escalation runtime

Compile active routing-policy escalation sequences into the existing workflow timer infrastructure. For each fired level:

1. derive the applicable department prefix from current work scope;
2. resolve the step responsibility through current authority/routing policy;
3. apply delegation, segregation, principal activity and visibility checks;
4. update the existing Workflow projection or create a routing-integrity intervention;
5. schedule only the next level using the existing deduplicated timer ledger.

Do not pre-materialize every future escalation level as tasks/timers.

### EIA-5 — minimal Configure surface + OSS reference deployment

Add one compact **Identity & access** Configure area for connection status, provisioning health, roles/capabilities, department scope, group mappings, and escalation simulation.

Provide reference authentik configuration for AD/LDAP/SAML bridging, but do not vendor or fork authentik.

## 9. Acceptance gates

### Identity / provisioning

- wrong OIDC issuer/audience/state/nonce/PKCE fails closed;
- unknown OIDC subject is not silently provisioned;
- disabled principal loses an existing ClearSight session on the next authenticated request;
- ordinary authenticated reads/commands make zero LDAP/SAML/SCIM/IdP network calls;
- session data contains no OIDC access/refresh token and no cached role/capability authorization truth;
- cross-origin unsafe requests outside the trusted application origin are rejected;
- post-login return paths cannot redirect to an external origin;
- SCIM credentials are tenant-bound once EIA-3 is implemented;
- duplicate provisioning is idempotent once EIA-3 is implemented;
- group membership alone never creates material authority.

### Departments / capabilities

- global and department-scoped role codes remain distinct;
- same capability in two departments resolves only at the exact requested department scope;
- parent department does not inherit child access unless an explicit future server-side rule says so;
- department capability cannot satisfy a tenant-wide administrative route;
- tenant/legal-entity scope remains authoritative above department scope;
- removing a role or deactivating a principal changes current access without rewriting authority routes or re-authenticating at the IdP.

### Escalation

- levels execute in configured order and only once once EIA-4 exists;
- non-monotonic/duplicate/invalid sequences are rejected before policy activation;
- department level traversal stops safely at the available hierarchy once EIA-4 exists;
- no-route/conflict cannot silently assign an administrator;
- escalation never bypasses authority, delegation, segregation, or protected-record visibility;
- completion/cancellation cancels the pending next timer once EIA-4 exists;
- policy changes are effective-dated and do not rewrite historical decisions.

## 10. Non-goals

Do not build in ClearSight:

- passwords or account recovery;
- LDAP/Active Directory clients;
- SAML XML processing;
- Kerberos;
- MFA/passkey enrollment;
- generic IAM administration;
- a second organization hierarchy;
- a second role/policy/authority engine;
- a second scheduler or workflow engine.

The result is a small enterprise interoperability layer around the authority model ClearSight already owns.
