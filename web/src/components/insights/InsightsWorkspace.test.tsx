import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { loadIndicatorPopulation } from "../../indicatorApi";
import type { IndicatorPopulationItem, IndicatorPopulationPage } from "../../indicatorTypes";
import type { RiskIndicatorKind } from "../../riskTypes";
import { InsightsWorkspace } from "./InsightsWorkspace";

vi.mock("../../indicatorApi", () => ({ loadIndicatorPopulation: vi.fn() }));
vi.mock("../../monitoringApi", () => ({ loadMonitoringResults: vi.fn().mockResolvedValue([]) }));

const actions = { onKindChange: vi.fn(), onTarget: vi.fn() };

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(loadIndicatorPopulation).mockReset();
  vi.mocked(loadIndicatorPopulation).mockResolvedValue({ items: [itemFor("check-1", "Mobile transaction success rate")], complete: true });
});

it("keeps native values, approved limits, concern and freshness distinct in the source-shaped register", async () => {
  render(<InsightsWorkspace {...actions} embedded kind="KRI" legalEntityName="Nigeria" organizationScopeID="scope-a" organizationScopeName="Payments"/>);
  const table = await screen.findByRole("table", { name: "Risk indicators" });
  await within(table).findByText("Mobile transaction success rate");
  expect(table.closest(".cs-data-table")?.getAttribute("data-responsive-to")).toBe("container");
  expect(within(table).getByText("98.70%")).toBeTruthy();
  expect(within(table).getByText("Limit ≥ 99.50%")).toBeTruthy();
  expect(within(table).getByText("100 / 100 concern")).toBeTruthy();
  expect(within(table).getByText("Breach")).toBeTruthy();
  expect(within(table).getByText(/^Updated /)).toBeTruthy();
  expect(screen.getByText("1 indicator shown")).toBeTruthy();
  expect(screen.queryByRole("searchbox")).toBeNull();

  fireEvent.click(within(table).getByRole("button", { name: "Review indicator for Mobile transaction success rate" }));
  expect(actions.onTarget).toHaveBeenCalledWith("check-1", "KRI");
  fireEvent.click(screen.getByRole("button", { name: /Indicator type/ }));
  fireEvent.click(await screen.findByRole("option", { name: "Control indicators" }));
  expect(actions.onKindChange).toHaveBeenCalledWith("KCI");
});

it("preserves server cursor queries and resets pagination for the next indicator type", async () => {
  vi.mocked(loadIndicatorPopulation).mockImplementation(async (params) => ({
    items: [itemFor(params?.cursor ? "check-2" : "check-1", params?.cursor ? "Recovery time" : "Mobile success rate", params?.kind)],
    complete: true,
    next_cursor: params?.cursor ? undefined : "indicator-page-2",
  }));
  const { rerender } = render(<InsightsWorkspace {...actions} kind="KRI" organizationScopeID="scope-a"/>);
  fireEvent.click(await screen.findByRole("button", { name: "Next page" }));
  await waitFor(() => expect(loadIndicatorPopulation).toHaveBeenCalledWith({ kind: "KRI", organizationScopeID: "scope-a", cursor: "indicator-page-2", limit: 50 }, expect.any(AbortSignal)));
  await screen.findByText("Recovery time");

  rerender(<InsightsWorkspace {...actions} kind="KCI" organizationScopeID="scope-a"/>);
  await waitFor(() => expect(loadIndicatorPopulation).toHaveBeenLastCalledWith({ kind: "KCI", organizationScopeID: "scope-a", cursor: undefined, limit: 50 }, expect.any(AbortSignal)));
  expect(vi.mocked(loadIndicatorPopulation).mock.calls.some(([params]) => params?.kind === "KCI" && params.cursor)).toBe(false);
  expect(screen.queryByRole("button", { name: "Previous page" })).toBeNull();
});

