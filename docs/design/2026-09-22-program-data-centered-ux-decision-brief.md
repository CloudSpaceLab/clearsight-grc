# Program data-centred UX decision brief

**Decision date:** 2026-09-22

**Status:** Approved for implementation

## Product job

A Program owner manages an ongoing obligation and the operational data collected for it
(for example, operational-risk responses from a branch). The Program record must make the
collected/submitted data the core of review: what was asked, who submitted it, when,
what answers and documents came back, when documents expire, and how evidence checks
confirm the Program outcome.

## Problem inventory

1. **Bloated Programs list.** Each row expands into a full detail pane inside the portfolio
   (status reasons, review digest, requirements and evidence expectation lists). The list
   duplicates the Program record, is slow to scan, and buries the one thing each row needs:
   go open the record.
2. **"Monitoring" names the wrong concept.** The cohort/section that collects structured
   responses is titled with a control-systems word while its job is collecting data from
   assigned staff. New users look for "Data collection".
3. **"Evidence & results" reads as empty.** The tab opens to evidence-check *contracts*
   (configuration) which renders as an empty state for new users. Users expect the submitted
   form data, artifacts and documents there.
4. **No deep navigation to collected data.** The Program offers no route from the record to
   completed responses, their documents (which carry `expires_on`) and their previews.

## Target structure

- **Programs list:** compact, scannable rows — identity (code · function · jurisdiction),
  operating/calculated state, and three bounded counts (requirements · evidence checks ·
  open issues). The row itself opens the Program record; inline expansion is removed. URLs,
  filters, hash persistence and keyset pagination stay.
- **Section label:** *Monitoring → Data collection* (section id `monitoring` stays stable so
  existing deep links keep working).
- **Evidence & results:** submitted responses for the Program are primary and come first
  (bounded read: `loadCompletedResponses({ subject_type: "PROGRAM", subject_id })`); the
  existing evidence-check contracts are supporting configuration underneath, headed
  *Evidence checks and results*. Selecting a response opens a wide `FocusedSheet` with
  **Answers / Documents / Review** tabs reusing `ResponseAssessment` and `DocumentBrowser`
  (documents show expiry and previews), matching the established production pattern in
  `ResponsesView`.

## State matrix

| Surface | Required fixtures |
| --- | --- |
| Programs list | loaded rows with counts; no programs in scope; filters match nothing; unavailable with retry; load-more; dense population at 1440/390/320; targeted record hand-off |
| Data collection section | label renders in desktop tabs and the mobile selector; collection records still show responder, expiry and currency states |
| Evidence & results | no responses yet (honest empty state naming the population and the next action in Data collection); responses with answers; response with documents showing `expires_on`; records the current person can review; unavailable with retry; deep sheet open at each tab; sheet on narrow viewports |

## Copy decisions

- Section label **Data collection** (id `monitoring`); other section labels unchanged.
- Submitted-data panel headed **Submitted data**; empty state is honest (population checked +
  current result + next action: start a collection in Data collection).
- Keep **Evidence & results** and **Evidence checks and results** as the section and the
  supporting panel heading.
- Other copy passes the customer-facing copy gate (`web/src/copyQuality.test.ts`): bank
  working language, no pitch or product-review commentary, Unknown denominators shown as
  Unknown.

## Non-goals

- No backend/API changes; reuse existing bounded APIs only.
- No program-wide document inventory load (response-scoped `DocumentBrowser` only).
- No new component families, design tokens, density modes or motion patterns.
- No URL or section-id changes.

## Implementation streams

| Stream | Scope |
| --- | --- |
| A — list de-bloat | `ProgramsWorkspace.tsx`, continuity.css list styles, `AppViews.tsx` header copy if needed, its tests |
| B — Data collection rename | `ProgramDetailSections.tsx` label, `MonitoringSetup.tsx`, `CollectionRecord.tsx`, `CollectionPolicyForm.tsx`, their tests |
| C — Evidence & results rebuild | `ProgramRecordWorkspace.tsx` evidence-results composition, new `ProgramResponsesPanel.tsx`, its CSS and tests, `ProgramRecordWorkspace.test.tsx`, deep navigation |

## Proof requirements

- Typecheck, targeted vitest + axe, copy-quality regression, runtime-truth and UI-contract
  checks run after integration.
- Rendered evidence at representative viewports (1440×900 desktop light/dark, 390×844 and
  320×800 mobile, plus the repository's 200% zoom proxy where the workflow supports it),
  including the empty and populated Evidence & results states and the deep sheet, following
  `docs/design/ui-delivery-workflow.md`.