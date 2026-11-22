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

it("keeps the work count snapshot consistent with its complete sample page", async () => {
  vi.stubEnv("VITE_STATIC_DEMO", "true");
  vi.stubEnv("VITE_UI_EVIDENCE", "true");
  installFreshEvidence();
  const counts = await (await fetch("/api/v1/work/assignment-counts")).json();
  const page = await (await fetch("/api/v1/work/assignments")).json();
  expect(page.has_more).toBe(false);
  expect(counts.as_of).toBe(page.as_of);
  expect(counts.total).toBe(page.items.length);
  const asOf = Date.parse(counts.as_of);
  const deadlines = page.items.map((item: { due_at?: string }) => item.due_at ? Date.parse(item.due_at) : undefined);
  expect(counts.overdue).toBe(deadlines.filter((date: number | undefined) => date !== undefined && date < asOf).length);
  expect(counts.due_soon).toBe(deadlines.filter((date: number | undefined) => date !== undefined && date >= asOf && date <= asOf + 7 * 86_400_000).length);
  expect(counts.undated).toBe(deadlines.filter((date: number | undefined) => date === undefined).length);
});
