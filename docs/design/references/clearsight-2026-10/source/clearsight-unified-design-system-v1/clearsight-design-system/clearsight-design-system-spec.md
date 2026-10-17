# ClearSight — Unified Interface Specification

Version 1.0 · 15 October 2026 · Design consolidation for [#462](https://github.com/CloudSpaceLab/clearsight-grc/issues/462)

**Status:** Design-approved direction, not an implementation claim. Gallery records and comparisons are illustrative. Implementation must use the existing ClearSight runtime, tokens, authority, and read models.

## 1. Interaction architecture

An operator should know **where they are, which records are included, whether the data can be trusted, and what happens next** without opening an explanation panel.

| Surface | Purpose | Stable presentation |
| --- | --- | --- |
| Shell | Orientation and authorized scope | One navigation hierarchy; identity, global search, group switcher, notifications and authorized organization selector |
| Home / Control Tower | CRO/CCO/CISO oversight | Up to four real posture metrics; scoped concentration and movement; separate Oversight / Attention / My work |
| Portfolio | Find an operating record | Searchable bounded lists with category-specific columns; persistent filters |
| Program | Own an enduring obligation | Posture, assessments, Requirements/Controls, source evidence, issues, history |
| Review | Make one governed decision | Single selected record, evidence, reviewer, authority state, one action; no parallel review dashboard |
| Configure | Change governance policy | Draft, simulate, impact preview, review, approval, activate, audit / rollback |
| Copilot | Optional source-grounded assistance | Contextual sheet, at most three suggestions; no automatic approval or compliance conclusion |

**Navigation decision:** do not promote every specialist page to the global sidebar. Keep the existing product routes (`Oversight`, `Programs`, `Work`, `Insights`, `Reports`, `Configure`) and expose ROPA/RCSA/KRI/loss/vendor-specific subviews contextually where appropriate. The historic designs used different sidebar lists; their *content* is authoritative, not those inconsistent navigation drafts.

## 2. Exact design system ownership

The codebase already uses an atomic design system; do not recreate these families in feature workspaces.

| Visual pattern | Existing source of truth | Required behavior |
| --- | --- | --- |
| Page / shell | `App.tsx`, `enterprise-shell.css`, `appRouting.ts`, `WorkspaceSwitcher` | Keep authenticated role and scope, no duplicate Home or hidden disconnected route |
| Color / spacing / typography / elevation | `web/src/design-system/tokens/{primitives,semantic,components}.css`, `DESIGN.md` | Use tokens; do not commit independent hex palettes from the prototype |
| Primary/secondary actions | `Button`, `ActionLink`, `IconButton` | One dominant consequential action, visible disabled reason |
| Search, selects, scope | `SearchField`, `SelectField`, `FilterBar`, `FilterChip`, `ScopeBar` | Query and route reflect authorized scope; clear selections |
| Review record | `DataTable`, `SelectableRecord`, `FocusedSheet`, `FocusedDialog` | Exact record, source, owner, decision and focus return |
| Summary cards | `MetricCard` | Stable measure, quality, comparable basis and drill; never fabricated global percentage |
| Trends / distributions | `MetricTrend`, `RankedBarList`, `StackedDistribution` | Proper temporal basis; loss flow != posture stock; chart and text parity |
| Tabs and status | `Tabs`, `StatusBadge`, `Notice`, `EmptyState` | Keyboard semantics; status distinct from quality and authority |
| Forms / onboarding | Existing form authoring, starter catalog, `RoleAwareOnboarding` | Answer vs assessment vs approval distinct; guide skippable and resumable |
| Framework reference | `internal/frameworkcatalog`, Program Requirements, #433/#437 | Edition/source/license/applicability before showing attributed scores |
| AI suggestions | Existing AI governance/gateway, source/evidence context | Actor-authorized evidence, source revision, at most three selectable suggestions |

### Visual mapping to current tokens

The reference image’s strong blue action color is **a visual target**, not a new independent production token. Existing primitive cyan role and light/dark semantic mappings are the official implementation owners; reconcile the final action hue in `DESIGN.md` and tokens through one design-review change. Never add local `#1268e6` declarations to feature CSS. Use the current 4px spacing scale; 6/10/14px radii; 44px comfortable / 40px compact pointer controls; 12/14/18/28/36px typography bands; 100/150/200ms motion tokens. Use `Inter, Segoe UI Variable, Segoe UI, system-ui`.

## 3. Copy contract

- The header names a function: **Enterprise Control Tower**, **Vendor relationships**, **Control library**, **Review evidence**, **Framework adoption**.
- The supporting sentence adds business scope or period only: **Risk, compliance and action in the last 90 days** (derived from confirmed period); **6 of 52 requirements unassessed**; **3 vendor documents expire within 30 days**.
- Labels say **All groups** and **Last 90 days**. Do not say “All custom groups”. Avoid “your intelligent partner”, “turn data into progress”, “stronger decisions”, ornamental availability badges, and repeated self-descriptions.
- Controls describe a direct outcome: **Import data**, **Review evidence**, **Send for approval**, **Add to Program**, **Show source**, **Export report**.
- If a process is blocked, name the blocking condition and recovery action. Data freshness and authority cannot be hidden in a tooltip.
- Maintain `web/src/copyQuality.test.ts` patterns for reliable failure modes. A phrase-scan passing does not replace full workflow copy review.

## 4. State matrix (mandatory)

| State | Display | May execute material command? |
| --- | --- | --- |
| Loading | Stable content skeleton; no artificial metrics | No |
| Current/complete | Explicit source/as-of, valid denominator and scope | Only with verified authority |
| Partial | `Coverage incomplete` plus excluded/unknown count | Only if policy allows; never infer compliant |
| Stale | `Needs refresh` and observed timestamp | Fail closed for freshness-dependent commands |
| Unknown | `Unknown`, no zero fallback | No implied pass or favorable status |
| Denied | Authorized scope unavailable; next valid escalation/recovery | No |
| Source unavailable | Previously known fact may be shown with `stale`, never `current` | No unsupported factual conclusion |
| Draft | Distinguish proposed from effective policy/assessment | Cannot activate without versioned review |
| Submitted | Receipt / accepted submission only | Not the same as verified evidence |
| Verified | Exact reviewed evidence and decision record | Authorized outcome checks still required |

Historical period controls must operate on **server-backed period-flow measures**. Current posture metrics are as-of measures even if the shell date changes. `OversightPeriodPicker` presently exposes 7/30/90 days plus a custom **start** date; a custom historical end-date must not be shown unless supported by the backend.

## 5. Layout system

- Desktop >= 1440 CSS px: 220px navigation rail, 32px content inset, 12–16px panel gaps, fixed top context, 4 headline cards maximum. One dominant chart per decision area. Detail sheet usually 425–560px where content fits.
- Laptop 1024–1439: 2-column analytical section, 2×2 metrics, collapsible contextual assistant. Never shrink a 3-column review so far that evidence text is unreadable.
- Tablet 768–1023: adaptive navigation, detail becomes sheet, charts stack, filters move into a single disclosure with applied chips.
- Mobile <= 767: tasks and titles before charts; selection details become full-height sheet; register rows become key/value cards; no horizontal page overflow. 44px touch hit areas. Date and search fields should not trigger unwanted viewport zoom.
- `prefers-reduced-motion: reduce`: eliminate nonessential animations. Do not animate counts, gauges, or an idle Copilot orb. Changes of scope or period reset old data only once the new authorized result is confirmed or visibly enter a loading/error state.

## 6. Screen-to-component inventory

| Design | Primary object | Recommended presentation |
| --- | --- | --- |
| 01 Control Tower | Organization posture | Stable measures, concentration, small trends and Attention link |
| 02 Framework Library | Verified edition metadata | Searchable edition listing + inspect/review sheet |
| 03 Framework Adoption | Edition-to-Program relationship | Clause reuse, version pin, source rights, reviewed adoption |
| 04 Program | Governed Program | Summary, requirement coverage, evidence, work and history |
| 05 Assessment | Form response | Question navigator, source suggestions, reviewable XLSX mapping |
| 06 Vendor | Legal identity + service relationship | Portfolio and relationship-specific onboarding + due diligence |
| 07 Operational Risk | RCSA, indicator, loss | Separate native visual semantics, same organization scope |
| 08 Privacy | ROPA activity, DPIA, incidents | Specialized registers and purpose-bound approvals |
| 09 Reports & Insights | Report definition/run | Bounded analytics, saved view, preview and authorized export |
| 10 Work & Approvals | Matter/task/decision | Queue + one focused review, exact authority |
| 11 Controls & Policies | Control implementation / published policy | Control library and governed policy revisions |
| 12 Enterprise Setup | Organization hierarchy and configuration | Tree + detail, context-specific setup prompts |
| 13 Integrations | Source / connection / view / binding | Health list, mapping editor, run audit |
| 14 Search | Authorized multi-object result | Query, filters, list + detail; ⌘K quick search |
| 15 Notifications | Delivery and activity receipts | Action / unread / history, not an approval substitute |
| 16 Audit & Assurance | Audit plan, request, finding | Planning and fieldwork review, audit trace |
| 17 People & Access | Identity + effective authority | Person, position, roles, routes and delegations separately |
| 18 Automation | Versioned automation policy | Draft → simulate → approval → activation with rollback |
| 19 Copilot | Active Program or assessment | Source-grounded contextual proposal in dismissible sheet |

## 7. Delivery sequence

1. **Gate A — governance:** Add this design spec to the existing repository `DESIGN.md` links and #462; preserve screenshot baselines and component gallery state fixtures. Do not commit a second CSS foundation.
2. **Gate B — shell/copy:** Align nav and role defaults, group naming, true date control, first-run guide. Verify desktop/mobile, light/dark, keyboard access.
3. **Gate C — Home/portfolios:** Recompose existing bounded metrics and domain-specific registers. Complete current/partial/stale and exact-drill parity.
4. **Gate D — working records:** Program, Forms, Vendors, Privacy, RCSA/KRI/Loss, Work and audit review; existing authority and persistence only.
5. **Gate E — catalog and seeds:** #437 source-reviewed framework releases, #433 Program calculation proofs; separate neutral reference content, private customer imports, non-production demonstration records and test fixtures. No “500+ supported frameworks” claim from an unpublished list.
6. **Gate F — final UX:** Copilot contextual availability, motion/reduced-motion, source-to-screen and report parity; cross-browser and responsive screenshots before merge.

### Final acceptance checklist

- [ ] Every enabled action has a real route/operation and appropriate authority behavior.
- [ ] Organization and period filters preserve exact scope and drill identity; no cross-tenant data leakage.
- [ ] Percentages reconcile to a denominator, current revision, coverage and authorized calculation.
- [ ] Unknown, stale, partial, denied, zero and unavailable states have test fixtures.
- [ ] RCSA/KRI/Loss, ROPA and vendor summaries resolve to native records and authorized results.
- [ ] One primary action per work state; no duplicate permanent quick-start or AI panel.
- [ ] 390/768/1024/1280/1440+ viewport proofs in light/dark, keyboard, screen reader and reduced-motion.
- [ ] Sources/rights/editions and private imports are segregated from demo data, verified by idempotency tests.
- [ ] Compare initial and final screenshots; remove redundant prose, charts and controls before release.

**Reference assets:** `clearsight-design-system.html`, `system.css`, `system.js`, and its `assets/` gallery. The prototype implements local behavior only; it is not production code and must not be transplanted into the application in place of its shared UI contracts.
