# Shell scope placement and oversight period refinement — decision brief

Date: 2026-10-02
Parent: #267
Baseline: `main@61f144ae1284fcebc0bdc7fd648d393f070b915a`

## Decision

Move the enterprise hierarchy action from the passive organization context on the left side of the shell into the right-side action cluster, occupying the space previously used by the redundant non-production badge.

The left side remains plain orientation: organization and current legal entity. The right side owns actions: notifications, display/admin/demo tools, current role and the hierarchy switcher.

Remove the separate `Non-production data` / `Stakeholder demo` pill. The existing Demo environment control remains the explicit entry point and identifies sample/non-production data inside its menu.

On Home, replace the vague `Current snapshot` card with the actual stored reporting period and put freshness/generated time underneath. Remove the duplicate period and projection-version text from the summary line; projection details remain under Data freshness.

## Date-range boundary

The current oversight backend stores a five-minute series of **90-day rolling snapshots**. It does not support an arbitrary start/end query, and current-open counts are not safely reconstructable by filtering the browser.

Therefore this change does **not** render an enabled date picker that cannot change the query. A future interactive range control must be backed by bounded server-side period semantics for both `/api/v1/oversight` and `/api/v1/metrics/home`, with point-in-time/current-state meaning made explicit.

## Authority and truth

- scope switching remains server-authorized and session-bound;
- the hierarchy action still renders only for a COMPLETE authorized hierarchy;
- removing the environment badge changes no environment/runtime behavior;
- the Home period comes directly from `period_start` and `period_end` returned by the projection;
- freshness and coverage remain visible and are not inferred from UI state.

## Responsive behavior

The right-side scope action is bounded on desktop, wraps with the shell controls on tablet, and takes the available row width on narrow screens. Passive organization/legal-entity context remains readable even when the switcher is unavailable.
