# Employee Work Profile Implementation Plan

> **For agentic workers:** Use executing-plans to implement this plan inline, task by task. Steps use checkbox syntax for tracking. Do not dispatch subagents unless the operator requests delegation.

**Goal:** Make internal employee names open a durable profile with visible assignments, completed work and recorded activity, including Hakeem's Cloudspace work.

**Architecture:** Add an actor-scoped read service over existing identities, canonical work records and recorded events. Keep the viewer identity distinct from the employee being inspected; filter records in the repository before counting or pagination. React consumes that service through a shared person link and a dedicated route.

**Tech Stack:** Go, PostgreSQL/pgx, React/TypeScript, existing shared UI components and hash routing.

**Approved specification:** `docs/superpowers/specs/2026-09-10-employee-work-profile-design.md`.

---

## Execution constraints

The operator approved the design and requested this implementation plan. This document does not claim implementation or deployment. Preserve the existing demo identities and curated records; no full reseed is required to create profiles. Verification is limited to meaningful access/attribution checks, existing affected checks and rendered inspection. Do not add another seed regression suite or manufacture activity to populate Hakeem's profile.

The inspected checkout is `codex/vendor-semantic-review`, with design-only commits ahead of an older main and unrelated untracked artifacts. Before implementation, use a clean worktree based on current main and carry only the employee design/plan changes across. Do not reset, clean or repurpose the existing checkout. All paths below are repository-relative to that implementation worktree.

Required reading at execution: root `AGENTS.md`, `README.md`, `DESIGN.md`, `docs/README.md`, the approved specification, `docs/architecture/application-architecture.md`, `docs/architecture/durable-schema-ownership.d/system-activity-audit.md`, `docs/engineering/enterprise-identity-access.md`, relevant Oversight sections of `docs/implementation-plan.md`, and existing activity/visibility acceptance checks.

## File ownership

| Files | Responsibility |
| --- | --- |
| New `internal/people/model.go`, `service.go`, `memory.go` | Public profile contracts, viewer/subject authorization and in-memory repository |
| New `internal/people/postgres.go`, `postgres_work.go`, `postgres_activity.go` | Exact identity reads, work/count queries and visible event queries |
| New `internal/people/service_test.go`, `postgres_integration_test.go` | Focused access and attribution checks |
| New `internal/httpapi/people_handlers.go`, `people_routes.go`, `people_handlers_test.go` | HTTP parsing and error mapping |
| Existing `internal/httpapi/server.go`, `route_catalog.go`, `actor_read_handlers.go` | Service dependency, route registration and navigation capabilities |
| Existing `cmd/api/services_postgres.go` and composition files identified in Task 1 | PostgreSQL and in-memory service wiring |
| Existing `internal/httpapi/record_operations.go`, `matter_operations.go`, `program_operations.go` | Resolved read-only principal references |
| Existing `api/runtime.openapi.json` | Executable route/access contract |
| New `web/src/peopleApi.ts`, `components/people/PrincipalLink.tsx`, `EmployeeProfile.tsx`, `EmployeeWork.tsx`, `EmployeeActivity.tsx`, `people.css` | Client contract, navigation and profile rendering |
| Existing `web/src/appRouting.ts`, `App.tsx`, `types.ts`, `api.ts` | Route, profile target, actor capabilities and responsible-party types |
| Existing owner/reviewer UI files listed in Task 6 | Shared clickable person names |
| New `web/src/components/people/EmployeeProfile.test.tsx`, `web/evidence/employee-profile.tsx` | Focused interaction checks and rendered fixtures |
| Existing `web/src/evidenceMain.tsx`, `web/ui-contract-migrations.json` | Evidence entry and accurate component adoption registration |
| New `docs/acceptance/2026-09-10-employee-work-profile.md` | Actual verification and hosted receipt |

Do not split or refactor unrelated modules. Add an index migration only if query inspection shows the required access path is absent; allocate the next migration number from the execution checkout, not this older checkout.

## Task 1: Establish the implementation baseline

