# Issue Workspace Follow-through Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the issue-to-form and issue-to-board-brief journeys so a bank user can find every linked form, inspect large request populations, act on a sent request, and understand an unavailable board brief without leaving the work ambiguous.

**Architecture:** Keep the issue workspace as a compact composition of existing Form Library, Sent Forms, response, vendor and reporting records. Add no issue-specific form or report model. Reuse the pending exact Form Library item route for form-draft handoff; add bounded origin filtering only where the issue needs a durable list of its own authored forms; use existing cursor APIs and exact distribution reads for request activity.

**Tech Stack:** React 19, TypeScript, Vite/Vitest, Go HTTP API, monitoring/evidence/reporting services, PostgreSQL migrations only if the existing origin index cannot support the bounded query.

---

## Review result and integration order

The earlier `2026-10-06-issue-workspace-remaining.md` delivery is complete. This plan addresses the follow-through gaps found in the merged implementation.

| Pull request | Review result | Required disposition |
| --- | --- | --- |
| #334, Forms exact-detail deep link | Clean merge simulation. It supplies the exact `#forms/{templateID}` handoff needed after an issue form is drafted. | Rebase on current `main`, re-run its focused checks, merge before Task 2. |
| #342, source IT exception register to ERM Risks | Correctly bounded to explicit source records, but its CI failed because two Go tests are not gofmt-formatted. It conflicts in `RiskRecord.tsx` and `RisksWorkspace.test.tsx`. | Fix formatting, rebase and resolve the two Risk UI conflicts separately. It does not block Tasks 2–4. |
| #347, exact decimal measurement and concern labels | The numeric integrity and label changes are worthwhile, but it conflicts with current monitoring and risk UI files: `internal/monitoring/model.go`, `scoring.go`, `MonitoringSetup.tsx`/test, and `RiskIndicatorsSection.tsx`. | Rebase and resolve as an independent risk/monitoring change. Do not combine its conflict resolution with issue-form work. |
| #191, V1 response contract gate | Includes #190's response DTO/client changes plus the CI gate. Merge simulation is clean but its September base is stale. | Rebase, run the contract gate, then merge once. |
| #190, canonical distribution response keys | Fully contained in #191. | Close as superseded; do not merge both. |

### Task 1: Integrate the exact Form Library deep link

**Files:**
- Modify: `web/src/components/MatterInternalFormRequestsPanel.tsx`
- Modify: `web/src/components/MatterInternalFormRequestsPanel.test.tsx`
- Dependency: PR #334 after rebase and merge (`web/src/components/FormsWorkspace.tsx`, `web/src/formsApi.ts`, and the exact library-item API route)

- [ ] **Step 1: Rebase and verify PR #334 before relying on its route.**

  Run:

  ```powershell
  git fetch origin --prune
  git switch fix/forms-exact-detail-deeplink
  git rebase origin/main
  npm --prefix web run test -- src/components/FormsWorkspace.location.test.tsx src/components/FormsWorkspace.dashboard.test.tsx
  npm --prefix web run typecheck
  ```

  Expected: the branch rebases without losing the exact `GET /api/v1/forms/library/{id}` authorization and a form outside the library page opens from `#forms/{id}`.

- [ ] **Step 2: Add a failing handoff test at the issue boundary.**

  In `MatterInternalFormRequestsPanel.test.tsx`, make the saved draft fixture return `id: "form-created"`, then assert that the success notice contains:

  ```tsx
  <ActionLink href="#forms/form-created">Open form draft</ActionLink>
  ```

  The test must also assert that the link is absent before a successful save and that the draft origin remains `{ type: "MATTER", id: "matter-a" }`.

- [ ] **Step 3: Run the new test and confirm the missing handoff.**

  Run:

  ```powershell
  npm --prefix web run test -- src/components/MatterInternalFormRequestsPanel.test.tsx
  ```

  Expected before implementation: the test fails because the saved draft has no next action.

