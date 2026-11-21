# Scoped dashboards and reports — V2.12

## User-visible behavior

The organization selector remains the single operational location filter. A report URL now carries `org=all` or an exact organizational identifier. A branch includes its departments and units; a department includes its units; a unit includes only itself. Invalid or empty explicit location identifiers fail closed. A location is never inferred from a display name.

The dashboard names the selected location, shows approved operational risk positions, accepted NGN loss amounts, expected KRI return coverage, and actionable exceptions. Location comparisons use the next actual hierarchy level. They retain event counts beside monetary amounts and state that exposure denominators are unavailable; the chart is not a performance ranking.

The branch KRI collection remains assigned to its explicit branch population. A department, unit or nonparticipating branch has **no expected return in that cycle**. Such a view displays “not collected at this level” or “no return expected.” It does not turn the branch answers into department observations, a missing response, a nil return or a zero-risk conclusion. New frozen packs preserve the same distinction. Older packs retain their original captured rendering.

### Assurance boundaries

Program targets are classified using explicit organizational targets, engagement scope, and the original bank/program boundary. A supplier program with multiple engagements is filtered per target, so one matching engagement does not bring unrelated targets into a local cohort.

The assurance page and dashboard separate:

- **Assessment targets within the selected location:** the original target and its own applicable requirements, reviewed count and satisfied count.
- **Broader assurance context:** relevant bank-wide or multi-location targets. They remain labelled with their original assessment boundary and are not local compliance results.

No combined branch compliance percentage is calculated. Bank-target results require exact whole-organization report authority and full organizational coverage. If broader context is unavailable, no numeric result is inserted. When response assignments require broader access than the current reader possesses, the response page blocks the whole cohort; a general report omits the entire response annex with a captured explanation instead of presenting a smaller, misleading response total.

### Read access is to a complete reporting population

`reports.view` authorizes compiled report content. It does not grant access to operational detail pages, editing, approval or source exports in another module; those entry points retain their own module checks.

The selected location must be readable in full. For example, a reporter assigned to Ikeja can select Ikeja, but a whole-organization report does not silently become an Ikeja aggregate. It displays a blocker and links to fully accessible organizational subtrees. A child deny blocks a broader aggregate even if an ancestor has an allow. The allowed choices compress only fully covered subtrees; a parent with an excluded descendant is not offered as fully covered.

Shared or broader operational sources require access across their complete source boundary. A union of branch grants can cover the organizational hierarchy but does not create exact-root authority for global or unassigned source records.

The report library is explicitly a list of **accessible saved views and captured packs within the selected location**, not an organization-wide aggregate. Published packs are checked against their captured population and any captured broader context. Changing the navigation location does not recalculate their content.

## Report authoring, exports and operational follow-ups

| Action | Required current capability |
|---|---|
| Read a live report | `reports.view` over the complete selected population and included broader source boundaries |
| Save a personal live view | `reports.view`; no report-author grant is inferred |
| Create or edit a report draft | `reports.view` and `reports.edit` over the complete captured population |
| Publish a frozen report | The draft permissions above plus `reports.submit` over that population |
| Export report JSON/CSV, selected loss rows or print through the report UI | `reports.export` over the complete relevant population, with current read checks |
| Open/export a published pack | Current read/export permission over its captured scope and captured contextual source boundaries |
| Create an exception follow-up | Current report read access plus `risks.edit` or `kri.edit` for the underlying source; rechecked at commit |

Report readers and board observers stay read-only. Without `reports.edit`, the draft page shows a captured preview without editable inputs. Prepare/edit/publish controls explain the missing capability. Capture-phase event guards stop legacy draft input/change handlers after permission revocation. Publishing does not constitute approval of a risk, loss, response, control, document or management decision.

The follow-up dialog rechecks the current operator, current source and source edit authority before creating a linked action. A reporting grant does not confer operational preparation rights. Existing canonical duplicate prevention, scope checks, due-date checks and source workflow checks continue to apply.

No new production or seeded role grants are added. In particular, the seeded CRO's reporting-read/export grant is not silently upgraded to author or publisher.

## Portable context

Live reporting links preserve the exact organizational identifier, supported occurrence-window selection, KRI cycle, explicit operational-loss date basis/window/currency/group, and applicable response filters. The original source identifier and target remain intact. Source links can therefore carry context into Documents, assessments and operational records without changing the source's own access rule.

The organizational selectors call the existing `scopeChange` workflow. The adapter prevents an old URL location from undoing the newly selected location during its render, then updates the URL. The cycle selector updates the URL before the existing cycle-render handler. Loss-filter form transitions make inherited scope and resolved defaults explicit in the resulting URL.

