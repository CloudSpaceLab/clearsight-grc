import { render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { RiskIndicatorDetail, RiskRecord } from "../../riskTypes";
import { RiskIndicatorsSection } from "./RiskIndicatorsSection";

const risk: RiskRecord = {
  id: "risk-1", tenant_id: "tenant-1", legal_entity_id: "entity-1", code: "NET-01",
  name: "Network resilience", category: "Operational", statement: "Service can fail.",
  cause: "Recovery failure", event: "Interruption", impact: "Customers cannot transact.",
  scope: {}, status: "ACTIVE", version: 1, created_at: "2026-10-01T09:00:00Z", updated_at: "2026-10-01T09:00:00Z",
};
const detail: RiskIndicatorDetail = {
  link: { id: "link-1", risk_id: risk.id, risk_version: 1, program_id: "program-1", monitoring_check_id: "check-1", monitoring_check_version: 1, kind: "KRI", measurement: "MONITORING_RISK_SCORE", created_at: "2026-10-01T09:00:00Z" },
  program_id: "program-1", program_name: "Resilience", check_id: "check-1", check_code: "RECOVERY", check_name: "Recovery health",
  claim: "Recovery meets its target.", check_status: "ACTIVE", check_version: 1, input_kind: "SOURCE",
  measurement: "MONITORING_RISK_SCORE", unit: "RISK_POINTS", denominator: 100,
  state: "UNKNOWN", reason: "Latest monitoring result is stale.", score: 74.99, band: "HIGH",
  coverage: 1, minimum_coverage: 0.95, freshness_minutes: 60, result_id: "result-1", evaluated_at: "2026-10-01T10:00:00Z",
  owner_display_name: "Jordan Ellis",
};

function show(item: RiskIndicatorDetail) {
  render(<RiskIndicatorsSection risk={risk} indicators={[item.link]} details={[item]} detailsComplete onReload={vi.fn().mockResolvedValue(undefined)}/>);
  return within(screen.getByRole("table", { name: "Risk indicators" }));
}

describe("indicator score meaning and ownership", () => {
  it("shows concern points, named owner and last recorded state without rounding", () => {
    const table = show(detail);
    expect(table.getByRole("columnheader", { name: "Concern score" })).toBeTruthy();
    expect(table.getByText("74.99 / 100")).toBeTruthy();
    expect(table.getByText("Owner: Jordan Ellis")).toBeTruthy();
    expect(table.getByText("Last recorded")).toBeTruthy();
    expect(table.getByText("Unknown")).toBeTruthy();
    expect(table.queryByText("75 / 100")).toBeNull();
    expect(table.queryByText(/% risk/)).toBeNull();
  });

  it("does not invent a zero or an unassigned owner for unavailable data", () => {
    const table = show({ ...detail, score: undefined, owner_display_name: undefined });
    expect(table.getByText("Not assessed")).toBeTruthy();
    expect(table.getByText("Owner name unavailable")).toBeTruthy();
    expect(table.queryByText("0 / 100")).toBeNull();
    expect(table.queryByText("Last recorded")).toBeNull();
    expect(table.queryByText("Not assigned")).toBeNull();
  });

  it("keeps a current zero score distinct from missing and stale results", () => {
    const table = show({ ...detail, state: "NORMAL", score: 0, band: "LOW", reason: "Latest complete result is in the low band." });
    expect(table.getByText("0 / 100")).toBeTruthy();
    expect(table.getByText("Normal")).toBeTruthy();
    expect(table.queryByText("Last recorded")).toBeNull();
  });
});
