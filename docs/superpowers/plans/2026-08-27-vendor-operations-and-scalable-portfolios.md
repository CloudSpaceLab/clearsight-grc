# Vendor Operations and Scalable Portfolios Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make vendor onboarding and due diligence usable end to end from the browser, and make Programs, Issues and Changes, and vendor linking efficient at bank-scale record counts.

**Architecture:** Extend the existing third-party identity aggregate, monitoring-form lifecycle, bounded continuity summary reads, and `FocusedSheet` component rather than adding parallel records or workflows. Reference data installs an approved form with separate maker/checker identities; production tenants use the same governed browser commands. Search and structured filters execute inside tenant- and legal-entity-scoped repositories with stable keyset cursors.

**Tech Stack:** Go 1.25, PostgreSQL/pgx, React 19 with TypeScript and Vite, Vitest/Testing Library/axe-core, CSS semantic tokens, GitHub Actions and Playwright.

---

## File map

### Reference due-diligence readiness

- Create `internal/bankverticals/vendor_due_diligence.go`: canonical reference form definition and idempotent lifecycle installation.
- Modify `internal/bankverticals/service.go`: optional monitoring service dependency.
- Modify `internal/bankverticals/install_service.go`: install the form after the NDPA Program is reconciled.
- Modify `internal/bankverticals/install_test.go`: lifecycle, idempotency, entity scope, and maker-checker tests.
- Modify `cmd/seed-bank-reference/main.go`: construct the PostgreSQL monitoring service and configure the installer.
- Modify `cmd/api/services_memory.go` and `cmd/api/services_postgres.go`: configure monitoring before demo/reference installation where applicable.

### Vendor identity and website

- Create `migrations/000051_vendor_registered_address.up.sql` and `.down.sql`: additive vendor identity column.
- Modify `internal/thirdparty/model.go`: `RegisteredAddress` API/model fields.
- Modify `internal/thirdparty/service.go`: address validation and HTTPS-homepage normalization.
- Modify `internal/thirdparty/memory.go` and `internal/thirdparty/postgres.go`: storage, history/outbox payload, and reconstruction.
- Modify `internal/thirdparty/service_test.go`, `postgres_integration_test.go`, `vendor_brand_test.go`, and schema tests: red-green coverage.
- Modify `internal/httpapi/third_party_handlers_test.go`: public request/response contract and verified actor protection.
- Modify `web/src/vendorTypes.ts`, `vendorApi.ts`, `vendorIdentity.ts` and their tests: browser contract and normalization.
- Modify `web/src/components/VendorsWorkspace.tsx`, `VendorsWorkspace.test.tsx`, and `web/src/vendors.css`: fields, validation, saved identity details and brand state.

### Governed no-form recovery

- Create `web/src/vendorDueDiligenceForm.ts`: one canonical starter form input shared by static demo and setup UI.
- Create `web/src/components/VendorFormReadiness.tsx` and `.test.tsx`: focused setup, submit, and activation flow.
- Modify `web/src/staticDemo.ts`: import the canonical form definition.
- Modify `web/src/components/VendorsWorkspace.tsx` and `VendorDueDiligence.tsx`: replace the dead end with current form state and setup action.
- Modify `web/src/components/vendor-due-diligence.css`: focused form-readiness states.

### Bounded portfolio filters

- Modify `internal/continuity/summaries.go`: typed `SummaryQuery` filters and validation.
- Modify `internal/continuity/summaries_memory.go` and `summaries_postgres.go`: identical scoped filter semantics and stable cursor application.
- Modify `internal/continuity/summaries_test.go` and PostgreSQL summary integration tests: combinations, visibility, and cursor tests.
- Modify `internal/httpapi/summary_handlers.go` and `summary_handlers_test.go`: query parsing and plain-language validation errors.
- Create `migrations/000052_portfolio_summary_filters.up.sql` and `.down.sql`: supporting partial/composite indexes after query-plan evidence.
- Modify `web/src/summaryTypes.ts`, `web/src/api.ts`, and tests: typed query parameters.
- Create `web/src/workspaceFilters.ts` and `.test.ts`: route-safe parse/serialize helpers.
- Modify `web/src/appRouting.ts` and `.test.ts`: route paths ignore filter queries while retaining record targets.
- Modify `web/src/components/ProgramsWorkspace.tsx`, `MattersWorkspace.tsx`, and focused tests: debounced search, structured filters, chips, truthful counts, route restoration and responsive filter sheet.
- Modify `web/src/App.css` or the existing workspace stylesheet that owns `.workspace-toolbar`: compact desktop and focused narrow behavior.

