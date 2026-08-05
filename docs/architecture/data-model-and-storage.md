# Data Model and Storage

## Authoritative stores

PostgreSQL owns material metadata and workflow state. Versioned object storage owns original source and evidence bytes. Search, graph, vector, work-queue and reporting stores are rebuildable projections.

## Governance foundation

The schema includes:

- tenants and legal entities;
- principals, organizational positions and role templates;
- position-role bindings, delegations and scoped responsibility assignments;
- authority grants and versioned routing policies;
- workflow instances, tasks and append-only workflow events;
- evidence requests and invitation grants;
- onboarding state;
- compliance Signals, drift assessments and readiness snapshots;
- automation policies;
- outbox and audit events.

## Temporal rules

Material assignment, role, authority, source, requirement, evidence and decision records preserve valid time and record time. Corrections supersede rather than overwrite. Active-record indexes use partial predicates.

## Identifiers

Authoritative database entities use UUIDv7 to reduce random-index write amplification while preserving globally unique sortable identifiers. Public opaque invitation and session tokens use separate high-entropy random values and are stored hashed.

## JSONB boundary

JSONB is appropriate for:

- bounded routing-policy definitions;
- scope selectors;
- wizard schemas;
- safe metadata;
- signal payloads;
- readiness dimensions.

Frequently filtered identities, states, times, responsibilities, policy versions and relationships remain typed columns.

## High-volume strategy

Signals, observations, evidence assertions, audit events and workflow events are append-heavy. Partition candidates are selected by tenant class and time after representative benchmarks. Large history moves to lower-cost archive without breaking point-in-time reconstruction.

## Query rules

- tenant and purpose lead authorization-aware indexes;
- keyset pagination replaces deep offsets;
- current queues have dedicated partial indexes;
- historical reconstruction uses effective/record time and stable IDs;
- counts and search results enforce authorization in the query;
- broad populations are never loaded then filtered in application memory.

## Current migrations

- `000001_foundation` — tenant, principal, assignment, authority, workflow, request, invitation, outbox and audit foundation;
- `000002_governance_autonomy` — organization, roles, delegation, routing policy, workflow tasks, onboarding, Signals, drift, readiness and automation policy.
