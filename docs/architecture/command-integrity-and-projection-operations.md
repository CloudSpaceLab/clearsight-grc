# Command integrity and Program status operations

## Purpose

Material changes must be attributable, authorized, atomic and recoverable. Calculated Program status must remain fast to read without making a successful command depend on a second synchronous calculation.

## Request actor boundary

Production traffic arrives through an identity-aware gateway. The gateway sends a short-lived, HMAC-signed identity envelope containing:

- tenant;
- principal;
- legal entity;
- actor kind;
- authentication method and assurance;
- session;
- issue and expiry times.

ClearSight verifies the signature, request path, method, clock window and expiry before placing the actor in request context. Body fields such as `actor_id`, `approved_by`, `assessed_by`, `authority_principal_id` and `reviewer_principal_id` are never trusted; handlers receive values bound from the verified actor.

Production requires signed identity and enforced command authorization. Development may use an explicitly configured demo identity and audit-only authorization.

## Command authorization

Each material command maps to:

- object type and object ID;
- responsibility required now;
- decision type;
- materiality;
- whether a service identity is permitted.

The current routing policy is resolved at command time. Execution proceeds only when the verified actor matches the selected principal. Missing routes, stale identity, tenant mismatch or authority-service failure block execution in enforced mode.

Primary error copy is operational:

- “Sign in is required to continue.”
- “This command is outside your signed-in bank scope.”
- “The approval route could not be checked. No change was made.”
- “You are not the person currently authorized for this change.”

## Transaction boundaries

The following are one PostgreSQL transaction:

```text
command validation
→ authoritative row changes
→ append-only command event
→ transactional outbox event
→ deduplicated Program-status update job
→ commit
```

Creating an issue with its first Program link is one transaction. Processing a deduplicated Program trigger, recording its command event, creating its issue and link, and queueing the status update is also one transaction.

No API response may report failure after a material command has already committed merely because a calculated-status refresh failed.

## Command version versus calculated-status version

A Program command version changes only when material Program facts change. Calculated Program status has an independent `projection_version` and records the Program command version it assessed.

This distinction allows the UI to say:

- **Current** — status reflects the latest Program version;
- **Updates pending** — a calculation is queued;
- **Delayed** — the oldest queued calculation exceeded the operating threshold;
- **Needs attention** — repeated calculation attempts failed.

A stale but known status remains visible with its assessed Program version. It must not be described as current.

## Durable status maintenance

`continuity_projection_jobs` provides:

- one active job per tenant, projection and Program;
- source-version coalescing;
- bounded `SKIP LOCKED` claims;
- worker identity and lease;
- stale-claim recovery;
- bounded retry;
- terminal failure state;
- health and lag reporting;
- reconciliation and manual rebuild.

The worker calculates status from authoritative Program facts and open linked issues, writes a separately versioned snapshot, emits a `PROGRAM_STATE` event and publishes through the transactional outbox.

## Operator experience

Configure shows **Program status updates**, not a technical projection console. It displays:

- waiting updates;
- failed updates;
- age of the oldest waiting update;
- most recent successful update;
- latest error where one exists;
- a governed **Check status records** action.

Unknown data remains unavailable rather than becoming a reassuring zero.

## Performance rules

- command authorization is a bounded route lookup;
- request bodies are limited to 1 MiB before command inspection;
- active status jobs are deduplicated;
- workers claim bounded batches;
- status calculation is removed from the synchronous command path;
- health reads aggregate indexed job metadata rather than replaying events;
- reconciliation is bounded and resumable.

## Remaining boundaries

This phase does not provide direct OIDC/SAML integration, gateway key rotation, emergency break-glass UI, bulk rebuild scheduling, production-scale lag evidence or automated dead-letter remediation. Those remain explicit release work.
