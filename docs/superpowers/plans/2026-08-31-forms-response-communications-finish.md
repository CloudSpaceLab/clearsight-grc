# Forms Response and Communications Finish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Correct shared dark-mode button contrast and replace the weak Responses and Communications compositions with accessible reusable selection and focused-form patterns.

**Architecture:** Remove the legacy unlayered reset that overrides component tokens, add one shared `SelectableRecord` primitive, and compose the existing Forms APIs into a two-column response review and two centered communication dialogs. No API or workflow state changes are required.

**Tech Stack:** React 19, TypeScript, React Aria Components, Vitest/Testing Library, CSS cascade layers and three-layer design tokens, Playwright evidence runner.

---

### Task 1: Protect shared button contrast from legacy cascade rules

**Files:**
- Modify: `web/src/styles.css`
- Modify: `web/scripts/ui-contract.nodecheck.mjs`
- Modify: `web/src/design-system/tokens/components.css`
- Modify: `DESIGN.md`

- [ ] **Step 1: Write the failing cascade-contract test**

Add a node check that reads `src/styles.css` and rejects unlayered global button font or foreground resets:

```js
test("legacy global CSS cannot override shared button typography or foreground", () => {
  const legacy = read("src/styles.css");
  assert.doesNotMatch(legacy, /button\s*,[^{}]*\{[^{}]*font\s*:/s);
  assert.doesNotMatch(legacy, /button\s*\{[^{}]*color\s*:/s);
});
```

- [ ] **Step 2: Run the contract and verify RED**

Run: `cd web && npm run check:ui-contracts`

Expected: FAIL identifying `button, input, select, textarea { font: inherit; }` or `button { color: inherit; }` in `styles.css`.

- [ ] **Step 3: Remove the conflicting legacy reset and document token ownership**

Delete those two declarations from `styles.css`; `design-system/reset.css` already owns inherited control fonts. Add explicit button selected-record tokens to `components.css`, and document that shared action foregrounds are owned by `--cs-button-*-text` in `DESIGN.md`.

- [ ] **Step 4: Run the contract and verify GREEN**

Run: `cd web && npm run check:ui-contracts`

Expected: all node checks pass.

- [ ] **Step 5: Commit**

```bash
git add web/src/styles.css web/scripts/ui-contract.nodecheck.mjs web/src/design-system/tokens/components.css DESIGN.md
git commit -m "fix(ui): preserve shared button contrast"
```

### Task 2: Add an accessible selectable-record component

**Files:**
- Create: `web/src/components/ui/SelectableRecord.tsx`
- Create: `web/src/components/ui/SelectableRecord.test.tsx`
- Modify: `web/src/components/ui/index.ts`
- Modify: `web/src/design-system/components/actions.css`
- Modify: `web/src/components/ui-gallery/UIComponentGallery.tsx`
- Modify: `web/ui-contract-migrations.json`

- [ ] **Step 1: Write the failing component test**

Cover title, metadata, supporting text, selected state and press behavior:

```tsx
render(<SelectableRecord title="Acme review" metadata="Open · 26 Sep 2027" description="2 versions" isSelected onPress={onPress}/>);
const record = screen.getByRole("button", { name: /Acme review/ });
expect(record).toHaveAttribute("aria-pressed", "true");
expect(record).toHaveAttribute("data-selected", "true");
fireEvent.click(record);
expect(onPress).toHaveBeenCalledOnce();
```

- [ ] **Step 2: Run the test and verify RED**

Run: `cd web && npm test -- src/components/ui/SelectableRecord.test.tsx`

Expected: FAIL because `SelectableRecord` does not exist.

- [ ] **Step 3: Implement the component and token-driven states**

Build on React Aria `Button`, expose `title`, `metadata`, optional `description`, `isSelected`, `onPress` and `aria-label`, and render only `.cs-selectable-record` class names. Define default, hover, pressed, selected, focus and disabled states using component tokens; add the component to the gallery, exports and migration manifest.

- [ ] **Step 4: Run the focused tests and contracts**

Run: `cd web && npm test -- src/components/ui/SelectableRecord.test.tsx && npm run check:ui-contracts`

Expected: focused test and UI contract suite pass.

