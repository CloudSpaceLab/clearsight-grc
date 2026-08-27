# Governed Form System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. The user selected inline execution; do not dispatch subagents.

**Goal:** Deliver reusable governed form templates, scalable distributions, resilient vendor response capture, document/AI-assisted authoring, and focused vendor-record refresh without duplicating ClearSight's existing form, evidence, import, authority, or third-party stores.

**Architecture:** Promote `monitoring_form_templates` through legal-entity Forms APIs while retaining its IDs and Program compatibility. A distribution composes one canonical `capture_request` for each To recipient, one shared versioned response workspace, protected access routes and immutable `capture_submissions`; document and AI outputs remain reviewable proposals until a maker accepts them into a draft revision. Vendor corrections and document replacements are applied only by current authority-checked third-party commands in the same transaction as their receipts, events and outbox messages.

**Tech Stack:** Go 1.25 modular monolith, PostgreSQL 18/pgx, existing runtime outbox and authority guards, React 19/Vite/TypeScript, IndexedDB/Web Crypto, constrained Lexical editor loaded only in Forms configuration, bounded Poppler/OOXML extraction, optional licensed parser adapters.

---

## Delivery tranches

1. **Forms foundation:** legal-entity template library, scoring modes, record intent, maker-checker revisions, direct Forms navigation and full builder parity.
2. **Distribution and response:** multi-recipient composition, three external-access policies, OTP, protected delivery, shared server drafts, immutable amendments and encrypted browser recovery.
3. **Document and AI authoring:** explicit extraction outcomes, bounded DOCX form controls, form-template proposals, governed AI diffs and optional parser gates.
4. **Vendor refresh and application:** frozen held-value baselines, focused reassessments, governed identity application, document supersession and deterministic expiry attention.

Each tranche must be deployable and reversible on its own. Compatibility routes and legacy request behavior remain active until the final acceptance task proves migration and rollback.

## File map

### Forms foundation

- Modify `internal/formcontract/model.go`, `validation.go`, `scoring.go` for scoring mode, section weights, collection intent, target key and browser-cache policy.
- Create `internal/formcontract/compliance_test.go` for exact 100% and applicable-denominator behavior.
- Create `migrations/000053_legal_entity_forms.up.sql` and `.down.sql` for optional Program ownership and searchable template metadata.
- Create `internal/monitoring/form_library.go`, `form_library_test.go`, `form_library_postgres_test.go` for legal-entity pages and revision creation.
- Create a versioned, clearly labelled vendor due-diligence starter fixture and an instantiation service that produces an ordinary governed draft.
- Modify `internal/monitoring/model.go`, `repository.go`, `memory.go`, `postgres.go`, `service.go` only where existing form persistence must expose the promoted semantics.
- Create `internal/httpapi/forms_handlers.go`, `forms_handlers_test.go`; modify `route_registry.go`, `server.go`, API composition and `api/runtime.openapi.json`.
- Create `web/src/formsTypes.ts`, `formsApi.ts`, `components/FormsWorkspace.tsx`, `components/FormsWorkspace.test.tsx`, `forms.css`.
- Modify `web/src/appRouting.ts`, `App.tsx`, `components/NavigationIcon.tsx`, `components/FormBuilder.tsx`, `monitoringTypes.ts`, `monitoringApi.ts` and affected tests.

### Distribution, protected delivery and recovery

- Create `migrations/000054_form_distributions.up.sql` and `.down.sql` for distributions, recipients, encrypted addresses, routes, OTP challenges, shared workspaces and response revisions.
- Create `migrations/000055_form_communications.up.sql` and `.down.sql` for versioned communication/branding configuration and delivery attempts.
- Create focused evidence files: `distribution.go`, `distribution_store.go`, `distribution_memory.go`, `distribution_postgres.go`, `access_policy.go`, `otp.go`, `protected_recipient.go`, `response_workspace.go`, `communications.go`, `smtp_delivery.go` and matching tests.
- Modify legacy `internal/evidence/model.go`, `service.go`, `draft.go`, PostgreSQL repository files and invitation administration only for compatibility adapters.
- Create `internal/httpapi/form_distribution_handlers.go`, `form_distribution_handlers_test.go`, `form_communication_handlers.go`, `form_communication_handlers_test.go`; modify route registry and executable API contract.
- Create web distribution, communication and recovery modules under `web/src/components/forms/`, `web/src/captureRecovery.ts`, `captureRecovery.test.ts`, `captureRecoveryCrypto.ts`, `captureRecoveryStore.ts`.
- Modify `ExternalCaptureApp.tsx`, shared capture components, `captureApi.ts`, `captureInvitationBrowser.ts`, `types.ts`, `capture-inputs.css` and static-demo fixtures/tests.

### Document and AI authoring

- Create `migrations/000056_form_template_proposals.up.sql` and `.down.sql`.
- Create `internal/documentimport/elements.go`, `docx_form_controls.go`, `form_template_proposal.go`, `parser_adapter.go` and golden-corpus tests.
- Modify `internal/documentimport/model.go`, `extractor.go`, `extraction_policy.go`, service/repository/PostgreSQL files and worker composition.
- Create `internal/monitoring/form_proposal.go`, `form_proposal_store.go`, memory/PostgreSQL implementations and tests.
- Create `internal/monitoring/form_ai.go`, `form_ai_client.go` and tests; route calls only through a configured governed AI-gateway workload.
- Extend Imports and Forms UI with source-anchored proposal review and exact-version AI diffs.

### Vendor refresh and application

- Create `migrations/000057_vendor_form_refresh.up.sql` and `.down.sql`.
- Create `internal/thirdparty/assessment_scope.go`, `record_target.go`, `assessment_application.go`, `document_supersession.go`, `refresh_maintenance.go` and matching memory/PostgreSQL tests.
- Modify assessment model/request/review/repositories, vendor identity transaction helpers, worker composition and vendor assessment HTTP handlers.
- Modify vendor assessment web types/API/UI for scoped refresh, baseline comparison, application conflicts and supersession.
- Synchronize `README.md`, `DESIGN.md`, architecture/product/acceptance docs and rendered UI evidence.

## Tranche 1 — Forms foundation

### Task 1: Extend the shared form contract test-first

**Files:**
- Modify: `internal/formcontract/model.go`
- Modify: `internal/formcontract/validation.go`
- Modify: `internal/formcontract/scoring.go`
- Create: `internal/formcontract/compliance_test.go`
- Test: `internal/formcontract/validation_test.go`

- [ ] **Step 1: Add failing compatibility and compliance tests**

```go
func TestNormalizeComplianceRequiresExactWeights(t *testing.T) {
	contract := Contract{
		ScoringMode: ScoringCompliance,
		Sections: []Section{{ID: "identity", Title: "Identity", Weight: 100}},
		Fields: []Field{{ID: "registered", SectionID: "identity", Label: "Registration verified", Type: TypeYesNo, Required: true, Options: []string{"Yes", "No"}, Scoring: &Scoring{Weight: 100, AnswerScores: map[string]int{"Yes": 100, "No": 0}}}},
	}
	if _, err := Normalize(contract); err != nil { t.Fatalf("normalize compliance form: %v", err) }
	contract.Fields[0].Scoring.Weight = 90
	if _, err := Normalize(contract); !errors.Is(err, ErrInvalid) { t.Fatalf("expected invalid weight total, got %v", err) }
}

func TestNormalizeKeepsExistingRiskWeightsBackwardCompatible(t *testing.T) {
	contract := Contract{Sections: []Section{{ID: "general", Title: "General"}}, Fields: []Field{
		{ID: "risk", SectionID: "general", Label: "Risk", Type: TypeYesNo, Options: []string{"Yes", "No"}, Scoring: &Scoring{Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}}},
	}}
	normalized, err := Normalize(contract)
	if err != nil || normalized.ScoringMode != ScoringRisk { t.Fatalf("risk compatibility failed: %#v %v", normalized, err) }
}
```

- [ ] **Step 2: Run the focused test and confirm it fails before implementation**

Run: `go test ./internal/formcontract -run 'TestNormalizeCompliance|TestNormalizeKeepsExistingRisk'`

Expected: FAIL because `ScoringMode`, `ScoringCompliance`, `Section.Weight` and compliance-total validation do not exist.

- [ ] **Step 3: Add the versioned contract types**

```go
type ScoringMode string
const (
	ScoringNone       ScoringMode = "NONE"
	ScoringRisk       ScoringMode = "RISK"
	ScoringCompliance ScoringMode = "COMPLIANCE"
)

type CollectionIntent string
const (
	IntentCapture             CollectionIntent = "CAPTURE"
	IntentConfirmOrCorrect    CollectionIntent = "CONFIRM_OR_CORRECT"
	IntentReplaceHeldDocument CollectionIntent = "REPLACE_HELD_DOCUMENT"
)

type BrowserCachePolicy string
const (
	BrowserCacheAllowed BrowserCachePolicy = "ALLOWED"
	BrowserCacheDenied  BrowserCachePolicy = "NO_BROWSER_CACHE"
)

type RecordTarget struct {
	Key                 string `json:"key"`
	RequiredSubjectType string `json:"required_subject_type"`
}
```

Add `Weight int` to `Section`; add `CollectionIntent`, `RecordTarget *RecordTarget` and `BrowserCachePolicy` to `Field`; add `ScoringMode` to `Contract`. Normalize an omitted scoring mode to `RISK` only when a field has scoring and to `NONE` otherwise. In compliance mode require scored section weights to total 100 and scored field weights within each scored section to total 100. Preserve existing risk scoring semantics unchanged.

- [ ] **Step 4: Add compliance calculation with explicit coverage and finality**

```go
type ComplianceResult struct {
	Score         *float64        `json:"score,omitempty"`
	Coverage      float64         `json:"coverage"`
	Final         bool            `json:"final"`
	Denominator   int             `json:"denominator"`
	CriticalRules []ScoreRuleResult `json:"critical_rules,omitempty"`
	RuleResults   []ScoreRuleResult `json:"rule_results"`
}

func ScoreCompliance(contract Contract, answers map[string]AnswerValue) (ComplianceResult, error)
```

Use the existing `VisibleFields(contract, answers)` for the applicable population and build the ID set from its result. Unanswered applicable scored fields contribute zero achievement and incomplete coverage; hidden/not-applicable fields leave the denominator and remaining weights are normalized within their section. Do not reuse `monitoring.bandFor`, because compliance achievement and risk points have opposite direction.

- [ ] **Step 5: Run shared form tests**

Run: `go test ./internal/formcontract ./internal/monitoring ./internal/evidence`

Expected: PASS, including existing risk fixtures and new exact-100 compliance fixtures.

- [ ] **Step 6: Commit**

```bash
git add internal/formcontract internal/monitoring internal/evidence
git commit -m "feat: extend governed form contract"
```

### Task 2: Promote existing templates into a legal-entity library

**Files:**
- Create: `migrations/000053_legal_entity_forms.up.sql`
- Create: `migrations/000053_legal_entity_forms.down.sql`
- Create: `internal/monitoring/form_library.go`
- Create: `internal/monitoring/form_library_test.go`
- Create: `internal/monitoring/form_library_postgres_test.go`
- Create: `internal/monitoring/starter_templates.go`
- Create: `internal/monitoring/starter_templates_test.go`
- Create: `internal/monitoring/starter_templates/vendor_due_diligence_v1.json`
- Modify: `internal/monitoring/model.go`
- Modify: `internal/monitoring/repository.go`
- Modify: `internal/monitoring/memory.go`
- Modify: `internal/monitoring/postgres.go`
- Test: `internal/monitoring/schema_test.go`

- [ ] **Step 1: Write failing repository tests for optional Program scope and keyset pagination**

```go
func TestFormLibraryListsOneCurrentRevisionPerTemplate(t *testing.T) {
	page, err := repo.ListFormLibrary(t.Context(), FormLibraryFilter{TenantID: "bank-a", LegalEntityID: "entity-a", Search: "vendor", Limit: 2})
	if err != nil { t.Fatal(err) }
	if len(page.Items) != 2 || page.Items[0].Version < page.Items[1].Version { t.Fatalf("unexpected page: %#v", page) }
	if page.NextCursor == "" { t.Fatal("expected keyset cursor") }
}
```

