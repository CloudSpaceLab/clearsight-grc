# Source, evidence and secure capture architecture

## Purpose

This layer maintains the operational facts behind compliance conclusions. It records where facts came from, whether the source is current, what information was requested, who or what submitted it, and the integrity state of uploaded artifacts.

It does not decide that evidence is sufficient, that a control operated effectively, or that a Program is compliant. Those conclusions remain separate governed records.

## Core objects

- **Evidence Source** — the business identity, authority class, owner and freshness policy for an operational or authoritative origin.
- **Source Connection** — a versioned technical access path beneath an Evidence Source.
- **Source View** — a versioned logical resource exposed through one exact Connection revision.
- **Source Binding** — a versioned, purpose-bound set of permitted operations, fields and limits over one exact View revision.
- **Source Observation** — a timestamped success, degradation or unavailability result.
- **Evidence Request** — a purpose-bound set of unresolved fields for a named subject and audience type.
- **Submission** — immutable answers received through an internal, magic-link or API channel.
- **Invitation** — an opaque, short-lived exchange credential stored only as a hash.
- **Capture Session** — a bounded server-side session created after invitation exchange.
- **Artifact Manifest** — file metadata, storage key, byte count, SHA-256 and inspection state.

## Connected source catalog

```text
Evidence Source
→ Connection revision
→ View revision
→ Binding revision
→ bounded consumer operation
```

Evidence Source remains the business-level authority record. Technical configuration is stored only in the source-access catalog. The former `evidence_sources.endpoint` column has been removed; a legacy endpoint is represented as a non-executable `REFERENCE` Connection.

Connections, Views and Bindings use stable resource IDs plus immutable version rows. Current children reference the exact current parent revision. A stable resource cannot move across tenant, Evidence Source or parent scope. The catalog does not store source records.

The detailed execution and storage contract is defined in [`connected-source-access.md`](connected-source-access.md).

## Source-health path

```text
Evidence Source
→ Observation
→ current / degraded / unavailable
→ freshness evaluation
→ stale when the last success exceeds policy
→ SourceHealthChanged outbox event
→ downstream drift and readiness evaluation
```

Freshness is deterministic. The worker evaluates bounded batches. Source updates and outbox events commit in one PostgreSQL transaction.

Connection-, View- and Binding-level observations are not yet implemented. When introduced, they must reconcile into the existing Evidence Source health path rather than create a second health or Signal/Drift model.

## Request and submission path

```text
Known facts + unresolved fields
→ Evidence Request
→ internal assignment or invitation
→ bounded capture session
→ validated answers
→ immutable Submission
→ request status/version update
→ EvidenceResponseSubmitted outbox event
→ evidence-sufficiency review
```

A Submission is not evidence sufficiency. A stored file is not approved evidence. A completed request is not a verified control outcome.

## Invitation security

Invitation and session tokens contain at least 256 bits of random entropy. Only SHA-256 hashes are stored.

An invitation is:

- tenant and request scoped;
- purpose labelled;
- audience-hinted without displaying a clear address;
- time limited;
- one-time by default;
- revocable;
- invalid when the request is no longer open.

Redemption occurs in one transaction: lock the invitation, re-check revocation, expiry, redemption count and request state, increment the redemption count, then create a bounded session. Failed session creation rolls the transaction back.

The current external-capture path is possession-bound. Verified recipient identity, OTP or step-up authentication and enterprise identity federation are not implemented in this flow and must not be implied by the interface.

## Artifact path

```text
bounded upload stream
→ allowed declared media type
→ object write
→ byte count + SHA-256 during streaming
→ atomic object completion
→ PostgreSQL manifest
→ STORED_UNSCANNED
→ malware/content inspection
→ AVAILABLE or QUARANTINED
```

Artifacts in `STORED_UNSCANNED` are not downloadable or eligible for evidence conclusions. The local filesystem adapter is limited to development and integration testing. Production object storage, encryption policy, malware scanning, content disarm, legal hold and retention workers are not implemented.

## Consistency and performance

Strong consistency is required for:

- Evidence Source creation and initial Reference Connection creation;
- request submission and version change;
- invitation redemption;
- revocation;
- artifact manifest creation after object completion;
- source-health state change and outbox event.

Large files never pass through JSON. Uploads stream with a hard byte limit and bounded multipart overhead.

Initial objectives:

| Operation | Objective |
| --- | ---: |
| source list | p95 ≤ 500 ms for 200 rows |
| request list | p95 ≤ 750 ms for 200 rows |
| invitation redemption | p95 ≤ 500 ms |
| request submission | p95 ≤ 750 ms |
| artifact manifest acknowledgement | p95 ≤ 750 ms after object write |
| source maintenance batch | 50 sources per transaction |

All population reads remain tenant-filtered and bounded. High-volume observation history uses tenant/source/time indexes. Composite foreign keys enforce tenant and parent consistency across the source catalog and capture records.

Operator source lists are also bound to one exact, current legal entity before keyset pagination and limits are applied. A wildcard bank identity must select one exact legal entity; an absent or ambiguous selection fails closed. Program evidence checks and Matter outcome checks revalidate every selected source as active and entity-matched in the material PostgreSQL transaction. A source outage or invalid identifier cannot create a linked contract, while an explicitly manual check with no registered source remains available.

External source reads use adapter-specific sessions rather than ClearSight's application database pool. The PostgreSQL source adapter uses a bounded separate pool, dedicated non-owner credentials, read-only repeatable-read transactions, native parameter types, response-byte limits and operation deadlines.

## Failure behavior

- A duplicate invitation redemption returns the same generic unavailable response as an unknown, expired or revoked token.
- If artifact metadata persistence fails, the newly written object is deleted on a best-effort basis.
- A failed source-health transaction creates neither the state change nor its event.
- A stale request version rejects submission without modifying answers or request state.
- Evidence Source creation and initial Reference Connection creation succeed or fail together.
- A failed source operation does not become an empty result or current evidence.
- Invalid configuration, unsupported capability, partial result, schema mismatch, source failure and caller timeout remain distinct.

## Current limitations

- Source catalog lifecycle transitions and maker-checker administration are not implemented.
- Source catalog API routes and administration UI are not implemented.
- Connection-, View- and Binding-level health reconciliation is not implemented.
- REST/JSON, tabular-file, event and non-PostgreSQL database adapters are not implemented.
- Production object storage, malware inspection, legal hold and retention orchestration are not implemented.
- OTP or step-up authentication and verified external identity are not implemented.
- Evidence contracts, sufficiency evaluation and reusable evidence matching remain separate work.
