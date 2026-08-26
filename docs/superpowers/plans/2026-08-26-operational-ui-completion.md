# Operational UI Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver and merge a complete, reachable and visually coherent Work/Programs/Imports experience whose static deployment remains truthful and fully operable.

**Architecture:** Finish the stateful static adapter behind the existing typed clients, then apply semantic input and surface tokens across existing components. Preserve live React behavior and exact backend authority while using targeted regressions, recursive bundle accounting and rendered-state review as release gates.

**Tech Stack:** Go, PostgreSQL-tagged repositories, React 19, optional static-only Preact compatibility, TypeScript, Vite, Vitest, Testing Library, Playwright and CSS custom properties.

---

### Task 1: Complete and harden the static workflow transport

**Files:** `web/src/staticDemo.ts`, `web/src/staticDemoBootstrap.ts`, `web/src/staticDemoWorkflowRuntime.js`, `web/src/staticDemo.test.ts`, `web/vite.config.ts`, `web/package.json`, `web/scripts/review-ui-flow-manifest.mjs`

- [ ] Add failing regressions for collected evidence request persistence, per-check evaluation truth, created-record isolation, runtime load recovery and exact base-path assets.
- [ ] Run `npx vitest run src/staticDemo.test.ts` and confirm the new cases fail for the missing behavior.
- [ ] Implement per-record state, exact evaluation fields, persisted request list/detail handling, Vite-hashed asset URLs and a retryable single-flight loader.
- [ ] Make the review script recursively count every delivered JavaScript file and fail above 163,840 gzip bytes.
- [ ] Run the targeted tests, typecheck, static build twice and compare hashes and exact total gzip bytes.
- [ ] Commit the transport as one reviewed unit.

### Task 2: Audit and correct semantic form controls

**Files:** `web/src/components/*.tsx`, affected component tests

- [ ] Inventory date/deadline/effective/observed/review fields and identify text inputs that represent defined temporal values.
- [ ] Add focused tests asserting the correct native input types and value conversions.
- [ ] Replace free-text temporal fields with `date`, `datetime-local` or `time`, preserving ISO API payloads and local display semantics.
- [ ] Verify keyboard labels, min/max constraints, error placement and 44px targets.
- [ ] Run affected component tests and typecheck.

### Task 3: Apply the focused visual system

**Files:** `web/src/styles.css`, `web/src/ui-preferences.css`, `web/src/continuity.css`, `web/src/*finish.css`, Imports/document components, `DESIGN.md`

- [ ] Add semantic document, overlay, focus, icon-target and typography tokens for both themes.
- [ ] Apply paper surfaces to document reading/review content, restrained scrim/backdrop blur to actual focused overlays, and consistent compact SVG sizing inside accessible targets.
- [ ] Ensure body copy is at least 16px/1.5 where it carries workflow instructions and eliminate sub-12px operational text.
- [ ] Update `DESIGN.md` with the new token and surface behavior.
- [ ] Run copy quality and accessibility tests.

### Task 4: Render and repair end-to-end states

**Files:** `web/ui-evidence/*`, rendered-state scripts and any components/CSS exposed by inspection

- [ ] Render Work, Program, monitoring, evidence, Imports and overlay states at desktop, tablet and phone sizes in light and dark themes.
- [ ] Inspect navigation, clipping, focus order, target size, date pickers, document contrast, blur/scrim strength and reduced motion.
- [ ] Fix the highest-impact defect, rerender and repeat until the review manifest passes with no material issue.

### Task 5: Verify, commit and merge

**Files:** all changed files; `main` working tree

- [ ] Run `go test ./... -count=1`, `go test -tags postgres ./... -count=1`, and both vet commands.
- [ ] Run full web tests, typecheck, live build, exact static Pages build, recursive gzip gate and UI review.
- [ ] Confirm `git diff --check`, clean feature worktree and review commit history.
- [ ] Merge `codex/program-work-operational-completeness` into local `main` without overwriting unrelated main changes.
- [ ] Rerun the release matrix on merged `main`, then remove the owned worktree and feature branch only after success.
