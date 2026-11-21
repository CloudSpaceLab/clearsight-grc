import assert from "node:assert/strict";
import test from "node:test";
import { validateCurrentCascade, validateCurrentMigration } from "./current-ui-migration.mjs";

test("the partial runtime cascade preserves token order without feature overrides", async () => {
  const diagnostics = await validateCurrentCascade(new URL("..", import.meta.url));
  assert.deepEqual(diagnostics, [], diagnostics.join("\n"));
});

test("the cascade rejects reversed token layers and unlayered token collisions", async () => {
  const reversed = await validateCurrentCascade(new URL("..", import.meta.url), (file, source) => file.endsWith("tokens.css") ? source.replace("v210-tokens.primitive, v210-tokens.semantic", "v210-tokens.semantic, v210-tokens.primitive") : source);
  assert.ok(reversed.some((message) => message.includes("order")));
  const collision = await validateCurrentCascade(new URL("..", import.meta.url), (file, source) => file.endsWith("home.css") ? `${source}\n:root { --v-panel-background: red; }` : source);
  assert.ok(collision.some((message) => message.includes("home.css") && message.includes("--v-panel-background")));
});

test("the current migration manifest retains its required coverage", async () => {
  const diagnostics = await validateCurrentMigration(new URL("..", import.meta.url));
  assert.deepEqual(diagnostics, [], diagnostics.join("\n"));
});

test("the migration validator rejects missing and duplicate coverage", async () => {
  for (const families of [[], ["panel", "panel"]]) {
    const diagnostics = await validateCurrentMigration(new URL("..", import.meta.url), { families });
    assert.ok(diagnostics.some((message) => message.includes("coverage")));
  }
});

test("the migration validator rejects a broken semantic chain and consumer", async () => {
  const diagnostics = await validateCurrentMigration(new URL("..", import.meta.url), undefined, (file, source) =>
    file.endsWith("tokens.css") ? source.replace("--v-panel-background: var(--v-color-surface)", "--v-panel-background: #fff") :
      file.endsWith("fresh.css") ? source.replace("background: var(--v-panel-background)", "background: white") : source);
  assert.ok(diagnostics.some((message) => message.includes("--v-panel-background")));
  assert.ok(diagnostics.some((message) => message.includes("panel consumer")));
});

test("the migration validator rejects a missing primitive link", async () => {
  const diagnostics = await validateCurrentMigration(new URL("..", import.meta.url), undefined, (file, source) =>
    file.endsWith("tokens.css") ? source.replace("--v-color-surface: var(--v-primitive-surface)", "--v-color-surface: white") : source);
  assert.ok(diagnostics.some((message) => message.includes("--v-color-surface")));
});
