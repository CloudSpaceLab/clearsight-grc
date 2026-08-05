# Command integrity and Program status review — 2026-08-05

## Scope

- request identity binding;
- material Program and issue authority checks;
- command-body actor fields;
- issue creation with initial Program link;
- Program-trigger handling;
- command/event/outbox transaction boundaries;
- calculated Program status versioning;
- maintenance queue, worker, health, reconcile and rebuild;
- operator copy and failure states.

## Findings resolved

1. Actor identifiers are no longer trusted from command JSON. Verified request identity supplies actor, approver and reviewer fields.
2. Production configuration refuses to start unless signed identity and enforced command authorization are enabled.
3. Material commands are mapped to a current owner, reviewer, authorizer or signatory route.
4. Tenant mismatch, missing identity, missing route and authority-service failure stop execution with clear copy.
5. Creating an issue and its first Program link is atomic.
6. Processing a deduplicated Program trigger, its Program event, generated issue and initial link is atomic.
7. Every material command writes its status-maintenance job in the same PostgreSQL transaction.
8. Calculated Program status no longer advances the Program command version.
9. Status jobs are deduplicated, leased, retried, recoverable and visible to operators.
10. Configure uses “Program status updates,” “Check status records,” “Updates pending,” “Delayed” and “Needs attention” rather than exposing projection-engine terminology.
11. The web client now preserves the server’s human-readable error message instead of replacing it with generic HTTP status text.

## Performance review

The synchronous command path contains only bounded identity verification, authority resolution and one transaction. Program status calculation moved to the worker path.

The maintenance queue has:

- a partial unique index for one active Program job;
- a due/lease claim index;
- `FOR UPDATE SKIP LOCKED` claims;
- bounded batches;
- source-version coalescing;
- indexed tenant/status health reads;
- bounded reconciliation.

## Validation required before merge

- default and race-enabled Go tests;
- PostgreSQL-tagged composition;
- migrations through `000010` on PostgreSQL 18.4;
- identity signature and tamper tests;
- authority allow/deny/audit tests;
- command-body actor overwrite and tenant-mismatch tests;
- command versus status-version tests;
- atomic issue/link and trigger idempotency tests;
- worker lease, completion and health tests;
- PostgreSQL integration for transactions, status lag, reconcile and historical state;
- TypeScript type-check and production build.

## Remaining boundaries

- direct OIDC/SAML integration and gateway key rotation;
- group/organization synchronization;
- emergency authority and break-glass workflow;
- business-calendar-aware status SLOs;
- bulk status rebuild and dead-letter remediation UI;
- production-scale concurrency and lag evidence;
- authority enforcement on evidence, workflow and governance administration commands outside the Program/Matter scope.
