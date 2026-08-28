import { describe, expect, it } from "vitest";
import { assessInteractionBundle, initialAssetPaths } from "./ui-bundle-budget.mjs";

describe("interaction bundle budget", () => {
  it("counts entry assets and module preloads without charging deferred chunks", () => {
    const assets = initialAssetPaths(`<!doctype html><html><head>
      <link rel="stylesheet" href="/assets/index.css">
      <link rel="modulepreload" href="/assets/vendor.js">
    </head><body>
      <script type="module" src="/assets/index.js"></script>
      <!-- deferred chunks are absent from the entry document by design -->
    </body></html>`);
    expect([...assets.javascript]).toEqual(["assets/vendor.js", "assets/index.js"]);
    expect([...assets.css]).toEqual(["assets/index.css"]);
  });

  it("keeps the startup budget strict while allowing bounded lazy code", () => {
    const metrics = {
      javascript: { raw: 900_000, gzip: 280_000, initial_gzip: 180_000, largest_raw_chunk: 480_000 },
      css: { raw: 150_000, gzip: 36_000, initial_gzip: 31_000 },
    };
    expect(assessInteractionBundle(metrics)).toEqual([]);

    expect(assessInteractionBundle({
      ...metrics,
      javascript: { ...metrics.javascript, initial_gzip: 200_000 },
    })).toEqual(["Initial JavaScript interaction bundle exceeds 192 KiB gzip (200000 bytes)"]);

    expect(assessInteractionBundle({
      ...metrics,
      javascript: { ...metrics.javascript, largest_raw_chunk: 700_000 },
    })).toEqual(["A JavaScript chunk exceeds 600 KiB raw (700000 bytes)"]);
  });
});
