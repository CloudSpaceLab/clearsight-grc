import { afterEach, describe, expect, it, vi } from "vitest";
import { installRopaEvidence } from "./ropaEvidence";

type EvidenceWindow = Window & { ropaEvidenceReads?: string[] };

const activityID = "ropa-activity-customer-account-opening";

async function json(response: Response) {
  return await response.json() as any;
}

afterEach(() => {
  vi.unstubAllGlobals();
  window.history.replaceState(null, "", "/");
});

describe("ROPA evidence transport", () => {
  it("serves the register, activity and history reads and records each handled URL", async () => {
    const previousFetch = vi.fn().mockResolvedValue(new Response(JSON.stringify({ ok: true }), { status: 200, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("fetch", previousFetch);
    window.history.replaceState(null, "", "/?fixture=ropa-stale");
    installRopaEvidence();

    const dashboard = await json(await fetch("/api/v1/ropa/dashboard?tenant_id=bank-demo"));
    expect(dashboard).toMatchObject({
      freshness: "STALE",
      generated_at: "2026-09-24T07:15:00Z",
      coverage: { population: expect.any(Number), excluded: expect.any(Number), unknown: expect.any(Number) },
      counts: { total: expect.any(Number), open: expect.any(Number), review_overdue: expect.any(Number) },
    });

    const list = await json(await fetch("/api/v1/ropa/processing-activities?tenant_id=bank-demo"));
    expect(list.Rows.some((activity: { name: string }) => activity.name === "Customer account opening")).toBe(true);

    const detail = await json(await fetch(`/api/v1/ropa/processing-activities/${activityID}?tenant_id=bank-demo`));
    expect(detail.activity.name).toBe("Customer account opening");
    expect(detail.closure_blockers).toEqual(["lawful basis", "named owner", "data subject category", "completed review"]);

    const history = await json(await fetch(`/api/v1/ropa/processing-activities/${activityID}/history?tenant_id=bank-demo&limit=100`));
    expect(history.events).toHaveLength(3);
    expect(history.has_more).toBe(false);

    expect((window as EvidenceWindow).ropaEvidenceReads).toEqual([
      "/api/v1/ropa/dashboard?tenant_id=bank-demo",
      "/api/v1/ropa/processing-activities?tenant_id=bank-demo",
      `/api/v1/ropa/processing-activities/${activityID}?tenant_id=bank-demo`,
      `/api/v1/ropa/processing-activities/${activityID}/history?tenant_id=bank-demo&limit=100`,
    ]);
    expect(previousFetch).not.toHaveBeenCalled();
  });

  it("omits unknown coverage values rather than presenting them as zero", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("{}", { status: 200 })));
    window.history.replaceState(null, "", "/?fixture=ropa-unknown");
    installRopaEvidence();

    const summary = await json(await fetch("/api/v1/ropa/dashboard?tenant_id=bank-demo"));
    expect(summary.freshness).toBe("CURRENT");
    expect(summary.coverage).toEqual({ population: expect.any(Number) });
    expect(summary.coverage).not.toHaveProperty("excluded");
    expect(summary.coverage).not.toHaveProperty("unknown");
  });

  it("passes unrelated API requests to the preceding transport", async () => {
    const previousFetch = vi.fn().mockResolvedValue(new Response(JSON.stringify({ tenant: { id: "bank-demo" } }), { status: 200, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("fetch", previousFetch);
    installRopaEvidence();

    const response = await fetch("/api/v1/context");
    expect(await response.json()).toEqual({ tenant: { id: "bank-demo" } });
    expect(previousFetch).toHaveBeenCalledWith("/api/v1/context", undefined);
  });
});
