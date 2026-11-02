# ClearSight — Integrations & Data Sources

Screen 13 · Design prototype · 15 October 2026

## Information hierarchy

1. Scope: verified entity/group selectors in the shell (sample-only local selection).
2. Three focused views: **Sources**, **Field mapping**, **Sync history**.
3. Sources: current health, last checked, business use and selected-source details.
4. Mapping: original columns alongside target fields; changes remain drafts until review.
5. History: period-scoped run outcomes; current connection state is not recomputed from the chosen period.

## Design rules

- No permanent Copilot panel. Assistant recommendations are optional and must reference actual source health or mapping decisions when implemented.
- Display connection freshness independently from import approval or compliance status.
- An import must preserve blank cells and the source's original coordinates/values.
- Connecting a source requires authorization, credential protection, tenant scope and an explicit activation review; the prototype accepts drafts only.
- Source statuses and figures in screenshots are fictional examples, not evidence of working vendor integrations.
- Reuse ClearSight's existing Source → Connection → View → Binding model and Integrations/Configure surfaces during implementation; this mockup does not justify new back-end resources.
- Supported layouts: 1880, 1280, 768 and 390 CSS pixels. No horizontal overflow in browser tests. Motion respects prefers-reduced-motion.

## Prototype interactions

- Switch Sources / Field mapping / Sync history.
- Filter sources by query, source type, status, and group. Selecting a source updates details.
- Review suggested field mappings and save a **local draft**.
- Change Sync history range (7/30/90 days).
- Open Connect source drawer, choose a type and save a **local draft**.
- Keyboard: Ctrl/Cmd+K search, Escape drawer, focus containment.

No network calls, data writes or real account access occur in the prototype.
