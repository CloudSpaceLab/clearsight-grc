import { afterEach, expect, it, vi } from "vitest";
import { installFreshEvidence } from "./v210/evidence";

const originalFetch = globalThis.fetch;
afterEach(() => { globalThis.fetch = originalFetch; vi.unstubAllEnvs(); window.history.replaceState(null, "", "/"); });

it("requires both evidence flags before replacing the network transport", () => {
  vi.stubEnv("VITE_STATIC_DEMO", "true");
  vi.stubEnv("VITE_UI_EVIDENCE", "false");
  expect(() => installFreshEvidence()).toThrow("Evidence transport is unavailable");
  expect(globalThis.fetch).toBe(originalFetch);
});

it("labels the Home sample and rejects unsupported scope or reporting periods", async () => {
  vi.stubEnv("VITE_STATIC_DEMO", "true");
  vi.stubEnv("VITE_UI_EVIDENCE", "true");
  installFreshEvidence();
  const context = await (await fetch("/api/v1/context")).json();
  expect(context.mode).toBe("sample");
  expect(context.tenant.name).toContain("Sample");
  expect((await fetch("/api/v1/oversight")).status).toBe(200);
  expect((await fetch("/api/v1/oversight?organization_scope_id=other")).status).toBe(403);
  expect((await fetch("/api/v1/metrics/home?start_date=2025-01-01&end_date=2026-11-20")).status).toBe(403);
  const metrics = await (await fetch("/api/v1/metrics/home?start_date=2026-10-21&end_date=2026-11-20")).json();
  expect(metrics.reporting_period.start_date).toBe("2026-10-21");
  expect(metrics.scope_id).toBe("sample-entity");
});

it("keeps the administrator role sample explicit", async () => {
  vi.stubEnv("VITE_STATIC_DEMO", "true");
  vi.stubEnv("VITE_UI_EVIDENCE", "true");
  window.history.replaceState(null, "", "/?fixture=admin-dashboard");
  installFreshEvidence();
  const context = await (await fetch("/api/v1/context")).json();
  expect(context.actor.role_codes).toEqual(["SYSTEM_ADMIN"]);
  expect(context.mode).toBe("sample");
});
