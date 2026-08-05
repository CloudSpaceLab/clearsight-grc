# Scaffold and Architecture Review

**Date:** 2026-08-05  
**Status:** foundation accepted for implementation; not production complete.

## Changes

- converted the documentation-only repository into an executable modular-monolith scaffold;
- added Go API and worker foundations;
- added Today, authority resolution, focused capture, and one-time invitation behavior with tests;
- added React/Vite/Tailwind application shell;
- added PostgreSQL 18 schema, OpenAPI, containers, CI, and k6 smoke test;
- added canonical application, data, engineering, performance, and operations documents;
- replaced stale high-level architecture and implementation mapping;
- updated README, documentation map, contributor rules, and implementation status.

## Validation

- `gofmt` clean;
- `go test ./...` passes;
- `go vet ./...` passes;
- live HTTP smoke passed for readiness, Today, and authority resolution;
- web install/build remains a CI validation because the execution environment had no package-registry access.

## Boundaries

The scaffold uses in-memory services. It does not yet provide production authentication, PostgreSQL repositories, row-level authorization, object storage, durable jobs, bank connectors, protected reporting, or real AI operations.

Next is Phase 1: tenant/context, principals, source-backed assignments, authority grants, policy versions, durable workflow, outbox, and audit persistence.
