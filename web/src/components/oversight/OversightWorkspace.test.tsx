import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { OversightWorkspace } from "./OversightWorkspace";
import type { AttentionItem } from "../../types";

const api = vi.hoisted(() => ({ loadOversight: vi.fn() }));
const metricApi = vi.hoisted(() => ({ loadHomeMetrics: vi.fn(), loadHomeMetricMembers: vi.fn() }));
vi.mock("../../oversightApi", () => api);
vi.mock("../../metricApi", () => metricApi);

beforeEach(() => {
  metricApi.loadHomeMetricMembers.mockReset();
  metricApi.loadHomeMetricMembers.mockRejectedValue(new Error("Exact membership not configured"));
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

function metric(id: string, label: string, value: number, filter: string) {
  return {
    id, label, value, unit: "COUNT", condition: value > 0 ? "ATTENTION" : "CLEAR",
    freshness: "CURRENT", completeness: "PARTIAL", population: 42, excluded: 1, unknown: 2,
    generated_at: "2026-09-01T07:55:00Z", source_revision: "oversight-v4", definition_revision: "home-oversight-v2", basis: "CURRENT_POSTURE",
    drill: { workspace: "oversight", filter, consistency: "CURRENT_STATE" },
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

it("leads with exact interventions and provides table alternatives for oversight measures", async () => {
  const onOpenMatter = vi.fn();
  render(<OversightWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onOpenMatter={onOpenMatter}/>);

  await screen.findByRole("heading", { name: "Risk and delivery oversight" });
  expect(screen.getByText("7")).toBeTruthy();
  expect(screen.getByText("42 issues checked · 1 excluded · 2 unknown")).toBeTruthy();
  const period = screen.getByRole("button", { name: /Reporting period/ });
  expect(period.textContent).toContain("Period");
  expect(period.textContent).toContain("Jun");
  expect(period.textContent).toContain("Sep");
  expect(screen.getByText(/Current · Updated/)).toBeTruthy();
  expect(screen.queryByText("Current snapshot")).toBeNull();
  fireEvent.click(screen.getByText("Data freshness"));
  expect(screen.getByText("Continuity Events")).toBeTruthy();
  expect(screen.getByText("12 of 14 completed issues have complete lifecycle events · 2 excluded because an opened or closed event is missing · employee handling time follows each recorded owner assignment; reassignment, return, blocked and reopen counts remain visible separately")).toBeTruthy();
  const pressureTable = screen.getByRole("table", { name: "Risk pressure by issue type" });
  expect(pressureTable.closest(".oversight-pressure-table")).toBeTruthy();
  expect(Array.from(pressureTable.querySelectorAll("thead th"), (header) => header.textContent)).toEqual(["Issue type", "Critical", "High", "Other", "Overdue"]);
  expect(screen.queryByText(/employee score/i)).toBeNull();

  fireEvent.click(screen.getByRole("button", { name: "Review Verify vendor address" }));
  expect(onOpenMatter).toHaveBeenCalledWith("matter-1");

  fireEvent.click(screen.getByRole("tab", { name: "Operating performance" }));
  expect(screen.getByText("87.5%")).toBeTruthy();
  expect(screen.getByText("8 completed · 8 measured")).toBeTruthy();
  expect(screen.getByText("2.2d p75 · 12h blocked")).toBeTruthy();
  expect(screen.getByRole("columnheader", { name: "Workflow history" })).toBeTruthy();
  expect(screen.getByText("1 blocked · 1 reopened · 2 reassigned · 1 returned")).toBeTruthy();
});

it("keeps canonical metrics visible when detailed analysis is unavailable", async () => {
  api.loadOversight.mockRejectedValueOnce(new Error("unavailable"));
  render(<OversightWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onOpenMatter={vi.fn()}/>);
  await waitFor(() => expect(screen.getByRole("heading", { name: "Oversight information is unavailable" })).toBeTruthy());
  expect(screen.getByText("7")).toBeTruthy();
  expect(screen.getByText("Detailed risk analysis is unavailable. Headline metrics remain separate and may still be current.")).toBeTruthy();
});

it("does not substitute Oversight counts when canonical metrics are unavailable", async () => {
  metricApi.loadHomeMetrics.mockRejectedValueOnce(new Error("unavailable"));
  render(<OversightWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onOpenMatter={vi.fn()}/>);
  await screen.findByRole("heading", { name: "Risk and delivery oversight" });
  expect(screen.getByRole("article", { name: /Critical and high: —.*Unavailable/ })).toBeTruthy();
  expect(screen.getByRole("article", { name: /Overdue: —.*Unavailable/ })).toBeTruthy();
});

it("filters interventions from an accessible metric and keeps Today work available in oversight", async () => {
  const onMetricFilterChange = vi.fn();
  const onOpenTodayItem = vi.fn();
  const todayItems: AttentionItem[] = [{
    id: "today-1", type: "MATTER", title: "Confirm the NDPA evidence owner", state: "ACTION_IN_PROGRESS",
    why_now: "The evidence review is due this week.", scope: "Clear Bank Nigeria", evidence: "NDPA program", owner: "Hakeem",
    due_at: "2026-09-25T10:00:00Z", primary_action: "Confirm evidence owner", action_target_type: "MATTER", action_target_id: "matter-1",
  }];
  render(<OversightWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onOpenMatter={vi.fn()}
    metricFilter="all" onMetricFilterChange={onMetricFilterChange} todayItems={todayItems} todayState="live" onOpenTodayItem={onOpenTodayItem}/>);

  await screen.findByRole("heading", { name: "Risk and delivery oversight" });
  fireEvent.click(screen.getByRole("button", { name: /Overdue.*4/i }));
  expect(onMetricFilterChange).toHaveBeenCalledWith("overdue");
  expect(screen.getByRole("heading", { name: "Your assigned work" })).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Open Confirm the NDPA evidence owner" }));
  expect(onOpenTodayItem).toHaveBeenCalledWith(todayItems[0]);
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

it("applies one exact server-backed period to both Home reads and keeps the end date current", async () => {
  render(<OversightWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onOpenMatter={vi.fn()}/>);
  await screen.findByRole("heading", { name: "Risk and delivery oversight" });

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
  expect(screen.getByRole("button", { name: /Reporting period/ }).textContent).toContain("Last 30 days");
  expect(screen.getByText("7")).toBeTruthy();

  fireEvent.click(screen.getByRole("tab", { name: "Operating performance" }));
  expect(screen.getByText("3 completed · 3 measured")).toBeTruthy();
});

it("submits a custom start date but never offers an editable historical end date", async () => {
  render(<OversightWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onOpenMatter={vi.fn()}/>);
  await screen.findByRole("heading", { name: "Risk and delivery oversight" });

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
  expect(await screen.findByText(/4 issues/)).toBeTruthy();
  expect(screen.getByText(/unassigned area excluded/)).toBeTruthy();
  expect(screen.getByText("Current issues in BANK / RISK and included sub-areas.")).toBeTruthy();
});


it("reorders existing Home sections for a my-work-first preference without hiding posture", async () => {
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
    homeFocus="MY_WORK"
  />);

  await screen.findByRole("heading", { name: "Risk and delivery oversight" });
  const workspace = screen.getByRole("heading", { name: "Risk and delivery oversight" }).closest(".oversight-workspace");
  const text = workspace?.textContent ?? "";
  expect(text.indexOf("Your assigned work")).toBeGreaterThanOrEqual(0);
  expect(text.indexOf("Critical and high")).toBeGreaterThanOrEqual(0);
  expect(text.indexOf("Your assigned work")).toBeLessThan(text.indexOf("Critical and high"));
  expect(screen.getByRole("button", { name: /Critical and high: 7/ })).toBeTruthy();
});
