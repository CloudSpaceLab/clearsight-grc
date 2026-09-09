# Forms editing and row actions — rendered evidence

This directory retains a local before/after review of the real Forms library, detail sheet and builder. The sample runtime is the isolated evidence build; these captures do not certify hosted deployment or backend permissions.

## Reproduce

Use Node 24 with the repository dependencies, Playwright 1.55.0 at `.codex-tmp/playwright-ci/node_modules/playwright`, and its installed Chromium. From `web`, build with `node node_modules/vite/bin/vite.js build --config vite.evidence.config.ts` and serve the resulting evidence build at `http://127.0.0.1:4188`. Run `capture.mjs` and `capture-canonical.mjs` from this directory. `PAGE_URL` changes the matrix's preview URL. The canonical runner currently uses port 4188.

`capture-before.mjs` uses the pre-change build preserved locally at `.codex-tmp/forms-edit-actions-before-dist`, served on port 4187. Its exact source revision is not certified. Eighteen captures preserve library, detail and editor views in light/dark at 1440, 390 and 320 pixels. Four earlier library captures from the 9 September UI audit are also retained in `before`.

## Scope and checks

The matrix uses `forms-library-lifecycle` in both themes at 1440 × 900, 390 × 844 and 320 × 800. It covers the library, editable draft, published form, published form with a newer draft, pending approval, denied author, unavailable authority, in-flight transition and direct editor entry. The script explicitly records its sample response overrides; they provide visual states without changing production routing or sample source records.

Checks reject page overflow, actions outside the viewport, unavailable or pending-approval editing, an offscreen sheet close action, misplaced editing actions, truncated version status, desktop action stacking, browser errors and reported axe color-contrast violations. Direct editing uses keyboard Enter, checks focus on the named editor region, then returns to the originating row button. Narrow editor checks verify the selected sample question's complete label, full type control and separate Required row, including 44-pixel targets. Extra question screenshots show the scrolled editing state.

The axe results retain incomplete checks. A zero reported contrast-violation count is not a claim that every obscured or offscreen element received an automatic contrast determination. Visual review complements these checks.

## Findings addressed

- The library still reserved a 350-pixel column for the old detail panel. A Forms-specific selector now gives the library its full available width.
- Row actions now align together on desktop and wrap on narrow cards. Version status wraps instead of losing its meaning to an ellipsis.
- The library uses the shared table's container-based card replacement at 700 pixels of available table space. This also prevents tablet-width action headers from being clipped when the page is wider than its table.
- Edit actions sit above version and owner facts in the detail sheet. Pending approval and unavailable authority retain their existing restrictions and explanations.
- The narrow editor inherited 18 pixels of generic builder padding in addition to workspace, canvas, section and question padding. Scoped existing spacing tokens recover the working area, and the type selector and Required control stack vertically.

The final machine receipts record source SHA-256 values and Chromium version. Canonical scenario 89 additionally exercises the unmodified sample runtime's direct edit and keyboard return path. Backend authority, optimistic concurrency, approval separation and deployment require their separate test and release receipts.

## Final focused run

The final Forms source passed 54 matrix states and four additional 1024/768-pixel library checks in Chromium 140.0.7339.16 with Playwright 1.55.0. Canonical scenario 89 passed keyboard editor entry and return, and the Forms scenario definition suite passed all 15 node checks. No matrix case reported page overflow, an inaccessible action, a browser error or an axe contrast violation. The receipts retain incomplete axe determinations and exact source digests.

Reviewed images include `after/library-light-1440.png`, `after/library-dark-390.png`, `after/draft-light-1440.png`, `after/pending-light-320.png`, `after/denied-dark-390.png`, `after/unavailable-light-320.png`, `after/busy-dark-1440.png`, `after/editor-light-320.png` and `after/editor-light-320-question.png`. The editor entry clears the fixed bank header; a second narrow screenshot shows the complete selected question and its type/Required controls after scrolling.

The full existing Forms suite runs through `run-forms-suite.mjs`, which supplies the local pinned Playwright resolver and imports the repository's canonical capture runner without changing its assertions. Its first attempt stopped on a stale template button selector; `suite-initial-selector-failure.json` preserves that failure. Six exact template selectors were updated from “Open” to “Details for”; sent-form Open actions were unchanged. A later partial attempt was stopped before rebuilding the container replacement, so no active capture read a changing build.

The final broader run completed all 75 canonical Forms scenarios without failure. `full-forms/manifest.json` records their viewport, theme, capabilities, layout and scenario-specific metrics; its paired PNGs retain the rendered states. This is the Forms suite only, not the complete application UI suite or a hosted smoke test.
