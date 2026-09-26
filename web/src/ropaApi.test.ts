import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { loadContext } from "./api";
import { fetchDashboard, fetchProcessingActivity, fetchProcessingActivityHistory, listProcessingActivities } from "./ropaApi";

vi.mock("./api", () => ({ loadContext: vi.fn() }));

const fetchMock = vi.fn();
const context = {
  tenant: { id: "tenant-1", name: "Clear Bank" },
  legal_entity: { id: "entity-1", name: "Clear Bank Nigeria" },
  actor: { id: "reviewer-1", name: "Privacy reviewer" },
  mode: "test",
};

const activity = {
  id: "activity-1",
  tenant_id: "tenant-1",
  legal_entity_id: "entity-1",
  code: "PA-001",
  name: "Customer onboarding",
  description: "Collect identity data during onboarding.",
  status: "OPEN",
  purpose: "Onboard customers",
  lawful_basis: "Contract",
  controller: "Clear Bank",
  processor: "Internal operations",
  automated_decision_making: false,
  data_subject_categories: "Customers",
  personal_data_categories: "Name; date of birth",
  security_measures: "Encryption at rest",
  retention_period: "7 years",
  version: 2,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-02-01T00:00:00Z",
};

const summary = {
  generated_at: "2026-02-01T00:00:00Z",
  projection_version: "ropa-v1",
  freshness: "CURRENT",
  source_high_water: "2026-02-01T00:00:00Z",
  coverage: { population: 10, excluded: null, unknown: null },
  counts: { total: 10, new: 1, open: 7, closed: 2, review_overdue: 1, missing_lawful_basis: 1, missing_owner: 0, no_data_subjects: 1, retired: 0 },
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

describe("ROPA API", () => {
  it("parses the scoped dashboard and list page", async () => {
    fetchMock
      .mockResolvedValueOnce(new Response(JSON.stringify(summary), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ Rows: [activity], NextCursor: "cursor/2", HasMore: true }), { status: 200 }));

    await expect(fetchDashboard()).resolves.toEqual(summary);
    await expect(listProcessingActivities({ status: "OPEN", search: "customer onboarding", cursor: "cursor/1", include_retired: true })).resolves.toEqual({ rows: [activity], next_cursor: "cursor/2", has_more: true });

    expect(String(fetchMock.mock.calls[0]?.[0])).toBe("/api/v1/ropa/dashboard?tenant_id=tenant-1");
    const listURL = new URL(String(fetchMock.mock.calls[1]?.[0]), "https://example.test");
    expect(listURL.pathname).toBe("/api/v1/ropa/processing-activities");
    expect(Object.fromEntries(listURL.searchParams)).toEqual({ tenant_id: "tenant-1", status: "OPEN", search: "customer onboarding", cursor: "cursor/1", include_retired: "true" });
  });

  it("parses an activity and its bounded history", async () => {
    const response = { state_label: "In progress", activity, closure_blockers: ["completed review"] };
    const history = { events: [], has_more: false };
    fetchMock
      .mockResolvedValueOnce(new Response(JSON.stringify(response), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(history), { status: 200 }));

    await expect(fetchProcessingActivity("activity/1")).resolves.toEqual(response);
    await expect(fetchProcessingActivityHistory("activity/1", { after_version: 3, limit: 25 })).resolves.toEqual(history);

    expect(String(fetchMock.mock.calls[0]?.[0])).toBe("/api/v1/ropa/processing-activities/activity%2F1?tenant_id=tenant-1");
    const historyURL = new URL(String(fetchMock.mock.calls[1]?.[0]), "https://example.test");
    expect(historyURL.pathname).toBe("/api/v1/ropa/processing-activities/activity%2F1/history");
    expect(Object.fromEntries(historyURL.searchParams)).toEqual({ tenant_id: "tenant-1", after_version: "3", limit: "25" });
  });

  it("turns a missing processing activity into an operator recovery message", async () => {
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ error: { code: "ropa_activity_not_found", message: "ropa_activity_not_found" } }), { status: 404 }));
    await expect(fetchProcessingActivity("activity-1")).rejects.toMatchObject({ message: "Couldn’t load processing activity.", kind: "not_found" });
  });

  it("turns a service failure into a register recovery message without exposing the response code", async () => {
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ error: { code: "ropa_unavailable", message: "ropa_unavailable" } }), { status: 500 }));
    const failure = await fetchDashboard().catch((error: unknown) => error);
    expect(failure).toMatchObject({ message: "Couldn’t load processing activities." });
    expect(failure).not.toHaveProperty("code", "ropa_unavailable");
  });

  it("forwards the caller's AbortSignal", async () => {
    const controller = new AbortController();
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify(summary), { status: 200 }));
    await fetchDashboard(controller.signal);

    const init = fetchMock.mock.calls[0]?.[1] as RequestInit | undefined;
    expect(init?.signal).toBe(controller.signal);
  });
});
