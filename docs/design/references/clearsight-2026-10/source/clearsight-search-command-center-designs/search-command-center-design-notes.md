# 14 — Search & Command Center

Date: 15 October 2026
Status: Visual and interaction prototype, not connected to ClearSight APIs.

## Design decisions

- **One search model**: the full results screen and `⌘/Ctrl K` quick-search dialog operate on the same record types; no competing global assistant surface.
- **Context first**: legal entity and organization group scope remain visible. Result list and selected record stay in one workspace, with details and permitted next steps beside the selection.
- **Small primary surface**: one search input, record-type tabs, two optional filters, a bounded list and a selected record inspector. No KPI wall, “AI insight” rail, calendar decoration, or vanity counts.
- **Result anatomy**: name, object type, stable ID, group, status, last update. Details add owner, next review/due date, source/record version reference and related records.
- **Commands**: keyboard quick search, arrow navigation, Enter to select, Escape to dismiss, `/` to focus search, record selection and a focused record drawer. Save search is locally simulated in this preview.
- **Responsive**: two-column results/inspector on desktop; inspector below the list on tablets and mobiles; search tabs scroll horizontally on small screens. Reduced-motion behavior is respected.
- **Data accuracy**: all sample results are explicitly marked as preview content. Changing the legal entity demonstrates empty scope rather than leaking another entity's records.

## Production implementation guidance

1. Place `⌘/Ctrl K` behind the existing global search trigger instead of creating a new application module or navigation route.
2. Federate **authorized, indexed, bounded** results across existing read APIs. Each source returns stable object ID, type, display title, freshness, tenant/entity/scope, permission-aware safe snippet, and exact route target; never load full records into the browser to filter or authorize them.
3. Server enforcement precedes query result ranking, counts, suggestions and record preview. Unauthorized results must not leak through facets or quick suggestions. Use fast indexed routes with bounded pagination and cancellation.
4. Display current/partial/stale provenance appropriately; do not interpret search matches as compliance outcomes or invent authority for available actions.
5. Use the existing route, shared controls and focus-management contracts. Search must work without AI. AI may only summarize user-authorized sources after source and policy checks.
6. Track empty, loading, partial, denied, stale, slow-network and no-permission cases, including keyboard, screen reader and mobile evidence.

## Verified in standalone prototype

- Client-side sample search, record-type tabs, status and group filters, sorting, selected-record preview, related-record transitions, local saved query, quick search keyboard actions and Escape dismissal.
- Chromium browser render at 1880, 1280, 768 and 390 CSS pixel widths with no document horizontal overflow; no runtime JS errors.

**Preview scope:** this does not prove backend search coverage, server-side tenant authorization, indexed performance, saved search persistence, or actual navigation into ClearSight records.
