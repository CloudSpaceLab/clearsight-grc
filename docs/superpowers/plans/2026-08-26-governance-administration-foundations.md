# Governance Administration Foundations Implementation Plan

**Goal:** Make governance policy and delegation administration legal-entity scoped, typed, bounded and transaction-safe so Configure can expose real maker-checker workflows.

**Architecture:** Preserve the existing governance lifecycle, add canonical entity columns and closed definition types, bind verified scope at HTTP boundaries, move activation revalidation into repository transactions, and expose a sanitized bounded administration projection.

## Task 1: Canonical entity and typed scope

- Add failing domain tests for missing/mixed legal entity, unknown scope fields and invalid object/materiality combinations.
- Add strict policy/delegation scope decoders and canonical `LegalEntityID` fields.
- Add migration `000036` with guarded backfill and tenant/entity indexes; ambiguous rows fail rather than split.
- Pass governance domain and PostgreSQL schema tests.

## Task 2: Transaction-safe maker-checker activation

- Add failing tests for self-approval, stale authority, conflicting active rules, delegation cycles and authority loss between submit and approve.
- Re-read and lock the draft, active version, authority route and conflict graph in the committing repository transaction.
- Commit authoritative version, decision, append-only event, outbox event, effective projection and maintenance job together.
- Prove rollback on every induced failure with PostgreSQL integration tests.

## Task 3: Verified, bounded HTTP contract

- Add failing route tests for verified tenant/entity/actor overwrite, entity mismatch and pre-limit filtering.
- Add keyset-bounded inventory/detail routes and material create/submit/approve/reject/revoke routes with separate read/write capabilities.
- Update OpenAPI and route-registry coverage.
- Pass HTTP, registry and contract tests.

## Task 4: Configure administration workspace

- Add failing React tests for inventory, draft submission, checker approval/rejection, delegation revoke and degraded-authority read preservation.
- Add a dedicated `GovernanceAdminWorkspace` with typed fields and server-derived people/object candidates.
- Show state, effective date, maker/checker and one next action; never expose raw JSON or raw principal IDs.
- Add copy-quality, accessibility, responsive fixtures and rendered evidence.

## Verification

Run governance packages with standard and PostgreSQL tags, HTTP/OpenAPI tests, web unit/type/build/copy-quality tests, then the affected Configure render and interaction suite.
