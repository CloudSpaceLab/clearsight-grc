import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { OversightWorkspace } from "./OversightWorkspace";
import type { AttentionItem } from "../../types";

const api = vi.hoisted(() => ({ loadOversight: vi.fn() }));
const metricApi = vi.hoisted(() => ({
  loadHomeMetrics: vi.fn(),
  loadHomeMetricMembers: vi.fn(),
  loadDomainMetrics: vi.fn(),
  loadDomainMetricMembers: vi.fn(),
  loadDomainMetricOrganizationBreakdown: vi.fn(),
  loadDomainMetricTrend: vi.fn(),
  loadDomainMetricOrganizationTrend: vi.fn(),
  loadLossPeriodMetrics: vi.fn(),
  loadLossPeriodMetricMembers: vi.fn(),
}));
vi.mock("../../oversightApi", () => api);
vi.mock("../../metricApi", () => metricApi);

beforeEach(() => {
  metricApi.loadHomeMetricMembers.mockReset();
  metricApi.loadHomeMetricMembers.mockRejectedValue(new Error("Exact membership not configured"));
  metricApi.loadDomainMetricMembers.mockReset();
  metricApi.loadDomainMetricMembers.mockRejectedValue(new Error("Exact domain membership not configured"));
  metricApi.loadDomainMetrics.mockReset();
  metricApi.loadDomainMetricTrend.mockReset();
  metricApi.loadDomainMetricTrend.mockRejectedValue(new Error("Risk history not configured"));
  metricApi.loadDomainMetricOrganizationTrend.mockReset();
  metricApi.loadDomainMetricOrganizationTrend.mockRejectedValue(new Error("Organization Risk history not configured"));
  metricApi.loadLossPeriodMetricMembers.mockReset();
  metricApi.loadLossPeriodMetricMembers.mockRejectedValue(new Error("Exact Loss membership not configured"));
  metricApi.loadLossPeriodMetrics.mockReset();
  metricApi.loadLossPeriodMetrics.mockResolvedValue(lossPeriodBundle());
  metricApi.loadDomainMetricOrganizationBreakdown.mockReset();
  metricApi.loadDomainMetricOrganizationBreakdown.mockResolvedValue({
    source_id: "8f710000-0000-4000-8000-000000000001",
    metric_id: "risks_outside_appetite",
    definition_revision: "enterprise-domain-v1",
    count: 9,
    items: [
      { key: "scope:technology", scope_id: "technology", label: "Technology", kind: "ORGANIZATION_SCOPE", value: 4 },
      { key: "scope:operations", scope_id: "operations", label: "Operations", kind: "ORGANIZATION_SCOPE", value: 3 },
      { key: "unattributed", label: "Unattributed", kind: "UNATTRIBUTED", value: 2 },
    ],
  });
  metricApi.loadDomainMetrics.mockResolvedValue({
    generated_at: "2026-09-01T07:55:00Z",
    posture_as_of: "2026-09-01T07:55:00Z",
    scope_id: "bank-ng",
    scope_kind: "LEGAL_ENTITY",
    source_id: "8f710000-0000-4000-8000-000000000001",
    source_revision: "enterprise-domain-v1",
    definition_revision: "enterprise-domain-v1",
    items: [
      domainMetric("risks_outside_appetite", "Outside appetite", 9, 20, 1),
      domainMetric("indicator_breaches", "Indicator breaches", 4, 12, 0),
      domainMetric("assurance_failures", "Assurance failures", 3, 20, 2),
      domainMetric("losses_without_issue", "Losses without issue", 2, 5, 0),
    ],
  });
  metricApi.loadHomeMetrics.mockResolvedValue({
    generated_at: "2026-09-01T07:55:00Z",
    period_start: "2026-06-03T08:00:00Z",
    period_end: "2026-09-01T08:00:00Z",
    reporting_period: { start_date: "2026-06-03", end_date: "2026-09-01", mode: "CURRENT_WINDOW", max_days: 365, historical_end_supported: false },
    posture_as_of: "2026-09-01T07:55:00Z",
    scope_id: "bank-ng",
    scope_kind: "LEGAL_ENTITY",
    freshness: "CURRENT",
    completeness: "PARTIAL",
    population: 42,
    excluded: 1,
    unknown: 2,
    source_revision: "oversight-v4",
    definition_revision: "home-oversight-v2",
    items: [
      metric("critical_high_open", "Critical and high", 7, "critical-high"),
      metric("overdue_open", "Overdue", 4, "overdue"),
      metric("routing_gaps", "Routing gaps", 1, "routing-gaps"),
      metric("outcome_failures", "Outcome failures", 1, "outcome-failures"),
    ],
  });
  api.loadOversight.mockResolvedValue({
    generated_at: "2026-09-01T07:55:00Z",
    period_start: "2026-06-03T08:00:00Z",
    period_end: "2026-09-01T08:00:00Z",
    reporting_period: { start_date: "2026-06-03", end_date: "2026-09-01", mode: "CURRENT_WINDOW", max_days: 365, historical_end_supported: false },
    posture_as_of: "2026-09-01T07:55:00Z",
    projection_version: "oversight-v4",
    freshness: "CURRENT",
    source_high_water: { matters: "2026-09-01T07:54:00Z", actions: "2026-09-01T07:53:00Z", workflow_tasks: "2026-09-01T07:52:00Z", verification_results: "2026-09-01T07:51:00Z", continuity_events: "2026-09-01T07:54:30Z" },
    coverage: { population: 42, excluded: 1, unknown: 2 },
    counts: { critical_high: 7, overdue: 4, due_soon: 3, routing_failures: 1, unassigned: 2, outcome_failures: 1 },
    interventions: [{ target_type: "MATTER", target_id: "matter-1", title: "Verify vendor address", category: "VENDOR_DEFICIENCY", state: "VERIFICATION", priority: 5, owner_name: "Ada Okafor", due_at: "2026-08-31T08:00:00Z", reason: "The issue is overdue and remains open.", next_action: "Review the issue and confirm the current recovery plan" }],
    pressure: [{ category: "VENDOR_DEFICIENCY", critical: 2, high: 3, other: 1, overdue: 2 }],
    aging: [{ label: "0–7 days", count: 3 }, { label: "8–30 days", count: 6 }],
    performance: [{ owner_id: "person-1", owner_name: "Ada Okafor", current_load: 5, completed: 8, median_hours: 30, p75_hours: 52, sla_attainment: .875, reassigned: 2, returned: 1, blocked: 1, blocked_hours: 12, reopened: 1, measurement_samples: 8 }],
    estimates: [{ category: "VENDOR_DEFICIENCY", sample_size: 12, median_hours: 48, lower_hours: 30, upper_hours: 72, confidence: "MEDIUM", estimated_by: "Closed issues of the same type in this legal entity during the selected period" }],
    history_quality: { completed_population: 14, complete_lifecycle: 12, missing_created_event: 1, missing_terminal_event: 1, excluded_from_durations: 2, reassigned_owner_excluded: 3, returned_owner_excluded: 2, blocked_owner_excluded: 1, reopened_owner_excluded: 1 },
  });
});

