# Enterprise Setup — Organization V2

Date: 31 October 2026 · Tracking: #462 · Starting main: `c40f4edd7ad4107fc2b36ebfd65af87bd5d5502b`

## Problem and source

Organization currently opens inside the People workspace, ahead of a data-boundary form, area table and position table. The approved Enterprise Setup page leads with a searchable hierarchy, compact structure/change context, selected-unit detail and contextual actions. At all five required widths, the retained current-runtime baseline shows no area or position record in the initial viewport.

Layout authority: `docs/design/references/clearsight-2026-10/source/clearsight-enterprise-setup-designs/enterprise-setup.html`, SHA-256 `5182fcd83331fafd42c7711fa6eafdbebb5d255dec51b6b73d42217e8ebdfd3a`. The self-contained `enterprise-setup-preview.html` has SHA-256 `cc7542e2a3047338b1d2fbbbe33756adbc9ee8c2084fb5920834da8aeedc04c7`. Original HTML, CSS, JavaScript, notes and manifest remain immutable.

## Composition and routing

- Keep `#configure/access/positions` as the Organization destination. Replace its People heading/contextual tab with one Enterprise setup heading, a compact verified-entity banner and source-order Organization / Groups / People & access / Jurisdictions / Frameworks / Data imports navigation.
- Reuse `IdentityAccessPanel` for reads, permissions and all commands. Keep the Settings overview and existing specialist configuration owners. Manage organization in the shared scope selector opens this existing Organization destination.
- Place the hierarchy beside compact loaded-structure and pending-change information; retain the lower contextual cards and selected-unit rail. At 1300px the rail follows the primary region; at 780px the inner hierarchy/summary columns stack; at 560px lower cards and context become one column. Use shared controls and theme tokens, 44px targets and concise operational copy.
- Search only returned areas and keep ancestry visible. Collapse is separate from selection. Selected-area identity is a bounded, encoded `unit` URL value; missing selections remain explicit. Guard missing parents and cyclic relationships without inventing ancestry. Selection on narrow screens focuses the context; a return action restores the selected row.
- Move positions/roles and data-boundary editing into contextual focused sheets. Reuse their existing proposal, independent decision, version, simulation, effective-date, history and restore implementations. Area proposal/review remains in `OrganizationScopeManager`.
- Groups and Jurisdictions retain their source menu positions with truthful unavailable panels. Directory groups are not groups of organization units; data residency is not an operating-jurisdiction selection. People, Frameworks and Imports hand off to their existing owners. Cross-workspace navigation must notify App rather than only changing Configure's local hash.

## Data and authority boundary

The access overview returns at most 500 current-entity areas with a truncation flag, 50 positions and tenant-wide profiles, and bounded proposal/history collections. It has no complete population total, cursor, overview timestamp or atomic snapshot. Counts describe loaded records; local search cannot retrieve omitted areas. An assigned occupant is not necessarily an active person.

Prototype six-area readiness, ownership coverage, member totals, invitations and reusable organization groups have no matching contracts. The header/banner use the verified current entity; structure and change summaries use their actual loaded populations. No readiness percentage, compliance conclusion, source example or additional organization store is introduced.

All writes retain the existing conjunction of identity configuration and governance-write permissions, verified actor/entity scope, optimistic version and maker/checker checks. A restore is a new proposal. Stale or denied reads cannot enable commands. Leaving a view or changing tenant/entity/actor closes its editors and discards inappropriate selection/data.

An existing PostgreSQL constraint excludes imported `ORGANIZATION_UNIT` kinds from direct Move/Retire revisions. This frontend tranche requires an explicit managed-kind Update and independent approval before enabling those actions. An imported-area edit must not silently choose Department. The direct API inconsistency remains a backend limitation; no migration is included here. A separate narrow correction hides Restore when configuration permission is absent.

## Verification plan

Retain same-screen original, before and after renders at 1440/1280/1024/768/390 in light/dark. Assert heading/menu order, hierarchy/summary/context replacement, first useful record, selection/search/collapse, context focus/return, 44px controls and no page overflow. The original is light-only; dark is the runtime adaptation.

Exercise loaded/empty/truncated/read-only/denied/error/retry states, missing-parent/cycle cases, missing selections, browser history, real destination handoffs, independent area review, explicit imported-kind conversion and editor closure. Preserve existing position, role, boundary, scope-remount and provisioning-token tests. Bind fixture/capture/source/build/image identities; exclude test transport from the customer build. Named reviewers must inspect final renders and the hosted artifact before merge.

## Acceptance

Implementation and rendered evidence are in progress. This is **partial V2 layout integration**. Whole-screen acceptance remains **0/19**. Live four-actor execution, backend authorization, source-to-report reconciliation and production cutover require separate retained evidence. The PR will record exact-head checks, merge and demo deployment receipts.
