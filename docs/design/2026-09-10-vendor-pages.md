# Vendor dashboard and register

## Decision and scope

The user requested shorter vendor widgets and separate dashboard/register pages while other work continues. Keep `#vendors` as the dashboard; add `#vendors/register` and accept `#vendors/dashboard`. Preserve `#vendors/{relationshipID}` record links. Two labelled navigation links identify the active page. The dashboard contains scoped metrics and findings/actions; the register contains search, service rows, selection and form requests. Exact vendor records retain their existing overview and Back to vendor register action.

Keep existing components and data boundaries rather than introduce another top-level module or hide the register beneath a collapsible dashboard. Cards use 12px vertical padding and remove the value's extra vertical padding, retaining 44px action targets and automatic height for Unknown/wrapped labels. The vendor count joins the loaded-service scope line rather than adding another line to every card's height. Four desktop columns become two, then one at the existing narrow breakpoints. No new tokens or shared component variants.

## Implementation and acceptance

1. Preserve the supplied screenshot and capture metric heights at 1440/390 before changing card spacing.
2. Add route tests and workspace separation/navigation tests; verify failures first.
3. Extend the route target with a vendor page; pass it from App, retain detail targets, clear selection on browser navigation, and render only the requested workspace.
4. Tighten vendor metric spacing in its existing stylesheet, including selected-vendor metrics.
5. Run route, vendor workflow, copy-quality tests and typecheck/build. Render dashboard, register and record at desktop/mobile in light/dark; check empty/error states, keyboard focus, navigation/history, card heights and overflow.

Dashboard counts retain their loaded-service scope. Register searches must not silently narrow the dashboard after page navigation. Partial reads remain Unknown. This is a local UI change; no deployment or material command authority changes.

## Local acceptance — 10 September 2026

- 134 tests passed across App, appRouting, VendorsWorkspace, VendorPortfolio and copyQuality; TypeScript checking passed.
- Production and evidence Vite builds passed, using separate temporary output directories to avoid the concurrent task's builds.
- 28 captures cover dashboard, register, exact record, no linked findings and unavailable findings/form summaries at 1440px and 390px in both themes. All passed horizontal-overflow checks. Unknown remains visible; notices and the optional guide do not block page navigation. Record opening, Back to vendor register and browser back/forward passed at all four viewport/theme combinations.
- Desktop metric height: 194.92px before spacing changes, 140.92px after (27.7% reduction). Mobile metrics now use 130.55px in these fixtures; height remains content-driven.
- Visual inspection removed the repeated register heading and the extra card scope line. Baseline, final renders and measurements are in [the evidence directory](../evidence/2026-09-10-vendor-pages/manifest.json). `user-before.png` preserves the supplied image; `before-*.png` preserves card geometry before the spacing change, after route separation.
- All changes remain local and uncommitted alongside the ongoing task. No data, authority or deployment change is included.
