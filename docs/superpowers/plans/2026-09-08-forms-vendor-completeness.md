# Forms and Vendor Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking.

**Goal:** Complete the approved Policies, Responses and vendor experience without replacing existing workflows.

**Architecture:** Correct scoped read projections, then compose existing shared Tabs, sheets, tables and document browser. Preserve immutable responses, server authority, bounded queries and separate acceptance/activation states.

**Tech Stack:** Go, PostgreSQL/pgx, React/TypeScript, Vitest and Playwright.

## Task 1 — Submitted vendor truth (#139)

Files: `internal/evidence/vendor_forms_postgres.go`, `internal/evidence/vendor_forms_postgres_integration_test.go`, existing memory/retirement tests as needed.

- [ ] Add a PostgreSQL regression creating an ordinary distribution and submitted revision while the reusable request remains open. Assert the list reports SUBMITTED, exposes the exact response ID, and summary/filter populations no longer count it awaiting vendor or overdue. Retain revoked/superseded exclusion and partial-replacement behavior.
- [ ] Run `go test -tags "postgres postgresintegration" ./internal/evidence -run TestPostgresVendorForms -count=1` with the isolated test database; record the expected red assertion.
- [ ] In the scoped SQL state CASE, after revoked/superseded handling, prefer the exact authorized current response: `WHEN r.id IS NOT NULL THEN 'SUBMITTED'`. Do not mutate request status or assessment lifecycle.
- [ ] Repeat integration and `go test -tags postgres ./internal/evidence`; inspect row, summary, filter and memory parity; commit only task files.

## Task 2 — Response and policy completion (#143)

Files: `web/src/components/forms/ResponsesView.tsx`, `ResponseAssessment.tsx`, `FormPoliciesView.tsx`, their tests/CSS; `web/src/formPoliciesApi.ts`, relevant `internal/httpapi/form_policy_*` and policy store reads only where a scoped link/receipt is missing.

- [ ] Add failing tests: NOT_REQUIRED hides review-permission warning and empty assessment totals; NOT_CONFIGURED coverage is unavailable, not 0%; a subject selector sends `VENDOR_RELATIONSHIP`; detail exposes Answers/Documents/Review/History without duplicate file actions; policy 403 is a non-retryable denied state; active approved revisions never prompt draft simulation; stored form name replaces UUID where available.
- [ ] Run focused Vitest files directly with `node node_modules/vitest/vitest.mjs run --maxWorkers=4 <files>` and record red assertions.
- [ ] Use exact enum options and existing Tabs; keep answer/evidence access available without required review. Resolve subject names only through currently authorized exact reads or a bounded server projection; unknown names remain explicitly unknown with identifiers in detail. No broad client-side population load for labels.
- [ ] Present policy lifecycle from persisted policy facts. Expose the recorded approval/simulation reference without calling it current impact; draft simulations retain their existing expiry/version requirements. Add issue/response links only through currently authorized bounded read contracts. Keep safe access-denied/retry distinctions.
- [ ] Re-run affected tests, `copyQuality.test.ts`, TypeScript and fixtures. Commit this coherent UI/read-contract slice.

## Task 3 — Vendor sections and sample presentation (#139/#138)

Files: `web/src/components/VendorsWorkspace.tsx`, `VendorFormsPanel.tsx`, associated tests/styles, existing sample installer/fixtures only as necessary.

- [ ] Add failing tests for Overview/Forms/Documents/Due diligence/History navigation; narrow selector; no repeated request controls in one context; selecting a different vendor resets scope; no permission or command change from navigation.
- [ ] Reuse shared `Tabs` with `compactLabel="Vendor section"`; Overview contains operational summary and collapsed identity metadata, Forms owns request/response work, Documents embeds the existing browser, Due diligence owns review and activation gates, History exposes existing immutable response history. Preserve existing inbound actions by selecting the relevant section.
- [ ] Replace engineering narration in newly displayed sample fixture labels with explicitly fictional business wording; do not rewrite immutable response histories or silently rename user-created records. Missing real vendor facts stay missing, not invented.
- [ ] Run vendor tests, copy-quality and the full-host rendered fixture matrix. Inspect highest-impact failure, correct and re-render. Commit scoped files.

## Task 4 — Integration, release and receipts (#147)

- [ ] Specification review then quality review for each task; fix material findings and repeat review. Keep #200 closed; add acceptance receipts to existing issues, not duplicate master issues.
- [ ] Run `go test ./...`, tagged tests/integration, vet, full Vitest, TypeScript, production/evidence builds, runtime isolation and UI-contract checks.
- [ ] Run the complete UI review and targeted light/dark 1440/390/320/720 fixtures. Preserve the unrelated presentation asset byte-for-byte when the runner generates its cover.
- [ ] Update root DESIGN, product specification, execution ledger and acceptance record with actual maturity and evidence.
- [ ] Push a focused PR, wait for exact-head required CI, merge only after passing gates, follow the normal deployment workflow and verify `/health/ready` identifies the merged SHA.
- [ ] Re-run read-only hosted checks as Program Owner and System Administrator; confirm submission/summary consistency, neutral no-review state, policy access/lifecycle and all vendor sections. Report remaining external acceptance separately; never call SMTP or a simulated antivirus treatment proof of delivery or a real scan.