- [ ] **Step 5: Commit**

```bash
git add web/src/components/ui/SelectableRecord.tsx web/src/components/ui/SelectableRecord.test.tsx web/src/components/ui/index.ts web/src/design-system/components/actions.css web/src/components/ui-gallery/UIComponentGallery.tsx web/ui-contract-migrations.json
git commit -m "feat(ui): add selectable record component"
```

### Task 3: Recompose Responses as a master/review workspace

**Files:**
- Modify: `web/src/components/forms/ResponsesView.tsx`
- Modify: `web/src/components/forms/Task11FormsViews.test.tsx`
- Modify: `web/src/forms-task11.css`

- [ ] **Step 1: Write failing response hierarchy assertions**

Extend the existing fixture test to require labelled distribution and version regions, selectable records, a current-version label and read-only amendment guidance:

```tsx
expect(screen.getByRole("region", { name: "Response distributions" })).toBeInTheDocument();
expect(screen.getByRole("region", { name: "Version history" })).toBeInTheDocument();
expect(screen.getByRole("button", { name: /Revision 2.*Current/ })).toHaveAttribute("aria-pressed", "true");
expect(screen.getByText(/Submitted versions cannot be changed/)).toBeInTheDocument();
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `cd web && npm test -- src/components/forms/Task11FormsViews.test.tsx`

Expected: FAIL because the regions and selectable-record semantics are absent.

- [ ] **Step 3: Implement the two-column hierarchy**

Use `SelectableRecord` for distributions and revisions. Keep distributions in a bounded master column; place version history above the selected revision facts in the review column. Replace the disabled edit button with a `Notice` explaining that an amendment is the valid update path. Preserve loading, errors, empty states and cursor pagination.

- [ ] **Step 4: Add responsive token-driven layout CSS**

Replace the three-column response grid with `minmax(260px, 320px) minmax(0, 1fr)`, card surfaces and a compact version rail. At `max-width: 1050px`, stack the columns and remove sticky assumptions.

- [ ] **Step 5: Run focused test and verify GREEN**

Run: `cd web && npm test -- src/components/forms/Task11FormsViews.test.tsx`

Expected: all Forms view tests pass.

- [ ] **Step 6: Commit**

```bash
git add web/src/components/forms/ResponsesView.tsx web/src/components/forms/Task11FormsViews.test.tsx web/src/forms-task11.css
git commit -m "feat(forms): clarify response history review"
```

### Task 4: Move communication profile creation into a focused dialog

**Files:**
- Modify: `web/src/components/forms/CommunicationsView.tsx`
- Modify: `web/src/components/forms/Task11FormsViews.test.tsx`
- Modify: `web/src/components/ui/TextField.tsx`
- Modify: `web/src/components/ui/TextField.test.tsx`

- [ ] **Step 1: Write failing profile-dialog and date-field tests**

Assert that pressing `Create profile revision` keeps Communications mounted, opens a dialog named `Create profile revision`, exposes shared date-time fields, and closes through `Cancel` with focus restored.

```tsx
fireEvent.click(await screen.findByRole("button", { name: "Create profile revision" }));
expect(screen.getByRole("heading", { name: "Communications" })).toBeInTheDocument();
expect(screen.getByRole("dialog", { name: "Create profile revision" })).toBeInTheDocument();
fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
expect(screen.queryByRole("dialog", { name: "Create profile revision" })).not.toBeInTheDocument();
```

- [ ] **Step 2: Run focused tests and verify RED**

Run: `cd web && npm test -- src/components/forms/Task11FormsViews.test.tsx src/components/ui/TextField.test.tsx`

Expected: FAIL because profile creation replaces the workspace and `datetime-local` is not supported by `TextField`.

- [ ] **Step 3: Extend `TextFieldType` and implement the profile dialog**

Add `date`, `time` and `datetime-local` to `TextFieldType`. Render `ProfileEditor` inside `FocusedDialog`, use shared fields for every input, add grouped explanatory copy, and provide Cancel plus primary `Save profile revision` actions. Do not close on a failed save.

- [ ] **Step 4: Run focused tests and verify GREEN**

Run: `cd web && npm test -- src/components/forms/Task11FormsViews.test.tsx src/components/ui/TextField.test.tsx`

Expected: both files pass.

- [ ] **Step 5: Commit**

```bash
git add web/src/components/forms/CommunicationsView.tsx web/src/components/forms/Task11FormsViews.test.tsx web/src/components/ui/TextField.tsx web/src/components/ui/TextField.test.tsx
git commit -m "feat(forms): focus communication profile revisions"
```

### Task 5: Move template revisions into the wide focused dialog

**Files:**
- Modify: `web/src/components/forms/CommunicationsView.tsx`
- Modify: `web/src/components/forms/CommunicationTemplateEditor.tsx`
- Modify: `web/src/components/forms/Task11FormsViews.test.tsx`
- Modify: `web/src/forms-task11.css`

- [ ] **Step 1: Write failing template-dialog assertions**

Assert that `Create template revision` opens a wide dialog, retains the Communications heading beneath it, includes `Cancel` and `Save template revision`, and closes without an API call when cancelled.

- [ ] **Step 2: Run the focused test and verify RED**

Run: `cd web && npm test -- src/components/forms/Task11FormsViews.test.tsx`

Expected: FAIL because the editor currently replaces the tab and has no cancel action.

- [ ] **Step 3: Make the editor composable and open it in `FocusedDialog`**

Add `onCancel` to `CommunicationTemplateEditor`. Use shared subject/date fields, shared quiet/secondary toolbar actions where practical, and a sticky dialog footer with Cancel and primary Save. In `CommunicationsView`, keep the workspace mounted and render the editor in `FocusedDialog size="wide"`.

- [ ] **Step 4: Improve Communications selection list**

Use `SelectableRecord` for template revisions and preserve the existing lifecycle, preview, impact, rollback and test-send commands. Keep `Create template revision` primary and `Create profile revision` secondary.

- [ ] **Step 5: Run focused tests and verify GREEN**

Run: `cd web && npm test -- src/components/forms/Task11FormsViews.test.tsx`

Expected: all view and communication command tests pass.

- [ ] **Step 6: Commit**

```bash
git add web/src/components/forms/CommunicationsView.tsx web/src/components/forms/CommunicationTemplateEditor.tsx web/src/components/forms/Task11FormsViews.test.tsx web/src/forms-task11.css
git commit -m "feat(forms): focus communication template revisions"
```

### Task 6: Extend rendered evidence and complete regression verification

**Files:**
- Modify: `web/scripts/forms-evidence-scenarios.mjs`
- Modify: `docs/design/forms-ux-decision-brief.md`
- Modify: `docs/design/ui-component-adoption.md`
- Modify: `DESIGN.md`

- [ ] **Step 1: Add evidence scenarios and capability assertions**

Add dark 1440×900 scenarios for the primary Create form CTA, the improved Responses workspace, the profile dialog and the template dialog. In the browser runner, compute the primary button foreground/background contrast and require at least 4.5:1 for normal text.

- [ ] **Step 2: Update design and adoption records**

Record `SelectableRecord`, the communication dialog composition, the response review layout and the remaining Imports/rich-text legacy boundaries.

- [ ] **Step 3: Run focused and full verification**

Run:

```bash
cd web
npm test
npm run typecheck
npm run build
npm run check:ui-contracts
npm run review:ui
```

Expected: zero test failures, successful typecheck/build/contracts, and all rendered capability, accessibility, contrast and bundle gates pass.

- [ ] **Step 4: Inspect retained frames manually**

Inspect the dark primary CTA, Responses, profile dialog and template dialog at original resolution. Confirm hierarchy, contrast, clipping, focus visibility, scroll containment and dominant-action order.

- [ ] **Step 5: Restore generated presentation assets and check the diff**

```bash
git restore -- docs/presentation-assets/clearsight-premium-first-run-cover.png
git diff --check
git status --short
```

Expected: no whitespace errors; local `node_modules/` remains untracked and unstaged.

- [ ] **Step 6: Commit**

```bash
git add web/scripts/forms-evidence-scenarios.mjs docs/design/forms-ux-decision-brief.md docs/design/ui-component-adoption.md DESIGN.md
git commit -m "test(forms): prove response and communication finish"
```