function lossPeriodBundle() {
  return {
    generated_at: "2026-09-01T07:56:00Z",
    period_start: "2026-06-03T00:00:00Z",
    period_end: "2026-09-01T23:59:59Z",
    scope_id: "bank-ng",
    scope_kind: "LEGAL_ENTITY",
    source_id: "8f790000-0000-4000-8000-000000000001",
    source_revision: "operational-loss-ledger-v1",
    definition_revision: "operational-loss-period-v1",
    event_count: 3,
    contributing_loss_count: 4,
    unattributed_event_count: 1,
    mixed_currencies: false,
    net_loss: { minor_units: "105000", currency: "NGN" },
    currencies: [{
      currency: "NGN",
      gross: { minor_units: "150000", currency: "NGN" },
      recovery: { minor_units: "50000", currency: "NGN" },
      reversal: { minor_units: "5000", currency: "NGN" },
      net: { minor_units: "105000", currency: "NGN" },
      loss_event_count: 3,
      recovery_event_count: 2,
      reversal_event_count: 1,
    }],
    organization_breakdown: [
      {
        key: "scope:technology", scope_id: "technology", label: "Technology", kind: "ORGANIZATION_SCOPE",
        loss_event_count: 2, contributing_loss_count: 3, mixed_currencies: false,
        net_loss: { minor_units: "80000", currency: "NGN" },
        currencies: [{
          currency: "NGN",
          gross: { minor_units: "100000", currency: "NGN" },
          recovery: { minor_units: "25000", currency: "NGN" },
          reversal: { minor_units: "5000", currency: "NGN" },
          net: { minor_units: "80000", currency: "NGN" },
          loss_event_count: 2, recovery_event_count: 1, reversal_event_count: 1,
        }],
      },
      {
        key: "unattributed", label: "Unattributed", kind: "UNATTRIBUTED",
        loss_event_count: 1, contributing_loss_count: 1, mixed_currencies: false,
        net_loss: { minor_units: "25000", currency: "NGN" },
        currencies: [{
          currency: "NGN",
          gross: { minor_units: "50000", currency: "NGN" },
          recovery: { minor_units: "25000", currency: "NGN" },
          reversal: { minor_units: "0", currency: "NGN" },
          net: { minor_units: "25000", currency: "NGN" },
          loss_event_count: 1, recovery_event_count: 1, reversal_event_count: 0,
        }],
      },
    ],
    flow_resolution: "WEEK",
    flow_points: [
      {
        start: "2026-06-03T00:00:00Z", end: "2026-06-09T23:59:59Z",
        loss_event_count: 1, contributing_loss_count: 1, mixed_currencies: false,
        net_loss: { minor_units: "50000", currency: "NGN" },
        currencies: [{ currency: "NGN", gross: { minor_units: "50000", currency: "NGN" }, recovery: { minor_units: "0", currency: "NGN" }, reversal: { minor_units: "0", currency: "NGN" }, net: { minor_units: "50000", currency: "NGN" }, loss_event_count: 1, recovery_event_count: 0, reversal_event_count: 0 }],
      },
      {
        start: "2026-08-27T00:00:00Z", end: "2026-09-01T23:59:59Z",
        loss_event_count: 2, contributing_loss_count: 3, mixed_currencies: false,
        net_loss: { minor_units: "55000", currency: "NGN" },
        currencies: [{ currency: "NGN", gross: { minor_units: "100000", currency: "NGN" }, recovery: { minor_units: "50000", currency: "NGN" }, reversal: { minor_units: "5000", currency: "NGN" }, net: { minor_units: "55000", currency: "NGN" }, loss_event_count: 2, recovery_event_count: 2, reversal_event_count: 1 }],
      },
    ],
    comparison: {
      period_start: "2026-03-04T00:00:00Z",
      period_end: "2026-06-02T23:59:59Z",
      event_count: 2,
      contributing_loss_count: 2,
      mixed_currencies: false,
      net_loss: { minor_units: "125000", currency: "NGN" },
      currencies: [],
      event_delta: 1,
      net_delta: { minor_units: "-20000", currency: "NGN" },
      direction: "IMPROVED",
      comparison_quality: "COMPLETE",
    },
  };
}

