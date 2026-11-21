# Framework Library inspector — partial V2 integration

Source: [Framework Library HTML](../../design/references/clearsight-2026-10/source/clearsight-framework-designs/framework-library.html), SHA-256 `193d13b0cf87381f20b7d84c3d4c50c80b0e064274a623fa98257506df8284bf`. The [decision](../../design/2026-11-03-framework-inspector-width.md) scopes the change. Reviewer: ChatGPT Codex `/root`, 3 November 2026, local uncommitted working tree; no immutable revision or production acceptance is asserted.

At 1440px and 1280px, the original source has a 340px inspector. The runtime previously allocated 394px ([1440px before](../2026-11-02-framework-alignment/before-inspector-1440.png)); [after](light-1440.png), the inspector is 340px and the register is 54px wider. The supported filters, explicit selection, catalog rights and adoption handoff remain unchanged. Source examples are not runtime catalog records; the source has no dark theme.

The [gate manifest](gate/manifest.json) records seven passing browser scenarios covering the five approved widths, selection, phone sheet and denied catalog. Additional light and dark captures at 1440, 1280, 1024, 768 and 390px show no document overflow. The inspector remains 310px at 1024px and is replaced by the focused sheet at compact widths. The Framework and navigation tests and copy-quality checks passed 20 tests; 18 UI and source integrity checks and TypeScript passed. A desktop automated WCAG A/AA scan had no definite violations, with `aria-prohibited-attr` and `color-contrast` incomplete and requiring manual review.

Synthetic fixture results do not establish catalog authorization, live backend result quality or whole-screen acceptance. The acceptance register remains 0/19 and production cutover remains blocked.
