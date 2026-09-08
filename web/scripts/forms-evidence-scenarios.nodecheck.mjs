import assert from "node:assert/strict";
import test from "node:test";
import { runInNewContext } from "node:vm";
import { setImmediate } from "node:timers/promises";

import { formsEvidenceScenarios, requiredFormsCapabilities, installDemoDocumentScenario, waitForNativePDFPage, assertSheetRecoveryVisible } from "./forms-evidence-scenarios.mjs";

const task22Capabilities = [
  "library-empty", "library-list", "library-search", "library-saved-filter", "library-context-detail", "library-bulk-action",
  "library-filter-picker", "library-advanced-filter", "library-status-scopes",
  "creation-blank", "creation-template", "creation-ai", "creation-import",
  "template-draft", "template-pending", "template-active", "template-retired", "weights-invalid", "weights-valid",
  "import-pending", "import-partial", "import-truncated", "import-failed", "import-proposal",
  "communication-compose", "communication-delivered", "communication-fallback", "communication-amended",
  "communication-rotated", "communication-superseded", "communication-revoked",
  "access-direct-link", "access-shared-otp", "access-direct-otp", "otp-expired", "otp-exhausted",
  "recovery-server-saved", "recovery-device-only", "recovery-conflict", "recovery-recovered", "recovery-file-reselection",
  "response-first", "response-amended",
  "vendor-confirm", "vendor-correct", "vendor-replace", "vendor-review", "vendor-conflict", "vendor-applied",
  "library-mobile-records", "builder-mobile-actions", "builder-pointer-reorder", "builder-large-performance", "builder-themed-select",
  "viewport-desktop", "viewport-mobile", "viewport-reflow-320", "zoom-200", "theme-light", "theme-dark",
  "foundation-component-variants", "select-themed-open", "focus-visible", "density-comfortable", "density-compact",
  "sent-empty-replacement", "sent-populated-table", "sent-responsive-sheet", "sent-partial-page", "sent-lifecycle-feedback",
  "forced-colors", "reduced-motion",
  "documents-file-types", "documents-quick-look", "documents-keyboard-return", "documents-vendor-launcher",
  "forms-section-resumption",
];

test("Forms scenarios cover every Task 22 capability", () => {
  assert.deepEqual(requiredFormsCapabilities, task22Capabilities);
  const covered = new Set(formsEvidenceScenarios.flatMap((scenario) => scenario.capabilities));
  assert.deepEqual(requiredFormsCapabilities.filter((capability) => !covered.has(capability)), []);
  assert.equal(new Set(formsEvidenceScenarios.map(({ name }) => name)).size, formsEvidenceScenarios.length);
  const allowedRoutes = new Set(["#forms", "#imports", "#vendors", "#ui-components", "/capture"]);
  for (const scenario of formsEvidenceScenarios) {
    assert.match(scenario.name, /^\d{2,3}[a-z]?-forms-/);
    assert.ok(allowedRoutes.has(scenario.route), scenario.route);
    assert.ok(scenario.fixture);
    assert.ok(scenario.state);
    assert.ok(["light", "dark"].includes(scenario.theme));
    assert.ok(scenario.viewport.width >= 320);
    assert.ok(scenario.viewport.height >= 640);
    assert.ok([1, 2].includes(scenario.zoom));
    assert.ok(["comfortable", "compact"].includes(scenario.density ?? "comfortable"));
    assert.ok(["none", "active"].includes(scenario.forcedColors ?? "none"));
    assert.ok(["no-preference", "reduce"].includes(scenario.reducedMotion ?? "no-preference"));
    assert.equal(typeof scenario.run, "function");
    assert.ok(scenario.capabilities.length > 0);
    for (const capability of scenario.capabilities) assert.ok(requiredFormsCapabilities.includes(capability), capability);
  }
});

test("demo document evidence covers preview and blocked files in both workspaces and themes", () => {
  const demo = formsEvidenceScenarios.filter((scenario) => scenario.state.startsWith("demo-document-"));
  assert.equal(demo.length, 26);
  for (const surface of ["forms", "vendors"]) for (const theme of ["light", "dark"]) for (const width of [1440, 390, 320]) {
    for (const state of ["demo-document-preview", "demo-document-blocked"]) {
      assert.ok(demo.some((scenario) => scenario.route === `#${surface}` && scenario.theme === theme && scenario.viewport.width === width && scenario.state === state), `${surface} ${theme} ${width} ${state}`);
    }
  }
  for (const theme of ["light", "dark"]) assert.ok(demo.some((scenario) => scenario.theme === theme && scenario.viewport.width === 1440 && scenario.state === "demo-document-pdf-preview"), `PDF ${theme}`);
});

