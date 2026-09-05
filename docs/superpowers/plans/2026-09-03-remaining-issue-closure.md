# Remaining Issue Closure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Broad implementation is **paused by the operator**. The subsequent request permits review and safe merge of the existing correction/documentation only; do not resume the remaining features from this document without a new request.

**Goal:** Complete the remaining V1 outcomes, reconcile every open issue with evidence, and retain broader roadmap issues until their own scope and acceptance are satisfied.

**Architecture:** Extend the existing identity, positions, authority, governance revisions, Programs, Matters, capture, scoring, timers and outbox boundaries. Keep assignment, evidence receipt, review, implementation, verified outcome and closure separate. Never add a parallel directory, workflow, notification, policy or evidence store to avoid integrating with current services.

**Tech Stack:** Go, PostgreSQL 18/pgx, React/TypeScript, Vitest/axe, rendered browser fixtures, GitHub Actions and Docker Compose.

---

## 1. Resume facts and scope

- Audited main: `bbe7397423d69894c8fa6a6f063477bf0ffd7795`.
- Isolated checkout: `C:\dev\pr158-review`, branch `codex/governed-hierarchy-v1`.
- Implementation commit: `c88c26004e1df57563a0d062f7978af69453431e` — malformed reporting-chain denial and current-owner fallback correction. **At the pause checkpoint it was not pushed, independently reviewed, merged or deployed.** Subsequent review/merge evidence belongs to the focused PR linked from #141; do not infer release from the checkpoint alone.
- Root checkout `C:\dev\clearsight-grc` is dirty on `codex/vendor-management`. Preserve it and all other worktrees.
- #176 is merged; #180 and #124 are closed. Do not redo that reconciliation.
- #144, #145 and #146 are closed with hosted evidence. Their #128 checkboxes were corrected on 3 September. Historical evidence is not a new current-head acceptance result.
- All 14 currently open issues still have unmet full-issue criteria. See [the evidence matrix](../../evidence/2026-09-03-open-issue-audit.md).
- #129 is the only open PR at this audit: draft branch `agent/128-static-truth-scoring-e2e-20260902`, head `69c7b1c9ff4902417a9cf7c1605877ae405f5179`. Extract unique scoring work; do not merge its superseded runtime changes.

This is the current execution/closure index, not permission to expand the V1 scope. The approved subsystem designs and focused plans below provide implementation detail. Before a large missing subsystem is coded, produce a focused test-first patch plan from its current interfaces; do not turn the roadmap sections into one unreviewable PR.

### Required reading

Read `AGENTS.md`, `README.md`, `DESIGN.md`, `docs/README.md`, `docs/implementation-plan.md`, and the affected product/architecture/acceptance documents. In particular:

- `docs/product/authority-routing-and-escalation.md`
- `docs/architecture/application-architecture.md`
- `docs/architecture/governance-runtime.md`
- `docs/superpowers/specs/2026-09-01-oversight-hierarchy-and-workspace-containment-design.md`
- `docs/superpowers/plans/2026-09-01-oversight-hierarchy-and-role-accurate-today.md` — remaining hierarchy tranche B, not completed Today/Oversight work.
- `docs/superpowers/specs/2026-09-01-governed-form-advanced-scoring-and-response-policies-design.md`
- `docs/superpowers/plans/2026-09-01-governed-form-advanced-scoring-and-response-policies.md`
- `docs/superpowers/plans/2026-09-02-v1-resolution-board.md` — retain its dependency semantics, but reconcile historical checkboxes with the audit.

## 2. Delivery order and closure ownership

| Order | Work | Closure rule |
| --- | --- | --- |
| 1 | Review and release local #141 safety fix | Partial #141 only; no full-issue closing keyword. |
| 2 | #137 canonical contracts | Remove all inventoried alternate contracts and prove obsolete-input rejection. |
| 3 | #141 governed hierarchy and handoff | Editable hierarchy plus full authority, concurrency, notification and rendered/hosted acceptance. |
| 4 | #142 sequence configuration | Depends on #141; three-level runtime and recovery proof. |
| 5 | #143 unique #129 extraction and scoring journey | Can proceed independently after relevant #137 contract decisions; exact persisted end-to-end proof. |
| 6 | #138 complete reference installation | Reuse completed #141–143 APIs; prove empty/install/restart/rebuild truth. |
| 7 | #139 and #140 hosted outcomes | Reuse merged implementations; fix only failures discovered in real journeys. |
| 8 | #147 release gate, then #128 | Every blocking child has evidence or an explicit operator-approved V1 exclusion. |
| Separate | #172, #74, #80, #57, #13 | Do not infer their full completion from V1 closure. Each needs its own bounded delivery plans and acceptance. |