- [ ] **Step 4: Preserve the created `FormTemplate` and render the one direct next action.**

  In `MatterInternalFormRequestsPanel.tsx`, add `const [createdDraft, setCreatedDraft] = useState<FormTemplate>();`, receive the `form` argument in `onSaved`, and replace the generic success string with a concise notice:

  ```tsx
  onSaved={(form) => {
    setAuthorOpen(false);
    setCreatedDraft(form);
    setNotice("Form draft created.");
  }}
  ```

  Render the link only beside that saved notice:

  ```tsx
  {notice && <Notice tone="success">
    {notice}
    {createdDraft && <> <ActionLink href={`#forms/${encodeURIComponent(createdDraft.id)}`}>Open form draft</ActionLink></>}
  </Notice>}
  ```

  Clear `createdDraft` when starting a new draft or employee request so an old action does not appear beside unrelated feedback.

- [ ] **Step 5: Verify the handoff and commit it separately.**

  Run:

  ```powershell
  npm --prefix web run test -- src/components/MatterInternalFormRequestsPanel.test.tsx src/components/FormsWorkspace.location.test.tsx
  npm --prefix web run typecheck
  git diff --check
  git add web/src/components/MatterInternalFormRequestsPanel.tsx web/src/components/MatterInternalFormRequestsPanel.test.tsx
  git commit -m "fix: hand off issue-created form drafts"
  ```

  Expected: the authored form opens directly in the standard Forms detail drawer, where the existing approval workflow remains authoritative.

### Task 2: Make issue form activity complete, bounded and actionable

**Files:**
- Modify: `web/src/components/forms/SubjectFormActivity.tsx`
- Modify: `web/src/components/forms/SubjectFormActivity.test.tsx`
- Modify: `web/src/components/forms/SentFormsView.tsx`
- Modify: `web/src/components/forms/sent/SentFormsView.test.tsx`
- Modify: `web/src/components/forms/formsLocation.ts` only if a shared hash-query helper avoids duplicated parsing without changing the Templates URL contract

- [ ] **Step 1: Write failing activity tests for both cursors and sent-form navigation.**

  Add a test where the first request and response pages both return `next_cursor`, then the second pages return new IDs. Assert that:

  ```tsx
  screen.getByRole("button", { name: "Load more form requests" })
  screen.getByRole("button", { name: "Load more submitted responses" })
  screen.getByRole("link", { name: "Open sent form" })
  ```

  exist, call the correct cursor-bearing API requests, retain the first-page rows, and remove the relevant button when the cursor is empty. The request link must be `#forms?section=sent-forms&distribution=distribution-open`.

- [ ] **Step 2: Run the focused test and confirm the present limitation.**

  Run:

  ```powershell
  npm --prefix web run test -- src/components/forms/SubjectFormActivity.test.tsx
  ```

  Expected before implementation: no load-more controls and no sent-form link exist.

