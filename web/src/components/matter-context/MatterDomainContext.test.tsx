import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { loadIndicatorPopulation } from "../../indicatorApi";
import { sampleMobileSuccessIndicator } from "../../indicatorEvidenceData";
import { getLoss, listLosses } from "../../lossApi";
import { loadMonitoringResult } from "../../monitoringApi";
import type { MatterAggregate } from "../../types";
import { formatLossMoney } from "../losses/lossPresentation";
import { MatterDomainContext } from "./MatterDomainContext";

vi.mock("../../indicatorApi", () => ({ loadIndicatorPopulation: vi.fn() }));
vi.mock("../../lossApi", () => ({ getLoss: vi.fn(), listLosses: vi.fn() }));
vi.mock("../../monitoringApi", () => ({ loadMonitoringResult: vi.fn() }));

function matter(overrides: Partial<MatterAggregate["matter"]> = {}): MatterAggregate {
  return {
    type_label: "Issue",
    status_label: "Assessment",
    next_action: "Review issue",
    matter: {
      id: "matter-1",
      tenant_id: "bank",
      legal_entity_id: "entity-a",
      reference: "MAT-1",
      type: "INCIDENT",
      status: "ASSESSMENT",
      priority: 3,
      title: "Review issue",
      summary: "Review the recorded issue.",
      scope: {},
      known_facts: {},
      missing_facts: [],
      contradictions: [],
      created_at: "2026-10-01T09:00:00Z",
      updated_at: "2026-10-01T09:00:00Z",
      version: 1,
      ...overrides,
    },
    links: [],
    decisions: [],
    actions: [],
    verification_contracts: [],
    verification_results: [],
    response_packages: [],
    closure: { ready: false, reasons: [] },
  };
}

beforeEach(() => {
  vi.clearAllMocks();
});

it("checks the bounded loss ledger for a generic Matter", () => {
	vi.mocked(listLosses).mockResolvedValue({ items: [] });
  render(<MatterDomainContext aggregate={matter()}/>);
  expect(vi.mocked(getLoss)).not.toHaveBeenCalled();
  expect(vi.mocked(listLosses)).toHaveBeenCalledWith({ matterID: "matter-1", limit: 10 }, expect.any(AbortSignal));
  expect(vi.mocked(loadMonitoringResult)).not.toHaveBeenCalled();
  expect(vi.mocked(loadIndicatorPopulation)).not.toHaveBeenCalled();
});

it("shows loss and recovery totals linked to a generic Matter", async () => {
  vi.mocked(listLosses).mockResolvedValue({
    items: [{
      loss: {
        id: "loss-linked", tenant_id: "bank", legal_entity_id: "entity-a", code: "LOSS-101", title: "ATM settlement shortfall",
        event_type: "EXECUTION_DELIVERY_PROCESS_MANAGEMENT", cause: "Settlement mismatch", description: "", gross_amount_minor: 1_000_000,
        currency: "NGN", occurred_at: "2026-09-30T10:00:00Z", discovered_at: "2026-09-30T10:15:00Z", matter_id: "matter-1", status: "ACTIVE", version: 2,
        created_at: "2026-09-30T11:00:00Z", updated_at: "2026-10-01T11:00:00Z",
      },
      totals: { gross_amount_minor: 1_000_000, recovered_amount_minor: 400_000, net_loss_minor: 600_000, currency: "NGN", recovery_status: "PARTIAL" },
    }],
  });
  const onOpenLoss = vi.fn();

  render(<MatterDomainContext aggregate={matter()} onOpenLoss={onOpenLoss}/>);

  expect(await screen.findByRole("region", { name: "Linked losses" })).toBeTruthy();
  expect(screen.getByText("LOSS-101 · ATM settlement shortfall")).toBeTruthy();
  expect(screen.getByText(/Net .*Partly recovered/)).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Open loss LOSS-101" }));
  expect(onOpenLoss).toHaveBeenCalledWith("loss-linked");
});

it("shows canonical loss amounts and opens the linked loss record", async () => {
  vi.mocked(getLoss).mockResolvedValue({
    loss: {
      id: "loss-1",
      tenant_id: "bank",
      legal_entity_id: "entity-a",
      code: "LOSS-001",
      title: "Payments outage loss",
      event_type: "BUSINESS_DISRUPTION_SYSTEM_FAILURES",
      cause: "Payments switch outage",
      description: "",
      gross_amount_minor: 12_000_000,
      currency: "NGN",
      occurred_at: "2026-09-30T10:00:00Z",
      discovered_at: "2026-09-30T10:15:00Z",
      matter_id: "matter-1",
      status: "ACTIVE",
      version: 2,
      created_at: "2026-09-30T11:00:00Z",
      updated_at: "2026-10-01T11:00:00Z",
    },
    recoveries: [],
    totals: {
      gross_amount_minor: 12_000_000,
      recovered_amount_minor: 2_000_000,
      net_loss_minor: 10_000_000,
      currency: "NGN",
      recovery_status: "PARTIAL",
    },
  });
  const onOpenLoss = vi.fn();

  render(<MatterDomainContext
    aggregate={matter({ type: "OPERATIONAL_LOSS", source_type: "OPERATIONAL_LOSS", source_id: "loss-1" })}
    onOpenLoss={onOpenLoss}
  />);

  expect(await screen.findByRole("region", { name: "Operational loss context" })).toBeTruthy();
  expect(screen.getByText("LOSS-001 · Payments outage loss")).toBeTruthy();
  expect(screen.getByText(formatLossMoney(12_000_000, "NGN"))).toBeTruthy();
  expect(screen.getByText(formatLossMoney(10_000_000, "NGN"))).toBeTruthy();
  expect(screen.getByText("Partly recovered")).toBeTruthy();

  fireEvent.click(screen.getByRole("button", { name: "Open loss record" }));
  expect(onOpenLoss).toHaveBeenCalledWith("loss-1");
});

