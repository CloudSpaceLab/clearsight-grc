# ClearSight UI Foundations and Sent Forms Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish the first production-grade ClearSight UI foundation, prove every supported state in a production-component gallery, and migrate the Sent forms workflow so its controls, filters, empty state, data presentation and responsive detail behavior are usable, themed and enforceably consistent.

**Architecture:** Keep existing Forms APIs, commands, authority behavior, route keys and distribution data contracts unchanged. Add a three-layer token system and ClearSight-owned component boundary over exact `react-aria-components@1.20.0`, then compose Sent forms from focused feature components. Legacy CSS remains available to non-migrated screens through compatibility aliases; an explicit migration manifest applies stricter source checks only to completed files.

**Tech Stack:** React 19, TypeScript 7, React Aria Components 1.20.0, CSS cascade layers and custom properties, Vitest, Testing Library, axe-core, Node test runner, Playwright evidence harness, Vite.

---

## Scope boundary

This plan implements Tranche 1 of the approved design: tokens, shared components, gallery, enforcement, Forms navigation tabs and the Sent forms screen. It does not migrate the immersive builder, Templates, Responses, Imports, Communications or unrelated product workspaces. Those retain compatibility styling until their own plans are approved.

The command and data boundary is unchanged:

```text
FormsWorkspace
  -> FormsNavigation (shared Tabs contract)
  -> SentFormsView (query, load, selection and command orchestration)
       -> SentFormsFilters (FilterBar + fields)
       -> SentFormsTable (DataTable + StatusBadge)
       -> SentFormDetail (status, facts and permitted commands)
       -> FocusedSheet below the inline-detail breakpoint
       -> existing DistributionComposer / DistributionChangePanel
```

## Task 1: Pin the interaction dependency and establish the token cascade

**Files:**

- Modify: `web/package.json`
- Modify: `web/package-lock.json`
- Modify: `web/src/main.tsx`
- Modify: `web/src/ui-preferences.css`
- Modify: `web/src/test/setup.ts`
- Create: `web/src/design-system/index.css`
- Create: `web/src/design-system/reset.css`
- Create: `web/src/design-system/base.css`
- Create: `web/src/design-system/utilities.css`
- Create: `web/src/design-system/overrides.css`
- Create: `web/src/design-system/tokens/index.css`
- Create: `web/src/design-system/tokens/primitives.css`
- Create: `web/src/design-system/tokens/semantic.css`
- Create: `web/src/design-system/tokens/components.css`
- Create: `web/src/design-system/components/index.css`
- Test: `web/scripts/ui-contract.nodecheck.mjs` (created in Task 7)

- [ ] Before changing source, retain the existing Sent forms render and bundle result. Run the existing fixture capture, keep `99-forms-distribution-access-history-light-1440x900.png` as the pre-migration comparison, and record the current `review.json` initial JavaScript/CSS gzip values for the final evidence note.

- [ ] Install the exact unstyled interaction package from `web`:

```bash
npm install --save-exact react-aria-components@1.20.0
```

Expected: `package.json` contains `"react-aria-components": "1.20.0"` and the lockfile resolves one exact package version. Do not install React Spectrum or a visual theme package.

- [ ] Add the global layer order and imports in `design-system/index.css`:

```css
@layer reset, tokens, base, components, features, utilities, overrides;

@import "./reset.css" layer(reset);
@import "./tokens/index.css" layer(tokens);
@import "./base.css" layer(base);
@import "./components/index.css" layer(components);
@import "./utilities.css" layer(utilities);
@import "./overrides.css" layer(overrides);
```

`tokens/index.css` imports primitives, semantic roles and component roles in that order. `components/index.css` imports the component style families added by later tasks.

- [ ] Add only minimal box sizing/form-font normalization to `reset.css`, document typography and non-component element defaults to `base.css`, and the existing screen-reader-only helper to `utilities.css`. Keep `overrides.css` empty except for a comment explaining that temporary compatibility rules require a separately reviewed migration; do not move the legacy application stylesheet into a layer in this tranche.

- [ ] Define only raw scales in `primitives.css`: neutral/brand/status ramps, 4px spacing steps, the approved type/line-height scale, 6/10/14/full radii, 1/2px borders, 16/20/24px icons, four elevations, 100/150/200/300ms durations, easing and the z-index ladder. Use the `--cs-primitive-*` prefix.

- [ ] Define role names in `semantic.css`, including this minimum contract:

```css
:root {
  --cs-space-page: var(--cs-primitive-space-8);
  --cs-space-section: var(--cs-primitive-space-6);
  --cs-space-group: var(--cs-primitive-space-4);
  --cs-space-inline: var(--cs-primitive-space-2);
  --cs-control-height-comfortable: 44px;
  --cs-control-height-compact: 40px;
  --cs-target-min: 44px;
  --cs-focus-width: var(--cs-primitive-border-2);
  --cs-focus-offset: 2px;
  --cs-content-wide: 1680px;
  --cs-reading-measure: 72ch;
}
```

