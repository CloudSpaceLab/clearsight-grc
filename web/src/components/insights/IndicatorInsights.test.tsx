import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import type { IndicatorInsight, IndicatorInsightsPage } from "../../indicatorInsightsApi";
import { IndicatorInsights } from "./IndicatorInsights";

const indicator: IndicatorInsight = {
  kind: "KRI",
  risk_count: 2,
  program_id: "program-1",
  program_name: "Digital channels",
  check_id: "check-1",
  check_code: "MOBILE-SUCCESS",
  check_name: "Mobile success rate",
  claim: "Mobile transaction success remains at or above the approved limit.",
  check_status: "ACTIVE",
  check_version: 4,
  input_kind: "SOURCE",
  owner_display_name: "Channel Operations",
  reviewer_display_name: "Technology Risk",
  measurement: "MONITORING_RISK_SCORE",
  unit: "RISK_POINTS",
  denominator: 100,
  native_measurement: {
    field: "success_rate",
    label: "Success rate",
    unit: "PERCENT",
    precision: 2,
    value: "98.70",
    limits: [{ operator: "GREATER_OR_EQUAL", expected: "99.50" }],
    condition: "BREACHED",
  },
  state: "BREACH",
  reason: "Latest native measurement is outside its approved limit.",
  score: 100,
  band: "CRITICAL",
  coverage: 1,
  minimum_coverage: 0.95,
  freshness_minutes: 60,
  result_id: "result-1",
  evaluated_at: "2026-10-06T06:30:00Z",
};

function page(items: IndicatorInsight[] = [indicator]): IndicatorInsightsPage {
  return { items, complete: true, generated_at: "2026-10-06T06:35:00Z" };
}

it("shows one business-facing Indicator row with native value and linked-risk count", async () => {
  const loadPage = vi.fn().mockResolvedValue(page());
  render(<IndicatorInsights legalEntityName="Clear Bank Nigeria" loadPage={loadPage}/>);

  const table = await screen.findByRole("table", { name: "Indicator operating view" });
  expect(within(table).getByText("Mobile success rate")).toBeTruthy();
  expect(within(table).getByText("98.70%")).toBeTruthy();
  expect(within(table).getByText("Limit ≥ 99.50%")).toBeTruthy();
  expect(within(table).getByText("Breach")).toBeTruthy();
  expect(within(table).getByText("2")).toBeTruthy();
  expect(within(table).getByText("Channel Operations")).toBeTruthy();
  expect(loadPage).toHaveBeenCalledWith({ kind: undefined, cursor: undefined, limit: 25 }, expect.any(AbortSignal));
});

it("filters the same operating view to KCI without creating another workspace", async () => {
  const loadPage = vi.fn().mockResolvedValue(page());
  render(<IndicatorInsights loadPage={loadPage}/>);

  await screen.findByRole("table", { name: "Indicator operating view" });
  fireEvent.click(screen.getByRole("button", { name: /Indicator type/ }));
  fireEvent.click(await screen.findByRole("option", { name: "Key control indicators" }));

  await waitFor(() => expect(loadPage).toHaveBeenLastCalledWith(
    { kind: "KCI", cursor: undefined, limit: 25 },
    expect.any(AbortSignal),
  ));
});

it("keeps partial composition explicit instead of presenting a complete population", async () => {
  const loadPage = vi.fn().mockResolvedValue({ ...page(), complete: false });
  render(<IndicatorInsights loadPage={loadPage}/>);

  expect(await screen.findByText("Some Indicator details are unavailable under the current source or access state.")).toBeTruthy();
});
