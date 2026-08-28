import { readdir, readFile } from "node:fs/promises";
import path from "node:path";
import { gzipSync } from "node:zlib";

export const interactionBundleBudget = Object.freeze({
  javascript: Object.freeze({ initialGzip: 192 * 1024, largestRawChunk: 600 * 1024 }),
  css: Object.freeze({ initialGzip: 32 * 1024 }),
});

export async function collectInteractionBundleMetrics(distDirectory) {
  const dist = path.resolve(distDirectory);
  const indexHTML = await readFile(path.join(dist, "index.html"), "utf8");
  const initialAssets = initialAssetPaths(indexHTML);
  const builtFiles = await walkFiles(dist);
  const metrics = {
    javascript: { raw: 0, gzip: 0, initial_gzip: 0, largest_raw_chunk: 0 },
    css: { raw: 0, gzip: 0, initial_gzip: 0 },
  };

  for (const filePath of builtFiles) {
    if (!filePath.endsWith(".js") && !filePath.endsWith(".css")) continue;
    const bytes = await readFile(filePath);
    const gzipBytes = gzipSync(bytes).length;
    const relative = normalizeAssetPath(path.relative(dist, filePath));
    if (filePath.endsWith(".js")) {
      metrics.javascript.raw += bytes.length;
      metrics.javascript.gzip += gzipBytes;
      metrics.javascript.largest_raw_chunk = Math.max(metrics.javascript.largest_raw_chunk, bytes.length);
      if (initialAssets.javascript.has(relative)) metrics.javascript.initial_gzip += gzipBytes;
    } else {
      metrics.css.raw += bytes.length;
      metrics.css.gzip += gzipBytes;
      if (initialAssets.css.has(relative)) metrics.css.initial_gzip += gzipBytes;
    }
  }
  return metrics;
}

export function assessInteractionBundle(metrics, budget = interactionBundleBudget) {
  const failures = [];
  if (metrics.javascript.largest_raw_chunk > budget.javascript.largestRawChunk) {
    failures.push(`A JavaScript chunk exceeds 600 KiB raw (${metrics.javascript.largest_raw_chunk} bytes)`);
  }
  if (metrics.javascript.initial_gzip > budget.javascript.initialGzip) {
    failures.push(`Initial JavaScript interaction bundle exceeds 192 KiB gzip (${metrics.javascript.initial_gzip} bytes)`);
  }
  if (metrics.css.initial_gzip > budget.css.initialGzip) {
    failures.push(`Initial CSS interaction bundle exceeds 32 KiB gzip (${metrics.css.initial_gzip} bytes)`);
  }
  return failures;
}

export function initialAssetPaths(indexHTML) {
  const javascript = new Set();
  const css = new Set();
  for (const match of indexHTML.matchAll(/<(script|link)\b[^>]*(?:src|href)=["']([^"']+)["'][^>]*>/gi)) {
    const [tag, url] = [match[1]?.toLowerCase(), match[2]];
    if (!url) continue;
    const asset = normalizeAssetURL(url);
    if (!asset) continue;
    if (tag === "script" && asset.endsWith(".js")) javascript.add(asset);
    if (tag === "link") {
      const markup = match[0].toLowerCase();
      if (markup.includes('rel="modulepreload"') || markup.includes("rel='modulepreload'")) {
        if (asset.endsWith(".js")) javascript.add(asset);
      } else if (markup.includes('rel="stylesheet"') || markup.includes("rel='stylesheet'")) {
        if (asset.endsWith(".css")) css.add(asset);
      }
    }
  }
  return { javascript, css };
}

function normalizeAssetURL(value) {
  if (/^(?:https?:)?\/\//i.test(value) || value.startsWith("data:")) return "";
  const clean = value.split(/[?#]/, 1)[0] ?? "";
  return normalizeAssetPath(clean.replace(/^\.?\//, ""));
}

function normalizeAssetPath(value) {
  return value.split(path.sep).join("/").replace(/^\/+/, "");
}

async function walkFiles(root) {
  const pending = [root];
  const files = [];
  while (pending.length) {
    const directory = pending.pop();
    if (!directory) continue;
    for (const entry of await readdir(directory, { withFileTypes: true })) {
      const entryPath = path.join(directory, entry.name);
      if (entry.isDirectory()) pending.push(entryPath);
      else files.push(entryPath);
    }
  }
  return files;
}