### Focused vendor linking and completion evidence

- Modify `web/src/components/VendorRelationshipLinks.tsx`, `.test.tsx`, and `web/src/vendor-relationship-links.css`: `FocusedSheet`, debounced search, branded result rows and accessible submission.
- Modify `web/src/components/FocusedSheet.tsx` and `.test.tsx` only if the link workflow exposes a reusable focus/backdrop defect.
- Modify `DESIGN.md`, `docs/implementation-plan.md`, `docs/quality/acceptance-tests.md`, and `docs/design/programs-matters-decision-brief.md`: behavior, maturity and visual decisions.
- Create/update deterministic UI fixtures and screenshots under the existing UI-evidence paths.

## Task 1: Install the governed reference due-diligence form

**Files:**
- Create: `internal/bankverticals/vendor_due_diligence.go`
- Modify: `internal/bankverticals/service.go`
- Modify: `internal/bankverticals/install_service.go`
- Test: `internal/bankverticals/install_test.go`

- [ ] **Step 1: Write the failing installer tests**

Add tests that construct a memory monitoring repository and assert:

```go
forms, err := monitoringService.ListReusableForms(ctx, monitoring.Actor{
	TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, PrincipalID: config.OwnerPrincipalID,
}, 100)
if err != nil { t.Fatal(err) }
form := currentFormByCode(forms, "VENDOR-DUE-DILIGENCE")
if form.Status != monitoring.LifecycleActive || !form.IsCurrent {
	t.Fatalf("vendor form was not activated: %#v", form)
}
if form.SubmittedBy != config.ActorID || form.ApprovedBy != config.ReviewerPrincipalID {
	t.Fatalf("maker/checker identity was not preserved: %#v", form.Lifecycle)
}
```

Run installation twice and assert one current form and no duplicate revision. Add a negative test where maker and checker match and expect `monitoring.ErrMakerChecker`.

- [ ] **Step 2: Run the tests and verify the missing dependency failure**

Run: `go test ./internal/bankverticals -run 'TestInstallSample.*VendorDueDiligence' -count=1`

Expected: FAIL because `Service` has no monitoring dependency and no form installer.

- [ ] **Step 3: Implement the canonical form and lifecycle**

Add an optional dependency without breaking existing callers:

```go
type Service struct {
	continuity *continuity.Service
	evidence   *evidence.Service
	monitoring *monitoring.Service
}

func (s *Service) ConfigureMonitoring(service *monitoring.Service) {
	if s != nil { s.monitoring = service }
}
```

In `vendor_due_diligence.go`, return the same four sections and eight typed fields currently used by the static demo. `ensureVendorDueDiligenceForm` lists legal-entity-scoped reusable forms, reuses the current matching code, creates a draft when absent, transitions it to `PENDING_APPROVAL` as `config.ActorID`, and transitions it to `ACTIVE` as `config.ReviewerPrincipalID`. It rejects conflicting non-reference records and never activates when maker and checker are equal.

Call the helper from `InstallSample` after `ensureNDPAProgram`. When `monitoring` is nil, preserve current installer behavior so non-demo production services do not gain implicit configuration.

- [ ] **Step 4: Run focused and package tests**

Run: `go test ./internal/bankverticals -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add internal/bankverticals
git commit -m "feat: install governed vendor due diligence form"
```

## Task 2: Wire reference installation through memory and PostgreSQL runtimes

**Files:**
- Modify: `cmd/seed-bank-reference/main.go`
- Modify: `cmd/api/services_memory.go`
- Modify: `cmd/api/services_postgres.go`
- Test: `cmd/api/services_test.go` or the closest existing runtime construction tests

- [ ] **Step 1: Write failing runtime-construction tests**

Assert that demo memory startup exposes a current active `VENDOR-DUE-DILIGENCE` form and that the reference seed command’s constructed installer has monitoring configured. Use the public service read, not private field inspection.

- [ ] **Step 2: Run the focused tests**

Run: `go test ./cmd/api/... -run 'Test.*VendorDueDiligence' -count=1`

