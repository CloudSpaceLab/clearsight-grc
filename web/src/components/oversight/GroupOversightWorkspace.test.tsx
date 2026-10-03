import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { GroupOversightSnapshot } from "../../groupOversightApi";
import { GroupOversightWorkspace } from "./GroupOversightWorkspace";

function fixture(): GroupOversightSnapshot {
  return {
    revision_id: "revision-secret",
    generated_at: "2026-10-03T20:00:00Z",
    projection_version: "group-oversight-v1",
    freshness: "CURRENT",
    coverage: {
      authorized_children: 2,
      included_children: 2,
      missing_children: 0,
      stale_children: 0,
      complete: true,
    },
    record_coverage: { population: 10, excluded: 0, unknown: 0 },
    counts: {
      critical_high: 5,
      overdue: 2,
      due_soon: 1,
      routing_failures: 1,
      unassigned: 0,
      outcome_failures: 3,
    },
    children: [
      {
        legal_entity_id: "entity-a",
        legal_entity_code: "A",
        legal_entity_name: "Alpha",
        jurisdiction: "NG",
        state: "AVAILABLE",
        child_snapshot_id: "snap-secret-a",
        child_generated_at: "2026-10-03T20:00:00Z",
        child_projection_version: "oversight-v5",
        counts: { critical_high: 2, overdue: 2, due_soon: 0, routing_failures: 0, unassigned: 0, outcome_failures: 1 },
        coverage: { population: 4, excluded: 0, unknown: 0 },
      },
      {
        legal_entity_id: "entity-b",
        legal_entity_code: "B",
        legal_entity_name: "Beta",
        jurisdiction: "GH",
        state: "AVAILABLE",
        child_snapshot_id: "snap-secret-b",
        child_generated_at: "2026-10-03T20:00:00Z",
        child_projection_version: "oversight-v5",
        counts: { critical_high: 3, overdue: 0, due_soon: 1, routing_failures: 1, unassigned: 0, outcome_failures: 2 },
        coverage: { population: 6, excluded: 0, unknown: 0 },
      },
    ],
  };
}

describe("GroupOversightWorkspace", () => {
  it("keeps revision details collapsed until the user asks for the exact data basis", async () => {
    const onOpenLegalEntity = vi.fn();
    render(<GroupOversightWorkspace
      organizationName="Clear Bank"
      onOpenLegalEntity={onOpenLegalEntity}
      loadGroup={vi.fn().mockResolvedValue(fixture())}
    />);

    expect(await screen.findByRole("heading", { name: "Group posture" })).toBeTruthy();
    expect(screen.getByText("5")).toBeTruthy();
    expect(screen.getByText("Alpha")).toBeTruthy();
    expect(screen.getByText("Beta")).toBeTruthy();
    expect(screen.queryByText("snap-secret-a")).toBeNull();
    expect(screen.queryByText("revision-secret")).toBeNull();

    const basisButton = screen.getByRole("button", { name: "Data basis" });
    expect(basisButton.getAttribute("aria-expanded")).toBe("false");
    fireEvent.click(basisButton);

    expect(await screen.findByRole("heading", { name: "Data basis" })).toBeTruthy();
    expect(screen.getByText("revision-secret")).toBeTruthy();
    expect(screen.getByText("snap-secret-a")).toBeTruthy();
    expect(screen.getByText("snap-secret-b")).toBeTruthy();
    expect(screen.getAllByText("oversight-v5").length).toBe(2);
    expect(screen.getByRole("button", { name: "Hide data basis" }).getAttribute("aria-expanded")).toBe("true");

    fireEvent.click(screen.getByRole("button", { name: /Open OpCo for Alpha/i }));
    expect(onOpenLegalEntity).toHaveBeenCalledWith("entity-a");
  });

  it("keeps a missing child explicit instead of presenting complete zero coverage", async () => {
    const value = fixture();
    value.freshness = "STALE";
    value.coverage.included_children = 1;
    value.coverage.missing_children = 1;
    value.coverage.complete = false;
    value.record_coverage.unknown = undefined;
    value.children[1] = {
      legal_entity_id: "entity-b",
      legal_entity_code: "B",
      legal_entity_name: "Beta",
      jurisdiction: "GH",
      state: "MISSING",
      counts: { critical_high: 0, overdue: 0, due_soon: 0, routing_failures: 0, unassigned: 0, outcome_failures: 0 },
      coverage: { population: 0 },
    };

    render(<GroupOversightWorkspace
      organizationName="Clear Bank"
      onOpenLegalEntity={() => {}}
      loadGroup={vi.fn().mockResolvedValue(value)}
    />);

    expect(await screen.findByText("1 OpCo has no current snapshot. Group totals are incomplete.")).toBeTruthy();
    expect(screen.getByText("No snapshot")).toBeTruthy();
    expect(screen.getAllByText("—").length).toBeGreaterThan(0);
  });
});