## 3. Task A — Review the committed handoff safety correction

**Files:** `internal/access/postgres.go`, `internal/access/reassignment_postgres_integration_test.go`, `internal/httpapi/command_lifecycle.go`, `internal/httpapi/reassignment_owner_route_test.go`.

- [ ] Read the complete `bbe73974..c88c2600` diff and perform independent specification review, then code-quality review.
- [ ] Confirm the exact contract: every active owner-position chain must reach an effective same-entity root within 12 edges; cycles, truncation, broken/out-of-scope/expired parents deny uniformly without a version hint. Valid direct/higher managers still pass. A denied current-owner route cannot regain authority through same-ID fallback.
- [ ] Run the committed regressions on an isolated current-schema PostgreSQL 18 database:

```powershell
go test ./internal/httpapi -run 'TestCurrentOwnerReassignmentCannotBypassCurrentAuthorityRoute|Reassign' -count=1
go test -tags 'postgres postgresintegration' ./internal/access -run TestPostgresReassignmentRequiresCompleteActiveReportingChain -count=2
go test ./internal/access ./internal/httpapi -count=1
go test -tags postgres ./internal/access ./internal/httpapi -count=1
```

Expected: no failures; the PostgreSQL regression contains 27 cases and must not skip. `TEST_DATABASE_URL` must target the disposable database, never hosted/customer data.

- [ ] If review changes code, demonstrate a failing regression before the correction and rerun the affected gates.
- [ ] Create a focused PR with `Refs #141`, exact head and test evidence. Merge only after required checks and review; verify main CI and deployed revision afterward.
- [ ] Record the explicit remainder: `max(position.version)` is **not** a hierarchy revision or a command-time concurrency fence. This commit does not provide hierarchy administration or complete #141.

## 4. Task B — Canonical contracts (#137)

**Files:** `internal/evidence/distribution.go`, `internal/evidence/completed_response.go`, `internal/formcontract/model.go`, `internal/evidence/draft_compatibility.go`, `internal/evidence/draft_compatibility_test.go`, `internal/httpapi/form_distribution_handlers.go`, `internal/httpapi/form_distribution_list.go`, `web/src/formsDistributionApi.ts`, `web/src/formsDistributionApi.test.ts`, `web/src/captureInvitationBrowser.ts`, `web/src/components/ExternalCaptureApp.test.tsx`.

- [ ] Inventory route producers and first-party callers before removal. For each distribution, recipient, response revision and bundle, record the one supported lowercase JSON DTO; do not globally rename internal Go fields.
- [ ] Add failing handler/client tests for canonical payloads and explicit rejection of PascalCase aliases, scalar answers, query invitation discovery and retired draft/session calls. Use structured answer examples from the current `AnswerValue` contract rather than inventing another shape.
- [ ] Add explicit DTOs/tags, migrate actual callers, then remove fallback normalization and unreachable facades. Remove compatibility-only tests; retain negative obsolete-input tests.
- [ ] Add a class-wide route/serialization regression, not only phrase or filename scans.
- [ ] Run `go test ./internal/evidence ./internal/formcontract ./internal/httpapi -count=1`, tagged PostgreSQL tests, and `npm --prefix web test -- formsDistributionApi ExternalCaptureApp`.
- [ ] Verify normal internal/external capture on the resulting deployed SHA before closing #137.

## 5. Task C — Governed hierarchy configuration (#141)

**New units:** `internal/organization/model.go`, `repository.go`, `service.go`, `postgres.go`, `impact.go`, `impact_postgres.go`, focused unit/integration tests; one next-unused migration pair; schema ownership entry.