Expected: FAIL because installers are not configured with monitoring.

- [ ] **Step 3: Configure the existing monitoring service**

For PostgreSQL seeding:

```go
monitoringService := monitoring.NewService(monitoring.NewPostgresRepository(pool), evidenceService)
installer := bankverticals.NewService(continuityService, evidenceService)
installer.ConfigureMonitoring(monitoringService)
```

Use the already-created monitoring service in API runtime builders; do not construct a second repository or workflow stack.

- [ ] **Step 4: Run tests and tagged compile**

Run: `go test ./cmd/api/... -count=1`

Run: `go test -tags postgres ./cmd/seed-bank-reference ./cmd/api -run '^$'`

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add cmd/api cmd/seed-bank-reference
git commit -m "feat: seed vendor review form in reference runtimes"
```

## Task 3: Add registered address and friendly website input end to end

**Files:**
- Create: `migrations/000051_vendor_registered_address.up.sql`
- Create: `migrations/000051_vendor_registered_address.down.sql`
- Modify: `internal/thirdparty/model.go`
- Modify: `internal/thirdparty/service.go`
- Modify: `internal/thirdparty/memory.go`
- Modify: `internal/thirdparty/postgres.go`
- Test: `internal/thirdparty/service_test.go`
- Test: `internal/thirdparty/postgres_integration_test.go`
- Test: `internal/httpapi/third_party_handlers_test.go`

- [ ] **Step 1: Write failing service and API tests**

Cover create and identity update with:

```go
input := thirdparty.CreateRelationshipInput{
	LegalName: "Example Bank", ServiceName: "Settlement banking",
	RegisteredAddress: "1 Marina Road\nLagos, Nigeria",
	WebsiteDomain: "https://www.example.com/about",
	Criticality: thirdparty.CriticalityImportant, PrivacyRole: thirdparty.PrivacyProcessor,
}
```

Assert the stored domain is `www.example.com`, the address is preserved with normalized line endings, and a brand job is committed. Add invalid `http`, credentials, IP literal, explicit port, malformed URL, and address-over-2000-character cases. Assert forged identity fields remain ignored at the HTTP boundary.

- [ ] **Step 2: Run tests and verify failure**

Run: `go test ./internal/thirdparty ./internal/httpapi -run 'Test.*(RegisteredAddress|WebsiteURL)' -count=1`

Expected: FAIL because `RegisteredAddress` is undefined and HTTPS input is rejected.

- [ ] **Step 3: Add the additive migration and model fields**

Migration:

```sql
ALTER TABLE third_party_vendors ADD COLUMN registered_address text;
ALTER TABLE third_party_vendors ADD CONSTRAINT third_party_vendors_registered_address_length
  CHECK (registered_address IS NULL OR char_length(registered_address) <= 2000);
```

Add `RegisteredAddress string \`json:"registered_address,omitempty"\`` to `Vendor`, `CreateRelationshipInput`, and `UpdateVendorIdentityInput`. Normalize CRLF to LF, trim blank outer lines, and reject values over 2000 characters.

Update PostgreSQL scans, inserts, updates, material events/outbox payloads, memory cloning and point-in-time reconstruction. Address and brand job remain in the same repository transaction.

- [ ] **Step 4: Accept a hostname or safe HTTPS URL**

Keep `WebsiteDomain` as the stored type. Before calling `NormalizeWebsiteDomain`, parse values containing `://`; accept only `https`, no user info, no explicit port, and a non-empty hostname. Discard path, query and fragment after validation, then run the existing hostname/IP protections.

- [ ] **Step 5: Run unit, schema and PostgreSQL-tag compile tests**

Run: `go test ./internal/thirdparty ./internal/httpapi -count=1`

Run: `go test -tags postgres ./internal/thirdparty ./internal/httpapi -run '^$'`

Expected: PASS; integration tests that require `TEST_DATABASE_URL` may report an explicit skip locally.

- [ ] **Step 6: Commit**

```powershell
git add migrations/000051_vendor_registered_address.* internal/thirdparty internal/httpapi/third_party_handlers_test.go
git commit -m "feat: capture vendor website and registered address"
```

## Task 4: Expose vendor identity fields and brand state in the web journey

