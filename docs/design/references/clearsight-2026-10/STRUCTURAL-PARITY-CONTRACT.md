# Approved HTML — structural parity contract

Tracking: #462. Source authority: `source/MANIFEST.json` (SHA-256-pinned original HTML and design notes). Updated: 2026-10-19.

**A screen is not migrated merely because its colors, cards, or typography resemble the source.** A release reviewer must verify the unified **global shell → approved page heading/actions → local tabs → filters → content sections → selected-detail/review panel** with the correct ordering, desktop/mobile layout, authorization and real operations. Do not count individual component merges as an accepted screen.

## Global information architecture

The [unified interface specification](source/clearsight-unified-design-system-v1/clearsight-design-system/clearsight-design-system-spec.md) §1 resolves the archived HTML packages' conflicting global sidebars. **Keep one authorized global navigation** (Home/Oversight, Programs/Portfolio, Work, Insights, Reports, Configure) and expose specialist views contextually; do not promote every historic prototype sidebar entry to a second global destination. Preserve existing route aliases and role/scope checks. **Inside each specialist page**, its approved menu, tabs, headings, section order and selected-record composition remain the target. The generic Portfolio lens must not appear above a page that already has its own approved local navigation, as in Framework Library.

At 1440/1280 desktop, compare logo/rail and top context position, page title/actions, local tabs and first visible content, not just individual elements. At 1024/768/390, verify the original source's responsive menu, title/CTA position, tab reflow and first substantive data panel without horizontal overflow or disappearing actions. Test both themes where supported; do not copy illustrative values or fabricate missing filters/indicators.

## Page structure inventory

All tab names below refer to **original HTML menu/section order**, not a claim that each currently has backing services. Counts in archived designs are examples and must not be copied.

| Screen / source folder | Approved top-level page structure | Current integration / structural gate |
| --- | --- | --- |
| Control Tower | Source absent: cannot assert page hierarchy without standalone approved HTML | Recover source first; keep blocked |
| Framework Library / `clearsight-framework-designs` | Heading/actions → **Explore · In my Programs · Edition updates** → search, jurisdiction/industry → edition rows ↔ detail inspector | #548 removes unrelated Portfolio lens and restores genuine Explore-first menu and register ordering. Industry and edition-update feeds remain unsupported; no fake counters |
| Framework Adoption / `clearsight-framework-designs` | **Program & Scope → Requirement Links → Questionnaire → Adoption Review** | Existing Program contract owns edition scope, clause links, Form and separate maker/checker |
| Program / `clearsight-program-designs` | Record heading/actions → **Overview · Requirements · Assessments · Evidence · Issues · Activity** → four measures → coverage / by-control-area / attention | Tab contract exists, but first-viewport geometry, coverage and attention composition remain unaccepted |
| Assessment / `clearsight-designs-01-05` | Assessment heading/actions → **Questions · Excel import · Review** → completion strip → sections / questionnaire / evidence/readiness | #541 replaces modal but remains structurally short of source; Excel mapping must be authorized and reviewed |
| Vendors / `clearsight-vendor-management-designs` | **Overview · Register · Onboarding · Assessments · Documents**; detail: Overview / Due diligence / Evidence / Issues & actions / Activity | Domain workflows exist; card repair #544 does not reconstruct full overview or tab navigation |
| Operational Risk / `clearsight-operational-risk-designs` | **Overview · KRIs · RCSA · Losses · Actions** | Preserve separate flow vs as-of semantics and actual record routes |
| Privacy / `clearsight-privacy-data-designs` | **Overview · ROPA · DPIAs · Breaches · Vendor reviews · Requests** | Existing ROPA detail alone is not equivalent to Privacy overview |
| Reports / `clearsight-reports-insights-designs` | **Overview · Explore · Reports · Schedules · Build report** | Reports library and saved run detail do not substitute for source overview/explore compositions |
| Work / `clearsight-work-approvals-designs` | **My work · Approvals · Team queue · Completed** → selected work and scoped action | Existing actor queue is genuine but Team/Completed cannot display unbacked counts |
| Controls / `clearsight-controls-policies-designs` | **Overview · Control library · Policies · Reviews · Certifications** | Need policy, review and certificate contracts |
| Enterprise Setup / `clearsight-enterprise-setup-designs` | **Organization · Groups · People & access · Jurisdictions · Frameworks · Data imports** | Existing Configuration sections need matched hierarchy and access decisions |
| Integrations / `clearsight-integrations-designs` | **Sources · Field mapping · Sync history** | Need source-backed mapping and health state |
| Search / `clearsight-search-command-center-designs` | Query → filters → scoped results → inspector | No global search until authorized backend query exists |
| Notifications / `clearsight-notifications-activity-designs` | **Inbox · Activity · Preferences** | Actor inbox and delivery receipts must be integrated |
| Audit / `clearsight-audit-assurance-designs` | **Overview · Audit plan · Evidence requests · Findings** | Must distinguish audit work from generic Matters |
| People / `clearsight-people-access-designs` | **People · Roles · Approval routes · Delegations** | Separate role membership, delegated authority and review status |
| Automation / `clearsight-automation-governance-designs` | **Automations · Schedules · Escalations · Activity** | Actual simulation, approval, activation and rollback required |
| Copilot / `clearsight-contextual-copilot-designs` | Contextual suggestions and evidence/mapping inspector on the originating record | Maximum three authorized source-grounded suggestions; never invent new judgment |

## Acceptance gate per page

- Record an exact source file/hash and the owning React route/component.
- Compare **menu destinations**, active navigation, view tabs, heading/actions, form/toolbar controls, content panel order, selected-record inspector and secondary states.
- Use **same-page, same-state** paired captures at 1440, 1280, 1024, 768 and 390px. A different domain-detail screenshot is **unpaired**, not accepted.
- Verify each visible action is routed to the existing authorized operation; unsupported controls may be labeled unavailable but never claim successful functionality.
- Verify scope/period/denominators and actor, tenant, entity/group denials independently of visual similarity.
- Only mark **ACCEPTED** when the entire page's source-backed structure, interaction, responsiveness and rendering pass. The current baseline remains **0/19**.
