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
    posture_freshness: "CURRENT",
    posture_coverage: {
      authorized_children: 2,
      included_children: 2,
      missing_children: 0,
      stale_children: 0,
      complete: true,
    },
    posture: {
      risks_outside_appetite: 9,
      indicator_breaches: 3,
      assurance_failures: 2,
    },
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
        source_high_water: { matters: "2026-10-03T19:59:00Z", workflow_tasks: "2026-10-03T19:58:00Z" },
        domain_state: "AVAILABLE",
        domain_source_id: "domain-secret-a",
        domain_generated_at: "2026-10-03T20:00:00Z",
        domain_definition_revision: "enterprise-domain-v1",
        domain_posture: { risks_outside_appetite: 6, indicator_breaches: 2, assurance_failures: 1 },
        domain_source_high_water: { risks: "2026-10-03T19:59:30Z" },
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
        source_high_water: { matters: "2026-10-03T19:57:00Z" },
        domain_state: "AVAILABLE",
        domain_source_id: "domain-secret-b",
        domain_generated_at: "2026-10-03T20:00:00Z",
        domain_definition_revision: "enterprise-domain-v1",
        domain_posture: { risks_outside_appetite: 3, indicator_breaches: 1, assurance_failures: 1 },
        domain_source_high_water: { risks: "2026-10-03T19:58:30Z" },
      },
    ],
  };
}

describe("GroupOversightWorkspace", () => {
  it("keeps revision identifiers hidden until the user opens the exact Group data basis", async () => {
    const onOpenLegalEntity = vi.fn();
    render(<GroupOversightWorkspace
      organizationName="Clear Bank"
      onOpenLegalEntity={onOpenLegalEntity}
      loadGroup={vi.fn().mockResolvedValue(fixture())}
    />);

    expect(await screen.findByRole("heading", { name: "Group Home" })).toBeTruthy();
    expect(screen.getByRole("tab", { name: "Oversight" }).getAttribute("aria-selected")).toBe("true");
    expect(screen.getByRole("button", { name: /Outside appetite: 9/ })).toBeTruthy();
    expect(screen.getByRole("list", { name: "Outside appetite by OpCo" })).toBeTruthy();
    expect(screen.getByText("Alpha")).toBeTruthy();
    expect(screen.getByText("Beta")).toBeTruthy();
    expect(screen.queryByText("snap-secret-a")).toBeNull();
    expect(screen.queryByText("revision-secret")).toBeNull();

    fireEvent.click(screen.getByText("Data basis · 2 authorized OpCos"));
    expect(await screen.findByText("revision-secret")).toBeTruthy();
    expect(screen.getByText("snap-secret-a")).toBeTruthy();
    expect(screen.getByText("snap-secret-b")).toBeTruthy();
    expect(screen.getByText("domain-secret-a")).toBeTruthy();
    expect(screen.getByText("domain-secret-b")).toBeTruthy();
    expect(screen.getByText(/Matters .* · Workflow Tasks .*/)).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Open Alpha" }));
    expect(onOpenLegalEntity).toHaveBeenCalledWith("entity-a");
  });

  it("keeps a missing child explicit instead of presenting complete zero coverage", async () => {
    const value = fixture();
    value.freshness = "STALE";
    value.posture_freshness = "STALE";
    value.coverage.included_children = 1;
    value.coverage.missing_children = 1;
    value.coverage.complete = false;
    value.record_coverage.unknown = undefined;
    value.posture_coverage.included_children = 1;
    value.posture_coverage.missing_children = 1;
    value.posture_coverage.complete = false;
    value.children[1] = {
      legal_entity_id: "entity-b",
      legal_entity_code: "B",
      legal_entity_name: "Beta",
      jurisdiction: "GH",
      state: "MISSING",
      counts: { critical_high: 0, overdue: 0, due_soon: 0, routing_failures: 0, unassigned: 0, outcome_failures: 0 },
      coverage: { population: 0 },
      domain_state: "MISSING",
      domain_posture: { risks_outside_appetite: 0, indicator_breaches: 0, assurance_failures: 0 },
    };

    render(<GroupOversightWorkspace
      organizationName="Clear Bank"
      onOpenLegalEntity={() => {}}
      loadGroup={vi.fn().mockResolvedValue(value)}
    />);

    expect(await screen.findByText("1 OpCo has no current CRO posture. Group posture is incomplete.")).toBeTruthy();
    expect(screen.getByText("No CRO posture available")).toBeFalsy;
    fireEvent.click(screen.getByText("Data basis · 2 authorized OpCos"));
    expect(await screen.findByText("No CRO posture contributed.")).toBeTruthy();
  });
});

  it("keeps Group Attention and My work separate from CRO Oversight", async () => {
    const onOpenWork = vi.fn();
    render(<GroupOversightWorkspace
      organizationName="Clear Bank"
      onOpenLegalEntity={vi.fn()}
      onOpenWork={onOpenWork}
      loadGroup={vi.fn().mockResolvedValue(fixture())}
    />);

    await screen.findByRole("heading", { name: "Group Home" });
    expect(screen.queryByText("Critical & high")).toBeNull();

    fireEvent.click(screen.getByRole("tab", { name: "Attention" }));
    expect(await screen.findByText("Critical & high")).toBeTruthy();
    expect(screen.getByRole("table", { name: "Group OpCo attention" })).toBeTruthy();

    fireEvent.click(screen.getByRole("tab", { name: "My work" }));
    expect(await screen.findByRole("heading", { name: "Assigned work stays within an OpCo" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Open current OpCo My work" }));
    expect(onOpenWork).toHaveBeenCalledTimes(1);
  });
});