**Files:**
- Modify: `web/src/vendorTypes.ts`
- Modify: `web/src/vendorApi.ts`
- Modify: `web/src/vendorIdentity.ts`
- Modify: `web/src/vendorIdentity.test.ts`
- Modify: `web/src/vendorApi.test.ts`
- Modify: `web/src/components/VendorsWorkspace.tsx`
- Modify: `web/src/components/VendorsWorkspace.test.tsx`
- Modify: `web/src/vendors.css`

- [ ] **Step 1: Write failing browser tests**

Assert that Add vendor exposes `Website` with `inputMode="url"` and `Registered address`, posts normalized `website_domain` plus `registered_address`, opens the created relationship, and renders the stored address and textual brand state. Assert invalid HTTP/IP/credentials stay inline without losing form values.

Update the API test expectation to:

```ts
expect(JSON.parse(String(init.body))).toEqual({
  legal_name: "Acme",
  website_domain: "acme.example",
  registered_address: "1 Marina Road\nLagos",
  service_name: "Payments",
  criticality: "IMPORTANT",
  privacy_role: "PROCESSOR",
});
```

- [ ] **Step 2: Run the focused tests and observe failure**

Run: `npm test -- --run vendorIdentity.test.ts vendorApi.test.ts components/VendorsWorkspace.test.tsx`

Working directory: `web`

Expected: FAIL because the create contract and form do not expose the new fields.

- [ ] **Step 3: Implement types, normalization and UI**

Add `registered_address?: string` to `Vendor` and create/update inputs. Extend `FormValues` with `website` and `registeredAddress`. `normalizeWebsiteDomain` accepts a safe HTTPS URL and returns its ASCII hostname. Render:

```tsx
<Field label="Website" error={errors.website}>
  <input id="vendor-website" type="url" inputMode="url" autoComplete="url"
    value={form.website} onChange={(event) => onChange("website", event.target.value)} />
</Field>
<Field label="Registered address" wide error={errors.registeredAddress}>
  <textarea id="vendor-address" rows={3} maxLength={2000} autoComplete="street-address"
    value={form.registeredAddress} onChange={(event) => onChange("registeredAddress", event.target.value)} />
</Field>
```

Show `Checking website`, `Logo available`, or `Logo not available` as text near `VendorBrandIcon`; never use brand state as a due-diligence status.

- [ ] **Step 4: Run focused tests, typecheck and copy gate**

Run from `web`:

`npm test -- --run vendorIdentity.test.ts vendorApi.test.ts components/VendorsWorkspace.test.tsx copyQuality.test.ts`

