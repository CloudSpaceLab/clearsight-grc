import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import type { MonitoringCheck } from "../../monitoringTypes";
import type { RiskIndicatorDetail, RiskRecord } from "../../riskTypes";
import type { ProgramSummary } from "../../summaryTypes";
import { RiskIndicatorsSection } from "./RiskIndicatorsSection";

const risk: RiskRecord = {
  id: "risk-1",
  tenant_id: "tenant-1",
  legal_entity_id: "entity-1",
  code: "NET-01",
  name: "Network resilience",
  category: "Operational resilience",
  statement: "Critical network service may exceed recovery tolerance.",
  cause: "Recovery paths can fail.",
  event: "Network interruption",
  impact: "Customers cannot use critical services.",
  scope: {},
  owner_principal_id: "owner-1",
  status: "ACTIVE",
  version: 4,
  created_at: "2026-10-02T09:00:00Z",
  updated_at: "2026-10-02T10:00:00Z",
};

const programSummary = {
  program: {
    id: "program-1",
    tenant_id: "tenant-1",
    legal_entity_id: "entity-1",
    code: "RESILIENCE",
    name: "Network resilience program",
    type: "ASSURANCE",
    status: "ACTIVE",
    owning_function: "Technology",
    scope: {},
    effective_from: "2026-01-01T00:00:00Z",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-10-02T09:00:00Z",
    version: 6,
  },
  state_label: "Current",
  overall_state: "CURRENT",
  reasons: [],
  reasons_total: 0,
  reasons_omitted: 0,
  open_matter_count: 0,
  requirement_count: 4,
  safeguard_count: 3,
  evidence_check_count: 2,
  program_version: 6,
  assessed_program_version: 6,
  projection_version: 2,
  projection_stale: false,
} satisfies ProgramSummary;

const activeCheck: MonitoringCheck = {
  id: "check-2",
  tenant_id: "tenant-1",
  program_id: "program-1",
  code: "FAILOVER",
  name: "Failover health",
  claim: "Failover remains within approved bounds.",
  input_kind: "SOURCE",
  binding_id: "binding-1",
  binding_version: 1,
  thresholds: { moderate_from: 25, high_from: 50, critical_from: 75 },
  freshness_minutes: 60,
  minimum_coverage: 0.95,
  failure_action: "RECOMMEND_MATTER",
  status: "ACTIVE",
  is_current: true,
  version: 2,
  created_at: "2026-10-01T09:00:00Z",
  updated_at: "2026-10-01T09:00:00Z",
};

const detail: RiskIndicatorDetail = {
  link: {
    id: "indicator-link-1",
    risk_id: risk.id,
    risk_version: 4,
    program_id: "program-1",
    monitoring_check_id: "check-1",
    monitoring_check_version: 3,
    kind: "KRI",
    measurement: "MONITORING_RISK_SCORE",
    created_at: "2026-10-02T09:30:00Z",
  },
  program_id: "program-1",
  program_name: "Network resilience program",
  check_id: "check-1",
  check_code: "RECOVERY",
  check_name: "Recovery health",
  claim: "Recovery remains within approved bounds.",
  check_status: "ACTIVE",
  check_version: 3,
  input_kind: "SOURCE",
  measurement: "MONITORING_RISK_SCORE",
  unit: "RISK_POINTS",
  denominator: 100,
  state: "UNKNOWN",
  reason: "Latest monitoring result is stale.",
  score: 62,
  band: "HIGH",
  coverage: 1,
  minimum_coverage: 0.95,
  freshness_minutes: 60,
  result_id: "result-1",
  evaluated_at: "2026-10-01T09:30:00Z",
};

it("shows Indicator state and explains unknown data without exposing principal identifiers", () => {
  render(<RiskIndicatorsSection
    risk={risk}
    actorID="viewer-1"
    indicators={[detail.link]}
    details={[{ ...detail, owner_display_name: "Jordan Ellis", reviewer_display_name: "Amina Okafor" }]}
    detailsComplete
    onReload={vi.fn()}
  />);

  const table = screen.getByRole("table", { name: "Risk indicators" });
  expect(within(table).getByText("KRI · Recovery health")).toBeTruthy();
  expect(within(table).getByText("Unknown")).toBeTruthy();
  expect(within(table).getByText("62 / 100")).toBeTruthy();
  expect(within(table).getByText("Latest monitoring result is stale.")).toBeTruthy();
  expect(screen.queryByText("owner-1")).toBeNull();
});

