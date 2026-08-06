# Governed Document Imports

This document describes the executable document-intake capability introduced for issue #15 and the remaining production boundaries from issue #13.

## Purpose

ClearSight can now accept real source files without treating an upload, extraction result or model suggestion as a compliance conclusion.

The import path preserves five distinct things:

1. the immutable original artifact metadata and SHA-256 digest;
2. normalized extracted sections with source-location metadata;
3. deterministic analysis proposals anchored to exact source text;
4. explicit human accept or reject decisions on each proposal;
5. the boundary between an accepted proposal and a governed Program, Requirement, Matter, control or decision.

An accepted proposal means only that an authorized reviewer wants governed follow-up. It does not automatically create or approve a material record.

## Supported files

Current deterministic extraction supports:

- UTF-8 plain text;
- Markdown;
- CSV;
- DOCX paragraphs and tables;
- XLSX worksheets and rows.

PDF originals are stored and hashed, but the current build deliberately reports extraction as unsupported. PDF text extraction and OCR require an approved adapter with page anchors, resource limits and failure evidence.

Image-only documents are not OCRed.

## Analysis contract

The deterministic analyzer proposes bounded candidate statements for:

- possible requirements;
- possible deadlines;
- possible authority references;
- possible control expectations;
- possible risks or consequences.

Each proposal contains:

- proposal type and working title;
- original statement;
- confidence score;
- exact quote;
- section, page, worksheet or row anchor where available;
- review status;
- reviewer identity, time and note after review.

The analyzer is intentionally conservative and remains usable without an AI provider. A future governed AI analyzer may add richer extraction, but it must preserve the same source-anchor, uncertainty, authority and human-review contract.

## Artifact safety

New imports start as `STORED_UNSCANNED`.

`CLEARSIGHT_DOCUMENT_IMPORT_ALLOW_UNSCANNED_ANALYSIS` controls whether deterministic local analysis may run before a production scanning service has marked the artifact available:

- development default: `true`;
- production default: `false`.

When disabled, supported text may still be extracted for the stored import record, but analysis proposals remain unavailable and the limitation is shown to the user.

This flag does not replace production object storage, antivirus/content scanning, encryption-key policy, retention, legal hold or deletion workers.

## Demo mode

`CLEARSIGHT_DEMO_MODE` controls stakeholder reference capability independently from real document imports.

When enabled:

- Nigerian-bank reference journeys may be installed and exposed;
- Explore is shown;
- sample capture and workflow fixtures may be available in development;
- the runtime context reports `reference_journeys: true`.

When disabled:

- reference fixtures are not seeded in the memory composition;
- sample capture requests and workflow tasks are not loaded;
- the bank-journey route is not registered;
- Explore and sample evidence actions are removed from the client;
- real Today, Programs, Work, Imports and Configure surfaces remain available;
- normal import APIs and persisted imports continue to work.

Production refuses to start with demo mode enabled.

## HTTP surface

The focused contract is in [`../../api/document-imports.openapi.yaml`](../../api/document-imports.openapi.yaml).

- `GET /api/v1/document-imports`
- `POST /api/v1/document-imports`
- `GET /api/v1/document-imports/{id}`
- `POST /api/v1/document-imports/{id}/proposals/{proposal_id}/review`

Tenant, legal entity and principal are taken from verified actor context. Client-supplied actor scope is not accepted.

## Persistence

Migration `000011_document_imports` stores:

- tenant and legal-entity scope;
- original artifact metadata and digest;
- extraction and analysis states;
- limitations;
- normalized sections;
- source-anchored proposals;
- immutable review evidence through optimistic versioning.

Large artifact bytes remain in the configured object store; PostgreSQL stores metadata, state and extracted review data.

## Workbench

The **Imports** workspace supports:

- file upload with purpose and source type;
- bounded import list;
- artifact hash and state inspection;
- visible extraction and analysis limitations;
- proposal source quotes and location anchors;
- explicit accept/reject handling;
- extracted-section inspection;
- loading, empty, error and responsive layouts.

## Acceptance evidence

Required tests cover:

- supported extraction and deterministic proposals;
- unscanned-analysis policy;
- immutable proposal review;
- actor-derived tenant and reviewer identity;
- cross-tenant not-found behavior;
- PostgreSQL migrations and repository composition;
- TypeScript strict checking and production build;
- demo-on and demo-off route/client behavior.

## Remaining production work

This slice does not claim:

- production malware and active-content scanning;
- PDF text extraction or OCR;
- password-protected document handling;
- resumable multipart upload;
- saved mapping and reconciliation workflows for repeated high-volume imports;
- accepted-proposal conversion into versioned governed records;
- AI-assisted legal interpretation;
- retention, legal hold and deletion execution;
- production object-store durability or key management;
- high-volume import benchmarks.

Those items remain release-gate work and must not be inferred from the existence of the import workbench.
