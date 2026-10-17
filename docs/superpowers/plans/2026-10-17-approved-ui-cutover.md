# ClearSight approved 2026 UI — implementation and acceptance plan

Date: 2026-10-17 · Source archive: [complete designs.zip](https://drive.google.com/file/d/1y1Ij8zyfZKUKfcxp68T9FmJHabyWREZ_/view?usp=drivesdk) · Tracking: #462 · Domain ownership: #437/#433.

## 1. Verified design inputs and precedence

The source archive contains 18 nested ZIPs. The canonical unified design-system document is `clearsight-unified-design-system-v1.zip/clearsight-design-system/clearsight-design-system-spec.md` (12,643 bytes). Other source packages include working HTML prototypes, associated CSS/JavaScript, mock data and design notes. Treat their content/layout as visual and interaction references, **not runtime data models**. Some packages share older/different global sidebars: the unified design spec §1 takes precedence.

Detailed reference inventory (screens are numbered according to the unified spec):

| # | Source package | Existing owner | Main nonnegotiable |
| --- | --- | --- | --- |
| 01 | `clearsight-designs-01-05.zip` / Control Tower | Home, Oversight, Scope | Four grounded as-of posture measures and bounded changes |
| 02 | `clearsight-framework-designs.zip` / Framework Library | #437 catalog, Program surfaces | Version, source, rights, applicability before actionable adoption |
| 03 | same / Framework Adoption | Program standard adoption, Requirement mappings | Reuse matching requirements, reviewed links, do not inflate denominator |
| 04 | `clearsight-program-designs.zip` / Program Workspace | ProgramsWorkspace, ProgramRecordWorkspace | Separate requirements, assessments, evidence, issues and historical activity |
| 05 | `clearsight-designs-01-05.zip` / Assessment | Forms/Capture and source import | Answers != assessment; blanks != zero/no; source suggestions need confirmation |
| 06 | `clearsight-vendor-management-designs.zip` | Vendor identity, service relationships, forms | Approval, renewal and expiry independently verified |
| 07 | `clearsight-operational-risk-designs.zip` | RCSA, KRI/KCI, Loss | Risk stock, KRI breaches and loss flows cannot share an arbitrary score |
| 08 | `clearsight-privacy-data-designs.zip` | ROPA, DPIA/consent | Distinct processing and review authority |
| 09 | `clearsight-reports-insights-designs.zip` | ReportingPage / Insights | Export uses exact authorized persisted report run |
| 10 | `clearsight-work-approvals-designs.zip` | Matters, Decisions, assigned Work | One governed decision in focused sheet |
| 11 | `clearsight-controls-policies-designs.zip` | Control catalog and policies | Reuse control identities; versioned policy reviews |
| 12 | `clearsight-enterprise-setup-designs.zip` | Organization scope/configure | Verified entity hierarchy and actor-permitted mutation |
| 13 | `clearsight-integrations-designs.zip` | Source configuration/integration | Provider run health, bindings and audit truth |
| 14 | `clearsight-search-command-center-designs.zip` | Existing search and scoped navigation | Authorized bounded results, keyboard access |
| 15 | `clearsight-notifications-activity-designs.zip` | Notification Center | Delivery receipt != approval |
| 16 | `clearsight-audit-assurance-designs.zip` | Governance, Matters, Evidence | Traceable plan/fieldwork/review states |
| 17 | `clearsight-people-access-designs.zip` | Identity and authorization | Person, position, roles, delegation distinguished |
| 18 | `clearsight-automation-governance-designs.zip` | Automation policies | Draft/simulate/approve/activate/rollback |
| 19 | `clearsight-contextual-copilot-designs.zip` | Existing AI gateway | <=3 evidence-backed contextual actions; unavailable fallback |

Use original `*-design-notes.md` as semantic authority over illustrative mock labels. Example: Program design notes explicitly state “52 applicable” and sample assessment counts are illustrative and that the 7/30/90 selector filters *activity*, not current requirement posture. Assessment notes explicitly state the file selector does not parse/upload a spreadsheet and that the preview “submit” does not persist. Production must provide real persisted operations before enabling analogous actions.

## 2. Architecture decision records

**ADR-01 No second design system.** Keep `web/src/design-system/tokens`, existing `./components/ui` contracts, `react-aria-components`, theme preferences and `DESIGN.md`. Reconcile blue primary actions through *semantic* tokens, never page-local pasted hex or copied global prototype CSS. Shadcn is not a wholesale prerequisite; only evaluate a missing primitive against existing contracts.

**ADR-02 Runtime data and scopes.** Present only authorized tenant/legal-entity/group-scoped API records and their server cursor/denominator. Chart selection must change real filter/query/route; never filter a hardcoded mock list. Unknown, stale, partial and denied remain distinct from zero.

**ADR-03 Temporal semantics.** Scope selection is server-authoritative. Historical period controls only change period-backed activity/flow. As-of posture is independent of historical event filter unless the backend supports snapshots. Do not expose unsupported custom end-date.

**ADR-04 Shared domain truth.** #437 owns edition publication and adoption; #433 owns scoring. Catalog clause IDs are *attribution*, not scored obligations. A single Program Requirement mapped to two standards remains a single scoring unit. No parallel form, task, report, approval, evidence, score or AI stores.

**ADR-05 Prototype isolation.** Preserve original HTML/CSS/JS in a reference-only documentation folder. Do not import them into `web/src` or deploy standalone routes; do not introduce illustrative names, figures, or statuses as defaults.

**ADR-06 Operational copy.** Functional headings, objects, states, exact dates, next actions. Remove decorative “AI” narratives and reassurance phrases. Update `copyQuality.test.ts` for known regressions.

## 3. Dependency-ordered delivery

Each tranche is independently reviewable, requires targeted tests plus rendered proof and must not duplicate work already owned by open PRs. Review current head/overlap immediately before development.

### T0 — Source governance and baseline (first tranche)

- Inventory all nested packages with exact SHA-256 and per-file sizes; annotate source precedence and relevant owner.
- Commit original text source files: canonical spec, reference HTML, external CSS and JS, plus design notes, segregated under `docs/design/references/clearsight-2026-10/source/`. Keep heavy PNG screenshots as review artifacts or LFS; never in shipped web asset graph.
- Add canonical design authority references to `DESIGN.md` and #462; snapshot existing shell and four key workflows in both themes.
- Freeze matrix of real routes/components/API reads/mutations, mark prototype-only behavior.
- Gate: reviewable source bytes and hashes, no runtime import, no build/test delta, no fake figures.
- **Current status:** source indexed and mapping committed in PR #497; original source-byte import and rendered baseline still pending. No T0 completion claim until those gates pass.

### T1 — Shared shell, tokens and interaction primitives

- Compare prototype anatomy against existing Button/SearchField/FilterBar/Tabs/DataTable/FocusedSheet/MetricCard/StatusBadge.
- Change shared tokens/layout rather than overwriting each page's component states.
- Real global search, scope hierarchy, notification state, date semantics and one navigation hierarchy; responsive 390/768/1024/1280/1440.
- Exit: zero duplicated navigation endpoints, light/dark, focus, keyboard and reduced-motion screenshots; no presentation regressions in existing routes.

### T2 — Framework Library (after #437 contract)

- Server-bounded edition catalog with search, filter, applicable-to context, publisher, source, rights, status, supersession and last verification.
- Detail sheet opens exact edition/source with allowed actions; empty/offline/partial/denied states.
- Catalog cannot advertise “installed” or “assessed” merely because a reference row exists.
- Exit: list/detail/route parity, cross-tenant isolation, source and release history, exact cursor behavior.

### T3 — Framework Adoption

- Existing Program setup entry points: Use a standard / Blank / Import, only if corresponding operation is real.
- Select Program entity scope and pinned edition, show candidate mappings and chosen/rejected links; preserve reviewed approval state.
- Reuse `loadFrameworkClauseCandidates`, `loadProgramFrameworkMappings`, `proposeProgramFrameworkMapping`, `decideProgramFrameworkMapping`, existing Program Requirement commands and version conflict handling.
- Existing governed Form selection; no prototype questionnaire store. Governed adopt/approve action and receipts required.
- Exit: two editions mapping same requirement scored once; stale version invalidation; segregated proposal/approval actors; source to mapped clause fidelity.

### T4 — Program / Assessment / Forms

- Recompose Program overview with four meaningful, *real* first-row values (no sample requirement counts), breakdown, assess/evidence/work and activity.
- Assessments use existing Capture/Forms and immutable response revisions; distinct answer completion, assessed evidence and approval.
- Spreadsheet mapping must persist original sheet/cell/raw values, blank protection and explicit acceptance.
- Exit: API snapshot to visible count parity, source tracing, permission and resume semantics.

### T5 — First operating wave

- Vendor, Work/Approvals, Reports/Insights, RCSA/KRI/Loss and Privacy interfaces using native domain panels.
- Ensure compact focused actions, scope propagation, data quality and report download source/role parity.
- Exit: administrator → owner → reviewer → authorizer → exported record journey in real backend.

### T6 — Governance wave

- Controls/Policies, Organization Setup, Integrations, Search, Notifications, People/Access, Audit/Assurance and Automation.
- No extra global route proliferation or duplicated capabilities.
- Exit: each enabled action mapped to permitted endpoint, loaded/empty/error/denied/stale cases and rollback path.

### T7 — Contextual Copilot, performance and final cutover

- Evidence-grounded <=3 suggestions, manual switch, no autonomous approval, honest unavailable state; reduced motion.
- Full visual regression and performance budget for every implemented surface.
- Cut over only after actual evidence and owner acceptance; remove redundant legacy CSS/components after usage inventory.

## 4. Mandatory test matrix

For each tranche test: (1) live success, loading, empty, partial, stale, denied, unknown, conflict and network recovery; (2) administrator, contributor, reviewer, authorizer; (3) tenant + entity + group authority isolation; (4) exact query/filter/route persistence; (5) keyboard/focus/aria/reduced-motion; (6) 390/768/1024/1280/1440+ responsive, light/dark; (7) screenshot diffs against approved original and pre-change current app; (8) targeted unit/integration, web build and CI on exact PR head.

Acceptance proof cannot consist of mocked UI screenshots. At least one real Program with governed framework source → scope → form response → evidence review → decision → authorized report must reconcile every displayed status, denominator and source revision. Existing #433/#437 contracts are dependencies, not replaced here.

## 5. Implementation discipline

One domain/component ownership per PR, small commits; do not merge draft or failing CI. Before each tranche: check main and active PRs for overlap; run component and CSS usage search; establish API shape and authority; create targeted tests before implementation; produce before/after screenshots and actual tests. Keep #462 tracker synchronized with independently proven work. Branch/PR #497 is a preparation branch, not complete cutover.

## 6. Vertical integration and release order

The horizontal T0–T7 headings above describe coverage, not permission to ship unconnected prototypes. Prioritize coherent, executable journeys. Each slice owns its route, authenticated API reads/mutations, decision and data-quality states, visual composition, evidence, tests and rollback.

| Slice | Existing runtime owner | Delivery gate |
| --- | --- | --- |
| V1: Mapping review | ProgramRequirementsPanel and framework mapping operations | Focus one pending link, clear per-link rationale on change/cancel, enforce role, refresh versioned Program; regression test |
| V2: Framework Library | #437 published edition catalog | Genuine edition search, source/rights/version, applicability and authorized bounded pagination; no synthetic scores |
| V3: Adoption | Program versioned framework adoption | Select Program/scope/edition; propose and independently approve; preserve history and version conflicts |
| V4: Requirement links | Existing Program Requirements and mapping APIs | Link existing obligations and review; multiple editions share one scored Requirement without duplication |
| V5: Assessments | Forms/Capture, evidence and assessment | Choose existing Form, submit, assess and decide; separate answer completion, evidence and compliance |
| V6: Reports | Persisted reporting runs and authorizations | Authorized export reconciles to exact source records, revision, snapshot, scope and denominator |
| V7: Other domains | Vendor, risk, privacy, work, controls, audit and admin surfaces | Execute an end-to-end domain journey before retiring each legacy presentation |

**Contract required for each PR:** current route, endpoint, authorized operation, scoped filters, input/output fields, loading/empty/denied/stale/conflict/failure states, desktop/mobile visual comparison, keyboard/focus/reduced-motion proof and test/CI result. An illustrative prototype control with no live endpoint is not an enabled production action.

**Dependencies:** V1 can ship independently of unfinished #437 catalog and #433 score work. V2 and V3 must wait for authoritative catalog/adoption contracts, never static arrays. A wholesale shell or shadcn rewrite is not a prerequisite. Scope and time semantics remain server-driven.

**Current proof:** V1 rationale isolation and regression coverage are committed to draft PR #497. CI and live actor-acceptance remain unverified, so V1 is not marked complete. T0 still requires source-byte import and running-app visual baselines.