function metric(id: string, label: string, value: number, filter: string) {
  return {
    id, label, value, unit: "COUNT", condition: value > 0 ? "ATTENTION" : "CLEAR",
    freshness: "CURRENT", completeness: "PARTIAL", population: 42, excluded: 1, unknown: 2,
    generated_at: "2026-09-01T07:55:00Z", source_revision: "oversight-v4", definition_revision: "home-oversight-v2", basis: "CURRENT_POSTURE",
    drill: { workspace: "oversight", filter, consistency: "CURRENT_STATE" },
  };
}

function domainMetric(id: string, label: string, value: number, population: number, unknown: number) {
  return {
    id, label, value, unit: "COUNT", condition: value > 0 ? "ATTENTION" : "CLEAR",
    freshness: "CURRENT", completeness: unknown > 0 ? "PARTIAL" : "COMPLETE",
    population, excluded: 0, unknown,
    generated_at: "2026-09-01T07:55:00Z",
    source_revision: "enterprise-domain-v1",
    definition_revision: "enterprise-domain-v1",
    basis: "CURRENT_POSTURE",
    drill: { workspace: id === "losses_without_issue" ? "losses" : "risks", filter: id, consistency: "SOURCE_SNAPSHOT" },
  };
}

function exactMetric(id: string, label: string, value: number, filter: string) {
  return {
    ...metric(id, label, value, filter),
    source_revision: "oversight-v5",
    definition_revision: "home-oversight-v3",
    drill: { workspace: "oversight", filter, consistency: "SOURCE_SNAPSHOT" },
  };
}

