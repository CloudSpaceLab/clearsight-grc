# Form response browser

## Decision and baseline

Program owners and form reviewers compare submitted responses, identify concerning or unscored results, and open the submitted answers, documents and assessment. The supplied Archer reference informs the filter rail and comparable result columns.

Before: Program submitted data is a four-column table with 20-row cursor loading and no filters or sorting controls. Forms has a priority selector, collapsed filters, seven columns and 25-row cursor loading. Both already use a focused review sheet. The existing screenshots under `docs/quality/screenshots/program-follow-up` preserve the Program baseline; this structure inventory records the Forms baseline. New renders cover both entry points.

Selected approach: reuse the response query, shared fields, badges, table and review sheet. A desktop filter rail accompanies results. Mobile replaces the rail with a Filters disclosure and stacks result cells. A toolbar contains title search, priority, batch size and optional columns. Sortable headers request supported stored-score/submission ordering. Title search applies before pagination and within existing actor/entity access. Unknown scores stay unknown; coverage is never a completion or document-validity claim.

Alternatives: a toolbar-only arrangement is smaller but makes repeated refinement harder; a general-purpose report builder is excessive for this task.

## Scope and behavior

- Share query controls between Program submitted data and Forms responses; retain exact Program subject scope.
- Search literal title text, case-insensitively, with a 200-byte query limit. PostgreSQL uses parameterized escaped ILIKE; memory reads match literal text. Keep keyset pagination and page sizes of 20/25/50/100 (Forms defaults to 25).
- Filters: concern, score meaning, score state, submission date range; Forms additionally offers subject type. Clear removes filters while retaining the selected batch size and priority. Date errors remain recoverable before any request.
- Sort: concern descending, newest submission, raw score ascending/descending. Header buttons expose the active direction through aria-sort. Do not invent unsupported title or historical ordering.
- Columns: form identity and review always remain; other result columns can be hidden without changing query scope or ordering. Review remains one visible action per row.
- Refresh/filters invalidate pending pages; stale requests cannot append records to a different query or Program. A failed next page retains prior rows and offers retry. Program changes close previous response detail.
- Scores, concern, scoring coverage and current/historical revisions remain separate facts. No progress bars imply review or document validation.

## Plan and proof

1. Add failing title-filter, sortable-header and Program query regression tests; run them.
2. Extend the existing query in HTTP, memory and PostgreSQL; add client serialization.
3. Add shared response controls and column selection, compose both entry points, and protect cursor/detail races.
4. Synchronize design/API documentation and evidence-only fixtures.
5. Run focused Go and web suites, typecheck/build, copy, runtime and UI-contract checks. Render desktop and narrow layouts in both themes plus empty/error/review states; inspect and fix the highest-impact defect.

Maturity: existing bounded response browsing extended with filters and presentation; no new workflow commands, permissions, scoring rules, saved-view storage, export, or delivery behavior.

Verification and limitations: [release proof](../quality/response-browser-2026-10-05.md).