- [ ] Inspect `git status --short --branch`, `git worktree list` and current main. Create `codex/employee-work-profile` in a dedicated worktree using the repository worktree conventions. Carry the approved specification and this plan without unrelated semantic-review changes.
- [ ] Read the required documents and compare current main with the file map. Locate service composition using `rg -n 'NewService|Activity:|Oversight:|AccessAdmin:' cmd/api`.
- [ ] Inspect the existing person, current-work, event and record visibility queries. Record exact canonical source tables and event types in the acceptance receipt. In particular, inspect `internal/activity/postgres_sources.go`, `internal/workflow/postgres.go`, `internal/continuity/imported_finding_postgres.go` and `internal/httpapi/principal_labels.go`.
- [ ] Retain the supplied Hakeem Action-card screenshot as the before-state reference. Read the current hosted revision and Hakeem assignment population; retain only non-sensitive IDs/counts in the receipt.
- [ ] Confirm whether imported assignments have an initiating actor and source event. When an actor is absent, the profile will state “Actor not recorded”; it must never infer that the employee performed the import.

## Task 2: Define the scoped profile contract

- [ ] Add `internal/people/model.go` with explicit viewer/subject inputs and bounded page contracts. The service contract is:

```go
type Scope struct {
    Viewer identity.Actor
    PersonID string
}
type PageQuery struct {
    Scope Scope
    Limit int
    Cursor string
    From *time.Time
    To *time.Time
    Category string
    State string
}
type Repository interface {
    Profile(context.Context, Scope) (Profile, error)
    Work(context.Context, PageQuery) (WorkPage, error)
    Assignments(context.Context, PageQuery) (AssignmentPage, error)
    Activity(context.Context, PageQuery) (ActivityPage, error)
}
```

`Profile` contains person ID/name/status, optional current positions/function, sample flag, identity-detail capability, source freshness and nullable metrics. The four metrics are `active`, `overdue`, `blocked`, `awaiting_outcome`. Completed work is available through the work-state filter, not silently merged into active work.

`WorkPage` rows contain stable ID, responsibility, object type/ID, parent record type/ID, title, status/label, deadline, material version and last-update time. `AssignmentPage` rows contain event ID, timestamp, employee ID, initiating actor ID/name when recorded, responsibility, previous/new assignment, reason and target. `ActivityPage` rows contain the existing safe activity fields plus an authorized target reference. All pages contain `items`, `next_cursor`, `as_of`, `coverage` and `state` (`CURRENT`, `PARTIAL`, `UNAVAILABLE`).

- [ ] Implement `service.go` authorization using the original verified viewer. Self is allowed; another employee requires current `OVERSIGHT_READ` or `IDENTITY_READ` for the applicable entity/department. Do not resolve the subject into a replacement request actor or call their Today endpoint as an impersonation shortcut.
- [ ] Normalize limit to 25 by default and cap at 100. Validate timestamps, date ordering, category/state enums and cursor integrity. Bind cursors to viewer, subject, entity, applied filters and an initial as-of time; changing filters starts a new page sequence.
- [ ] Define explicit not-found, invalid-query and unavailable errors. Unknown/missing subject and denied subject access return the same not-found result. A failed identity lookup must fail closed when it prevents proving subject scope; a failed optional position lookup may produce partial details after scope is independently established.
- [ ] Add a small service test covering self, permitted reader, unauthorized peer and viewer/subject identity preservation. Provide a memory repository implementing the same contract for development; it returns actual supplied records or an empty population, never synthetic successful activity.

## Task 3: Implement canonical work, identity and assignment reads

