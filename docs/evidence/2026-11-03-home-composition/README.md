# Home composition — reference-guided integration

The user authorized a Home design based on other approved HTML because no
standalone Home HTML is present in the verified archive. The [decision](../../design/2026-11-03-home-composition.md)
uses the open heading, local navigation and shared control patterns in the
[Framework Library source](../../design/references/clearsight-2026-10/source/clearsight-framework-designs/framework-library.html)
(SHA-256 `193d13b0cf87381f20b7d84c3d4c50c80b0e064274a623fa98257506df8284bf`)
and [Program Workspace source](../../design/references/clearsight-2026-10/source/clearsight-program-designs/program-workspace-preview.html)
(SHA-256 `b91aa4a96bfae4fc712f49e883996eb770d74dc3ae4b588ab6e573b4ec9d846f`).
These are specialist-page references, not same-screen Home comparisons.
Reviewer: ChatGPT Codex `/root`, 3 November 2026, local uncommitted working
tree; no immutable revision or production acceptance is asserted.

The [entity Home before](before-light-1440.png) and [after](final-review/home-t7-entity-light-desktop.png)
show the boxed heading and separated navigation replaced by an open heading,
adjacent three-view navigation, scope, posture and freshness. Full-shell
captures such as [desktop](final-review/home-t7-shell-light-1440.png) and [phone](final-review/home-t7-shell-light-390.png)
use an explicitly labelled synthetic sample. The phone tabs now fit one row,
and metric labels no longer break into individual letters. The original
capability-disabled [Home fallback](fallback-light-390.png) remains available.
The first phone posture card begins at approximately y=728px in the 844px
viewport with the sample guide collapsed; the entire card is not initially visible.
The earlier `geometry-gate` retained a failure with the first-run guide open;
the final guide scenario verifies dismissal before reviewing the ordinary layout.
The [issue-aging bars](aging-final-light.png) now use the shared blue and surface
tokens instead of the browser-default green.

The [browser receipt](final-review/home-t7.json) covers 21 entity, Group,
full-shell and unavailable scenarios, including light/dark, all five required
widths, first-run guide dismissal, 320px and a 200% CSS-zoom proxy. It checks source/freshness ordering,
first-viewport content, single-row phone tabs, readable label width, 44px
targets, no document overflow, Group scope and keyboard interaction. Additional
manual keyboard checks covered entity tabs and the period picker; scoped
unavailable fixtures retain the retry. The fixture does not prove live
authorization, production backend quality, actual browser zoom or source-to-report
reconciliation. The evidence runner found no definite axe A/AA violations but
reported incomplete checks requiring manual review.

The Home, Group, posture and copy tests passed 34 tests; the isolated shell
fixture passed two further tests, and TypeScript, source-integrity and evidence
build checks passed. This is reference-guided integration, not a claim of exact
Home source parity or whole-screen acceptance. The production cutover remains
blocked and the acceptance register remains 0/19.
