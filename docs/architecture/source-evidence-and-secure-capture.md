# Source, evidence and secure capture architecture

## Purpose

This layer maintains the operational facts behind compliance conclusions. It records where facts came from, whether the source is current, what information was requested, who or what submitted it, and the integrity state of uploaded artifacts.

It does **not** decide that evidence is sufficient, that a control operated effectively, or that a Program is compliant. Those conclusions remain separate governed records.

## Core objects

- **Evidence Source** — an authoritative or operational origin with ownership, authority class and expected freshness.
- **Source Observation** — a timestamped success, degradation or unavailability result.
- **Evidence Request** — a purpose-bound set of unresolved fields for a named subject and audience type.
- **Submission** — immutable answers received through an internal, magic-link or API channel.
- **Invitation** — an opaque, short-lived exchange credential stored only as a hash.
- **Capture Session** — a bounded server-side session created after invitation exchange.
- **Artifact Manifest** — file metadata, storage key, byte count, SHA-256 and inspection state.

## Reusable connected-source access

An Evidence Source remains the canonical business and authority identity. Reusable technical access sits beneath it:

```text
Evidence Source
→ Source Connection
→ Source View
→ Source Binding
→ assurance, forms, evidence, workflows or AI governance
```

The executable T0 boundary is documented in [`connected-source-access.md`](connected-source-access.md).

A Connection/View/Binding is not another source registry and does not copy a source population into ClearSight. It records a reusable technical path, native logical resource and purpose-bound mapping/read contract. The same Binding can be referenced by several product consumers without copying a query, URL, credential or field mapping into each workflow.

Source-access operation receipts establish what connection, view, binding, adapter and schema version produced a bounded result. They do not establish evidence sufficiency or a compliance conclusion. A consuming evidence or workflow domain persists a receipt only when its own reconstruction contract requires it.

## Source-health path

```text
Source definition
→ Observation
→ Current / degraded / unavailable
→ Freshness timer
→ Stale when last success exceeds policy
→ SourceHealthChanged outbox event
→ downstream drift/readiness evaluation
```

Freshness is deterministic. AI is not required to determine that a source missed its expected update interval.

The worker evaluates only bounded batches. Source updates and their outbox events commit in one PostgreSQL transaction.

Connection/View/Binding observations may later roll up into this existing Source health path. They must not create a second connector-health product or parallel Signal/Drift path.

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
→ later evidence-sufficiency review
```

A submission is not evidence sufficiency. A stored file is not approved evidence. A completed request is not a verified control outcome.

## Magic-link security

Invitation tokens and session tokens contain at least 256 bits of random entropy. Only SHA-256 hashes are stored.

An invitation is:

- tenant and request scoped;
- purpose labelled;
- audience-hinted without storing a clear address for display;
- time limited;
- one-time by default;
- revocable;
- invalid when the request is no longer open.

Redemption occurs in one transaction: lock invitation, re-check revocation, expiry, redemption count and request state, increment redemption count, then create a bounded session. Failed session creation rolls the transaction back.

The current foundation is possession-bound. Verified recipient identity, OTP/step-up authentication and enterprise identity federation remain separate work and must not be implied by the UI.

## Artifact path

```text
bounded upload stream
→ allowed declared media type
→ local/object-store write
→ byte count + SHA-256 during streaming
→ atomic object completion
→ PostgreSQL manifest
→ STORED_UNSCANNED
→ future malware/content inspection
→ AVAILABLE or QUARANTINED
```

Artifacts in `STORED_UNSCANNED` are not downloadable or eligible for evidence conclusions. The local filesystem adapter exists for development and integration testing only. Production object storage, encryption-key policy, malware scanning, content disarm, legal hold and retention workers are not yet implemented.

## Consistency and performance

Strong consistency is required for:

- request submission and version change;
- invitation redemption;
- revocation;
- artifact manifest creation after object completion;
- source-health state change and outbox event.

Large files never pass through JSON. Uploads stream with a hard byte limit and bounded multipart overhead.

Initial objectives:

| Operation | Objective |
|---|---:|
| source list | p95 ≤ 500 ms for 200 rows |
| request list | p95 ≤ 750 ms for 200 rows |
| invitation redemption | p95 ≤ 500 ms |
| request submission | p95 ≤ 750 ms |
| artifact manifest acknowledgement | p95 ≤ 750 ms after object write |
| source maintenance batch | 50 sources per transaction |

All population reads remain tenant-filtered and bounded. High-volume observation history uses tenant/source/time indexes. Composite foreign keys enforce tenant and request consistency across sources, observations, invitations, sessions, submissions and artifacts even when application repositories are bypassed.

External source reads use adapter-specific pools/sessions, limits and cancellation rather than ClearSight's authoritative application database pool. The current PostgreSQL source adapter is read-only, repeatable-read and bounded by connection, statement, row, field, byte and time ceilings.

## Failure behavior

- A duplicate invitation redemption returns the same generic unavailable response as an unknown, expired or revoked token.
- If artifact metadata persistence fails, the newly written object is deleted on a best-effort basis.
- A failed source-health transaction creates neither the state change nor its event.
- A stale request version rejects submission without modifying answers or request state.
- API or worker unavailability does not change existing source, request or artifact records.
- A failed source-access operation does not become an empty result or current evidence.
- An invalid Binding, unsupported capability, partial result, schema mismatch, source failure and caller timeout remain distinct.

## Production work still required

- durable governed Source Connection/View/Binding lifecycle after the shared T0 contract is proven by real consumers;
- REST/JSON, tabular-file and webhook/event adapters driven by pilot source requirements;
- production object-storage adapter and encryption policy;
- malware scanning and quarantine release workflow;
- legal hold, retention and deletion orchestration;
- OTP/step-up and verified external identity;
- protected-report identity/content separation;
- resumable multipart uploads and large import jobs;
- evidence contracts, sufficiency evaluation and reusable evidence matching.
