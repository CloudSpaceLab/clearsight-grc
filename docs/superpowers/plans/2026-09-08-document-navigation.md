# Document navigation implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking.

**Goal:** Keep Forms sections and document types easy to choose on narrow screens while preserving the existing Finder experience and readable filenames.

**Architecture:** Opt-in responsive selection on the shared Tabs component reuses SelectField without remounting its content on resize. DocumentBrowser uses the same controlled file-kind state for its desktop sidebar and compact selector. An opt-in full-width mobile DataColumn layout gives names room without changing other tables.

**Tech Stack:** Existing React, TypeScript, React Aria, CSS tokens, Vitest and Playwright evidence runner; no new dependency.

## Task 1: Responsive controls and filename layout

Files: `web/src/components/ui/Tabs.tsx`, `Tabs.test.tsx`, `DataTable.tsx`, `DataTable.test.tsx`; `web/src/design-system/components/navigation.css`, `data-display.css`; `web/src/components/forms/FormsNavigation.tsx`; `web/src/components/documents/DocumentBrowser.tsx`, `DocumentBrowser.test.tsx`, `documents.css`; existing UI component gallery and `DESIGN.md`.

- [ ] Add failing component tests for opt-in selection, one mounted content tree, controlled selected value and full-width mobile column marker. Reuse current test fixtures. The public additions are:

```ts
// TabsProps: absent keeps all other tab workspaces unchanged.
compactLabel?: string;
// DataColumn: absent keeps ordinary stacked key/value rows unchanged.
mobileLayout?: "full-width";
```

- [ ] Run `node node_modules/vitest/vitest.mjs run src/components/ui/Tabs.test.tsx src/components/ui/DataTable.test.tsx src/components/documents/DocumentBrowser.test.tsx`; verify new assertions fail before implementation.
- [ ] Add a selector bound to the existing Tabs `selectedKey` and `onSelectionChange`. Use SelectField with no empty choice, existing item IDs/labels, and a guard against unchanged selections. Keep one TabPanel subtree at all viewport widths. CSS exposes the selector and hides only the tab list at the compact breakpoint; desktop remains unchanged. Preserve correct accessible panel identity and keyboard behavior.

```tsx
<SelectField label={compactLabel} value={selectedKey} placeholder={compactLabel}
  options={items} allowsEmpty={false}
  onChange={(key) => { if (key !== undefined && key !== selectedKey) onSelectionChange(key); }}/>
```

- [ ] Opt Forms into the variant using `compactLabel="Forms section"`. In DocumentBrowser render an equivalently controlled SelectField labelled `File type`, sharing `kind` and clearing `pages` on an actual type change. At <=760px show the selector and hide the file-type sidebar; do not duplicate content, query effects or data state. Reuse current file-type labels and icon/sidebar selected state on desktop.
- [ ] Render `data-mobile-layout={column.mobileLayout}` on cells and add the shared mobile rule below; opt only the document Name column into it. Other metadata remains stacked key/value. Do not hide or abbreviate filenames.

```css
/* Inside the existing <=700px table replacement. */
.cs-data-table tbody tr td[data-mobile-layout="full-width"] {
  grid-template-columns: minmax(0, 1fr);
}
```

- [ ] Verify component/workflow tests, no duplicate selected callbacks, and unchanged query/body scope. Update DESIGN.md and gallery to describe/render both opt-in variants, using existing tokens only. Commit only task files after verification.

## Task 2: Render and history acceptance

Files: `web/scripts/forms-evidence-scenarios.mjs`, its existing nodecheck contract if capability list changes, and this plan's verification receipt.

- [ ] Make the existing `openFormsTab` helper select the visible compact control when present, otherwise the desktop tab; do not weaken section/history assertions. Adapt document Word selection likewise. Keep all existing scenarios and gates.
- [ ] Add browser assertions that only the correct navigation is visible at the current width, selected labels agree with state, all types remain selectable, filenames have full card width, and resizing with selected/preview content does not reset it. Retain Space/Escape focus restoration, unavailable/empty behavior and scoped API checks.
- [ ] Preserve existing 320/390/1440px document baseline images outside the runner output directory and back up the unrelated presentation cover before running the full review; restore the exact cover bytes in a finally block.
- [ ] Run Node 24 `npm test`, `npm run typecheck`, `npm run check:ui-contracts`, `npm run check:runtime-truth`, then `npm run review:ui` without concurrent test CPU contention. Inspect all affected light/dark renders; repair highest-impact failure and re-run. Record actual counts and limitations, not inferred visual completeness.
- [ ] Obtain specification review, then code-quality review; fix and repeat any findings. Run relevant Go/full release gates. Push a bounded PR, verify exact-head CI, merge and verify main/deployment plus hosted navigation before recording release. Do not close #200 or the separate sample-data/vendor work on this slice.

## Related approved work

The [approved scope](../specs/2026-09-08-document-navigation-and-samples-design.md) also requires vendor activation recovery/handoffs and connected sample data. Those need independent code-path-specific plans and verification; no placeholder implementation is included in this navigation change.
