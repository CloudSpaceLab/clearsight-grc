import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import path from "node:path";
import test from "node:test";

const sourceRoot = path.resolve("src");
const runtimeEntry = path.join(sourceRoot, "main.tsx");
const forbiddenRuntimeMarkers = [
  "staticDemoBootstrap",
  "LifecycleTodayEvidencePage",
  "OperatingMutationsEvidencePage",
  "OversightEvidencePage",
  "ui-gallery",
  "VITE_STATIC_DEMO",
  "VITE_UI_EVIDENCE",
];

test("the customer runtime import graph excludes fixture and evidence modules", async () => {
  const visited = new Set();
  const violations = [];
  await visit(runtimeEntry, visited, violations);
  assert.deepEqual(violations, [], violations.join("\n"));
});

async function visit(filename, visited, violations) {
  const normalized = path.normalize(filename);
  if (visited.has(normalized)) return;
  visited.add(normalized);

  const source = await readFile(normalized, "utf8");
  for (const marker of forbiddenRuntimeMarkers) {
    if (source.includes(marker)) {
      violations.push(`${path.relative(process.cwd(), normalized)} contains ${marker}`);
    }
  }

  const imports = source.matchAll(/(?:import|export)\s+(?:[^"']+?\s+from\s+)?["'](\.[^"']+)["']|import\(["'](\.[^"']+)["']\)/g);
  for (const match of imports) {
    const specifier = match[1] ?? match[2];
    const resolved = await resolveModule(path.dirname(normalized), specifier);
    if (resolved && resolved.startsWith(sourceRoot + path.sep)) {
      await visit(resolved, visited, violations);
    }
  }
}

async function resolveModule(parent, specifier) {
  const base = path.resolve(parent, specifier);
  for (const candidate of [base, `${base}.ts`, `${base}.tsx`, `${base}.js`, `${base}.jsx`, path.join(base, "index.ts"), path.join(base, "index.tsx")]) {
    try {
      await readFile(candidate);
      return candidate;
    } catch {
      // Try the next TypeScript or JavaScript resolution candidate.
    }
  }
  return null;
}
