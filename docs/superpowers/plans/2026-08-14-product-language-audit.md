# Product Language Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace product commentary and implementation jargon across customer-facing interfaces with concise banking workflow language, and prevent regressions.

**Architecture:** Keep copy at its existing ownership boundary so no workflow behavior or API shape changes. Add a source-level copy-quality regression test for known commentary patterns, then update React screens, server-provided guides and demo/runtime messages. Verify semantics with focused component and Go tests, followed by rendered production review.

**Tech Stack:** React 19, TypeScript, Vitest, Go 1.25, GitHub Actions, Docker deployment

---

### Task 1: Copy-quality regression

**Files:**
- Create: `web/src/copyQuality.test.ts`

- [ ] **Step 1: Write a failing source audit test**

Create a Vitest test that scans non-test TypeScript interface sources and reports the file, line and prohibited product-commentary phrase. Cover the concrete anti-patterns found during the audit, including `generic dashboard`, `exact record`, `authoritative server`, `projection cycle`, `bounded daily digest`, `current canonical`, `second directory console`, `governed candidate set`, `without needing to know`, `ClearSight resolved`, `product behaviour`, `atomically activating`, `Program truth`, and `is inferred`.

- [ ] **Step 2: Confirm the test fails on current copy**

Run: `npm test -- --run src/copyQuality.test.ts`

Expected: FAIL with matches in `AppViews.tsx`, `BankJourneysWorkspace.tsx`, `DocumentImportWorkspace.tsx`, configuration components and command panels.

### Task 2: Primary workspace language

**Files:**
- Modify: `web/src/App.tsx`
- Modify: `web/src/AppViews.tsx`
- Modify: `web/src/components/ProgramsWorkspace.tsx`
- Modify: `web/src/components/MattersWorkspace.tsx`
- Modify: `web/src/components/EvidenceWorkspace.tsx`
- Modify: `web/src/components/TodayInterventions.tsx`
- Modify: `web/src/components/ReadinessPanel.tsx`
- Modify: `web/src/components/BankJourneysWorkspace.tsx`
- Test: `web/src/components/TodayInterventions.test.tsx`
- Test: `web/src/components/ExactTargetWorkspaces.test.tsx`

- [ ] **Step 1: Update rendered assertions before production copy**

Change focused expectations to the direct task, status and action wording defined in the decision brief.

- [ ] **Step 2: Run focused tests and confirm failure**

Run: `npm test -- --run src/components/TodayInterventions.test.tsx src/components/ExactTargetWorkspaces.test.tsx`

Expected: FAIL on changed text.

- [ ] **Step 3: Rewrite primary screens**

Remove product explanations from page introductions, authority details, empty states and workspace summaries. Preserve scope and unavailable-state accuracy. Replace connected-bank and inferred-state commentary with the concrete loaded population or recovery action.

- [ ] **Step 4: Run focused tests**

Run: `npm test -- --run src/components/TodayInterventions.test.tsx src/components/ExactTargetWorkspaces.test.tsx src/copyQuality.test.ts`

Expected: primary workspace tests pass; copy-quality test may still report configuration and import files.

### Task 3: Import and coverage language

**Files:**
- Modify: `web/src/components/DocumentImportWorkspace.tsx`
- Modify: `internal/documentimport/service.go`
- Test: `web/src/components/DocumentImportWorkspace.test.tsx`
- Test: `internal/documentimport/service_test.go`

- [ ] **Step 1: Add assertions for plain import and coverage copy**

Assert that stale comparisons, extraction completeness, proposal review and unmatched obligations use direct descriptions and actions without internal architecture terms.

- [ ] **Step 2: Confirm focused tests fail**

Run: `npm test -- --run src/components/DocumentImportWorkspace.test.tsx`

Expected: FAIL on the previous stale/completeness text.

- [ ] **Step 3: Rewrite import and server limitation messages**

Describe retained and omitted sections numerically, tell reviewers when to rerun a comparison, distinguish proposal acceptance from approval, and keep the document-specific compliance disclaimer.

- [ ] **Step 4: Verify import tests**

Run: `npm test -- --run src/components/DocumentImportWorkspace.test.tsx`

Run: `go test ./internal/documentimport ./internal/documentcoverage`

Expected: PASS.

### Task 4: Configuration and command language

**Files:**
- Modify: `web/src/components/AutomationPolicies.tsx`
- Modify: `web/src/components/IdentityAccessPanel.tsx`
- Modify: `web/src/components/ProjectionHealthCard.tsx`
- Modify: `web/src/components/MatterWorkCommandPanel.tsx`
- Modify: `web/src/components/ProgramLifecycleControls.tsx`
- Modify: `web/src/components/ProgramReviewDigest.tsx`
- Modify: `web/src/components/OperatingMutationsEvidencePage.tsx`
- Modify: `web/src/components/PremiumIllustration.tsx`
- Test: `web/src/components/AutomationPolicies.test.tsx`
- Test: `web/src/components/ProgramReviewDigest.test.tsx`
- Test: `web/src/components/OperatingMutations.test.tsx`

- [ ] **Step 1: Change focused assertions to customer language**

Cover automation limits, review history, saved actions, lifecycle changes and administrator approval consequences.

- [ ] **Step 2: Confirm focused tests fail**

Run: `npm test -- --run src/components/AutomationPolicies.test.tsx src/components/ProgramReviewDigest.test.tsx src/components/OperatingMutations.test.tsx`

Expected: FAIL on previous terminology.

- [ ] **Step 3: Rewrite configuration and command surfaces**

Use direct labels for policies, directories, escalations, assigned actions and Program status. Preserve independent approval and authority constraints without narrating server or data architecture.

- [ ] **Step 4: Verify focused tests and audit**

Run: `npm test -- --run src/components/AutomationPolicies.test.tsx src/components/ProgramReviewDigest.test.tsx src/components/OperatingMutations.test.tsx src/copyQuality.test.ts`

Expected: PASS.

### Task 5: Server-provided and sample language

**Files:**
- Modify: `internal/onboarding/service.go`
- Modify: `web/src/staticDemo.ts`
- Test: `internal/onboarding/service_test.go`

- [ ] **Step 1: Extend onboarding copy assertions**

Check every guide title, description, step and action against the same commentary patterns and assert key role guides use direct working language.

- [ ] **Step 2: Confirm the Go test fails where applicable**

Run: `go test ./internal/onboarding`

Expected: FAIL if server guide copy violates the standard; otherwise retain the passing guard and continue.

- [ ] **Step 3: Rewrite remaining sample and guide text**

Keep sample-data and legal-review notices factual. Remove product demonstrations or implementation explanations from displayed fixture limitations and messages.

- [ ] **Step 4: Verify server tests**

Run: `go test ./internal/onboarding ./internal/bankverticals ./internal/workflow`

Expected: PASS.

### Task 6: Full verification, rendering and deployment

**Files:**
- Update rendered evidence under: `docs/screenshots/`

- [ ] **Step 1: Run frontend checks**

Run: `npm test -- --run`

Run: `npm run typecheck`

Run: `npm run build`

Expected: PASS.

- [ ] **Step 2: Run backend and repository checks**

Run: `go test ./...`

Run: `git diff --check`

Expected: PASS.

- [ ] **Step 3: Render major interfaces**

Review Today, Programs, Work, Imports, Explore and Configure at desktop and narrow widths. Confirm the guide is non-blocking, account switching is accessible and no screen teaches users about product implementation.

- [ ] **Step 4: Commit and deploy**

Commit the reviewed change, push `main`, monitor CI and deployment, then repeat the copy audit on the production site.
