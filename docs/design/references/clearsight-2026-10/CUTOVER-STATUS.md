# Approved UI cutover — runtime reconciliation

Updated: 2026-10-17. Tracking: [#462](https://github.com/CloudSpaceLab/clearsight-grc/issues/462).
Authority: the immutable [approved source archive](./source/README.md), its [manifest](./source/MANIFEST.json), the [implementation mapping](./IMPLEMENTATION-MAPPING.md) and the unified design specification. Source HTML is a **visual/interaction benchmark, not a data or permission contract**. No source-example totals, statuses, suggestions or approvals may ship as runtime facts.

## What is integrated

- **#498:** approved design source, hashes and mapping in `main` (reference-only).
- **#497:** bounded Framework Library, Program adoption browser, authorized catalog cursor/source inspection, route handoff, tests. This is not the full four-stage adoption experience.
- **#499:** Program Overview composition, lifecycle label and review/current-position separation.
- **#500:** four Program record-backed overview measures; stale/unknown/partial state and exact-section drill, with readable metric headings.
- **#501:** Program record title is the single detail-page heading; register and Framework Library headings remain.
- **#495:** four-actor/report acceptance **plan**; not hosted execution.
- **#502:** active-standard requirement/form follow-through proposed; evaluate its exact-head checks before integration.

These merges establish a tested partial runtime migration, **not complete approved-HTML visual parity or production release acceptance**.

## Screen-by-screen reconciliation

“Foundation” means existing domain/UI owners were identified; it does **not** mean approved visual parity or the journey has passed. Prioritize existing shared components and backed routes; do not add a second work queue, audit engine, scoring engine, form library, approval service, or global navigation destination.

| Approved source area | Current runtime owner or contract | Remaining specific cutover gate |
| --- | --- | --- |
| Unified visual shell | Existing `App.tsx`, `AppViews.tsx`, enterprise shell and shared design-system tokens | Compare navigation, blue wave/header, scope/period controls, density, both themes and all approved widths before shared-token release |
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
