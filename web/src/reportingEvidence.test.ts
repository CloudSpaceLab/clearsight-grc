import { afterEach, describe, expect, it, vi } from "vitest";
import { installReportingEvidence } from "./reportingEvidence";

type EvidenceWindow = Window & { reportingEvidenceReads?: string[] };

async function json(response: Response) {
  return await response.json() as any;
}

afterEach(() => {
  vi.unstubAllGlobals();
  window.history.replaceState(null, "", "/");
});

describe("reporting evidence transport", () => {
  it("serves the real reporting reads and records each handled URL", async () => {
    const previousFetch = vi.fn().mockResolvedValue(new Response(JSON.stringify({ delegated: true }), { status: 200, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("fetch", previousFetch);
    window.history.replaceState(null, "", "/?fixture=report-definitions");
    installReportingEvidence();

    const fields = await json(await fetch("/api/v1/reports/filter-fields"));
    expect(fields.fields.some((field: { field: string }) => field.field === "name")).toBe(true);
    const definitions = await json(await fetch("/api/v1/reports/definitions?tenant_id=bank-demo"));
    expect(definitions.items.map((item: { status: string }) => item.status)).toEqual(expect.arrayContaining(["ACTIVE", "PENDING_REVIEW", "REVIEWED"]));

    const activeID = definitions.items.find((item: { status: string }) => item.status === "ACTIVE").id;
    const detail = await json(await fetch(`/api/v1/reports/definitions/${activeID}?tenant_id=bank-demo`));
    expect(detail.name).toBe("Processing activities with open exceptions");
    expect(detail.description).toMatch(/sample data/i);
    expect(detail.description).toMatch(/4 open exceptions across 8 seeded activities/);
    expect(detail.description).toMatch(/Cloudspace OEM/);
    expect(detail.description).toMatch(/Azure/);
    const crossBorder = definitions.items.find((item: { code: string }) => item.code === "ROPA-CROSS-BORDER-TRANSFERS");
    expect(crossBorder.description).toMatch(/domestic Nigerian/);
    const history = await json(await fetch(`/api/v1/reports/definitions/${activeID}/history?tenant_id=bank-demo`));
    expect(history.items[0]).toMatchObject({ decision: "APPROVED", maker_id: expect.any(String), reviewed_by: expect.any(String), approved_by: expect.any(String) });

    const runs = await json(await fetch("/api/v1/reports/runs?tenant_id=bank-demo&limit=50"));
    expect(runs.items.some((run: { status: string; row_count: number; source_boundary: { population: number; population_complete: boolean } }) => run.status === "READY" && run.row_count === 4 && run.source_boundary.population === 8 && run.source_boundary.population_complete)).toBe(true);
    expect(runs.items.some((run: { status: string; failure_code: string; row_count: number }) => run.status === "FAILED" && run.failure_code === "row_limit_exceeded" && run.row_count === 0)).toBe(true);

    const runID = runs.items.find((run: { status: string }) => run.status === "FAILED").id;
    const run = await json(await fetch(`/api/v1/reports/runs/${runID}?tenant_id=bank-demo`));
    expect(run.failure_code).toBe("row_limit_exceeded");

    expect((window as EvidenceWindow).reportingEvidenceReads).toEqual([
      "/api/v1/reports/filter-fields",
      "/api/v1/reports/definitions?tenant_id=bank-demo",
      `/api/v1/reports/definitions/${activeID}?tenant_id=bank-demo`,
      `/api/v1/reports/definitions/${activeID}/history?tenant_id=bank-demo`,
      "/api/v1/reports/runs?tenant_id=bank-demo&limit=50",
      `/api/v1/reports/runs/${runID}?tenant_id=bank-demo`,
    ]);
    expect(previousFetch).not.toHaveBeenCalled();

    const delegated = await fetch(new URL("/api/v1/context", window.location.origin));
    expect(await delegated.json()).toEqual({ delegated: true });
    expect(previousFetch).toHaveBeenCalledWith(expect.any(URL), undefined);
  });

  it("normalizes Request inputs and returns the bounded-stop fixture for the named review state", async () => {
    const previousFetch = vi.fn();
    vi.stubGlobal("fetch", previousFetch);
    window.history.replaceState(null, "", "/?fixture=report-run-failed");
    installReportingEvidence();

    const response = await fetch(new Request("https://example.test/api/v1/reports/runs?tenant_id=bank-demo&definition_id=report-active"));
    const body = await json(response);
    expect(body.items).toHaveLength(1);
    expect(body.items[0]).toMatchObject({ status: "FAILED", failure_code: "row_limit_exceeded", row_count: 0 });
    expect((window as EvidenceWindow).reportingEvidenceReads).toEqual(["/api/v1/reports/runs?tenant_id=bank-demo&definition_id=report-active"]);
  });

  it("downloads the seeded exception report with the open Cloudspace and Azure findings", async () => {
    vi.stubGlobal("fetch", vi.fn());
    window.history.replaceState(null, "", "/?fixture=report-definitions");
    installReportingEvidence();

    const response = await fetch("/api/v1/reports/runs/report-ready/download?tenant_id=bank-demo");
    const body = await response.text();
    expect(body).toContain("Cloudspace OEM — POS Support/PTSP");
    expect(body).toContain("Azure user access management");
    expect(body).toContain("Completed review");
  });

  it("returns a not-found response for an unknown definition or run", async () => {
    vi.stubGlobal("fetch", vi.fn());
    window.history.replaceState(null, "", "/");
    installReportingEvidence();

    const definition = await fetch("/api/v1/reports/definitions/missing?tenant_id=bank-demo");
    expect(definition.status).toBe(404);
    expect((await definition.json()).error.code).toBe("report_not_found");
    const run = await fetch("/api/v1/reports/runs/missing?tenant_id=bank-demo");
    expect(run.status).toBe(404);
    expect((await run.json()).error.code).toBe("report_not_found");
  });
});
