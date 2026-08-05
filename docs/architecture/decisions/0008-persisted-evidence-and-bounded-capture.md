# ADR 0008 — Persisted evidence requests and bounded capture sessions

- **Status:** Accepted
- **Date:** 2026-08-05

## Context

ClearSight needs to request facts and files from bank staff, vendors, customers and authorities without treating email links, browser state or uploaded files as authoritative workflow state. Source freshness and external capture also need to operate continuously and safely at bank scale.

## Decision

1. PostgreSQL is authoritative for source definitions, observations, evidence requests, submissions, invitation/session hashes and artifact manifests.
2. Invitation and session credentials are high-entropy opaque tokens. Only hashes are stored.
3. Invitation redemption is transactional, one-time by default, request-state aware and revocable.
4. Capture sessions are short-lived server-side records; tokens are not stored in browser persistence.
5. Artifact bytes are stored outside PostgreSQL. PostgreSQL records immutable integrity metadata and inspection state.
6. The development adapter writes to a bounded local filesystem root. It is not a production storage recommendation.
7. New artifacts begin as `STORED_UNSCANNED` and cannot be represented as approved evidence.
8. Source-health maintenance is deterministic, bounded and emits transactional outbox events.
9. The legacy in-memory capture demo remains temporarily available while product screens migrate to the persisted evidence domain.

## Consequences

- Replay, cancellation and revocation are enforceable without exposing token values.
- Request and source state can be reconstructed independently of delivery channels.
- Production storage and inspection can replace the local adapter behind a narrow interface.
- The system must operate storage cleanup for partial failures and future retention/legal-hold policies.
- Verified audience identity still requires a separate step-up mechanism; token possession alone is not described as identity proof.

## Rejected alternatives

- Store invitation tokens in clear text.
- Put file bytes in PostgreSQL JSON or bytea rows by default.
- Trust browser-local state as the capture session.
- Mark uploads available before inspection.
- Couple source freshness to AI interpretation.
