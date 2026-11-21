# Issue Workspace Reuse Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make program issues immediately legible and actionable while reusing the current activity, action, form-distribution, vendor-work, loss and report systems.

**Architecture:** Integrate the completed loss/risk refinement first. Enrich bounded issue summaries and exact issue reads with resolved presentation context, then compose that context in the current issue workspace rather than introducing a second record model. Reuse generic employee distributions and vendor-work distributions, persist an optional form origin for issue-created drafts, and generate a matter-scoped board brief through the existing report-run lifecycle.

**Tech Stack:** Go, PostgreSQL/pgx, React, TypeScript, Vitest, existing report worker and object storage.

---

## File structure

| Path | Responsibility |
| --- | --- |
| internal/continuity/summaries.go | Stable bounded issue-summary presentation contract. |
| internal/continuity/summaries_memory.go | In-memory parity for summary presentation and pagination. |
| internal/continuity/summaries_postgres.go | Single indexed PostgreSQL summary query with organisation and owner labels. |
| internal/oploss/model.go and internal/oploss/list_postgres.go | Exact matter_id list filter for linked loss summaries. |
| internal/monitoring/model.go and form-library files | Immutable issue origin on a normal form draft/revision. |
| internal/reporting/matter_board_brief*.go | One bounded matter-board-brief projection and deterministic PDF renderer. |
| web/src/components/MatterRecordWorkspace.tsx | Issue record composition only; no new command workflow. |
| web/src/components/MatterRecordContext.tsx | Fixed four-fact rail and overview context. |
| web/src/components/MatterCollectionsPanel.tsx | Existing form remediation, employee distribution and vendor work in one task-oriented tab. |

### Task 1: Integrate the reusable loss and risk refinement

