# Architecture Decision Records

ADRs capture durable implementation decisions that affect product semantics, security, data integrity, performance, deployment, or operability.

## Accepted foundation decisions

- [`0001-modular-core-and-authoritative-stores.md`](0001-modular-core-and-authoritative-stores.md)
- [`0002-authority-routing-and-policy-resolution.md`](0002-authority-routing-and-policy-resolution.md)
- [`0003-request-scoped-invitations.md`](0003-request-scoped-invitations.md)
- [`0004-durable-workflows-outbox-and-consistency.md`](0004-durable-workflows-outbox-and-consistency.md)

## Required before implementation

Additional ADRs must cover:

- technology stack and deployment topology;
- tenant isolation and residency modes;
- evidence encryption, retention, legal hold, and deletion;
- search, graph, vector, and reporting projections;
- protected reporting and authority-case isolation;
- model gateway and external automation adapters;
- offline capture;
- design-system implementation;
- backup, recovery, observability, and SLO enforcement.

## ADR template

```text
# ADR-NNNN — Title

Status: Proposed | Accepted | Superseded | Rejected
Date: YYYY-MM-DD

## Context
What problem, constraints, workload, and product invariants apply?

## Decision
What is being decided?

## Consequences
What becomes easier, harder, or explicitly deferred?

## Guardrails
Which rules must implementations preserve?

## Validation
Which benchmarks, tests, or production evidence can confirm or challenge the decision?

## Revisit when
Which measurable condition would justify supersession?
```

ADRs do not override canonical product specifications. A changed decision must update affected architecture, implementation, quality, and operational documents.
