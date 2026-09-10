# Vendor Exception Overview Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace oversized vendor KPI cards and repeated finding cards with a compact, prioritized exception overview that links to canonical Matters.

**Architecture:** Keep `VendorPortfolio` as the data-loading boundary and add pure presentation helpers for attention ranking, next-action selection and filtering. Render a compact summary/filter region followed by semantic, responsive exception rows; preserve the existing Matter navigation rather than adding a duplicate detail surface.

**Tech Stack:** React 19, TypeScript, React Aria UI primitives, Vitest/Testing Library, CSS design tokens, Playwright evidence harness.

---

### Task 1: Deterministic exception presentation

**Files:**
- Create: `web/src/vendorExceptionPresentation.ts`
- Create: `web/src/vendorExceptionPresentation.test.ts`

- [ ] **Step 1: Write failing ranking and filtering tests**

Cover overdue before blocked, blocked before missing assignment, due-soon before routine open, earliest deadline tie-breaking, closed exclusion, explicit missing deadline, and the `ATTENTION`, `OVERDUE`, `OPEN` and `ALL` filters using real `VendorRiskFinding` objects.

- [ ] **Step 2: Run the focused test and confirm RED**

Run: `npm exec -- vitest run src/vendorExceptionPresentation.test.ts`

Expected: FAIL because `presentVendorExceptions` and its presentation types do not exist.

- [ ] **Step 3: Implement the pure presentation helper**

Define:

```ts
export type VendorExceptionFilter = "ATTENTION" | "OVERDUE" | "OPEN" | "ALL";
export type VendorExceptionRow = {
  item: VendorRiskFinding;
  band: "OVERDUE" | "BLOCKED" | "INCOMPLETE" | "DUE_SOON" | "OPEN" | "CLOSED";
  nextAction?: MatterAction;
  openActionCount: number;
  overdueActionCount: number;
  owner?: string;
  sourceRating?: string;
};
export function presentVendorExceptions(items: VendorRiskFinding[], filter: VendorExceptionFilter, now?: number): VendorExceptionRow[];
```

Use only stored action states, action deadlines, Matter state and `known_facts.source_owner`/`known_facts.source_rating`. Do not infer severity or compliance.

- [ ] **Step 4: Run the focused test and confirm GREEN**

Run: `npm exec -- vitest run src/vendorExceptionPresentation.test.ts`

Expected: PASS.

### Task 2: Compact summary and exception queue

**Files:**
- Modify: `web/src/components/VendorPortfolio.tsx`
- Modify: `web/src/components/VendorPortfolio.test.tsx`

- [ ] **Step 1: Write failing interaction tests**

Assert one `Vendor overview summary` region, absence of legacy metric cards and repeated full recommendations, default attention ordering, summary-count filtering, owner/vendor/rating filtering, one sample-data scope indicator, exact owner/deadline/next-action content, partial totals as Unknown, and canonical Matter navigation from **Review exception**.

- [ ] **Step 2: Run the focused component test and confirm RED**

Run: `npm exec -- vitest run src/components/VendorPortfolio.test.tsx`

Expected: FAIL on the legacy four-card and repeated metadata layout.

- [ ] **Step 3: Implement the compact component**

Replace `.vendor-metrics` and nested action lists with:

```tsx
<div className="vendor-overview-summary" role="region" aria-label="Vendor overview summary">…</div>
<section className="vendor-exceptions" aria-labelledby="vendor-exceptions-title">
  <header>…attention filters and bounded owner/vendor/rating controls…</header>
  <ol className="vendor-exception-list">…compact rows…</ol>
</section>
```

Each row renders exception title, vendor/service, recorded source rating, action owner, next action, relative overdue/due state, workflow badge, open/overdue action counts when greater than one, and one **Review exception** button. Keep loading, partial, retry, empty and implemented-outcome notices.

- [ ] **Step 4: Run the focused component test and confirm GREEN**

Run: `npm exec -- vitest run src/components/VendorPortfolio.test.tsx`

Expected: PASS.

### Task 3: Dense responsive presentation

**Files:**
- Modify: `web/src/components/vendor-portfolio.css`
- Modify: `web/scripts/capture-vendor-portfolio-evidence.mjs`
- Modify: `docs/evidence/2026-09-10-vendor-findings/*`

- [ ] **Step 1: Add evidence assertions before styling**

Require the exception list to begin near the top, reject legacy metric groups, verify at least five representative rows within the desktop capture viewport, assert no horizontal overflow at 1440px and 390px, and retain Overview/Register route separation checks.

- [ ] **Step 2: Run the evidence harness and confirm RED**

Run the existing `build:evidence`, preview and `capture-vendor-portfolio-evidence.mjs` commands.

Expected: FAIL because the compact layout and assertions are not satisfied.

- [ ] **Step 3: Implement token-driven desktop and mobile CSS**

Use a one-line/four-segment summary on desktop, compact filter toolbar, 64–76px desktop rows, visible focus states and text-plus-color urgency. At 760px and below, use two-line cards with no horizontal scrolling or obscured actions. Remove legacy card/action-box selectors rather than retaining dead styling.

- [ ] **Step 4: Rebuild, capture and inspect evidence**

Run the focused evidence build and capture. Inspect light/dark 1440px and 390px Overview images, fix the highest-impact layout failure, and recapture.

Expected: route, density and overflow assertions pass; exceptions appear immediately after the compact summary.

### Task 4: Focused release verification and deployment

**Files:**
- Modify: `docs/superpowers/specs/2026-09-10-vendor-exception-overview-design.md`
- Modify: `docs/superpowers/plans/2026-09-10-vendor-exception-overview.md`

- [ ] **Step 1: Record the clarified attention-filter boundary and complete plan checkboxes**

Confirm the design and implementation agree that Needs attention is bands 1–4 and All open adds routine open exceptions.

- [ ] **Step 2: Run focused verification**

Run:

```text
npm exec -- vitest run src/vendorExceptionPresentation.test.ts src/components/VendorPortfolio.test.tsx
npm run typecheck
npm run build
git diff --check
```

Expected: all focused tests, typecheck, build and whitespace check pass. Do not run the broad demo regression suite.

- [ ] **Step 3: Commit, push, merge and deploy**

Merge the exact reviewed head. Build only the changed web image, revision-wrap unchanged API/worker images, preserve `CLEARSIGHT_DEMO_SEED_MODE=manual`, and use the existing ClearSight release script.

- [ ] **Step 4: Verify the hosted workflow**

Authenticate as Hakeem and confirm the deployed revision, compact summary counts `1 / 5 / 5 / 5`, first exception visibility without metric-card scrolling, overdue-first queue order, canonical Matter navigation, Register separation, Control One health and retained source-curated data.
