import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { validateCssSource, validateProject, validateTsxSource } from "./ui-contract.mjs";

const read = (path) => readFile(new URL(`../${path}`, import.meta.url), "utf8");

test("the design system declares the approved cascade and three token layers", async () => {
  const [entry, tokenEntry, primitives, semantic, components, packageJSON] = await Promise.all([
    read("src/design-system/index.css"),
    read("src/design-system/tokens/index.css"),
    read("src/design-system/tokens/primitives.css"),
    read("src/design-system/tokens/semantic.css"),
    read("src/design-system/tokens/components.css"),
    read("package.json"),
  ]);

  assert.match(entry, /@layer reset, tokens, base, components, features, utilities, overrides;/);
  assert.match(tokenEntry, /primitives\.css[\s\S]*semantic\.css[\s\S]*components\.css/);
  assert.match(primitives, /--cs-primitive-color-neutral-950:/);
  assert.match(semantic, /--cs-bg-canvas:/);
  assert.match(components, /--cs-button-height:/);
  assert.equal(JSON.parse(packageJSON).dependencies["react-aria-components"], "1.20.0");
});

test("the shell does not force document overflow at the supported 320px viewport", async () => {
  const styles = await read("src/styles.css");
  assert.doesNotMatch(styles, /body\s*\{[^}]*min-width:\s*320px/i);
});

test("legacy global CSS cannot override shared button typography or foreground", async () => {
  const styles = await read("src/styles.css");
  assert.doesNotMatch(styles, /button\s*,[^{}]*\{[^{}]*font\s*:/is);
  assert.doesNotMatch(styles, /(?:^|[},])\s*button\s*\{[^{}]*color\s*:/ims);
});

test("mobile data cards wrap long identifiers within the available column", async () => {
  const styles = await read("src/design-system/components/data-display.css");
  assert.match(styles, /\.cs-data-table tbody tr td\s*\{[^}]*min-inline-size:\s*0[^}]*overflow-wrap:\s*anywhere/is);
});

test("focused-sheet close actions remain above sticky feature content", async () => {
  const tokens = await read("src/design-system/tokens/components.css");
  assert.match(tokens, /--cs-overlay-close-z:\s*var\(--cs-z-menu\)/);
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

test("V2 feature layouts may arrange shared controls and configure existing geometry tokens", async () => {
  const tokens = await read("src/design-system/tokens/components.css");
  const componentTokens = new Set([...tokens.matchAll(/(--cs-[\w-]+)\s*:/g)].map((match) => match[1]));
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

test("document import surfaces reject local dark palette values", async () => {
  const source = await read("src/document-import.css");
  const diagnostics = validateCssSource({ file: "src/document-import.css", source });
  assert.deepEqual(diagnostics, [], diagnostics.join("\n"));
  assert.doesNotMatch(source, /rgba?\(/i);
  assert.match(source, /\.document-import-form\s*\{[^}]*background:\s*var\(--surface-2\)/is);
});

test("the current migration manifest satisfies the executable contract", async () => {
  const diagnostics = await validateProject(new URL("..", import.meta.url).pathname.replace(/^\/(?:[A-Za-z]:)/, (match) => match.slice(1)));
  assert.deepEqual(diagnostics, [], diagnostics.join("\n"));
});