for (const kind of ["image", "pdf"]) test(`${kind} browser fixture keeps pending filename, media and bytes consistent`, async () => {
  const window = { fetch: async () => Response.json({ items: [{ id: "authorized-occurrence" }] }) };
  const page = { evaluate: (callback, args) => runInNewContext(`(${callback})(args)`, { window, args, URL, Request, Response, Uint8Array, atob, location: { href: "http://localhost/" } }) };
  const expected = await installDemoDocumentScenario(page, kind);
  const { items: [sample, pending] } = await (await window.fetch("/api/v1/forms/documents")).json();
  assert.equal(pending.file_name, kind === "pdf" ? "Supplier insurance schedule.pdf" : "Supplier office statement.png");
  assert.equal(pending.media_type, kind === "pdf" ? "application/pdf" : "image/png");
  assert.equal(pending.file_kind, kind === "pdf" ? "PDF" : "IMAGE");
  assert.equal(pending.demo_preview_available, false);
  assert.equal(sample.demo_preview_available, true);
  assert.equal(sample.size_bytes, expected.size);
  await assert.rejects(window.fetch("/api/v1/forms/documents/genuine-pending-artifact/content"), /Blocked file requested content/);
});

for (const navigating of [false, true]) test(`native PDF readiness waits for the loaded page (${navigating ? "new" : "existing"} viewer frame)`, async () => {
  const loading = Promise.withResolvers();
  const viewer = {
    url: () => "chrome-extension://mhjfbmdgcfjbbpaeojofohoefgiehjai/index.html",
    locator: () => ({ waitFor: async () => {}, textContent: async () => "1" }),
    getByRole: (role) => role === "progressbar" ? { waitFor: () => loading.promise } : { inputValue: async () => "1" },
  };
  const page = {
    waitForFunction: async () => {}, // iframe readyState is already complete while the viewer is loading.
    frames: () => navigating ? [] : [viewer],
    waitForEvent: async (event, options) => {
      assert.equal(event, "framenavigated");
      assert.equal(options.predicate({ url: () => "about:blank" }), false);
      assert.equal(options.predicate(viewer), true);
      assert.equal(options.timeout, 10000);
      return viewer;
    },
  };
  let settled = false;
  const ready = waitForNativePDFPage(page).then((result) => { settled = true; return result; });
  await setImmediate();
  assert.equal(settled, false, "iframe readiness must not allow a capture before PDF loading finishes");
  loading.resolve();
  assert.deepEqual(await ready, { page_number: 1, page_count: 1 });
});

test("native PDF readiness rejects an unfinished viewer and a wrong page count", async () => {
  const viewer = {
    url: () => "chrome-extension://mhjfbmdgcfjbbpaeojofohoefgiehjai/index.html",
    locator: () => ({ waitFor: async () => {}, textContent: async () => "0" }),
    getByRole: (role) => role === "progressbar" ? { waitFor: async () => {} } : { inputValue: async () => "1" },
  };
  const page = { frames: () => [viewer] };
  await assert.rejects(waitForNativePDFPage(page), /Expected the one-page insurance PDF, found page 1 of 0/);
  viewer.getByRole = () => ({ waitFor: async () => { throw new Error("PDF loading timed out"); } });
  await assert.rejects(waitForNativePDFPage(page), /PDF loading timed out/);
});

