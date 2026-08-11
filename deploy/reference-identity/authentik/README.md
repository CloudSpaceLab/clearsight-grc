# authentik reference bridge for ClearSight

This directory documents the **reference interoperability pattern** for banks that need to bridge LDAP/Active Directory, SAML or other legacy identity sources into ClearSight.

It is not a bundled IAM stack and ClearSight does not require authentik. If the institution already operates a standards-compliant OIDC provider such as Keycloak, Entra ID or Okta, connect it directly to ClearSight and provision through SCIM where available.

## Boundary

```text
legacy identity / directory
Active Directory · LDAP · SAML
        ↓
     authentik
        ├── OIDC → ClearSight /auth/oidc/*
        └── SCIM → ClearSight /scim/v2
                         ↓
                 ClearSight principal
                         ↓
          local role/capability eligibility
                         ↓
       existing authority / command guard
```

ClearSight remains the source of truth for role-template capabilities, department-scoped group-role mappings, responsibility, authority grants, delegation, segregation, protected-record visibility and material command authorization.

## 1. Operate authentik independently

Use an institution-managed authentik deployment and **pin a supported patched release**. Do not use a floating container tag in production. ClearSight does not own authentik upgrades, backups, database lifecycle or HA.

For Active Directory or LDAP, configure the directory as an authentik source. Prefer encrypted directory transport and the bank's existing certificate/trust management. Do not add a direct LDAP client to ClearSight.

## 2. Configure OIDC sign-in

Create an OAuth2/OIDC provider and application in authentik for ClearSight.

Use a **strict** redirect URI matching the ClearSight API callback exactly:

```text
https://<clearsight-api-host>/auth/oidc/callback
```

Configure ClearSight with the corresponding provider values:

```dotenv
CLEARSIGHT_IDENTITY_MODE=oidc
CLEARSIGHT_ALLOWED_ORIGIN=https://<clearsight-web-host>
CLEARSIGHT_OIDC_ISSUER=https://<authentik-host>/application/o/<application-slug>/
CLEARSIGHT_OIDC_CLIENT_ID=<client-id>
CLEARSIGHT_OIDC_CLIENT_SECRET=<secret-reference-or-injected-secret>
CLEARSIGHT_OIDC_REDIRECT_URL=https://<clearsight-api-host>/auth/oidc/callback
CLEARSIGHT_OIDC_SECURE_COOKIES=true
```

Do not commit the client secret. ClearSight validates OIDC state, nonce, PKCE, issuer, audience and expiry and then correlates the immutable issuer + subject to a locally provisioned principal.

## 3. Create the ClearSight SCIM source

In **Configure → Identity & access → Sign-in & provisioning**, create a provisioning source.

Choose:

- a stable source code, for example `AUTHENTIK`;
- the authentik OIDC issuer when OIDC/SCIM correlation is required;
- `externalId` as the stable subject unless the integration has deliberately standardized another immutable mapping.

ClearSight returns a high-entropy SCIM bearer token **once**. Copy it immediately into the authentik SCIM provider. ClearSight persists only the SHA-256 digest and cannot display the token later.

If the token is lost or exposed, use **Rotate token**. The previous token stops authenticating immediately.

## 4. Configure outbound SCIM in authentik

Create a SCIM provider in authentik with:

```text
Base URL: https://<clearsight-api-host>/scim/v2
Token:    <reveal-once token from ClearSight>
```

Associate that SCIM provider as the application's backchannel provisioning provider.

Provision Users and Groups. ClearSight intentionally supports direct user membership only in this tranche; do not rely on nested group expansion inside ClearSight. If the upstream directory uses nested groups, flatten or normalize the effective group membership before it reaches the ClearSight SCIM boundary.

## 5. Keep OIDC and SCIM identity correlation stable

The safest reference mapping is:

```text
authentik OIDC subject == authentik SCIM externalId
```

ClearSight then correlates:

```text
OIDC issuer + subject
        ↕
SCIM source + externalId
        ↓
one local principal
```

Never use mutable email addresses as the default identity correlation key.

## 6. Map directory groups to ClearSight roles explicitly

SCIM group membership alone grants **nothing**.

In **Configure → Identity & access → Group → role mappings**, an authorized ClearSight administrator explicitly maps a current directory group to an existing ClearSight role template and optionally an exact department path.

Example:

```text
Directory group: Risk Reviewers
Role:            RISK_REVIEWER
Department:      BANK / RISK / OPERATIONS
```

That mapping provides coarse product capability eligibility only. It does not create responsibility, approval authority, signatory authority, delegation or protected-record access.

## 7. Validate before production cutover

At minimum prove:

1. OIDC login succeeds only for a provisioned subject.
2. SCIM creates and updates the expected Users and Groups.
3. OIDC subject and SCIM `externalId` resolve to the same principal.
4. Removing a user from a mapped group removes the corresponding capability on the next application request.
5. Revoking the SCIM source disables source-derived eligibility without deleting historical principal records.
6. Token rotation invalidates the old SCIM credential.
7. Department-scoped mappings do not satisfy legal-entity-wide permissions.
8. A mapped role still cannot execute a material command unless current ClearSight authority, delegation, segregation and visibility checks allow it.
9. Escalation preview matches the configured department ancestry while the actual actor is resolved only when a level fires.

## Operational rule

An ordinary ClearSight authenticated request must not call authentik, LDAP, SAML or SCIM to decide authorization. Identity is verified at sign-in; current role/capability and authority state is evaluated from ClearSight's own PostgreSQL state on the application request path.
