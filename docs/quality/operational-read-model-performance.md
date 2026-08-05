# Operational read-model performance gate

This gate covers the repeated Program and Issues/Changes list paths introduced after the event-backed foundation.

## Required scenarios

1. First Program-summary page with 50 rows.
2. Second Program-summary page using the returned cursor.
3. Program search by code/name token.
4. First open-Matter summary page with 50 rows.
5. Second Matter-summary page using the returned cursor.
6. Matter search by reference/title token.
7. Open one Program detail after the list is visible.
8. Open one Matter detail after the list is visible.

## Data profiles

- CI: at least 250 Programs and 300 Matters in one tenant.
- Pre-release: at least 100,000 Programs/Matter rows across tenants, with realistic state snapshots, links, actions and outcome results.
- Tenant isolation: a second tenant with similarly named records must never appear.

## Initial objectives

| Operation | Objective |
|---|---:|
| Program summary page | p95 ≤ 500 ms |
| Matter summary page | p95 ≤ 500 ms |
| Indexed summary search | p95 ≤ 750 ms |
| Cursor page | p95 ≤ 500 ms |
| Program detail | p95 ≤ 1.5 s |
| Matter detail | p95 ≤ 1.5 s |

The CI integration gate verifies correctness, cursor stability and indexed schema composition. Environment-scale p95/p99 evidence is required before production release.

## Query requirements

- exactly one bounded SQL statement for each summary page;
- no reads from `continuity_events` on summary endpoints;
- keyset pagination rather than offset;
- tenant predicate at the start of every indexable path;
- generated full-text document for supported search fields;
- no protected evidence or arbitrary JSON in search documents;
- `EXPLAIN (ANALYZE, BUFFERS)` retained for release evidence on representative data.

## UI requirements

- Today loads independently from Programs, Work and Configure;
- Program and Matter summary APIs run only when their workspace is opened;
- row expansion fetches only the selected aggregate;
- list failure preserves navigation and offers retry;
- pagination appends without duplicating rows;
- all displayed totals state whether they cover the loaded page or a governed full population.
