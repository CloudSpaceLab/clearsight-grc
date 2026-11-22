import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import type { MonitoringResult } from "../../monitoringTypes";
import type { RiskIndicatorDetail } from "../../riskTypes";
import { IndicatorDetail } from "../../v210/IndicatorDetail";
import { ApiError } from "../../http";

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
    program_id: "program-1",
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
      rule_results: [{
        rule_id: "minimum-success",
        field_id: "success_rate",
        outcome: "FAIL",
        points: 100,
        critical: true,
        reason: "Observed success rate is below the approved limit.",
      }],
    },
  },
  {
    id: "result-1",
    program_id: "program-1",
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
  expect(screen.getByText("≥ 99.50%")).toBeTruthy();
  expect(screen.getByText("Ada Okafor")).toBeTruthy();
  expect(screen.getByText("Tunde Bello")).toBeTruthy();
  await waitFor(() => expect(loadResults).toHaveBeenCalledWith("check-1", 4));
  expect(await screen.findByRole("img", { name: "Mobile success rate evaluation trend" })).toBeTruthy();
  expect(screen.getByText(/Range: 98.70%–99.80%/)).toBeTruthy();
  const table = await screen.findByRole("table", { name: "Mobile success rate evaluation history" });
  expect(within(table).getByText("99.80%")).toBeTruthy();
  expect(within(table).getByText("Outside limit")).toBeTruthy();
  expect(within(table).getByText("Within limit")).toBeTruthy();
  expect(within(table).getByText("Critical concern")).toBeTruthy();
  expect(within(table).getByText("Low concern")).toBeTruthy();
  fireEvent.click(within(table).getAllByRole("button", { name: /View evaluation for/ })[0]!);
  expect(screen.getByRole("heading", { name: "Evaluation detail" })).toBeTruthy();
  expect(screen.getByText("Observed success rate is below the approved limit.")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Close detail" }));
  expect(screen.queryByRole("heading", { name: "Evaluation detail" })).toBeNull();
});

it.each([
  ["changed limit", { limits: [{ operator: "GREATER_OR_EQUAL", expected: "85" }] }],
  ["unknown result", { condition: "UNKNOWN" }],
  ["changed unit", { unit: "COUNT" }],
  ["changed duration unit", { duration_unit: "DAYS" }],
  ["unsafe numeric value", { value: "12345678901234567890" }],
])("does not graph history with a %s", async (_label, change) => {
  const results = [history[0]!, { ...history[1]!, evaluation: { ...history[1]!.evaluation, measurement: { ...history[1]!.evaluation.measurement!, ...change } } }];
  render(<IndicatorDetail indicator={indicator} loadResults={vi.fn().mockResolvedValue(results)}/>);
  await screen.findByRole("table", { name: "Mobile success rate evaluation history" });
  expect(screen.queryByRole("img", { name: /evaluation trend/ })).toBeNull();
});

it("does not graph native history without a current native measurement", async () => {
  render(<IndicatorDetail indicator={{ ...indicator, native_measurement: undefined }} loadResults={vi.fn().mockResolvedValue(history)}/>);
  await screen.findByRole("table", { name: "Mobile success rate evaluation history" });
  expect(screen.queryByRole("img", { name: /evaluation trend/ })).toBeNull();
});

it("shows a money indicator without converting its value to concern points", async () => {
  const money = { ...indicator, check_id: "check-loss", link: { ...indicator.link, monitoring_check_id: "check-loss" }, check_name: "Monthly operational loss", native_measurement: { field: "loss_amount", unit: "MONEY" as const, currency: "NGN", precision: 2, value: "1250000", limits: [{ operator: "LESS_OR_EQUAL" as const, expected: "1000000" }], condition: "BREACHED" as const } };
  const loadResults = vi.fn().mockResolvedValue([{ id: "loss-result", program_id: "program-1", monitoring_check_id: "check-loss", monitoring_check_version: 4, evaluated_at: "2026-10-05T10:00:00Z", evaluation: { score: 100, band: "CRITICAL", coverage: 1, measurement: money.native_measurement } }]);
  render(<IndicatorDetail indicator={money} loadResults={loadResults}/>);
  expect(screen.getByText("NGN 1,250,000.00")).toBeTruthy();
  expect(screen.getByText("≤ NGN 1,000,000.00")).toBeTruthy();
  expect(screen.getByText("100 / 100 concern")).toBeTruthy();
  expect(within(await screen.findByRole("table", { name: "Monthly operational loss evaluation history" })).getByText("Outside limit")).toBeTruthy();
});