it("links an exact active MonitoringCheck revision through bounded Program search", async () => {
  const searchPrograms = vi.fn().mockResolvedValue({ items: [programSummary], generated_at: "2026-10-02T10:00:00Z" });
  const loadChecks = vi.fn().mockResolvedValue([
    { ...activeCheck, id: "check-linked", version: 1, name: "Already linked", code: "LINKED" },
    activeCheck,
    { ...activeCheck, id: "check-paused", status: "PAUSED", is_current: true, name: "Paused check", code: "PAUSED" },
  ]);
  const linkIndicator = vi.fn().mockResolvedValue({
    risk: { ...risk, version: 5 },
    indicator: {
      id: "indicator-link-2",
      risk_id: risk.id,
      risk_version: 5,
      program_id: "program-1",
      monitoring_check_id: activeCheck.id,
      monitoring_check_version: activeCheck.version,
      kind: "KRI",
      measurement: "MONITORING_RISK_SCORE",
      created_at: "2026-10-02T10:01:00Z",
    },
  });
  const onReload = vi.fn().mockResolvedValue(undefined);

  render(<RiskIndicatorsSection
    risk={risk}
    actorID="owner-1"
    indicators={[{
      id: "indicator-linked",
      risk_id: risk.id,
      risk_version: 3,
      program_id: "program-1",
      monitoring_check_id: "check-linked",
      monitoring_check_version: 1,
      kind: "KCI",
      measurement: "MONITORING_RISK_SCORE",
      created_at: "2026-10-02T09:00:00Z",
    }]}
    details={[]}
    detailsComplete
    onReload={onReload}
    searchPrograms={searchPrograms}
    loadChecks={loadChecks}
    linkIndicator={linkIndicator}
  />);

  fireEvent.click(screen.getByRole("button", { name: "Link indicator" }));
  await waitFor(() => expect(searchPrograms).toHaveBeenCalledWith(""));

  fireEvent.click(screen.getByRole("button", { name: /Choose a Program/ }));
  fireEvent.click(await screen.findByRole("option", { name: "Network resilience program · RESILIENCE" }));
  await waitFor(() => expect(loadChecks).toHaveBeenCalledWith("program-1"));

  expect(screen.queryByRole("option", { name: /Already linked/ })).toBeNull();
  expect(screen.queryByRole("option", { name: /Paused check/ })).toBeNull();

  fireEvent.click(screen.getByRole("button", { name: /Failover health/ }));
  fireEvent.click(await screen.findByRole("option", { name: /Failover health · FAILOVER/ }));
  fireEvent.click(screen.getByRole("button", { name: "Link indicator" }));

  await waitFor(() => expect(linkIndicator).toHaveBeenCalledWith("risk-1", 4, "check-2", 2, "KRI"));
  expect(onReload).toHaveBeenCalledTimes(1);
});

it("keeps Indicator linking hidden from a non-owner", () => {
  render(<RiskIndicatorsSection
    risk={risk}
    actorID="someone-else"
    indicators={[]}
    details={[]}
    detailsComplete
    onReload={vi.fn()}
  />);
  expect(screen.queryByRole("button", { name: "Link indicator" })).toBeNull();
});

it("opens the source Program from a linked Indicator", () => {
  const onOpenProgram = vi.fn();
  render(<RiskIndicatorsSection
    risk={risk}
    actorID="viewer-1"
    indicators={[detail.link]}
    details={[detail]}
    detailsComplete
    onReload={vi.fn()}
    onOpenProgram={onOpenProgram}
  />);

  fireEvent.click(screen.getByRole("button", { name: /Open Program/ }));
  expect(onOpenProgram).toHaveBeenCalledWith("program-1");
});


it("opens the visible intervention Matter without replacing the source Program action", () => {
  const onOpenProgram = vi.fn();
  const onOpenMatter = vi.fn();
  const withMatter: RiskIndicatorDetail = {
    ...detail,
    state: "BREACH",
    reason: "Latest complete result is in a high or critical band.",
    open_matter_id: "matter-1",
    open_matter_reference: "MAT-001",
    open_matter_status: "TRIAGE",
  };

  render(<RiskIndicatorsSection
    risk={risk}
    actorID="viewer-1"
    indicators={[withMatter.link]}
    details={[withMatter]}
    detailsComplete
    onReload={vi.fn()}
    onOpenProgram={onOpenProgram}
    onOpenMatter={onOpenMatter}
  />);

  const table = screen.getByRole("table", { name: "Risk indicators" });
  fireEvent.click(within(table).getByRole("button", { name: "MAT-001" }));
  expect(onOpenMatter).toHaveBeenCalledWith("matter-1");

  fireEvent.click(within(table).getByRole("button", { name: /Open Program/ }));
  expect(onOpenProgram).toHaveBeenCalledWith("program-1");
});