Theme-specific colors map to semantic names such as `--cs-bg-canvas`, `--cs-bg-surface-*`, `--cs-text-*`, `--cs-border-*`, `--cs-action-primary-*`, `--cs-status-*`, `--cs-focus-ring` and `--cs-overlay-backdrop`. Component and feature CSS must not reference primitive colors directly.

- [ ] Define Button, Field, Select, Tabs, overlay, feedback, surface, FilterBar and DataTable internal roles in `components.css`. Every component value must reference a semantic token; page or domain names are prohibited.

- [ ] Refactor the palette/density block at the top of `ui-preferences.css` so System/Light/Dark and Comfortable/Compact map semantic roles to primitives. Preserve current public aliases (`--canvas`, `--surface`, `--border`, `--text`, `--muted`, `--cyan`, and the document/illustration variables) by mapping them to the new semantic roles. This keeps non-migrated screens stable while migrated components consume only `--cs-*` roles.

- [ ] Enforce the target rule in CSS: comfortable is 44px; compact is 40px only under `(pointer: fine)`; touch, mobile and reflow restore at least 44px.

- [ ] Import `./design-system/index.css` in `main.tsx` immediately before `ui-preferences.css`. Do not reorder unrelated feature styles in this tranche.

- [ ] Add deterministic jsdom shims required by React Aria to `src/test/setup.ts`:

```ts
class TestResizeObserver implements ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}

Object.defineProperty(globalThis, "ResizeObserver", { configurable: true, value: TestResizeObserver });
Object.defineProperty(window, "matchMedia", {
  configurable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener() {},
    removeListener() {},
    addEventListener() {},
    removeEventListener() {},
    dispatchEvent: () => false,
  }),
});
```

Add a `PointerEvent` fallback only when jsdom does not provide it.

- [ ] Run `npm run typecheck && npm run build` from `web`.

Expected: both pass, and the existing application still builds with the static-demo Preact aliases.

- [ ] Commit:

```bash
git add web/package.json web/package-lock.json web/src/main.tsx web/src/ui-preferences.css web/src/test/setup.ts web/src/design-system
git commit -m "feat(ui): establish token and cascade foundations"
```

## Task 2: Build Button, feedback and surface primitives test-first

**Files:**

- Create: `web/src/components/ui/Button.tsx`
- Create: `web/src/components/ui/Button.test.tsx`
- Create: `web/src/components/ui/StatusBadge.tsx`
- Create: `web/src/components/ui/Notice.tsx`
- Create: `web/src/components/ui/Surface.tsx`
- Create: `web/src/components/ui/EmptyState.tsx`
- Create: `web/src/components/ui/Feedback.test.tsx`
- Create: `web/src/components/ui/index.ts`
- Create: `web/src/design-system/components/actions.css`
- Create: `web/src/design-system/components/feedback.css`
- Create: `web/src/design-system/components/surfaces.css`
- Modify: `web/src/design-system/components/index.css`

- [ ] Write failing Button tests for semantic role/name, the closed `primary | secondary | quiet | destructive` variant contract, disabled behavior and loading repeat-click prevention. The loading assertion must preserve the label while exposing status:

```tsx
it("keeps the action name and prevents repeat activation while loading", () => {
  const run = vi.fn();
  render(<Button variant="primary" isLoading onPress={run}>Send form</Button>);
  const button = screen.getByRole("button", { name: "Send form" });
  fireEvent.click(button);
  expect(run).not.toHaveBeenCalled();
  expect((button as HTMLButtonElement).disabled).toBe(true);
  expect(screen.getByRole("status", { name: "Send form in progress" })).toBeTruthy();
});
```

- [ ] Run `npm test -- src/components/ui/Button.test.tsx`.

Expected: FAIL because the UI boundary and Button do not exist.

- [ ] Implement `Button` by wrapping React Aria's Button inside the UI boundary. Keep the public props closed and do not expose raw color/radius/style props:

```tsx
export type ButtonVariant = "primary" | "secondary" | "quiet" | "destructive";
export type ButtonSize = "comfortable" | "compact";

export type ButtonProps = Omit<AriaButtonProps, "className" | "style" | "children"> & {
  variant?: ButtonVariant;
  size?: ButtonSize;
  isLoading?: boolean;
  children: ReactNode;
};
```

Use React Aria state data attributes for hover, pressed, focus-visible and disabled styling. Loading must not change outer dimensions.

- [ ] Implement `IconButton` in the same file with a required `aria-label` and the minimum target contract. It may accept only the approved Button variants and icon size tokens. Implement the visual `link` action as `ActionLink`, a React Aria Link wrapper that requires a real `href`; do not allow a button callback to masquerade as navigation.