function exactMetricBundle() {
  return {
    generated_at: "2026-09-01T07:55:00Z",
    period_start: "2026-06-03T08:00:00Z",
    period_end: "2026-09-01T08:00:00Z",
    reporting_period: { start_date: "2026-06-03", end_date: "2026-09-01", mode: "CURRENT_WINDOW", max_days: 365, historical_end_supported: false },
    posture_as_of: "2026-09-01T07:55:00Z",
    scope_id: "bank-ng",
    scope_kind: "LEGAL_ENTITY",
    freshness: "CURRENT",
    completeness: "PARTIAL",
    population: 42,
    excluded: 1,
    unknown: 2,
    source_id: "8f700000-0000-4000-8000-000000000001",
    source_revision: "oversight-v5",
    definition_revision: "home-oversight-v3",
    items: [
      exactMetric("critical_high_open", "Critical and high", 7, "critical-high"),
      exactMetric("overdue_open", "Overdue", 4, "overdue"),
      exactMetric("routing_gaps", "Routing gaps", 2, "routing-gaps"),
      exactMetric("outcome_failures", "Outcome failures", 1, "outcome-failures"),
    ],
  };
}

it("keeps oversight analysis separate from attention and assigned work", async () => {
  render(<OversightWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onOpenMatter={vi.fn()}/>);

  await screen.findByRole("heading", { name: "Home" });
  expect(screen.getByRole("tab", { name: "Oversight" }).getAttribute("aria-selected")).toBe("true");
  expect(screen.getByRole("table", { name: "Risk pressure by issue type" })).toBeTruthy();
  expect(screen.getByRole("button", { name: /Outside appetite: 9/ })).toBeTruthy();
  expect(screen.getByRole("button", { name: /Indicator breaches: 4/ })).toBeTruthy();
  expect(await screen.findByText("Net operational loss")).toBeTruthy();
  expect(await screen.findByRole("list", { name: "Net operational Loss by organization area" })).toBeTruthy();
  expect(screen.getByRole("img", { name: "Net operational loss trend" })).toBeTruthy();
  expect(await screen.findByRole("list", { name: "Outside-appetite risks by organization area" })).toBeTruthy();
  const riskConcentration = screen.getByRole("list", { name: "Outside-appetite risks by organization area" });
  expect(riskConcentration.textContent).toContain("Technology");
  expect(riskConcentration.textContent).toContain("Unattributed");
  expect(screen.queryByText("Critical and high")).toBeNull();
  expect(screen.queryByRole("heading", { name: "Your assigned work" })).toBeNull();

  const period = screen.getByRole("button", { name: /Reporting period/ });
  expect(period.textContent).toContain("Period");
  expect(screen.getByText(/Current · Updated/)).toBeTruthy();
  fireEvent.click(screen.getByText("Data freshness"));
  expect(screen.getByText("Continuity Events")).toBeTruthy();

  fireEvent.click(screen.getByRole("tab", { name: "Operating performance" }));
  expect(screen.getByText("87.5%")).toBeTruthy();
  expect(screen.getByText("8 completed · 8 measured")).toBeTruthy();
  expect(screen.getByRole("columnheader", { name: "Workflow history" })).toBeTruthy();
});

it("keeps canonical metrics visible when detailed analysis is unavailable", async () => {
  api.loadOversight.mockRejectedValueOnce(new Error("unavailable"));
  render(<OversightWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onOpenMatter={vi.fn()} homeTab="attention"/>);
  await screen.findByRole("heading", { name: "Home" });
  expect(screen.getByText("7")).toBeTruthy();
  expect(screen.getByText("Priority intervention detail is unavailable. Headline metrics may still be current.")).toBeTruthy();
});

it("does not substitute Oversight counts when canonical metrics are unavailable", async () => {
  metricApi.loadHomeMetrics.mockRejectedValueOnce(new Error("unavailable"));
  render(<OversightWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onOpenMatter={vi.fn()} homeTab="attention"/>);
  await screen.findByRole("heading", { name: "Home" });
  expect(screen.getByRole("article", { name: /Critical and high: —.*Unavailable/ })).toBeTruthy();
  expect(screen.getByRole("article", { name: /Overdue: —.*Unavailable/ })).toBeTruthy();
});