it("rejects a linked loss that points to another Matter", async () => {
  vi.mocked(getLoss).mockResolvedValue({
    loss: {
      id: "loss-1",
      tenant_id: "bank",
      legal_entity_id: "entity-a",
      code: "LOSS-001",
      title: "Payments outage loss",
      event_type: "OTHER",
      cause: "Unknown",
      description: "",
      gross_amount_minor: 100,
      currency: "NGN",
      occurred_at: "2026-09-30T10:00:00Z",
      discovered_at: "2026-09-30T10:15:00Z",
      matter_id: "different-matter",
      status: "ACTIVE",
      version: 1,
      created_at: "2026-09-30T11:00:00Z",
      updated_at: "2026-09-30T11:00:00Z",
    },
    recoveries: [],
    totals: { gross_amount_minor: 100, recovered_amount_minor: 0, net_loss_minor: 100, currency: "NGN", recovery_status: "NONE" },
  });

  render(<MatterDomainContext aggregate={matter({ type: "OPERATIONAL_LOSS", source_type: "OPERATIONAL_LOSS", source_id: "loss-1" })}/>);
  expect(await screen.findByText("Linked loss record is unavailable. Issue work remains available.")).toBeTruthy();
  expect(screen.queryByText("Payments outage loss")).toBeNull();
});

it("shows the exact source observation instead of substituting the current indicator value", async () => {
  vi.mocked(loadMonitoringResult).mockResolvedValue({
    id: "result-old",
    monitoring_check_id: "check-1",
    monitoring_check_version: 4,
    evaluated_at: "2026-09-05T10:00:00Z",
    evaluation: {
      score: 100,
      band: "CRITICAL",
      coverage: 0.98,
      measurement: {
        field: "success_rate",
        label: "Success rate",
        unit: "PERCENT",
        precision: 2,
        value: "98.10",
        limits: [{ operator: "GREATER_OR_EQUAL", expected: "99.50" }],
        condition: "BREACHED",
        reporting_period_start: "2026-08-01T00:00:00Z",
        reporting_period_end: "2026-08-31T23:59:59Z",
      },
    },
  });
  vi.mocked(loadIndicatorPopulation).mockResolvedValue({
    complete: true,
    items: [{
      indicator: {
        ...sampleMobileSuccessIndicator,
        check_id: "check-1",
        native_measurement: { ...sampleMobileSuccessIndicator.native_measurement!, value: "99.90", condition: "WITHIN" },
      },
      risk_count: 1,
      risks: [],
    }],
  });
  const onOpenIndicator = vi.fn();

  render(<MatterDomainContext
    aggregate={matter({ type: "CONTROL_GAP", source_type: "MONITORING_RESULT", source_id: "result-old", trigger_type: "MONITORING_RESULT_ADVERSE" })}
    onOpenIndicator={onOpenIndicator}
  />);

  expect(await screen.findByRole("region", { name: "Indicator observation context" })).toBeTruthy();
  expect(screen.getByText("98.10%")).toBeTruthy();
  expect(screen.queryByText("99.90%")).toBeNull();
  expect(screen.getByText("Limit ≥ 99.50%")).toBeTruthy();
  expect(screen.getByText("Outside limit")).toBeTruthy();

  fireEvent.click(screen.getByRole("button", { name: "Open indicator" }));
  expect(onOpenIndicator).toHaveBeenCalledWith("check-1", "KRI");
});

it("keeps the exact source observation visible when current indicator details cannot load", async () => {
  vi.mocked(loadMonitoringResult).mockResolvedValue({
    id: "result-old",
    monitoring_check_id: "check-1",
    monitoring_check_version: 4,
    evaluated_at: "2026-09-05T10:00:00Z",
    evaluation: {
      score: 80,
      band: "HIGH",
      coverage: 1,
      measurement: {
        field: "failed_transactions",
        unit: "COUNT",
        precision: 0,
        value: "112",
        limits: [{ operator: "LESS_OR_EQUAL", expected: "100" }],
        condition: "BREACHED",
      },
    },
  });
  vi.mocked(loadIndicatorPopulation).mockRejectedValue(new Error("unavailable"));

  render(<MatterDomainContext aggregate={matter({ type: "KRI_BREACH", source_type: "MONITORING_RESULT", source_id: "result-old" })}/>);

  expect(await screen.findByText("112")).toBeTruthy();
  expect(screen.getByText("Limit ≤ 100")).toBeTruthy();
  expect(screen.getByText("Current indicator details are unavailable. The source observation remains available.")).toBeTruthy();
  await waitFor(() => expect(loadIndicatorPopulation).toHaveBeenCalledTimes(1));
});
