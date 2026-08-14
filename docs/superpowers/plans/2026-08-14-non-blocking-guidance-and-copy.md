# Non-blocking Guidance and Product Copy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace blocking onboarding modals and scripted tour copy with a compact, conventional guide that leaves every workspace action available.

**Architecture:** Keep the existing guide API, saved state and step action callbacks. Change `IntroGuide` from a modal overlay into an inline complementary region, mount it in the main content flow, and update canonical server guide copy plus the static demo fixture.

**Tech Stack:** React 19, TypeScript, Vitest, Testing Library, CSS, Go.

---

### Task 1: Lock the expected behavior in tests

**Files:**
- Modify: `web/src/components/RoleAwareOnboarding.test.tsx`
- Modify: `web/src/components/DemoAuthGate.test.tsx`
- Modify: `internal/onboarding/service_test.go`

- [ ] Add a component test that expects a labelled complementary region, no dialog, no `aria-modal`, and unchanged external focus after automatic guide display.
- [ ] Update restart assertions to expect the guide region and revised Help label.
- [ ] Update demo-login heading assertions to “Choose a demo account”.
- [ ] Add a server guide copy test that rejects the reported scripted phrases.
- [ ] Run focused Vitest and Go tests and confirm they fail for the intended missing behavior and copy.

### Task 2: Implement the non-blocking guide

**Files:**
- Modify: `web/src/components/IntroGuide.tsx`
- Modify: `web/src/components/RoleAwareOnboarding.tsx`
- Modify: `web/src/App.tsx`
- Modify: `web/src/product-finish.css`
- Modify: `web/src/styles.css`
- Modify: `web/src/ui-preferences.css`

- [ ] Remove the modal backdrop, dialog semantics, focus movement and Tab trapping.
- [ ] Render only the active step with a concise step count and familiar action labels.
- [ ] Mount onboarding after the context bar inside `main` so it participates in page layout.
- [ ] Add responsive inline guide styles and remove obsolete modal guide rules.
- [ ] Run the focused component tests and confirm they pass.

### Task 3: Replace scripted copy at its sources

**Files:**
- Modify: `internal/onboarding/service.go`
- Modify: `web/src/staticDemo.ts`
- Modify: `web/src/components/DemoLoginPage.tsx`
- Modify: `web/src/components/DemoAuthGate.test.tsx`
- Modify: `docs/stakeholder-demo/index.html`

- [ ] Rewrite every role guide with direct operational headings, descriptions and buttons.
- [ ] Keep the static demo guide synchronized with the server guide.
- [ ] Replace the demo login headline and supporting text with direct account-selection copy.
- [ ] Remove the obsolete blocking tour markup from the stakeholder demo document.
- [ ] Run focused frontend and onboarding tests and confirm they pass.

### Task 4: Verify and release

**Files:**
- Verify all changed files.

- [ ] Run formatting, TypeScript typecheck, frontend tests, Go tests and `git diff --check`.
- [ ] Render desktop and mobile states and verify the workspace remains operable while guidance is visible.
- [ ] Commit and push the changes to `main` under the existing authorization.
- [ ] Monitor CI and deployment, then verify the production demo copy and interaction.
