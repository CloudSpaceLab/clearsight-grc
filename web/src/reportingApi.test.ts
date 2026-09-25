import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { loadContext } from "./api";
import {
  createReportRun,
  createReportDefinition,
  downloadReportRun,
  getReportDefinition,
  getReportDefinitionHistory,
  getReportRun,
  listReportDefinitions,
  listReportFilterFields,
  listReportRunPage,
  listReportRuns,
  transitionReportDefinition,
} from "./reportingApi";

vi.mock("./api", () => ({ loadContext: vi.fn() }));

const fetchMock = vi.fn();
const context = {
  tenant: { id: "tenant-1", name: "Clear Bank" },
  legal_entity: { id: "entity-1", name: "Clear Bank Nigeria" },
  actor: { id: "privacy-officer", name: "Privacy officer" },
  mode: "test",
};

const fields = [{ field: "status", label: "Processing activity status", dataset: "PROCESSING_ACTIVITIES", operators: ["is"], indexed: true }];
const definition = {
  id: "definition-1",
  tenant_id: "tenant-1",
  legal_entity_id: "entity-1",
  code: "ROPA-EXCEPTIONS",
  name: "Open processing exceptions",
  description: "Reports activities with missing closure facts.",
  dataset: "PROCESSING_ACTIVITIES",
  scope_kind: "LEGAL_ENTITY",
  format: "CSV",
  filter: { kind: "group", operator: "and", children: [] },
  status: "DRAFT",
  current_version: 1,
  effective: false,
  checksum: "a".repeat(64),
  maker_id: "maker-1",
  version: 1,
  created_at: "2026-09-24T08:00:00Z",
  updated_at: "2026-09-24T08:00:00Z",
};
const run = {
  id: "run-1",
  tenant_id: "tenant-1",
  legal_entity_id: "entity-1",
  definition_id: "definition-1",
  definition_version: 1,
  definition_code: "ROPA-EXCEPTIONS",
  definition_checksum: "a".repeat(64),
  scope_kind: "LEGAL_ENTITY",
  requested_by_ref: "performer-1",
  as_of: "2026-09-24T08:00:00Z",
  filter: { kind: "group", operator: "and", children: [] },
  dataset: "PROCESSING_ACTIVITIES",
  format: "CSV",
  status: "READY",
  attempt_count: 1,
  row_count: 2,
  created_at: "2026-09-24T08:01:00Z",
  completed_at: "2026-09-24T08:02:00Z",
  expires_at: "2026-10-01T08:01:00Z",
  source_boundary: {
    captured_at: "2026-09-24T08:00:00Z",
    projection_version: "ropa-report-v1",
    source_high_water: { processing_activities: "2026-09-24T07:59:00Z" },
    population: 2,
    population_complete: true,
  },
};