it.each(["scope", "kind"] as const)("does not reuse an old page item after the selected %s changes", async (change) => {
  const nextRead = deferred<IndicatorPopulationPage>();
  vi.mocked(loadIndicatorPopulation).mockImplementation((params) => {
    if (params?.checkID) return nextRead.promise;
    const original = params?.kind === "KRI" && params.organizationScopeID === "scope-a";
    return Promise.resolve({ items: original ? [itemFor("check-1", "Previous indicator")] : [], complete: true });
  });
  const { rerender } = render(<InsightsWorkspace {...actions} kind="KRI" organizationScopeID="scope-a" targetID="check-1"/>);
  await screen.findByRole("dialog", { name: "Previous indicator indicator" });

  const nextKind = change === "kind" ? "KCI" : "KRI";
  const nextScope = change === "scope" ? "scope-b" : "scope-a";
  rerender(<InsightsWorkspace {...actions} kind={nextKind} organizationScopeID={nextScope} targetID="check-1"/>);
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(screen.queryByText("Previous indicator")).toBeNull();
  await waitFor(() => expect(loadIndicatorPopulation).toHaveBeenCalledWith({ kind: nextKind, organizationScopeID: nextScope, checkID: "check-1", limit: 1 }, expect.any(AbortSignal)));
  await act(async () => { nextRead.resolve({ items: [itemFor("check-1", "Current scope indicator", nextKind)], complete: true }); });
  expect(await screen.findByRole("dialog", { name: "Current scope indicator indicator" })).toBeTruthy();
});

it("clears the previous selection while the next indicator loads and ignores a late response", async () => {
  const secondRead = deferred<IndicatorPopulationPage>();
  vi.mocked(loadIndicatorPopulation).mockImplementation((params) => {
    if (!params?.checkID) return Promise.resolve({ items: [], complete: true });
    if (params.checkID === "check-2") return secondRead.promise;
    return Promise.resolve({ items: [itemFor(params.checkID, `Indicator ${params.checkID}`)], complete: true });
  });
  const { rerender } = render(<InsightsWorkspace {...actions} kind="KRI" targetID="check-1"/>);
  await screen.findByRole("dialog", { name: "Indicator check-1 indicator" });

  rerender(<InsightsWorkspace {...actions} kind="KRI" targetID="check-2"/>);
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(screen.getByText("Loading selected indicator…")).toBeTruthy();
  rerender(<InsightsWorkspace {...actions} kind="KRI" targetID="check-3"/>);
  await screen.findByRole("dialog", { name: "Indicator check-3 indicator" });
  await act(async () => { secondRead.resolve({ items: [itemFor("check-2", "Delayed indicator")], complete: true }); });
  expect(screen.getByRole("dialog", { name: "Indicator check-3 indicator" })).toBeTruthy();
  expect(screen.queryByText("Delayed indicator")).toBeNull();

  rerender(<InsightsWorkspace {...actions} kind="KRI"/>);
  expect(screen.queryByRole("dialog")).toBeNull();
});

it("does not reopen an indicator after its pending selection is removed", async () => {
  const pending = deferred<IndicatorPopulationPage>();
  vi.mocked(loadIndicatorPopulation).mockImplementation((params) => params?.checkID ? pending.promise : Promise.resolve({ items: [], complete: true }));
  const { rerender } = render(<InsightsWorkspace {...actions} kind="KRI" targetID="check-1"/>);
  await waitFor(() => expect(loadIndicatorPopulation).toHaveBeenCalledWith(expect.objectContaining({ checkID: "check-1" }), expect.any(AbortSignal)));
  rerender(<InsightsWorkspace {...actions} kind="KRI"/>);
  await act(async () => { pending.resolve({ items: [itemFor("check-1", "Closed indicator")], complete: true }); });
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(screen.queryByText("Closed indicator")).toBeNull();
});

it.each(["wrong ID", "incomplete empty result"])("treats an exact reader's %s as unavailable", async (result) => {
  vi.mocked(loadIndicatorPopulation).mockImplementation(async (params) => ({ items: params?.checkID && result === "wrong ID" ? [itemFor("other-check", "Unrelated indicator")] : [], complete: result !== "incomplete empty result" }));
  render(<InsightsWorkspace {...actions} kind="KRI" targetID="check-1"/>);
  expect(await screen.findByText("The selected indicator could not be loaded.")).toBeTruthy();
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(screen.queryByText("Unrelated indicator")).toBeNull();
});

