# Automation & governance — V2 layout integration

Date: 1 November 2026 · Tracking: #462 · Starting main: `f9b5bfd9c27641c8acb7928b2e59d8c59b00ec64`

## Problem and source

The deployed Automation view repeated Configuration, Automation and Automation policies headings and used an expandable list without the approved local navigation, loaded summaries, filters or selected-policy panel. The first policy already fitted the initial viewport at all five baseline widths; this change addresses the missing page composition, not an absent first record.

Layout authority: `docs/design/references/clearsight-2026-10/source/clearsight-automation-governance-designs/automation-governance.html`, SHA-256 `e24b9c012375175a2692be98adb2741dbea353cae0979b919fb0a18261583a98`.
The self-contained render source, `automation-governance-preview.html`, has SHA-256 `7dad1ffc651fd03821cc996ff13f1203cb1ec1344a8a5bdec7b00b49035a7b6f`. Original HTML, CSS, JavaScript, notes and manifest stay immutable.

The preceding People #589, Work #597/#598 and Enterprise Organization #599 tranches are merged. Main's CI `41207566044`, UI review `41207566080` and exact-SHA demo deployment `41208368371` passed. Their receipts remain distinct from this tranche.

## Composition and existing ownership

- Preserve `#configure/automation` and the shared Settings entry. This specialist view owns one heading and the source-order **Automations / Schedules / Escalations / Activity** local menu; the general Configuration sidebar no longer repeats inside it.
- Use the existing shared Tabs, Button, SearchField, SelectField, SelectableRecord, MetricCard and StatusBadge. Four compact summaries describe loaded latest revisions: all, Active, Draft and Suspended. Choosing a summary clears other filters so its list reconciles to the displayed count.
- Put the policy heading/count before the query controls, followed by the selectable list. The selected context follows the source's rule-before-configuration order: recorded eligibility, maximum scope and outcome checks, then effective dates and update time.
- At 1440/1280px the list sits beside a 330px context panel. At 1100px and below the context follows the list and summaries become two columns. At phone width search occupies a full row, the two selectors share the next row and policy records become labelled cards. Every enabled control retains its shared 44px target.
- Explicit narrow-screen selection focuses and scrolls the context below the sticky shell; Back to policies restores focus to the selected row. Loading and filter changes do not move focus. A direct missing policy identity remains explicit rather than silently selecting another policy.
- Store tab, selected policy, search, action class and status in the existing hash query. Back/Forward and direct entry restore them. Filter changes clear an explicit selection; peer tabs preserve all filters and selection.
- Schedules and Activity retain source menu positions with truthful unavailable panels. Escalations links to the existing `#configure/access/escalation` owner. No editor, simulation, activation, schedule, execution or delivery action is invented.

## Read contract and meaning

`GET /api/v1/compliance/automation-policies` continues to require verified configuration-read permission. Its additive envelope retains `items` and adds `tenant_id`, `scope: TENANT`, `revision_basis: LATEST_PER_CODE`, `limit`, `returned_count` and `truncated`.

The repository selects the latest revision **per policy code**, independent of lifecycle state. A newer draft can replace an older Active revision in this list; neither loaded Active counts nor an empty list prove which revisions are executing. PostgreSQL and memory use policy-code ordering and a 201st sentinel, while the service returns at most 200 revisions. No complete total, execution result or entity-specific policy population is inferred. The visible truncation notice precedes the register; search and filters cover the loaded revisions only.

The response tenant identifier matches the verified runtime-context identifier. Existing row tenant aliases may be slugs even when the verified context uses a UUID; the envelope is checked without rejecting those aliases. No schema, policy store, approval service or mutation endpoint changes.

The UI waits for verified context and explicit configuration-read capability. Tenant, entity, actor, session or capability changes immediately remount the workspace and cancel obsolete reads. Late responses are ignored. Refresh clears previous records, failures remain separate from empty results, and retry performs a fresh read. Invalid population metadata or mismatched tenant envelopes are not displayed.

Stored guardrail values remain unchanged; absent values are labelled **Not recorded**. Long names, codes, values and keys wrap within the context. Dates show recorded facts, not a computed execution status. The policy panel does not infer successful action or verified outcome from an Active label.

## Verification and acceptance

The evidence matrix compares the original light source, deployed baseline and runtime at 1440/1280/1024/768/390 in both runtime themes. It measures heading/menu/register order, summary columns, context proportions/replacement, the complete first policy, sticky-header clearance, 44px controls and horizontal overflow. Browser behavior covers distinct selection through Back/Forward/reload, filters, narrow context focus/return, existing-route handoffs and loading/empty/truncated/read-only/denied/error/recovery/long-content states.

The initial implementation failed the tablet/phone first-record gate because shared metric minimum heights survived the local layout styling. It also split phone labels and allowed long guardrail keys to overflow. Scoped existing metric tokens, source-sized phone typography and key wrapping correct those defects; the original failed captures remain distinct from final evidence.

Independent harness review strengthened history to change the actual selected record and retained pre-reload transport observations so a reload cannot erase attempted commands. These checks do not replace live backend authorization.

Final results, inspected image identities and publication receipts belong in the [evidence record](../../evidence/2026-11-01-automation-v2/README.md) and implementation PR. This is **partial V2 layout integration**. Whole-screen operational acceptance remains **0/19**: real four-actor execution, complete policy lifecycle, execution/delivery history and production acceptance require separate evidence.