**Integration files:** `internal/access/postgres.go`, `internal/authority/postgres.go`, `internal/httpapi/command_lifecycle.go`, `internal/httpapi/identity_access_handlers.go`, canonical route registry/OpenAPI, API/worker composition, `web/src/components/IdentityAccessPanel.tsx`, `web/src/components/access/OrganizationInventory.tsx`, new focused organization API/workspace components and their state fixtures.

### C1 — Freeze the transaction and revision contract

- [ ] Preserve stable `org_positions.id` values: assignments, grants and role bindings already reference them. Store immutable prior/proposed snapshots and a monotonic entity hierarchy revision; do not fabricate retrospective approval for seeded positions.
- [ ] Define bounded proposed changes and snapshot fields for position, occupant, parent, department and role bindings. Position-role eligibility remains distinct from responsibility and material authority.
- [ ] Define Draft → preview → submit → independent approval → scheduled/active → superseded history. Rollback creates a newly approved monotonic revision restoring prior content, never rewinds history.
- [ ] Select the existing runtime timer path with an explicitly owned configuration workflow, or justify a narrowly owned activation job. Scheduling, row changes, governance decision and outbox fact must be one transaction; never call a pool scheduler after commit.
- [ ] Document first entity placement as a governed assignment of a source-backed tenant principal. Do not demand an existing position that would make first placement impossible.
- [ ] Use existing scoped approved delegations for supported absence handling. A deputy label must not silently create substitution or blanket authority; unsupported emergency/substitution execution fails closed and remains a documented residual until its bounded policy is approved.

### C2 — Test-first persistence and authority

- [ ] Add failure-first fixtures for stale base/command/preview versions, self-approval, duplicate active codes, inactive/revoked occupants, tenant/entity mismatch, cycles, depth overflow and current-authority outage.
- [ ] Add transaction-bound adapters to the existing access/authority resolvers rather than copy their SQL. Reuse the tenant/entity governance lock, but do not claim it protects SCIM/role writers that do not take that lock.
- [ ] Prove concurrent occupant revocation, role revocation, policy replacement and hierarchy activation with explicit dependency locking/version checks. The activation winner must use current authority and the loser must fail without partial writes.
- [ ] Inject faults before position update, governance decision, outbox insert and activation scheduling; assert no partial commit. Build the durable command result before commit; do not return failure because a post-commit read failed.
- [ ] Prove scheduled activation before/at due time, lease recovery, duplicate delivery, failed activation preserving the current hierarchy, and rollback preserving stable position references.

### C3 — Bounded preview and usable administration

- [ ] Query exact affected descendants, active assignments, role/position selectors, escalation paths and delegations with depth/result caps. Return checked population, safe bounded samples, unknown/truncated counts and source revision. A truncated scan cannot report “no gaps.”
- [ ] Use the same runtime resolver for representative route simulation; preview does not change authority or active records.
- [ ] Add bounded People/Position/Role lookup with source/activity state. The existing tenant-wide overview is not a sufficient candidate-search API.
- [ ] Add position/reporting proposal sheets, impact preview, distinct-checker approval, effective date, activation failure recovery and change history/rollback to Configure. Use shared controls and preserve entered input after conflicts.
- [ ] Consolidate each Program/Matter/Action handoff to one contextual control and cover current owner, direct/higher manager, approved delegate, absence, conflict, revocation, stale record and cross-entity denial.
- [ ] Verify Program/Matter/Action assignment events and protected notifications share assignment truth; delivery failure does not roll back handoff and ambiguous SMTP acceptance does not auto-resend.
- [ ] Run default/tagged/integration tests for `internal/organization`, `internal/access`, `internal/authority`, `internal/governance`, `internal/workflow`, `internal/httpapi`; render affected workflows in both themes at 320/390/768/1440px with keyboard/200% reflow.
- [ ] Close #141 only after an administrator configures a real multi-level hierarchy through the deployed UI and all adversarial/notification cases have evidence.

## 6. Task D — Escalation sequences (#142)

**Files:** `internal/governance/escalation.go`, `escalation_revision.go`, `policy_revision_postgres.go`, escalation tests; `internal/workflow/matter_escalation_postgres.go` and integration tests; `internal/httpapi/identity_access_handlers.go`; `web/src/identityAccessApi.ts`, `web/src/components/IdentityAccessPanel.tsx`, focused escalation workspace/components/tests.

