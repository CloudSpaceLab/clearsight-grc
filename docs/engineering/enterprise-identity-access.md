# Enterprise identity and access

**Status:** EIA-0 through EIA-4 implemented on PR #59; EIA-5 remains  
**Scope:** enterprise sign-in, provisioning, department-aware access, and governed escalation  
**Supersedes:** the greenfield LDAP/SAML portions of the older enterprise-productization P2/P3 plan

## 1. Boundary

ClearSight is not an IAM product. Its native enterprise boundary is deliberately small:

```text
authentication      OIDC
provisioning        SCIM 2.0
session             server-side ClearSight session
coarse access       existing roles + local capabilities
governed actions    existing responsibility / authority / delegation / segregation
legacy federation   upstream open-source IAM bridge
```

Reference OSS bridge: **authentik**. Existing bank Keycloak, Entra, Okta, or another standards-compliant OIDC provider can connect directly. LDAP/AD, SAML, Kerberos, passwords, MFA, passkeys, and recovery stay upstream.

Ordinary authenticated reads and commands must make zero LDAP/SAML/SCIM/IdP network calls.

## 2. Existing foundations remain authoritative

Do not replace:

- `principals`;
- `org_positions`, `role_templates`, and `position_role_bindings`;
- `responsibility_assignments`, `authority_grants`, `delegations`, and segregation checks;
- `routing_policies` / `routing_policy_versions` / `effective_authority_routes`;
- `workflow_tasks`, `workflow_timers`, outbox/inbox, and current worker infrastructure;
- `internal/identity`, `internal/authority`, and `internal/commandauth` trust boundaries.

Enterprise identity establishes a local principal and coarse eligibility. Existing object visibility and material command authority remain final.

## 3. Departments and capabilities

Departments are stable hierarchical paths on existing organization/access state, for example:

```text
["BANK", "OPERATIONS", "PAYMENTS"]
```

No separate department subsystem exists. Empty path means legal-entity scope. Non-empty paths are exact scope unless a governed server-side rule explicitly says otherwise; a client-supplied path never selects authorization scope.

`role_templates.capabilities` owns bounded coarse capabilities. Effective role sources are only:

1. existing position → role bindings; and
2. approved directory group → existing role bindings.

```text
empty department role
→ legal-entity-wide actor role/capability

non-empty department role
→ exact department grant
```

A capability answers **which class of product function may this actor use?** The existing authority resolver still answers **may this actor perform this governed action on this object now?**

## 4. Multi-level escalation contract

Escalation sequences live inside existing versioned `routing_policy_versions.definition`; there is no escalation-policy table or second routing engine.

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

Rules:

- array order is escalation order;
- only one sequence may exist for a trigger in a policy version;
- `department_levels_up: 0` means the triggering work department, `1` its parent, etc.;
- omitted `department_levels_up` means legal-entity routing scope;
- steps select responsibility + scope, never a hard-coded person;
- `after` values are cumulative thresholds and strictly increasing;
- only the next level is scheduled;
- assignment by escalation does **not** grant material command authority.

The policy schema recognizes `OVERDUE`, `NO_ROUTE`, `AUTHORITY_INSUFFICIENT`, `MATERIALITY_INCREASE`, `RECIPIENT_UNAVAILABLE`, and `CONFLICT`. **EIA-4 makes `OVERDUE` executable.** The others remain schema-valid until their real domain event adapters are implemented; do not manufacture timestamps/events merely to claim coverage.

## 5. OSS component boundary

Native libraries:

- OIDC: `github.com/coreos/go-oidc/v3/oidc` + `golang.org/x/oauth2`;
- sessions: `github.com/alexedwards/scs/v2` + `github.com/alexedwards/scs/pgxstore`;
- SCIM: exact-pinned `github.com/elimity-com/scim` commit `2641426a1539`, contained behind `internal/scimapi`.

External IAM:

- **authentik** is the reference LDAP/AD/SAML → OIDC/SCIM bridge;
- existing **Keycloak** is a first-class direct OIDC target.

Do not add Casbin, OPA, Cerbos, another workflow engine, event bus, scheduler, or policy database for this work.

## 6. Implemented tranches

### EIA-0 — shared access/escalation contract

- `org_positions.department_path`;
- `role_templates.capabilities`;
- strict bounded escalation parser/validation in existing routing-policy definitions;
- no department table, escalation table, authorization engine, scheduler, or UI framework.

### EIA-1 — OIDC + server session

- Authorization Code + state + nonce + PKCE S256;
- immutable tenant-bound issuer + subject correlation through `principal_identities`;
- unknown subjects fail closed rather than privileged JIT creation;
- one federated identity maps to a principal, not a legal entity;
- SCS server sessions with maintained pgx store;
- no OIDC access/refresh token or cached authorization truth in browser session state;
- bounded trusted-origin callback/CORS/cross-origin protections;
- signed-gateway and development modes remain compatibility modes.

### EIA-2 — local department-aware capability resolution

