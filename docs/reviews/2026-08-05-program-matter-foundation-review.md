# Program, Matter and operational-read review — 2026-08-05

## Scope reviewed

- Program, Requirement, applicability, safeguard and evidence-check aggregates;
- calculated Program status;
- trigger deduplication and linked issue creation;
- typed issue/change lifecycle, decisions, actions, response packages and outcome checks;
- closure and reopening;
- PostgreSQL tenant integrity, optimistic versions, event history and outbox;
- Programs and Issues/Changes user surfaces;
- plain-language copy;
- repeated list, search, pagination and lazy-detail paths.

## Foundation findings resolved

1. Programs begin in **Setup in progress** and cannot become active without an accountable owner, authority, rationale and at least one approved requirement.
2. Program status is calculated from recorded facts and includes reasons; it is not a manually selected colour.
3. Evidence checks validate approved sources, freshness, coverage, independence and contradiction handling.
4. A repeated trigger cannot create duplicate open work for the same Program and dedupe key.
5. Action implementation remains separate from a confirmed outcome.
6. Closure is typed. Findings, gaps, regulatory changes, exceptions and authority responses have different completion requirements.
7. Failed outcome checks follow an explicit response: reopen, request a decision, create follow-up work or block closure.
8. Programs and issues can be reconstructed from append-only events at a requested timestamp.
9. Composite tenant foreign keys prevent cross-tenant Program, requirement, safeguard, source and issue relationships.
10. Primary UI language uses familiar working terms while APIs and audit history retain precise codes.

## Copy and UI findings corrected during release review

The release-candidate review also corrected:

- sample work appearing silently when the Today API failed;
- an invented claim that six Programs had no material changes;
- overdue work being counted as due soon;
- readiness remaining indefinitely in a loading state;
- an old passed outcome result hiding a newer failed result;
- a UI-only `VERIFIED` action status not supported by the domain;
- raw enum values and vague labels on working screens;
- Program status statements implying a wider population than was loaded;
- issue facts appearing only as counts rather than the actual recorded facts.

Primary-screen language includes **Does this apply?**, **Evidence incomplete**, **Decision needed**, **Work completed; outcome not confirmed**, **Confirming outcome**, **Population not connected**, and **Issues and changes**. The canonical term **Matter** remains in specialist detail, APIs and audit history.

## Performance problem corrected

The first list implementation loaded full aggregates and replayed each event stream. On PostgreSQL this created an N+1 read path and the web client also loaded Programs, issues, sources and configuration before the user opened those workspaces.

The finishing phase adds:

- bounded Program and issue summary endpoints;
- one SQL statement per summary page;
- latest-state and latest-outcome lateral reads;
- opaque keyset cursors rather than offset pagination;
- tenant-leading sort indexes;
- generated full-text search documents and GIN indexes;
- detail reconstruction only after a row is opened;
- first-use loading for hidden workspaces;
- server-backed search, filters and load-more controls;
- page-scoped totals and explicit unavailable/retry states;
- a k6 summary-page smoke profile.

Full event replay remains the authoritative detail and historical path. Summary reads do not replace closure evidence or point-in-time reconstruction.

## Acceptance evidence

The reviewed implementation passed:

- `gofmt`;
- race-enabled Go tests;
- PostgreSQL-tagged composition;
- migrations through `000009` on PostgreSQL 18.4;
- PostgreSQL integration using 250 Programs and 300 issues;
- cursor stability and duplicate-page checks;
- generated-index search checks;
- HTTP summary, copy and invalid-cursor contracts;
- `go vet`;
- TypeScript type-check;
- Vite production build.

## Remaining production boundaries

- caller-supplied actor identity is not yet bound to authenticated identity;
- authority routing is not invoked automatically for every material command;
- multi-record commands such as issue creation plus initial links still need explicit transactional review;
- linked Program refresh and follow-up creation need stronger atomic failure contracts;
- representative 100,000-row p95/p99 evidence and retained query plans are not complete;
- correlated summary counts may require dedicated maintained projections under sustained high-cardinality load;
- Program template publication, bulk setup and shared-control dependency propagation remain future work.

These are release gates, not implied capabilities.