- [ ] Resolve the subject by exact principal ID and tenant. Require `PERSON` and evidence of membership in the viewer's legal entity from current or retained historical scoped records. Do not use display-name matching or a limited directory overview to locate the subject.
- [ ] Read positions and function using effective-dated organization rows. Inactive account status and current position occupancy are independent facts; do not infer one from the other. Directory source and sign-in address require identity-read access; credentials/session values never enter this DTO.
- [ ] In `postgres_work.go`, normalize canonical responsibilities: Matter accountability, Action performance, Program accountability/authorization, safeguard performance, evidence response/review and currently routed workflow steps. Use Workflow Tasks only for responsibilities without an equivalent canonical assignment row.
- [ ] Deduplicate by `(record type, record ID, subresource ID, responsibility, employee ID)`. Count Action performance separately from Matter accountability because they are distinct duties; suppress duplicate Workflow Tasks for the same Action duty.
- [ ] Join every work source to its canonical legal entity and protected parent. Apply the existing viewer visibility policy, department limits and demo archive exclusions before aggregates and `LIMIT`. Assignment to the subject must never grant the viewer access to the subject's restricted records.
- [ ] Calculate counts and first work page in one consistent read snapshot. Active excludes completed/cancelled duties; overdue counts only active duties past a valid deadline; blocked uses stored blocked state; awaiting outcome identifies implemented work still awaiting a separate outcome check. Missing due dates do not count as overdue. Null metrics indicate failed/incomplete source coverage.
- [ ] Add current/completed work filters and separate assignment-history pagination. Reconstruct assignment intervals from recorded creation/change events, retaining the initiating actor separately from the employee receiving work. Where an interval cannot be reconstructed, keep recorded facts and report the history limitation.
- [ ] Inspect execution plans for exact subject reads and cursor pages. Reuse existing principal/owner indexes. If needed, add indexes aligned to tenant, subject and descending event time/ID through an additive migration; its downgrade drops only those new indexes. Use bounded query timeouts and return an unavailable/partial result on timeout, never fabricated zero totals.
- [ ] Add one repository integration scenario with Hakeem-like Action assignments, a reassignment by another actor, an implemented Action and a restricted record. Assert counts, before-limit visibility, no duplicate Action/task row, historical attribution and cursor continuation. Reuse repository fixture conventions and transaction cleanup.

## Task 4: Implement personal activity without changing attribution

- [ ] Reuse the canonical source definitions in `internal/activity/postgres_sources.go`; add `postgres_activity.go` for profile-specific scope and visibility joins. If query construction is shared, extract only the shared expressions into an exported helper with typed inputs; do not make the existing platform endpoint available to all profile readers.
- [ ] Match the selected employee against the recorded initiating actor. Never match `owner_principal_id`, recipient, subject or assignee as the event actor. Import events remain attributed to the installer when that is what the event records.
- [ ] Resolve events to supported Matter/Action, Program, form/evidence and vendor targets. Resolve entity through canonical target joins when event metadata omits it. Omit events whose entity/visibility cannot be established; mark coverage partial for unsupported sources without exposing hidden-record counts or names.
- [ ] Apply viewer visibility and date/category/actor predicates before keyset pagination. Deduplicate the same logical event when represented by multiple source ledgers, preferring its authoritative domain event identifier.
- [ ] Keep configuration/system activity subject to existing platform permissions and exclude secrets, tokens, raw payloads and protected response answers. Preserve recorded outcome; do not label an unknown or failed action succeeded by default.
- [ ] Extend the repository scenario to prove an assignment made by a manager appears in Assignment history but not in the employee's Activity, and a real employee-performed change appears in Activity. An empty feed must remain empty.

## Task 5: Wire HTTP and runtime composition

- [ ] Add authenticated read routes using `people_routes.go` and register them through `route_catalog.go`:

```text
GET /api/v1/people/{principal_id}/work-profile
GET /api/v1/people/{principal_id}/work
GET /api/v1/people/{principal_id}/assignments
GET /api/v1/people/{principal_id}/activity
```

The additional work/assignments endpoints supply continuation pages for the approved bounded profile design. They introduce no mutation or directory administration.

