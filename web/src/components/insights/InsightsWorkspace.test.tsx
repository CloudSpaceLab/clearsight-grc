import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import type { IndicatorPortfolioPage } from "../../indicatorInsightsApi";
import type { MonitoringResult } from "../../monitoringTypes";
import { InsightsWorkspace } from "./InsightsWorkspace";

const page: IndicatorPortfolioPage = {
  generated_at: "2026-10-06T07:00:00Z",
  items: [
    {
      kind: "KRI",
      program_id: "program-1",
      program_name: "Digital channels",
      check_id: "check-mobile",
      check_code: "MOBILE-SUCCESS",
      check_name: "Mobile success rate",
      claim: "Mobile transaction success remains at or above the approved limit.",
      check_status: "ACTIVE",
      check_version: 4,
      input_kind: "SOURCE",
      owner_display_name: "Channel Operations",
      reviewer_display_name: "Technology Risk",
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
      evaluated_at: "2026-10-06T06:55:00Z",
      risks: [
        { id: "risk-1", name: "Mobile channel availability" },
        { id: "risk-2", name: "Digital transaction failure" },
      ],
    },
    {
      kind: "KCI",
      program_id: "program-2",
      program_name: "Privileged access",
      check_id: "check-pam",
      check_code: "PAM-EXCEPTIONS",
      check_name: "Privileged access exceptions",
      claim: "Unresolved privileged-access exceptions remain within the approved limit.",
      check_status: "ACTIVE",
      check_version: 2,
      input_kind: "SOURCE",
      owner_display_name: "Security Operations",
      native_measurement: {
        field: "exceptions",
        label: "Open exceptions",
        unit: "COUNT",
        precision: 0,
        value: "2",
        limits: [{ operator: "LESS_OR_EQUAL", expected: "3" }],
        condition: "WITHIN",
      },
      state: "NORMAL",
      reason: "Latest native measurement is within its approved limit.",
      score: 0,
      band: "LOW",
      coverage: 1,
      minimum_coverage: 1,
      freshness_minutes: 120,
      result_id: "result-2",
      evaluated_at: "2026-10-06T06:45:00Z",
      risks: [{ id: "risk-3", name: "Privileged access misuse" }],
    },
  ],
};

const history: MonitoringResult[] = [{
  id: "result-1",
  monitoring_check_id: "check-mobile",
  monitoring_check_version: 4,
  evaluated_at: "2026-10-06T06:55:00Z",
  evaluation: {
    score: 100,
    band: "CRITICAL",
    coverage: 1,
    measurement: page.items[0]!.native_measurement,
  },
}];

it("shows each governed indicator once even when it supports multiple risks", async () => {
  const loadPortfolio = vi.fn().mockResolvedValue(page);
  render(<InsightsWorkspace
    organizationName="Fidelity Bank"
    legalEntityName="Nigeria"
    loadPortfolio={loadPortfolio}
    loadIndicatorResults={vi.fn().mockResolvedValue(history)}
  />);

  const table = await screen.findByRole("table", { name: "KRI and KCI indicators" });
  expect(within(table).getAllByRole("row")).toHaveLength(3);
  expect(within(table).getByText("98.70%")).toBeTruthy();
  expect(within(table).getByText("Limit ≥ 99.50%")).toBeTruthy();
  expect(within(table).getByText("Mobile channel availability · +1")).toBeTruthy();
  expect(within(table).getByText("Breach")).toBeTruthy();
  expect(loadPortfolio).toHaveBeenCalledWith(expect.objectContaining({ limit: 50 }), expect.any(AbortSignal));
});

it("opens the shared Indicator detail and keeps exact history on the selected check revision", async () => {
  const loadPortfolio = vi.fn().mockResolvedValue(page);
  const loadIndicatorResults = vi.fn().mockResolvedValue(history);
  render(<InsightsWorkspace
    organizationName="Fidelity Bank"
    legalEntityName="Nigeria"
    loadPortfolio={loadPortfolio}
    loadIndicatorResults={loadIndicatorResults}
  />);

  const table = await screen.findByRole("table", { name: "KRI and KCI indicators" });
  fireEvent.click(within(table).getByRole("button", { name: /Open indicator for KRI, Mobile success rate/ }));

  expect(await screen.findByRole("heading", { name: "Mobile success rate" })).toBeTruthy();
  await waitFor(() => expect(loadIndicatorResults).toHaveBeenCalledWith("check-mobile", 4));
  expect(screen.getByText("Mobile channel availability, Digital transaction failure")).toBeTruthy();
});

it("applies bounded type and condition filters without introducing a separate KRI app", async () => {
  const loadPortfolio = vi.fn().mockResolvedValue(page);
  render(<InsightsWorkspace
    organizationName="Fidelity Bank"
    legalEntityName="Nigeria"
    loadPortfolio={loadPortfolio}
    loadIndicatorResults={vi.fn().mockResolvedValue([])}
  />);

  await screen.findByRole("table", { name: "KRI and KCI indicators" });
  fireEvent.change(screen.getByLabelText("Type"), { target: { value: "KRI" } });
  fireEvent.change(screen.getByLabelText("Condition"), { target: { value: "BREACH" } });

  await waitFor(() => expect(loadPortfolio).toHaveBeenLastCalledWith(
    expect.objectContaining({ kind: "KRI", state: "BREACH", limit: 50 }),
    expect.any(AbortSignal),
  ));
});
