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

test("feature CSS rejects raw visual values and component-internal reach-through", () => {
  const diagnostics = validateCssSource({
    file: "src/forms-sample.css",
    source: ".sample .cs-button { color: #fff; border-radius: 9px; transition: color 200ms; z-index: 99; }",
  }).join("\n");
  assert.match(diagnostics, /must not reach into \.cs-/);
  assert.match(diagnostics, /color must use a design token/);
  assert.match(diagnostics, /radius token/);
  assert.match(diagnostics, /duration token/);
  assert.match(diagnostics, /z-index token/);
});

test("the current migration manifest satisfies the executable contract", async () => {
  const diagnostics = await validateProject(new URL("..", import.meta.url).pathname.replace(/^\/(?:[A-Za-z]:)/, (match) => match.slice(1)));
  assert.deepEqual(diagnostics, [], diagnostics.join("\n"));
});
