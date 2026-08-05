# ClearSight Data Model and Storage

This document converts product semantics into storage rules without exposing database structure as the user experience.

## Storage classes

### Authoritative relational state

PostgreSQL owns material structured truth: tenants/entities/scopes, principals/assignments/authority, Programs/Requirements/controls/Claims, Matters/workflows/decisions/actions/verification, requests/invitations/responses, versions/legal holds/audit, and outbox/inbox/jobs.

### Versioned object storage

Object storage owns source documents, evidence files/media/spreadsheets, generated packages, derivatives, and manifests. Relational rows retain object version, hash, size, media type, classification, encryption context, retention, hold, and lineage.

### Rebuildable projections

Search, graph, vector, work queues, dashboards, indicators, and reports are projections—not independent truth systems.

## Identity and tenancy

Use UUIDv7-style time-ordered identifiers for write-heavy tables. Public IDs remain opaque. Every tenant-owned table has non-null `tenant_id`; dominant indexes lead with tenant and query scope/state. Row-level security may add defense in depth, but application queries must include tenant and purpose.

Protected domains may use separate schemas, databases, keys, or deployments according to threat model and policy.

## Temporal model

Material records carry valid time and record time. Corrections create a new version and supersession link. Historical reconstruction distinguishes what was true from what was known. Operational lease/heartbeat fields may update in place when they do not represent institutional history.

## Transaction boundaries

One short transaction may update one Program or Matter aggregate, its workflow/assignments, directly related decision/action/verification state, safe audit metadata, and outbox events. Large populations/artifacts use chunked manifests. Transactions never span model calls, uploads, external APIs, or human review.

## High-volume design

Observations/assertions, audit events, outbox/jobs, import rows, reconciliation, projection changes, and notification deliveries require annual row/byte forecasts, retention, partition decisions, hot-index analysis, vacuum strategy, and replay plans.

Default partition candidates are tenant plus effective/capture month, implemented only after benchmark evidence. Avoid thousands of tiny partitions.

## Index and query rules

- Index actual query shapes.
- Prefer partial indexes for active/unresolved/unpublished/unredeemed/current rows.
- Use covering indexes for hot queues only after measurement.
- Use GIN only for bounded JSONB queries; frequent fields become typed columns.
- Use stable keyset pagination, not deep offsets.
- Validate population and authority plans with realistic authorization predicates.

## JSONB boundary

JSONB fits bounded policy conditions, request schemas, scope expressions, safe event metadata, and adapter envelopes. It does not replace core concepts, authority predicates, common filters, money, dates, or relationships.

## Invitations

Persist only token hash, request, audience hash, policy version, issued/expiry/redeemed/revoked times, and safe delivery metadata. Raw tokens never persist. Answers become versioned Observations and remain distinct from evidence conclusions.

## Outbox and jobs

Material transactions write outbox events atomically. Workers claim bounded batches with skip-locked semantics, leases, attempts, and result state. Consumers keep idempotent inbox/effect records. The system promises at-least-once delivery with idempotent effects and reconciliation—not exactly-once external execution.

## Retention and migrations

Retention evaluates purpose, classification, jurisdiction, legal hold, investigation, authority request, and source obligations. Deletion propagates across authoritative rows, objects, derivatives, projections, caches, and backups according to policy.

Schema changes document lock/backfill impact; large backfills are asynchronous/restartable; expand-migrate-contract is preferred; destructive changes require backup and an operational plan. Down migrations are local aids, not production recovery for data-bearing releases.

The foundation migration intentionally includes only tenancy, principals, responsibility, authority, workflow, request, invitation, outbox, and audit foundations. Domain tables arrive with their first vertical slice.