`npm run typecheck`

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add web/src/vendor* web/src/components/VendorsWorkspace* web/src/vendors.css
git commit -m "feat: add complete vendor identity fields to onboarding"
```

## Task 5: Replace the no-form dead end with governed setup

**Files:**
- Create: `web/src/vendorDueDiligenceForm.ts`
- Create: `web/src/components/VendorFormReadiness.tsx`
- Create: `web/src/components/VendorFormReadiness.test.tsx`
- Modify: `web/src/staticDemo.ts`
- Modify: `web/src/components/VendorsWorkspace.tsx`
- Modify: `web/src/components/VendorDueDiligence.tsx`
- Modify: `web/src/components/VendorsWorkspace.test.tsx`
- Modify: `web/src/components/vendor-due-diligence.css`

- [ ] **Step 1: Write failing governed-recovery tests**

Mock `loadFormTemplates()` as empty and assert “Set up due-diligence form” opens a dialog. Mock `loadProgramSummaries`, select a Program, create the canonical draft, transition to `PENDING_APPROVAL`, and show “Waiting for an independent reviewer.” Rerender as a different authorized actor with the pending form, activate it, then assert `Start due diligence` becomes available. Add forbidden and conflict recovery cases.

- [ ] **Step 2: Run the focused tests**

Run from `web`:

`npm test -- --run components/VendorFormReadiness.test.tsx components/VendorsWorkspace.test.tsx`

Expected: FAIL because the setup component does not exist.

- [ ] **Step 3: Implement one canonical starter form**

Export `vendorDueDiligenceStarterForm: CreateFormTemplateInput` with the approved four sections and eight fields. Static demo spreads that input into its active fixture so demo and live setup cannot drift.

- [ ] **Step 4: Implement the focused lifecycle UI**

`VendorFormReadiness` uses `FocusedSheet`. It loads a bounded Program page, creates a draft using `createFormTemplate`, and calls `transitionFormTemplate` for submission or activation. It never sends an approver ID. Copy names the current state and recovery action. On activation, call `onReady(form)` so the vendor workspace updates without a full reload.

`VendorDueDiligence` receives `onSetUpForm` and renders it only when no current active form and no active assessment exists.

- [ ] **Step 5: Run focused tests, accessibility checks and typecheck**

Run from `web`:

`npm test -- --run components/VendorFormReadiness.test.tsx components/VendorsWorkspace.test.tsx components/FocusedSheet.test.tsx`

`npm run typecheck`

Expected: PASS.

- [ ] **Step 6: Commit**

```powershell
git add web/src/vendorDueDiligenceForm.ts web/src/staticDemo.ts web/src/components/VendorFormReadiness* web/src/components/VendorDueDiligence.tsx web/src/components/VendorsWorkspace.tsx web/src/components/vendor-due-diligence.css
git commit -m "feat: add governed due diligence form setup"
```

## Task 6: Extend bounded Program and Matter summary filters

**Files:**
- Modify: `internal/continuity/summaries.go`
- Modify: `internal/continuity/summaries_memory.go`
- Modify: `internal/continuity/summaries_postgres.go`
- Modify: `internal/continuity/summaries_test.go`
- Modify: `internal/continuity/state_truth_postgres_integration_test.go`
- Modify: `internal/httpapi/summary_handlers.go`
- Modify: `internal/httpapi/summary_handlers_test.go`
- Create: `migrations/000052_portfolio_summary_filters.up.sql`
- Create: `migrations/000052_portfolio_summary_filters.down.sql`

- [ ] **Step 1: Write failing service and HTTP filter tests**

Extend the query contract:

```go
type SummaryQuery struct {
	Search, Status, ProgramID, OverallState, Jurisdiction string
	MatterType, DueCondition, Cursor string
	Priority int
	AssignedToMe bool
	Limit int
}
```

Create mixed fixtures and assert Program status/state/jurisdiction/mine combinations, Matter status/type/priority/program/due/mine combinations, restricted-record exclusion, invalid enum errors, and stable next cursors. “Mine” must match the principal from `identity.Require(ctx)`, never a query-supplied ID.

- [ ] **Step 2: Run focused tests and observe failure**

Run: `go test ./internal/continuity ./internal/httpapi -run 'Test.*Summary.*Filter' -count=1`

Expected: FAIL because the typed filters are absent.

- [ ] **Step 3: Implement normalized service validation**

Normalize enums to upper case. Permit Program overall states already defined by `ProgramState`, Matter types already defined by `MatterType`, priorities 1–5, and due values `OVERDUE`, `DUE_7_DAYS`, `NO_DUE_DATE`. Invalid values return an error that the handler maps to HTTP 400 without running a broader query.

- [ ] **Step 4: Implement memory and PostgreSQL parity**

Apply filters before pagination. PostgreSQL predicates remain inside the existing scoped query. `AssignedToMe` compares `owner_principal_id` with the verified actor principal. Program state filters use the latest state snapshot. Matter due filters use the service/repository current UTC instant consistently: `< now`, `[now, now+7 days]`, or `IS NULL`. Retain the existing sort keys and cursor encodings.

Add only indexes justified by `EXPLAIN` for the final predicates; the migration down file removes exactly those indexes.

- [ ] **Step 5: Run unit, HTTP and PostgreSQL-tag tests**

Run: `go test ./internal/continuity ./internal/httpapi -count=1`

Run: `go test -tags postgres ./internal/continuity ./internal/httpapi -run '^$'`

Expected: PASS.

- [ ] **Step 6: Commit**

```powershell
git add internal/continuity/summaries* internal/httpapi/summary_handlers* migrations/000052_portfolio_summary_filters.*
git commit -m "feat: add bounded portfolio summary filters"
```

## Task 7: Add responsive, route-preserved portfolio filter UX

**Files:**
- Modify: `web/src/summaryTypes.ts`
- Modify: `web/src/api.ts`
- Test: `web/src/api.test.ts`
- Create: `web/src/workspaceFilters.ts`
- Create: `web/src/workspaceFilters.test.ts`
- Modify: `web/src/appRouting.ts`
- Modify: `web/src/appRouting.test.ts`
- Modify: `web/src/components/ProgramsWorkspace.tsx`
- Modify: `web/src/components/MattersWorkspace.tsx`
- Modify: `web/src/components/ExactTargetWorkspaces.test.tsx`
- Create: `web/src/components/PortfolioFilters.test.tsx`
- Modify: the stylesheet that owns `.workspace-toolbar`

- [ ] **Step 1: Write failing API, route and component tests**

Assert typed parameters serialize as `state`, `jurisdiction`, `type`, `priority`, `due`, `program_id`, and `mine=true`. Assert `parseRoute("#programs/program-1?q=privacy&status=ACTIVE")` retains `program-1`. With fake timers, type a query and assert no request before 300 ms, then one request after; Enter applies immediately. Assert chips remove one filter, Clear filters removes all, and record back-navigation restores the list query.

- [ ] **Step 2: Run focused tests and observe failure**

Run from `web`:

`npm test -- --run api.test.ts appRouting.test.ts workspaceFilters.test.ts components/PortfolioFilters.test.tsx`

Expected: FAIL because typed filters and route helpers are absent.

- [ ] **Step 3: Implement typed query serialization and route helpers**

Extend `SummaryQuery` with `state`, `jurisdiction`, `matterType`, `priority`, `due`, and `mine`. `workspaceFilters.ts` reads the hash query into a validated filter object and serializes only non-default values with `URLSearchParams`. Update `parseRoute` to split the hash path at `?` before splitting route segments.

- [ ] **Step 4: Implement Program and Matter controls**

Use one 300 ms effect for search draft, immediate select/checkbox changes, removable chips, and a labelled mobile “Filters” button opening `FocusedSheet`. Keep common desktop filters visible. Every filter change clears items/cursor and calls the bounded summary API. Empty and summary copy always says “loaded” unless a server total exists.

When opening a record, append the current list query to its hash. Back actions reconstruct `#programs?<query>` or `#work/matters?<query>`.