Saved live views retain their complete report route and settings in an additive `analyticsContext` property on the existing `S.savedInsightViews` records. Opening a view revalidates its hierarchy, period, cycle, perspective, route/scope agreement and current read access before changing navigation state. Old saved views still use their captured scope, period, cycle and grouping; missing old settings are not reconstructed as historical facts.

Report CSVs retain their existing financial/source columns and add:

- Exact report scope ID and readable hierarchy path.
- Workspace snapshot date.
- Loss date basis, explicit start/end, and reporting currency.
- Selected KRI cycle and its reporting month.
- A route containing the reporting context.

There is no foreign-exchange conversion. The older operational summary remains NGN only; the dedicated loss report reports one chosen currency at a time. Occurrence, recognition, reporting month and cash receipt/reversal dates retain their canonical meanings. New accepted-loss, verified-receipt, reversal and near-miss calculations are not reimplemented by this module.

## Snapshot contract

New captures add `snapshot.scopedAnalytics`:

```js
{
  version: '2.12',
  scopeId, scope, label, kind, path,
  descendantScopeIds,
  requiredScopeIds,
  programTargets: [{
    programId, programName, target, targetName,
    scopeIds, cohort, boundary, scopeLabel, resultIncluded
  }],
  settings: {period, cycle, loss, responses},
  route,
  basis
}
```

The report capture, source calculations, board commentary and publication store remain canonical. `requiredScopeIds` records broader included boundaries for later access checks. Numeric results absent because of missing broader authority remain absent in that snapshot even if a future reader has wider access.

The frozen renderer uses captured metadata only. It does not add live labels, scope paths, target posture, financial values or a new annex to older packs. The focused suite compares both new and old frozen output byte-for-byte after navigation and source mutations.

## Integration API

Load `assets/scoped-analytics.js` after `role-experience.js` and the existing V2.11 interaction/report adapters, before later document/external adapters and bootstrap. Load its CSS with the application styles. It has no top-level render or boot call. Its narrow render wrapper calls `applyRoute()` before the previous render; later wrappers must preserve that chain.

```js
CLEAR_SIGHT_SCOPED_ANALYTICS.context({
  permission: 'reports.view',
  scopeId: selectedScope()
})
// {valid, scopeId, scope, label, kind, path,
//  descendantScopeIds, permission,
//  access:{any,full,allowedScopeIds,deniedScopeIds,roots,reason,...}}

CLEAR_SIGHT_SCOPED_ANALYTICS.route(base, patch = {}, context?)
// Preserves base route/query identifiers, adds explicit location and supported
// context fields. Null/undefined/empty patch values remove that query field.
// An explicit {scopeId,params:''} context does not inherit current filters.

CLEAR_SIGHT_SCOPED_ANALYTICS.applyRoute()
// Applies explicit org and valid cycle before canonical render.

CLEAR_SIGHT_SCOPED_ANALYTICS.sourceAccess(scopeIds, permission?)
CLEAR_SIGHT_SCOPED_ANALYTICS.snapshotAccess(snapshot, permission?)
CLEAR_SIGHT_SCOPED_ANALYTICS.programCohort(scopeId?)
CLEAR_SIGHT_SCOPED_ANALYTICS.chips(context, {frozen,settings}?)
```

The role module remains authoritative for `capability` and descendant `coverage`. This adapter never changes grants or silently filters the selected population to an actor's allowed rows. No new source store, duplicate financial model or background service is introduced.

## Validation and regression fixtures

- `node tests/scoped_analytics_v212_checks.js`: **24/24**, actual application sources with minimal DOM substitution. Covers hierarchy/invalid scope, partial coverage and child denies, whole-organization context, target-level filtering, KRI populations, route/selectors, saved-view restoration/revalidation, CSV metadata, current export checks, frozen pack immutability, independent author/publisher permissions, read-only inputs, source follow-up permissions and invalid-filter recovery.
- `node tests/loss_reports_v29_checks.js`: **18/18** after the root-authorized fixture setup change. Existing financial, scope, source, offset, currency and snapshot assertions are unchanged.
- `node tests/board_reports_v28_model_checks.js`: **11/11**; the existing pure model suite remains separate from report UI authorization.

The V2.9 report suite's original file is retained in `tests/history/loss_reports_v29_checks.pre_v212.js`. Its only setup changes select `USR-02`, the seeded whole-organization CRO reader/exporter, at its report runtimes. For report-author tests, add explicit synthetic `reports.edit` and `reports.submit` grants over the captured population. A root capture containing bank-wide targets requires exact-root authority as well as complete descendant coverage. These are fixture grants, not application seed changes.

Browser layout was not validated in this environment. The module uses the existing responsive panels/tables and small context-chip styles, with printable captured context. The local application demonstrates role-conditioned UI and command checks; local state and direct JavaScript access do not constitute a production authorization boundary. Server-side retrieval, export and mutation controls remain necessary for deployment.