- [ ] **Step 3: Replace the passive cursor flag with independent cursor state and bounded append operations.**

  In `SubjectFormActivity.tsx`:

  ```tsx
  const [requestCursor, setRequestCursor] = useState<string>();
  const [responseCursor, setResponseCursor] = useState<string>();
  const [loadingMoreRequests, setLoadingMoreRequests] = useState(false);
  const [loadingMoreResponses, setLoadingMoreResponses] = useState(false);
  ```

  Load the first page as today, storing `page.next_cursor`. Add `loadMoreRequests` and `loadMoreResponses` that call the same APIs with `cursor`, append only new IDs, retain prior rows on failure, and show a local retry notice. Do not fetch an unbounded population or display the current page length as a total.

  Render a `Load more form requests` button below Requests and a `Load more submitted responses` button below Submitted responses. Replace “More available” with the button; retain `n shown` as a page-derived count only.

  Render `Open sent form` for every request:

  ```tsx
  <ActionLink href={`#forms?section=sent-forms&distribution=${encodeURIComponent(request.id)}`}>
    Open sent form
  </ActionLink>
  ```

- [ ] **Step 4: Make Sent Forms honour the hash deep link and read its filters from the hash.**

  In `SentFormsView.tsx`, replace `window.location.search` parsing with the query part of `window.location.hash`. Read an optional `distribution` value. When set, load that exact distribution through the existing `loadDistribution(id)` read even when it is not on the current list page; keep the list filter state unchanged. On close, remove only `distribution` from the hash query.

  The implementation must not infer access from the client: a denied or unavailable exact read remains an explicit detail error while the list continues to render.

- [ ] **Step 5: Test the deep link, pagination and degradation paths.**

  Add tests proving:

  ```tsx
  window.history.replaceState(null, "", "/#forms?section=sent-forms&distribution=distribution-99");
  ```

  opens the exact distribution detail outside the first page; a forbidden exact read does not clear the sent-form list; request-page failure retains the original rows and offers retry; and response pagination preserves the `Review response` deep link.

- [ ] **Step 6: Verify and commit the activity slice.**

  Run:

  ```powershell
  npm --prefix web run test -- src/components/forms/SubjectFormActivity.test.tsx src/components/forms/sent/SentFormsView.test.tsx src/components/FormsWorkspace.location.test.tsx
  npm --prefix web run typecheck
  git diff --check
  git add web/src/components/forms/SubjectFormActivity.tsx web/src/components/forms/SubjectFormActivity.test.tsx web/src/components/forms/SentFormsView.tsx web/src/components/forms/sent/SentFormsView.test.tsx web/src/components/forms/formsLocation.ts
  git commit -m "fix: complete issue form activity navigation"
  ```

### Task 3: Retain a bounded history of forms authored for an issue

**Files:**
- Modify: `internal/monitoring/model.go`
- Modify: `internal/monitoring/repository.go`
- Modify: `internal/monitoring/memory.go`
- Modify: `internal/monitoring/postgres.go`
- Modify: `internal/monitoring/service.go`
- Modify: `internal/httpapi/forms_handlers.go`
- Modify: `web/src/formsTypes.ts`
- Modify: `web/src/formsApi.ts`
- Modify: `web/src/components/MatterInternalFormRequestsPanel.tsx`
- Modify: `web/src/components/MatterInternalFormRequestsPanel.test.tsx`
- Test: `internal/monitoring/form_library_test.go`
- Test: `internal/httpapi/forms_handlers_test.go`

- [ ] **Step 1: Define a narrow origin filter and write service failures first.**

  Extend `FormLibraryFilter` with an all-or-none origin pair:

  ```go
  OriginType FormOriginType
  OriginID   string
  ```

  Add tests that a `MATTER` origin filter returns only forms whose immutable origin is that exact issue within the verified tenant/legal entity, rejects a missing half-pair, and cannot retrieve a cross-entity issue form.

- [ ] **Step 2: Run the monitoring tests to confirm the filter is absent.**

  Run:

  ```powershell
  go test ./internal/monitoring -run "FormLibrary.*Origin|ListFormLibrary" -count=1
  ```

  Expected before implementation: the filter fields and origin-scoped behaviour do not exist.

- [ ] **Step 3: Implement the filter in existing library reads, not a second issue-form endpoint.**

  Validate the pair in `ListFormLibrary`; pass it to both memory and PostgreSQL repositories. PostgreSQL must constrain `tenant_id`, `legal_entity_id`, `origin_type`, and `origin_id` before the existing keyset limit. Reuse the existing `form_template_matter_origin` columns; add an index only if `EXPLAIN` on the bounded production-shaped query does not use an origin-leading bounded path.

  The HTTP list route accepts `origin_type` and `origin_id` and overwrites tenant/entity from verified identity exactly as the existing library list does.

- [ ] **Step 4: Render authored forms as their own compact group.**

  In `MatterInternalFormRequestsPanel.tsx`, load the first bounded page of library items with:

  ```ts
  loadFormTemplatePage({ origin_type: "MATTER", origin_id: matterID, limit: 6 })
  ```

  Render `Linked forms` above request activity. Each row shows name, revision, lifecycle status, and `Open form` linking to `#forms/{id}`. Include a `Load more linked forms` action only when the server supplies a cursor. Do not duplicate form fields, approval actions, or lifecycle controls; the Forms detail remains the operational record.

- [ ] **Step 5: Verify authority and source-boundary behaviour.**

  Add API tests for verified scope and browser tests for loading, empty, unavailable, next-page, draft and active states. Confirm no form title from another legal entity appears before the list limit.

- [ ] **Step 6: Verify and commit the durable history slice.**

  Run:

  ```powershell
  go test ./internal/monitoring ./internal/httpapi -run "FormLibrary.*Origin|FormOrigin|Library" -count=1
  npm --prefix web run test -- src/components/MatterInternalFormRequestsPanel.test.tsx
  npm --prefix web run typecheck
  git diff --check
  git add internal/monitoring internal/httpapi web/src/formsTypes.ts web/src/formsApi.ts web/src/components/MatterInternalFormRequestsPanel.tsx web/src/components/MatterInternalFormRequestsPanel.test.tsx
  git commit -m "feat: list forms linked to an issue"
  ```