test("actual-file scenarios use authored expiry dates and submissions after document issue", () => {
  for (const scenario of formsEvidenceScenarios.filter((value) => value.state.startsWith("demo-document-"))) {
    const pdf = scenario.state === "demo-document-pdf-preview";
    const metadata = scenario.documentMetadata;
    assert.equal(metadata?.issued_on, pdf ? "2026-04-01" : "2026-09-02", `${scenario.name} authored issue date`);
    assert.equal(metadata?.expires_on, pdf ? "2026-09-30" : "2027-09-02", `${scenario.name} authored expiry`);
    assert.equal(metadata.uploaded_at, "2026-09-08T09:00:00Z", `${scenario.name} sample upload`);
    assert.equal(metadata.submitted_at, "2026-09-08T09:15:00Z", `${scenario.name} sample submission`);
    assert.ok(Date.parse(metadata.uploaded_at) >= Date.parse(metadata.issued_on), "upload must follow the authored issue date");
    assert.ok(Date.parse(metadata.submitted_at) >= Date.parse(metadata.issued_on), "submission must follow the authored issue date");
    assert.ok(Date.parse(metadata.submitted_at) >= Date.parse(metadata.uploaded_at), "submission must follow upload");
  }
});

function sheetGeometryFixture({ hidden = false, missing = false, offscreen = false, frameInterval = 1000, neverFrame = false } = {}) {
  const oldSheet = { x: 33, y: 33, width: 324, height: 1144 };
  const currentSheet = { x: 1, y: 1, width: 388, height: 951 };
  const currentClose = { x: offscreen ? 380 : 325, y: 21, width: 44, height: 44 };
  let elapsed = 0;
  const timers = new Map();
  const frames = new Set();
  let sequence = 0;
  const element = (rect) => ({ getBoundingClientRect: () => rect, getClientRects: () => hidden ? [] : [rect] });
  const close = missing ? null : element(currentClose);
  const dialog = {
    boundingBox: async () => oldSheet,
    getByRole: () => ({ count: async () => missing ? 0 : 1, boundingBox: async () => missing ? null : currentClose, elementHandle: async () => close }),
    evaluate: (callback, args) => runInNewContext(`(${callback})(sheet,args)`, {
      sheet: element(currentSheet), args, innerWidth: 390, innerHeight: 844,
      performance: { now: () => elapsed },
      requestAnimationFrame: (callback) => {
        const id = ++sequence; frames.add(id);
        if (!neverFrame) queueMicrotask(() => { if (frames.delete(id)) { elapsed += frameInterval; callback(); } });
        return id;
      },
      cancelAnimationFrame: (id) => frames.delete(id),
      setTimeout: (callback, delay) => {
        const id = ++sequence; timers.set(id, callback);
        if (neverFrame) queueMicrotask(() => { if (timers.has(id)) { elapsed += delay; callback(); } });
        return id;
      },
      clearTimeout: (id) => timers.delete(id),
      getComputedStyle: () => ({ visibility: hidden ? "hidden" : "visible" }),
    }),
  };
  return { page: { viewportSize: () => ({ width: 390, height: 844 }) }, dialog, frames: () => elapsed / frameInterval, pending: () => timers.size + frames.size };
}

test("sheet recovery compares one layout snapshot across a responsive replacement", async () => {
  // Real Chromium resize measurements: old sheet y=33 and new Close y=21 falsely fail;
  // the simultaneous mobile sheet y=1 and Close y=21 are both inside the viewport.
  const { page, dialog, frames, pending } = sheetGeometryFixture();
  await assertSheetRecoveryVisible(page, dialog);
  assert.equal(frames(), 2, "acceptance requires matching geometry in consecutive rendered frames");
  assert.equal(pending(), 0, "completion must cancel the deadline and any pending frame");
});

for (const timing of [{ neverFrame: true }, { frameInterval: 6000 }]) test(`sheet recovery deadline rejects ${timing.neverFrame ? "missing frames" : "a second valid frame at twelve seconds"}`, async () => {
  const { page, dialog, pending } = sheetGeometryFixture(timing);
  const outcome = assertSheetRecoveryVisible(page, dialog);
  await assert.rejects(Promise.race([outcome, setImmediate().then(() => { throw new Error("The independent geometry deadline did not settle"); })]), /visible viewport/);
  assert.equal(pending(), 0, "deadline failure must clean up timers and pending frames");
});

for (const failure of ["hidden", "missing", "offscreen"]) test(`sheet recovery rejects a permanently ${failure} close control with geometry`, async () => {
  const { page, dialog } = sheetGeometryFixture({ [failure]: true });
  await assert.rejects(assertSheetRecoveryVisible(page, dialog), (error) => {
    assert.match(error.message, /visible viewport/);
    assert.match(error.message, /requested/);
    assert.match(error.message, /actual/);
    return true;
  });
});
