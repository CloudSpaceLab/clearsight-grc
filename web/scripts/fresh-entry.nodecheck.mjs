import { readFile, readdir, stat } from "node:fs/promises";
import assert from "node:assert/strict";
import test from "node:test";
import path from "node:path";
import { fileURLToPath } from "node:url";

test("CI runtime boundary includes the v2.10 entry dependency check", async () => {
  const pkg = JSON.parse(await readFile(new URL("../package.json", import.meta.url), "utf8"));
  assert.match(pkg.scripts["check:runtime-truth"], /\bscripts\/fresh-entry\.nodecheck\.mjs\b/);
});

test("application and evidence entries exclude the old UI", async () => {
  const main = await readFile(new URL("../src/main.tsx", import.meta.url), "utf8");
  const evidence = await readFile(new URL("../src/evidenceMain.tsx", import.meta.url), "utf8");
  assert.ok(main.includes("./v210/FreshApp"));
  assert.ok(main.includes("./v210/fresh.css"));
  assert.ok(main.includes("./v210/Capture"));
  assert.match(main, /<Capture\s+invitationToken=\{invitation\}/);
  for (const old of ["./App", "./appStyles", "./components/SessionGate", "./components/ExternalCaptureApp"]) {
    assert.ok(!main.includes(old));
    assert.ok(!evidence.includes(old));
  }
  assert.ok(evidence.includes('import("./main")'));
});

test("retired shell entry and global stylesheet have been removed", async () => {
  for (const path of ["../src/appStyles.ts", "../src/App.tsx", "../src/styles.css"]) {
    await assert.rejects(stat(new URL(path, import.meta.url)), { code: "ENOENT" });
  }
});

test("runtime dependency graph excludes legacy UI and root stylesheets", async () => {
  const sourceRoot = fileURLToPath(new URL("../src/", import.meta.url));
  const pending = [path.join(sourceRoot, "main.tsx"), path.join(sourceRoot, "evidenceMain.tsx")];
  const visited = new Set();
  while (pending.length) {
    const file = pending.pop();
    if (visited.has(file)) continue;
    visited.add(file);
    const relative = path.relative(sourceRoot, file).replaceAll("\\", "/");
    assert.ok(!relative.startsWith("components/") && !relative.startsWith("design-system/"), `legacy UI imported: ${relative}`);
    assert.ok(!(/^[^/]+\.css$/).test(relative), `legacy stylesheet imported: ${relative}`);
    if (!/\.[cm]?[jt]sx?$/.test(file)) continue;
    const code = await readFile(file, "utf8");
    const imports = Array.from(code.matchAll(/(?:\b(?:import|export)\s+(?!type\b)(?:[^;]*?\s+from\s+)?|\bimport\s*\()\s*["']([^"']+)["']/g), (match) => match[1]);
    for (const specifier of imports.filter((item) => item.startsWith("."))) {
      const base = path.resolve(path.dirname(file), specifier);
      const candidates = [base, ...[".ts", ".tsx", ".js", ".jsx", ".css"].map((ext) => base + ext), ...["index.ts", "index.tsx"].map((name) => path.join(base, name))];
      let found;
      for (const candidate of candidates) { try { if ((await stat(candidate)).isFile()) { found = candidate; break; } } catch { /* try the next extension */ } }
      assert.ok(found, `unresolved import ${specifier} from ${relative}`);
      if (found.startsWith(sourceRoot)) pending.push(found);
    }
  }
  assert.deepEqual((await readdir(sourceRoot)).filter((name) => name.endsWith(".css")), []);
  for (const retired of ["components", "design-system"]) {
    const files = await readdir(path.join(sourceRoot, retired), { recursive: true });
    assert.deepEqual(files.filter((name) => name.endsWith(".css")), [], `legacy styles remain under ${retired}`);
  }
});