**Files:**
- Modify: branch history only; merge origin/feat/346-s5-loss-risk-refinement into a branch based on current origin/main.
- Verify: internal/oploss/**, internal/risk/**, web/src/components/losses/**, web/src/components/risks/**, web/src/App.tsx.

- [ ] **Step 1: Create an isolated integration branch from current main**

    git fetch --prune
    git switch -c codex/issue-workspace-reuse origin/main
    git merge --no-ff origin/feat/346-s5-loss-risk-refinement

Expected: a merge commit retains dde462957 report-download ancestry and the loss/risk branch commits. Do not add or remove the user’s untracked .pptx files.

- [ ] **Step 2: Resolve only verified overlap**

Keep report-download authorization changes from 46ab7fdbb, 973ddabf5, and 46b7f848b; keep named loss/risk ownership, organisation labels, recovery authoring, and current-position work from the feature branch. Do not replace either side wholesale in web/src/App.tsx or HTTP handlers.

- [ ] **Step 3: Run the imported behavior checks**

    go test ./internal/oploss ./internal/risk ./internal/reporting
    npm --prefix web test -- src/components/losses/LossRecord.test.tsx src/components/losses/LossRegister.test.tsx src/components/risks/RiskRecord.test.tsx
    npm --prefix web run typecheck

Expected: exit code 0 from each command.

- [ ] **Step 4: Commit the integration boundary**

    git add internal/oploss internal/risk web/src/App.tsx web/src/App.test.tsx web/src/components/losses web/src/components/risks web/src/lossApi.ts web/src/lossTypes.ts web/src/riskTypes.ts
    git commit -m "feat: integrate loss and risk refinement"

### Task 2: Add bounded issue presentation context to summaries and exact reads

**Files:**
- Modify: internal/continuity/summaries.go
- Modify: internal/continuity/summaries_memory.go
- Modify: internal/continuity/summaries_postgres.go
- Modify: internal/continuity/current_postgres.go
- Modify: internal/continuity/summaries_test.go
- Modify: internal/continuity/current_read_postgres_integration_test.go
- Modify: web/src/summaryTypes.ts
- Modify: web/src/types.ts
- Modify: web/src/components/MattersWorkspace.tsx
- Modify: web/src/components/ProgramIssuesPanel.tsx
- Test: web/src/components/MattersWorkspace.test.tsx
- Test: web/src/components/ProgramIssuesPanel.test.tsx

- [ ] **Step 1: Write failing summary and exact-read tests**

Add fixtures with a Matter that has an organization_scope_id, stored owner and due date. Assert the returned summary and exact aggregate expose a presentation object without a raw UUID:

    if got := page.Items[0].Presentation; got.OrganizationPath != "Technology / Identity" || got.OwnerDisplayName != "Hakeem Bello" {
        t.Fatalf("matter presentation = %#v", got)
    }

Assert that a restricted actor receives neither the Matter nor its labels, and that the cursor is still generated from priority, timestamp and ID only.

- [ ] **Step 2: Run the focused tests and confirm the missing contract**

    go test ./internal/continuity -run "MatterSummary|CurrentRead"

Expected: failure because MatterPresentation does not exist.

- [ ] **Step 3: Add one reusable read model**

In internal/continuity/summaries.go, define and embed this read-only value in both MatterSummary and MatterAggregate:

    type MatterPresentation struct {
        OrganizationPath string `json:"organization_path,omitempty"`
        OwnerDisplayName string `json:"owner_display_name,omitempty"`
        OwnerLabelState  string `json:"owner_label_state"`
    }

Use OwnerLabelState values AVAILABLE, UNASSIGNED, and UNAVAILABLE; do not substitute a principal ID for a missing name. Resolve the organisation path from the effective authorised scope hierarchy and the owner from the stored responsibility/principal relation in the same bounded server-side read.

- [ ] **Step 4: Keep memory and PostgreSQL reads equivalent**

Update summarizeMatter, ListMatterSummaries, and current aggregate decoration so memory and PostgreSQL reads return the same presentation shape. In summaries_postgres.go, use scoped joins for the one current organisation path and stored owner label; retain the existing Matter visibility predicate, filters, ordering and keyset cursor. Do not perform a per-row authority lookup.

- [ ] **Step 5: Expose the contract in both list surfaces**

Add matching presentation types in web/src/summaryTypes.ts and web/src/types.ts. Update MattersWorkspace and ProgramIssuesPanel to render the fixed rail in this order:

    <dl className="matter-summary-facts">
      <div><dt>Applies to</dt><dd>{presentation.organization_path || "Not recorded"}</dd></div>
      <div><dt>Owner</dt><dd>{ownerLabel(summary)}</dd></div>
      <div><dt>Due</dt><dd>{matterDeadlinePresentation(summary.matter.due_at).label}</dd></div>
      <div><dt>Impact</dt><dd>{matterPriorityLabel(summary.matter.priority)}</dd></div>
    </dl>

Keep Affected service or process out of the rail; it belongs to expanded detail.

- [ ] **Step 6: Run focused verification**

    go test ./internal/continuity -run "MatterSummary|CurrentRead|Visibility"
    npm --prefix web test -- src/components/MattersWorkspace.test.tsx src/components/ProgramIssuesPanel.test.tsx
    npm --prefix web run typecheck

Expected: exit code 0; tests prove no raw principal UUID is rendered and no inaccessible summary leaks.

- [ ] **Step 7: Commit the bounded presentation contract**

    git add internal/continuity/summaries.go internal/continuity/summaries_memory.go internal/continuity/summaries_postgres.go internal/continuity/current_postgres.go internal/continuity/summaries_test.go internal/continuity/current_read_postgres_integration_test.go web/src/summaryTypes.ts web/src/types.ts web/src/components/MattersWorkspace.tsx web/src/components/ProgramIssuesPanel.tsx web/src/components/MattersWorkspace.test.tsx web/src/components/ProgramIssuesPanel.test.tsx
    git commit -m "feat: show issue scope owner due date and impact"

### Task 3: Make linked loss and recovery context reusable from an issue

**Files:**
- Create: migrations/000127_operational_loss_matter_list_idx.up.sql
- Create: migrations/000127_operational_loss_matter_list_idx.down.sql
- Modify: internal/oploss/model.go
- Modify: internal/oploss/memory.go
- Modify: internal/oploss/list_postgres.go
- Modify: internal/oploss/service_test.go
- Modify: internal/oploss/postgres_integration_test.go
- Modify: internal/httpapi/operational_loss_handlers.go
- Modify: web/src/lossApi.ts
- Create: web/src/components/matter-context/MatterLinkedLosses.tsx
- Test: web/src/components/matter-context/MatterLinkedLosses.test.tsx

- [ ] **Step 1: Write failing filter tests**

Add a page containing losses for two different matter_id values. Assert ListFilter{MatterID: issueID, Limit: 5} returns only matching losses, preserves recovery totals, and cannot cross tenant or legal-entity scope.

- [ ] **Step 2: Run the new failing tests**

    go test ./internal/oploss -run "MatterID|MatterLink"

Expected: compilation failure because ListFilter.MatterID is absent.

- [ ] **Step 3: Add the exact indexed filter**

Add MatterID string to oploss.ListFilter and this migration:

    CREATE INDEX operational_losses_matter_list_idx
        ON operational_losses (tenant_id, legal_entity_id, matter_id, updated_at DESC, id DESC)
        WHERE matter_id IS NOT NULL;

Apply l.matter_id = NULLIF($n::text, '')::uuid in the existing PostgreSQL list predicate and the equivalent filter in memory. The API accepts matter_id only as a read filter; it is not a command actor or authority input.

- [ ] **Step 4: Render a compact contextual card**

MatterLinkedLosses calls listLosses({ matterID, limit: 5 }), renders gross, recovered, net and recovery status for each authorised match, and opens the canonical loss record through the existing callback. Loading, unavailable and empty states are distinct. It must not expose recovery authoring controls.

- [ ] **Step 5: Verify the end-to-end read**

    go test ./internal/oploss ./internal/httpapi -run "Loss.*Matter|OperationalLoss"
    npm --prefix web test -- src/components/matter-context/MatterLinkedLosses.test.tsx

Expected: exit code 0 and an unavailable loss read leaves issue work available.

- [ ] **Step 6: Commit the linked-loss read**

    git add migrations/000127_operational_loss_matter_list_idx.up.sql migrations/000127_operational_loss_matter_list_idx.down.sql internal/oploss internal/httpapi/operational_loss_handlers.go web/src/lossApi.ts web/src/components/matter-context/MatterLinkedLosses.tsx web/src/components/matter-context/MatterLinkedLosses.test.tsx
    git commit -m "feat: show linked loss recovery in issue records"

### Task 4: Persist a normal form draft’s issue origin and reuse existing distribution routes

**Files:**
- Create: migrations/000128_monitoring_form_origin.up.sql
- Create: migrations/000128_monitoring_form_origin.down.sql
- Modify: internal/monitoring/model.go
- Modify: internal/monitoring/form_library.go
- Modify: internal/monitoring/form_proposal_authoring.go
- Modify: internal/monitoring/postgres.go
- Modify: internal/monitoring/form_library_service_test.go
- Modify: internal/httpapi/forms_handlers.go
- Modify: web/src/monitoringTypes.ts
- Modify: web/src/formsTypes.ts
- Modify: web/src/components/FormBuilder.tsx
- Modify: web/src/components/forms/DistributionComposer.tsx
- Test: web/src/components/FormBuilder.test.tsx
- Create: web/src/components/forms/DistributionComposer.test.tsx

- [ ] **Step 1: Write failing origin-preservation tests**

Create a draft with OriginSubjectType: "MATTER" and a valid Matter ID. Assert draft creation, revision and reload preserve that origin; assert any other origin type, cross-entity Matter, or edited origin on a revision is rejected.

- [ ] **Step 2: Run the failing test**

    go test ./internal/monitoring -run "Form.*Origin"

Expected: compilation failure because a form has no immutable origin.

- [ ] **Step 3: Add an immutable optional origin to normal form revisions**

Add nullable origin_subject_type and origin_subject_id columns to monitoring_form_templates, constrained as an all-or-none pair. Add:

    type FormOrigin struct {
        SubjectType string `json:"subject_type"`
        SubjectID   string `json:"subject_id"`
    }

Allow only MATTER in this feature. Validate the referenced Matter through the verified actor’s tenant and legal entity before writing; copy the origin unchanged into every revision. Do not add a separate form-link table.

- [ ] **Step 4: Preselect the issue from the issue workspace**

Add an optional origin prop to FormBuilder and an optional locked subject prop to DistributionComposer:

    type LockedSubject = { type: "MATTER"; id: string; label: string };

When locked, display the issue label and omit editable subject type/identifier inputs. Existing recipient search continues to resolve internal principals. Do not alter its generic distribution authorization.

- [ ] **Step 5: Keep vendor delivery relationship-scoped**

Do not send a vendor request through a free-text external address. VendorWorkPanel targetType="MATTER" remains the vendor request flow and requires an active VendorRelationshipLink. The issue UI may select between Employee and Vendor, but each branch opens its established component.

- [ ] **Step 6: Verify form behavior**

    go test ./internal/monitoring ./internal/httpapi -run "Form.*Origin|Form.*Scope"
    npm --prefix web test -- src/components/FormBuilder.test.tsx src/components/forms/DistributionComposer.test.tsx src/components/VendorWorkPanel.test.tsx
    npm --prefix web run typecheck

Expected: exit code 0; employee distribution sends subject_type: "MATTER", while vendor delivery requires a linked vendor relationship.

- [ ] **Step 7: Commit form-origin and distribution reuse**

    git add migrations/000128_monitoring_form_origin.up.sql migrations/000128_monitoring_form_origin.down.sql internal/monitoring internal/httpapi/forms_handlers.go web/src/monitoringTypes.ts web/src/formsTypes.ts web/src/components/FormBuilder.tsx web/src/components/forms/DistributionComposer.tsx web/src/components/FormBuilder.test.tsx web/src/components/forms/DistributionComposer.test.tsx
    git commit -m "feat: link form drafts and distributions to issues"

### Task 5: Recompose the issue workspace around existing work and activity

**Files:**
- Create: web/src/components/MatterRecordContext.tsx
- Create: web/src/components/MatterCollectionsPanel.tsx
- Modify: web/src/components/MatterRecordWorkspace.tsx
- Modify: web/src/components/MatterDetailsPanel.tsx
- Modify: web/src/components/matter-context/MatterDomainContext.tsx
- Modify: web/src/matter-record.css
- Modify: web/src/AppViews.tsx
- Modify: web/src/components/MattersWorkspace.tsx
- Test: web/src/components/MatterRecordWorkspace.test.tsx
- Test: web/src/components/MatterActivityTimeline.test.tsx

- [ ] **Step 1: Write failing workspace tests**

Render an issue with presentation context, an affected area, a linked loss, one action and activity. Assert the header shows Applies to, Owner, Due, and Impact; Overview shows the affected service/process; Work retains action controls; Evidence and requests contains existing response/remediation/vendor widgets; Activity has its existing comment/mention composer.

- [ ] **Step 2: Run the focused test and confirm the old tab labels**

    npm --prefix web test -- src/components/MatterRecordWorkspace.test.tsx -t "Applies to|Evidence and requests|Activity"

Expected: failure because the current tabs are Details, Actions, Evidence and Decisions and the header lacks scope/owner.

- [ ] **Step 3: Add small compositional components**

MatterRecordContext receives the aggregate and operations presentation and renders the four fact rail, affected service/process, source context and MatterLinkedLosses.

MatterCollectionsPanel composes, without copying logic:

    <MatterFormRemediationPanel {...remediationProps} />
    <EmployeeMatterDistribution matter={aggregate.matter} />
    <VendorRelationshipLinks targetType="MATTER" targetID={aggregate.matter.id} />
    <VendorWorkPanel targetType="MATTER" targetID={aggregate.matter.id} onOpenRequest={onOpenRequest} />

EmployeeMatterDistribution is a thin focused-sheet wrapper around DistributionComposer with a locked Matter subject. It must not reproduce recipient search, delivery, expiry or notification logic.

- [ ] **Step 4: Rename and order the tabs without changing workflow ownership**

Replace the current tab list with:

    [
      { id: "overview", label: "Overview" },
      { id: "work", label: "Work" },
      { id: "collections", label: "Evidence and requests" },
      { id: "decisions", label: "Decisions" },
    ]

Map MatterActionsPanel to Work, existing decision/outcome panels to Decisions, and retain MatterActivityTimeline in the right rail. The activity component remains the sole comment, @mention and history implementation.

- [ ] **Step 5: Implement responsive replacement behavior**

At desktop widths use a bounded main column plus activity rail. At narrow widths move the activity rail behind an Activity tab after the primary tabs; do not render a narrow second column or duplicate the composer. Preserve focus restoration when a collection/distribution sheet closes.

- [ ] **Step 6: Verify behavior and copy**

    npm --prefix web test -- src/components/MatterRecordWorkspace.test.tsx src/components/MatterActivityTimeline.test.tsx src/components/MatterActionsPanel.test.tsx src/components/MatterFormRemediationPanel.test.tsx src/components/VendorWorkPanel.test.tsx src/copyQuality.test.ts
    npm --prefix web run typecheck

Expected: exit code 0; comments still queue mentions, update requests remain action-scoped, and no enabled control lacks its underlying command.

- [ ] **Step 7: Commit the workspace composition**

    git add web/src/components/MatterRecordContext.tsx web/src/components/MatterCollectionsPanel.tsx web/src/components/MatterRecordWorkspace.tsx web/src/components/MatterDetailsPanel.tsx web/src/components/matter-context/MatterDomainContext.tsx web/src/matter-record.css web/src/AppViews.tsx web/src/components/MattersWorkspace.tsx web/src/components/MatterRecordWorkspace.test.tsx web/src/components/MatterActivityTimeline.test.tsx
    git commit -m "feat: organize issue work around context and activity"

### Task 6: Generate an executive board brief through the report-run lifecycle

**Files:**
- Modify: internal/reporting/model.go
- Modify: internal/reporting/service.go
- Create: internal/reporting/matter_board_brief.go
- Create: internal/reporting/matter_board_brief_postgres.go
- Create: internal/reporting/pdf.go
- Modify: internal/reporting/memory.go
- Modify: internal/reporting/report_postgres.go
- Modify: internal/httpapi/reporting_handlers.go
- Modify: web/src/reportingTypes.ts
- Modify: web/src/reportingApi.ts
- Modify: web/src/components/reports/ReportsWorkspace.tsx
- Modify: web/src/components/MatterRecordWorkspace.tsx
- Test: internal/reporting/matter_board_brief_test.go
- Test: internal/reporting/matter_board_brief_postgres_integration_test.go

- [ ] **Step 1: Write failing report tests**

Create an active MATTER_BOARD_BRIEF definition scoped to one Matter. Assert that a queued run captures a point-in-time source boundary, produces application/pdf, contains the Matter reference/owner/due/impact and linked-loss values, and excludes internal comments plus inaccessible recipient details.

- [ ] **Step 2: Run the failing tests**

    go test ./internal/reporting -run "MatterBoardBrief"

Expected: compilation failure because DatasetMatterBoardBrief and FormatPDF do not exist.

- [ ] **Step 3: Add one report dataset and one output format**

    const DatasetMatterBoardBrief ReportDataset = "MATTER_BOARD_BRIEF"
    const FormatPDF ReportFormat = "PDF"

Allow the dataset only with ScopeMatter; allow PDF only for that dataset. Extend definition validation, run validation, filename extension and content-type handling accordingly. Existing CSV, NDJSON and XLSX behavior must remain unchanged.

- [ ] **Step 4: Build a bounded board-brief projection**

Define MatterBoardBrief in matter_board_brief.go. Its loader resolves one authorised Matter, presentation context, active program links, actions, decisions, outcome state, exact matter_id loss summaries, form/distribution response summaries, vendor work summaries and source/integration context. Each child collection has an explicit limit and a complete flag. Internal comments and protected recipient addresses are never selected.

- [ ] **Step 5: Render a deterministic PDF**

pdf.go renders a fixed-order document from MatterBoardBrief, not a browser print view. Its sections are Issue position, Actions and decisions, Financial impact, Evidence and requests, Connected sources, and Report receipt. Render unavailable, not-recorded and incomplete collection states as explicit text. Store the PDF and manifest through the existing report object lifecycle and keep checksum verification unchanged.

- [ ] **Step 6: Queue from the issue only when an active governed definition exists**

Expose Generate board brief only when the exact Matter has an effective authorised definition. The action queues a standard report run; Reports workspace remains the central download/history location. A missing definition, authority route or report service leaves the action unavailable with an explanation.

- [ ] **Step 7: Verify report and UI behavior**

    go test ./internal/reporting ./internal/httpapi -run "MatterBoardBrief|Report.*Download|Report.*Scope"
    npm --prefix web test -- src/components/MatterRecordWorkspace.test.tsx src/components/reports/ReportsWorkspace.test.tsx
    npm --prefix web run typecheck

Expected: exit code 0; a report run is generated and downloaded through the existing permission-checked endpoint.

- [ ] **Step 8: Commit the board brief**

    git add internal/reporting internal/httpapi/reporting_handlers.go web/src/reportingTypes.ts web/src/reportingApi.ts web/src/components/reports/ReportsWorkspace.tsx web/src/components/MatterRecordWorkspace.tsx web/src/components/MatterRecordWorkspace.test.tsx
    git commit -m "feat: generate issue board briefs through reports"

### Task 7: Document, render and verify the complete workflow

**Files:**
- Modify: DESIGN.md
- Modify: docs/quality/program-matter-acceptance-tests.md
- Modify: docs/superpowers/specs/2026-10-06-issue-workspace-reuse-design.md
- Modify: web/src/copyQuality.test.ts
- Create: web/src/components/MatterRecordWorkspace.responsive.test.tsx

- [ ] **Step 1: Add workflow acceptance fixtures**

Document one accountable-owner journey containing a scoped issue, linked loss with partial recovery, employee form request, linked vendor request, mention notification, update request, and board-brief run. State which source is sample data and which result remains unverified.

- [ ] **Step 2: Add render assertions**

Create desktop and narrow fixtures. The desktop assertion checks one activity rail; the narrow assertion checks one Activity tab and no duplicate activity composer. Both fixtures verify the four header facts and Affected service or process are separately visible.

- [ ] **Step 3: Add narrow copy-quality regressions**

Extend copyQuality.test.ts only for newly introduced forbidden product-narration patterns. Do not broadly match business phrases such as issue, report, evidence, activity, or source.

- [ ] **Step 4: Run the release candidate checks**

    go test ./internal/continuity ./internal/oploss ./internal/monitoring ./internal/reporting ./internal/httpapi
    npm --prefix web test -- src/components/MatterRecordWorkspace.test.tsx src/components/MatterRecordWorkspace.responsive.test.tsx src/components/MatterActivityTimeline.test.tsx src/components/MattersWorkspace.test.tsx src/components/ProgramIssuesPanel.test.tsx src/components/forms/DistributionComposer.test.tsx src/components/VendorWorkPanel.test.tsx src/components/reports/ReportsWorkspace.test.tsx src/copyQuality.test.ts
    npm --prefix web run typecheck
    git diff --check

Expected: exit code 0 from each command and no whitespace errors.

- [ ] **Step 5: Capture rendered evidence and commit documentation**

Capture the approved desktop and narrow issue record states through the repository’s existing UI-evidence workflow. Update DESIGN.md, the acceptance document and specification with final route/fixture names. Then run:

    git add DESIGN.md docs/quality/program-matter-acceptance-tests.md docs/superpowers/specs/2026-10-06-issue-workspace-reuse-design.md web/src/copyQuality.test.ts web/src/components/MatterRecordWorkspace.responsive.test.tsx
    git commit -m "docs: verify issue workspace operating journey"

## Self-review

- The plan covers each approved specification area: remote reuse, list facts, exact issue context, linked loss/recovery, form draft origin, employee/vendor distribution, retained activity, board brief, authority/performance, responsive design and customer-facing copy.
- Employee and vendor delivery remain separate because their existing access models are different; neither introduces a new request model.
- The plan never relies on a browser-side broad population load. Summary pagination, linked loss reads and report collections are scoped and bounded.
- Names and types are defined before later tasks use them: MatterPresentation, LockedSubject, FormOrigin, DatasetMatterBoardBrief, and FormatPDF.