it("keeps Attention and My work isolated while preserving their actions", async () => {
  const onMetricFilterChange = vi.fn();
  const onOpenTodayItem = vi.fn();
  const onOpenWork = vi.fn();
  const todayItems: AttentionItem[] = [{
    id: "today-1", type: "MATTER", title: "Confirm the NDPA evidence owner", state: "ACTION_IN_PROGRESS",
    why_now: "The evidence review is due this week.", scope: "Clear Bank Nigeria", evidence: "NDPA program", owner: "Hakeem",
    due_at: "2026-09-25T10:00:00Z", primary_action: "Confirm evidence owner", action_target_type: "MATTER", action_target_id: "matter-1",
  }];

  render(<OversightWorkspace
    organizationName="Clear Bank"
    legalEntityName="Clear Bank Nigeria"
    onOpenMatter={vi.fn()}
    metricFilter="all"
    onMetricFilterChange={onMetricFilterChange}
    todayItems={todayItems}
    todayState="live"
    onOpenTodayItem={onOpenTodayItem}
    onOpenWork={onOpenWork}
    homeTab="attention"
  />);

  await screen.findByRole("heading", { name: "Home" });
  expect(screen.getByRole("heading", { name: "Priority interventions" })).toBeTruthy();
  expect(screen.queryByRole("heading", { name: "Your assigned work" })).toBeNull();
  expect(metricApi.loadDomainMetrics).not.toHaveBeenCalled();
  expect(metricApi.loadDomainMetricTrend).not.toHaveBeenCalled();
  expect(metricApi.loadDomainMetricOrganizationTrend).not.toHaveBeenCalled();
  expect(metricApi.loadLossPeriodMetrics).not.toHaveBeenCalled();

  fireEvent.click(screen.getByRole("button", { name: /Overdue.*4/i }));
  expect(onMetricFilterChange).toHaveBeenCalledWith("overdue");

  fireEvent.click(screen.getByRole("button", { name: "Open Work" }));
  expect(onOpenWork).toHaveBeenCalledTimes(1);
});

it("drills a retained v3 metric to the exact snapshot population with typed record actions", async () => {
  const onOpenMatter = vi.fn();
  const onOpenProgram = vi.fn();
  metricApi.loadHomeMetrics.mockResolvedValueOnce(exactMetricBundle());
  metricApi.loadHomeMetricMembers.mockResolvedValueOnce({
    source_id: "8f700000-0000-4000-8000-000000000001",
    metric_id: "routing_gaps",
    definition_revision: "home-oversight-v3",
    count: 2,
    items: [
      {
        member_id: "8f700000-0000-4000-8000-000000000010",
        target_type: "MATTER",
        target_id: "8f700000-0000-4000-8000-000000000020",
        target_title: "Assign control gap",
        state: "READY",
        accessible: true,
      },
      {
        member_id: "8f700000-0000-4000-8000-000000000011",
        target_type: "PROGRAM",
        target_id: "8f700000-0000-4000-8000-000000000021",
        target_title: "Assign Program review",
        state: "BLOCKED",
        accessible: true,
      },
    ],
  });

  render(<OversightWorkspace
    organizationName="Clear Bank"
    legalEntityName="Clear Bank Nigeria"
    onOpenMatter={onOpenMatter}
    onOpenProgram={onOpenProgram}
    metricFilter="routing-gaps"
  />);

  expect(await screen.findByText("2 exact records in this metric snapshot.")).toBeTruthy();
  expect(screen.getByRole("table", { name: "Exact routing gaps snapshot members" })).toBeTruthy();
  expect(screen.getByText("Assign control gap")).toBeTruthy();
  expect(screen.getByText("Assign Program review")).toBeTruthy();
  expect(metricApi.loadHomeMetricMembers).toHaveBeenCalledWith(
    "routing_gaps",
    "8f700000-0000-4000-8000-000000000001",
    "home-oversight-v3",
    undefined,
    undefined,
    50,
    expect.any(AbortSignal),
  );

  fireEvent.click(screen.getByRole("button", { name: /Open record for Assign control gap/ }));
  expect(onOpenMatter).toHaveBeenCalledWith("8f700000-0000-4000-8000-000000000020");
  fireEvent.click(screen.getByRole("button", { name: /Open record for Assign Program review/ }));
  expect(onOpenProgram).toHaveBeenCalledWith("8f700000-0000-4000-8000-000000000021");
});

it("fails closed when retained membership count disagrees with the selected card", async () => {
  metricApi.loadHomeMetrics.mockResolvedValueOnce(exactMetricBundle());
  metricApi.loadHomeMetricMembers.mockResolvedValueOnce({
    source_id: "8f700000-0000-4000-8000-000000000001",
    metric_id: "overdue_open",
    definition_revision: "home-oversight-v3",
    count: 3,
    items: [],
  });

  render(<OversightWorkspace
    organizationName="Clear Bank"
    legalEntityName="Clear Bank Nigeria"
    onOpenMatter={vi.fn()}
    metricFilter="overdue"
  />);

  expect(await screen.findByText("Exact snapshot detail is unavailable. The card value is unchanged.")).toBeTruthy();
  expect(screen.queryByText(/exact records in this metric snapshot/i)).toBeNull();
});

