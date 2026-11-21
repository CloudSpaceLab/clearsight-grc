import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const read = (name) => readFile(new URL(`../src/v210/${name}`, import.meta.url), "utf8");

test("the v2.10 shell exposes a primitive, semantic and component token chain", async () => {
  const [tokens, shell] = await Promise.all([read("tokens.css"), read("fresh.css")]);
  assert.match(tokens, /--v-primitive-canvas\s*:\s*#[\da-f]+/i);
  assert.match(tokens, /--v-color-canvas\s*:\s*var\(--v-primitive-canvas\)/);
  assert.match(tokens, /--v-shell-background\s*:\s*var\(--v-color-canvas\)/);
  assert.match(tokens, /--v-focus-color\s*:\s*var\(--v-color-focus\)/);
  assert.match(shell, /@import\s+["']\.\/tokens\.css["']/);
  assert.match(shell, /background:\s*var\(--v-shell-background\)/);
  assert.match(shell, /outline:\s*3px\s+solid\s+var\(--v-focus-color\)/);
});

test("mobile program cells allow long unbroken identifiers to wrap", async () => {
  const css = await read("fresh.css");
  assert.match(css, /\.v-program-table td\s*\{[^}]*overflow-wrap:\s*anywhere[^}]*\}/s);
  assert.match(css, /@media\s*\(max-width:\s*700px\)[\s\S]*\.v-program-table tr\s*\{[^}]*minmax\(0,\s*1fr\)/);
});

test("the keyboard focus ring applies to buttons, links and form controls", async () => {
  const css = await read("fresh.css");
  for (const control of ["button", "a", "input", "select", "textarea"]) assert.match(css, new RegExp(`${control}:focus-visible`));
  assert.match(css, /:focus-visible\s*\{[^}]*outline-offset:\s*2px/);
});

test("CI retains browser evidence for the current navigation close control", async () => {
  const [workflow, script] = await Promise.all([
    readFile(new URL("../../.github/workflows/ui-evidence.yml", import.meta.url), "utf8"),
    readFile(new URL("./v215-shell-navigation-evidence.mjs", import.meta.url), "utf8"),
  ]);
  assert.match(workflow, /node scripts\/v215-shell-navigation-evidence\.mjs/);
  assert.match(script, /elementFromPoint/);
  assert.match(script, /closeAboveContent/);
});

test("the primary action uses component tokens mapped through semantic and primitive roles", async () => {
  const [tokens, shell] = await Promise.all([read("tokens.css"), read("fresh.css")]);
  assert.match(tokens, /--v-primitive-action:\s*#246cf0/i);
  assert.match(tokens, /--v-color-action:\s*var\(--v-primitive-action\)/);
  assert.match(tokens, /--v-button-primary-background:\s*var\(--v-color-action\)/);
  assert.match(tokens, /--v-button-primary-text:\s*var\(--v-color-action-text\)/);
  assert.match(tokens, /--v-button-height:\s*var\(--v-control-height\)/);
  assert.match(shell, /\.v-primary\s*\{[^}]*background:\s*var\(--v-button-primary-background\)/s);
  assert.match(shell, /\.v-primary\s*\{[^}]*color:\s*var\(--v-button-primary-text\)/s);
  assert.match(shell, /\.v-login select,\s*\.v-login button,\s*\.v-primary\s*\{[^}]*min-height:\s*var\(--v-button-height\)/s);
});

test("the current token chain uses ordered primitive, semantic and component cascade layers", async () => {
  const tokens = await read("tokens.css");
  assert.match(tokens, /@layer\s+v210-tokens\.primitive,\s*v210-tokens\.semantic,\s*v210-tokens\.component\s*;/);
  for (const layer of ["primitive", "semantic", "component"]) assert.match(tokens, new RegExp(`@layer\\s+v210-tokens\\.${layer}\\s*\\{`));
  assert.match(tokens, /@layer\s+v210-tokens\.primitive\s*\{[\s\S]*?--v-primitive-action:\s*#246cf0/i);
  assert.match(tokens, /@layer\s+v210-tokens\.semantic\s*\{[\s\S]*?--v-color-action:\s*var\(--v-primitive-action\)/);
  assert.match(tokens, /@layer\s+v210-tokens\.component\s*\{[\s\S]*?--v-button-primary-background:\s*var\(--v-color-action\)/);
});

test("the active panel consumes component tokens mapped to semantic and primitive roles", async () => {
  const [tokens, shell] = await Promise.all([read("tokens.css"), read("fresh.css")]);
  assert.match(tokens, /--v-primitive-panel-border:\s*#dce7f5/i);
  assert.match(tokens, /--v-color-panel-border:\s*var\(--v-primitive-panel-border\)/);
  assert.match(tokens, /--v-panel-border:\s*var\(--v-color-panel-border\)/);
  assert.match(tokens, /--v-panel-background:\s*var\(--v-color-surface\)/);
  assert.match(tokens, /--v-panel-radius:\s*var\(--v-radius-panel\)/);
  assert.match(shell, /\.v-metric,\s*\.v-panel\s*\{[^}]*border:\s*1px solid var\(--v-panel-border\)[^}]*border-radius:\s*var\(--v-panel-radius\)[^}]*background:\s*var\(--v-panel-background\)/s);
});

test("search and vendor intake fields consume shared field tokens", async () => {
  const [tokens, shell] = await Promise.all([read("tokens.css"), read("fresh.css")]);
  assert.match(tokens, /--v-field-border:\s*var\(--v-color-panel-border\)/);
  assert.match(tokens, /--v-field-background:\s*var\(--v-color-surface\)/);
  assert.match(tokens, /--v-field-text:\s*var\(--v-color-text\)/);
  assert.match(tokens, /--v-field-height:\s*var\(--v-control-height\)/);
  for (const selector of ["\\.v-program-search input", "\\.v-intake input,\\s*\\.v-intake select"]) {
    const block = shell.match(new RegExp(`${selector}\\s*\\{([^}]*)\\}`, "s"))?.[1] ?? "";
    assert.match(block, /min-height:\s*var\(--v-field-height\)/);
    assert.match(block, /border:\s*1px solid var\(--v-field-border\)/);
    assert.match(block, /background:\s*var\(--v-field-background\)/);
  }
});

test("CI verifies the active search and intake field tokens in a browser", async () => {
  const [workflow, script] = await Promise.all([
    readFile(new URL("../../.github/workflows/ui-evidence.yml", import.meta.url), "utf8"),
    readFile(new URL("./v215-field-tokens-evidence.mjs", import.meta.url), "utf8"),
  ]);
  assert.match(workflow, /node scripts\/v215-field-tokens-evidence\.mjs/);
  assert.match(script, /getComputedStyle/);
  assert.match(script, /--v-field-background/);
  assert.match(script, /scrollWidth\s*<=\s*innerWidth/);
});

test("shared status badges use semantic status tokens", async () => {
  const [tokens, shell] = await Promise.all([read("tokens.css"), read("fresh.css")]);
  for (const [component, semantic, primitive] of [
    ["--v-status-text", "--v-color-status-text", "--v-primitive-status-text"],
    ["--v-status-background", "--v-color-status-background", "--v-primitive-status-background"],
    ["--v-status-high-text", "--v-color-danger-text", "--v-primitive-danger-text"],
    ["--v-status-high-background", "--v-color-danger-background", "--v-primitive-danger-background"],
  ]) {
    assert.ok(tokens.includes(`${component}: var(${semantic})`), component);
    assert.ok(tokens.includes(`${semantic}: var(${primitive})`), semantic);
  }
  assert.match(shell, /\.v-status\s*\{[^}]*color:\s*var\(--v-status-text\)[^}]*background:\s*var\(--v-status-background\)/s);
  assert.match(shell, /\.v-status--high\s*\{[^}]*color:\s*var\(--v-status-high-text\)[^}]*background:\s*var\(--v-status-high-background\)/s);
});

test("secondary actions use the shared color and geometry token chain", async () => {
  const [tokens, shell] = await Promise.all([read("tokens.css"), read("fresh.css")]);
  for (const [component, semantic] of [
    ["--v-button-secondary-border", "--v-color-panel-border"],
    ["--v-button-secondary-background", "--v-color-surface"],
    ["--v-button-secondary-text", "--v-color-text"],
    ["--v-button-radius", "--v-radius-control"],
  ]) assert.ok(tokens.includes(`${component}: var(${semantic})`), component);
  const block = shell.match(/\.v-program-search button,\s*\.v-program-table > button,\s*\.v-panel > button\s*\{([^}]*)\}/s)?.[1] ?? "";
  for (const token of ["--v-button-height", "--v-button-secondary-border", "--v-button-radius", "--v-button-secondary-background", "--v-button-secondary-text"]) assert.ok(block.includes(`var(${token})`), token);
});