beforeEach(() => {
  vi.mocked(loadContext).mockResolvedValue(context);
  fetchMock.mockReset();
  fetchMock.mockResolvedValue(new Response(JSON.stringify({}), { status: 200, headers: { "Content-Type": "application/json" } }));
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("reporting API", () => {
  it("reads the published filter vocabulary without inventing fields", async () => {
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ fields }), { status: 200 }));
    await expect(listReportFilterFields()).resolves.toEqual({ fields });
    expect(String(fetchMock.mock.calls[0]?.[0])).toBe("/api/v1/ropa/reports/filter-fields");
  });

  it("scopes definition, history, detail and run reads to the verified tenant", async () => {
    fetchMock
      .mockResolvedValueOnce(new Response(JSON.stringify({ items: [definition] }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ items: [] }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(definition), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ items: [run] }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(run), { status: 200 }));

    await expect(listReportDefinitions()).resolves.toEqual([definition]);
    await expect(getReportDefinitionHistory("definition/1")).resolves.toEqual([]);
    await expect(getReportDefinition("definition/1")).resolves.toEqual(definition);
    await expect(listReportRuns({ limit: 20, definitionId: "definition-1" })).resolves.toEqual([run]);
    await expect(getReportRun("run/1")).resolves.toEqual(run);

    expect(Object.fromEntries(new URL(String(fetchMock.mock.calls[0]?.[0]), "https://example.test").searchParams)).toEqual({ tenant_id: "tenant-1" });
    expect(String(fetchMock.mock.calls[1]?.[0])).toBe("/api/v1/ropa/reports/definitions/definition%2F1/history?tenant_id=tenant-1");
    expect(String(fetchMock.mock.calls[2]?.[0])).toBe("/api/v1/ropa/reports/definitions/definition%2F1?tenant_id=tenant-1");
    const runsURL = new URL(String(fetchMock.mock.calls[3]?.[0]), "https://example.test");
    expect(runsURL.pathname).toBe("/api/v1/ropa/reports/runs");
    expect(Object.fromEntries(runsURL.searchParams)).toEqual({ tenant_id: "tenant-1", limit: "20", definition_id: "definition-1" });
    expect(String(fetchMock.mock.calls[4]?.[0])).toBe("/api/v1/ropa/reports/runs/run%2F1?tenant_id=tenant-1");
  });

  it("forwards the opaque report history cursor without interpreting it", async () => {
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ items: [run], next_cursor: "next-page" }), { status: 200 }));

    await expect(listReportRunPage({ limit: 50, cursor: "current-page" })).resolves.toEqual({
      items: [run],
      next_cursor: "next-page",
    });

    const url = new URL(String(fetchMock.mock.calls[0]?.[0]), "https://example.test");
    expect(Object.fromEntries(url.searchParams)).toEqual({
      tenant_id: "tenant-1",
      limit: "50",
      cursor: "current-page",
    });
  });

  it("sends governed definition and run commands without trusting browser actor fields", async () => {
    fetchMock
      .mockResolvedValueOnce(new Response(JSON.stringify(definition), { status: 201 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(definition), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(run), { status: 201 }));

    await createReportDefinition({ code: "ROPA-EXCEPTIONS", name: "Open processing exceptions", description: "Report description", dataset: "PROCESSING_ACTIVITIES", scope_kind: "LEGAL_ENTITY", format: "CSV", filter: { kind: "group", operator: "and", children: [] } });
    await transitionReportDefinition("definition-1", "review", { expected_version: 1, checksum_seen: "a".repeat(64), note: "Checked the filter and scope." });
    await createReportRun("definition-1", 1);

    const firstBody = JSON.parse(String((fetchMock.mock.calls[0]?.[1] as RequestInit).body));
    expect(firstBody).toMatchObject({ code: "ROPA-EXCEPTIONS", dataset: "PROCESSING_ACTIVITIES" });
    expect(firstBody).not.toHaveProperty("maker_id");
    expect(firstBody).not.toHaveProperty("tenant_id");
    const transitionInit = fetchMock.mock.calls[1]?.[1] as RequestInit;
    expect(transitionInit.method).toBe("POST");
    expect(String(fetchMock.mock.calls[1]?.[0])).toBe("/api/v1/ropa/reports/definitions/definition-1/review");
    expect(JSON.parse(String(transitionInit.body))).toEqual({ expected_version: 1, checksum_seen: "a".repeat(64), note: "Checked the filter and scope." });
    expect(JSON.parse(String((fetchMock.mock.calls[2]?.[1] as RequestInit).body))).toEqual({ definition_id: "definition-1", expected_definition_version: 1 });
  });

  it("downloads a protected report as a file without exposing a stored object key", async () => {
    fetchMock.mockResolvedValueOnce(new Response("id,name\n1,Customer account opening\n", { status: 200, headers: { "Content-Disposition": "attachment; filename=\"ROPA-EXCEPTIONS.csv\"" } }));
    const result = await downloadReportRun("run-1");
    expect(result.filename).toBe("ROPA-EXCEPTIONS.csv");
    expect(await result.blob.text()).toContain("Customer account opening");
    expect(String(fetchMock.mock.calls[0]?.[0])).toBe("/api/v1/ropa/reports/runs/run-1/download?tenant_id=tenant-1");
  });

  it("turns report service failures into an operator recovery message", async () => {
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ error: { code: "report_request_invalid", message: "raw server detail" } }), { status: 400 }));
    await expect(listReportDefinitions()).rejects.toMatchObject({ message: "Report definitions and runs could not be loaded. Check the connection and try again.", kind: "validation" });
  });
});