it("keeps v2 current-state metric drills on the existing bounded intervention path", async () => {
  render(<OversightWorkspace
    organizationName="Clear Bank"
    legalEntityName="Clear Bank Nigeria"
    onOpenMatter={vi.fn()}
    metricFilter="overdue"
  />);

  await screen.findByRole("heading", { name: "Overdue interventions" });
  expect(metricApi.loadHomeMetricMembers).not.toHaveBeenCalled();
  expect(screen.getByText(/ranked issue shown for overdue interventions/i)).toBeTruthy();
});

it("passes the applied Home reporting period into Insights rather than opening current-only indicators", async () => {
  const onOpenInsights = vi.fn();
  render(<OversightWorkspace
    organizationName="Clear Bank"
    legalEntityName="Clear Bank Nigeria"
    onOpenMatter={vi.fn()}
    onOpenInsights={onOpenInsights}
  />);
  const action = await screen.findByRole("button", { name: "Open in Insights" });
  fireEvent.click(action);
  expect(onOpenInsights).toHaveBeenCalledExactlyOnceWith({ start_date: "2026-06-03", end_date: "2026-09-01" });
  fireEvent.click(screen.getByRole("tab", { name: "My work" }));
  expect(screen.queryByRole("button", { name: "Open in Insights" })).toBeNull();
});

it("applies one exact server-backed period to both Home reads and keeps the end date current", async () => {
  render(<OversightWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onOpenMatter={vi.fn()}/>);
  await screen.findByRole("heading", { name: "Home" });

  fireEvent.click(screen.getByRole("button", { name: /Reporting period/ }));
  expect(screen.queryByLabelText("From")).toBeNull();
  expect(screen.getByText("Current posture stays current. History measures use this window.")).toBeTruthy();

  const nextPeriod = { start_date: "2026-08-03", end_date: "2026-09-01", mode: "CURRENT_WINDOW" as const, max_days: 365, historical_end_supported: false as const };
  const currentSnapshot = await vi.mocked(api.loadOversight).mock.results[0]!.value;
  const currentMetrics = await vi.mocked(metricApi.loadHomeMetrics).mock.results[0]!.value;
  vi.mocked(api.loadOversight).mockResolvedValueOnce({
    ...currentSnapshot,
    period_start: "2026-08-02T00:00:00Z",
    reporting_period: nextPeriod,
    performance: currentSnapshot.performance.map((item: any) => ({ ...item, completed: 3, measurement_samples: 3 })),
  });
  vi.mocked(metricApi.loadHomeMetrics).mockResolvedValueOnce({
    ...currentMetrics,
    period_start: "2026-08-02T00:00:00Z",
    reporting_period: nextPeriod,
  });
  vi.mocked(api.loadOversight).mockClear();
  vi.mocked(metricApi.loadHomeMetrics).mockClear();

  fireEvent.click(screen.getByRole("button", { name: "30 days" }));

  await waitFor(() => expect(api.loadOversight).toHaveBeenCalledWith({ start_date: "2026-08-03", end_date: "2026-09-01" }, undefined));
  expect(metricApi.loadHomeMetrics).toHaveBeenCalledWith({ start_date: "2026-08-03", end_date: "2026-09-01" }, undefined);
  await waitFor(() => expect(metricApi.loadLossPeriodMetrics).toHaveBeenCalledWith(
    { start_date: "2026-08-03", end_date: "2026-09-01" },
    undefined,
    expect.any(AbortSignal),
  ));
  expect(screen.getByRole("button", { name: /Reporting period/ }).textContent).toContain("Last 30 days");
  expect(screen.queryByText("7")).toBeNull();

  fireEvent.click(screen.getByRole("tab", { name: "Attention" }));
  expect(screen.getByText("7")).toBeTruthy();
  fireEvent.click(screen.getByRole("tab", { name: "Oversight" }));
  fireEvent.click(screen.getByRole("tab", { name: "Operating performance" }));
  expect(screen.getByText("3 completed · 3 measured")).toBeTruthy();
});