- [ ] **Step 5: Run focused tests, copy gate and typecheck**

Run from `web`:

`npm test -- --run api.test.ts appRouting.test.ts workspaceFilters.test.ts components/PortfolioFilters.test.tsx components/ExactTargetWorkspaces.test.tsx copyQuality.test.ts`

`npm run typecheck`

Expected: PASS.

- [ ] **Step 6: Commit**

```powershell
git add web/src/api.ts web/src/summaryTypes.ts web/src/workspaceFilters* web/src/appRouting* web/src/components/ProgramsWorkspace.tsx web/src/components/MattersWorkspace.tsx web/src/components/*PortfolioFilters* web/src/components/ExactTargetWorkspaces.test.tsx web/src/*.css
git commit -m "feat: add scalable portfolio filter experience"
```

## Task 8: Move vendor linking into a focused searchable sheet

**Files:**
- Modify: `web/src/components/VendorRelationshipLinks.tsx`
- Modify: `web/src/components/VendorRelationshipLinks.test.tsx`
- Modify: `web/src/vendor-relationship-links.css`

- [ ] **Step 1: Write failing focused-link tests**

Assert “Link vendor” opens a named dialog, focuses Close, debounces vendor search, renders `VendorBrandIcon`, legal name, service, criticality and status, keeps purpose selection, disables submit until valid, closes after success, and restores focus to the trigger. Cover Escape, duplicate conflict, search failure and narrow layout semantics.

- [ ] **Step 2: Run the focused tests and observe failure**

Run from `web`:

`npm test -- --run components/VendorRelationshipLinks.test.tsx components/FocusedSheet.test.tsx`

Expected: FAIL because linking is an inline form without branded contextual rows or debounced search.

- [ ] **Step 3: Implement `FocusedSheet` composition**

Replace the conditional inline `<form>` with:

```tsx
{linking && <FocusedSheet label={`Link vendor to this ${targetName}`} onClose={closeLinking} panelClassName="vendor-link-sheet">
  <form className="vendor-link-form" onSubmit={submit}>
    {/* search, selectable contextual rows, purpose and one primary action */}
  </form>
</FocusedSheet>}
```