- [ ] Parse only the path subject and permitted filters in `people_handlers.go`. Set viewer/tenant/entity from `identity.Require`. Do not accept `viewer_id`, tenant overrides or user-supplied authorization flags. Map denial/missing subject to 404, invalid filters to 400 and service failure to 503 with concise retry copy.
- [ ] Add `People *people.Service` to API dependencies and wire real PostgreSQL/memory repositories in the existing composition roots. Add `people_read` to actor context for other-person navigation; self-links remain available from the verified actor ID. Per-profile authorization still runs on every request.
- [ ] Extend resolved responsibility DTOs with optional read-only `principal_id` for resolved `PERSON` records. Unresolved names and non-person roles retain plain text. Keep current command authority routes unchanged.
- [ ] Update `api/runtime.openapi.json` through the existing contract ownership procedure. Add one HTTP check for authenticated routing, rejected scope overrides and consistent not-found responses.
- [ ] Run the focused Go checks and the existing route contract checks once. Commands from repository root:

```powershell
go test ./internal/people ./internal/httpapi
go test -tags postgres ./internal/people
```

Use the established disposable PostgreSQL test configuration for the second command. Confirm integration tests ran rather than skipped. If unavailable, record that limitation and verify the same read scenarios against a disposable database before release.

## Task 6: Add reusable person navigation and profile UI

- [ ] Define matching TypeScript DTOs and fetchers in `web/src/peopleApi.ts`. Carry AbortSignal or request-generation guards so switching profiles cannot display a prior person's results. Clear protected state on actor/entity changes.
- [ ] Add `people` to the existing View union and `personID` to WorkspaceTarget in `appRouting.ts`. Parse both `#people/{id}` and `#/people/{id}`; generate the canonical approved route. Handle malformed/missing IDs without loading a different employee.
- [ ] Add the profile branch in `App.tsx`. Keep the existing application shell and browser history. A direct URL has a valid Work/Oversight fallback action when no prior app route exists; never send users to an unrelated external history entry.
- [ ] Implement the shared person link with a navigation-only contract:

```tsx
type PrincipalLinkProps = {
  principalID?: string;
  displayName: string;
  kind?: string;
  actorID?: string;
  canReadPeople: boolean;
};
```

Use the shared `ActionLink` for an authorized resolved person; otherwise return the display name as text. Authorization is `principalID === actorID || canReadPeople` plus `kind === 'PERSON'`. The destination API remains decisive. Do not turn words embedded in source prose into guessed links.

- [ ] Replace name rendering in `MatterActionsPanel.tsx`, `MatterDetailsPanel.tsx`, `MatterCurrentHandoff.tsx`, `MatterOutcomePanel.tsx`, `ProgramDetailsPanel.tsx`, `ProgramCurrentPosition.tsx`, `ProgramSafeguardsPanel.tsx`, `ProgramEvidencePanel.tsx`, `components/oversight/OversightWorkspace.tsx`, `components/configure/SystemActivityPanel.tsx` and `components/access/IdentityAccessInventory.tsx`. Pass resolved IDs/kinds from DTOs; extend owning read responses only where a verified reference is missing.
- [ ] Check adjacent bank reviewer and vendor accountable-owner labels in `components/forms/VendorResponseReview.tsx` and `VendorsWorkspace.tsx`; use the same link when an internal principal reference is present. Vendor organizations and external respondents remain ordinary vendor/response links.
- [ ] Implement `EmployeeProfile.tsx` with monogram/name/status, current positions/function, sample indication, freshness and four nullable metric cards. Build `EmployeeWork.tsx` with current/completed filters, explicit responsibility/status/deadline and direct record links. Build `EmployeeActivity.tsx` with date/category filters, separate Assignment history and cursor pagination.
- [ ] Reuse `Tabs`, `DataTable`, `StatusBadge`, `Notice`, `Button` and existing form fields. Use semantic tokens in `people.css`: desktop parallel facts, two metric columns on tablet and stacked facts on mobile. Use the compact Tabs selector at the established breakpoint. Keep inline person links readable without expanding every name into a large pill.
- [ ] Represent partial identity, failed counts, failed activity, inactive people, unavailable names and empty history explicitly. Keep successful sections usable during another section's failure. No score/ranking, fabricated completion or generic audit-success badge.

