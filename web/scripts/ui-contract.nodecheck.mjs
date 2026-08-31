import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

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
