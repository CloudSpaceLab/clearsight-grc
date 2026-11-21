import { afterEach, expect, it, vi } from "vitest";
import { installHomeShellEvidence } from "./homeShellEvidence";

const originalFetch = globalThis.fetch;
afterEach(() => { globalThis.fetch = originalFetch; });

it("limits the full-shell sample to its named fixture, scope and period", async () => {
  globalThis.fetch = vi.fn(async () => Response.json({ tenant: { name: "Bank" }, capabilities: {} }));
  installHomeShellEvidence("home-shell");
  const context = await (await fetch("/api/v1/context")).json();
  expect(context.tenant.name).toContain("Sample");
  expect(context.capabilities.oversight_read).toBe(true);
  expect((await fetch("/api/v1/oversight")).status).toBe(200);
  expect((await fetch("/api/v1/oversight?organization_scope_id=other")).status).toBe(403);
  expect((await fetch("/api/v1/metrics/home?start_date=2025-01-01")).status).toBe(403);
});

it("leaves other fixtures untouched and simulates an unavailable Home when requested", async () => {
  const previous = vi.fn(async () => Response.json({ original: true }));
  globalThis.fetch = previous;
  installHomeShellEvidence("other");
  expect(globalThis.fetch).toBe(previous);
  installHomeShellEvidence("home-shell-unavailable");
  expect((await fetch("/api/v1/oversight")).status).toBe(503);
});

it("isolates the populated Home insight sample to its named fixture", async () => {
  globalThis.fetch = vi.fn(async () => Response.json({ original: true }));
  installHomeShellEvidence("home-shell-insights");
  const data = await (await fetch("/api/v1/metrics/domain")).json();
  expect(data.items.find((item: { id: string }) => item.id === "risks_outside_appetite")).toMatchObject({ value: 4, population: 8, unknown: 0 });
  expect((await fetch("/api/v1/metrics/domain?organization_scope_id=other")).status).toBe(403);
});