- [ ] Write failing feedback/surface tests proving:

  - StatusBadge always has text and maps a closed tone to a non-color-only marker;
  - Notice uses `role="alert"` for error and `role="status"` for other tones;
  - EmptyState names the checked population, result and next valid action; and
  - Surface/Card do not add interactive semantics.

- [ ] Implement the four components with closed tones (`neutral`, `info`, `success`, `warning`, `error`, `unknown`) and semantic classes. Product status-code translation remains the feature's responsibility.

- [ ] Add action, feedback and surface styles using component tokens only. Include forced-colors and reduced-motion rules. No literal color, radius, shadow, control-height, duration or z-index may appear in these files.

- [ ] Export the components only from `components/ui/index.ts`, and run:

```bash
npm test -- src/components/ui/Button.test.tsx src/components/ui/Feedback.test.tsx
```

Expected: PASS.

- [ ] Commit:

```bash
git add web/src/components/ui web/src/design-system/components
git commit -m "feat(ui): add action feedback and surface primitives"
```

## Task 3: Build field and themed Select contracts test-first

**Files:**

- Create: `web/src/components/ui/FormField.tsx`
- Create: `web/src/components/ui/TextField.tsx`
- Create: `web/src/components/ui/TextArea.tsx`
- Create: `web/src/components/ui/SelectField.tsx`
- Create: `web/src/components/ui/Field.test.tsx`
- Create: `web/src/components/ui/SelectField.test.tsx`
- Create: `web/src/design-system/components/fields.css`
- Modify: `web/src/components/ui/index.ts`
- Modify: `web/src/design-system/components/index.css`

- [ ] Write failing field tests for label/description/error relationships, required copy, read-only versus disabled semantics, invalid state and long labels. Use accessible queries rather than CSS selectors.

- [ ] Implement `FormField` as the shared label/description/error anatomy and implement TextField/TextArea with React Aria field mechanics. Public props accept business labels and messages, not prebuilt IDs. Keep mobile input text at 16px or larger.

- [ ] Define the bounded Select API with a stable option type:

```tsx
export type SelectOption<T extends string> = {
  id: T;
  label: string;
  description?: string;
};

export type SelectFieldProps<T extends string> = {
  label: string;
  value?: T;
  placeholder: string;
  options: readonly SelectOption<T>[];
  onChange: (value: T | undefined) => void;
  description?: string;
  errorMessage?: string;
  isDisabled?: boolean;
  isRequired?: boolean;
};
```

- [ ] Write failing SelectField tests for opening, arrow navigation, Home/End, typeahead, Enter/Space selection, Escape cancellation, selected-value announcement and focus restoration. Include an axe check for an open listbox composition.

- [ ] Run:

```bash
npm test -- src/components/ui/Field.test.tsx src/components/ui/SelectField.test.tsx
```

Expected: FAIL before the implementations exist.

- [ ] Implement SelectField using React Aria `Select`, `Button`, `SelectValue`, `Popover`, `ListBox` and `ListBoxItem`. The popup must use the shared overlay z-index/surface/elevation tokens and must not import feature CSS.

- [ ] Add field/select CSS for default, filled, focus-visible, invalid, disabled, read-only and loading states. Verify dark-theme popup color comes exclusively from semantic variables.

- [ ] Re-run the focused tests and `npm run typecheck`.

Expected: PASS.

- [ ] Commit:

```bash
git add web/src/components/ui web/src/design-system/components
git commit -m "feat(ui): add accessible field and select contracts"
```

## Task 4: Build Tabs and FocusedSheet contracts test-first

**Files:**

- Create: `web/src/components/ui/Tabs.tsx`
- Create: `web/src/components/ui/Tabs.test.tsx`
- Create: `web/src/components/ui/FocusedSheet.tsx`
- Create: `web/src/components/ui/FocusedSheet.test.tsx`
- Modify: `web/src/components/FocusedSheet.tsx`
- Modify: `web/src/components/FocusedSheet.test.tsx`
- Create: `web/src/design-system/components/navigation.css`
- Create: `web/src/design-system/components/overlays.css`
- Modify: `web/src/components/ui/index.ts`
- Modify: `web/src/design-system/components/index.css`

- [ ] Write failing Tabs tests for selected semantics, Left/Right/Home/End roving focus, automatic activation and focus-visible without a second selected indicator. Use this closed item shape:

```ts
export type TabItem<T extends string> = { id: T; label: string };
```

- [ ] Implement Tabs with React Aria Tabs/TabList/Tab/TabPanel. The wrapper owns the peer-panel relationship. At compact widths, CSS changes to a wrapped or menu-backed replacement before horizontal clipping; it must not become an uncontrolled native select.