The PostgreSQL fixture must include one Program-owned form, one legal-entity-only form, an older revision, another entity's form, a retired form and a historical unscoped form. Assert entity filtering happens in SQL before `LIMIT`, unscoped rows stay out of canonical Forms pages, existing risk/non-scored rows receive the correct scoring mode, and the unsafe down migration preflight preserves all rows.

- [ ] **Step 2: Run the tests and confirm the library methods are absent**

Run: `go test ./internal/monitoring -run 'TestFormLibrary|TestMonitoringSchema'`

Expected: FAIL with undefined `FormLibraryFilter` and missing migration assertions.

- [ ] **Step 3: Add the compatibility-safe schema migration**

```sql
ALTER TABLE monitoring_form_templates
  DROP CONSTRAINT monitoring_form_templates_scope_pair_ck,
  ADD COLUMN owner_principal_id uuid,
  ADD COLUMN responsible_team text NOT NULL DEFAULT '',
  ADD COLUMN approved_uses text[] NOT NULL DEFAULT '{}',
  ADD COLUMN tags text[] NOT NULL DEFAULT '{}',
  ADD COLUMN jurisdiction text NOT NULL DEFAULT '',
  ADD COLUMN industry text NOT NULL DEFAULT '',
  ADD COLUMN sensitivity text NOT NULL DEFAULT 'INTERNAL',
  ADD COLUMN scoring_mode text,
  ADD COLUMN next_review_at timestamptz,
  ADD COLUMN starter_catalog_code text,
  ADD COLUMN starter_catalog_version bigint,
  ADD CONSTRAINT monitoring_form_templates_entity_scope_ck CHECK (program_id IS NULL OR legal_entity_id IS NOT NULL),
  ADD CONSTRAINT monitoring_form_templates_scoring_mode_ck CHECK (scoring_mode IN ('NONE','RISK','COMPLIANCE')),
  ADD CONSTRAINT monitoring_form_templates_starter_pair_ck CHECK (
    (starter_catalog_code IS NULL AND starter_catalog_version IS NULL)
    OR (starter_catalog_code=btrim(starter_catalog_code) AND char_length(starter_catalog_code) BETWEEN 1 AND 128 AND starter_catalog_version > 0)
  ),
  ADD FOREIGN KEY (owner_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
  ADD FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id);

UPDATE monitoring_form_templates f
SET scoring_mode = CASE
  WHEN EXISTS (SELECT 1 FROM jsonb_array_elements(f.fields) field WHERE field ? 'scoring') THEN 'RISK'
  ELSE 'NONE'
END;
ALTER TABLE monitoring_form_templates ALTER COLUMN scoring_mode SET NOT NULL;

DROP INDEX monitoring_form_templates_legacy_current_code_idx;
CREATE UNIQUE INDEX monitoring_form_templates_entity_current_code_idx
  ON monitoring_form_templates(tenant_id,legal_entity_id,code)
  WHERE is_current AND program_id IS NULL AND legal_entity_id IS NOT NULL;
CREATE UNIQUE INDEX monitoring_form_templates_unscoped_current_code_idx
  ON monitoring_form_templates(tenant_id,code)
  WHERE is_current AND program_id IS NULL AND legal_entity_id IS NULL;

CREATE INDEX monitoring_form_templates_library_idx
  ON monitoring_form_templates(tenant_id,legal_entity_id,updated_at DESC,id DESC,version DESC)
  WHERE legal_entity_id IS NOT NULL;
CREATE INDEX monitoring_form_templates_library_search_idx
  ON monitoring_form_templates(tenant_id,legal_entity_id,lower(name),updated_at DESC,id DESC)
  WHERE legal_entity_id IS NOT NULL;

CREATE TABLE form_saved_views (
  id uuid PRIMARY KEY DEFAULT uuidv7(), tenant_id uuid NOT NULL REFERENCES tenants(id), legal_entity_id uuid NOT NULL,
  principal_id uuid NOT NULL, name text NOT NULL, filter jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  UNIQUE (tenant_id,legal_entity_id,principal_id,name),
  FOREIGN KEY (principal_id,tenant_id) REFERENCES principals(id,tenant_id),
  FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
  CHECK (jsonb_typeof(filter)='object' AND octet_length(filter::text) <= 8192)
);
```

Historical forms with both scope columns null remain readable only through compatibility routes; every canonical Forms create/revise/use command requires verified legal-entity scope. The down migration performs a transactional preflight and fails closed if legal-entity-only templates or saved views exist. When safe, it removes only the added rows/indexes/columns/constraints, restores the prior indexes and nullable-pair constraint, and never deletes form records or rewrites IDs. Operational rollback disables the new Forms routes first, so existing Program and legacy routes remain available even when schema rollback is unsafe after adoption.

- [ ] **Step 4: Add library types and repository methods**

```go
type FormLibraryFilter struct {
	TenantID, LegalEntityID, Search, ProgramID, OwnerPrincipalID, Use, Tag, Status, Cursor string
	Limit int
}
type FormLibraryItem struct { Template FormTemplate `json:"template"`; ActiveVersion int64 `json:"active_version,omitempty"`; ActiveStatus LifecycleStatus `json:"active_status,omitempty"` }
type FormTemplatePage struct { Items []FormLibraryItem `json:"items"`; NextCursor string `json:"next_cursor,omitempty"` }
type SavedFormView struct { ID, Name string; Filter FormLibraryFilter; CreatedAt, UpdatedAt time.Time }

type FormLibraryRepository interface {
	ListFormLibrary(context.Context, FormLibraryFilter) (FormTemplatePage, error)
	ReusableFormRevision(context.Context, string, string, string, int64) (FormTemplate, error)
	ListSavedFormViews(context.Context, string, string, string) ([]SavedFormView, error)
	SaveFormView(context.Context, string, string, string, SavedFormView) (SavedFormView, error)
	DeleteSavedFormView(context.Context, string, string, string, string) error
}
```

Select the greatest stored version per stable template ID, then attach the independently current active/paused version. This keeps a newer draft visible without mislabelling it active. Use `(updated_at,id) < ($cursor_time,$cursor_id)` and fetch `limit+1`. Existing `ListReusableFormRevisions` remains a compatibility read of exact active/current revisions.

- [ ] **Step 5: Add a governed starter-template catalog**

Add a bounded embedded starter catalog. `vendor_due_diligence_v1.json` is jurisdiction-neutral product reference data with its catalog version, publication date and **Review this draft against the bank's policy before approval** label. Instantiation copies it into a normal legal-entity `DRAFT`, records starter ID/version provenance and never activates or sends it. Any future jurisdiction-specific starter must record official source/date and a not-legal-advice label.

- [ ] **Step 6: Run migration, repository and starter-catalog tests**

Run: `go test ./internal/monitoring`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add migrations/000053_legal_entity_forms.* internal/monitoring
git commit -m "feat: promote reusable form library"
```

### Task 3: Add authority-checked Forms commands and HTTP contracts

**Files:**
- Modify: `internal/monitoring/service.go`
- Create: `internal/monitoring/form_library_service_test.go`
- Create: `internal/httpapi/forms_handlers.go`
- Create: `internal/httpapi/forms_handlers_test.go`
- Modify: `internal/httpapi/route_registry.go`
- Modify: `internal/httpapi/server.go`
- Modify: `cmd/api/services.go`
- Modify: `cmd/api/services_memory.go`
- Modify: `cmd/api/services_postgres.go`
- Modify: `cmd/api/main.go`
- Modify: `api/runtime.openapi.json`

- [ ] **Step 1: Add failing identity, authority and lifecycle tests**

Test that create uses the verified legal entity and principal even when the body supplies another scope; revision creation pins the expected active/draft version; maker cannot approve their own revision; tenant/entity mismatch and authority failure return no data; retiring blocks new use without invalidating existing exact revisions.

```go
func TestCreateLibraryFormUsesVerifiedIdentity(t *testing.T) {
	ctx := identity.WithActor(t.Context(), identity.Actor{
		TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "maker-a",
		Kind: "PERSON", AuthenticationMethod: "TEST", AssuranceLevel: "HIGH",
		SessionID: "session-a", ExpiresAt: time.Now().Add(time.Hour),
	})
	created, err := service.CreateLibraryForm(ctx, CreateFormInput{LegalEntityID: "entity-b", Name: "Vendor review", Code: "VENDOR", Purpose: "Collect current vendor evidence.", Fields: validFields()})
	if err != nil { t.Fatal(err) }
	if created.LegalEntityID != "entity-a" || created.CreatedBy != "maker-a" { t.Fatalf("unverified scope used: %#v", created) }
}
```

- [ ] **Step 2: Run focused service/HTTP tests and confirm failure**

Run: `go test ./internal/monitoring ./internal/httpapi -run 'FormLibrary|LibraryForm|FormsRoute'`

Expected: FAIL because legal-entity Forms commands and routes do not exist.

- [ ] **Step 3: Implement service commands using current command identity**

Add `CreateLibraryForm`, `CreateFormRevision`, `GetLibraryForm`, `ListFormLibrary`, `ListStarterTemplates`, `InstantiateStarterTemplate`, `ListSavedFormViews`, `SaveFormView` and `DeleteSavedFormView`, and reuse `TransitionForm`. Resolve identity with `identity.Require(ctx)` inside material commands, use `commandauth.Guard` for `LEGAL_ENTITY` creation/starter instantiation and `FORM_TEMPLATE` revision/transition, and overwrite all actor/scope fields from the verified decision. Saved views are always principal-owned from verified context. A starter instantiation and a revision both create normal `DRAFT` records and never mutate or activate the current row.

- [ ] **Step 4: Register the canonical routes**

```text
GET  /api/v1/forms/templates
POST /api/v1/forms/templates
GET  /api/v1/forms/templates/{id}/revisions/{version}
POST /api/v1/forms/templates/{id}/revisions
POST /api/v1/forms/templates/{id}/transition
GET  /api/v1/forms/starter-templates
POST /api/v1/forms/starter-templates/{code}/instantiate
GET  /api/v1/forms/saved-views
POST /api/v1/forms/saved-views
DELETE /api/v1/forms/saved-views/{id}
```

List/detail are actor-scoped reads. Saved-view filters accept only the canonical bounded query keys and never store tenant/entity/principal overrides. Create/revise/transition are material commands with `noActorField`; the handler accepts no actor, approver, tenant or legal-entity authority field. Preserve `/api/v1/form-templates` and Program routes as projections/adapters during migration.

- [ ] **Step 5: Update executable route contract and run route guards**

Run: `go test ./internal/httpapi -run 'Forms|RouteRegistry|RuntimeContract|CommandGuard'`

Expected: PASS with all ten routes classified and unauthorized/tampered requests failing closed.

- [ ] **Step 6: Commit**

```bash
git add internal/monitoring internal/httpapi cmd/api api/runtime.openapi.json
git commit -m "feat: expose governed Forms API"
```

### Task 4: Build the direct Forms workspace and searchable library

**Files:**
- Create: `web/src/formsTypes.ts`
- Create: `web/src/formsApi.ts`
- Create: `web/src/formsApi.test.ts`
- Create: `web/src/components/FormsWorkspace.tsx`
- Create: `web/src/components/FormsWorkspace.test.tsx`
- Create: `web/src/forms.css`
- Modify: `web/src/appRouting.ts`
- Modify: `web/src/appRouting.test.ts`
- Modify: `web/src/App.tsx`
- Modify: `web/src/components/NavigationIcon.tsx`
- Modify: `web/src/Accessibility.test.tsx`

- [ ] **Step 1: Write failing route, API and state tests**

```tsx
it("keeps the Forms search and selected template in the URL", async () => {
  window.location.hash = "#forms/template-a?search=vendor&status=ACTIVE";
  const route = parseRoute(window.location.hash);
  expect(route).toMatchObject({ view: "forms", target: { formTemplateID: "template-a" } });
});

