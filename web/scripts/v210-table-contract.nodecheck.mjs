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
    let wrapped = [...line.matchAll(/<div\s+className=["'](?:[^"']*\s)?v-program-table(?:\s[^"']*)?["'][^>]*>[^\n]*?<table(?:\s|>)/g)].length;
    // Indicator history has one named table and a separately tested stacked layout.
    if (filename === "IndicatorDetail.tsx" && /<div className="v-indicator-history-wrap"><table aria-label=/.test(line)) wrapped++;
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


test("indicator history has labelled cells and a stacked responsive contract", async () => {
  const component = await readFile(new URL("../src/v210/IndicatorDetail.tsx", import.meta.url), "utf8");
  const css = await readFile(new URL("../src/v210/indicator-detail.css", import.meta.url), "utf8");
  assert.equal(inspectTables(component, "IndicatorDetail.tsx").missing.length, 0);
  assert.equal((component.match(/<table(?:\s|>)/g) ?? []).length, 1);
  for (const label of ["Evaluated", "Value", "Limit", "Result", "Concern"]) assert.match(component, new RegExp(`<td data-label="${label}">`));
  assert.ok(component.includes('<td><button type="button" onClick='));
  assert.match(component, /View evaluation for/);
  assert.match(css, /\.v-indicator-history-wrap table,[^{]*tbody,[^{]*tr,[^{]*td\s*\{[^}]*display:\s*block;[^}]*width:\s*100%/s);
  assert.match(css, /\.v-indicator-history-wrap thead\s*\{[^}]*clip-path:\s*inset\(50%\)/s);
  assert.match(css, /\.v-indicator-history-wrap td\s*\{[^}]*overflow-wrap:\s*anywhere/s);
  assert.match(css, /\.v-indicator-history-wrap td::before\s*\{[^}]*content:\s*attr\(data-label\)/s);
  assert.deepEqual(inspectTables('<div className="v-indicator-history-wrap"><table aria-label="x"/></div>', 'Other.tsx').missing, ['Other.tsx:1']);
});
