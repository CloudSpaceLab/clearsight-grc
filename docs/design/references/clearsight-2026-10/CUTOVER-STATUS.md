# Approved UI cutover — runtime reconciliation

> **VISUAL PARITY RESET · 2026-10-19** — **0 of 19 original HTML screens accepted**. This file's historical "Partial" entries refer to feature/contract integration only, not an approved redesign. The 1440px Program, Framework Library and Assessment source/runtime captures have material structural differences; the 390px Program view also diverges. See [verified side-by-side review](VISUAL-PARITY-RESET-2026-10-19.md). Successful CI and 226/226 screenshots prove flow execution and artifact completeness, **not source parity**. Stop counting cosmetic or data-quality fixes as UI cutover. Full cutover is BLOCKED until named screen-by-screen comparison approval.

Updated: 2026-10-21. Tracking: [#462](https://github.com/CloudSpaceLab/clearsight-grc/issues/462).
Authority: the immutable [approved source archive](./source/README.md), its [manifest](./source/MANIFEST.json), the [implementation mapping](./IMPLEMENTATION-MAPPING.md), the unified design specification and the [21 October navigation adaptation decision](../../decisions/2026-10-21-v2-navigation-adaptation.md). The newer user instruction supersedes the archived specification's prohibition on adapting the global menu. Source HTML is a **visual/interaction benchmark, not a data or permission contract**. No source-example totals, statuses, suggestions or approvals may ship as runtime facts.

## 21 October — shared V2 menu adaptation

**Status: partial V2 layout integration; 0/19 whole screens accepted.** The implementation adapts the menu itself to the common source order: Home, Frameworks, Programs, Forms, Vendors, Risk & Losses, Privacy, Work & Approvals, Reports, with permission-gated Settings in the footer. It uses existing routes, separates Frameworks from Programs selection, replaces the generic Portfolio lens, and provides Risk & Losses contextual destinations without a second risk or reporting engine.

The shared rail target is 228px above 1024px, 72px at 561–1024px and 56px at 560px or below. A labelled compact-menu trigger reuses `FocusedSheet`; there is no duplicate mobile bar. The decision records the exact source paths/hashes, source label and breakpoint conflicts, current-route mappings and required verification. The archived source files and manifest remain unchanged.

The [implementation review](../../decisions/2026-10-21-v2-navigation-adaptation.md#implementation-review--21-october-2026) records the named AI review and exact implementation checkpoint `32ac98ce80f21424904b38a57ef5e31443cac2cd`: 14 passing menu states, five source pairs, short-height/boundary/resize checks and six scoped axe runs with zero violations. This verifies the recorded shared-menu adaptation, not complete specialist screens. Final-head CI and artifact review remain PR gates; four-actor execution, backend authorization and source-to-report receipts remain separate release gates. The [Reports #578 receipt](../../decisions/2026-10-21-reports-v2-layout-review.md) remains a partial Overview/library adaptation, not full-screen acceptance.

## 21 October — Reports Explore and Builder composition

The [next Reports tranche](../../decisions/2026-10-21-reports-explore-builder-v2.md) adapts both remaining source layouts and their local menu: a selected-measure Data rail/chart/table in Explore, and setup/paper/output in Builder with the original responsive rearrangements. Build report is contextual and its draft survives local navigation. Existing saved-definition governance and generation commands remain in use. Exact source/runtime pairs, named visual review and final CI are tracked in that decision. **Partial V2 layout integration; 0/19 remains unchanged.** This does not establish hosted actor execution or source-to-report reconciliation.

## Design fidelity register — 18 October 2026

The reference artifact, **not the existing React screen**, defines each view's approved
information hierarchy, actions, responsive anatomy and interaction states.
The [source manifest](./source/MANIFEST.json) pins SHA-256 for all 96 original files.
The [21 October navigation decision](../../decisions/2026-10-21-v2-navigation-adaptation.md)
now resolves conflicting prototype menu variants; the archived unified specification remains unchanged. Sample records, scores and HTML-only
actions are *fixtures* and may not be promoted into production data or authority.

**Cutover means exact UI composition and real workflows, not incremental beautification.**
Every screen must have (a) approved-HTML component/interaction inventory, (b) React owner
and real API/authority wiring, (c) reference/runtime images at 1440/1280/1024/768/390,
light and dark, (d) loading, empty, stale, partial, denied and conflict verification,
(e) focused action round-trip, keyboard/focus and reduced-motion checks.
A green screenshot job without a **side-by-side reference review** is *not* visual parity.

| Approved screen (source under `source/`) | Runtime composition / cutover gate | Verified status |
| --- | --- | --- |
| Control Tower — `clearsight-designs-01-05/` | Existing Home/Oversight, four genuine scoped measures, activity and Attention, shared shell | Partial; full design comparison and actor proof outstanding |
| Framework Library — `clearsight-framework-designs/framework-library.html` | Edition register/inspector, source rights, filters, governed selection | Partial; #497/#498/#505/#507/#508/#509/#513 merged, global catalog search and full parity outstanding |
| Framework Adoption — `clearsight-framework-designs/framework-adoption.html` | Program & scope → requirement links → questionnaire → independent adoption review | Partial; #511/#512/#514 merged; no complete four-stage screen or global Form association |
| Program — `clearsight-program-designs/program-workspace-preview.html` | **Six** tabs: Overview / Requirements / Assessments / Evidence / Issues / Activity; Data collection contextual | Partial; #516/#517/#519/#520/#521 merged; #522 Activity review-digest in CI; global layout/source-to-runtime parity outstanding |
| Assessment — `clearsight-designs-01-05/assessment-workspace-preview.html` | Three-column sections / form / evidence-readiness; mobile section rail; exact XLSX mapping | Partial; #523 adds governed CaptureForm wizard section rail. Source-backed third-column guidance, import/mapping and 1440/390 parity still outstanding |
| Vendor — `clearsight-vendor-management-designs/vendor-workspace-preview.html` | Relationship register, due diligence, expiring evidence and approval | Foundation only; parity/flows outstanding |
| Operational Risk — `clearsight-operational-risk-designs/operational-risk-preview.html` | RCSA, indicator and loss have distinct semantics and authorized scope | Foundation only |
| Privacy — `clearsight-privacy-data-designs/privacy-workspace-preview.html` | Processing register, DPIA, purpose-specific reviews | Foundation only |
| Reports — `clearsight-reports-insights-designs/reports-workspace-preview.html` | Report definitions, persisted runs, source parity, authorized download digest | Foundation only; live four-actor report proof missing |
| Work/Approvals — `clearsight-work-approvals-designs/work-approvals-preview.html` | Existing queues + independent decisions/focused record | Foundation only |
| Controls/Policies — `clearsight-controls-policies-designs/controls-policies.html` | Existing control/policy lifecycle and versioned decisions | Foundation only |
| Enterprise Setup — `clearsight-enterprise-setup-designs/enterprise-setup.html` | Real hierarchy, branches/departments/group scope | Foundation only |
| Integrations — `clearsight-integrations-designs/integrations-workspace-preview.html` | Source connections, reviewable mappings and run health | Foundation only |
| Search — `clearsight-search-command-center-designs/search-command-center-preview.html` | Authorized indexed/bounded query and exact result inspector | Blocked on authoritative federated search contract |
| Notifications — `clearsight-notifications-activity-designs/notifications-activity-preview.html` | Actor-safe inbox, state and receipts | Foundation only |
| Audit/Assurance — `clearsight-audit-assurance-designs/audit-assurance.html` | Plans, fieldwork and evidence lineage | Foundation only |
| People/Access — `clearsight-people-access-designs/people-access-preview.html` | Identity, positions, delegations and effective authority | Foundation only |
| Automation — `clearsight-automation-governance-designs/automation-governance.html` | Draft, simulate, review, approve, activate and rollback | Foundation only |
| Copilot — `clearsight-contextual-copilot-designs/ai-copilot.html` | Three or fewer human-chosen, source-grounded suggestions | Foundation only; no fabricated answers |

**Program design discrepancy confirmed:** the older runtime combined submitted forms and evidence
under “Evidence & results,” placed “Data collection” in the primary tablist, and lacked a distinct
Assessments destination. The approved [Program source](./source/clearsight-program-designs/program-workspace-preview.html)
and [design notes](./source/clearsight-program-designs/program-workspace-design-notes.md)
require the six primary tabs above. The Program cutover tranche separates existing
`ProgramResponsesPanel` (Assessments) and `ProgramEvidencePanel` (Evidence)
without inventing backend statuses or an assessment system; `/monitoring` remains a
contextual route for existing deep links and authorized collection actions.

**October 18 source-vs-runtime visual inspection:** compared the approved Program HTML rendering at 1440px with the generated runtime Program Overview and Activity screenshot artifacts. The runtime's top-level Program header, shell scale, score-card hierarchy and spacing still differ materially from the approved composition. The Activity evidence script intentionally scrolls to the selected section (211px desktop; 298px mobile), so its clipped page header is not proof of a runtime clipping bug. Capture separate top-of-page and section-focused reference pairs before claiming visual parity.

**Release blocking:** full screenshot/geometry comparisons, forms/assessment desktop-three-column
composition, four-actor workflow, authorized report-to-source parity and cross-tenant/LE/group
denial receipts. Issue #462 stays open until these are verified, not inferred from CI.

## What is integrated

- **#498:** approved design source, hashes and mapping in `main` (reference-only).
- **#497:** bounded Framework Library, Program adoption browser, authorized catalog cursor/source inspection, route handoff, tests. This is not the full four-stage adoption experience.
- **#499:** Program Overview composition, lifecycle label and review/current-position separation.
- **#500:** four Program record-backed overview measures; stale/unknown/partial state and exact-section drill, with readable metric headings.
- **#501:** Program record title is the single detail-page heading; register and Framework Library headings remain.
- **#495:** four-actor/report acceptance **plan**; not hosted execution.
- **#502:** active-standard requirement/form follow-through proposed; evaluate its exact-head checks before integration.
- **#515/#516/#517/#519/#520/#521:** scoped Program safeguards, six-section workspace, Overview attention, recent assessments, compact mobile navigation and exact submitted-response review merged.
- **#522:** Program Activity from the per-actor governed review digest. Current state is a draft PR undergoing exact-head CI; not a complete chronological event stream.
- **#523:** Assessment wizard section rail implemented using the existing capture form, draft save, validation and response-review contracts; pending CI and visual acceptance.

These merges establish a tested partial runtime migration, **not complete approved-HTML visual parity or production release acceptance**.

## Screen-by-screen reconciliation

“Foundation” means existing domain/UI owners were identified; it does **not** mean approved visual parity or the journey has passed. Prioritize existing shared components and backed routes; do not add a second work queue, audit engine, scoring engine, form library or approval service. Exposing an existing route in the shared V2 menu is required by the current navigation decision and does not authorize a parallel workspace or unsupported capability.

| Approved source area | Current runtime owner or contract | Remaining specific cutover gate |
| --- | --- | --- |
| Unified visual shell | Existing `App.tsx`, `AppViews.tsx`, enterprise shell and shared design-system tokens | Shared V2 menu adaptation verified in the 21 October review: order, real routes, active state, responsive rail, compact-menu focus and denied Settings. Specialist local menus, full screen composition and operational acceptance remain open |
| Framework Library | `FrameworkLibraryWorkspace`, `useFrameworkEditionCatalog`, #437 | Authenticated server-side edition search/direct lookup and rights/provenance; approved catalog screenshots against current desktop/mobile |
| Framework adoption | `ProgramStandardsPanel`, `ProgramRequirementsPanel`, Program maker/checker commands | Program/scope → clause links → **existing** Form → governed review journey; versioned source and actor acceptance; never adopt from a catalog selection alone |
| Program workspace | `ProgramRecordWorkspace`, `ProgramCurrentPosition`, `programOverviewMetrics` and #433 | Approved first-viewport proportions, original tab/action hierarchy, source-backed assessed/support/gap populations when available; no invented counts |
| Assessment / questionnaire | `FormsWorkspace`, `ResponseAssessment`, `ResponseAnswerSheet`, `DocumentImportWorkspace` | Section/form/evidence review composition, per-answer lineage, spreadsheet blank preservation, independent assessment and reviewed source |
| Vendor management | `VendorsWorkspace`, vendor assessment and form/evidence contracts | Register/relationship view parity; DPO gate, expiry and maker-checker decision; exact mobile record actions |
| Operational risk (RCSA, KRI, losses) | `RCSAWorkspace`, `RisksWorkspace`, `LossesWorkspace`, `InsightsWorkspace` | Period-vs-as-of semantics; currency/FX and recovery reconciliation; scope-backed risk and KRI drill; no synthetic percentages |
| Privacy & data | `RopaRegisterPage`, `RopaActivityPage`, existing forms/vendor/work contracts | DPIA screening vs approval, privacy vendor gate, breach/DSR lifecycle evidence and ROPA source-backed reviews |
| Reports & Insights | `ReportsWorkspace`, `InsightsWorkspace` and #495 acceptance plan | Approved report builder/explore composition, authorization, saved report/source snapshot parity and byte digests |
| Work & approvals | `AssignedWorkQueue`, `MattersWorkspace`, `EvidenceWorkspace`, governed authority | One contextual queue; approvals/independent review and negative authorization checks; no second task database |
| Controls & policies | Program controls/evidence, policy and authority components | Governed policy revision, control test denominators, certification renewal independent of substantive assurance |
| Enterprise setup | `ConfigureWorkspace`, organization scope, role-aware onboarding, imports | Real group hierarchy and permissions, source import lineage, setup readiness separate from compliance posture |
| Integrations & sources | `DataIntegrationsSection`, `DocumentImportWorkspace`, existing source/connection/binding contracts | Source health vs mapping review, credential/activation gate and sync-history provenance |
| Notifications & activity | `NotificationCenter`, `NotificationPreferences`, `SystemActivityPanel` | Bounded actor inbox, delivery/receipt states, exact event source and authorization-safe filters |
| Search & command center | Existing search fields, `App` route targets and `Explore` capabilities | Verify one authorized indexed search contract and common shortcut before any UI exposure; no broad client-side aggregation |
| People & access | `PeopleAccessSection`, `EscalationRoutesWorkspace`, existing identity/routing | Distinguish role membership from approval authority; invitation/delegation receipts and fail-closed routing |
| Audit & assurance | Program/issue/evidence/review/report foundations | Decide exact audit-plan source and reviewed evidence denominators before presenting an audit overview; don't synthesize an audit record container |
| Automation & governance | `AutomationSection`, `AutomationPolicies`, AIGovernance controls | Maker/checker activation, simulation, kill switch, notification-vs-outcome evidence and revision/source history |
| Contextual Copilot | Existing governed AI gateway and Program/Forms/Evidence sources | At most 3 authorized suggestions, one visible, manual selection; citation snapshot and allowed proposal-preview route; functional without AI |

## Release and review sequence

1. **Current integration first:** accept #502 only on exact-head CI and UI/UX. Reconcile independent #470 (adopted standard fact) with the latest Program composition; leave #469 edition preflight in #437 ownership.
2. **Shared visual gate:** record actual runtime snapshots at 1440 / 1280 / 1024 / 768 / 390px, light and dark, reduced motion, keyboard. Generate source-reference renders from the committed HTML and measure differences; the existing UI/UX review screenshot artifact is a runtime baseline, **not** automatically an approved-design comparison.
3. **Framework/Program/assessment flow:** close the unimplemented approved steps using existing Program Requirements, Form templates, mapping and adoption commands. Where #437 lacks server-side direct edition lookup/search, surface the limitation instead of extending clients with unbounded reads.
4. **Operating surfaces:** vendor, risk/loss, privacy, work and reports. Preserve exact scope, source population, as-of/period distinction and actor action across the routed handoffs.
5. **Administration/context:** hierarchy, people/access, audit, controls/policies, automation, integrations, notifications, global search and Copilot only where authoritative contract exists. Keep a single shell and user journey.
6. **Release acceptance:** execute the merged [four-actor/report plan](../../../acceptance/issue-462-actor-report.md) against a real isolated authenticated deployment: administrator, contributor, reviewer, authorizer, negative tenant/entity/group authorization, immutable source-to-report parity, download digest. Test only approved nonproduction data.

### Definition of completed

Each screen needs: exact source route/design version; component owner; authorized source API and snapshot; meaningful zero/unknown/stale/partial/denied states; keyboard/mobile/dark/light screenshots compared with approved HTML; full action round-trip (or explicit unavailable reason); targeted CI; documented reviewer acceptance. Do not close #462, deploy as a full cutover, or report visual parity until these receipts are linked.