it("shows a partial read and keeps an unknown condition separate from an available native value", async () => {
  const stale = itemFor("check-1", "Stale measurement");
  stale.indicator.state = "UNKNOWN";
  stale.indicator.reason = "The latest observation is outside the freshness window.";
  vi.mocked(loadIndicatorPopulation).mockResolvedValueOnce({ items: [stale], complete: false });
  render(<InsightsWorkspace {...actions} kind="KRI"/>);
  expect(await screen.findByText("Some indicator detail is unavailable in this scope. Available values remain unchanged.")).toBeTruthy();
  expect(screen.getByText("Unknown")).toBeTruthy();
  expect(screen.getByText("98.70%")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Retry" }));
  expect(await screen.findByText("Mobile transaction success rate")).toBeTruthy();
});

it("does not claim an empty population when the indicator read is incomplete", async () => {
  vi.mocked(loadIndicatorPopulation).mockResolvedValueOnce({ items: [], complete: false });
  render(<InsightsWorkspace {...actions} kind="KRI"/>);
  expect(await screen.findByText("Indicator count unavailable")).toBeTruthy();
  expect(screen.queryByText("No indicators in this scope")).toBeNull();
  expect(screen.queryByText("0 indicators shown")).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Retry" }));
  expect(await screen.findByText("Mobile transaction success rate")).toBeTruthy();
});

it.each([
  { description: "truncated references", riskCount: 2, scopeIDs: ["scope-a"], expected: "2 linked Risks · scope details incomplete" },
  { description: "an unresolved scope name", riskCount: 1, scopeIDs: ["internal-scope-reference"], expected: "1 linked Risk · scope details incomplete" },
  { description: "three known distinct areas", riskCount: 3, scopeIDs: ["scope-a", "scope-b", "scope-c"], expected: "3 linked Risks across multiple areas" },
  { description: "complete entity-level references", riskCount: 1, scopeIDs: [undefined], expected: "Clear Bank Nigeria" },
])("uses supported scope copy for $description", async ({ riskCount, scopeIDs, expected }) => {
  const item = itemFor("check-1", "Scoped indicator");
  item.risk_count = riskCount;
  item.risks = scopeIDs.map((organization_scope_id, index) => ({ ...item.risks[0]!, id: `risk-${index}`, organization_scope_id }));
  vi.mocked(loadIndicatorPopulation).mockResolvedValue({ items: [item], complete: true });
  render(<InsightsWorkspace {...actions} kind="KRI" legalEntityName="Clear Bank Nigeria" organizationScopes={[{ id: "scope-a", name: "Payments", kind: "DEPARTMENT" }]}/>);
  const table = await screen.findByRole("table", { name: "Risk indicators" });
  expect(await within(table).findByText(expected)).toBeTruthy();
  expect(within(table).queryByText("internal-scope-reference")).toBeNull();
});

function itemFor(id: string, name: string, kind: RiskIndicatorKind = "KRI"): IndicatorPopulationItem {
  return {
    indicator: {
      link: { id: `link-${id}`, risk_id: "risk-1", risk_version: 3, program_id: "program-1", monitoring_check_id: id, monitoring_check_version: 4, kind, measurement: "MONITORING_RISK_SCORE", created_at: "2026-10-05T08:00:00Z" },
      program_id: "program-1", program_name: "Channel resilience", check_id: id, check_code: "MOBILE-SUCCESS", check_name: name,
      claim: "Mobile transaction success remains above the approved limit.", check_status: "ACTIVE", check_version: 4, input_kind: "SOURCE",
      owner_display_name: "Ada Okafor", reviewer_display_name: "Tunde Bello", measurement: "MONITORING_RISK_SCORE", unit: "RISK_POINTS", denominator: 100,
      native_measurement: { field: "success_rate", label: "Success rate", unit: "PERCENT", precision: 2, value: "98.70", limits: [{ operator: "GREATER_OR_EQUAL", expected: "99.50" }] },
      state: "BREACH", reason: "Latest complete result is in a high or critical band.", score: 100, band: "CRITICAL", coverage: 1, minimum_coverage: .95, freshness_minutes: 60,
      result_id: "result-2", evaluated_at: "2026-10-05T10:00:00Z",
    },
    risks: [{ id: "risk-1", code: "RISK-1", name: "Payment interruption", organization_scope_id: "scope-a", status: "ACTIVE" }],
    risk_count: 1,
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((accept) => { resolve = accept; });
  return { promise, resolve };
}