- [ ] Port the current FocusedSheet contract into `components/ui/FocusedSheet.tsx` using React Aria ModalOverlay/Modal/Dialog. Preserve label, Escape dismissal, outside dismissal, focus containment, body scroll lock and trigger focus restoration. Keep `panelClassName` only for layout composition; do not expose backdrop styling as a feature escape hatch.

- [ ] Replace `components/FocusedSheet.tsx` with a compatibility re-export:

```ts
export { FocusedSheet } from "./ui";
export type { FocusedSheetProps } from "./ui";
```

This keeps existing imports working while the reusable owner moves under the UI boundary.

- [ ] Move the existing sheet assertions to the new test and retain a compatibility test proving old imports render the same labelled dialog.

- [ ] Add token-only navigation and overlay CSS, including full-height mobile sheet replacement, reduced motion, safe-area padding and a forced-colors boundary.

- [ ] Run:

```bash
npm test -- src/components/ui/Tabs.test.tsx src/components/ui/FocusedSheet.test.tsx src/components/FocusedSheet.test.tsx
```

Expected: PASS with focus restored to the invoker after close.

- [ ] Commit:

```bash
git add web/src/components/ui web/src/components/FocusedSheet.tsx web/src/components/FocusedSheet.test.tsx web/src/design-system/components
git commit -m "feat(ui): add navigation and focused overlay contracts"
```

## Task 5: Build FilterBar and DataTable composition contracts test-first

**Files:**

- Create: `web/src/components/ui/FilterBar.tsx`
- Create: `web/src/components/ui/DataTable.tsx`
- Create: `web/src/components/ui/DataTable.test.tsx`
- Create: `web/src/design-system/components/data-display.css`
- Modify: `web/src/components/ui/index.ts`
- Modify: `web/src/design-system/components/index.css`

- [ ] Write failing tests for a populated table, selected row, loading, error, pagination and stacked mobile facts. Assert the row has a complete accessible name and every cell exposes its column label through `data-label`.

- [ ] Define the DataTable contract without domain knowledge:

```ts
export type DataColumn<Row> = {
  id: string;
  header: string;
  kind?: "text" | "number" | "status" | "action";
  render: (row: Row) => ReactNode;
  accessibleText: (row: Row) => string;
};

export type DataTableProps<Row> = {
  ariaLabel: string;
  rows: readonly Row[];
  rowKey: (row: Row) => string;
  rowName: (row: Row) => string;
  columns: readonly DataColumn<Row>[];
  selectedKey?: string;
};
```

DataTable renders data only. Empty and error states replace it at the feature composition level so an empty table can never create overflow.

- [ ] Implement FilterBar as a labelled composition with `fields`, optional result count and a clear action. At narrow widths it stacks into full-width groups instead of shrinking controls. It must not reach into field internals.

- [ ] Add DataTable styles for header/row density, hover, selected, focus and aligned status/action cells. At mobile/reflow widths, use the single semantic table DOM and display each populated row as a stacked record. Do not set a permanent table `min-width`.

- [ ] Run:

```bash
npm test -- src/components/ui/DataTable.test.tsx
npm run typecheck
```

Expected: PASS.

- [ ] Commit:

```bash
git add web/src/components/ui web/src/design-system/components
git commit -m "feat(ui): add filter and data presentation contracts"
```

## Task 6: Add the production-component gallery and its state contract

**Files:**

- Create: `web/src/components/ui-gallery/UIComponentGallery.tsx`
- Create: `web/src/components/ui-gallery/UIComponentGallery.test.tsx`
- Create: `web/src/ui-gallery.css`
- Modify: `web/src/main.tsx`
- Modify: `web/src/copyQuality.test.ts` only if the complete gallery copy review exposes a reliably detectable new narration class

- [ ] Write a failing gallery test that renders production exports from `components/ui`, not replicas. Assert labelled sections exist for Actions, Fields, Selection, Navigation, Feedback, Surfaces, Data and Overlays, and that every sample is labelled as sample component data.

- [ ] Add a static-demo-only, lazy-loaded route in `main.tsx` so the gallery is excluded from the ordinary production entry bundle:

```tsx
const uiGalleryEvidence = import.meta.env.VITE_STATIC_DEMO === "true" && fixture === "ui-component-gallery";
```

When true, render `UIComponentGallery` directly inside `DisplayPreferencesRoot`. Import `ui-gallery.css` from that lazy static-demo gallery entry and wrap its rules in `@layer features` so the CSS is both code-split and correctly ordered. The route must not be reachable as an ordinary production workspace.

- [ ] Build the gallery from the barrel exports. Show the supported variants and semantic states: default, disabled, loading, invalid, read-only, selected, long content, empty, error and overlay. Actual hover, pressed, focus and open-popup states are exercised by the browser scenario rather than fake public props.

