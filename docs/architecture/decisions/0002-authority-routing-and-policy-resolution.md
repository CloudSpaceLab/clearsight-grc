# ADR-0002 — Authority Routing and Policy Resolution

**Status:** Accepted  
**Date:** 2026-08-04

## Context

ClearSight workflows depend on responsible performers, owners, reviewers, challengers, authorizers, signatories, and escalation roles. A single assignee, static RBAC, or hard-coded approval chain cannot model bank scope, delegation, segregation of duties, materiality, committee authority, or organization change.

## Decision

Use versioned policy records for:

- role templates and organizational positions;
- scoped responsibility assignments;
- authority grants and decision limits;
- routing and escalation policies;
- delegation, substitution, handoff, conflict, and recusal;
- sequence, parallel, quorum, veto, and challenge steps.

Maintain a materialized assignment index for common runtime resolution. Effective authority remains the intersection of identity, assignment, scope, purpose, relationship, workflow state, decision class, limits, delegation, conflict, and current policy.

Material actions re-evaluate authority at execution. Cached routes are not final authorization.

## Consequences

The product can express complex bank governance without customer-specific code. Configuration and policy resolution become first-class capabilities requiring versioning, simulation, performance budgets, and recovery.

## Guardrails

- responsibility and authority remain distinct;
- no self-approval or privilege amplification;
- delegation cannot exceed the delegator’s authority;
- routing failure is explicit and never defaults to a global administrator;
- organization and role changes re-evaluate in-flight work;
- one dominant next action is calculated per actor and state;
- protected work uses isolated routes and queues.

## Validation

Test routine review, material exception, absence, role change, conflict, missing authorizer, committee quorum, and break-glass scenarios at the reference workload.

## Revisit when

Revisit the policy representation if real pilot routing cannot be expressed clearly, simulation becomes impractical, or resolution cannot meet the documented latency and correctness targets.
