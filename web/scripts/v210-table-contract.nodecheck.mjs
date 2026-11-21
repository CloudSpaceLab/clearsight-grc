import assert from "node:assert/strict";
import { readdir, readFile } from "node:fs/promises";
import test from "node:test";

// Register JSX keeps the wrapper and table on one source line. Reject any
// other form until it has an explicit equivalent responsive contract.
function inspectTables(source, filename) {
  const missing = []; let count = 0;
  for (const [index, line] of source.split(/\r?\n/).entries()) {
    const tables = [...line.matchAll(/<table(?:\s|>)/g)].length;
    if (!tables) continue;
    count += tables;
    const wrapped = [...line.matchAll(/<div\s+className=["'](?:[^"']*\s)?v-program-table(?:\s[^"']*)?["'][^>]*>[^\n]*?<table(?:\s|>)/g)].length;
    if (wrapped !== tables) missing.push(`${filename}:${index + 1}`);
  }
  return { count, missing };
}

test("every v2.10 data table uses the responsive wrapping contract", async () => {
  const dir = new URL("../src/v210/", import.meta.url);
  const files = (await readdir(dir)).filter((name) => name.endsWith(".tsx") && !name.endsWith(".test.tsx"));
  let count = 0; const missing = [];
  for (const name of files) { const result = inspectTables(await readFile(new URL(name, dir), "utf8"), name); count += result.count; missing.push(...result.missing); }
  assert.ok(count >= 7, `expected the known register tables, found ${count}`);
  assert.deepEqual(missing, []);
});

test("a newly introduced table without the responsive wrapper is rejected", () => {
  assert.deepEqual(inspectTables("export const Page = () => <div><table><tbody /></table></div>", "example.tsx").missing, ["example.tsx:1"]);
});
