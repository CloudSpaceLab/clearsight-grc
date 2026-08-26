# Governance administration foundations

**Status:** Approved direction
**Date:** 2026-08-26
**Delivery:** Safety foundation before the Configure workspace exposes policy or delegation changes

## Decision

ClearSight will expose governance policy and delegation administration only after every definition is bound to a verified legal entity, every scope is typed and fail-closed, and activation is revalidated inside the same transaction that records the authoritative version, governance decision, event and outbox work.

The first increment adds these foundations and a bounded administration read model. It does not expose raw JSON, accept principal IDs as ordinary form input, or present an enabled policy builder whose definition cannot be simulated before activation.

## Current defect

The backend already supports policy and delegation lifecycles, but the deployed Configure workspace exposes only summaries and a narrow escalation guard. Existing policy definitions can span an implicit tenant scope, delegation scope is arbitrary JSON, list reads are unbounded, and some conflict or cycle checks occur before the committing transaction. Adding UI controls over those contracts would allow a bank user to create an ambiguous or raceable authority route.

## Operating contract

### Legal-entity scope

- Every routing policy version and delegation stores one canonical `legal_entity_id`.
- The HTTP layer obtains the actor, tenant and legal entity from verified identity. Body or query fields cannot widen them.
- A policy definition may reference only its stored legal entity. Existing mixed-entity definitions fail migration or validation; they are never silently split.
- List and exact reads filter by tenant and legal entity before ordering or limiting.

### Typed definitions

- Delegation scope uses a closed schema: legal entity, optional object type and exact object, optional decision type, and optional materiality range.
- Unknown fields, invalid combinations and unsupported object types fail validation.
- Policy rules use the same canonical entity and object vocabulary. The UI will later use server-provided candidates and typed controls.

### Maker-checker and material activation

- A maker creates or revises a draft and submits it for review.
- A different, currently authorized checker approves or rejects it.
- Approval re-reads and locks the current draft, authority route, active version, conflicting rules and delegation graph inside the transaction.
- The authoritative row/version, decision, append-only event, outbox event, effective projection and required maintenance job commit together.
- Rollback is a forward revision that supersedes the current version; historical versions remain reconstructable.

### Bounded administration read model

The Configure workspace may read a bounded inventory containing policy/delegation state, version, effective dates, legal entity, maker/checker labels and latest decision. It must not expose raw token material or internal JSON as the primary representation. Authority-service failure preserves reads and disables mutation with a recovery explanation.

## Delivery states

Policy: `DRAFT -> IN_REVIEW -> APPROVED -> ACTIVE -> SUPERSEDED|RETIRED`, with rejection returning a new draft revision rather than mutating approved history.

Delegation: `DRAFT -> IN_REVIEW -> APPROVED -> ACTIVE -> EXPIRED|REVOKED`. Activation cannot exceed the delegator's current authority and cannot create a self-route, conflict or cycle.

## Failure behavior

Missing verified identity, entity mismatch, stale version, changed authority, inactive participant, conflict, cycle or persistence failure returns a non-committed command failure. Derived explanation failure after commit cannot turn success into failure. The UI refreshes the bounded inventory and explains who must act next.

## Acceptance proof

- Domain tests reject unknown scope fields, mixed entities, self-approval, excess authority, conflicts and cycles.
- PostgreSQL tests prove activation revalidation and row/event/outbox/projection atomicity.
- HTTP tests prove verified scope binding and bounded entity-filtered reads.
- Configure tests prove read preservation and mutation disablement when authority resolution is unavailable.
- OpenAPI, product copy, state fixtures and rendered evidence are updated before the administration controls are declared complete.

## Non-goals for this increment

This foundation does not yet deliver responsibility matrices, absence/substitution administration, organization-position management or the final visual rule builder. Those follow on the same typed and transaction-safe contracts.
