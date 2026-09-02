# Email, Runtime Truth, and Oversight Release Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Execute the three approved implementation plans in dependency order, merge one exact green commit line, deploy it safely, and prove both hosted inbox journeys plus stored oversight completeness.

**Architecture:** Runtime truth isolation lands first so subsequent acceptance cannot be satisfied by fixture transport. Oversight history lands second because it changes only reference installation/projection paths and can be verified independently. Email/link journey hardening lands third and is followed by a single exact-head regression, rendered review, merge, deployment, reference installation, and controlled hosted traversal. PR #129 remains unmerged; only independently reimplemented, tested ideas may be used.

**Tech Stack:** Git/GitHub Actions, Go 1.26, PostgreSQL 18, Node 24, React/Vite/Vitest, Docker deployment, SMTP STARTTLS.

---

## Source plans

1. `docs/superpowers/plans/2026-09-02-seeded-runtime-truth.md`
2. `docs/superpowers/plans/2026-09-02-oversight-history-completeness.md`
3. `docs/superpowers/plans/2026-09-02-real-vendor-email-link-journeys.md`

### Task 1: Establish a clean implementation baseline

- [ ] **Step 1: Confirm branch and preserve unrelated work**

Run:

```bash
git status --short
git branch --show-current
git rev-parse HEAD
git rev-parse origin/main
```

Expected branch: `codex/oversight-hierarchy`. Preserve the operator-owned modification to `docs/presentation-assets/clearsight-premium-first-run-cover.png`; never stage, alter, or revert it.

- [ ] **Step 2: Confirm PR #129 remains excluded**

Verify no merge commit or cherry-pick from PR #129 is present. Reuse only behavior independently implemented under a failing test. In particular, do not introduce `cmd/api/demo_runtime_context.go`, hardcoded demo label maps, unscored form-policy selectors, or its incomplete acceptance shortcut.

- [ ] **Step 3: Run a pre-change baseline**

Run focused current-contract tests for `internal/httpapi`, `internal/today`, `internal/oversight`, `internal/thirdparty`, `internal/evidence`, and `internal/workflow`; record any pre-existing failure before implementation.

### Task 2: Execute stored runtime truth plan

- [ ] **Step 1: Complete every checkbox in the runtime truth plan**

Use `docs/superpowers/plans/2026-09-02-seeded-runtime-truth.md` task-by-task, including the resolver, fixture removal, evidence-only entry, architecture gates, docs, and rendered inspection.

- [ ] **Step 2: Review checkpoint**

Before proceeding, verify:

- `/api/v1/context` resolves exact stored tenant/legal-entity/principal labels;
- missing directory rows cannot produce a friendly invented fallback;
- Today reads only actor-scoped tasks/operations;
- normal `web/src/main.tsx` cannot reach static fixture transport;
- normal `dist` excludes fixture markers while `dist-evidence` contains the isolated test runtime;
- backend and web architectural gates pass.

### Task 3: Execute oversight history completeness plan

- [ ] **Step 1: Complete every checkbox in the oversight plan**

Use `docs/superpowers/plans/2026-09-02-oversight-history-completeness.md` task-by-task.

- [ ] **Step 2: Review checkpoint**

Before proceeding, verify:

- at least five comparable completed reference Matters and multiple types exist;
- every Matter was created and completed through continuity commands;
- event/outbox/history is reconstructable and the second install is unchanged;
- estimate minimums, sparse suppression, coverage, freshness, and high-water marks are tested;
- reassignment, return, reopen, SLA, median, and p75 are derived from stored history;
- no employee rank or precomputed metric fixture was introduced.

### Task 4: Execute real email/link journey plan

- [ ] **Step 1: Complete automated and UI tasks first**

Use `docs/superpowers/plans/2026-09-02-real-vendor-email-link-journeys.md` through its exact-head automated verification task. Do not perform the hosted send before the branch is merged/deployed.

- [ ] **Step 2: Review checkpoint**