- [ ] Extend existing routing-policy revisions from editing guards to creating/revising the complete OVERDUE sequence. Keep the current approved revision live while a proposal is pending.
- [ ] Expose bounded monotonic thresholds, ordered levels, source responsibility, target role/group/position restrictions, terminal unresolved handling and recovery. Add no trigger without canonical events/timestamps.
- [ ] Add failing tests for at least three levels, one next timer, replay/restart, resolution cancellation, stale directory, vacant/conflicted/no recipient and changed policy. Escalation must never grant approval/review/challenge/signing authority.
- [ ] Replace department-only preview with safe bounded affected-work scenarios resolved by the same runtime resolver. Show expected candidate set, ambiguity/no-route, current policy/version and limitations.
- [ ] Reuse independent approval, scheduling and versioned rollback; do not treat guard activation alone as full sequence governance.
- [ ] Display current level, next due time, active policy revision and exact recovery action on affected work.
- [ ] Run `go test ./internal/governance ./internal/workflow ./internal/httpapi -count=1` and the serialized PostgreSQL counterparts; run affected IdentityAccess/Matter tests and rendered fixtures.
- [ ] Close only after a deployed three-level scenario stops on canonical resolution and negative/recovery cases remain visible and safe.

## 7. Task E — Scoring and response-policy outcome (#143)

**Files:** `web/src/components/forms/FormPolicyEditor.tsx`, `FormPolicyEditor.test.tsx`, `FormPoliciesView.tsx`, `FormPoliciesView.test.tsx`; `internal/formpolicy/*`; `internal/evidence/completed_response*`; `internal/formcontract/advanced_scoring.go`; `cmd/seed-bank-reference/main.go` plus extracted `scoring_acceptance.go`, `form_policy_acceptance.go`, `form_policy_acceptance_store.go`, `form_policy_authority_seed.go`; API/worker form-policy authority adapters.

- [ ] Compare #129 with fresh main. Exclude its runtime-context/Today/static-truth changes. Audit its seed commands and shared authority adapters before retaining them; no direct database state patches may impersonate business acceptance.
- [ ] First write failing editor tests: only approved active scored form revisions can be selected; no raw form ID/revision fields; empty/unavailable/stale selection cannot submit. Load form choices independently so a form-list failure does not hide existing policies or recovery actions.
- [ ] Implement the selector with the existing bounded form-reference API and shared `SelectField`; retain exact ID and revision, revalidate server-side, and keep role/scope limitations explicit.
- [ ] Persist good, borderline, poor, no-score and failed-score examples through normal distribution/capture/submission. Include weighted contributions, nested conditional effects and exact form/profile/evaluator revisions.
- [ ] Exercise score/concern/priority/completion/subject/policy-outcome queries with repository-side scope before limits, stable keyset cursors and retained index plans. Preserve at least the approved 10,000-response/1,000-policy performance scenario.
- [ ] Simulate a bounded population, independently approve and activate the policy. Assert execution receipt + one adverse episode/Matter + event/outbox/maintenance facts commit atomically under current authority.
- [ ] Assert replay and another poor response reuse the episode; good/unavailable scores do not create a false Matter. Complete authorized remediation, independent outcome verification and explicit closure; only a later poor response may create a new episode.
- [ ] Run `go test ./internal/formcontract ./internal/evidence ./internal/formpolicy ./cmd/seed-bank-reference -count=1`, serialized PostgreSQL integration, `npm --prefix web test -- FormPolicyEditor FormPoliciesView`, copy/type/build/UI gates and normal hosted acceptance.
- [ ] Link a replacement PR before closing #129 as superseded. Close #143 only with exact-head persisted outcome evidence, not the historical 1 September branch report.

## 8. Task F — Complete reference installation (#138)

**Files:** `cmd/seed-bank-reference/main.go`, `internal/bankverticals/install_service.go`, reference fixtures/installation tests, `cmd/api/services_memory.go`, `internal/runtimecontext` tests, `web/scripts/runtime-fixture-boundary.nodecheck.mjs`.

