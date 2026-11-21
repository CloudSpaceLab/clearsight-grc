import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { validateCssSource, validateTsxSource } from "./ui-contract.mjs";

const read = (path) => readFile(new URL(`../${path}`, import.meta.url), "utf8");

test("the v2.10 shell supports the 320px viewport", async () => {
  const styles = await read("src/v210/fresh.css");
  const body = styles.match(/body\s*\{([^}]*)\}/i)?.[1] ?? "";
  const minimum = body.match(/min-width:\s*(\d+)px/i);
  assert.ok(!minimum || Number(minimum[1]) <= 320, "shell minimum width exceeds 320px");
});

test("retired global CSS stays absent and the fresh shell uses inherited control typography", async () => {
  await assert.rejects(read("src/styles.css"), { code: "ENOENT" });
  const styles = await read("src/v210/fresh.css");
  assert.match(styles, /button\s*,\s*input\s*,\s*select\s*,\s*textarea\s*\{\s*font:\s*inherit;\s*\}/i);
  assert.doesNotMatch(styles, /(?:^|[},])\s*button\s*\{[^{}]*color\s*:/ims);
});

test("v2.10 data cards wrap long identifiers within the available column", async () => {
  const styles = await read("src/v210/fresh.css");
  assert.match(styles, /\.v-program-table td\s*\{[^}]*min-width:\s*0[^}]*overflow-wrap:\s*anywhere/is);
});

test("focused-sheet close actions remain above sticky feature content", async () => {
  const [component, styles, tokens, consumer, browser, workflow] = await Promise.all([
    read("src/v210/FocusedSheet.tsx"), read("src/v210/focused-sheet.css"), read("src/v210/tokens.css"),
    read("src/v210/QuestionOutline.tsx"), read("scripts/v215-focused-sheet-evidence.mjs"), read("../.github/workflows/ui-evidence.yml"),
  ]);
  assert.match(component, /\.showModal\(\)/);
  assert.match(component, /onCancel/);
  assert.match(styles, /\.v-focused-sheet-header\s*\{[^}]*position:\s*sticky[^}]*z-index:\s*var\(--v-sheet-header-layer\)/s);
  assert.match(tokens, /--v-sheet-header-layer:\s*var\(--v-layer-sheet-header\)/);
  assert.match(tokens, /--v-layer-sheet-header:\s*var\(--v-primitive-layer-sheet-header\)/);
  assert.match(tokens, /--v-primitive-layer-sheet-header:\s*2/);
  assert.match(consumer, /<FocusedSheet/);
  assert.match(browser, /elementFromPoint/);
  assert.match(browser, /:modal/);
  assert.match(workflow, /node scripts\/v215-focused-sheet-evidence\.mjs/);
});

test("raw controls report an exact migrated-file diagnostic", () => {
  const diagnostics = validateTsxSource({
    file: "src/components/forms/sent/SentFormsFilters.tsx",
    source: "export const Filters = () => <select aria-label='Status'><option>Open</option></select>;",
  });
  const result = new Error(diagnostics.join("\n"));
  assert.match(result.message, /src\/components\/forms\/sent\/SentFormsFilters\.tsx:\d+.*use SelectField/);
});

test("an exact documented native-control exception is accepted", () => {
  const diagnostics = validateTsxSource({
    file: "src/components/example/NativeDate.tsx",
    source: "export const NativeDate = () => <input type='date'/>;",
    exceptions: [{ file: "src/components/example/NativeDate.tsx", tag: "input", inputType: "date", reason: "The bounded browser date picker is the approved date-entry contract." }],
  });
  assert.deepEqual(diagnostics, []);
});

test("direct React Aria imports outside the shared boundary are rejected", () => {
  const diagnostics = validateTsxSource({
    file: "src/components/forms/UnboundedControl.tsx",
    source: "import { Button } from 'react-aria-components'; export const Control = Button;",
    scanNative: false,
  });
  assert.match(diagnostics.join("\n"), /import shared controls from components\/ui/);
});

test("feature CSS rejects palette and interaction overrides even when shared selectors are scoped", () => {
  const diagnostics = validateCssSource({
    file: "src/forms-sample.css",
    source: ".sample .cs-button { color: #fff; border-radius: 9px; transition: color 200ms; z-index: 99; }",
  }).join("\n");
  assert.match(diagnostics, /color is owned by the shared component/);
  assert.match(diagnostics, /color must use a design token/);
  assert.match(diagnostics, /radius token/);
  assert.match(diagnostics, /duration token/);
  assert.match(diagnostics, /z-index token/);
});

test("the legacy CSS validator permits documented scoped geometry contracts", () => {
  // This tests the validator in isolation. It does not assert that the retired
  // component token file belongs to the new runtime.
  const componentTokens = new Set(["--cs-action-card-padding", "--cs-metric-card-gap", "--cs-metric-card-heading-columns", "--cs-metric-card-label-transform", "--cs-metric-card-label-spacing"]);
  const diagnostics = validateCssSource({ file: "src/components/reports/source-layout.css", componentTokens, source: `
    .reports .cs-selectable-record { min-block-size: 74px; --cs-action-card-padding: var(--cs-primitive-space-3); }
    .reports .cs-metric-card { --cs-metric-card-gap: var(--cs-space-inline); --cs-metric-card-heading-columns: minmax(0, 1fr); --cs-metric-card-label-transform: none; --cs-metric-card-label-spacing: 0; min-inline-size: 0; }
    .reports > .cs-tabs .cs-tabs__list { display: flex; flex-wrap: nowrap; overflow-x: auto; gap: var(--cs-space-group); }
    .reports .cs-button { justify-self: start; min-height: var(--cs-control-height); }
    .reports .cs-metric-card__label { white-space: normal; overflow-wrap: anywhere; font-size: var(--cs-font-size-caption); }
    @media (max-width: 700px) { .reports .cs-button:first-child { grid-column: 1 / -1; } }
  ` });
  assert.deepEqual(diagnostics, []);
});

