# Approved UI source inspection and implementation mapping

Date: 2026-10-17. Source: [complete designs.zip](https://drive.google.com/file/d/1y1Ij8zyfZKUKfcxp68T9FmJHabyWREZ_/view?usp=drivesdk). Related: #462, #437, #433.

## Inspected files

- `clearsight-framework-designs.zip/framework-library.html` (31,857 bytes): edition catalog, scoped filter buttons, adoption-status tabs, edition updates, card actions and adoption review. One embedded script and stylesheet; static prototype.
- `clearsight-framework-designs.zip/framework-adoption.html` (51,429 bytes): Program & Scope, Requirement Links, Questionnaire, Adoption Review, linked/to-review tabs, source viewer and confirm action. Two embedded styles, one script; no forms. Static prototype.
- `clearsight-program-designs.zip/program-workspace-preview.html` (71,240 bytes): Program overview, Requirements, Assessments, Evidence, Issues, Activity, action panels and responsive visual treatment.
- `clearsight-unified-design-system-v1.zip/clearsight-design-system/clearsight-design-system-spec.md` (12,643 bytes), `system.css`, `system.js`, HTML preview and component gallery.
- Headings, action labels, markup and embedded scripts of the 18 extracted design folders inspected. The designs include overlapping framework/program files and inconsistent historical sidebar drafts. Unified spec §1 explicitly identifies sidebar divergence and keeps existing routes authoritative.
- Current `web/src/components/ProgramsWorkspace.tsx`, `ProgramSetupWorkspace.tsx`, `ProgramRequirementsPanel.tsx`, `web/src/programFrameworkCandidatesApi.ts`, `DESIGN.md` inspected on current main.
- **21 October Notifications:** `source/clearsight-notifications-activity-designs/notifications-activity-preview.html` (79,068 bytes; SHA-256 `18a5fa9e4ef0b306da152444e3c2223fc6fe8ffe359c00e0002c5098609a03bc`) and companion HTML/CSS/JS inspected for full Inbox, Activity and Preferences composition. The [decision record](../../decisions/2026-10-21-notifications-workspace-v2.md) maps that structure to the existing scoped notification, preference and system-activity APIs. Status: **locally implemented and reviewed; publication pending**. The [evidence receipt](../../decisions/2026-10-21-notifications-workspace-evidence.json) records the scoped local review; remote CI and authorized-result acceptance remain open.

## Grounded conclusions

1. **Preserve system ownership.** UI files are static examples, not state or API contracts. Never inject prototype's standalone CSS, demo figures, local-only JavaScript state or sidebar as production code.
2. **Framework catalog**: govern edition identity/source/version/rights via #437. Render in existing shell and shared `SearchField`, `FilterBar`, `DataTable`, `FocusedSheet`, `StatusBadge`. Show unknown/unavailable provenance explicitly. No unverified “7 examples” as deployed coverage.
3. **Adoption**: derive draft/review/active state, application scope and version from existing Program framework adoption contract; build clauses as bounded paginated mapping records. Reuse `loadFrameworkClauseCandidates`, `loadProgramFrameworkMappings`, `proposeProgramFrameworkMapping`, `decideProgramFrameworkMapping` and existing `ProgramRequirementsPanel` actions. Inspect pending/approved/rejected status separately. Do not auto-map suggestions or count catalog clauses twice.
4. **Program**: adopt prototype's visual hierarchy without another Program state store, calculate only from authorized live Program projection; preserve status, stale, unknown, partial, source revision and exact drill.
5. **Forms**: “Change form” must use existing governed Form picker and authorization, never synthesize a separate questionnaire database.
6. **Interaction**: prototype actions “Manage regulators”, “Edition updates”, “Send for review”, “Confirm”, “Browse all” are not production-ready simply because they look clickable; route to verified actions or disable with concrete explanation.
7. **Navigation — amended 21 October**: the user's instruction requires menu conformity alongside page layout. Follow the [shared navigation decision](../../decisions/2026-10-21-v2-navigation-adaptation.md): Home, Frameworks, Programs, Forms, Vendors, Risk & Losses, Privacy, Work & Approvals, Reports, with authorized Settings in the footer. This supersedes the prior Home/Programs/Work/Insights/Reports/Configure presentation while preserving existing authenticated routes, identity, scope and permission ownership. Adapt the source menu with shared controls; keep original prototype files immutable and do not introduce unsupported destinations.
8. **Design-system**: reconcile action blue and other visual differences with existing semantic tokens globally, not a new page-local palette. Use the existing shared component library and accessibility mechanics.
9. **Data**: respect #433 scored Program Requirements and #437 edition ownership. Demo sample values (e.g. 52 requirements, 8 assessments, reference editions) must never enter live summaries.
10. **Acceptance**: desktop 1440+, 1280, 1024, 768 and 390px; light and dark, keyboard, reduced motion, zero/partial/stale/denied, source-version conflict, tenant/organization scope and role-authorized review/approval/report flow.

## Recommended small tranches

1. **Source & baselines**: original HTML/CSS/JS/spec and design notes (96 files) are committed under `source/`, with exact SHA-256 manifest and verified original archive provenance. Running-app screenshots and prototype-to-runtime parity review remain pending under PR #497.
2. **Shared shell & tokens**: single token and layout adjustment, responsive/scope/date consistency; tests.
3. **Framework catalog**: edition list/search/filter/source detail from backend, page-level state matrix; tests.
4. **Framework adoption**: guided scope → mapping → existing form selection → review; source/version checks and two-actor gates; tests.
5. **Program & assessment surfaces**: recompose real data, forms and evidence; no second engines; tests.
6. **Other operating surfaces**: Vendor, Risk/Loss, Privacy, Reporting, Work, Controls, Setup, Integrations, People, Audit, Automation, Search, Notifications and contextual Copilot, prioritized by exact user journey.
7. **End-to-end proof**: administrator, contributor, reviewer, authorizer, report-to-source parity, access isolation, performance and render-baseline gates.

## Decision on shadcn/ui

Do **not** wholesale migrate to a second component library. Current `react-aria-components` accessibility, shared token, and atomic component contracts should remain primary. Adapt shared components and, if necessary, narrowly introduce a missing primitive only after repository-wide reuse analysis.
