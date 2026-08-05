# Architecture Decision Records

ADRs capture durable implementation decisions affecting product semantics, security, data integrity, performance, deployment, or operability.

## Accepted foundation decisions

- [`0001-modular-core-and-authoritative-stores.md`](0001-modular-core-and-authoritative-stores.md)
- [`0002-authority-routing-and-policy-resolution.md`](0002-authority-routing-and-policy-resolution.md)
- [`0003-request-scoped-invitations.md`](0003-request-scoped-invitations.md)
- [`0004-durable-workflows-outbox-and-consistency.md`](0004-durable-workflows-outbox-and-consistency.md)
- [`0005-implementation-stack.md`](0005-implementation-stack.md)

## Still required before affected capability ships

- tenant isolation and residency modes;
- evidence encryption, retention, legal hold, and deletion;
- search, graph, vector, and reporting projections;
- protected reporting and authority-case isolation;
- model gateway and external automation adapters;
- offline capture;
- design-system implementation;
- backup, recovery, observability, and SLO enforcement.

## Template

```text
# ADR-NNNN: Title
Status: Proposed | Accepted | Superseded | Rejected
Date: YYYY-MM-DD

## Context
## Decision
## Consequences
## Guardrails
## Validation
## Revisit when
```

ADRs do not override canonical product specifications. A changed decision must update affected architecture, implementation, quality, and operations documents.
