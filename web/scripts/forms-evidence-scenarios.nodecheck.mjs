import assert from "node:assert/strict";
import test from "node:test";

import { formsEvidenceScenarios, requiredFormsCapabilities } from "./forms-evidence-scenarios.mjs";

const task22Capabilities = [
  "library-empty", "library-list", "library-search", "library-saved-filter", "library-context-detail", "library-bulk-action",
  "creation-blank", "creation-template", "creation-ai", "creation-import",
  "template-draft", "template-pending", "template-active", "template-retired", "weights-invalid", "weights-valid",
  "import-pending", "import-partial", "import-truncated", "import-failed", "import-proposal",
  "communication-compose", "communication-delivered", "communication-fallback", "communication-amended",
  "communication-rotated", "communication-superseded", "communication-revoked",
  "access-direct-link", "access-shared-otp", "access-direct-otp", "otp-expired", "otp-exhausted",
  "recovery-server-saved", "recovery-device-only", "recovery-conflict", "recovery-recovered", "recovery-file-reselection",
  "response-first", "response-amended",
  "vendor-confirm", "vendor-correct", "vendor-replace", "vendor-review", "vendor-conflict", "vendor-applied",
  "library-mobile-records", "builder-mobile-actions", "builder-pointer-reorder", "builder-large-performance",
  "viewport-desktop", "viewport-mobile", "viewport-reflow-320", "zoom-200", "theme-light", "theme-dark",
];

test("Forms scenarios cover every Task 22 capability", () => {
  assert.deepEqual(requiredFormsCapabilities, task22Capabilities);
  const covered = new Set(formsEvidenceScenarios.flatMap((scenario) => scenario.capabilities));
  assert.deepEqual(requiredFormsCapabilities.filter((capability) => !covered.has(capability)), []);
  assert.equal(new Set(formsEvidenceScenarios.map(({ name }) => name)).size, formsEvidenceScenarios.length);
  assert.equal(new Set(formsEvidenceScenarios.map(({ fixture }) => fixture)).size, formsEvidenceScenarios.length);
  const allowedRoutes = new Set(["#forms", "#imports", "#vendors", "/capture"]);
  for (const scenario of formsEvidenceScenarios) {
    assert.match(scenario.name, /^\d{2,3}-forms-/);
    assert.ok(allowedRoutes.has(scenario.route), scenario.route);
    assert.ok(scenario.fixture);
    assert.ok(scenario.state);
    assert.ok(["light", "dark"].includes(scenario.theme));
    assert.ok(scenario.viewport.width >= 320);
    assert.ok(scenario.viewport.height >= 640);
    assert.ok([1, 2].includes(scenario.zoom));
    assert.equal(typeof scenario.run, "function");
    assert.ok(scenario.capabilities.length > 0);
    for (const capability of scenario.capabilities) assert.ok(requiredFormsCapabilities.includes(capability), capability);
  }
});