it("states the bounded population when no template matches", async () => {
  render(<FormsWorkspace initialSearch="outsourcing" />);
  expect(await screen.findByText("No form templates match ‘outsourcing’ in this legal entity.")) .toBeVisible();
  expect(screen.getByRole("button", { name: "Create form template" })).toBeEnabled();
  expect(screen.getByRole("button", { name: "Use a starter template" })).toBeEnabled();
});
```

- [ ] **Step 2: Run the tests and confirm the Forms view is absent**

Run: `cd web && npm test -- appRouting.test.ts formsApi.test.ts FormsWorkspace.test.tsx`

Expected: FAIL because `forms` is not a `View` and the workspace files do not exist.

- [ ] **Step 3: Add route/API types with server-side keyset filters**

```ts
export type FormTemplateQuery = {
  search?: string; status?: LifecycleStatus; owner?: string; program?: string;
  use?: string; tag?: string; cursor?: string; limit?: number;
};
export type FormTemplatePage = { items: FormTemplate[]; next_cursor?: string };
```

Serialize only defined values. Use the existing scoped `requestJSON`; do not load all templates and filter in memory.

- [ ] **Step 4: Add the `forms` navigation destination**

Extend `View`, `WorkspaceTarget`, `parseRoute` and `routeHash` with `formTemplateID`. Add a labelled document icon in `NavigationIcon`. Lazy-load `FormsWorkspace` in `App.tsx` and place Forms directly after Programs. Mobile behavior uses the existing labelled bottom navigation and must remain usable at 320 CSS pixels and 200% zoom.

- [ ] **Step 5: Implement Templates, Sent forms, Responses, Imports and Communications tabs**

Templates is functional in this task; the other tabs render honest bounded empty/degraded states and are activated by Tranches 2–3. The library provides delayed search, status/owner/Program/use/tag filters, user-owned saved views, a server-backed recently updated section, table/grid density, keyset **Load more**, exact revision state and one dominant **Create form template** action. The no-active-template journey can instantiate the labelled starter as an ordinary draft and then directs the maker to **Send for approval**; it never implies that the draft can be used before checker activation. Accessible bulk selection appears only for rows sharing the same permitted lifecycle command; mixed or unauthorized selections show why no bulk action is available.

- [ ] **Step 6: Run web checks**

Run: `cd web && npm test -- appRouting.test.ts formsApi.test.ts FormsWorkspace.test.tsx Accessibility.test.tsx && npm run typecheck`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add web/src
git commit -m "feat: add direct Forms workspace"
```

### Task 5: Complete builder parity and approval-quality checks

**Files:**
- Modify: `web/src/components/FormBuilder.tsx`
- Modify: `web/src/components/FormBuilder.test.tsx`
- Create: `web/src/components/forms/FormPropertyPanel.tsx`
- Create: `web/src/components/forms/FormQualityPanel.tsx`
- Create: `web/src/components/forms/FormPreview.tsx`
- Modify: `web/src/formsTypes.ts`
- Modify: `web/src/formsApi.ts`
- Modify: `web/src/monitoringTypes.ts`
- Modify: `web/src/monitoringApi.ts`
- Modify: `web/src/monitoring.css`
- Modify: `web/src/forms.css`

- [ ] **Step 1: Add failing builder/renderer conformance tests**

Cover all field types, date/calendar inputs, conditions, section duplication with stable regenerated keys, reusable section insertion from an exact active template revision, option paste/deduplication, compliance remaining weight, required sign-off insertion, file limits, responsive preview and the exact normalized payload shared with `CaptureForm`.

```tsx
it("blocks approval while compliance weight remains", async () => {
  render(<FormBuilder initialValue={complianceForm({ weight: 80 })} />);
  expect(screen.getByText("20% remains to allocate in Vendor identity")) .toBeVisible();
  expect(screen.getByRole("button", { name: "Send for approval" })).toBeDisabled();
});
```

- [ ] **Step 2: Run focused tests and confirm the quality controls fail**

Run: `cd web && npm test -- FormBuilder.test.tsx CaptureForm.test.tsx`

Expected: FAIL on missing weight, quality and preview controls.

- [ ] **Step 3: Implement focused authoring components**

Keep `FormBuilder` as orchestration and move field properties, quality results and capture preview into the three focused files. Recommendations are deterministic versioned constants returned by the server Forms endpoint; they include placeholders, common option lists, validation bounds and approved evidence media types. Implement **Insert section from template** by reading a selected exact active revision and copying its normalized section/fields with regenerated IDs and rewritten internal condition references; this provides reusable field groups without adding another schema/store. Client suggestions never become active semantics without a normal saved revision.

- [ ] **Step 4: Enforce accessible document editing presentation**

Use a white `.form-document-canvas` in light mode and the corresponding semantic dark surface in dark mode. Use labelled 16–20 px icons, native `date`/`datetime-local` controls, visible focus, keyboard reordering, reduced motion and subtle backdrop blur only for focused sheets/dialogs.

- [ ] **Step 5: Run builder, copy and production-build checks**

Run: `cd web && npm test -- FormBuilder.test.tsx CaptureForm.test.tsx copyQuality.test.ts && npm run build`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src
git commit -m "feat: complete governed form authoring"
```

## Tranche 2 — Distribution, protected delivery and response recovery

### Task 6: Add distribution, access and workspace persistence

**Files:**
- Create: `migrations/000054_form_distributions.up.sql`
- Create: `migrations/000054_form_distributions.down.sql`
- Create: `internal/evidence/distribution.go`
- Create: `internal/evidence/distribution_store.go`
- Create: `internal/evidence/distribution_memory.go`
- Create: `internal/evidence/distribution_postgres.go`
- Create: `internal/evidence/distribution_postgres_integration_test.go`
- Modify: `internal/evidence/repository.go`
- Modify: `internal/evidence/model.go`
- Test: `internal/evidence/legal_entity_migration_test.go`

- [ ] **Step 1: Write failing schema and aggregate tests**

Assert one distribution pins one exact template revision, To recipients each reference one canonical `capture_request`, CC recipients have no request/edit capability, a shared workspace is unique per distribution, recipient role/type combinations are checked, cross-tenant/entity references fail, and `(deadline,id)` plus `(updated_at,id)` keyset indexes exist.

- [ ] **Step 2: Run focused tests and confirm failure**

Run: `go test ./internal/evidence -run 'Distribution|Workspace|LegalEntityMigration'`

Expected: FAIL because distribution records and migration `000054` do not exist.

- [ ] **Step 3: Create the compatibility-safe schema**

```sql
CREATE TABLE capture_form_distributions (
  id uuid PRIMARY KEY DEFAULT uuidv7(), tenant_id uuid NOT NULL REFERENCES tenants(id),
  legal_entity_id uuid NOT NULL, form_template_id uuid NOT NULL, form_template_version bigint NOT NULL,
  subject_type text NOT NULL, subject_id uuid NOT NULL, title text NOT NULL, purpose text NOT NULL,
  access_policy text NOT NULL CHECK (access_policy IN ('DIRECT_MAGIC_LINK','SHARED_LINK_EMAIL_OTP','DIRECT_LINK_EMAIL_OTP')),
  status text NOT NULL CHECK (status IN ('DRAFT','READY','OPEN','LOCKED','COMPLETED','EXPIRED','REVOKED','SUPERSEDED')),
  deadline timestamptz NOT NULL, route_expires_at timestamptz NOT NULL,
  reminder_policy jsonb NOT NULL DEFAULT '{}'::jsonb, created_by uuid NOT NULL, version bigint NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL DEFAULT clock_timestamp(), updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  UNIQUE (id,tenant_id,legal_entity_id),
  FOREIGN KEY (tenant_id,form_template_id,form_template_version) REFERENCES monitoring_form_templates(tenant_id,id,version),
  CHECK (route_expires_at <= deadline)
);

