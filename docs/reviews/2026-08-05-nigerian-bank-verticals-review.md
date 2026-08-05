# Nigerian bank verticals implementation review

Date: 5 August 2026

## Review findings

### Existing demo was not an end-to-end acceptance journey

The prior demo created a privacy Program and an isolated gap. It did not prove regulatory-change decisions, protected regulator correspondence, response acknowledgement, independent outcome checking or typed closure.

### Initial vertical read design was too expensive

The first design located four records by loading and reconstructing up to 200 Programs and 200 issues. That would repeat event replay and correlated reads for records unrelated to Explore.

The implemented design uses exact indexed lookups and loads only the four relevant aggregates.

### Restricted content could have leaked through general lists

Tenant scoping alone is insufficient for sensitive authority correspondence. The generic issue and evidence-request read paths now apply record-level visibility checks based on the verified actor and the Matter's explicit principal allow-list.

### Sample data needed a clear boundary

Reference records are marked `sample: true` and Explore labels them as reference data. Product copy does not describe them as the bank's complete compliance position or as legal advice.

### Completion needed verified outcomes

The legacy finding now closes only after an independent outcome check passes. The regulatory-change journey remains open until its implementation result is evidenced. The authority request closes only after acknowledgement.

## Implemented scope

- four connected journey projections;
- exact Program, issue, request and source lookups;
- idempotent memory/PostgreSQL sample seed;
- five-obligation Nigeria data-protection Program;
- focused evidence collection for three unresolved high-risk changes;
- regulatory-change decision/action/outcome path;
- restricted authority response and acknowledgement path;
- legacy finding to independently verified closure;
- Today projection from open journey actions;
- lazy Explore workspace with semantic vector icons;
- restricted issue and evidence-request read filtering;
- memory, HTTP and PostgreSQL acceptance tests;
- OpenAPI, implementation-plan and contributor-rule updates.

## Explicit boundaries

This phase does not claim:

- a complete Nigerian banking regulatory library;
- automatic legal interpretation;
- direct NDPC or CBN source ingestion;
- production document classification or legal privilege controls;
- external authority-channel transmission;
- identity-group synchronization for restricted teams;
- production-scale journey benchmarking;
- that reference fixture dates remain current after 5 August 2026.