- [ ] Credit the already-clean customer import graph and removed `DemoTasks()`. Do not remove evidence-only deterministic rendering support.
- [ ] Compose one idempotent installation workflow over the existing identity/authority/domain installers. Explicitly cover tenant/entity, people/positions, authority, Programs, Matters, forms, vendors, distributions, submitted responses, policies and history.
- [ ] Use stable reference keys and labelled provenance; rerun reconciles partial installation without overwriting unrelated records or inventing approval.
- [ ] From an empty database, verify truthful empty states; install; restart API/worker/browser; switch actors; rebuild projections and compare safe record IDs/versions/counts.
- [ ] Run `node --test web/scripts/runtime-fixture-boundary.nodecheck.mjs`, reference installation and architecture tests, then the deployed normal-API journeys.
- [ ] Close after every advertised demo workspace derives from persisted canonical records and #129 overlap is resolved.

## 9. Task G — Finish hosted business acceptance (#139 and #140)

**Existing implementation:** `internal/thirdparty/activation_policy*`, address-verification and vendor-work services, `internal/continuity/matter_form_remediation*`, their HTTP handlers and React workspaces. Follow the approved vendor activation and linked-form remediation designs; do not rebuild merged features.

- [ ] Obtain current authorized non-production participant identities and approved inbox access. If the recipient must act manually, ask for that action; never fabricate a response, inbox receipt or independent approval.
- [ ] #139: registration invitation → real submission → required address-verification Matter/Action → governed staff handoff/notification → evidence → independent review/outcome → explicit closure → fail-closed relationship activation → certification refresh submission/review/closure.
- [ ] #140: inspect the existing acceptance target before sending anything new: Matter `019ff790-a425-70f6-82bf-4cc600bfccfc`, request `01a06461-d3d8-7179-8eaa-0e1ea83be065`, binding `01a06461-72e3-7758-b75d-5f182a69f0c0`. Continue from current state; reissue only through governed expiry/recovery if needed.
- [ ] Submit the real response, independently apply its exact final revision to mapped items, run/record deterministic verification and close only through current authority when every gate passes.
- [ ] Exercise expiry, revocation, wrong audience, stale version, replay, reassignment and delivery-failure recovery. Submission alone must not close a Matter or declare a Program current.
- [ ] Record only safe IDs, revisions, states, timestamps and delivery outcomes. Never commit tokens, OTPs, recipients, message bodies or credentials.
- [ ] Fix any discovered defect test-first in a focused PR, rerun the affected journey on the resulting deployed SHA, then close the corresponding issue.

## 10. Separate capability plans — do not silently fold into V1

Each item below is a separate subproject. Its first deliverable is a focused approved design/implementation plan with concrete fixtures and code-level steps, not a speculative generic framework.

| Issue | Existing code to extend | Remaining deliveries and closure proof |
| --- | --- | --- |
| #172 | `internal/aigovernance/gateway_transport_*`, baseline services; `internal/aigateway/transport_runtime.go`; `web/src/components/configure/AIGatewayTransportControl.tsx`, `AIGatewayControlPlane.tsx`; `docs/acceptance/ai-gateway-control-plane.md` | Stable endpoint/capability display and copy; safe connectivity/config/policy simulation; effective overlay preview and bounded baseline exceptions; explicit tool allowlist/side-effect approval integration; actual emergency outbound freeze with measured propagation; provider/secret/apply failures and full CP5 enterprise proof. Prior-known-good fallback must not defeat emergency suspension. |
| #74 | `web/src/components/AIGovernancePanel.tsx`, `internal/aigovernance`, existing workload/policy routes | Workload registration/revisions and protected rotation/cutover; policy composer/simulation/independent promotion; bounded decision filters/detail, revision comparison and exact investigation/approval handoff. Test empty/forbidden/degraded states and real actor journeys. #172 gateway editors do not substitute for this workspace. |
| #80 | `internal/thirdparty`, vendor UI, existing Program/Matter/evidence/source-access modules | Contract/obligation metadata; governed evidence reuse and repeated-import reconciliation; continuation/restriction/suspension and independently verified exit; assurance-driven reassessment; full vertical security/performance/bank-user acceptance. Credit activation already implemented, but do not mark the full lifecycle complete. |
| #57 | `internal/assurance`, `internal/sourceaccess`, `docs/architecture/continuous-assurance-execution.md` | Durable population/rule/run/episode ownership, approved scheduling and bounded affected-state projection; Signal/Program/Matter integration; Sources & checks UX; CDC/transition/KRI/window semantics only with real source contracts; multi-domain, unknown/degraded, lease/recovery and bank-scale proof. Retain T0 kernel and existing source executor. |
| #13 | `docs/implementation-plan.md`, product/use-case catalogue, release/quality/security documents | Explicit operator-approved pilot/GA scope; completed or explicitly excluded requirements for storage/scanning/retention, identity/security, external channels, DR/load/restore and representative-user validation. This umbrella cannot close from a generic test run or V1 child closure. |

