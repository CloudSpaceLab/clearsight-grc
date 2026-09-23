# Matter Collaboration Workspace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a scalable issue workspace with durable internal comments, safe mentions, update requests and a practical queue/detail workflow.

**Architecture:** Matter collaboration is represented by append-only Matter events, so history, reconstruction and the transactional outbox retain one source of truth. The HTTP API exposes bounded activity pages and verified-actor commands; the React workspace composes existing record panels into retained work tabs beside a paged activity timeline. Vendor exceptions move from in-memory slicing to a server-bounded query contract.

**Tech Stack:** Go, PostgreSQL/pgx, existing continuity event/outbox runtime, React/TypeScript/Vite, existing shared UI contracts, Playwright rendered evidence.

---

### Task 1: Define collaboration event and read contracts

**Files:**
- Modify: `internal/continuity/model.go`
- Modify: `internal/continuity/matter_edits.go`
- Modify: `internal/continuity/service.go`
- Modify: `internal/continuity/memory.go`
- Test: `internal/continuity/matter_collaboration_test.go`

- [ ] **Step 1: Write the failing domain tests**

```go
func TestAddMatterCommentRecordsMentionedPeople(t *testing.T) {
    // Create a Matter, add a comment with one permitted principal, then assert
    // the returned activity item has the verified actor, body and mention.
}

func TestRequestMatterActionUpdateLeavesActionOpen(t *testing.T) {
    // Request an update for an open Action and assert the action state is
    // unchanged while the activity contains an AWAITING_RESPONSE request.
}
```

- [ ] **Step 2: Run the focused domain tests and confirm the expected missing-contract failure**

Run: `go test ./internal/continuity -run 'Test(AddMatterComment|RequestMatterActionUpdate)' -count=1`

Expected: FAIL because collaboration command types and methods do not exist.

- [ ] **Step 3: Add minimal append-only command and page types**

```go
type AddMatterCommentInput struct { TenantID, MatterID string; ExpectedVersion int64; ActorID, Body, ActionID string; MentionedPrincipalIDs []string }
type RequestMatterActionUpdateInput struct { TenantID, MatterID, ActionID string; ExpectedVersion int64; ActorID, Message string; DueAt time.Time }
type MatterActivityPage struct { Items []MatterActivityItem `json:"items"`; NextCursor string `json:"next_cursor,omitempty"`; GeneratedAt time.Time `json:"generated_at"` }
```

Create `MATTER_COMMENT_ADDED` and `MATTER_ACTION_UPDATE_REQUESTED` events, validate non-empty bounded bodies, action membership and future response deadline, then apply the event through the existing `applyMatterValueAndResult` path. Keep comments and requests out of aggregate decision/action lifecycle projection so a comment cannot change material state.

- [ ] **Step 4: Run the focused domain tests and confirm they pass**

Run: `go test ./internal/continuity -run 'Test(AddMatterComment|RequestMatterActionUpdate)' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the domain contract**

```bash
git add internal/continuity/model.go internal/continuity/matter_edits.go internal/continuity/service.go internal/continuity/memory.go internal/continuity/matter_collaboration_test.go
git commit -m "feat: add Matter collaboration events"
```

### Task 2: Persist and page activity with verified HTTP commands

**Files:**
- Modify: `internal/continuity/repository.go`
- Modify: `internal/continuity/postgres.go`
- Modify: `internal/httpapi/continuity_handlers.go`
- Modify: `internal/httpapi/command_lifecycle.go`
- Modify: `internal/httpapi/matter_operations.go`
- Modify: `internal/httpapi/route_registry.go`
- Test: `internal/httpapi/matter_collaboration_handlers_test.go`

- [ ] **Step 1: Write failing HTTP tests for bounded access and actor binding**

```go
func TestMatterActivityUsesKeysetCursorAndVisibilityScope(t *testing.T) { /* newest 20 only; cursor obtains next page */ }
func TestMatterCommentIgnoresBodyActorAndRejectsInaccessibleMention(t *testing.T) { /* verified actor is stored; unpermitted mention is rejected */ }
func TestUpdateRequestRequiresActionManagementAuthority(t *testing.T) { /* performer cannot request an update unless route permits it */ }
```

- [ ] **Step 2: Run them and confirm route/handler absence**

Run: `go test ./internal/httpapi -run 'TestMatter(ActivityUsesKeyset|CommentIgnores|UpdateRequestRequires)' -count=1`

Expected: FAIL because the activity route and commands are not registered.

- [ ] **Step 3: Implement the bounded API surface**

Add `GET /api/v1/matters/{id}/activity?limit=20&cursor=` as an authenticated visibility-gated read. Add material `POST /api/v1/matters/{id}/comments` and `POST /api/v1/matters/{id}/actions/{action_id}/update-requests`; bind the actor from identity and validate mentions against a repository access query before event persistence. Add `matter.comment.add` and `matter.action.update.request` operations with exact Matter/Action authority. Query events by descending `(occurred_at,id)` keyset and resolve display names with one bounded join.

- [ ] **Step 4: Run focused HTTP tests**

Run: `go test ./internal/httpapi -run 'TestMatter(ActivityUsesKeyset|CommentIgnores|UpdateRequestRequires)' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the HTTP contract**

