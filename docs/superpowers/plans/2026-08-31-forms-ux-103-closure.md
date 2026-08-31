# Governed Forms Issue #103 Closure Plan

> **For Codex:** Execute this plan test-first. Keep the current form lifecycle, authority boundaries, and advanced-filter query model intact.

**Goal:** Close the verified functional and responsive gaps that remain after the latest governed-forms changes, then produce enough automated and rendered evidence to identify any acceptance work that still cannot be completed in code.

**Architecture:** Preserve the existing `FormsWorkspace` orchestration and bounded server-side query model. Reuse `FocusedSheet` for modal focus management, keep one canonical table DOM that changes presentation on narrow screens, and add pointer reordering without removing the existing keyboard move controls. All material lifecycle commands remain server-authorized.

**Tech stack:** Go, React, TypeScript, Vitest, Testing Library, Playwright evidence scripts, CSS.

---

### Task 1: Restore the filter accessibility contract

**Files:**
- Modify: `web/src/components/forms/filters/FormStatusScopes.tsx`
- Modify: `web/src/components/forms/filters/filterRegistry.ts`
- Test: `web/src/components/forms/filters/FormStatusScopes.test.tsx`

1. Run the focused status-scope test and retain the reproduced accessible-name failure.
2. Update the test to require explicit count-bearing accessible names and business-facing approval wording.
3. Add explicit accessible labels so visible label/count markup cannot concatenate unpredictably.
4. Replace the ambiguous `In review` pending-approval label with `Awaiting approval` wherever the filter registry exposes it.
5. Re-run the focused test and the filter-model tests.

### Task 2: Preserve selected forms and manage drawer focus

**Files:**
- Modify: `web/src/components/FormsWorkspace.tsx`
- Modify: `web/src/components/forms/dashboard/TemplateDetailDrawer.tsx`
- Test: `web/src/components/FormsWorkspace.dashboard.test.tsx`
- Test: `web/src/components/FormsWorkspace.location.test.tsx`

1. Add a failing test proving that clearing filters preserves the selected form ID and reloads the selected record.
2. Add a failing test proving that the detail surface is a labelled modal dialog, receives focus, traps focus, closes with Escape, and restores focus to the invoking row.
3. Keep the target ID when clearing filters and replace the URL with the cleared query plus the same target.
4. Rebuild the detail drawer on the shared `FocusedSheet` primitive while preserving its loading, unavailable, transition, edit, and clear-filter states.
5. Re-run the dashboard, location, and shared-sheet tests.

### Task 3: Replace mobile table overflow with a stacked record layout

**Files:**
- Modify: `web/src/components/forms/dashboard/TemplateLibraryTable.tsx`
- Modify: `web/src/forms-dashboard-table.css`
- Test: `web/src/components/FormsWorkspace.dashboard.test.tsx`
- Modify: `web/scripts/forms-evidence-scenarios.mjs`
- Modify: `web/scripts/capture-forms-evidence.mjs`

1. Add cell labels required for a meaningful stacked presentation and test that every populated row retains form, owner, program, response, update, status, and action context.
2. At the mobile breakpoint, remove the hard minimum table width and present each row as a stacked card while retaining the single canonical table and its row action.
3. Add a populated 390px dashboard evidence scenario and assert that the workspace does not introduce horizontal document overflow.
4. Render and inspect the populated mobile state; correct the highest-impact density, clipping, or action-placement issue.

### Task 4: Make form reordering operable and keep all mobile authoring actions available

**Files:**
- Modify: `web/src/components/forms/builder/FormCanvas.tsx`
- Modify: `web/src/components/FormBuilder.tsx`
- Modify: `web/src/form-builder-workspace.css`
- Test: `web/src/components/FormBuilder.test.tsx`
- Modify: `web/scripts/forms-evidence-scenarios.mjs`
- Modify: `web/scripts/capture-forms-evidence.mjs`

1. Add a failing test that drags one question onto another and asserts the saved question order changes.
2. Add an explicit reorder callback and implement pointer drag/drop on the handle; keep Move up/Move down as keyboard alternatives.
3. Remove the enabled no-op button behavior and expose clear drag instructions without making the handle the only reorder path.
4. Replace the narrow-screen toolbar hiding rule with a wrapped action layout that keeps Preview, Save draft, Submit, and Publish available with 44px targets.
5. Add 320px and 390px populated-builder evidence scenarios and assert Preview is visible and the page has no horizontal document overflow.
6. Re-run the builder tests and inspect the rendered mobile states.

### Task 5: Synchronize decision evidence and acceptance boundaries

**Files:**
- Modify: `docs/design/forms-ux-decision-brief.md`
- Modify: `docs/superpowers/plans/2026-08-27-governed-form-system.md`
- Modify: `web/scripts/forms-evidence-scenarios.mjs`
- Modify: `web/scripts/capture-forms-evidence.mjs`

1. Record the implemented mobile table replacement, dialog focus behavior, filter-target continuity, pointer/keyboard reorder behavior, and exact evidence scenarios.
2. Distinguish automated acceptance from external representative-user timing and hosted end-to-end evidence that code cannot manufacture.
3. Generate the evidence manifest and confirm every claimed scenario has an image and measured assertion.

### Task 6: Run closure gates and review the final diff

**Files:** All changed files.

1. Run `gofmt` on changed Go files, if any, and `git diff --check`.
2. Run the affected Go tests, the complete web suite, web build, copy-quality regression, and any repository lint/type checks used by CI.
3. Run the forms evidence capture at desktop, 390px, and 320px and inspect the materially affected images.
4. Request an independent code review of the final diff and fix confirmed findings.
5. Re-run every affected gate after review fixes.
6. Report precisely what is complete and separately list any remaining external acceptance, deployment, or authority-dependent work before issue #103 can be closed.