## 11. Task H — Exact-head release (#147), then parent (#128)

- [ ] Refresh issue/PR status; confirm all required V1 children are merged with their own acceptance evidence. Do not treat an unchecked historic checklist as missing code or a closed issue as new current-head proof.
- [ ] Use a fresh PostgreSQL 18 database and serialize **all** integration commands globally, not just packages inside one process. Existing fixed-ID fixture cleanup failed on repeat runs during this session; isolate or repair the test harness separately and retain failures in the evidence log.
- [ ] Run these exact-head commands with configured integration prerequisites:

```powershell
git diff --check
go vet ./...
go test -race ./... -count=1
go test -tags postgres ./... -count=1
go test -count=1 -p 1 -tags 'postgres postgresintegration' ./internal/...
npm --prefix web run typecheck
npm --prefix web test
npm --prefix web run check:runtime-truth
npm --prefix web run check:ui-contracts
npm --prefix web run build
npm --prefix web run review:ui
```

- [ ] Use Node >=24. On this Windows host the bundled binary directory is `C:\Users\Son\.cache\codex-runtimes\codex-primary-runtime\dependencies\node\bin`.
- [ ] Verify clean current-schema installation and changed-migration safety according to current CI. Do not resurrect unreleased compatibility/parity paths. Check the next migration number against fresh main before creating one.
- [ ] Inspect rendered desktop/tablet/mobile, both themes/densities, keyboard, reduced motion and 200% reflow for every affected workflow; fix and re-render the highest-impact defect.
- [ ] Synchronize README, DESIGN, docs map, implementation ledger, product/architecture, schema ownership, OpenAPI and acceptance matrix. Update stale statements that activation/reference foundations are absent.
- [ ] Request independent review, inspect all required checks on the final PR SHA, mark ready and merge with `gh pr merge --squash --match-head-commit <verified-head>` using the actual verified SHA. Any new commit restarts affected gates.
- [ ] Verify main CI/UI, deploy API/worker/web at the same merge SHA, and confirm `/health/ready` reports it. Then execute final hosted journeys and attach safe evidence/rollback instructions.
- [ ] Close #147 and #128 only when every V1 requirement is proven or explicitly excluded by the operator. Leave broader capability issues open until their separate criteria pass.

## 12. Pause-state verification and handoff

At the stop request, the local safety commit had these results:

- 27 PostgreSQL reporting-chain cases passed twice; original code exposed 12 failing cases before the fix.
- The owner-route regression first returned HTTP 200 and committed despite denial; corrected code returns HTTP 403 without mutation while valid owner/manager paths pass.
- Full default and `postgres`-tagged access/httpapi suites passed.
- Full access integration passed initially; repeating the existing admin fixture hit `tenants_pkey` after broken cleanup. A broader evidence run hit an existing fixed `org_positions_pkey` collision. These are **not** claimed as full green integration runs.
- Governance/workflow and form-policy integration runs passed separately. No repository-wide release gate or independent review was completed for `c88c2600`.

Two disposable local databases were created in WSL Docker container `clearsight-hierarchy-test-pg18`: `clearsight_hierarchy` (contains test leftovers) and `clearsight_v1_verify` (fresh migrations, not yet used for the full gate). Stop the container on pause; it may be restarted explicitly for verification. Do not reuse the contaminated database as clean-install evidence and do not touch the unrelated `hostshell-postgres` container.

No new feature implementation should continue while this pause is in effect. The operator subsequently authorized safe merge of the existing commits; check #141 and its linked PR for Task A's latest release evidence. The remaining tasks are not marked complete by this planning commit or by release of that prerequisite.