- each OIDC application request reloads current principal/entity/access-source/role/capability state from PostgreSQL;
- IdP role/group/permission claims are not authorization truth;
- position and governed group bindings use the same capability evaluator;
- department grants remain exact-scope and cannot satisfy unscoped permissions;
- current revocation/expiry takes effect without IdP reauthentication;
- material commands still pass visibility + `commandauth.Guard` + current authority/delegation/segregation.

### EIA-3 — SCIM Users + Groups

- isolated `/scim/v2` machine edge with tenant-scoped bearer credentials stored only as SHA-256 digests;
- bounded discovery, Users/Groups CRUD, PATCH, equality filters, and pagination;
- SCIM Users create and own canonical `PERSON` principals; no email-based takeover of unrelated principals;
- explicit source correlation may map `externalId` or `userName` to the EIA-1 OIDC issuer/subject model;
- direct same-source group membership only; nested groups are rejected and no closure table exists;
- SCIM cannot write group→role bindings;
- ClearSight-owned effective-dated group→existing-role bindings provide legal-entity or exact-department capability eligibility;
- directory state cannot manufacture responsibility, signatory rights, authority grants, delegation, or protected-record visibility.

### EIA-4 — executable multi-level OVERDUE escalation

Implemented on the existing runtime stack with **no new durable table**:

```text
overdue Matter workflow task
→ active pinned routing-policy escalation sequence
→ one workflow_timers row for next level
→ existing timer claim/fire
→ WorkflowTimerFired outbox event
→ idempotent escalation consumer / inbox receipt
→ re-read current Matter + task authorization facts
→ current authority resolution
→ protected-Matter visibility
→ delegation + authority grants + segregation
→ exact department boundary for scoped level
→ update same Workflow task as ESCALATED overlay
→ schedule next level only
```

Important semantics:

- the timer carries only escalation lineage (`task`, workflow, policy version, sequence, level, baseline due date) rather than stale materiality/legal-entity/decision authority facts;
- each fired level re-reads current Matter ID, legal entity, decision type, materiality, activity/visibility and authority state;
- the triggering task's department path is resolved when level 0 actually fires and then pinned as the ancestry base for that escalation lineage;
- authority resolution remains unchanged; department filtering is applied to the already authority/delegation/grant/segregation-filtered candidate set;
- exactly one candidate is required for an automatic reassignment; no-route, ambiguous, hidden, or multi-candidate states record `WORK_ESCALATION_UNRESOLVED` and never silently assign an administrator;
- successful levels record `WORK_ESCALATED` and update the existing task, not a parallel escalation task;
- outbox replay is idempotent via the existing inbox ledger and per-level attempt key;
- stale due-date/policy races fail closed and do not schedule a stale next level;
- a tiny database overlay invariant prevents ordinary projector reconciliation from erasing an active escalation when the canonical work requirement, due date and policy are unchanged; genuine source changes can replace it;
- completing/cancelling work cancels the pending next escalation timer;
- escalation assignment remains a work-routing projection: a material action still requires the normal command guard and current authority at execution time.

## 7. EIA-5 remaining

Add one compact **Configure → Identity & access** surface for:

- sign-in connection status;
- provisioning/source health and token bootstrap/rotation;
- people/groups inspection;
- role/capability and group→role administration;
- department scope inspection;
- escalation policy simulation/status.

Provide reference authentik configuration for AD/LDAP/SAML bridging, but do not vendor or fork authentik.

## 8. Acceptance invariants

Identity/provisioning:

- wrong OIDC issuer/audience/state/nonce/PKCE fails closed;
- unknown/disabled principal cannot silently retain access;
- ordinary requests never query LDAP/SAML/SCIM/IdP;
- group membership alone never creates material authority;
- provisioning cannot seize an unrelated principal by email.

Departments/capabilities:

- global and department-scoped roles remain distinct;
- exact department grants do not inherit parent/child access automatically;
- department scope never broadens tenant/legal-entity/object visibility.

Escalation:

- levels execute in configured order and at most once per lineage attempt;
- only the next level is pre-scheduled;
- every fired level uses current materiality/legal entity/visibility/authority facts;
- department traversal fails closed if the base path is absent, ambiguous, or exhausted;
- no-route/ambiguity cannot assign an administrator by fallback;
- projector reconciliation cannot erase an active escalation without a genuine source change;
- completion/cancellation cancels the pending next timer;
- policy version + baseline due date preserve escalation lineage without rewriting historical decisions;
- escalation assignment ≠ command authority.

## 9. Non-goals

Do not build inside ClearSight:

- passwords/recovery/MFA/passkey enrollment;
- LDAP/AD clients, SAML XML, or Kerberos;
- generic IAM administration;
- nested-directory-group materialization without a proven requirement;
- a second organization hierarchy, role engine, authority engine, scheduler, workflow engine, or escalation task stack.

The result is a small enterprise interoperability and work-routing layer around the authority model ClearSight already owns.