## Task 7: Verify the affected experience and document delivery

- [ ] Add focused interaction checks in `EmployeeProfile.test.tsx`: open Hakeem from an Action, change profile while a request is pending, inspect completed work, paginate activity and return to the original record. Reuse existing routing checks for parse/generate coverage.
- [ ] Add the separate evidence fixture `web/evidence/employee-profile.tsx` and register it only through `evidenceMain.tsx`. Fixtures cover populated, empty activity, partial counts, unavailable, inactive, denied and long-name states. Keep evidence modules outside the customer dependency graph.
- [ ] Run from `web`:

```powershell
npm run build
npm test -- src/components/people/EmployeeProfile.test.tsx src/appRouting.test.ts src/copyQuality.test.ts
npm run check:runtime-truth
```

Expected: build and selected checks pass. Run any additional required repository UI-contract checks only for contracts actually adopted or changed. Do not repeat broad suites after documentation-only edits.

- [ ] Render Hakeem's full profile and the source Action card at 1440px and 390px in both themes; inspect 320px reflow and keyboard focus once. Verify Back/deep links, date/category filters, long action titles and no status-pill stretching. Correct any visible defect and recheck only its affected states.
- [ ] Verify displayed metrics against the same API population and confirm all five source Cloudspace Actions retain their canonical IDs, owner and deadline. Confirm no personal activity was invented to fill an empty feed.
- [ ] Update `DESIGN.md`, `docs/README.md`, `docs/implementation-plan.md`, relevant use-case traceability (`UC-HIST-01`) and the acceptance receipt. Record actual checks, screenshots, query limitations and coverage. Mark completion only for observed behavior.
- [ ] Review the scoped diff, run `git diff --check`, and commit implementation in cohesive API/UI/documentation changes. Preserve unrelated work and avoid staging the workspace wholesale.

## Task 8: Deploy the approved demo change and confirm it

- [ ] Read the current deployment configuration without printing secrets. Confirm `CLEARSIGHT_DEMO_SEED_MODE=manual`, current/previous release IDs and sufficient disk space. Retain a recoverable database backup through the existing deployment procedure before any additive migration.
- [ ] Build/stage API, worker and web images with the same full commit SHA using the repository release workflow. Do not overwrite the manual sample population. If source associations are missing, inspect the retained import receipt and repair only identified sample associations through the existing importer; never run a broad replacement seed for this profile feature.
- [ ] Invoke `deploy/scripts/release.sh` through the established CI/deployment entrypoint. Its arguments are the full SHA and validated `/opt/clearsight-grc/incoming/<sha>.<suffix>` staging directory. The script runs migrations, starts the matching images and invokes `verify-hosted-release.sh`; retain its rollback behavior.
- [ ] On `https://clearsight.cloudspacetechs.com`, sign in as the existing CRO and open Hakeem from the Cloudspace Action. Verify five assignments, March 31, 2026 deadlines, profile reload and real history. Sign in as Hakeem using the existing demo credentials and verify his own profile. Verify an unauthorized peer cannot read another employee's profile.
- [ ] Confirm runtime revision equals the deployed SHA and API/worker/web are healthy. Record the actual Hakeem profile URL, release SHA and smoke-check results in the acceptance receipt.
- [ ] Remove only this release's identified temporary staging/build artifacts after success, preserving current/previous releases, backups, source workbooks and other applications. Resolve and inspect every deletion target first; report recovered space if anything material is removed.

## Plan review

- [x] The plan covers all approved entry points, profile states, work history, completed work, Activity, assignment attribution and sample identity requirements.
- [x] Viewer authorization and subject filtering remain distinct; restricted data is filtered before counts/pagination.
- [x] Missing event actors, identity lookup failure, partial coverage and inactive positions have explicit behavior.
- [x] Work and assignment history have continuation APIs; filters and cursor scope are defined.
- [x] Validation is focused and the plan does not create a seed regression suite.
- [x] Deployment preserves manual curated data and includes observed hosted acceptance and recovery.
