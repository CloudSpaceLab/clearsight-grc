import { render, screen, waitFor, within } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import type { MonitoringResult } from "../../monitoringTypes";
import type { RiskIndicatorDetail } from "../../riskTypes";
import { IndicatorDetail } from "./IndicatorDetail";

const indicator: RiskIndicatorDetail = {
  link: {
    id: "link-1",
    risk_id: "risk-1",
    risk_version: 3,
    program_id: "program-1",
    monitoring_check_id: "check-1",
    monitoring_check_version: 4,
    kind: "KRI",
    measurement: "MONITORING_RISK_SCORE",
    created_at: "2026-10-05T08:00:00Z",
  },
  program_id: "program-1",
  program_name: "Channel resilience",
  check_id: "check-1",
  check_code: "MOBILE-SUCCESS",
  check_name: "Mobile success rate",
  claim: "Mobile transaction success remains above the approved limit.",
  check_status: "ACTIVE",
  check_version: 4,
  input_kind: "SOURCE",
  owner_display_name: "Ada Okafor",
  reviewer_display_name: "Tunde Bello",
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
  },
  state: "BREACH",
  reason: "Latest complete result is in a high or critical band.",
  score: 100,
  band: "CRITICAL",
  coverage: 1,
  minimum_coverage: 0.95,
  freshness_minutes: 60,
  result_id: "result-2",
  evaluated_at: "2026-10-05T10:00:00Z",
};

const history: MonitoringResult[] = [
  {
    id: "result-2",
    monitoring_check_id: "check-1",
    monitoring_check_version: 4,
    evaluated_at: "2026-10-05T10:00:00Z",
    evaluation: {
      score: 100,
      band: "CRITICAL",
      coverage: 1,
      measurement: {
        field: "success_rate",
        label: "Success rate",
        unit: "PERCENT",
        precision: 2,
        value: "98.70",
        limits: [{ operator: "GREATER_OR_EQUAL", expected: "99.50" }],
        condition: "BREACHED",
      },
    },
  },
  {
    id: "result-1",
    monitoring_check_id: "check-1",
    monitoring_check_version: 4,
    evaluated_at: "2026-10-05T09:00:00Z",
    evaluation: {
      score: 0,
      band: "LOW",
      coverage: 1,
      measurement: {
        field: "success_rate",
        label: "Success rate",
        unit: "PERCENT",
        precision: 2,
        value: "99.80",
        limits: [{ operator: "GREATER_OR_EQUAL", expected: "99.50" }],
        condition: "WITHIN",
      },
    },
  },
];

it("shows the native value, responsibility and exact revision history", async () => {
  const loadResults = vi.fn().mockResolvedValue(history);
  render(<IndicatorDetail indicator={indicator} loadResults={loadResults}/>);

  expect(screen.getByRole("heading", { name: "Mobile success rate" })).toBeTruthy();
  expect(screen.getByText("98.70%")).toBeTruthy();
  expect(screen.getByText("Limit ≥ 99.50%")).toBeTruthy();
  expect(screen.getByText("Ada Okafor")).toBeTruthy();
  expect(screen.getByText("Tunde Bello")).toBeTruthy();

  await waitFor(() => expect(loadResults).toHaveBeenCalledWith("check-1", 4));
  const table = await screen.findByRole("table", { name: "Mobile success rate observation history" });
  expect(within(table).getByText("99.80%")).toBeTruthy();
  expect(within(table).getByText("Outside limit")).toBeTruthy();
  expect(within(table).getByText("Within limit")).toBeTruthy();
  expect(within(table).getByText("Critical concern")).toBeTruthy();
  expect(within(table).getByText("Low concern")).toBeTruthy();
});

it("shows a money Indicator without converting it to concern points", async () => {
  const money = {
    ...indicator,
    check_id: "check-loss",
    check_code: "LOSS-AMOUNT",
    check_name: "Monthly operational loss",
    claim: "Monthly operational loss remains within the approved limit.",
    native_measurement: {
      field: "loss_amount",
      label: "Loss amount",
      unit: "MONEY" as const,
      currency: "NGN",
      precision: 2,
      value: "1250000",
      limits: [{ operator: "LESS_OR_EQUAL" as const, expected: "1000000" }],
      condition: "BREACHED" as const,
    },
    score: 100,
  };
  const loadResults = vi.fn().mockResolvedValue([{
    id: "loss-result",
    monitoring_check_id: "check-loss",
    monitoring_check_version: 4,
    evaluated_at: "2026-10-05T10:00:00Z",
    evaluation: {
      score: 100,
      band: "CRITICAL" as const,
      coverage: 1,
      measurement: money.native_measurement,
    },
  }]);

  render(<IndicatorDetail indicator={money} loadResults={loadResults}/>);
  expect(screen.getByText("NGN 1,250,000.00")).toBeTruthy();
  expect(screen.getByText("Limit ≤ NGN 1,000,000.00")).toBeTruthy();
  expect(screen.getByText("100 / 100 concern")).toBeTruthy();

  const table = await screen.findByRole("table", { name: "Monthly operational loss observation history" });
  expect(within(table).getByText("Outside limit")).toBeTruthy();
});
