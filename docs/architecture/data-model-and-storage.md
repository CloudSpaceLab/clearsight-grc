# Data Model and Storage

## Authoritative stores

PostgreSQL owns material metadata, aggregate state, decisions and workflow. Versioned object storage owns original source and evidence bytes. Search, graph, vector, work-queue and reporting stores are rebuildable projections.

## Current authoritative domains

The schema includes:

- tenants and legal entities;
- principals, organizational positions and role templates;
- position-role bindings, delegations and scoped responsibility assignments;
- authority grants and versioned routing policies;
- workflow instances, tasks, timers and append-only workflow events;
- Source Registry, source observations and health;
- capture requests, submissions, invitations, sessions and artifact manifests;
- Programs, Requirements and applicability decisions;
- Control Objectives, scoped Control Implementations and requirement mappings;
- Evidence Contracts and evidence assessments;
- calculated Program status snapshots and trigger records;
- typed Matters, links, decisions, actions, response packages and outcome checks;
- append-only Program/Matter continuity events;
- onboarding state;
- compliance Signals, drift assessments and readiness snapshots;
- automation policies;
- outbox, inbox and audit events.

## Program and Matter persistence

Normalized tables provide current bounded reads. Every successful Program or Matter command also appends a `continuity_events` record with aggregate version and safe actor metadata in the same transaction.

The event stream supports point-in-time reconstruction. Projections may be rebuilt from it, but it is not a general event-sourcing claim for unrelated modules.

Program triggers have tenant-scoped dedupe keys. Supported triggers create no more than one open Matter per key.

## Tenant integrity

Frequently linked material records use composite `(id, tenant_id)` keys and composite foreign keys. This prevents direct database writes from linking a Program, Requirement, control, evidence source, Matter, decision, action, response or outcome check across tenants.

## Temporal rules

Material assignment, role, authority, source, requirement, applicability, evidence, decision and outcome records preserve valid time or append-only history as appropriate. Corrections supersede rather than silently overwrite material history.

## Identifiers

Authoritative database entities use UUIDv7 to reduce random-index write amplification while preserving globally unique sortable identifiers. Public invitation and session tokens use separate high-entropy random values and are stored hashed.

## JSONB boundary

JSONB is appropriate for bounded, versioned shapes such as:

- scope selectors;
- policy and request schemas;
- known and missing facts;
- evidence basis and thresholds;
- safe metadata;
- signal payloads;
- status dimensions and reasons.

Frequently filtered identities, states, times, responsibilities, policy versions and relationships remain typed columns.

## High-volume strategy

Signals, observations, evidence assessments, audit events, workflow events and continuity events are append-heavy. Partitioning is deferred until representative volume tests identify the appropriate tenant/time strategy. Large history may move to lower-cost archive without breaking point-in-time reconstruction.

## Query rules

- tenant and purpose lead authorization-aware indexes;
- keyset pagination replaces deep offsets;
- current queues have dedicated indexes;
- historical reconstruction uses stable IDs and event time;
- counts and search results enforce authorization in the query;
- broad populations are never loaded then filtered in application memory;
- Program and Matter lists are bounded in the foundation and require projection-first baselines before high-cardinality release.

## Current migrations

- `000001_foundation` — tenant, principal, assignment, authority, workflow, request, invitation, outbox and audit foundation;
- `000002_governance_autonomy` — organization, roles, delegation, routing policy, workflow tasks, onboarding, Signals, drift, readiness and automation policy;
- `000003_merge_readiness_fixes` — merge and readiness corrections;
- `000004_governance_runtime` — maker-checker governance, timers, outbox leases and inbox receipts;
- `000005_evidence_capture` — Source Registry and persisted capture records;
- `000006_evidence_session_guard` — request-state guard for session creation;
- `000007_capture_tenant_integrity` — composite tenant integrity for evidence and capture;
- `000008_programs_matters` — Programs, Requirements, controls, evidence checks, status snapshots, typed Matters and continuity events.
