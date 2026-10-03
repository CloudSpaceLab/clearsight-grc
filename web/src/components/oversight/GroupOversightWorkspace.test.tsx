import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { GroupOversightResponse } from "../../groupOversightApi";
import { GroupOversightWorkspace } from "./GroupOversightWorkspace";

function fixture(): GroupOversightResponse {
  return {
    snapshot: {
      scope_id: "tenant-1",
      scope_name: "Clear Bank",
      scope_kind: "ORGANIZATION",
      generated_at: "2026-10-03T20:00:00Z",
      posture_as_of: "2026-10-03T19:59:00Z",
      freshness: "CURRENT",
      projection_version: "group-oversight-v1",
      contributor_revision: "revision-secret",
      coverage: {
        authorized_children: 2,
        contributing_children: 2,
        missing_children: 0,
        stale_children: 0,
        population: 10,
        excluded: 0,
        unknown: 0,
      },
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
          code: "A",
          name: "Alpha",
          jurisdiction: "NG",
          state: "CURRENT",
          snapshot_id: "snap-secret-a",
          counts: { critical_high: 2, overdue: 2, due_soon: 0, routing_failures: 0, unassigned: 0, outcome_failures: 1 },
          coverage: { population: 4, excluded: 0, unknown: 0 },
        },
        {
          legal_entity_id: "entity-b",
          code: "B",
          name: "Beta",
          jurisdiction: "GH",
          state: "CURRENT",
          snapshot_id: "snap-secret-b",
          counts: { critical_high: 3, overdue: 0, due_soon: 1, routing_failures: 1, unassigned: 0, outcome_failures: 2 },
          coverage: { population: 6, excluded: 0, unknown: 0 },
        },
      ],
      contributors: [
        { legal_entity_id: "entity-a", snapshot_id: "snap-secret-a", generated_at: "2026-10-03T20:00:00Z", posture_as_of: "2026-10-03T20:00:00Z", projection_version: "oversight-v5" },
        { legal_entity_id: "entity-b", snapshot_id: "snap-secret-b", generated_at: "2026-10-03T20:00:00Z", posture_as_of: "2026-10-03T20:00:00Z", projection_version: "oversight-v5" },
      ],
    },
    metrics: {
      generated_at: "2026-10-03T20:00:00Z",
      posture_as_of: "2026-10-03T19:59:00Z",
      scope_id: "tenant-1",
      scope_kind: "ORGANIZATION",
      freshness: "CURRENT",
      completeness: "COMPLETE",
      population: 10,
      excluded: 0,
      unknown: 0,
      source_revision: "revision-secret",
      definition_revision: "home-oversight-v2",
      items: [
        metric("critical_high_open", "Critical and high", 5, "critical-high"),
        metric("overdue_open", "Overdue", 2, "overdue"),
        metric("routing_gaps", "Routing gaps", 1, "routing-gaps"),
        metric("outcome_failures", "Outcome failures", 3, "outcome-failures"),
      ],
    },
  };
}

function metric(id: string, label: string, value: number, filter: string) {
  return {
    id,
    label,
    value,
    unit: "COUNT" as const,
    condition: value > 0 ? "ATTENTION" as const : "CLEAR" as const,
    freshness: "CURRENT" as const,
    completeness: "COMPLETE" as const,
    population: 10,
    excluded: 0,
    unknown: 0,
    generated_at: "2026-10-03T20:00:00Z",
    source_revision: "revision-secret",
    definition_revision: "home-oversight-v2",
    basis: "CURRENT_POSTURE" as const,
    drill: { workspace: "group-oversight", filter, consistency: "CURRENT_STATE" as const },
  };
}

describe("GroupOversightWorkspace", () => {
  it("shows canonical metrics and OpCo posture without exposing contributor identifiers", async () => {
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

    fireEvent.click(screen.getByRole("button", { name: /Open OpCo for Alpha/i }));
    expect(onOpenLegalEntity).toHaveBeenCalledWith("entity-a");
  });

  it("keeps a missing child explicit instead of presenting complete zero coverage", async () => {
    const value = fixture();
    value.snapshot.freshness = "STALE";
    value.snapshot.coverage.contributing_children = 1;
    value.snapshot.coverage.missing_children = 1;
    value.snapshot.coverage.unknown = undefined;
    value.snapshot.children[1] = {
      legal_entity_id: "entity-b",
      code: "B",
      name: "Beta",
      jurisdiction: "GH",
      state: "MISSING",
    };
    value.metrics.freshness = "STALE";
    value.metrics.completeness = "UNKNOWN";
    value.metrics.unknown = undefined;
    for (const item of value.metrics.items) {
      item.freshness = "STALE";
      item.completeness = "UNKNOWN";
      item.unknown = undefined;
    }

    render(<GroupOversightWorkspace
      organizationName="Clear Bank"
      onOpenLegalEntity={() => {}}
      loadGroup={vi.fn().mockResolvedValue(value)}
    />);

    expect(await screen.findByText("1 OpCo has no current snapshot. Group totals are incomplete.")).toBeTruthy();
    expect(screen.getByText("No snapshot")).toBeTruthy();
  });
});
