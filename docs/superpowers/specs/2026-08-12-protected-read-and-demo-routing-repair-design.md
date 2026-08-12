# Protected Read and Demo Routing Repair Design

**Status:** Approved direction; written-spec review pending  
**Date:** 12 August 2026

## Problem

The deployed demonstration environment contains the referenced evidence requests and Matters, but direct reads return `404`. PostgreSQL read models serialize `tenant_id` as the tenant slug while the verified demo actor carries the canonical tenant UUID. HTTP visibility checks compare those values literally and intentionally mask the records as not found.

The demo foundation also contains principals without an active governed authority policy, so the Program workspace's read-only authority check receives `422 routing_failed`. Finally, the login gate probes the protected context endpoint before it knows whether a demo session exists, producing an expected but distracting `401` on every signed-out load.

## Considered approaches

### 1. Patch only the deployed demo data

Change the demo actor to use the tenant slug and insert one authority row manually on the server. This is fast, but it preserves two tenant representations in runtime records, would be overwritten by later releases, and leaves production PostgreSQL identities vulnerable to the same masked-read defect.

### 2. Teach visibility checks that one UUID and one slug are equivalent

Resolve aliases at each HTTP access check. This preserves current read shapes, but spreads database-aware identity logic into request handlers, risks inconsistent enforcement, and makes future protected reads easy to implement incorrectly.

### 3. Canonicalize PostgreSQL runtime identity and seed governed demo routing

Return canonical tenant UUIDs from actor-facing PostgreSQL read models while continuing to accept UUID or slug as bounded lookup input. Add idempotent, demo-only governed authority fixtures and a non-error session probe. This keeps authorization comparisons exact, preserves fail-closed behavior, repairs both current records, and prevents the same mismatch elsewhere.

**Decision:** Approach 3.

## Design

### Canonical tenant identity

PostgreSQL repositories that produce actor-facing domain records will serialize `tenant_id` as `tenants.id::text`, not `tenants.slug`. Queries may continue accepting either form so installer and operator ergonomics do not regress. Slugs remain presentation and lookup aliases; they are not authorization identity.

The first repair covers protected records and the shared read paths they depend on:

- Program and Matter current aggregates and summaries;
- evidence requests, recipients and sources;
- workflow tasks used for actor-facing work;
- other actor-protected PostgreSQL reads found by the regression audit.

HTTP visibility rules remain unchanged: tenant equality stays exact after canonicalization, restricted Matter allow-lists remain mandatory, and unauthorized direct reads continue returning `404`.

### Governed demo authority

The durable demo foundation seeder will install stable, idempotent non-production routing records using existing routing-policy, policy-version, responsibility-assignment and effective-route semantics. It will not add a bypass or make the audit-mode command guard the source of authority.

The fixture will provide the responsibilities exercised by the reference journeys and Program lifecycle inspection, mapped to the existing durable demo principals. Records will be clearly demo-scoped, effective-dated, rerunnable, and projected through the existing effective-authority refresh function. Production bootstrap remains unable to enable demo mode.

The Program page may therefore resolve the exact current candidate set. It will show mutation controls only when the signed-in actor is actually among those candidates; other roles remain read-only.

### Quiet authentication discovery

Add an identity-safe session-status read that returns `200` with only the current session state and whether demo login is available. It returns no tenant, principal, role, or protected resource data.

`DemoAuthGate` will use this endpoint before requesting protected context:

1. authenticated session: load `/api/v1/context` and enter the application;
2. unauthenticated demo runtime: load the supplied demo accounts and show login without issuing a protected request;
3. unauthenticated non-demo runtime: preserve the existing non-demo authentication path.

Invalid credentials and protected resources still return `401`; only the intentional signed-out discovery request becomes non-erroring.

## Data and deployment behavior

No new PostgreSQL instance or database is introduced. Existing canonical rows are reused. The deployment's idempotent foundation/reference installation repairs missing authority configuration and leaves business records intact.

No migration is required if the existing authority schema supports the fixture. If implementation proves a schema change unavoidable, it must be additive, backward-compatible for the current release, and separately justified before inclusion.

## Error handling

- A record with a different canonical tenant UUID remains invisible.
- A malformed restricted scope continues to fail closed.
- Missing or ambiguous authority continues to return a routing failure; demo success comes from valid fixture data, not handler fallback.
- A failed session-status request enters the existing degraded authentication behavior and does not assume a user is authenticated.
- Seeder collisions with non-demo policy codes or incompatible rows fail explicitly rather than overwriting bank-owned configuration.

## Test strategy

1. PostgreSQL integration tests create a tenant with distinct UUID and slug, authenticate with the UUID, and prove exact Program, Matter, evidence and actor-work reads return canonical UUID records.
2. Negative tests prove another tenant, a non-recipient, and a principal outside a restricted allow-list still receive no data.
3. Demo foundation tests run the seeder twice, prove stable row counts, refresh effective routes, and resolve the Program transition plus journey responsibilities to the intended candidate sets.
4. API tests prove the session-status response is `200` and contains no identity details when signed out, while protected context remains `401`.
5. React tests prove the signed-out demo gate does not call protected context before login and still loads context after successful login.
6. Full Go, PostgreSQL, TypeScript, Vitest, accessibility, deployment-script and shell validation run before push.
7. After automatic deployment, authenticated API probes and a browser network/console pass verify the reported evidence and Matter URLs return `200`, authority resolution returns `200`, and signed-out load emits no intentional `401`.

## Acceptance criteria

- Both reported existing UUID records open for actors who are entitled to see them.
- Actors who are not the evidence recipient or creator do not gain evidence access.
- Restricted Matters remain limited to their explicit allow-lists.
- Program authority inspection resolves from governed current routes rather than fallback logic.
- Signed-out demo load reaches the login page without a failed protected-context request.
- The default URL remains the guided demo and `?demo=0` remains presentation-only.
- The deployment continues using the existing native PostgreSQL 18 instance and the proven main-branch auto-deploy workflow.