it("submits a custom start date but never offers an editable historical end date", async () => {
  render(<OversightWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onOpenMatter={vi.fn()}/>);
  await screen.findByRole("heading", { name: "Home" });

  fireEvent.click(screen.getByRole("button", { name: /Reporting period/ }));
  fireEvent.click(screen.getByRole("button", { name: "Custom start date" }));
  const from = screen.getByLabelText("From") as HTMLInputElement;
  expect(from.min).toBe("2025-09-02");
  expect(from.max).toBe("2026-09-01");
  expect(screen.getByLabelText("To 2026-09-01").textContent).toContain("Current reporting date");

  fireEvent.change(from, { target: { value: "2026-07-15" } });

  const currentSnapshot = await vi.mocked(api.loadOversight).mock.results[0]!.value;
  const currentMetrics = await vi.mocked(metricApi.loadHomeMetrics).mock.results[0]!.value;
  const customPeriod = { start_date: "2026-07-15", end_date: "2026-09-01", mode: "CURRENT_WINDOW" as const, max_days: 365, historical_end_supported: false as const };
  vi.mocked(api.loadOversight).mockResolvedValueOnce({ ...currentSnapshot, reporting_period: customPeriod, period_start: "2026-07-15T00:00:00Z" });
  vi.mocked(metricApi.loadHomeMetrics).mockResolvedValueOnce({ ...currentMetrics, reporting_period: customPeriod, period_start: "2026-07-15T00:00:00Z" });
  vi.mocked(api.loadOversight).mockClear();
  vi.mocked(metricApi.loadHomeMetrics).mockClear();

  fireEvent.click(screen.getByRole("button", { name: "Apply" }));
  await waitFor(() => expect(api.loadOversight).toHaveBeenCalledWith({ start_date: "2026-07-15", end_date: "2026-09-01" }, undefined));
  expect(metricApi.loadHomeMetrics).toHaveBeenCalledWith({ start_date: "2026-07-15", end_date: "2026-09-01" }, undefined);
});


it("reloads both Home reads for an organization scope and excludes unattributed legal-entity work", async () => {
  const baseSnapshot = await api.loadOversight();
  const baseMetrics = await metricApi.loadHomeMetrics();
  api.loadOversight.mockClear();
  metricApi.loadHomeMetrics.mockClear();
  const scopedSnapshot = {
    ...baseSnapshot,
    organization_scope_id: "scope-risk",
    coverage: { population: 4, excluded: 0 },
    counts: { ...baseSnapshot.counts, critical_high: 2 },
  };
  const scopedMetrics = {
    ...baseMetrics,
    scope_id: "scope-risk",
    scope_kind: "ORGANIZATION_SCOPE" as const,
    completeness: "UNKNOWN" as const,
    population: 4,
    unknown: undefined,
    items: baseMetrics.items.map((item: any) => ({ ...item, population: 4, unknown: undefined, completeness: "UNKNOWN" as const })),
  };
  vi.mocked(api.loadOversight).mockResolvedValue(scopedSnapshot);
  vi.mocked(metricApi.loadHomeMetrics).mockResolvedValue(scopedMetrics);

  render(<OversightWorkspace
    organizationName="Clear Bank"
    legalEntityName="Clear Bank Nigeria"
    organizationScopeID="scope-risk"
    organizationScopeName="BANK / RISK"
    onOpenMatter={vi.fn()}
  />);

  await waitFor(() => expect(api.loadOversight).toHaveBeenCalledWith(undefined, "scope-risk"));
  expect(metricApi.loadHomeMetrics).toHaveBeenCalledWith(undefined, "scope-risk");
  await waitFor(() => expect(metricApi.loadLossPeriodMetrics).toHaveBeenCalledWith(
    { start_date: "2026-06-03", end_date: "2026-09-01" },
    "scope-risk",
    expect.any(AbortSignal),
  ));
  expect(await screen.findByText(/4 issues/)).toBeTruthy();
  expect(screen.getByText(/unassigned area excluded/)).toBeTruthy();
  expect(screen.getByText("Current risk posture and operating context in BANK / RISK.")).toBeTruthy();
});


it("shows My work as a separate bounded Home intent", async () => {
  const todayItems: AttentionItem[] = [{
    id: "today-focus", type: "MATTER", title: "Review assigned exception", state: "ACTION_IN_PROGRESS",
    why_now: "The exception needs a decision.", scope: "Clear Bank Nigeria", evidence: "Current issue", owner: "Ada",
    due_at: "2026-09-25T10:00:00Z", primary_action: "Review exception", action_target_type: "MATTER", action_target_id: "matter-1",
  }];

  render(<OversightWorkspace
    organizationName="Clear Bank"
    legalEntityName="Clear Bank Nigeria"
    onOpenMatter={vi.fn()}
    todayItems={todayItems}
    todayState="live"
    homeTab="my-work"
  />);

  await screen.findByRole("heading", { name: "Home" });
  expect(screen.getByRole("heading", { name: "Your assigned work" })).toBeTruthy();
  expect(screen.queryByText("Critical and high")).toBeNull();
  expect(screen.queryByRole("table", { name: "Risk pressure by issue type" })).toBeNull();
  expect(screen.queryByRole("button", { name: /Reporting period/ })).toBeNull();
  expect(metricApi.loadDomainMetrics).not.toHaveBeenCalled();
  expect(metricApi.loadDomainMetricTrend).not.toHaveBeenCalled();
  expect(metricApi.loadDomainMetricOrganizationTrend).not.toHaveBeenCalled();
  expect(metricApi.loadLossPeriodMetrics).not.toHaveBeenCalled();
});