- [ ] Include concise contract copy for each family: job, allowed variants, keyboard behavior and prohibited substitution. Use operational component language, not a bank-status claim or product-review commentary.

- [ ] Style only gallery layout in `ui-gallery.css`; do not restyle component internals. Support 1440px, 1280px, 1024px, 390px and 320px layouts.

- [ ] Run:

```bash
npm test -- src/components/ui-gallery/UIComponentGallery.test.tsx src/copyQuality.test.ts
npm run typecheck
```

Expected: PASS.

- [ ] Commit:

```bash
git add web/src/components/ui-gallery web/src/ui-gallery.css web/src/main.tsx web/src/copyQuality.test.ts
git commit -m "feat(ui): add production component gallery"
```

## Task 7: Enforce the migrated UI boundary

**Files:**

- Create: `web/ui-contract-migrations.json`
- Create: `web/scripts/ui-contract.mjs`
- Create: `web/scripts/ui-contract.nodecheck.mjs`
- Modify: `web/package.json`
- Modify: `web/src/components/ui-gallery/UIComponentGallery.tsx`
- Modify: `DESIGN.md`

- [ ] Create a manifest with three explicit sets: migrated TSX files, migrated feature/component CSS files and documented native-control exceptions. Also list every shared component family and its supported variants.

- [ ] Implement the TSX scanner with the installed TypeScript compiler API, not a broad JSX regex. For manifested product files, report the exact file and line for raw `button`, `select`, `input` or `textarea`. Permit a native control only when an exact manifest exception names the file, tag, input type and reason.

- [ ] Scan all `web/src` TypeScript/TSX files for direct `react-aria-components` imports outside `components/ui`. Report the approved alternative: import from `components/ui`.

- [ ] Scan manifested CSS declaration-by-declaration for hard-coded colors and raw radius, shadow, control-height, duration or z-index values. Reject selectors combining a feature class with a `.cs-*` component-internal class. Token declaration files are not feature CSS and are validated separately for three-layer naming.

- [ ] Make component/gallery drift executable. Each component family in the manifest must be exported by `components/ui/index.ts`, represented by a `data-component-contract` section in the gallery and documented in `DESIGN.md`. TypeScript closed unions enforce the variant side.

- [ ] Write node tests with temporary fixture strings that prove each violation is caught and each documented native exception is accepted. Include the required diagnostic assertion:

```js
assert.match(result.message, /src\/components\/forms\/sent\/SentFormsFilters\.tsx:\d+.*use SelectField/);
```

- [ ] Add `"check:ui-contracts": "node --test scripts/ui-contract.nodecheck.mjs"` to `package.json`.

- [ ] Run the test before adding migrated files to the manifest.

Expected: PASS for the scanner's fixture suite.

- [ ] Add the component files, gallery and component CSS to the manifest. Do not add Sent forms until Tasks 8–9 remove its raw controls.

- [ ] Run `npm run check:ui-contracts`.

Expected: PASS.

- [ ] Commit:

```bash
git add web/ui-contract-migrations.json web/scripts/ui-contract.mjs web/scripts/ui-contract.nodecheck.mjs web/package.json web/src/components/ui-gallery/UIComponentGallery.tsx DESIGN.md
git commit -m "test(ui): enforce migrated component contracts"
```

## Task 8: Refactor and migrate Sent forms test-first

**Files:**

- Modify: `web/src/components/forms/SentFormsView.tsx`
- Create: `web/src/components/forms/sent/SentFormsFilters.tsx`
- Create: `web/src/components/forms/sent/SentFormsTable.tsx`
- Create: `web/src/components/forms/sent/SentFormDetail.tsx`
- Create: `web/src/components/forms/sent/distributionPresentation.ts`
- Create: `web/src/components/forms/sent/distributionPresentation.test.ts`
- Create: `web/src/components/forms/sent/SentFormsView.test.tsx`
- Modify: `web/src/components/forms/Task11FormsViews.test.tsx`
- Create: `web/src/forms-sent.css`
- Modify: `web/src/design-system/index.css`
- Modify: `web/ui-contract-migrations.json`

- [ ] Extract business labels and tones into pure presentation functions with exhaustive maps. Never render a raw API status or access-policy code:

```ts
export const distributionStatusLabel: Record<DistributionStatus, string> = {
  DRAFT: "Draft",
  READY: "Ready to send",
  OPEN: "Responses open",
  LOCKED: "Responses locked",
  COMPLETED: "Completed",
  EXPIRED: "Expired",
  REVOKED: "Access revoked",
  SUPERSEDED: "Replaced",
};
```

Add equivalent maps for due state and access method. Test exhaustiveness and the visible wording.

