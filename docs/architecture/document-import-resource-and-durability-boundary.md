# Document import resource and durability boundary

**Status:** implemented by #32 P1.5  
**Scope:** governed document intake, bounded extraction, proposal review, restart/retry truth

## 1. Canonical flow

Production PostgreSQL document intake uses the existing artifact store and runtime outbox rather than a second document-job framework:

```text
HTTP multipart boundary
→ stream original artifact to configured object store
→ one PostgreSQL transaction:
   document_imports(PENDING)
   + DocumentImportProcessingRequested outbox event
→ commit
→ return durable stored/processing receipt

existing outbox worker
→ reopen stored artifact
→ bounded extraction
→ bounded deterministic proposal analysis
→ optimistic terminal processing update
→ mark outbox event published
```

The request path never waits for DOCX/XLSX extraction or proposal analysis. A committed import receipt therefore means the original artifact and processing request are durable; it does **not** mean extraction or analysis succeeded.

Memory/demo repositories deliberately retain synchronous processing so deterministic local tests do not require a background worker.

## 2. Artifact and worker durability

The API and PostgreSQL worker use the same configured artifact root. The worker can therefore restart and reopen an artifact written by the API rather than depending on a process-local memory store.

A horizontally separated deployment must mount or replace that root with storage accessible to both API and worker processes. Production object-storage adapters, malware scanning and retention policy remain enterprise-productization work; P1.5 does not pretend local storage is a distributed object store.

Processing reuses the existing outbox retry/dead-letter machinery. The outbox work class has a bounded timeout/lease and runs independently of Program projection, evidence maintenance, workflow timers and delegation lifecycle. No document-specific queue, timer, receipt or worker framework is added.

## 3. Resource budgets

`ExtractionPolicy` supplies one explicit boundary for compressed-office inputs and semantic materialization. Default hard limits are:

- archive entries: 2,048;
- aggregate expanded archive bytes: 64 MiB;
- per-entry compression ratio: 200:1 for entries at or above the ratio floor;
- worksheets: 64;
- rows: 100,000;
- columns: 256;
- cells: 1,000,000;
- cell/shared-string item bytes: 256 KiB;
- shared strings: 200,000;
- aggregate shared-string bytes: 16 MiB;
- retained extracted text: 8 MiB;
- retained source sections: 5,000;
- retained proposal candidates: 500.

Archive path traversal, declared expanded-size overflow and extreme compression are rejected before OOXML parsing. CSV is consumed row-by-row. XLSX worksheet XML and shared strings are token-streamed rather than decoded into complete worksheet row maps.

Hard structural/resource violations produce a terminal `FAILED` extraction while retaining the original artifact. Retention limits such as section/text/proposal caps produce explicit truncation/omission metadata rather than silently presenting a partial result as complete.

## 4. Completeness contract

Document detail exposes:

- `sections_total` and `sections_omitted`;
- `proposals_total` and `proposals_omitted`;
- `content_truncated`;
- `processed_at`;
- extraction and analysis lifecycle state.

The original artifact hash/storage record remains the reconstruction anchor even when retained review material is bounded.

A proposal total describes candidates discovered from the retained extracted material. When source sections/text were omitted, `content_truncated` and section omission counts explicitly prevent that proposal total from being interpreted as exhaustive over the original artifact.

## 5. Read boundary

Import collection reads use `DocumentSummary`; they do not transfer section text or proposal bodies for every item. Full reconstruction and proposal bodies are detail-only.

The browser can normalize legacy static-demo fixtures into the summary shape, but production PostgreSQL responses populate the explicit completeness fields directly.

## 6. Proposal review boundary

Review is optimistic and proposal-specific:

```text
verified tenant + document id + proposal id + expected document version
→ locate PENDING_REVIEW proposal inside current JSONB array
→ update exactly that array element with status/reviewer/time/note
→ increment document version once
```

The service no longer performs an application-level read/modify/rewrite of the complete proposal array. A stale version conflicts; an already-reviewed proposal cannot be silently reviewed again. The UI refreshes after a conflict so a lost response/retry resolves to the authoritative current record rather than duplicating a review.

Accepting a proposal records human review only. It does not create or approve an obligation, control, legal interpretation or compliance conclusion.

## 7. UI truth

The Imports workspace distinguishes:

- original stored / processing;
- extracted and review required;
- extracted with no proposals;
- stored but unsupported for extraction;
- extraction failed with original retained.

Pending imports are polled while selected. Polling is interval-based, avoids overlapping requests and stops automatically when the detail becomes terminal. A transient poll failure preserves the last durable receipt instead of replacing it with a false terminal state.

## 8. Acceptance evidence

Permanent tests cover:

- durable `PENDING` PostgreSQL receipt plus transactional processing outbox event;
- processing after constructing a fresh service/object-store instance from the same persisted root;
- duplicate processing delivery idempotency;
- atomic single-proposal review and stale-version conflict;
- body-free list summaries;
- archive entry, expanded-size and compression-ratio limits;
- worksheet, row, column, cell and per-cell limits;
- retained-text and section truncation/completeness;
- cancellation;
- rendered processing/review/unavailable states and accessibility.

P1.5 closes the correctness/resource boundary. Wider capacity tuning, distributed object storage, malware scanning, PDF/OCR provider isolation and production retention remain later enterprise work and must be validated from measured deployment requirements rather than added speculatively.