Verify the combined PostgreSQL acceptance reconstructs both journeys and negative link/delivery states. Confirm invitation, SMTP, assignment, Vendor Work, artifact, review, outcome, sign-off, and closure records are distinct.

### Task 5: Run the combined exact-head gate

- [ ] **Step 1: Backend verification**

```bash
files="$(gofmt -l $(find cmd internal -name '*.go' -type f))"
test -z "$files"
go test -race -coverprofile=coverage.out ./...
go test -tags postgres ./...
go test -count=1 -p 1 -tags "postgres postgresintegration" ./internal/...
go vet ./...
```

- [ ] **Step 2: Web verification on Node 24**

```bash
cd web
npm ci --no-audit --no-fund
npm run check:runtime-truth
npm run typecheck
npm test
npm run check:ui-contracts
npm run build
npm run check:runtime-truth
npm run build:evidence
npm run review:ui
```

- [ ] **Step 3: Deployment and documentation verification**

```bash
python -m unittest discover -s deploy/tests -p 'test*.py'
bash -n deploy/scripts/verify-email-readiness.sh
git diff --check
git status --short
```

- [ ] **Step 4: Secret and fixture scan**

Scan changed source, build output, UI evidence, logs, and docs for SMTP credentials, recipient addresses, OTPs, invitation selectors, full secure URLs, and code-bound business fixtures. Delete generated unsafe evidence rather than redacting it in place if a secure URL was captured.

- [ ] **Step 5: Request final code review**

Use the requesting-code-review skill. Resolve every correctness/security finding, re-run affected focused tests, then re-run the entire combined exact-head gate.

### Task 6: Merge and deploy the exact green revision

- [ ] **Step 1: Push and open one PR**

Push `codex/oversight-hierarchy`, open a PR that links issue #128 and the approved design, and explicitly states that PR #129 is superseded/not merged.

- [ ] **Step 2: Wait for required checks**

Require backend, web, and UI/UX review checks on the exact head. Do not merge a newer unverified head or bypass a failing check.

- [ ] **Step 3: Merge and record SHA**

Merge using the repository's normal strategy and record the resulting `main` SHA in the deployment receipt.

- [ ] **Step 4: Deploy without changing protected secrets**

Preserve SMTP secret references, recipient keyring, active key, access HMAC, capture URL, and sender configuration. Deploy the recorded SHA and confirm API/worker readiness reports the same revision.

- [ ] **Step 5: Run redacted readiness**

Run `deploy/scripts/verify-email-readiness.sh` and require status-only success. Stop before sending if the sender, environment, capture origin, tenant, or legal entity differs from the approved non-production scope.

### Task 7: Install history and complete hosted acceptance

- [ ] **Step 1: Install and verify reference history**

Run the reference installer with existing protected scope/principal IDs. Verify the second run is unchanged, the oversight snapshot is current, and the UI shows explainable history without implying bank performance.

- [ ] **Step 2: Traverse registration and address verification**

Use the approved vendor and staff inboxes already supplied by the operator. Observe each message in the inbox, traverse the exact received link, and complete registration, pending address verification, staff evidence, bank acceptance, passed outcome, sign-off, and Matter closure.

- [ ] **Step 3: Traverse certification refresh**

Send the active ISO 27001/PCI DSS refresh form to the registered reference vendor, traverse the exact received link, submit labelled test evidence, perform per-item review, and confirm the final stored Matter/work state.

- [ ] **Step 4: Verify negative link controls without exposing secrets**

Using deliberately reissued/expired test invitations, verify wrong audience/OTP, expiry, revocation, replay, and old-session invalidation. Do not capture or persist the protected values.

- [ ] **Step 5: Record completion and exact remainder**

Update `docs/quality/acceptance-tests.md` with the merge/deploy SHA, redacted timestamps/IDs, final stored states, oversight counts/sample sizes/freshness, and remaining exclusions from the approved design. Do not mark issue #128 complete until both human-observed inbox receipts and final UI/audit states are proven.