- [ ] Add failing SentFormsView tests for these exact behaviors:

  - loading reserves the result region and names the population being loaded;
  - a 401 `ApiError` renders a sign-in-required recovery action;
  - an ordinary load failure names Sent forms and offers Retry;
  - an empty result renders neither table nor distribution detail;
  - filter changes preserve `safe=1`, the `#forms` hash and all `dist_*` query keys after reload;
  - a populated page exposes the partial-page Load more state only when `next_cursor` exists;
  - selecting a row loads that exact distribution;
  - a lifecycle command disables only its active action and reports confirmed success separately from a later refresh failure; and
  - the representative composition passes axe.

- [ ] Run:

```bash
npm test -- src/components/forms/sent/SentFormsView.test.tsx src/components/forms/Task11FormsViews.test.tsx
```

Expected: FAIL on the new behaviors while the existing amendment and supersession tests continue to identify preserved workflow contracts.

- [ ] Keep API/query/command orchestration in `SentFormsView`. Model initial list state as `loading | live | sign-in-required | error`; keep detail loading/error separate so one failed detail read does not erase a valid list.

- [ ] Change refresh selection behavior so a missing/filtered row clears the selection instead of automatically opening the first record. Preserve the current selection only when it remains in the new page.

- [ ] Compose filters with two themed SelectFields and three TextFields. Use the existing query names and write through `history.replaceState`; trimming a value to empty removes only that key.

- [ ] Compose rows with DataTable and StatusBadge. The Open action retains the exact accessible name `Open {title}`. Empty/error/sign-in states replace the data region with EmptyState or Notice and therefore cannot own a table scrollbar.

- [ ] Compose the detail facts/actions in `SentFormDetail`. Preserve the exact `lock`, `reopen`, `revoke`, amendment and supersession command calls and version arguments. Render only server-state-permitted lifecycle actions already supported by the current workflow; the UI does not infer authority.

- [ ] Keep `DistributionComposer` and `DistributionChangePanel` as existing governed subflows. Replace only the entry/cancel actions encountered in the Sent forms composition when they are inside a manifested migrated file; do not silently add their entire files to the migration manifest.

- [ ] Add `forms-sent.css` for feature composition only: heading grid, filter placement, result/detail columns and responsive replacement. Use semantic tokens for every visual value and do not select inside `.cs-*` component classes. Import it from `design-system/index.css` with `layer(features)` so migrated feature composition participates in the declared cascade.

- [ ] Add the migrated Sent forms files and `forms-sent.css` to `ui-contract-migrations.json`, then run:

```bash
npm test -- src/components/forms/sent src/components/forms/Task11FormsViews.test.tsx
npm run check:ui-contracts
npm run typecheck
```

Expected: PASS; the manifest reports no raw controls in migrated files.

- [ ] Commit:

```bash
git add web/src/components/forms/SentFormsView.tsx web/src/components/forms/sent web/src/components/forms/Task11FormsViews.test.tsx web/src/forms-sent.css web/src/design-system/index.css web/ui-contract-migrations.json
git commit -m "feat(forms): migrate sent forms to shared UI contracts"
```

## Task 9: Migrate Forms navigation and responsive master-detail behavior

**Files:**

- Create: `web/src/components/forms/FormsNavigation.tsx`
- Create: `web/src/components/forms/FormsNavigation.test.tsx`
- Modify: `web/src/components/FormsWorkspace.tsx`
- Modify: `web/src/components/FormsWorkspace.test.tsx`
- Modify: `web/src/components/forms/sent/SentFormsView.test.tsx`
- Modify: `web/src/forms-foundation.css`
- Modify: `web/src/forms-sent.css`
- Modify: `web/ui-contract-migrations.json`

- [ ] Write a failing FormsNavigation test proving the five peer sections use tab semantics, arrow-key navigation and one selected indicator. Preserve the current tab names and reset editor/launcher state through the existing `onChange` callback.

- [ ] Replace the raw `.forms-tabs` button loop in FormsWorkspace with `FormsNavigation`. Keep active-tab state in FormsWorkspace and render the current content through the shared Tabs panel contract. Remove only the obsolete `.forms-tabs button` internal styling from `forms-foundation.css`.

- [ ] Add a deterministic `matchMedia` helper inside the Sent forms feature or UI boundary for the inline-detail threshold. Add tests for both branches:

```tsx
setMediaQuery("(min-width: 1180px)", false);
render(<SentFormsView />);
fireEvent.click(await screen.findByRole("button", { name: "Open Quarterly control review" }));
expect(await screen.findByRole("dialog", { name: "Quarterly control review details" })).toBeTruthy();
```

At or above the threshold, selected detail is an inline labelled complementary region. Below it, the same `SentFormDetail` content opens in FocusedSheet and closing the sheet restores focus to the row action.

- [ ] Ensure an unselected populated desktop list may show a concise selection prompt, but mobile renders no empty sheet. An empty result renders neither prompt nor detail region.