CREATE TABLE capture_distribution_recipients (
  id uuid PRIMARY KEY DEFAULT uuidv7(), distribution_id uuid NOT NULL, tenant_id uuid NOT NULL,
  legal_entity_id uuid NOT NULL, role text NOT NULL CHECK (role IN ('TO','CC')),
  recipient_type text NOT NULL CHECK (recipient_type IN ('INTERNAL_PRINCIPAL','EXTERNAL_AUDIENCE')),
  principal_id uuid, request_id uuid, address_hash bytea, address_ciphertext bytea, address_key_id text,
  audience_hint text NOT NULL DEFAULT '', contact_label text NOT NULL DEFAULT '',
  state text NOT NULL CHECK (state IN ('PENDING','DELIVERED','VERIFIED','REVOKED','COMPLETED')),
  version bigint NOT NULL DEFAULT 1, created_at timestamptz NOT NULL DEFAULT clock_timestamp(), updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  FOREIGN KEY (distribution_id,tenant_id,legal_entity_id) REFERENCES capture_form_distributions(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
  FOREIGN KEY (request_id,tenant_id) REFERENCES capture_requests(id,tenant_id),
  CHECK ((recipient_type='INTERNAL_PRINCIPAL' AND principal_id IS NOT NULL AND address_hash IS NULL AND address_ciphertext IS NULL)
      OR (recipient_type='EXTERNAL_AUDIENCE' AND principal_id IS NULL AND address_hash IS NOT NULL AND address_ciphertext IS NOT NULL)),
  CHECK ((role='TO' AND request_id IS NOT NULL) OR (role='CC' AND request_id IS NULL))
);
```

The same migration creates `capture_access_routes`, `capture_otp_challenges`, `capture_response_workspaces`, `capture_response_workspace_edits` and `capture_response_revisions`; adds nullable `distribution_id` to `capture_requests` and `capture_submissions`; and adds exact composite foreign keys. `capture_response_revisions` references an existing immutable submission and stores `revision`, `supersedes_revision_id`, achieved assurance, sign-off summary, compliance score, scored-weight coverage, final/provisional state, critical-field results, scoring-policy version and current flag. Legacy rows remain null-distribution records.

- [ ] **Step 4: Add domain types with explicit assurance**

```go
type AccessPolicy string
const (
	AccessDirectMagicLink AccessPolicy = "DIRECT_MAGIC_LINK"
	AccessSharedEmailOTP  AccessPolicy = "SHARED_LINK_EMAIL_OTP"
	AccessDirectEmailOTP  AccessPolicy = "DIRECT_LINK_EMAIL_OTP"
)
type AccessAssurance string
const (
	AssuranceLinkPossession AccessAssurance = "LINK_POSSESSION"
	AssuranceEmailVerified  AccessAssurance = "EMAIL_VERIFIED"
)
type RecipientRole string
const (RecipientTo RecipientRole = "TO"; RecipientCC RecipientRole = "CC")
type DistributionStatus string
const (
	DistributionDraft DistributionStatus = "DRAFT"; DistributionReady DistributionStatus = "READY"
	DistributionOpen DistributionStatus = "OPEN"; DistributionLocked DistributionStatus = "LOCKED"
	DistributionCompleted DistributionStatus = "COMPLETED"; DistributionExpired DistributionStatus = "EXPIRED"
	DistributionRevoked DistributionStatus = "REVOKED"; DistributionSuperseded DistributionStatus = "SUPERSEDED"
)

type FormDistribution struct {
	ID, TenantID, LegalEntityID, FormTemplateID, SubjectType, SubjectID, Title, Purpose string
	FormTemplateVersion int64
	AccessPolicy AccessPolicy
	Status DistributionStatus
	Deadline, RouteExpiresAt time.Time
	Version int64
}
```

Use separate safe recipient projections that contain hint/label/state but never ciphertext, hashes or full addresses.

- [ ] **Step 5: Implement atomic PostgreSQL creation**

`CreateDistribution` opens one transaction, checks the pinned active form revision and current authority inputs, inserts the distribution, creates one existing-format `capture_request` per To recipient, stores protected recipient rows, creates exactly one response workspace, appends the distribution event and inserts the outbox row. Any failure rolls back all rows. Memory behavior must match.

- [ ] **Step 6: Run evidence and PostgreSQL integration tests**

Run: `go test ./internal/evidence`

CI/PostgreSQL run: `go test -tags 'postgres postgresintegration' ./internal/evidence -run 'Distribution|Workspace'`

Expected: PASS. Do not start Docker locally; use the configured CI/PostgreSQL environment.

- [ ] **Step 7: Commit**

```bash
git add migrations/000054_form_distributions.* internal/evidence
git commit -m "feat: persist form distributions and workspaces"
```

### Task 7: Protect external addresses and implement the three access policies

**Files:**
- Create: `internal/evidence/protected_recipient.go`
- Create: `internal/evidence/protected_recipient_test.go`
- Create: `internal/evidence/access_policy.go`
- Create: `internal/evidence/access_policy_test.go`
- Create: `internal/evidence/otp.go`
- Create: `internal/evidence/otp_test.go`
- Modify: `internal/evidence/distribution_postgres.go`
- Modify: `internal/evidence/service.go`
- Modify: `internal/evidence/model.go`
- Modify: `internal/platform/config/config.go`
- Modify: `internal/platform/config/config_test.go`
- Modify: `.env.example`

- [ ] **Step 1: Add failing cryptography and access-ceremony tests**

Test AES-256-GCM round trip with tenant/distribution/recipient AAD, wrong-scope decryption failure, active-key rotation, no plaintext in serialized values/errors, direct link `LINK_POSSESSION`, route-bound random selector IDs, shared masked selection, OTP verification, ten-minute expiry, single use, attempt/resend caps, generic unknown-recipient errors, route rotation, route/session revocation and deadline clamps.

- [ ] **Step 2: Run the tests and confirm the protected components are absent**

Run: `go test ./internal/evidence ./internal/platform/config -run 'ProtectedRecipient|AccessPolicy|OTP|RecipientEncryption'`

Expected: FAIL.

- [ ] **Step 3: Add a purpose-specific encryption keyring**

```go
type RecipientKeyring struct { ActiveID string; Keys map[string][32]byte }
type ProtectedAddress struct { Ciphertext []byte; KeyID, HashHex, Hint string }

func (k RecipientKeyring) Protect(tenantID, distributionID, recipientID, address string) (ProtectedAddress, error)
func (k RecipientKeyring) Reveal(tenantID, distributionID, recipientID string, value ProtectedAddress) (string, error)
```

Load `CLEARSIGHT_RECIPIENT_KEYRING` as a JSON object of base64 32-byte keys plus `CLEARSIGHT_RECIPIENT_ACTIVE_KEY_ID`. Production refuses to enable external distribution delivery without a valid active key. Use normalized address hash for equality, random 12-byte nonce for AES-GCM and `tenant|distribution|recipient` as AAD.

- [ ] **Step 4: Implement route inspection and OTP challenges**

```go
type MaskedRecipient struct {
	SelectorID string `json:"selector_id"`
	Hint string `json:"hint"`
	ContactLabel string `json:"contact_label,omitempty"`
}
type AccessStart struct {
	Policy AccessPolicy `json:"policy"`
	Recipients []MaskedRecipient `json:"recipients,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
}
type OTPChallenge struct { ID, RouteID, RecipientID string; Digest []byte; ExpiresAt time.Time; Attempts, Resends int; ConsumedAt *time.Time }
```

Direct magic links redeem without an audience string and store `LINK_POSSESSION`. OTP send resolves/decrypts only the route-bound random selector for an eligible To recipient inside the protected boundary; database recipient IDs are never public selectors. Digest `challengeID|sixDigitCode` with a separate configured HMAC key, compare in constant time, cap attempts at 5 and resends at 3, and return the same public error for unknown, expired, revoked, exhausted or mismatched challenges.

- [ ] **Step 5: Preserve legacy invitations through an adapter**

Existing one-recipient invitations continue using `IssueInvitation`/`RedeemInvitation`. New distribution routes use `StartDistributionAccess`, `SendOTP`, `VerifyOTP` and `RedeemDirectRoute`. Both produce the existing bounded `SessionRequest` capability shape, extended with distribution/recipient/assurance fields. No body actor or audience value can override the stored route policy.

- [ ] **Step 6: Run evidence security tests**

Run: `go test ./internal/evidence ./internal/platform/config`

Expected: PASS, including legacy invitation tests.

- [ ] **Step 7: Commit**

```bash
git add internal/evidence internal/platform/config .env.example
git commit -m "feat: add adaptive external form access"
```

### Task 8: Move new drafts to the shared workspace and add immutable amendments

**Files:**
- Create: `internal/evidence/response_workspace.go`
- Create: `internal/evidence/response_workspace_test.go`
- Create: `internal/evidence/response_workspace_postgres_test.go`
- Modify: `internal/evidence/draft.go`
- Modify: `internal/evidence/draft_postgres.go`
- Modify: `internal/evidence/service.go`
- Modify: `internal/evidence/postgres.go`
- Modify: `internal/evidence/memory.go`

- [ ] **Step 1: Write failing workspace conflict and amendment tests**

Cover two To sessions editing different fields, stale same-field conflict, server-returned changed fields, field actor/assurance provenance, first immutable submission, reopened edit before deadline, amended revision supersession, lock/revoke/deadline blocking, and legacy session-owned draft behavior.

- [ ] **Step 2: Run focused tests and confirm failure**

Run: `go test ./internal/evidence -run 'ResponseWorkspace|Amendment|Draft'`

Expected: FAIL on missing shared-workspace methods.

- [ ] **Step 3: Add workspace command contracts**

```go
type FieldEdit struct { FieldID string; Value formcontract.AnswerValue; BaseSequence int64 }
type SaveWorkspaceInput struct { ExpectedVersion int64; PresentationMode formcontract.PresentationMode; Edits []FieldEdit }
type FieldChange struct { FieldID string `json:"field_id"`; ServerValue formcontract.AnswerValue `json:"server_value"`; Sequence int64 `json:"sequence"` }
type WorkspaceConflict struct { CurrentVersion int64; Changed []FieldChange `json:"changed_fields"` }
type SubmitWorkspaceInput struct { ExpectedVersion int64; AttestationFieldIDs []string }
```

Reject duplicate field IDs and entire-answer-map replacement for distribution saves. Validate changed values against the pinned request contract. Store one edit-provenance row per accepted field change with current session/grant/assurance; update workspace, append event and outbox in one transaction.

- [ ] **Step 4: Implement immutable response revisions**

`SubmitWorkspace` validates and scores the exact pinned template revision, snapshots all current answers into the existing `capture_submissions` table, creates one `capture_response_revisions` row and marks the prior response revision non-current. Store compliance score, scored-weight coverage, final/provisional state, critical-field results, scoring-policy version and calculation time on the response revision. It does not close the distribution before deadline unless the sender locks it. The submission channel is `MAGIC_LINK` for external sessions and `INTERNAL` for verified principals; achieved assurance is stored separately and never inferred from the channel.

- [ ] **Step 5: Keep legacy draft methods unchanged for null-distribution requests**

`GetDraft`, `SaveDraft`, `DeleteDraft` dispatch to the workspace implementation when `request.DistributionID` exists and retain session ownership for legacy requests. This makes migration deployable without rewriting old drafts.

- [ ] **Step 6: Run evidence tests**

Run: `go test ./internal/evidence`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/evidence
git commit -m "feat: add shared response revisions"
```

### Task 9: Add distribution and access HTTP APIs

**Files:**
- Create: `internal/httpapi/form_distribution_handlers.go`
- Create: `internal/httpapi/form_distribution_handlers_test.go`
- Modify: `internal/httpapi/evidence_handlers.go`
- Modify: `internal/httpapi/route_registry.go`
- Modify: `internal/httpapi/server.go`
- Modify: `cmd/api/services.go`
- Modify: `cmd/api/services_memory.go`
- Modify: `cmd/api/services_postgres.go`
- Modify: `cmd/api/main.go`
- Modify: `api/runtime.openapi.json`

- [ ] **Step 1: Add failing route-class, identity and redaction tests**

Test create/list/detail/amend/rotate-access/supersede/lock/reopen/revoke; public access inspection/OTP send/OTP verify/direct redeem; session workspace get/save/submit; exact actor scope; CC exclusion; masked-only public data; no token/address/OTP in JSON errors; compatible-answer carry-forward preview; and material-command reauthorization.

- [ ] **Step 2: Run the focused HTTP tests**

Run: `go test ./internal/httpapi -run 'FormDistribution|FormAccess|ResponseWorkspace|RuntimeContract'`

Expected: FAIL because the routes are not registered.

- [ ] **Step 3: Register authenticated distribution routes**

```text
GET  /api/v1/forms/distributions
POST /api/v1/forms/distributions
GET  /api/v1/forms/distributions/{id}
POST /api/v1/forms/distributions/{id}/amend
POST /api/v1/forms/distributions/{id}/access-routes/{route_id}/rotate
POST /api/v1/forms/distributions/{id}/supersede
POST /api/v1/forms/distributions/{id}/lock
POST /api/v1/forms/distributions/{id}/reopen
POST /api/v1/forms/distributions/{id}/revoke
```

All mutations are material commands using verified identity and current version. Amend changes recipients, reminder policy or policy-valid deadline with an impact preview and notification. Route rotation atomically revokes the old route and all sessions issued from it before creating a replacement. Supersede pins a new exact template revision and requires the sender to preview and confirm stable-key compatible-answer carry-forward; incompatible/removed values stay only in prior history. List is keyset-paginated by due state/subject/owner and filters before limit.

- [ ] **Step 4: Register bounded public/session routes**

```text
POST /api/v1/evidence/access/start
POST /api/v1/evidence/access/otp/send
POST /api/v1/evidence/access/otp/verify
POST /api/v1/evidence/access/redeem
GET  /api/v1/evidence/session/workspace
PATCH /api/v1/evidence/session/workspace
POST /api/v1/evidence/session/workspace/submissions
```

Secrets occur only in POST bodies or Authorization headers. Public handlers set `Cache-Control: no-store`, a strict referrer policy and generic failure bodies. Session handlers resolve the distribution from the token; they accept no tenant, recipient or assurance override.

- [ ] **Step 5: Run route and full HTTP tests**

Run: `go test ./internal/httpapi`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/httpapi cmd/api api/runtime.openapi.json
git commit -m "feat: expose form distribution lifecycle"
```

### Task 10: Add governed communications, branding and SMTP delivery

**Files:**
- Create: `migrations/000055_form_communications.up.sql`
- Create: `migrations/000055_form_communications.down.sql`
- Create: `internal/evidence/communications.go`
- Create: `internal/evidence/communications_test.go`
- Create: `internal/evidence/communications_postgres.go`
- Create: `internal/evidence/communications_postgres_test.go`
- Create: `internal/evidence/smtp_delivery.go`
- Create: `internal/evidence/smtp_delivery_test.go`
- Create: `internal/evidence/communication_delivery_worker.go`
- Create: `internal/evidence/communication_delivery_worker_test.go`
- Create: `internal/httpapi/form_communication_handlers.go`
- Create: `internal/httpapi/form_communication_handlers_test.go`
- Modify: `internal/platform/config/config.go`
- Modify: `cmd/api/main.go`
- Modify: `cmd/worker/services.go`
- Modify: `cmd/worker/services_postgres.go`
- Modify: `.env.example`
- Modify: `api/runtime.openapi.json`

- [ ] **Step 1: Add failing structured-content and delivery tests**

Cover invitation/reminder/due-soon/expired/change/amendment/completion actions, allowlisted nodes and placeholders, unsafe URL/HTML rejection, required secure-link placeholder, logo validation, locale fallback, maker-checker activation, effective dating, impact preview, rollback to an exact prior version, HTML/plain-text rendering, protected-value redaction, SMTP STARTTLS, idempotent retry and deadline/revocation stop.

- [ ] **Step 2: Run tests and confirm failure**

Run: `go test ./internal/evidence ./internal/httpapi -run 'Communication|SMTP|Branding'`

Expected: FAIL.

- [ ] **Step 3: Persist versioned communication configuration**

Create `form_communication_profiles`, `form_communication_templates`, `form_brand_assets` and `form_delivery_attempts`. Profiles/templates are versioned by legal entity, action and locale with `effective_from`/`effective_until`, maker/checker identities, rollback origin and append-only audit/outbox records. Template bodies use a bounded JSON document with paragraph, heading, strong, emphasis, link, list, divider, callout and primary-action nodes. Store logo artifacts through the existing object-store/inspection boundary; do not store arbitrary HTML or remote image URLs.

- [ ] **Step 4: Implement server rendering and placeholder expansion**

```go
type protectedString struct { value string }
func (protectedString) String() string { return "[REDACTED]" }
func (protectedString) GoString() string { return "[REDACTED]" }
type RenderedMessage struct { Subject, PlainText, HTML protectedString }
type CommunicationContext struct {
	RecipientName, BankName, FormTitle, TaskSummary, DueTime, LinkExpiry, AccessInstructions, SupportContact string
	SecureFormLink protectedString
}
func RenderCommunication(template CommunicationTemplate, context CommunicationContext) (RenderedMessage, error)
```

Validate all placeholders at approval and again at send. Resolve the exact effective requested locale, then the configured legal-entity default; missing both fails with a recovery action rather than silently selecting unrelated copy. Preview uses labelled sample values and cannot construct a route or OTP. Protected context and rendered values implement redacted `String`/`GoString` behavior so logs never print body recipients or links.

- [ ] **Step 5: Add configured SMTP transport and scheduled reminder delivery without a new queue**

Use the existing runtime outbox delivery job and `InvitationDelivery` interface. Configure host, port, username, secret reference, from address and TLS mode. Require implicit TLS or STARTTLS outside development. Process invitation and due reminder events in bounded, leased, idempotent outbox batches; calculate reminder eligibility from the stored policy and stop scheduling after completion, revocation, lock or deadline. Provider receipts retain only message ID, status, attempt time and redacted hint. CC messages contain status information but no response route.

- [ ] **Step 6: Add governed configuration routes**

Register list/detail/create-revision/preview/test-send/impact/transition/rollback routes under `/api/v1/forms/communications`; creation and activation require configuration authority and maker-checker. Logo upload uses the existing bounded artifact path.

- [ ] **Step 7: Run communication and worker tests**

Run: `go test ./internal/evidence ./internal/httpapi ./cmd/worker`

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add migrations/000055_form_communications.* internal/evidence internal/httpapi internal/platform/config cmd/api cmd/worker .env.example api/runtime.openapi.json
git commit -m "feat: deliver governed form communications"
```

### Task 11: Build the distribution, OTP and communication UI

**Files:**
- Create: `web/src/components/forms/DistributionComposer.tsx`
- Create: `web/src/components/forms/DistributionComposer.test.tsx`
- Create: `web/src/components/forms/SentFormsView.tsx`
- Create: `web/src/components/forms/ResponsesView.tsx`
- Create: `web/src/components/forms/CommunicationEditor.tsx`
- Create: `web/src/components/forms/CommunicationEditor.test.tsx`
- Modify: `web/src/components/FormsWorkspace.tsx`
- Modify: `web/src/formsTypes.ts`
- Modify: `web/src/formsApi.ts`
- Modify: `web/src/formsApi.test.ts`
- Modify: `web/package.json`
- Modify: `web/src/forms.css`

- [ ] **Step 1: Add failing workflow tests**

Test multi-row To/CC composition, internal bounded search, access-policy explanation, shared-link recommended state, exact calendar deadline/expiry, expiry clamp notice, recipient delivery/revocation state, reminder/amend/rotate/supersede/lock/reopen actions, carry-forward preview, authorized route-copy fallback, communication placeholder picker, logo alt text and accessible mobile preview.

- [ ] **Step 2: Run the tests and confirm failure**

Run: `cd web && npm test -- DistributionComposer.test.tsx CommunicationEditor.test.tsx FormsWorkspace.test.tsx`

Expected: FAIL.

- [ ] **Step 3: Implement the distribution composer**

Use repeatable labelled recipient rows with `To`/`CC`, internal/external type and bounded candidate search. Default vendor due diligence to shared link plus email OTP. Show concise policy cards: **Open with link**, **Verify email from shared link**, **Verify email from personal link**. A primary **Send form** button stays disabled until the exact template revision, subject, at least one To recipient, deadline, effective expiry and communication preview are valid.

- [ ] **Step 4: Implement Sent forms and Responses views**

Lists use server pages and preserve filter/search state in the URL. Detail shows exact revision, recipient role/delivery/verification/revocation, achieved assurance, current workspace version, response revision history, sign-off and stale-review warnings. It supports due/reminder/recipient amendments, access-route rotation, superseding-distribution carry-forward preview, and an authorized **Copy secure link** fallback that repeats the selected assurance/forwarding warning and never exposes recipient or OTP data. Controls invoke real APIs or are disabled with a visible reason.

- [ ] **Step 5: Implement the constrained WYSIWYG**

Add `lexical`, `@lexical/react`, `@lexical/rich-text`, `@lexical/list` and `@lexical/link`; lazy-load the editor only on the Communications tab. Serialize only the server allowlist, expose keyboard-labelled toolbar buttons, sample placeholder insertion, locale/default selection, desktop/mobile and plain-text previews, impact/rollback and maker-checker state. The Configure workspace links directly to this same Communications route rather than mounting a second editor.

- [ ] **Step 6: Run web unit, accessibility and build checks**

Run: `cd web && npm test -- DistributionComposer.test.tsx CommunicationEditor.test.tsx FormsWorkspace.test.tsx Accessibility.test.tsx copyQuality.test.ts && npm run build`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add web/package.json web/package-lock.json web/src
git commit -m "feat: add form distribution workspace"
```

### Task 12: Add encrypted IndexedDB recovery and the external OTP journey

**Files:**
- Create: `web/src/captureRecoveryStore.ts`
- Create: `web/src/captureRecoveryCrypto.ts`
- Create: `web/src/captureRecovery.ts`
- Create: `web/src/captureRecovery.test.ts`
- Modify: `web/src/captureApi.ts`
- Modify: `web/src/captureApi.test.ts`
- Modify: `web/src/captureInvitationBrowser.ts`
- Modify: `web/src/components/ExternalCaptureApp.tsx`
- Modify: `web/src/components/ExternalCaptureApp.test.tsx`
- Modify: `web/src/components/capture/CaptureForm.tsx`
- Modify: `web/src/components/capture/CaptureForm.test.tsx`
- Modify: `web/src/capture-inputs.css`
- Modify: `web/src/staticExternalCapture.ts`
- Modify: `web/src/staticExternalCapture.test.ts`

- [ ] **Step 1: Write failing cache exclusion, expiry and merge tests**

```ts
it("never places secrets, files or signatures in the recovery envelope", async () => {
  await recovery.save(context, fields, answersWithTextFileAndSignature);
  const raw = await store.readRaw(context.key);
  expect(raw).not.toContain("session-token");
  expect(raw).not.toContain("file-bytes");
  expect(raw).not.toContain("signature-data");
  expect(await recovery.restore(context)).toMatchObject({ answers: { legal_name: { text: "Example Ltd" } }, filesToReselect: ["certificate"] });
});
```

Also cover refresh/crash, offline save, corrupt/decryption failure, schema mismatch, `NO_BROWSER_CACHE`, seven-day/deadline/route TTL, clear, successful-submit purge, revoked-session denial, untrusted recovered-answer rendering without HTML execution, server/device status copy and field-level merge.

- [ ] **Step 2: Run tests and confirm failure**

Run: `cd web && npm test -- captureRecovery.test.ts ExternalCaptureApp.test.tsx CaptureForm.test.tsx`

Expected: FAIL.

- [ ] **Step 3: Implement the recovery envelope and store abstraction**

```ts
export type RecoveryEnvelope = {
  distributionID: string; workspaceID: string; schemaVersion: number; serverVersion: number;
  page: number; answers: CaptureAnswerInputs; filesToReselect: string[];
  localSequence: number; updatedAt: string; expiresAt: string;
};
export type EncryptedEnvelope = {
  version: 1; algorithm: "AES-GCM"; iv: ArrayBuffer; ciphertext: ArrayBuffer;
  expiresAt: string; schemaVersion: number;
};
export interface RecoveryStore {
  get(key: string): Promise<EncryptedEnvelope | undefined>;
  put(key: string, value: EncryptedEnvelope): Promise<void>;
  delete(key: string): Promise<void>;
  getOrCreateDeviceKey(key: string): Promise<CryptoKey>;
}
```

The production adapter uses IndexedDB and stores a non-exportable AES-GCM `CryptoKey`; tests use an in-memory adapter. Bind ciphertext AAD to origin, legal entity, distribution and schema revision. Do not restore until a current server session has authorized that exact distribution.

- [ ] **Step 4: Integrate immediate local save and debounced server save**

Write permitted scalar changes locally immediately. Debounce the PATCH workspace call, flush on page transition and `visibilitychange`, and show exactly **Saved to ClearSight**, **Saving**, **Saved on this device — waiting to sync**, **Resolve changed answers**, or **Save failed — retry**. A conflict renders server/local values field by field; no automatic last-write-wins.

- [ ] **Step 5: Replace audience typing with the policy-driven access journey**

After consuming `capture_invite` from the URL, call access start. Direct magic link redeems immediately. Shared OTP shows only server-provided masked To rows, sends the challenge after selection and accepts an accessible one-time-code input. Direct OTP skips selection. Resend shows remaining wait time. No full email, token or OTP enters session/local storage, analytics or page title.

- [ ] **Step 6: Handle files and final submission truthfully**

Cache file metadata only. After recovery, each unsynced file field says **Reselect file to upload**. Purge the envelope only after the server confirms the final response revision; keep it when submission fails. A later permitted amendment begins from the server workspace.

- [ ] **Step 7: Run web checks**

Run: `cd web && npm test -- captureRecovery.test.ts captureApi.test.ts ExternalCaptureApp.test.tsx CaptureForm.test.tsx staticExternalCapture.test.ts Accessibility.test.tsx copyQuality.test.ts && npm run build`

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add web/src
git commit -m "feat: recover long form responses safely"
```

## Tranche 3 — Document and AI-assisted form authoring

### Task 13: Make extraction outcomes and structure explicit

**Files:**
- Modify: `internal/documentimport/model.go`
- Modify: `internal/documentimport/extractor.go`
- Modify: `internal/documentimport/extraction_policy.go`
- Create: `internal/documentimport/elements.go`
- Create: `internal/documentimport/elements_test.go`
- Modify: `internal/documentimport/resource_limits_test.go`
- Modify: `internal/documentimport/pdf_extractor_test.go`
- Modify: `internal/documentimport/service.go`
- Modify: `internal/documentimport/postgres.go`

- [ ] **Step 1: Add failing explicit-outcome and element tests**

```go
func TestExtractionNeverReturnsEmptySuccessAfterParserFailure(t *testing.T) {
	result := Extract("broken.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", []byte("not a zip"))
	if result.Status == ExtractionExtracted && len(result.Elements) == 0 { t.Fatalf("empty success: %#v", result) }
}

func TestCollectorMarksRetainedOutputTruncated(t *testing.T) {
	result := ExtractWithPolicy(t.Context(), "large.txt", "text/plain", []byte(strings.Repeat("section\n\n", 100)), ExtractionPolicy{MaxSections: 2})
	if result.Status != ExtractionTruncated || !result.ContentTruncated || result.SectionsOmitted == 0 { t.Fatalf("missing truncation truth: %#v", result) }
}
```

- [ ] **Step 2: Run focused tests and confirm failure**

Run: `go test ./internal/documentimport -run 'ExtractionNever|CollectorMarks|Element'`

Expected: FAIL because `ExtractionTruncated` and structured elements do not exist.

- [ ] **Step 3: Add compatible statuses and structured elements**

```go
const (
	ExtractionPartial   ExtractionStatus = "PARTIAL"
	ExtractionTruncated ExtractionStatus = "TRUNCATED"
)
type ElementKind string
const (
	ElementHeading ElementKind = "HEADING"; ElementParagraph ElementKind = "PARAGRAPH"
	ElementTable ElementKind = "TABLE"; ElementFormControl ElementKind = "FORM_CONTROL"
	ElementImage ElementKind = "IMAGE"; ElementLink ElementKind = "LINK"
)
type BoundingBox struct { X0, Y0, X1, Y1 float64 }
type FormControl struct { Kind, Label, Help string; Options []string; Checked *bool }
type Degradation struct { Code, Message string; Recoverable bool; Anchor *SourceAnchor }
type SourceAnchor struct {
	Page int `json:"page,omitempty"`
	Sheet string `json:"sheet,omitempty"`
	RowStart int `json:"row_start,omitempty"`
	RowEnd int `json:"row_end,omitempty"`
	Paragraph string `json:"paragraph,omitempty"`
	Table string `json:"table,omitempty"`
	Cell string `json:"cell,omitempty"`
	BoundingBox *BoundingBox `json:"bounding_box,omitempty"`
}
type ExtractedElement struct { Kind ElementKind `json:"kind"`; Text string `json:"text,omitempty"`; Values [][]string `json:"values,omitempty"`; Control *FormControl `json:"control,omitempty"`; Anchor SourceAnchor `json:"anchor"` }
```

Keep `Sections` for current proposal and UI compatibility. Add parser/adapter version and structured degradations to `ExtractionResult`/`Document`. A recoverable missing element class is `PARTIAL`; retention caps are `TRUNCATED`; structural/resource/parser failure is `FAILED`.

- [ ] **Step 4: Persist extraction details without widening list payloads**

Add bounded `extraction_details jsonb` to `document_imports` in migration `000056` from Task 15. Full detail exposes elements/degradations; `DocumentSummary` exposes only status, parser version and counts. Do not put element bodies in list reads.

- [ ] **Step 5: Run document-import tests**

Run: `go test ./internal/documentimport`

Expected: PASS with prior extraction behavior represented by the compatible statuses.

- [ ] **Step 6: Commit**

```bash
git add internal/documentimport
git commit -m "feat: expose document extraction structure"
```

### Task 14: Extract bounded DOCX form controls and establish the golden corpus

**Files:**
- Create: `internal/documentimport/docx_form_controls.go`
- Create: `internal/documentimport/docx_form_controls_test.go`
- Create: `internal/documentimport/golden_corpus_test.go`
- Create: `internal/documentimport/testdata/golden/manifest.json`
- Modify: `internal/documentimport/extractor.go`
- Modify: `internal/documentimport/extraction_policy.go`

- [ ] **Step 1: Add failing generated-fixture tests for Word controls**

Use an in-test bounded ZIP builder to create `word/document.xml`, relationships and numbering fixtures. Cover plain text input, checkbox, dropdown options, content-control title/help, numbered questions, nested table cells, relationship hyperlinks, path traversal, entry count, expanded size and compression ratio.

```go
func TestDOCXFormControlsKeepQuestionAndOptions(t *testing.T) {
	data := zipFixture(t, map[string][]byte{
		"[Content_Types].xml": []byte(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="xml" ContentType="application/xml"/></Types>`),
		"word/document.xml": []byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:sdt><w:sdtPr><w:alias w:val="Country"/><w:dropDownList><w:listItem w:displayText="Nigeria" w:value="NG"/><w:listItem w:displayText="Ghana" w:value="GH"/></w:dropDownList></w:sdtPr><w:sdtContent/></w:sdt></w:p></w:body></w:document>`),
	})
	result := Extract("questionnaire.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", data)
	for _, element := range result.Elements {
		if element.Control != nil && element.Control.Kind == "DROPDOWN" && element.Control.Label == "Country" && slices.Equal(element.Control.Options, []string{"Nigeria", "Ghana"}) { return }
	}
	t.Fatalf("dropdown control was not preserved: %#v", result.Elements)
}
```

- [ ] **Step 2: Run the DOCX tests and confirm failure**

Run: `go test ./internal/documentimport -run 'DOCXForm|GoldenCorpus'`

Expected: FAIL because controls and manifest runner are absent.

- [ ] **Step 3: Implement streaming OOXML mappings**

Parse `w:sdt`, `w:fldSimple`, complex field begin/separate/end sequences, `w:checkBox`, `w:dropDownList`, `w:textInput`, paragraphs, numbering and tables with `encoding/xml`. Resolve only allowlisted relationships inside the validated archive. Do not enable an XML huge-tree/unbounded entity mode. Consume the existing archive, text, section, row and cell budgets for every retained value.

- [ ] **Step 4: Define golden expectations**

`manifest.json` lists fixture name, source kind, expected status, ordered element kinds, expected labels/options/table dimensions/anchors, required degradations and maximum extraction duration class. Test builders supply native/scanned PDF, AcroForm-adapter contract, DOCX, XLSX, legacy-conversion, malformed and bomb fixtures without checking binary artifacts into Git unless their license and source are recorded.

- [ ] **Step 5: Run the full import package and compile check**

Run: `go test ./internal/documentimport && go test ./...`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/documentimport
git commit -m "feat: extract Word form controls safely"
```

### Task 15: Add reviewable document-to-template proposals

**Files:**
- Create: `migrations/000056_form_template_proposals.up.sql`
- Create: `migrations/000056_form_template_proposals.down.sql`
- Create: `internal/documentimport/form_template_proposal.go`
- Create: `internal/documentimport/form_template_proposal_test.go`
- Create: `internal/monitoring/form_proposal.go`
- Create: `internal/monitoring/form_proposal_store.go`
- Create: `internal/monitoring/form_proposal_memory.go`
- Create: `internal/monitoring/form_proposal_postgres.go`
- Create: `internal/monitoring/form_proposal_test.go`
- Create: `internal/httpapi/form_proposal_handlers.go`
- Create: `internal/httpapi/form_proposal_handlers_test.go`
- Modify: `internal/httpapi/route_registry.go`
- Modify: `cmd/worker/services.go`
- Modify: `cmd/worker/services_postgres.go`
- Modify: `api/runtime.openapi.json`

- [ ] **Step 1: Add failing deterministic proposal tests**

Test spreadsheet header/type/option inference, DOCX heading/control/table inference, stable field keys, source anchors, confidence, unresolved ambiguity, weight not guessed, no automatic activation, exact source hash/version and selective acceptance into a normal draft revision.

- [ ] **Step 2: Run tests and confirm failure**

Run: `go test ./internal/documentimport ./internal/monitoring ./internal/httpapi -run 'FormTemplateProposal'`

Expected: FAIL.

- [ ] **Step 3: Create proposal persistence**

```sql
ALTER TABLE document_imports ADD COLUMN extraction_details jsonb NOT NULL DEFAULT '{}'::jsonb;
CREATE TABLE form_template_proposals (
  id uuid PRIMARY KEY DEFAULT uuidv7(), tenant_id uuid NOT NULL REFERENCES tenants(id), legal_entity_id uuid NOT NULL,
  source_kind text NOT NULL CHECK (source_kind IN ('DOCUMENT','AI')), source_document_id uuid,
  source_sha256 text NOT NULL DEFAULT '', base_template_id uuid, base_template_version bigint,
  status text NOT NULL CHECK (status IN ('GENERATING','REVIEW_REQUIRED','ACCEPTED','REJECTED','FAILED')),
  proposed_contract jsonb NOT NULL DEFAULT '{}'::jsonb, field_changes jsonb NOT NULL DEFAULT '[]'::jsonb,
  unresolved_items jsonb NOT NULL DEFAULT '[]'::jsonb, provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_by uuid NOT NULL, reviewed_by uuid, created_at timestamptz NOT NULL DEFAULT clock_timestamp(), reviewed_at timestamptz,
  version bigint NOT NULL DEFAULT 1,
  CHECK (octet_length(proposed_contract::text) <= 1048576 AND jsonb_array_length(field_changes) <= 500)
);
```

Add tenant/entity/status/time indexes and document/form foreign keys. The down migration removes proposal rows/details only; it does not alter imported originals or templates.

- [ ] **Step 4: Implement deterministic transformation**

```go
func ProposeFormTemplate(document Document, policy ProposalPolicy) (FormTemplateProposal, error)
```

Map extracted controls directly; infer spreadsheet fields only from bounded samples; generate stable keys from normalized anchor plus label; attach every field to a page/sheet/row/paragraph/table anchor; surface type/requiredness/options/section uncertainty. Set scoring mode `NONE` and require author input for weights.

- [ ] **Step 5: Accept selected changes through the normal form service**

The maker sends proposal version plus selected change IDs. Re-read the proposal, source document hash, target base revision and current authority. Apply only selected changes, normalize through `formcontract.Normalize`, and call `CreateFormRevision`; mark the proposal accepted and append its receipt/event in the same material transaction where PostgreSQL composition supports it. Activation remains a separate checker command.

- [ ] **Step 6: Add routes and worker job**

```text
POST /api/v1/document-imports/{id}/form-template-proposals
GET  /api/v1/forms/proposals/{id}
POST /api/v1/forms/proposals/{id}/accept
POST /api/v1/forms/proposals/{id}/reject
```

Generation is outbox-driven for PostgreSQL and synchronous only in memory/demo. Routes never return source text outside the authorized exact-document read.

- [ ] **Step 7: Run tests and commit**

Run: `go test ./internal/documentimport ./internal/monitoring ./internal/httpapi ./cmd/worker`

Expected: PASS.

```bash
git add migrations/000056_form_template_proposals.* internal/documentimport internal/monitoring internal/httpapi cmd/worker api/runtime.openapi.json
git commit -m "feat: propose forms from imported documents"
```

### Task 16: Add optional parser adapters and governed AI diffs

**Files:**
- Create: `internal/documentimport/parser_adapter.go`
- Create: `internal/documentimport/parser_adapter_test.go`
- Create: `internal/documentimport/legacy_office_converter.go`
- Create: `internal/documentimport/legacy_office_converter_test.go`
- Create: `internal/monitoring/form_ai.go`
- Create: `internal/monitoring/form_ai_client.go`
- Create: `internal/monitoring/form_ai_test.go`
- Modify: `internal/platform/config/config.go`
- Modify: `internal/platform/config/config_test.go`
- Modify: `internal/httpapi/form_proposal_handlers.go`
- Modify: `internal/httpapi/form_proposal_handlers_test.go`
- Modify: `cmd/api/services.go`
- Modify: `cmd/api/services_memory.go`
- Modify: `cmd/api/services_postgres.go`
- Modify: `cmd/worker/services.go`
- Modify: `cmd/worker/services_postgres.go`
- Modify: `api/runtime.openapi.json`
- Modify: `.env.example`

- [ ] **Step 1: Add failing adapter and AI failure-mode tests**

Test adapter timeout/output/page limits, invalid anchors, partial response, disabled adapter, license gate, deterministic fallback, isolated legacy `.xls` conversion timeout/output/archive limits, strict AI response schema, tenant/workload binding, prompt/source provenance, exact base-version diff, provider failure leaving drafts unchanged and selective acceptance.

- [ ] **Step 2: Run focused tests and confirm failure**

Run: `go test ./internal/documentimport ./internal/monitoring -run 'ParserAdapter|FormAI'`

Expected: FAIL.

- [ ] **Step 3: Define a narrow optional parser contract**

```go
type ParserAdapter interface {
	Name() string
	Extract(context.Context, ParserRequest) (ParserResponse, error)
}
type ParserRequest struct { ArtifactID, FileName, MediaType string; Data io.Reader; MaxBytes int64; MaxPages int; Deadline time.Time }
type ParserResponse struct { ParserVersion string; Elements []ExtractedElement; Degradations []Degradation; Pages int; OutputBytes int64 }
```

The worker invokes an adapter only for configured formats and only after the default extractor reports the configured insufficiency. Validate the returned cardinality, anchors and sizes again in Go. `CLEARSIGHT_PDF_PARSER_ADAPTER=PYMUPDF` is rejected unless `CLEARSIGHT_PYMUPDF_LICENSE_APPROVED=true` and the endpoint is HTTPS outside development. No PyMuPDF package or Docker image is installed on developer machines by this plan.

Legacy `.xls` files use an explicitly configured LibreOffice executable only in the CI/production import worker. Run conversion in a fresh bounded temporary directory with a hard timeout, input/output byte caps, one expected output file and no network dependency, then feed the resulting `.xlsx` through the normal extractor and golden corpus. Missing or failed conversion returns an explicit recoverable `FAILED`/`PARTIAL` outcome; it never reports empty success. This plan does not install LibreOffice locally.

- [ ] **Step 4: Implement the governed AI-gateway client**

Use a dedicated active workload credential and call the configured AI gateway, not a provider directly. Request a strict JSON object containing bounded template metadata and field changes. Persist workload ID, policy/receipt ID, model alias, prompt version, source hashes/anchors, base revision and validation results in proposal provenance. Reject unknown field types, target keys, conditions or scoring totals.

- [ ] **Step 5: Add generate/revise routes**

```text
POST /api/v1/forms/proposals/ai
POST /api/v1/forms/templates/{id}/revisions/{version}/ai-proposals
```

Both create proposal records; neither mutates a draft until the normal accept command. Provider/gateway unavailable returns a recoverable proposal failure and leaves manual/deterministic authoring usable.

- [ ] **Step 6: Run tests and commit**

Run: `go test ./internal/documentimport ./internal/monitoring ./internal/httpapi ./cmd/worker`

Expected: PASS.

```bash
git add internal/documentimport internal/monitoring internal/httpapi internal/platform/config cmd/api cmd/worker api/runtime.openapi.json .env.example
git commit -m "feat: govern advanced form proposals"
```

### Task 17: Add source-anchored import and AI review UI

**Files:**
- Create: `web/src/components/forms/FormProposalReview.tsx`
- Create: `web/src/components/forms/FormProposalReview.test.tsx`
- Create: `web/src/components/forms/FormAIComposer.tsx`
- Create: `web/src/components/forms/FormAIComposer.test.tsx`
- Modify: `web/src/components/DocumentImportWorkspace.tsx`
- Modify: `web/src/components/DocumentImportWorkspace.test.tsx`
- Modify: `web/src/components/FormsWorkspace.tsx`
- Modify: `web/src/formsApi.ts`
- Modify: `web/src/formsTypes.ts`
- Modify: `web/src/documentApi.ts`
- Modify: `web/src/documentTypes.ts`
- Modify: `web/src/forms.css`
- Modify: `web/src/document-import.css`

- [ ] **Step 1: Add failing review-workflow tests**

Cover import progress, explicit unsupported/partial/truncated/failed states, source anchor navigation, confidence and unresolved items, select-all/select-changes, exact-version conflict, no inferred weights, AI generate/revise degradation, and actual capture-renderer preview.

- [ ] **Step 2: Run tests and confirm failure**

Run: `cd web && npm test -- FormProposalReview.test.tsx FormAIComposer.test.tsx DocumentImportWorkspace.test.tsx`

Expected: FAIL.

- [ ] **Step 3: Implement the Imports-to-Forms journey**

Add **Turn into form template** after successful/partial extraction. Show source and proposal side-by-side on desktop and sequential source/change panels on mobile. Each proposed field displays source location, confidence, unresolved reason and acceptance checkbox. The dominant action is **Create draft from selected fields**.

- [ ] **Step 4: Implement AI generation and revision as diffs**

The composer requires an objective and optional selected source anchors. It states the configured failure/recovery when AI is unavailable. Results display added/changed/removed fields, logic, options and scoring; acceptance posts selected change IDs and refreshes the resulting normal draft.

- [ ] **Step 5: Run web checks and commit**

Run: `cd web && npm test -- FormProposalReview.test.tsx FormAIComposer.test.tsx DocumentImportWorkspace.test.tsx Accessibility.test.tsx copyQuality.test.ts && npm run build`

Expected: PASS.

```bash
git add web/src
git commit -m "feat: review generated form proposals"
```

## Tranche 4 — Vendor refresh, governed application and supersession

### Task 18: Freeze held values and compose focused assessment distributions

**Files:**
- Create: `migrations/000057_vendor_form_refresh.up.sql`
- Create: `migrations/000057_vendor_form_refresh.down.sql`
- Create: `internal/thirdparty/record_target.go`
- Create: `internal/thirdparty/record_target_test.go`
- Create: `internal/thirdparty/assessment_scope.go`
- Create: `internal/thirdparty/assessment_scope_test.go`
- Modify: `internal/thirdparty/assessment_model.go`
- Modify: `internal/thirdparty/assessment_request.go`
- Modify: `internal/thirdparty/assessment_request_test.go`
- Modify: `internal/thirdparty/assessment_repository.go`
- Modify: `internal/thirdparty/assessment_memory.go`
- Modify: `internal/thirdparty/assessment_postgres.go`
- Modify: `internal/evidence/model.go`

- [ ] **Step 1: Add failing target-catalog, baseline and dependency-closure tests**

Test the initial allowlist (`VENDOR.IDENTITY.LEGAL_NAME`, `TRADING_NAME`, `REGISTRATION_REFERENCE`, `JURISDICTION`, `REGISTERED_ADDRESS`, `WEBSITE_DOMAIN`, and bounded `VENDOR.DOCUMENT.<TYPE>`), wrong subject rejection, exact vendor/document version baseline, selected-field condition dependencies, sensitive-value masking, full-scope compatibility and immutable selection after send.

- [ ] **Step 2: Run focused tests and confirm failure**

Run: `go test ./internal/thirdparty ./internal/evidence -run 'RecordTarget|AssessmentScope|Baseline'`

Expected: FAIL.

- [ ] **Step 3: Extend assessment and request snapshots**

Migration `000057` adds `selected_field_ids jsonb NOT NULL DEFAULT '[]'`, `scope_kind` (`FULL`/`FOCUSED`) and `scope_version` to assessments; adds `supersedes_document_id` and `SUPERSEDED` support to third-party documents; and creates `third_party_response_application_receipts` plus due-document indexes. Existing assessments default to full scope.

```go
type RecordBaseline struct {
	TargetKey, SubjectType, SubjectID, RecordID, DisplayValue, SourceLabel string
	RecordVersion int64
	ObservedOrConfirmedAt time.Time
	ExpiresAt *time.Time
}
```

Add `CollectionIntent`, `RecordTarget` and `RecordBaseline` to `evidence.Field`. Baselines are serialized into the request snapshot; they are not re-resolved during response display.

- [ ] **Step 4: Implement the target catalog**

```go
type RecordTargetResolver interface {
	Resolve(context.Context, Actor, Aggregate, formcontract.RecordTarget) (evidence.RecordBaseline, error)
}
```

Use an explicit map from target key to bounded reader and value type. The browser never supplies table/column names or an unvalidated vendor/document ID. Document targets select the one exact current validated/expired document by relationship and document type; ambiguity fails visibly.

- [ ] **Step 5: Compose selected fields plus their dependencies**

For focused scope, include selected fields, every field referenced by their conditions and their containing sections. Reject unknown/stale keys and empty focused scopes. Freeze baselines before transaction commit. Compose a normal distribution with internal/external To and CC recipients; replace the old external-only `SendAssessmentRequestInput` adapter without removing it until migration acceptance passes.

- [ ] **Step 6: Run third-party/evidence tests and commit**

Run: `go test ./internal/thirdparty ./internal/evidence`

Expected: PASS.

```bash
git add migrations/000057_vendor_form_refresh.* internal/thirdparty internal/evidence
git commit -m "feat: scope vendor refresh forms"
```

### Task 19: Apply accepted identity changes and supersede documents atomically

**Files:**
- Create: `internal/thirdparty/assessment_application.go`
- Create: `internal/thirdparty/assessment_application_test.go`
- Create: `internal/thirdparty/assessment_application_postgres_test.go`
- Create: `internal/thirdparty/document_supersession.go`
- Create: `internal/thirdparty/document_supersession_test.go`
- Modify: `internal/thirdparty/service.go`
- Modify: `internal/thirdparty/postgres.go`
- Modify: `internal/thirdparty/assessment_document.go`
- Modify: `internal/thirdparty/assessment_postgres.go`
- Modify: `internal/thirdparty/assessment_review.go`
- Modify: `internal/httpapi/third_party_assessment_handlers.go`
- Modify: `internal/httpapi/third_party_assessment_handlers_test.go`
- Modify: `api/runtime.openapi.json`

- [ ] **Step 1: Add failing review/application tests**

Cover baseline/submitted/current comparison, per-field accept/reject, stale vendor version, unauthorized reviewer, missing current route, complete identity command assembly, receipt idempotency, event/outbox atomicity, exact document supersession, new artifact unavailable, wrong document type, already superseded baseline and reconstruction of both document versions.

- [ ] **Step 2: Run focused tests and confirm failure**

Run: `go test ./internal/thirdparty ./internal/httpapi -run 'AssessmentApplication|DocumentSupersession|ApplyResponse'`

Expected: FAIL.

- [ ] **Step 3: Add explicit application command types**

```go
type ApplyAssessmentResponseInput struct { ExpectedAssessmentVersion, ExpectedSubmissionRevision int64; Decisions []FieldApplicationDecision }
type FieldApplicationDecision struct { FieldID string; Decision string; Rationale string }
type ResponseApplicationReceipt struct {
	ID, AssessmentID, DistributionID, ResponseRevisionID, VendorID, ActorPrincipalID string
	AcceptedFieldIDs, RejectedFieldIDs []string
	PriorVendorVersion, ResultVendorVersion int64
	AppliedAt time.Time
}
```

The HTTP body contains field IDs, decisions, rationale and expected versions only. Target keys, prior versions, vendor identity and actor come from the stored response/baseline and verified command context.

- [ ] **Step 4: Refactor vendor identity persistence into a transaction helper**

Extract the existing `UpdateVendorIdentity` PostgreSQL mutation into `updateVendorIdentityTx(ctx, tx, record)` and retain the current public repository method as a wrapper. `ApplyAssessmentResponse` re-reads the response revision, baselines, current vendor and current authority; assembles one full `UpdateVendorIdentityRecord`; updates the vendor; stores the application receipt; appends event/outbox; and commits once. A derived calculation after commit cannot turn the committed command into an error response.

- [ ] **Step 5: Implement document supersession in the same review transaction**

Validation of a replacement names `supersedes_document_id` from the frozen baseline. Lock both rows, require the old row to remain the current document for relationship/type, require the new artifact to be `AVAILABLE`, set the old row `SUPERSEDED`, insert/validate the replacement, append event/outbox and retain both artifacts. Rejected or merely expired documents are not marked superseded without an exact validated replacement.

- [ ] **Step 6: Add material application route**

```text
POST /api/v1/vendor-assessments/{id}/responses/{revision_id}/apply
```

Classify it as a material command with current reviewer/owner authority and `noActorField`. Return the immutable receipt and refreshed assessment review. Version conflict returns the new held/current comparison rather than overwriting.

- [ ] **Step 7: Run tests and commit**

Run: `go test ./internal/thirdparty ./internal/httpapi`

Expected: PASS.

```bash
git add internal/thirdparty internal/httpapi api/runtime.openapi.json
git commit -m "feat: apply governed vendor response changes"
```

### Task 20: Detect stale facts and expiring documents deterministically

**Files:**
- Create: `internal/thirdparty/refresh_maintenance.go`
- Create: `internal/thirdparty/refresh_maintenance_test.go`
- Create: `internal/thirdparty/refresh_maintenance_postgres.go`
- Create: `internal/thirdparty/refresh_maintenance_postgres_test.go`
- Modify: `internal/thirdparty/assessment_jobs.go`
- Modify: `internal/thirdparty/assessment_jobs_memory.go`
- Modify: `internal/thirdparty/assessment_jobs_postgres.go`
- Modify: `cmd/worker/services.go`
- Modify: `cmd/worker/services_memory.go`
- Modify: `cmd/worker/services_postgres.go`
- Modify: `internal/platform/config/config.go`
- Modify: `.env.example`

- [ ] **Step 1: Add failing maintenance tests**

Test exact midnight/date expiry, configurable lead time, stale fact confirmation interval, keyset/bounded claims, lease expiry, dedupe on relationship/target/version/window, event/outbox atomicity, retry/dead-letter, no automatic recipient choice/send, and an approved Automation Policy requirement for automatic dispatch.

- [ ] **Step 2: Run tests and confirm failure**

Run: `go test ./internal/thirdparty ./cmd/worker -run 'RefreshMaintenance|ExpiryAttention'`

Expected: FAIL.

- [ ] **Step 3: Add the bounded maintenance contract**

```go
type RefreshCandidate struct { Scope; RelationshipID string; TargetKeys []string; Reason string; ObservedVersions map[string]int64 }
type RefreshMaintenancePolicy struct { BatchSize int; Lease time.Duration; DocumentLead time.Duration; FactConfirmationInterval time.Duration }
func (m *RefreshMaintainer) RunBatch(context.Context) (RefreshBatchReceipt, error)
```

Claim at most the configured batch with `FOR UPDATE SKIP LOCKED`. Mark elapsed validated documents expired and append the material event/outbox in the same transaction. Create owner attention containing only target keys/reasons/versions. Do not select an external address or issue a route.

- [ ] **Step 4: Wire the existing worker lifecycle**

Use the current timer/outbox/inbox leasing, failure and dead-letter paths. Add configuration for bounded batch, cadence, document lead and fact interval with production validation. Do not create a document-specific scheduler.

- [ ] **Step 5: Run tests and commit**

Run: `go test ./internal/thirdparty ./cmd/worker`

Expected: PASS.

```bash
git add internal/thirdparty cmd/worker internal/platform/config .env.example
git commit -m "feat: detect vendor refresh needs"
```

### Task 21: Deliver the end-to-end vendor refresh and review UI

**Files:**
- Modify: `web/src/vendorAssessmentTypes.ts`
- Modify: `web/src/vendorAssessmentApi.ts`
- Modify: `web/src/vendorAssessmentApi.test.ts`
- Modify: `web/src/components/VendorDueDiligence.tsx`
- Modify: `web/src/components/VendorDueDiligence.test.tsx`
- Create: `web/src/components/forms/HeldValueField.tsx`
- Create: `web/src/components/forms/HeldValueField.test.tsx`
- Create: `web/src/components/forms/VendorResponseReview.tsx`
- Create: `web/src/components/forms/VendorResponseReview.test.tsx`
- Modify: `web/src/components/capture/CaptureFieldControl.tsx`
- Modify: `web/src/components/capture/CaptureReview.tsx`
- Modify: `web/src/components/VendorsWorkspace.tsx`
- Modify: `web/src/components/VendorsWorkspace.test.tsx`
- Modify: `web/src/components/vendor-due-diligence.css`

- [ ] **Step 1: Add failing complete-journey tests**

Cover starting onboarding with the active form picker; the no-active-template starter-draft recovery; full/focused refresh; stale facts and expired documents preselected; recipient roles/access policy/dates; held-value confirm/correct; replacement upload; final assertion review; bank held/submitted/current comparison; selective accept/reject; stale conflict; application receipt; superseded document link and one dominant next action per state.

- [ ] **Step 2: Run tests and confirm failure**

Run: `cd web && npm test -- VendorDueDiligence.test.tsx HeldValueField.test.tsx VendorResponseReview.test.tsx VendorsWorkspace.test.tsx`

Expected: FAIL.

- [ ] **Step 3: Replace the old external-email send form with distribution composition**

Reuse `DistributionComposer` in vendor context with the selected relationship/assessment already bound. When no active due-diligence template exists, show the searched legal-entity population plus **Use a starter template** and **Open Forms**. Starter use creates a governed draft and takes the maker to its review/approval journey; production never silently activates it. Use real date/calendar controls for review due time, response deadline and link expiry. Preview the exact selected fields, baselines and recipient population before **Send due-diligence form**.

- [ ] **Step 4: Render confirmation, correction and replacement fields**

`HeldValueField` shows value, source/freshness and **Confirm this is accurate** / **Update this information**. A replacement field shows document type/reference/issuer/expiry and the existing bounded vendor-document upload. Final review groups confirmed values, proposed changes, new files and replacements without implying that submission updates the vendor.

- [ ] **Step 5: Implement the bank review comparison**

Show frozen request value/version, submitted value/provenance, current held value/version and achieved assurance. Each eligible field has accept/reject plus rationale. The primary apply action remains disabled until conflicts and required document decisions are resolved. After apply, show receipt version/time/actor and links to prior/current documents.

- [ ] **Step 6: Run web checks and commit**

Run: `cd web && npm test -- VendorDueDiligence.test.tsx HeldValueField.test.tsx VendorResponseReview.test.tsx VendorsWorkspace.test.tsx Accessibility.test.tsx copyQuality.test.ts && npm run build`

Expected: PASS.

```bash
git add web/src
git commit -m "feat: complete vendor refresh workflow"
```

## Final synchronization and proof

### Task 22: Synchronize documentation, render every material state and verify deployment

**Files:**
- Modify: `README.md`
- Modify: `DESIGN.md`
- Modify: `docs/implementation-plan.md`
- Modify: `docs/product/respond-and-capture.md`
- Modify: `docs/architecture/source-evidence-and-secure-capture.md`
- Modify: `docs/architecture/evidence-recipient-boundary.md`
- Modify: `docs/architecture/document-import-resource-and-durability-boundary.md`
- Modify: `docs/architecture/durable-schema-ownership.md`
- Modify: `docs/quality/acceptance-tests.md`
- Modify: `docs/quality/release-gates-and-traceability.md`
- Modify: `docs/quality/rendered-ui-evidence.md`
- Create: `internal/integration/form_system_scale_test.go`
- Modify: `web/src/copyQuality.test.ts` only when the complete workflow review identifies a reliably detectable narration defect
- Modify: `web/scripts/review-ui-flow-manifest.mjs`
- Create: `web/scripts/capture-forms-evidence.mjs`

- [ ] **Step 1: Update canonical truth before claiming completion**

Document promoted Forms ownership, compatibility routes, one request per To recipient plus shared workspace, access assurance modes, OTP limits, encrypted device recovery, communication configuration, extraction statuses/adapters/license gate, vendor baselines/application receipts/supersession and deterministic refresh. Revise the current external-connectivity guidance to permit only policy-bound encrypted browser recovery for eligible scalar fields, with sensitive fields excluded and the server workspace remaining authoritative. Remove the obsolete statements that invitations are always one-time, capture drafts are always session-owned, or external recovery is unavailable.

- [ ] **Step 2: Add permanent end-to-end acceptance fixtures**

The flow manifest must render and interact with: template library empty/list/search/saved-filter/recent/bulk-action; draft/pending/active/retired revisions; invalid/valid compliance weights; import pending/partial/truncated/failed/proposal; distribution compose/delivered/fallback/amended/rotated/superseded/revoked; direct link/shared OTP/direct OTP; OTP expired/exhausted; server-saved/device-only/conflict/recovered/file-reselection; first/amended response; vendor confirm/correct/replace/review/conflict/applied; and desktop/mobile/200%-zoom/light/dark states.

Add a PostgreSQL scale fixture with at least 1,000 legal-entity templates, 400 distributions, multiple recipients and response revisions. Assert keyset page size/cursor stability, tenant/entity filtering before `LIMIT`, exact-ID response lookup, bounded reminder/maintenance claims and expected index usage with normalized `EXPLAIN` assertions. Add point-in-time reconstruction fixtures for template, communication, distribution, response, application receipt and superseded document history. Document production cardinality, partition/retention thresholds and the owners of purge/archive jobs; do not introduce deletion until approved retention policy exists.

- [ ] **Step 3: Run backend verification without local Docker installation**

Run:

```powershell
$clearSightGoFiles = rg --files cmd internal -g '*.go'
gofmt -w $clearSightGoFiles
go test ./...
go vet ./...
```

Expected: PASS. PostgreSQL migration rollback/reapply and serialized integration tests run in the existing CI database job:

```bash
go test -tags postgres ./...
go test -tags 'postgres postgresintegration' ./internal/integration ./internal/evidence ./internal/monitoring ./internal/documentimport ./internal/thirdparty
```

Expected: PASS on the exact final commit.

- [ ] **Step 4: Run frontend, copy, accessibility and production checks**

Run:

```bash
cd web
npm test
npm run typecheck
npm run build
npm run review:ui
```

Expected: PASS with no blocking axe/copy/runtime/flow findings.

- [ ] **Step 5: Inspect rendered evidence and fix the highest-impact defect**

Run: `cd web && node scripts/capture-forms-evidence.mjs`

Inspect the produced desktop, 200%-zoom and mobile captures in light/dark mode. Record the before-state baseline, defect found, correction and re-rendered result in `docs/quality/rendered-ui-evidence.md`. Re-run affected tests after the correction.

- [ ] **Step 6: Commit the synchronized documentation and proof**

```bash
git add README.md DESIGN.md docs internal/integration/form_system_scale_test.go web/scripts web/src/copyQuality.test.ts
git commit -m "docs: verify governed form system"
```

- [ ] **Step 7: Push main and verify CI deployment**

Run: `git status --short --branch && git log -1 --oneline && git push origin main`

Expected: clean `main`, push accepted. Wait for the repository's existing CI/deployment workflow; do not run Docker locally.

- [ ] **Step 8: Smoke-test the deployed host**

At `https://clearsight.cloudspacetechs.com/`, verify the exact deployed commit and complete one live/reference journey for template creation/approval, vendor distribution, shared-link OTP, device recovery, response amendment, bank review/application and document supersession. Confirm revoked/expired/unauthorized links fail without record disclosure. Record deployment time, commit, role, browser/viewport and results in the release evidence.

## Plan self-review

- **Spec coverage:** Tasks 1–5 cover canonical templates, the no-active-template starter recovery, saved/scale-ready discovery, reusable sections, builder parity and compliance weights; Tasks 6–12 cover distributions, access assurance, OTP, communications, amendments, route rotation/supersession and recovery; Tasks 13–17 cover bounded extraction, legacy XLS conversion, MultiXtract/Simplify-Docx pattern reuse, optional PyMuPDF licensing gates and AI diffs; Tasks 18–21 cover held data, selected refresh, identity application, document supersession and maintenance; Task 22 covers documentation, scale/reconstruction checks, rendered evidence, CI deployment and hosted smoke testing.
- **No duplicate stores:** Existing form templates, requests, submissions, artifacts, import jobs, authority routing, worker/outbox and vendor identity commands remain canonical. New tables represent only the missing distribution, access, workspace, communication, proposal and application-receipt concepts.
- **Type consistency:** `AccessPolicy`, `AccessAssurance`, `CollectionIntent`, `RecordTarget`, `RecordBaseline`, `FormTemplateProposal`, `FormDistribution` and workspace versions retain the same names across backend, HTTP and TypeScript tasks.
- **Safety boundary:** All material commands derive actor/tenant/entity from verified context, re-evaluate current authority and write authoritative state/event/outbox/receipt together. Link possession is never labelled identity verification. Device recovery never authorizes, submits or stores protected binary/secret material.
- **Execution constraint:** Execute inline using `superpowers:executing-plans`; do not dispatch subagents. Do not install or start Docker locally. Use existing local unit tooling and CI for PostgreSQL/deployment proof.