it("links the source Program and the scoped issue", () => {
  const withIntervention = { ...indicator, intervention: { matter_id: "matter-7", reference: "INT-7", status: "ASSESSMENT", priority: 1, created_at: "2026-10-05T10:05:00Z" } };
  render(<IndicatorDetail indicator={withIntervention} loadResults={vi.fn().mockResolvedValue(history)}/>);
  expect(screen.getByRole("link", { name: "View program" }).getAttribute("href")).toBe("#/programs/program-1");
  expect(screen.getByRole("link", { name: "View issue INT-7" }).getAttribute("href")).toBe("#/work/issues/matter-7");
  expect(screen.getByText("Issue status: Assessment")).toBeTruthy();
});

it.each([{ monitoring_check_version: 5 }, { program_id: "other" }, { monitoring_check_id: "other" }])("withholds history returned for a different Program or check revision", async (mismatch) => {
  render(<IndicatorDetail indicator={indicator} loadResults={vi.fn().mockResolvedValue([{ ...history[0], ...mismatch }])}/>);
  expect(await screen.findByText("Evaluation history unavailable.")).toBeTruthy();
  expect(screen.queryByRole("table")).toBeNull();
});

it("does not plot distinct exact values that collapse to the same floating point number", async () => {
  const values = ["1.00000000000000001", "1.00000000000000002", "2"];
  const results = values.map((value, index) => ({ ...history[0], id: `r${index}`, evaluated_at: `2026-10-0${index + 1}T10:00:00Z`, evaluation: { ...history[0]!.evaluation, measurement: { ...history[0]!.evaluation.measurement!, value } } }));
  render(<IndicatorDetail indicator={indicator} loadResults={vi.fn().mockResolvedValue(results)}/>);
  expect(await screen.findByText("1.00000000000000001%")).toBeTruthy();
  expect(screen.queryByRole("img", { name: /evaluation trend/ })).toBeNull();
});

it("withholds a concern ratio with an invalid denominator", () => {
  render(<IndicatorDetail indicator={{ ...indicator, denominator: undefined } as unknown as RiskIndicatorDetail} loadResults={vi.fn().mockResolvedValue([])}/>);
  expect(screen.queryByText(/undefined concern/)).toBeNull();
  expect(screen.queryByText("100 / 100 concern")).toBeNull();
});

it("does not plot non-decimal source values as numeric readings", async () => {
  const results = history.map((result, index) => ({ ...result, evaluation: { ...result.evaluation, measurement: { ...result.evaluation.measurement!, value: index === 0 ? "0x10" : "2" } } }));
  render(<IndicatorDetail indicator={indicator} loadResults={vi.fn().mockResolvedValue(results)}/>);
  expect(await screen.findByText("0x10%")).toBeTruthy();
  expect(screen.queryByRole("img", { name: /evaluation trend/ })).toBeNull();
});

it("keeps the unknown status and reason beside the last recorded value", () => {
  render(<IndicatorDetail indicator={{ ...indicator, state: "UNKNOWN", reason: "Latest monitoring result is stale." }} loadResults={vi.fn().mockResolvedValue([])}/>);
  expect(screen.getByText("Unknown")).toBeTruthy();
  expect(screen.getByText(/Latest monitoring result is stale/)).toBeTruthy();
  expect(screen.getByText("98.70%")).toBeTruthy();
});

it("shows the recorded reporting period only when supplied", () => {
  const view = render(<IndicatorDetail indicator={indicator} loadResults={vi.fn().mockResolvedValue([])}/>);
  expect(screen.queryByText("Reporting period")).toBeNull();
  view.rerender(<IndicatorDetail indicator={{ ...indicator, native_measurement: { ...indicator.native_measurement!, reporting_period_start: "2026-10-01T00:00:00Z", reporting_period_end: "2026-10-02T00:00:00Z" } }} loadResults={vi.fn().mockResolvedValue([])}/>);
  expect(screen.getByText("Reporting period")).toBeTruthy();
  expect(screen.getByText(/Oct 1, 2026.*Oct 2, 2026/)).toBeTruthy();
});

it("clears selected indicator details when access is revoked", async () => {
  const denied = vi.fn();
  render(<IndicatorDetail indicator={indicator} loadResults={vi.fn().mockRejectedValue(new ApiError(403, "Denied"))} onAccessDenied={denied}/>);
  expect(await screen.findByText("Indicator unavailable.")).toBeTruthy();
  expect(screen.queryByText("Mobile success rate")).toBeNull();
  expect(denied).toHaveBeenCalledOnce();
});


it("does not infer that no issue exists when scoped issue information is absent", () => {
  render(<IndicatorDetail indicator={indicator} loadResults={vi.fn().mockResolvedValue([])}/>);
  expect(screen.queryByRole("link", { name: /View issue/ })).toBeNull();
  expect(screen.queryByText(/No open issue/)).toBeNull();
});