- [ ] Test 44px action/field targets by component contract and assert the Sent forms DOM has no legacy `forms-primary`, raw `select`, ordinary text `input` or raw `button` elements authored by migrated feature files.

- [ ] Add FormsNavigation to the migration manifest and run:

```bash
npm test -- src/components/forms/FormsNavigation.test.tsx src/components/FormsWorkspace.test.tsx src/components/forms/sent/SentFormsView.test.tsx src/components/ui/FocusedSheet.test.tsx
npm run check:ui-contracts
```

Expected: PASS.

- [ ] Commit:

```bash
git add web/src/components/FormsWorkspace.tsx web/src/components/FormsWorkspace.test.tsx web/src/components/forms/FormsNavigation.tsx web/src/components/forms/FormsNavigation.test.tsx web/src/components/forms/sent/SentFormsView.test.tsx web/src/forms-foundation.css web/src/forms-sent.css web/ui-contract-migrations.json
git commit -m "feat(forms): standardize navigation and responsive detail"
```

## Task 10: Add deterministic fixtures and full-host rendered contracts

**Files:**

- Modify: `web/src/staticDemo.ts`
- Modify: `web/src/staticDemo.test.ts`
- Modify: `web/scripts/forms-evidence-scenarios.mjs`
- Modify: `web/scripts/forms-evidence-scenarios.nodecheck.mjs`
- Modify: `web/scripts/capture-forms-evidence.mjs`
- Modify: `web/scripts/review-ui-flow-manifest.mjs`

- [ ] Add deterministic fixtures for Sent forms empty, populated/selected, partial page, lifecycle action and unauthorized/error states. Every fixture uses plausible sample records and is explicitly sample data. Do not add persuasive compliance counts.

- [ ] Test the fixture API for exact bounded distribution items, next cursor, selected detail and 401/error envelopes.

- [ ] Extend `requiredFormsCapabilities` and its nodecheck with the foundation/Sent contracts: component variants, themed open Select, keyboard focus, comfortable/compact density, empty replacement, populated table, responsive sheet, partial page, lifecycle feedback, 320px reflow and 200% reflow.

- [ ] Add scenarios 117 onward without renumbering existing evidence:

  - component gallery, light comfortable, 1440×900;
  - component gallery, dark compact with Select open, 1280×720;
  - Sent forms empty, dark, 1440×900;
  - Sent forms populated with responsive detail sheet, light, 1024×768;
  - Sent forms populated, light, 390×844 and 320×800;
  - Sent forms at effective 200% zoom from a 1440×900 physical viewport; and
  - component gallery forced-colors/focus and reduced-motion state.

- [ ] In scenario assertions, measure the defect contract directly:

```js
const primary = await page.getByRole("button", { name: "Send form" }).evaluate((element) => {
  const style = getComputedStyle(element);
  const rect = element.getBoundingClientRect();
  return { height: rect.height, border: style.borderStyle, background: style.backgroundColor };
});
if (primary.height < 44 || primary.border === "none" || primary.background === "rgba(0, 0, 0, 0)") {
  throw new Error("Send form must render as the dominant 44px action.");
}
```

Also assert every field has a visible boundary, the dark Select popup has a dark semantic surface, empty Sent forms has no table/detail/overflow, populated stacked rows fit the viewport and the sheet does not overlap its close/recovery actions.

- [ ] Correct the 200% proxy so it changes the effective CSS layout viewport, not only device pixel ratio. For a scenario with `zoom: 2`, create the context at half the declared physical width/height and device scale factor 2; record both physical and layout viewports in the manifest. Update nodecheck/review validation accordingly.

- [ ] Allow scenario-specific `density`, `forcedColors` and `reducedMotion` values in the capture context and manifest. Compact evidence must use a fine pointer; touch scenarios remain at least 44px.

- [ ] Run:

```bash
node --test scripts/forms-evidence-scenarios.nodecheck.mjs
npm test -- src/staticDemo.test.ts
npm run build
npm run review:ui
```

Expected: all scenarios pass; the manifest has no duplicate names, missing capabilities or horizontal-overflow failures.

- [ ] Inspect every new PNG at original resolution. Identify the highest-impact remaining visual or interaction failure, add a failing component or scenario assertion for it, implement the smallest correction, rerun the affected tests and `npm run review:ui`, then inspect the replacement image again. Record both observations in Task 11 documentation.

- [ ] In addition to the repeatable CI proxy, open the full built host in a real Chromium window at 1440×900, set browser zoom to 200%, and inspect Sent forms with the Select open and with detail active. Record this manual real-browser result separately; do not relabel the automated effective-layout proxy as native browser zoom.

- [ ] Commit:

