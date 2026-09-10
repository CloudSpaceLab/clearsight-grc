# Vendor exception overview design

**Date:** 10 September 2026  
**Status:** Approved interaction design  
**Scope:** Vendor Overview only; the Vendor Register and canonical Matter records remain separate.

## Purpose

The Vendor Overview must help a CRO or Program Owner identify the vendor exceptions requiring attention, their accountable owner, the next action and the deadline without scrolling through oversized metrics or opening every record. It is an operational index into governed Matters, not a second case-management workspace.

## Daily workflow

1. The reviewer opens Vendor Overview and sees the loaded population, freshness and four reconciled counts in one compact line.
2. The default **Needs attention** queue ranks overdue, blocked and incomplete work before remaining open exceptions.
3. Each row exposes the exception, vendor and service, recorded source rating, action owner, next action, deadline state and workflow state.
4. The reviewer filters by attention state, vendor, owner or source rating only when that changes the decision.
5. **Review exception** opens the existing Matter record for evidence, commentary, history, reassignment and governed state changes.
6. Browser Back returns to the same filter and list position.

Assigned performers complete work through Today/My Work or the Matter. Assessors review in the Matter. External vendors continue through governed forms. The Vendor Overview links to this work without recreating those workspaces.

## Information architecture

### Compact status line

Replace the four metric cards with a single summary region, targeted at 72–88px high on desktop:

`1 vendor · 5 open exceptions · 5 open actions · 5 overdue · Checked 10 Sep, 2:46 PM`

Each count is a labelled control with at least a 44px target. Selecting a count applies the corresponding queue filter. The selected state remains visible through more than color. The summary contains no explanatory captions or duplicate “Review” links.

Counts must reconcile with the records returned for the loaded vendor-service population. A partial read displays **Unknown** and the existing recovery action; it never substitutes zero.

### Exception queue

Rename the listing **Vendor exceptions**. Use one full-width, bounded queue rather than cards, a spreadsheet grid or a duplicate detail pane. Desktop rows target 64–76px and retain a clear primary/secondary reading order:

- exception title;
- vendor and service;
- recorded source rating;
- action owner;
- next open action;
- relative deadline state, with the exact date secondary;
- current workflow state;
- one **Review exception** action.

The queue summarizes multiple actions as the earliest actionable deadline plus open/overdue counts. It does not render every action inside the row. Full recommendations, source ranges, comments, evidence, assessment history, reassignment controls and activity history remain in the Matter.

Because the hosted fixture population is explicitly sample data, one persistent scope indicator identifies the queue as sample data. Repeating “Sample data” on every row is unnecessary.

## Deterministic attention order

The client derives presentation order only from stored states and dates:

1. open exception with an overdue open action;
2. open exception with a blocked action;
3. open exception whose next action has no owner or deadline;
4. open exception with an action due within 30 days;
5. remaining open exceptions;
6. closed or cancelled records only when **All** is selected.

Within a band, order by earliest deadline and then exception title. Missing values remain explicit. The presentation does not infer material risk, compliance status or authority.

## Filters

Retain only controls that support an operating decision:

- Needs attention;
- Overdue;
- All open;
- Vendor;
- Owner;
- Source rating.

The default is **Needs attention**. No column chooser, configurable widgets, bulk selection, charts or saved-view machinery is introduced. Existing backend bounds remain authoritative; the client must not imply that a partially loaded population is complete.

## Navigation and continuity

Vendor Overview remains `#vendors/overview`; Vendor Register remains `#vendors/register`. **Review exception** opens the canonical Matter route. Return navigation restores the Overview filter and scroll position for the current browser session. The Overview must not add a flyout, nested scrolling region or alternate Matter representation.

## Responsive behavior

At desktop widths, the summary remains one compact region and several exception rows are visible in a 1440×900 viewport. At narrow widths, each row becomes a compact two-line record with exception, vendor, owner, deadline state and action retained. Secondary filters collapse into one labelled control. The page must not require horizontal scrolling, and fixed navigation must not obscure queue rows.

## Copy and accessibility

Visible copy names the record, state, owner, deadline or action. It does not explain the interface. Status and urgency use text in addition to color. Queue controls have visible focus, logical keyboard order and a minimum 44px target. The heading hierarchy remains sequential, and loading, partial, empty and unavailable states remain announced.

## State handling

- **Loading:** retain a compact reserved queue area to avoid layout shift.
- **Empty:** state the loaded population and active filter, then offer the next valid filter or Register action.
- **Partial:** show known rows, mark aggregate counts Unknown and retain the existing retry action.
- **Unavailable:** preserve the current fail-closed error and retry path.
- **No actionable deadline:** display **No deadline recorded**; do not rank it as current or overdue.

## Non-goals

- No new risk calculation, severity model, approval state or authority route.
- No duplicate Matter detail, side inspector or exception editing on Overview.
- No changes to vendor forms, Register behavior, seed semantics or backend commands.
- No generalized analytics/dashboard framework.

## Acceptance

- A reviewer can identify the most overdue exception, its vendor, owner and next action from the Overview without opening a record.
- Exceptions begin within approximately 220px of the Vendors heading at 1440px width.
- At least five representative exception rows are visible in a 1440×900 rendered state.
- Counts reconcile with the filtered source records; partial totals remain Unknown.
- Default ordering follows the deterministic attention bands.
- The queue shows no repeated source range, full recommendation or sample-data label per row.
- Opening and returning from a Matter preserves the active filter and practical reading position.
- Desktop, dark theme and 390px renders have no horizontal overflow or obscured actions.
- Focused component, routing, typecheck and production-build checks pass. Broad regression suites are outside this demo refinement.