### Task 4: Make board-brief availability recoverable

**Files:**
- Modify: `web/src/components/MatterBoardBriefAction.tsx`
- Modify: `web/src/components/MatterBoardBriefAction.test.tsx`

- [ ] **Step 1: Write failing tests for both non-generatable states.**

  Cover an availability response without a definition but with `reason: "No active board brief setup is available for this issue."`, and a definition where `can_run` is false. Assert a labelled read-only state is visible in both cases, neither exposes `Generate board brief`, and the missing-setup state includes `Open Reports`.

- [ ] **Step 2: Run the component test to confirm the missing-setup state is currently hidden.**

  Run:

  ```powershell
  npm --prefix web run test -- src/components/MatterBoardBriefAction.test.tsx
  ```

  Expected before implementation: the no-definition case returns `null` even though the API supplied a recovery reason.

- [ ] **Step 3: Render a concise recovery state without weakening report governance.**

  In `MatterBoardBriefAction.tsx`, handle the no-definition case before returning `null`:

  ```tsx
  if (!availability?.definition) {
    return <p className="matter-board-brief-state">
      {availability?.reason || "No active board brief setup is available for this issue."} <ActionLink href="#reports">Open Reports</ActionLink>
    </p>;
  }
  ```

  Keep the existing no-authority and not-assigned states read-only. Do not add a client-side bypass, create a definition from the issue, or reveal hidden report definitions.

- [ ] **Step 4: Verify and commit the recovery state.**

  Run:

  ```powershell
  npm --prefix web run test -- src/components/MatterBoardBriefAction.test.tsx src/components/reports/ReportsWorkspace.test.tsx
  npm --prefix web run typecheck
  git diff --check
  git add web/src/components/MatterBoardBriefAction.tsx web/src/components/MatterBoardBriefAction.test.tsx
  git commit -m "fix: explain unavailable issue board briefs"
  ```

### Task 5: Integrated acceptance and release decision

**Files:**
- Modify: `docs/superpowers/plans/2026-10-06-issue-workspace-follow-through.md` only to mark verified tasks complete after evidence is captured
- Modify: `web/scripts/capture-matter-context-evidence.mjs` only if its existing journey cannot capture the new direct links and paged states

- [ ] **Step 1: Capture the four operational states.**

  Record desktop and narrow evidence for: a newly drafted linked form and its direct Forms handoff; an issue with more than six sent forms/responses; an exact sent-form deep link outside the first page; and no active board-brief setup.

- [ ] **Step 2: Run targeted behavioural checks.**

  Run:

  ```powershell
  go test ./internal/monitoring ./internal/httpapi ./internal/reporting -count=1
  npm --prefix web run test -- src/components/MatterRecordWorkspace.test.tsx src/components/MatterInternalFormRequestsPanel.test.tsx src/components/forms/SubjectFormActivity.test.tsx src/components/forms/sent/SentFormsView.test.tsx src/components/MatterBoardBriefAction.test.tsx src/components/reports/ReportsWorkspace.test.tsx
  npm --prefix web run typecheck
  npm --prefix web run check:ui-contracts
  git diff --check
  ```

- [ ] **Step 3: Check the release order before deployment.**

  Confirm #334 is merged, #190 is closed as superseded, #191 is either merged or explicitly deferred, and #342/#347 have been independently rebased rather than accidentally pulled into this change. Do not deploy while any migration, Forms deep link, or report download check is failing.

- [ ] **Step 4: Commit the verified plan status.**

  Run:

  ```powershell
  git add docs/superpowers/plans/2026-10-06-issue-workspace-follow-through.md web/scripts/capture-matter-context-evidence.mjs
  git commit -m "docs: record issue workspace follow-through evidence"
  ```

## Acceptance criteria

- An issue-created draft has a direct, exact handoff to its normal Forms record and retains immutable issue origin.
- The issue displays a bounded, paginated list of its authored forms, sent requests and submitted responses without reporting page size as a total.
- A request row opens its exact sent-form record even when it is outside the current list page; a response opens its exact review record.
- Request and response read failures preserve other available issue work and provide a local recovery action.
- An unavailable board brief names the actual condition and the valid recovery path; only an authorised, active definition permits generation.
- All reads remain verified-tenant and legal-entity scoped, and the implementation adds no separate issue-form, issue-report or client-authorised workflow.
