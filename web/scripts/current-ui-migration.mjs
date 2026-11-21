import { readFile, readdir } from "node:fs/promises";

// This is a coverage contract for the identified migrated families only.
// It is not a declaration that the full runtime design system is migrated.
const families = {
  panel: { token: ["--v-panel-background", "--v-color-surface"], consumer: ["fresh.css", /\.v-metric,\s*\.v-panel\s*\{[^}]*background:\s*var\(--v-panel-background\)/s] },
  "primary-action": { token: ["--v-button-primary-background", "--v-color-action"], consumer: ["fresh.css", /\.v-primary\s*\{[^}]*background:\s*var\(--v-button-primary-background\)/s] },
  "secondary-action": { token: ["--v-button-secondary-background", "--v-color-surface"], consumer: ["fresh.css", /\.v-program-search button,[^{]*\{[^}]*background:\s*var\(--v-button-secondary-background\)/s] },
  field: { token: ["--v-field-border", "--v-color-panel-border"], consumer: ["fresh.css", /\.v-program-search input\s*\{[^}]*border:\s*1px solid var\(--v-field-border\)/s] },
  status: { token: ["--v-status-background", "--v-color-status-background"], consumer: ["fresh.css", /\.v-status\s*\{[^}]*background:\s*var\(--v-status-background\)/s] },
  sheet: { token: ["--v-sheet-background", "--v-color-surface"], consumer: ["focused-sheet.css", /\.v-focused-sheet\s*\{[^}]*background:\s*var\(--v-sheet-background\)/s] },
};
const omissions = ["full-cascade", "dark-theme", "other-controls", "other-status-variants", "feature-palettes"];

export async function validateCurrentCascade(root, transform = (_file, source) => source) {
  const diagnostics = [];
  const directory = new URL("src/v210/", root);
  const files = (await readdir(directory)).filter((file) => file.endsWith(".css"));
  const styles = new Map(await Promise.all(files.map(async (file) => [file, transform(file, await readFile(new URL(file, directory), "utf8"))])));
  const tokens = styles.get("tokens.css") ?? "";
  if (!/@layer\s+v210-tokens\.primitive,\s*v210-tokens\.semantic,\s*v210-tokens\.component\s*;/.test(tokens)) diagnostics.push("token cascade order must be primitive, semantic, component");
  for (const layer of ["primitive", "semantic", "component"]) {
    if (!(new RegExp(`@layer\\s+v210-tokens\\.${layer}\\s*\\{`).test(tokens))) diagnostics.push(`missing ${layer} token layer`);
  }
  if (!/@import\s+["']\.\/tokens\.css["']/.test(styles.get("fresh.css") ?? "")) diagnostics.push("the active shell must load the token cascade");
  for (const [file, raw] of styles) {
    if (file === "tokens.css") continue;
    const css = raw.replace(/\/\*[\s\S]*?\*\//g, "");
    for (const match of css.matchAll(/(--v-[\w-]+)\s*:/g)) diagnostics.push(`${file}: ${match[1]} must be owned by the token cascade`);
  }
  return diagnostics;
}

export async function validateCurrentMigration(root, manifestOverride, transform = (_file, source) => source) {
  const read = async (name) => transform(name, await readFile(new URL(name, root), "utf8"));
  const manifest = manifestOverride ?? JSON.parse(await read("ui-contract-v210.json"));
  const diagnostics = [];
  const coverage = Array.isArray(manifest.families) ? manifest.families : [];
  const required = Object.keys(families);
  if (coverage.length !== required.length || new Set(coverage).size !== coverage.length || required.some((name) => !coverage.includes(name))) diagnostics.push("migration coverage must list each supported family exactly once");
  if (!manifestOverride && (manifest.status !== "partial" || omissions.some((name) => !manifest.remaining?.includes(name)))) diagnostics.push("migration coverage must disclose the outstanding design-system work");
  const tokens = await read("src/v210/tokens.css");
  for (const [semantic, primitive] of [
    ["--v-color-surface", "--v-primitive-surface"],
    ["--v-color-action", "--v-primitive-action"],
    ["--v-color-panel-border", "--v-primitive-panel-border"],
    ["--v-color-status-background", "--v-primitive-status-background"],
  ]) {
    if (!tokens.includes(`${semantic}: var(${primitive})`) || !tokens.includes(`${primitive}:`)) diagnostics.push(`${semantic} must resolve to a defined primitive ${primitive}`);
  }
  for (const name of required) {
    const { token: [component, semantic], consumer: [file, pattern] } = families[name];
    if (!tokens.includes(`${component}: var(${semantic})`)) diagnostics.push(`${component} must resolve to ${semantic}`);
    const css = await read(`src/v210/${file}`);
    if (!pattern.test(css)) diagnostics.push(`${name} consumer must use ${component}`);
  }
  return diagnostics;
}
