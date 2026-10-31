import { createHash } from "node:crypto";
import { readFile } from "node:fs/promises";
import path from "node:path";

const sha = (bytes) => createHash("sha256").update(bytes).digest("hex");
export async function reviewWorkSourceEvidence(outputDirectory, runtimeBuild) {
  const failures = [], directory = path.join(outputDirectory, "work-source");
  const receipt = await readFile(path.join(directory, "receipt.json"), "utf8").then(JSON.parse).catch(() => null);
  if (!receipt) return { failures: ["Work source-count evidence is missing or invalid"], detail: "No receipt" };
  if (receipt.baseline || receipt.failure || receipt.kind !== "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION") failures.push("Work source-count receipt is not a passing runtime review");
  if (runtimeBuild && (receipt.build?.digest !== runtimeBuild.digest || receipt.revision !== runtimeBuild.revision)) failures.push("Work source-count images belong to a different application build");
  for (const [key, filename] of [["fixtureSHA256", "src/staticDemoWorkSourceRuntime.js"], ["captureSHA256", "scripts/capture-work-source-evidence.mjs"]]) {
    if (receipt.inputs?.[key] !== sha(await readFile(filename))) failures.push(`Work source-count capture input changed: ${filename}`);
  }
  if (receipt.source?.sha256 !== "202ed9c2b22ac4e921d9ba14aa41b61644ea54cbe55abcc81a86ec10b10ee8d2") failures.push("Work source-count review lost the approved HTML identity");
  const layouts = receipt.layouts ?? [], sources = receipt.sources ?? [];
  if (layouts.length !== 10 || sources.length !== 5) failures.push("Work source-count viewport/theme matrix is incomplete");
  for (const width of [1440, 1280, 1024, 768, 390]) {
    if (sources.filter((item) => item.viewport?.width === width).length !== 1) failures.push(`Work source HTML render missing or duplicated at ${width}px`);
    for (const theme of ["light", "dark"]) {
      const matches = layouts.filter((item) => item.viewport?.width === width && item.theme === theme);
      const record = matches[0], geometry = record?.geometry;
      if (matches.length !== 1 || !geometry?.sourceOrder || !geometry.firstRecordInViewport
        || geometry.scrollWidth > geometry.clientWidth + 1 || record.errors?.length
        || !Array.isArray(record.accessibility) || record.accessibility.length
        || !geometry.controls?.length || geometry.controls.some((box) => box.width < 44 || box.height < 44)) {
        failures.push(`Work source-count layout/interaction evidence failed at ${width}px ${theme}`);
      }
      if (geometry?.measureColumns !== sources.find((item) => item.viewport?.width === width)?.geometry?.measureColumns) failures.push(`Work measure layout differs from approved source at ${width}px ${theme}`);
    }
  }
  const behaviors = receipt.behaviors ?? [];
  for (const name of ["deadline-drills-light-1440", "deadline-drills-dark-390"]) {
    const record = behaviors.find((item) => item.name === name), checks = record?.checks;
    if (!checks || !checks.cardClearsOtherFilters || checks.overdueFirstPage !== 50 || !checks.overdueComplete
      || !checks.upcomingInclusiveMicrosecondBounds || !checks.clearRestoresLoadedFeed || !checks.approvalsStayLoaded
      || record.commands?.length || record.errors?.length || record.reads?.filter((entry) => entry.query.cursor).length < 2) {
      failures.push(`Work source-count drill evidence failed: ${name}`);
    }
  }
  for (const name of ["counts-unavailable", "counts-scope", "page-scope"]) {
    const record = behaviors.find((item) => item.name === name);
    if (!record?.passed || record.observation?.commands?.length) failures.push(`Work source-count fallback evidence failed: ${name}`);
  }
  const images = receipt.images ?? [], imageNames = new Set(images.map((item) => item.file));
  if (imageNames.size !== images.length) failures.push("Work source-count review repeats image identities");
  for (const record of [...layouts, ...sources, ...behaviors]) {
    for (const key of ["image", "overdueImage", "upcomingImage"]) if (record[key] && !imageNames.has(record[key])) failures.push(`Work source-count image receipt is missing: ${record[key]}`);
  }
  for (const item of images) {
    if (path.basename(item.file) !== item.file) { failures.push("Invalid Work source-count image path"); continue; }
    const bytes = await readFile(path.join(directory, item.file)).catch(() => null);
    if (!bytes || item.bytes !== bytes.length || item.sha256 !== sha(bytes)) failures.push(`Work source-count image bytes differ: ${item.file}`);
  }
  return { failures, detail: `${layouts.length} layouts, ${behaviors.length} behavior scenarios, ${images.length} retained images` };
}
