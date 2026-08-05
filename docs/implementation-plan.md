# ClearSight Implementation Plan

This plan delivers ClearSight as a source-led, correctly routed, performant, AI-assisted bank GRC operating system. Checkboxes describe current repository status.

## Completed foundation

- [x] Canonical Programs/Matters product model.
- [x] Use-case catalogue and maturity model.
- [x] Responsibility, authority, delegation and escalation specification.
- [x] Request-scoped invitation and capture specification.
- [x] System/data/performance architecture and first ADR set.
- [x] Go API/worker, React web shell, PostgreSQL schema, OpenAPI, CI and performance-smoke scaffold.
- [x] Executable Today, authority-resolution and focused-capture demo services.

## Delivery rule

Every vertical slice must prove:

```text
source, schedule, event or report
→ Program update or typed Matter
→ authorized context assembly
→ correct actor and authority resolution
→ existing evidence search and minimum request
→ grounded recommendation or manual fallback
→ authorized decision/action/response
→ verification or acknowledgement
→ current projections and reconstructable history
```

No isolated CRUD, generic form builder, hard-coded approval chain, dashboard shell, or ungoverned AI feature is considered progress.

# Phase 1 — Durable trust foundation

- [ ] PostgreSQL repository layer and migration runner.
- [ ] Tenant, institution, legal entity, jurisdiction and scope aggregates.
- [ ] Enterprise identity boundary and principal lifecycle.
- [ ] Source-backed organizational positions and role templates.
- [ ] Responsibility assignments, authority grants and versioned routing policies.
- [ ] Conflict, segregation of duties, delegation, substitution, absence and escalation.
- [ ] Durable workflow instances, steps, timers, joins, cancellation and resume.
- [ ] Transactional outbox/inbox, worker leases, idempotency and dead-letter review.
- [ ] Material audit and point-in-time reconstruction.
- [ ] Configure role/authority matrix, sequence builder, simulation, maker-checker and rollback.

**Gate:** 100k+ active assignment/grant reference profile resolves p95 ≤100 ms; material commands re-evaluate authority; identity change safely reroutes in-flight work; no self-approval or circular delegation.

# Phase 2 — Sources, evidence, imports and capture

- [ ] Source Registry, authority, limitation, freshness, health and purpose.
- [ ] Observation, Evidence Item, Claim, Evidence Contract and contradiction persistence.
- [ ] Versioned object storage, scanning, hash, lineage, retention and legal hold.
- [ ] Resumable spreadsheet/document import, mapping, matching, partial acceptance and reconciliation.
- [ ] Minimum-question compiler and recipient ranking.
- [ ] Internal SSO and external invitation sessions with step-up and revocation.
- [ ] Draft, resume, correction, follow-up, receipt and notification lifecycle.
- [ ] Approved offline/mobile capture.
- [ ] Protected-report identity/content separation and anonymous mailbox.

**Gate:** ordinary request median <3 minutes; repeat import active effort <5 minutes; invitation forward/replay/expiry/revocation fails safely; 1-million-row benchmark does not breach interactive SLOs.

# Phase 3 — Program engine

- [ ] Program, Authority Source, provision, Requirement and applicability.
- [ ] Control Objectives and scoped Implementations.
- [ ] Evidence Contracts, schedules, filings, tests and assurance.
- [ ] Incremental trigger/invalidation and multidimensional Compliance State.
- [ ] Program overview, Requirement worklist, exception review and filing history.
- [ ] Program-to-Matter creation.
- [ ] Grounded extraction/mapping/change-summary proposals with deterministic fallback.

**Gate:** no manually edited authoritative RAG; state updates incrementally; review by exception preserves denominator and full-review triggers; common Program pages meet SLO.

# Phase 4 — Matter, decision, action and verification

- [ ] Typed Matter registry and subtype lifecycle/closure contracts.
- [ ] Actor-specific workspace composition and parallel work.
- [ ] redirect, delegate, recuse, escalate, merge, split, cancel, supersede and reopen.
- [ ] Decision options, challenge, approval, conditions, expiry and invalidation.
- [ ] Action and external execution references.
- [ ] Response Package, redaction, signatory, transmission and acknowledgement.
- [ ] Verification Contract, observation, acceptance, failure and reopening.
- [ ] Timeline and complete historical reconstruction.

**Gate:** task completion cannot close material work; merge/split preserves authority and protection; stale concurrent writes conflict safely; source/model outages retain manual operation.

# Phase 5 — Pilot verticals

- [ ] Continuous NDPA Program: ROPA, DPIA, breach, vendor and filing.
- [ ] CBN regulatory change: source to verified implementation.
- [ ] Protected authority request: legal review, subjects, directives, focused tasks and response.
- [ ] Legacy finding/exception: import to verified remediation.

**Gate:** all four pass use-case traceability, timed UX, authority, capture, evidence, AI, performance, recovery, protection and point-in-time tests.

# Phase 6 — Expanded bank use cases

- [ ] Risk situation, risk acceptance and waiver.
- [ ] Incident, operational loss and resilience.
- [ ] Third-party onboarding, reassessment, deficiency and exit.
- [ ] BIA, RCSA and KRI cycles.
- [ ] Audit, assurance and policy lifecycle.
- [ ] Complaint, conduct and data-subject rights.
- [ ] Protected reporting and investigator workflow.
- [ ] Executive/committee pack, dissent, decision and follow-up.

Each is a complete vertical slice, not a generic module.

# Phase 7 — Governed AI and automation

- [ ] Operator/capability/model/tool registry.
- [ ] authorization-aware retrieval, structured output and policy pipeline.
- [ ] prompt-injection and protected-data defenses.
- [ ] simulation, blast-radius preview, canary, approval, monitoring, suspension, rollback, expiry and verification.
- [ ] constrained Probo, ITSM, IAM, email and messaging adapters.
- [ ] evaluation and outcome-linked monitoring.

# Phase 8 — Enterprise scale and GA

- [ ] Multi-entity/jurisdiction and group/local authority.
- [ ] residency-aware storage, evidence and model routes.
- [ ] dedicated, VPC, on-premises and sovereign profiles.
- [ ] workload isolation, partitioning, archive and capacity automation.
- [ ] backup, restore, failover and DR exercises.
- [ ] support, break-glass, migration, offboarding, deletion and legal hold.
- [ ] independent security, privacy, accessibility, usability and performance validation.
- [ ] operational runbooks, SLOs and cost models.

## Definition of done

A milestone is complete only when its use cases, actors, authority, states, invitations, evidence, performance, recovery, accessibility, degraded operation, history, APIs, migrations, docs and tests agree. Correct but cumbersome or unproven-at-scale work is not complete.
