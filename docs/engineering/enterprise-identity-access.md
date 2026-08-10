# Enterprise identity and access

**Status:** focused implementation plan  
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

ClearSight must never require an LDAP, SAML, or IdP network call on an ordinary read or command path.

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

Departments are represented as a **stable hierarchical path of source-backed codes** on organizational positions and role bindings:

```text
["BANK", "OPERATIONS", "PAYMENTS"]
```

Rules:

- path values are stable codes, not display names;
- an empty path means tenant/legal-entity scope rather than an invented department;
- a position may belong to one current department path;
- a direct role binding may be restricted to one department path;
- department hierarchy comes from path prefixes;
- directory synchronization may update paths, but ClearSight preserves historical command attribution through existing principal/event history.

This supports department-level access and escalation without adding `departments`, department-membership, or department-history tables before they are proven necessary.

## 4. Capability model

`role_templates` gains a bounded set of coarse capabilities. Initial families should remain small:

- `PROGRAM_READ`, `PROGRAM_CONFIGURE`;
- `MATTER_READ`, `MATTER_CREATE`;
- `EVIDENCE_READ`, `EVIDENCE_REQUEST`, `EVIDENCE_RESPOND`, `EVIDENCE_REVIEW`;
- `IMPORT_READ`, `IMPORT_CREATE`, `IMPORT_REVIEW`;
- `AUTHORITY_READ`, `AUTHORITY_CONFIGURE`;
- `IDENTITY_READ`, `IDENTITY_CONFIGURE`;
- `PLATFORM_OPERATIONS_READ`, `PLATFORM_OPERATIONS_WRITE`.

Effective capabilities are the union of currently effective role bindings from:

1. direct principal role bindings;
2. the principal's current organizational position role bindings;
3. approved directory-group role bindings once SCIM Groups are implemented.

Department scope narrows role eligibility; it never broadens object visibility.

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

Runtime support can be introduced trigger-by-trigger. Schema existence must not be represented as executable escalation until the matching compiler/timer path exists.

## 6. Open-source component boundary

### Native libraries

- OIDC relying party: `github.com/coreos/go-oidc/v3/oidc` + `golang.org/x/oauth2`;
- server-side sessions: `github.com/alexedwards/scs/v2` with its pgx store;
- SCIM protocol adapter: `github.com/elimity-com/scim`, contained behind a ClearSight service adapter.

### External open-source IAM

**authentik** is the reference compatibility bridge for LDAP/AD/SAML to ClearSight OIDC + SCIM. **Keycloak** is a first-class direct OIDC compatibility target when already deployed by the bank. ClearSight does not depend on either vendor's internal APIs for authorization.

Do not add Casbin, OPA, Cerbos, another workflow engine, another event bus, or another policy database for this work.

## 7. Implementation sequence

### EIA-0 — shared access and escalation contract

Implement first:

- `org_positions.department_path`;
- `role_templates.capabilities`;
- current/effective `principal_role_bindings` with optional department scope;
- validated multi-level `escalations` inside existing routing policy definitions;
- documentation and migration ownership gates.

No OIDC, SCIM, UI, or escalation execution in this tranche.

### EIA-1 — OIDC + server session

Add OIDC connection configuration, immutable `issuer + subject` principal correlation, authorization-code + PKCE login, nonce/state protection, server-side SCS session, logout/revocation, and same-origin mutation protection.

Keep the current signed-gateway authenticator for development/compatibility. Native OIDC must not trust externally supplied ClearSight permissions.

### EIA-2 — local capability evaluator

Resolve current capabilities from direct/position role bindings and department scope. Bind the existing route permission metadata to this evaluator. Material commands additionally continue through `commandauth.Guard` and current authority resolution.

### EIA-3 — SCIM Users + Groups

Expose the minimal SCIM 2.0 service-provider surface required for enterprise provisioning. Map Users into existing principals; map Groups into a narrow directory-group projection; support explicit approved group-to-role bindings.

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

## 8. Acceptance gates

### Identity / provisioning

- wrong OIDC issuer/audience/state/nonce/PKCE fails closed;
- disabled principal loses current access;
- SCIM credentials are tenant-bound;
- duplicate provisioning is idempotent;
- group membership alone never creates material authority;
- ordinary reads/commands make zero LDAP/SAML/SCIM network calls.

### Departments / capabilities

- same role in two departments resolves only in the requested department scope;
- parent department does not inherit child access unless an explicit rule says so;
- tenant/legal-entity scope remains authoritative above department scope;
- removing a role removes its capabilities without rewriting authority routes.

### Escalation

- levels execute in configured order and only once;
- non-monotonic/duplicate/invalid sequences are rejected before policy activation;
- department level traversal stops safely at the available hierarchy;
- no-route/conflict cannot silently assign an administrator;
- escalation never bypasses authority, delegation, segregation, or protected-record visibility;
- completion/cancellation cancels the pending next timer;
- policy changes are effective-dated and do not rewrite historical decisions.

## 9. Non-goals

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

The result should be a small enterprise interoperability layer around the strong authority model that already exists.