Use `loadVendorRelationships({search, limit: 20})` after 300 ms and render the existing same-origin `VendorBrandIcon`. Preserve all existing link/end-link behavior outside the sheet. The backdrop uses the established panel blur token and an opaque fallback.

- [ ] **Step 4: Run tests, typecheck and accessibility regression**

Run from `web`:

`npm test -- --run components/VendorRelationshipLinks.test.tsx components/FocusedSheet.test.tsx`

`npm run typecheck`

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add web/src/components/VendorRelationshipLinks* web/src/vendor-relationship-links.css
git commit -m "feat: focus and simplify vendor linking"
```

## Task 9: Synchronize product documentation and acceptance coverage

**Files:**
- Modify: `DESIGN.md`
- Modify: `docs/implementation-plan.md`
- Modify: `docs/quality/acceptance-tests.md`
- Modify: `docs/design/programs-matters-decision-brief.md`

- [ ] **Step 1: Update maturity and interaction documentation**

Record that vendor onboarding now captures website/address, reference form readiness is installed through maker-checker, real tenants have governed setup, portfolio filters are bounded and route-preserved, and vendor linking uses a focused accessible sheet. Keep third-party lifecycle maturity truthful; this work does not claim contract exit or verified outcome completeness.

- [ ] **Step 2: Add exact acceptance assertions**

Extend Golden Journeys M and N plus portfolio coverage with checkboxes for:

```markdown
- [ ] a new vendor can start due diligence without API or database intervention;
- [ ] missing production configuration leads to governed draft and approval, not silent activation;
- [ ] website discovery failure leaves vendor creation and due diligence usable;
- [ ] combined filters remain tenant-, legal-entity- and visibility-scoped across keyset pages;
- [ ] link-vendor focus, keyboard and error recovery work at desktop, mobile and 200% zoom.
```

- [ ] **Step 3: Run documentation and copy checks**

Run: `git diff --check`

Run from `web`: `npm test -- --run copyQuality.test.ts`

Expected: PASS.

- [ ] **Step 4: Commit**

```powershell
git add DESIGN.md docs/implementation-plan.md docs/quality/acceptance-tests.md docs/design/programs-matters-decision-brief.md
git commit -m "docs: complete vendor operations acceptance coverage"
```

## Task 10: Verify, render, deploy and inspect the hosted journey

**Files:**
- Update rendered evidence and fixtures only where the existing review command produces tracked artifacts.

- [ ] **Step 1: Run the complete non-Docker verification suite**

Run:

```powershell
go test ./...
go test -tags postgres ./... -run '^$'
Push-Location web
npm test -- --run
npm run typecheck
npm run build
npm run review:ui
Pop-Location
git diff --check
git status --short
```

Expected: all runnable tests and builds pass; database-dependent tests explicitly skip only when `TEST_DATABASE_URL` is absent.

- [ ] **Step 2: Inspect required rendered states**

Render before/after or current after-states for Add vendor, missing-form setup, Programs filters, Issues and Changes filters, and Link vendor at desktop and mobile widths in light and dark themes. Inspect 200% zoom, keyboard focus, reduced motion, and backdrop-filter fallback. Fix the highest-impact defect and rerun the affected render and test.

- [ ] **Step 3: Verify branch integrity and merge state**

Confirm all task commits are on `main`, the worktree is clean, and no unrelated user changes were overwritten.

- [ ] **Step 4: Push `main` and wait for CI deployment**

Use the repository’s existing authenticated push and CI workflow. Do not install or start local Docker. Wait for the exact pushed SHA’s tests and deployment to finish.

- [ ] **Step 5: Verify `clearsight.cloudspacetechs.com` end to end**

Using the hosted signed-in reference role:

1. Add a vendor with HTTPS website and registered address.
2. Confirm the saved identity and queued/available/unavailable textual logo state.
3. Start due diligence without the old missing-form message.
4. Confirm maker-checker setup remains available in a no-form test tenant or fixture.
5. Search/filter Programs and Issues and Changes with combined filters and return from a record without losing them.
6. Link the vendor to a Program and an Issue or Change through the focused sheet.
7. Confirm no remote logo URL, token, actor ID or protected record leaks through markup or requests.

- [ ] **Step 6: Record final evidence**

Report the exact deployed SHA, CI result, hosted checks performed, any environment-dependent limitation, and remaining third-party lifecycle maturity without overstating completion.
