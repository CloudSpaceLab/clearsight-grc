# Demo role login

**Scope:** stakeholder/demo mode only  
**Production effect:** none — production configuration rejects demo mode and `/api/v1/demo/*` is not registered.

ClearSight demo mode does not silently inject one staff actor. A stakeholder chooses a supplied role on the login page, signs in with the visible demo credential, and receives a short-lived signed demo session. Switching role logs out and unmounts the application before the next identity is selected so role-scoped browser state cannot carry across identities.

## Built-in accounts

All built-in demo accounts use password `demo`.

| Role | Username | Notable demo role codes |
| --- | --- | --- |
| Chief Risk Officer | `cro@demo.clearsight.local` | `CRO`, `EXECUTIVE` |
| Chief Compliance Officer | `cco@demo.clearsight.local` | `CCO`, `EXECUTIVE`, `COMPLIANCE_OFFICER` |
| Chief Information Security Officer | `ciso@demo.clearsight.local` | `CISO`, `EXECUTIVE` |
| GRC Administrator | `grc-admin@demo.clearsight.local` | `GRC_ADMIN` |
| System Administrator | `system-admin@demo.clearsight.local` | `SYSTEM_ADMIN` |
| Internal Auditor | `auditor@demo.clearsight.local` | `AUDITOR`, `REVIEWER` |
| Program Owner | `owner@demo.clearsight.local` | `PROGRAM_OWNER` |
| Evidence Respondent | `evidence@demo.clearsight.local` | `EVIDENCE_RESPONDENT` |

The CCO includes `COMPLIANCE_OFFICER` so the source-role escalation guard can be demonstrated. The System Administrator receives the existing development-only administrator capabilities and can demonstrate the EIA-5 Identity & Access surface. Production authorization never derives capabilities from these demo role names.

## Security and architecture boundary

- demo login is enabled only when `CLEARSIGHT_DEMO_MODE=true` with development identity mode;
- production validation rejects demo mode;
- the three demo endpoints live in the existing canonical route registry with access class `DEMO_ONLY`; there is no second route inventory;
- when demo mode is disabled the routes are absent and return 404;
- demo credentials are intentionally visible because they are fixed non-production fixtures, never customer credentials;
- successful login creates an HttpOnly, SameSite=Lax, HMAC-signed cookie with an eight-hour lifetime;
- the signing key is generated in-process, so process restart invalidates existing demo sessions;
- tampered and expired demo sessions fail closed;
- bounded bearer/capture credentials are never interpreted as a staff demo identity;
- explicit development demo headers remain supported for automated fixtures/tests;
- OIDC and signed-gateway authentication paths are unchanged.

## Browser behavior

The top-level `DemoAuthGate` first calls the public `/api/v1/session/status` endpoint. Its response contains only `authenticated` and `demo_login_available` booleans: it never exposes a tenant, principal, role, legal entity or permission. A signed-out demo browser can therefore load the role catalogue without intentionally probing the protected `/api/v1/context` endpoint and producing a 401. Once authenticated, the gate loads the normal runtime context. If status discovery is unavailable during a mixed-version rollout, the previous context-first behavior remains as a compatibility fallback.

In a non-demo deployment the catalogue endpoints remain absent, so the gate does not invent a demo login or alter the configured production identity flow. The protected context endpoint still returns 401 for an unauthenticated caller.

The compact `Viewing as` account control lists the other available demo accounts. Choosing one logs out the current demo session and unmounts the full application before signing in to the selected account. This deliberately clears cached Today work, evidence, configuration, routing and other role-dependent UI state rather than trying to selectively reset individual stores.

The sign-in surface shows each role once and uses the server-supplied credentials behind one `Continue as` action. Shared demo passwords are not repeated across the page, and the selected account email remains secondary read-only context.

## Acceptance

- no cookie/header means no silently injected demo actor;
- signed-out session discovery returns only the two safe booleans and does not request protected context;
- invalid credentials return 401;
- successful role login produces the chosen principal/roles in `/api/v1/context`;
- System Administrator can reach development-only Identity & Access administration;
- CCO exposes the Compliance Officer source-role demonstration;
- tampered/expired cookies do not authenticate;
- logout expires the cookie;
- `/api/v1/demo/*` is absent outside demo mode;
- the login surface passes strict TypeScript, rendered-state/axe checks and the normal production web build.
