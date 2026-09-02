# Operational read models

Programs and Matters retain event-backed authoritative detail. Repeated list screens do not reconstruct every aggregate.

## Read paths

```text
Program list
→ bounded Program summaries
→ latest state snapshot + counts
→ open one Program
→ reconstruct authoritative detail and history

Issues and changes list
→ bounded Matter summaries
→ current row + counts + latest outcome
→ open one item
→ reconstruct authoritative detail and closure basis
```

Summary reads are projections. They may omit detail but must never invent status, totals or authority. Every response includes `generated_at`; pagination cursors are opaque and scoped to the current query.

## Ordering and pagination

Program summaries are ordered by operational status, latest update and UUID. Matter summaries are ordered by priority, latest update and UUID. Both use keyset pagination with a maximum page size of 100. Deep offset pagination is prohibited.

A cursor encodes only the last sort keys from the preceding page. Invalid cursors return a client error and require a first-page reload.

## Search

Programs and Matters use generated PostgreSQL `tsvector` documents with GIN indexes. Search covers only named summary fields:

- Program code, name, owning function and jurisdiction;
- Matter reference, title, summary and type.

Search does not inspect evidence content, protected reports or unrestricted JSON.

## Consistency

Material commands and full aggregate reads remain strongly consistent. Summary lists read current relational rows and latest persisted snapshots. A list may be slightly behind an in-flight command, but it must expose its generation time and a detail open always fetches the authoritative aggregate.

## Performance boundaries

- one SQL statement per summary page;
- no continuity-event replay on list paths;
- no application-memory tenant filtering;
- page size capped at 100;
- summary rows exclude full event streams, evidence bodies and action history;
- full detail is fetched only when the user opens a row;
- inactive workspaces do not load at application startup.

## Failure behavior

A failed summary request displays an unavailable state and retry action. It does not substitute sample data or preserve a prior count without a stale-data label. A failed detail request leaves the list usable and offers a row-level retry.

## Oversight snapshot

The legal-entity oversight snapshot is a bounded 90-day projection over included Matter scope. It records the checked population, excluded and unknown scope counts, generation time, projection version and high-water marks for Matters, actions, workflow tasks, verification results and continuity events. Reassignment and returned-decision counts are derived from append-only events; aggregate-only memory composition leaves those measures unknown.

Resolution ranges require at least five closed Matters of the same type in the selected legal entity and period. The range reports the observed quartiles and median, not a promised completion date. Owner rows show current load, completed/measured work, observed cycle time, SLA attainment, blocked work, reopening, reassignment and returned-decision history. These are operating-context facts and must not be combined into an employee score or rank.