test("feature geometry exceptions do not permit global duplicates or unscoped selector branches", () => {
  for (const selector of [".cs-button", ".reports .cs-button, .cs-button", ".cs-button .reports"]) {
    const diagnostics = validateCssSource({ file: "src/reports.css", source: `${selector} { padding: var(--cs-space-inline); }` }).join("\n");
    assert.match(diagnostics, /layout must be scoped by a feature class/, selector);
  }
});

test("feature geometry exceptions do not permit focus, state, content or interaction overrides", () => {
  for (const selector of [".reports .cs-button:focus-visible", ".reports .cs-button:hover",
    ".reports .cs-button[data-pressed]", ".reports .cs-button[aria-pressed=true]", ".reports .cs-button::after"]) {
    assert.match(validateCssSource({ file: "src/reports.css", source: `${selector} { width: 100%; }` }).join("\n"), /state selectors belong/);
  }
  const diagnostics = validateCssSource({ file: "src/reports.css", source: `
    .reports .cs-button { color: var(--cs-text-primary); outline: none; pointer-events: none; opacity: 1; }
    .reports .cs-button { display: none; overflow: hidden; width: 100% !important; }
    .reports .cs-ranked-bars__heading { display: contents; }
  ` }).join("\n");
  for (const property of ["color", "outline", "pointer-events", "opacity"]) assert.match(diagnostics, new RegExp(`${property} is owned by the shared component`));
  assert.match(diagnostics, /must not hide/);
  assert.match(diagnostics, /preserve content and focus/);
  assert.match(diagnostics, /must not override the cascade/);
});

test("feature custom properties must be existing geometry contracts, including inherited overrides", () => {
  const diagnostics = validateCssSource({ file: "src/reports.css",
    componentTokens: new Set(["--cs-action-card-padding", "--cs-button-primary-bg", "--cs-button-focus"]),
    source: `.reports { --cs-action-card-padding: var(--cs-space-inline); --cs-button-primary-bg: var(--cs-action-primary); --cs-button-focus: none; --cs-invented-width: 20px; }
      :root { --cs-action-card-padding: 0; }`,
  }).join("\n");
  for (const property of ["--cs-button-primary-bg", "--cs-button-focus", "--cs-invented-width"]) assert.match(diagnostics, new RegExp(`${property} is not an existing component geometry token`));
  assert.match(diagnostics, /layout must be scoped by a feature class/);
  assert.doesNotMatch(diagnostics, /--cs-action-card-padding is not/);
});

test("inherited component geometry cannot bypass feature scope or interaction ownership", () => {
  const componentTokens = new Set(["--cs-action-card-padding"]);
  for (const selector of [".reports, :root", ".reports, body", ".reports .cs-button, :root", ".reports:hover", ".reports[data-selected]", ".reports:focus-within"]) {
    const diagnostics = validateCssSource({ file: "src/reports.css", componentTokens,
      source: `${selector} { --cs-action-card-padding: 0; }`,
    }).join("\n");
    assert.match(diagnostics, /scoped by a feature class|state selectors belong/, selector);
  }
});

test("feature control geometry preserves shared height contracts", () => {
  const componentTokens = new Set(["--cs-button-height"]);
  const accepted = validateCssSource({ file: "src/reports.css", componentTokens,
    source: ".reports { --cs-button-height: var(--cs-control-height); } .reports .cs-select-field__trigger { min-block-size: var(--cs-field-height); }",
  });
  assert.deepEqual(accepted, []);
  for (const value of ["20px", "1rem", "0", "var(--cs-primitive-space-2)"]) {
    const diagnostics = validateCssSource({ file: "src/reports.css", componentTokens,
      source: `.reports { --cs-button-height: ${value}; } .reports .cs-search-field__control { min-block-size: ${value}; }`,
    }).join("\n");
    assert.match(diagnostics, /--cs-button-height must use a control-height token/, value);
    assert.match(diagnostics, /min-block-size must use a control-height token/, value);
  }
});

test("the replacement import details inherit the shared surface without a local palette", async () => {
  const [source, component, shell, tokens] = await Promise.all([
    read("src/v210/document-imports.css"), read("src/v210/DocumentImports.tsx"),
    read("src/v210/fresh.css"), read("src/v210/tokens.css"),
  ]);
  const diagnostics = validateCssSource({ file: "src/v210/document-imports.css", source });
  assert.deepEqual(diagnostics, [], diagnostics.join("\n"));
  assert.doesNotMatch(source, /#[\da-f]{3,8}\b|(?:rgb|hsl)a?\(/i);
  assert.match(component, /className="v-panel v-imported-document-detail"/);
  assert.match(shell, /\.v-metric,\s*\.v-panel\s*\{[^}]*background:\s*var\(--v-panel-background\)/s);
  assert.match(tokens, /--v-panel-background:\s*var\(--v-color-surface\)/);
  assert.match(source, /@media\s*\(max-width:\s*850px\)[\s\S]*?grid-template-columns:\s*minmax\(0,\s*1fr\)/);
});