```bash
git add web/src/staticDemo.ts web/src/staticDemo.test.ts web/scripts/forms-evidence-scenarios.mjs web/scripts/forms-evidence-scenarios.nodecheck.mjs web/scripts/capture-forms-evidence.mjs web/scripts/review-ui-flow-manifest.mjs web/src/forms-sent.css web/src/design-system
git commit -m "test(ui): prove component and sent forms states"
```

## Task 11: Synchronize the design contract, adoption inventory and final evidence

**Files:**

- Modify: `DESIGN.md`
- Create: `docs/design/ui-component-adoption.md`
- Modify: `docs/design/ui-delivery-workflow.md`
- Modify: `docs/quality/rendered-ui-evidence.md`
- Modify: `docs/superpowers/specs/2026-08-31-clearsight-ui-foundations-and-forms-migration-design.md`
- Review: all changed files

- [ ] Update DESIGN.md with the layer order, three token levels, supported semantic roles, component inventory, 44/40px density rule, React Aria ownership boundary, native-control exceptions and migration rule. Remove the older 8px-only/radius guidance where it conflicts with the approved token scale; do not erase still-valid institutional visual direction.

- [ ] Create the adoption inventory with rows for every product workspace and columns for Actions, Fields, Selection, Navigation, Overlays, Feedback, Surfaces and Data. Mark only UI Gallery, Forms navigation and Sent forms as migrated in this tranche. Mark Builder, Templates, Responses, Imports, Communications and all non-Forms workspaces as not yet migrated, with their next approved tranche or separate-plan requirement.

- [ ] Update the delivery workflow to require the migration manifest, gallery state, full-host evidence and highest-impact repair for each newly migrated file.

- [ ] Update rendered evidence with:

  - the retained before-state name and observed defects;
  - exact new scenario names and viewports;
  - the first render's highest-impact defect;
  - the correction and re-render result;
  - before/after initial and total JS/CSS gzip measurements;
  - the exact final Git head; and
  - the explicit statement that later Forms and product surfaces remain outside this tranche.

- [ ] Mark the approved specification's Tranche 1 status as implemented only after every gate below passes. Do not mark the overall Forms migration or product-wide visual consistency complete.

- [ ] Run the focused gates:

```bash
cd web
npm run check:ui-contracts
node --test scripts/forms-evidence-scenarios.nodecheck.mjs
npm test -- src/components/ui src/components/ui-gallery src/components/forms/sent src/components/forms/FormsNavigation.test.tsx src/components/FormsWorkspace.test.tsx src/components/forms/Task11FormsViews.test.tsx src/staticDemo.test.ts src/copyQuality.test.ts
npm run typecheck
npm run build
```

Expected: every command passes with no skipped migrated-component test.

- [ ] Run the complete repository gates from the worktree root:

```bash
go test ./...
cd web
npm test
npm run review:ui
```

Expected: Go tests, all web tests and the complete rendered review pass.

- [ ] Run `git diff --check`, inspect `git diff --stat`, and search the changed docs/source for unresolved markers and prohibited claims:

```powershell
git diff --check
$patterns = @(('TO' + 'DO'), ('TB' + 'D'), ('FIX' + 'ME'), 'bug[- ]free', 'fully complete', 'product-wide consistent')
rg -n ($patterns -join '|') DESIGN.md docs web/src/components/ui web/src/components/forms/sent web/src/forms-sent.css
```

Expected: no unresolved marker or unsupported completion claim.

- [ ] Review copy across the complete affected workflow. Confirm every heading/action names a business task or object, API codes are translated, errors provide recovery, and no gallery copy makes a bank-state claim. Extend `copyQuality.test.ts` only for a reliable semantic anti-pattern found during this review.

- [ ] Review the exact final diff against the approved specification. Confirm no domain command, authority rule, data model, API route or invitation behavior changed.

- [ ] Commit documentation and any verified final correction:

```bash
git add DESIGN.md docs/design/ui-component-adoption.md docs/design/ui-delivery-workflow.md docs/quality/rendered-ui-evidence.md docs/superpowers/specs/2026-08-31-clearsight-ui-foundations-and-forms-migration-design.md web/src/copyQuality.test.ts
git commit -m "docs(ui): record foundation adoption and rendered proof"
```

- [ ] Re-run the focused gates after the final commit and record `git rev-parse HEAD`. Do not report completion from results produced before the final documentation/correction commit.

## Completion report

Report these outcomes separately:

1. Foundation components and enforced migrated boundary completed in code.
2. Sent forms default, loading, sign-in-required, empty, error, populated, partial, selected and lifecycle states verified.
3. Full-host theme, density, responsive, open-overlay, forced-colors, effective 200% CI evidence and manual real-browser 200% evidence inspected on the exact head.
4. Bundle delta and any accepted route-level loading consequence.
5. Remaining work after merge: immersive builder, remaining Forms surfaces, progressive product adoption and representative-user/assistive-technology acceptance that repository automation cannot manufacture.