it("ignores a previous organization-scope response that arrives after a newer scope", async () => {
  const baselineSnapshot = await api.loadOversight();
  const baselineMetrics = await metricApi.loadHomeMetrics();
  let resolveOldSnapshot!: (value: typeof baselineSnapshot) => void;
  let resolveOldMetrics!: (value: typeof baselineMetrics) => void;
  const oldSnapshot = new Promise<typeof baselineSnapshot>((resolve) => { resolveOldSnapshot = resolve; });
  const oldMetrics = new Promise<typeof baselineMetrics>((resolve) => { resolveOldMetrics = resolve; });
  const currentSnapshot = { ...baselineSnapshot, organization_scope_id: "scope-new", coverage: { population: 7, excluded: 0 }, counts: { ...baselineSnapshot.counts, critical_high: 3 } };
  const currentMetrics = { ...baselineMetrics, scope_id: "scope-new", scope_kind: "ORGANIZATION_SCOPE" as const, population: 7, items: baselineMetrics.items.map((item: any) => item.id === "critical_high_open" ? { ...item, value: 3 } : item) };
  api.loadOversight.mockImplementation((_period: unknown, scope: string) => scope === "scope-old" ? oldSnapshot : Promise.resolve(currentSnapshot));
  metricApi.loadHomeMetrics.mockImplementation((_period: unknown, scope: string) => scope === "scope-old" ? oldMetrics : Promise.resolve(currentMetrics));

  const props = { organizationName: "Clear Bank", legalEntityName: "Clear Bank Nigeria", onOpenMatter: vi.fn(), homeTab: "attention" as const };
  const { rerender } = render(<OversightWorkspace {...props} organizationScopeID="scope-old"/>);
  await waitFor(() => expect(api.loadOversight).toHaveBeenCalledWith(undefined, "scope-old"));
  rerender(<OversightWorkspace {...props} organizationScopeID="scope-new"/>);
  await waitFor(() => expect(api.loadOversight).toHaveBeenCalledWith(undefined, "scope-new"));
  expect(await screen.findByText(/7 issues/)).toBeTruthy();
  expect(screen.getByRole("button", { name: /Critical and high: 3/ })).toBeTruthy();

  resolveOldSnapshot({ ...baselineSnapshot, organization_scope_id: "scope-old", coverage: { population: 88, excluded: 0 } });
  resolveOldMetrics({ ...baselineMetrics, scope_id: "scope-old", scope_kind: "ORGANIZATION_SCOPE" });
  await waitFor(() => expect(screen.getByText(/7 issues/)).toBeTruthy());
  expect(screen.queryByText(/88 issues/)).toBeNull();
  expect(screen.getByRole("button", { name: /Critical and high: 3/ })).toBeTruthy();
});

it("reloads Home projections when the actor invalidation revision changes", async () => {
  const { rerender } = render(<OversightWorkspace refreshToken="rev-1" organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onOpenMatter={vi.fn()}/>);
  await screen.findByRole("heading", { name: "Home" });
  await waitFor(() => expect(api.loadOversight).toHaveBeenCalledTimes(1));
  expect(metricApi.loadHomeMetrics).toHaveBeenCalledTimes(1);
  await waitFor(() => expect(metricApi.loadLossPeriodMetrics).toHaveBeenCalledTimes(1));

  rerender(<OversightWorkspace refreshToken="rev-2" organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onOpenMatter={vi.fn()}/>);
  await waitFor(() => expect(api.loadOversight).toHaveBeenCalledTimes(2));
  expect(metricApi.loadHomeMetrics).toHaveBeenCalledTimes(2);
  await waitFor(() => expect(metricApi.loadLossPeriodMetrics).toHaveBeenCalledTimes(2));
});