```bash
git add internal/continuity/repository.go internal/continuity/postgres.go internal/httpapi/continuity_handlers.go internal/httpapi/command_lifecycle.go internal/httpapi/matter_operations.go internal/httpapi/route_registry.go internal/httpapi/matter_collaboration_handlers_test.go
git commit -m "feat: expose Matter collaboration activity"
```

### Task 3: Deliver update and mention notifications through the existing outbox

**Files:**
- Modify: `internal/workflow/assignment_notification.go`
- Modify: `internal/workflow/assignment_notification_postgres.go`
- Create: `migrations/000091_matter_collaboration_notification.up.sql`
- Create: `migrations/000091_matter_collaboration_notification.down.sql`
- Test: `internal/workflow/matter_collaboration_notification_test.go`

- [ ] **Step 1: Write failing delivery tests**

```go
func TestUpdateRequestNotifiesCurrentActionPerformer(t *testing.T) { /* event resolves current performer and creates one receipt */ }
func TestMentionDoesNotDeliverToPersonWithoutMatterAccess(t *testing.T) { /* delivery records recipient rejection, no message */ }
```

- [ ] **Step 2: Run the focused worker tests**

Run: `go test ./internal/workflow -run 'Test(UpdateRequestNotifies|MentionDoesNotDeliver)' -count=1`

Expected: FAIL because these event types are ignored by the worker.

- [ ] **Step 3: Extend durable delivery handling**

Use the existing governed email builder and idempotent receipt table pattern. Resolve the action performer at delivery time for update requests and confirm Matter visibility for a mentioned person. Store one delivery receipt per outbox event and recipient, preserve temporary retry behavior, and report unavailable/rejected mail as a final delivery condition. Add the migration with tenant, entity, outbox-event and recipient indexes.

- [ ] **Step 4: Run the focused worker tests**

