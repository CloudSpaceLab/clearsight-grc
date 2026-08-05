# ADR-0006 — Signal, Drift and Readiness Separation

Status: Accepted  
Date: 2026-08-05

## Context

Continuous compliance needs fast detection of change without allowing source events or AI output to silently become incidents, legal conclusions or material decisions.

## Decision

Represent source changes as idempotent Signals. Deterministic policy converts relevant Signals into versioned Drift records. Continuous Readiness is a rebuildable multidimensional projection over active drift, Program state, evidence and routing health.

Signals and drift cannot directly approve applicability, change material risk, accept an exception, represent the bank externally or close a Matter.

## Consequences

- change detection remains explainable and resilient without AI;
- evidence aging and source degradation do not exaggerate underlying risk;
- high-volume Signal ingestion can scale independently later;
- readiness can be rebuilt and exposes freshness;
- material handling still requires governed workflow and authority.

## Validation

Idempotency, missed-drift, false-materiality, latency, point-in-time reconstruction and failed-verification scenarios are release gates.
