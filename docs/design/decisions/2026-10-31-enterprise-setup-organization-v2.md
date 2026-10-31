# Enterprise Setup — Organization V2

Date: 31 October 2026 · Tracking: #462 · Starting main: `c40f4edd7ad4107fc2b36ebfd65af87bd5d5502b`

Final truncation correction: published `74603165763e69d00b8dc2cb8e804225053f6a68` maps to local `613f07f705c383201f576504f7debf7b84592151`; both have tree `c9adf95492efe0942d3d5803d9184ddb19ae6183`. **Organization is implemented as partial V2 layout integration with a passed final scoped local browser gate.** Earlier full-suite and compatibility validation remains bound to its recorded revision below. Hosted review and release status are recorded in [PR #599](https://github.com/CloudSpaceLab/clearsight-grc/pull/599).

## Problem and source

At the starting main, Organization opened inside the People workspace, ahead of a data-boundary form, area table and position table. The approved Enterprise Setup page leads with a searchable hierarchy, compact structure/change context, selected-unit detail and contextual actions. At all five required widths, the retained before-state shows no area or position record in the initial viewport. The implemented Organization page replaces that anatomy while retaining the existing data and command owners.

The starting main includes merged People #589, Work deadline drills #597 and PostgreSQL population regression #598. Its CI, UI review and exact-SHA demo deployment passed. The [31 October release reconciliation](../references/clearsight-2026-10/CUTOVER-STATUS.md#31-october--released-baseline-and-enterprise-organization-integration) retains those receipts separately from this tranche's local review and separately recorded hosted release status.

Layout authority: `docs/design/references/clearsight-2026-10/source/clearsight-enterprise-setup-designs/enterprise-setup.html`, SHA-256 `5182fcd83331fafd42c7711fa6eafdbebb5d255dec51b6b73d42217e8ebdfd3a`. The self-contained `enterprise-setup-preview.html` has SHA-256 `cc7542e2a3047338b1d2fbbbe33756adbc9ee8c2084fb5920834da8aeedc04c7`. Original HTML, CSS, JavaScript, notes and manifest remain immutable.

## Composition and routing

- Keep `#configure/access/positions` as the Organization destination. Replace its People heading/contextual tab with one Enterprise setup heading, a compact verified-entity banner and source-order Organization / Groups / People & access / Jurisdictions / Frameworks / Data imports navigation.
- Use manual keyboard activation for this six-tab navigation: arrows move focus; Enter or Space activates the focused destination. Existing shared Tabs consumers retain automatic activation unless they opt in.
- Reuse `IdentityAccessPanel` for reads, permissions and all commands. Keep the Settings overview and existing specialist configuration owners. Manage organization in the shared scope selector opens this existing Organization destination.
- Place the hierarchy beside compact loaded-structure and pending-change information; retain the lower contextual cards and selected-unit rail. At 1300px the rail follows the primary region; at 780px the inner hierarchy/summary columns stack; at 560px lower cards and context become one column. Use shared controls and theme tokens, 44px targets and concise operational copy.
- Search only returned areas and keep ancestry visible. Collapse is separate from selection. Selected-area identity is a bounded, encoded `unit` URL value; missing selections remain explicit. Guard missing parents and cyclic relationships without inventing ancestry. Selection on narrow screens focuses the context; a return action restores the selected row.
- Move positions/roles and data-boundary editing into contextual focused sheets. Reuse their existing proposal, independent decision, version, simulation, effective-date, history and restore implementations. Area proposal/review remains in `OrganizationScopeManager`.
- Groups and Jurisdictions retain their source menu positions with truthful unavailable panels. Directory groups are not groups of organization units; data residency is not an operating-jurisdiction selection. People, Frameworks and Imports hand off to their existing owners. Cross-workspace navigation must notify App rather than only changing Configure's local hash.

## Data and authority boundary

The access overview returns up to 500 current-entity areas with a truncation flag, up to 50 current-entity positions, up to 50 tenant-wide profiles and bounded role/proposal/history collections. It has no complete population total, cursor, overview timestamp or atomic snapshot. Counts describe loaded records; local search cannot retrieve omitted areas. An assigned occupant is not necessarily an active person.

When the area response is truncated, the existing hierarchy heading states “{count} loaded areas · More areas not shown.” The detailed limitation remains below the list. This makes the qualification visible before a user would have to traverse 500 rows; it adds no extra heading row or unsupported way to retrieve the omitted records. Nontruncated responses retain “Search covers this list.”

Position filtering uses the selected area's exact ID and follows `parent_scope_id` through the loaded scope map. A matching display path is insufficient: duplicate or stale paths cannot admit an unrelated position, and missing parents or cycles stop traversal before a selected ancestor is reached. An empty area filter shows all loaded positions; a missing selected area shows no positions until the user changes or clears that filter. The hierarchy's display adaptation does not become a second organization store or an authorization source.

Prototype six-area readiness, ownership coverage, member totals, invitations and reusable organization groups have no matching contracts. The header/banner use the verified current entity; structure and change summaries use their actual loaded populations. No readiness percentage, compliance conclusion, source example or additional organization store is introduced.

All writes retain the existing conjunction of identity configuration and governance-write permissions, verified actor/entity scope, optimistic version and maker/checker checks. A restore is a new proposal. Stale or denied reads cannot enable commands. Selection changes close area editors and decisions; loss of configuration permission removes mutation controls and open editors. Changing tenant/entity/actor remounts data and editor state. A URL-selected area not returned for the new scope remains explicitly missing instead of silently selecting a different record.

Area review shows the proposal's operation, base version, code, current/proposed name and type, maker, time and available impact counts. A proposed parent is identified by its loaded full path and code, with an exact-ID fallback when it was not returned. Move cannot submit an unavailable current parent until the user explicitly chooses a loaded destination or the legal entity. Search exposes matching areas and their parents; disabled collapse controls explain that clearing search restores collapse.

An existing PostgreSQL constraint excludes imported `ORGANIZATION_UNIT` kinds from direct Move/Retire revisions. This frontend tranche requires an explicit managed-kind Update and independent approval before enabling those actions. An imported-area edit must not silently choose Department. The direct API inconsistency remains a backend limitation; no migration is included here. A separate narrow correction hides Restore when configuration permission is absent.

## Verification plan

Retain same-screen original, before and after renders at 1440/1280/1024/768/390 in light/dark. Assert heading/menu order, hierarchy/summary/context replacement, first useful record, selection/search/collapse, context focus/return, 44px controls and no page overflow. The original is light-only; dark is the runtime adaptation.

Exercise loaded/empty/truncated/read-only/denied/error/retry states, missing-parent/cycle cases, missing selections, browser history, real destination handoffs, independent area review, explicit imported-kind conversion and editor closure. Preserve existing position, role, boundary, scope-remount and provisioning-token tests. Bind fixture/capture/source/build/image identities; exclude test transport from the customer build. Named reviewers must inspect final renders and the hosted artifact before merge.

## Implementation and review checkpoints

The implementation now includes the source-order Enterprise shell, searchable ID-based hierarchy, loaded structure/change summaries, responsive selected-area context, contextual sheets and the existing owner handoffs. Focused regressions cover hierarchy cycles/missing parents, exact selection and versioned decisions, imported-type conversion, restore permission, explicit Move-parent selection, duplicate-name proposal destinations and position ancestry independent of display paths.

### Retained validation before the truncation correction

Published runtime `f2cc3e90dbdada92be03c13ef6ba8e4ccec60e9a` maps to local `2264dfd18bc070fe6d5260422272b1533966b8dc`, both with tree `a56a650c8e087d27c2f29ff9595a1936c6dd1361`. These checks remain bound to that earlier runtime:

| Check at local runtime `2264dfd` | Result |
| --- | --- |
| Full web suite | PASS — 277 files / 2,299 tests |
| TypeScript, customer build and evidence build | PASS |
| Customer bundle budgets | PASS — initial JavaScript 161,937 bytes gzip; initial CSS 32,126 bytes gzip; largest JavaScript chunk 573,982 bytes raw |
| Customer fixture boundary and three existing UI contracts | PASS |
| Existing Configuration and People compatibility | PASS — 10 Configuration cases, 92 People cases and 107 PNG byte/hash receipts; independent review by `/root/ci_review` |

The preceding browser checkpoint also passed 10 layouts / 11 behavior scenarios / 42 PNG receipts at local capture `ed46dd743417ace1cd4af588aae2121055544ba1`, published as `7da3ba6d998e65f034c31073737bd3f1b9410535`, tree `0c345728a5f523453eb5acddf094d46203cfc6c7`. That capture changed an imported-area dialog label in the harness while retaining the `2264dfd` application assets. Its build digest is `7d92d25660e43fdf132676b7f3099ef82430bbd763422c00a12045b0a5e5acc3`; receipt SHA-256 is `6a0045c61bee2f0f9bf2bc6b5072f6b575f36fe6f6e0c0a68dc7bb08b184fa12`. The baseline, validation and compatibility evidence remains an immutable historical checkpoint.

### Final visible-truncation correction

Local `613f07f705c383201f576504f7debf7b84592151` / published `74603165763e69d00b8dc2cb8e804225053f6a68` changes the truncated heading and strengthens the existing browser scenario to require that qualification in the viewport. It retains the detailed footer and all nontruncated wording. Final local checks are targeted; the full-suite and compatibility counts above are not attributed to this later revision.

| Check at local correction `613f07f` | Result |
| --- | --- |
| Scope Manager and copy-quality suites | PASS — 2 files / 15 tests |
| TypeScript, customer build and evidence build | PASS |
| Four Node verification gates | PASS |
| Customer bundle budgets | PASS — initial JavaScript 161,935 bytes gzip; initial CSS 32,126 bytes gzip; largest JavaScript chunk 573,982 bytes raw |
| Final Enterprise browser/evidence gate | PASS — 10 layouts, 11 behavior scenarios, 42 PNG receipts; `failure: null` |
| Final scoped axe checks | Zero reported violations; color-contrast checks remain incomplete |

The final receipt identifies local `613f07f705c383201f576504f7debf7b84592151`, equivalent to published `74603165763e69d00b8dc2cb8e804225053f6a68` at the tree above. Build digest before and after capture: `28e6d798572e8d37da39a977b639694fb68842515f73b4011517ff51cfe3d4b1`. Receipt SHA-256: `3ac8caa8a93ba6eb2e5bf8e6440e6a7c75f338835be5f468b39a82ba348fafd0`. The [Enterprise evidence record](../../evidence/2026-10-31-enterprise-setup/README.md) retains the final source/runtime comparison, named review and earlier checkpoints.

All ten final viewport captures begin at `scrollY = 0` and show the first organization area fully inside the viewport. At 390×844, that row spans y=684–733.89px in both themes; the before-state's first area began at y=1203px. The strengthened truncated scenario passes with 500 returned areas and the qualification visible in the initial viewport. `/root` directly inspected its final frame: “More areas not shown” spans y=544.11–562.11px, below the 64px shell. The detailed footer remains available after the list.

This is a scoped local result from an isolated synthetic transport, not proof of backend identity, authorization or material command execution. The original is light-only, remote font requests were blocked and environment fallback fonts were used. Axe's incomplete color-contrast results are disclosed separately; zero reported violations is not a complete contrast or accessibility certification. [PR #599](https://github.com/CloudSpaceLab/clearsight-grc/pull/599) records hosted CI, artifact inspection, merge and demo release status for this tranche.

## First-opening runtime follow-up

Hosted CI subsequently exposed a dropped first Positions & roles activation. A deterministic one-click regression reproduced the initial passive reset overriding the open state; changing only the two panel-reset hooks to layout effects made the same interaction pass. Runtime `3a32ab85e0cb0615903f76fcac1c08362db306c8` (local `aa666af542fdcba2f2883a58821434f3fbc18740`, shared tree `9e670cd7477ddd25b0e5c9e1d01961ae3ecfbacc`) preserves selection-change and permission-revocation closure, plus read-only viewing. The original progressive workflow assertions are unchanged.

The [first-opening evidence](../../evidence/2026-10-31-enterprise-setup/first-open-followup/README.md) retains the failed CI checkpoint, red/green proof, 19 focused tests, full 277-file/2,301-test suite and fresh 10-layout/11-behavior/42-image capture at build `40e12a493b9e595206607cba4729db53c7d685dc39cd5169c330fc268542649c`. Earlier captures remain historical, including the separate Select-focus synchronization and its unproven historical timing. Hosted release status remains recorded in [PR #599](https://github.com/CloudSpaceLab/clearsight-grc/pull/599).

## Acceptance

Organization and the final visible-truncation correction pass the scoped local browser/evidence gate. Groups of organization units and operating-jurisdiction selection remain unsupported; their truthful unavailable panels retain the source navigation positions and existing alternatives. This is **partial V2 layout integration**. Whole-screen acceptance remains **0/19**, and the production cutover block remains in place. Live four-actor execution, backend authorization and source-to-report reconciliation require separate retained evidence. [PR #599](https://github.com/CloudSpaceLab/clearsight-grc/pull/599) is the release record for the published revision, merge and demo deployment receipts.
