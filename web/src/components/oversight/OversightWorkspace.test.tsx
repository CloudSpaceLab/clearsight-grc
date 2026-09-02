import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { OversightWorkspace } from "./OversightWorkspace";

const api = vi.hoisted(() => ({ loadOversight: vi.fn() }));
vi.mock("../../oversightApi", () => api);

beforeEach(() => {
  api.loadOversight.mockResolvedValue({
    generated_at: "2026-09-01T07:55:00Z",
    period_start: "2026-06-03T08:00:00Z",
    period_end: "2026-09-01T08:00:00Z",
    projection_version: "oversight-v2",
    freshness: "CURRENT",
    source_high_water: { matters: "2026-09-01T07:54:00Z", actions: "2026-09-01T07:53:00Z", workflow_tasks: "2026-09-01T07:52:00Z", verification_results: "2026-09-01T07:51:00Z", continuity_events: "2026-09-01T07:54:30Z" },
    coverage: { population: 42, excluded: 1, unknown: 2 },
    counts: { critical_high: 7, overdue: 4, due_soon: 3, routing_failures: 1, unassigned: 2, outcome_failures: 1 },
    interventions: [{ target_type: "MATTER", target_id: "matter-1", title: "Verify vendor address", category: "VENDOR_DEFICIENCY", state: "VERIFICATION", priority: 5, owner_name: "Ada Okafor", due_at: "2026-08-31T08:00:00Z", reason: "The issue is overdue and remains open.", next_action: "Review the issue and confirm the current recovery plan" }],
    pressure: [{ category: "VENDOR_DEFICIENCY", critical: 2, high: 3, other: 1, overdue: 2 }],
    aging: [{ label: "0–7 days", count: 3 }, { label: "8–30 days", count: 6 }],
    performance: [{ owner_id: "person-1", owner_name: "Ada Okafor", current_load: 5, completed: 8, median_hours: 30, p75_hours: 52, sla_attainment: .875, reassigned: 2, returned: 1, blocked: 1, reopened: 1, measurement_samples: 8 }],
    estimates: [{ category: "VENDOR_DEFICIENCY", sample_size: 12, median_hours: 48, lower_hours: 30, upper_hours: 72, confidence: "MEDIUM", estimated_by: "Closed issues of the same type in this legal entity during the selected period" }],
  });
});

it("leads with exact interventions and provides table alternatives for oversight measures", async () => {
  const onOpenMatter = vi.fn();
  render(<OversightWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onOpenMatter={onOpenMatter}/>);

  await screen.findByRole("heading", { name: "Risk and delivery oversight" });
  expect(screen.getByText("7")).toBeTruthy();
  expect(screen.getByText("42 issues checked · 1 excluded · 2 unknown")).toBeTruthy();
  expect(screen.getByText("oversight-v2")).toBeTruthy();
  fireEvent.click(screen.getByText("Data freshness"));
  expect(screen.getByText("Continuity Events")).toBeTruthy();
  expect(screen.getByRole("table", { name: "Risk pressure by issue type" })).toBeTruthy();
  expect(screen.queryByText(/employee score/i)).toBeNull();

  fireEvent.click(screen.getByRole("button", { name: "Review Verify vendor address" }));
  expect(onOpenMatter).toHaveBeenCalledWith("matter-1");

  fireEvent.click(screen.getByRole("tab", { name: "Operating performance" }));
  expect(screen.getByText("87.5%")).toBeTruthy();
  expect(screen.getByText("8 completed · 8 measured")).toBeTruthy();
  expect(screen.getByRole("columnheader", { name: "Workflow history" })).toBeTruthy();
  expect(screen.getByText("1 blocked · 1 reopened · 2 reassigned · 1 returned")).toBeTruthy();
});

it("keeps unavailable projection state explicit instead of substituting sample metrics", async () => {
  api.loadOversight.mockRejectedValueOnce(new Error("unavailable"));
  render(<OversightWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onOpenMatter={vi.fn()}/>);
  await waitFor(() => expect(screen.getByRole("heading", { name: "Oversight information is unavailable" })).toBeTruthy());
  expect(screen.queryByText("7")).toBeNull();
});