Run: `go test ./internal/workflow -run 'Test(UpdateRequestNotifies|MentionDoesNotDeliver)' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit delivery support**

```bash
git add internal/workflow/assignment_notification.go internal/workflow/assignment_notification_postgres.go migrations/000091_matter_collaboration_notification.up.sql migrations/000091_matter_collaboration_notification.down.sql internal/workflow/matter_collaboration_notification_test.go
git commit -m "feat: notify Matter update recipients"
```

### Task 4: Add web API bindings and activity timeline

**Files:**
- Modify: `web/src/api.ts`
- Modify: `web/src/types.ts`
- Create: `web/src/matterCollaborationApi.ts`
- Create: `web/src/components/MatterActivityTimeline.tsx`
- Create: `web/src/components/MatterActivityTimeline.test.tsx`

- [ ] **Step 1: Write the failing timeline tests**

```tsx
it("loads an older activity page without replacing the newest items", async () => { /* mock two cursor pages */ });
it("sends an update request only for the selected open action", async () => { /* choose action then submit */ });
```

- [ ] **Step 2: Run the component tests**

Run: `npm test -- MatterActivityTimeline.test.tsx --runInBand`

Expected: FAIL because the component and collaboration API do not exist.

- [ ] **Step 3: Implement the minimal timeline composition**

Expose typed `loadMatterActivity`, `addMatterComment` and `requestMatterActionUpdate` clients. Render comment and change filters, newest-first entries, a load-older control and an accessible composer. Mention selection is a supplied eligible candidate list; it emits plain text plus principal IDs, not email addresses. Give update-request state an explicit action, assignee, deadline and delivery label.

- [ ] **Step 4: Run the component tests**

Run: `npm test -- MatterActivityTimeline.test.tsx --runInBand`

Expected: PASS.

- [ ] **Step 5: Commit web collaboration primitives**

```bash
git add web/src/api.ts web/src/types.ts web/src/matterCollaborationApi.ts web/src/components/MatterActivityTimeline.tsx web/src/components/MatterActivityTimeline.test.tsx
git commit -m "feat: add Matter activity timeline"
```

### Task 5: Recompose the issue record around work tabs and activity

**Files:**
- Modify: `web/src/components/MatterRecordWorkspace.tsx`
- Modify: `web/src/components/MatterActionsPanel.tsx`
- Modify: `web/src/matter-record.css`
- Modify: `web/src/components/MatterRecordWorkspace.test.tsx`

- [ ] **Step 1: Write failing workspace behavior tests**

```tsx
it("keeps the Actions draft after switching to Evidence and back", async () => { /* retained tab panel */ });
it("keeps accountable owner reassignment separate from action performer reassignment", async () => { /* distinct actions and labels */ });
```

- [ ] **Step 2: Run the focused workspace tests**

Run: `npm test -- MatterRecordWorkspace.test.tsx --runInBand`

Expected: FAIL because all panels are currently rendered in one grid.

- [ ] **Step 3: Implement the two-column workspace**

Keep the header and current handoff compact. Place Details, Actions, Evidence and Decisions in retained shared tabs on the left; place `MatterActivityTimeline` in the right activity rail. Move action reassignment, action status and update request into a concise action menu/sheet while keeping issue-owner reassignment a separately named accountable-owner workflow. At narrow widths replace the rail with Work/Activity tabs and maintain a readable single-column order.

- [ ] **Step 4: Run the focused workspace tests**

Run: `npm test -- MatterRecordWorkspace.test.tsx --runInBand`

Expected: PASS.

- [ ] **Step 5: Commit workspace composition**

```bash
git add web/src/components/MatterRecordWorkspace.tsx web/src/components/MatterActionsPanel.tsx web/src/matter-record.css web/src/components/MatterRecordWorkspace.test.tsx
git commit -m "feat: streamline Matter work workspace"
```

### Task 6: Replace vendor exception slicing with server-bounded pagination

**Files:**
- Modify: `internal/httpapi/vendor_work_handlers.go`
- Modify: `internal/thirdparty/relationship_link_postgres.go`
- Modify: `web/src/vendorRiskWork.ts`
- Modify: `web/src/components/VendorPortfolio.tsx`
- Test: `internal/httpapi/vendor_exception_page_test.go`
- Test: `web/src/components/VendorPortfolio.test.tsx`

- [ ] **Step 1: Write the failing pagination tests**

```go
func TestVendorExceptionPageUsesCursorBeforeMaterialization(t *testing.T) { /* 21 findings yields 20 and next cursor */ }
```

```tsx
it("restores the selected exception page after opening and closing a Matter", async () => { /* route query state */ });
```

- [ ] **Step 2: Run both focused tests**

Run: `go test ./internal/httpapi -run TestVendorExceptionPageUsesCursorBeforeMaterialization -count=1`

Run: `npm test -- VendorPortfolio.test.tsx --runInBand`

Expected: FAIL because vendor exceptions are currently materialized then locally sliced.

- [ ] **Step 3: Implement page-first vendor exceptions**

Add a visibility-scoped endpoint that accepts service/filter/cursor/limit and returns 20 linked open findings with an opaque next cursor. Build row metadata in the query rather than loading the complete relationship and Matter set in the browser. Bind queue filters and cursor to the vendor route and restore scroll only after the page has rendered.

- [ ] **Step 4: Run focused pagination tests**

Run: `go test ./internal/httpapi -run TestVendorExceptionPageUsesCursorBeforeMaterialization -count=1`

Run: `npm test -- VendorPortfolio.test.tsx --runInBand`

Expected: PASS.

- [ ] **Step 5: Commit queue pagination**

```bash
git add internal/httpapi/vendor_work_handlers.go internal/thirdparty/relationship_link_postgres.go web/src/vendorRiskWork.ts web/src/components/VendorPortfolio.tsx internal/httpapi/vendor_exception_page_test.go web/src/components/VendorPortfolio.test.tsx
git commit -m "feat: paginate vendor exceptions"
```

### Task 7: Record acceptance evidence and release

**Files:**
- Create: `docs/acceptance/2026-09-10-matter-collaboration-workspace.md`
- Create: `docs/evidence/2026-09-10-matter-collaboration-workspace/README.md`
- Create: rendered evidence files under `docs/evidence/2026-09-10-matter-collaboration-workspace/`

- [ ] **Step 1: Run only affected checks**

Run: `go test ./internal/continuity ./internal/httpapi ./internal/workflow -run 'Test(AddMatterComment|RequestMatterActionUpdate|MatterActivityUsesKeyset|MatterCommentIgnores|UpdateRequestRequires|UpdateRequestNotifies|MentionDoesNotDeliver|VendorExceptionPage)' -count=1`

Run: `npm test -- MatterActivityTimeline.test.tsx MatterRecordWorkspace.test.tsx VendorPortfolio.test.tsx copyQuality.test.ts --runInBand`

Run: `npm run build`

Expected: focused checks and production web build pass.

- [ ] **Step 2: Capture rendered states**

Capture desktop light, desktop dark and 390px narrow layouts for an issue with activity and a paginated vendor exception queue. Inspect each capture; correct the highest-impact density, clipping, focus or copy failure before preserving evidence.

- [ ] **Step 3: Write acceptance evidence**

Record exact commit, commands, captured states, known mail delivery condition and the observed ownership/reassignment separation. Do not claim an email was delivered unless the local controlled delivery receipt confirms it.

- [ ] **Step 4: Commit release evidence**

```bash
git add docs/acceptance/2026-09-10-matter-collaboration-workspace.md docs/evidence/2026-09-10-matter-collaboration-workspace
git commit -m "docs: record Matter collaboration acceptance"
```

## Plan review

- Coverage: Tasks 1–3 implement durable comments, mentions, update requests and delivery receipts; Tasks 4–5 implement the usable issue workspace; Task 6 covers scalable outstanding items; Task 7 provides targeted verification and visual evidence.
- Boundaries: accountable issue ownership and action performer reassignment remain distinct; mention notification never grants access; update requests never complete actions.
- Placeholder check: all new types, routes, test commands and expected outcomes are specified. Migration `000091` follows the current contiguous migration sequence.
