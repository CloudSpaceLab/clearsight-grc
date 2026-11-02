import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { GroupOversightSnapshot } from "../../groupOversightApi";
import type { GroupPostureBundle } from "../../groupPostureApi";
import { GroupOversightWorkspace } from "./GroupOversightWorkspace";

function attentionFixture(): GroupOversightSnapshot {
  return {
    revision_id: "attention-revision",
    generated_at: "2026-10-07T11:55:00Z",
    projection_version: "group-oversight-v1",
    freshness: "CURRENT",
    coverage: { authorized_children: 2, included_children: 2, missing_children: 0, stale_children: 0, complete: true },
    record_coverage: { population: 10, excluded: 0, unknown: 0 },
    counts: { critical_high: 5, overdue: 2, due_soon: 1, routing_failures: 1, unassigned: 0, outcome_failures: 3 },
    children: [
      {
        legal_entity_id: "entity-a",
        legal_entity_code: "A",
        legal_entity_name: "Alpha",
        jurisdiction: "NG",
        state: "AVAILABLE",
        child_snapshot_id: "snapshot-a",
        child_generated_at: "2026-10-07T11:55:00Z",
        child_projection_version: "oversight-v5",
        coverage: { population: 4, excluded: 0, unknown: 0 },
        counts: { critical_high: 2, overdue: 2, due_soon: 0, routing_failures: 0, unassigned: 0, outcome_failures: 1 },
        source_high_water: { matters: "2026-10-07T11:54:00Z" },
      },
      {
        legal_entity_id: "entity-b",
        legal_entity_code: "B",
        legal_entity_name: "Beta",
        jurisdiction: "GH",
        state: "AVAILABLE",
        child_snapshot_id: "snapshot-b",
        child_generated_at: "2026-10-07T11:55:00Z",
        child_projection_version: "oversight-v5",
        coverage: { population: 6, excluded: 0, unknown: 0 },
        counts: { critical_high: 3, overdue: 0, due_soon: 1, routing_failures: 1, unassigned: 0, outcome_failures: 2 },
        source_high_water: { matters: "2026-10-07T11:53:00Z" },
      },
    ],
  };
}

function postureFixture(): GroupPostureBundle {
  return {
    generated_at: "2026-10-07T12:00:00Z",
    period_start: "2026-09-08T00:00:00Z",
    period_end: "2026-10-07T12:00:00Z",
    definition_revision: "enterprise-domain-v1",
    risk_coverage: {
      authorized_children: 2,
      included_children: 2,
      missing_children: 0,
      stale_children: 0,
      partial_children: 0,
      complete: true,
    },
    loss_coverage: {
      authorized_children: 2,
      included_children: 2,
      missing_children: 0,
      stale_children: 0,
      partial_children: 0,
      complete: true,
    },
    counts: {
      risks_outside_appetite: 9,
      indicator_breaches: 3,
      assurance_failures: 2,
      loss_events: 5,
    },
    children: [
      {
        legal_entity_id: "entity-a",
        legal_entity_code: "A",
        legal_entity_name: "Alpha",
        jurisdiction: "NG",
        risk_state: "AVAILABLE",
        completeness: "COMPLETE",
        source_id: "domain-a",
        source_generated_at: "2026-10-07T11:55:00Z",
        source_revision: "enterprise-domain-v1",
        definition_revision: "enterprise-domain-v1",
        counts: { risks_outside_appetite: 6, indicator_breaches: 2, assurance_failures: 1, loss_events: 4 },
        unknown: 0,
        excluded: 0,
        freshness: "CURRENT",
      },
      {
        legal_entity_id: "entity-b",
        legal_entity_code: "B",
        legal_entity_name: "Beta",
        jurisdiction: "GH",
        risk_state: "AVAILABLE",
        completeness: "COMPLETE",
        source_id: "domain-b",
        source_generated_at: "2026-10-07T11:54:00Z",
        source_revision: "enterprise-domain-v1",
        definition_revision: "enterprise-domain-v1",
        counts: { risks_outside_appetite: 3, indicator_breaches: 1, assurance_failures: 1, loss_events: 1 },
        unknown: 0,
        excluded: 0,
        freshness: "CURRENT",
      },
    ],
  };
}

describe("GroupOversightWorkspace", () => {
  it("defaults to CRO Oversight and ranks authorized OpCos without record-level Group drill", async () => {
    const loadPosture = vi.fn().mockResolvedValue(postureFixture());
    const onOpenLegalEntity = vi.fn();

    render(<GroupOversightWorkspace
      organizationName="Clear Bank"
      initialSnapshot={attentionFixture()}
      onOpenLegalEntity={onOpenLegalEntity}
      loadPosture={loadPosture}
      now={new Date("2026-10-07T12:00:00Z")}
    />);

    expect(await screen.findByRole("heading", { name: "Group Home" })).toBeTruthy();
    expect(screen.getByRole("tab", { name: "Oversight" }).getAttribute("aria-selected")).toBe("true");
    await waitFor(() => expect(loadPosture).toHaveBeenCalledWith(
      { start_date: "2026-09-08", end_date: "2026-10-07" },
      expect.any(AbortSignal),
    ));

    expect(screen.getByRole("button", { name: /Outside appetite: 9/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: /Loss events: 5/ })).toBeTruthy();
    expect(screen.getByRole("list", { name: "Outside appetite by OpCo" })).toBeTruthy();
    expect(screen.queryByText("Critical & high")).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Open Alpha" }));
    expect(onOpenLegalEntity).toHaveBeenCalledWith("entity-a");

    fireEvent.click(screen.getByRole("button", { name: /Loss events: 5/ }));
    expect(screen.getByRole("list", { name: "Loss events by OpCo" })).toBeTruthy();
    expect(screen.getByText("4")).toBeTruthy();
  });

  it("changes Group Loss period locally without mutating Attention state", async () => {
    const loadPosture = vi.fn().mockResolvedValue(postureFixture());

    render(<GroupOversightWorkspace
      organizationName="Clear Bank"
      initialSnapshot={attentionFixture()}
      onOpenLegalEntity={vi.fn()}
      loadPosture={loadPosture}
      now={new Date("2026-10-07T12:00:00Z")}
    />);

    await waitFor(() => expect(loadPosture).toHaveBeenCalledTimes(1));
    fireEvent.click(screen.getByRole("button", { name: /Reporting period, Last 30 days/ }));
    fireEvent.click(screen.getByRole("button", { name: "90 days" }));

    await waitFor(() => expect(loadPosture).toHaveBeenLastCalledWith(
      { start_date: "2026-07-10", end_date: "2026-10-07" },
      expect.any(AbortSignal),
    ));

    fireEvent.click(screen.getByRole("tab", { name: "Attention" }));
    expect(await screen.findByText("Critical & high")).toBeTruthy();
    expect(screen.getByRole("table", { name: "Group OpCo attention" })).toBeTruthy();
  });


  it("does not display previous-period Group totals while the new period loads", async () => {
    let resolveNext: ((value: GroupPostureBundle) => void) | undefined;
    const pending = new Promise<GroupPostureBundle>((resolve) => { resolveNext = resolve; });
    const loadPosture = vi.fn().mockResolvedValueOnce(postureFixture()).mockReturnValueOnce(pending);

    render(<GroupOversightWorkspace
      organizationName="Clear Bank"
      initialSnapshot={attentionFixture()}
      onOpenLegalEntity={vi.fn()}
      loadPosture={loadPosture}
      now={new Date("2026-10-07T12:00:00Z")}
    />);

    expect(await screen.findByRole("button", { name: /Outside appetite: 9/ })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /Reporting period, Last 30 days/ }));
    fireEvent.click(screen.getByRole("button", { name: "90 days" }));

    expect(screen.queryByRole("button", { name: /Outside appetite: 9/ })).toBeNull();
    expect(screen.getByRole("button", { name: /Outside appetite: —.*Loading/ })).toBeTruthy();

    const next = postureFixture();
    next.counts.risks_outside_appetite = 12;
    resolveNext?.(next);
    expect(await screen.findByRole("button", { name: /Outside appetite: 12/ })).toBeTruthy();
  });

  it("keeps Risk quality separate from complete Group Loss-event coverage", async () => {
    const posture = postureFixture();
    posture.risk_coverage = {
      ...posture.risk_coverage,
      included_children: 1,
      missing_children: 1,
      complete: false,
    };
    posture.children[1] = {
      ...posture.children[1]!,
      risk_state: "MISSING",
      completeness: "UNKNOWN",
      source_id: undefined,
      source_generated_at: undefined,
      source_revision: undefined,
      definition_revision: undefined,
      counts: { ...posture.children[1]!.counts, risks_outside_appetite: 0, indicator_breaches: 0, assurance_failures: 0 },
      freshness: "STALE",
    };

    render(<GroupOversightWorkspace
      organizationName="Clear Bank"
      initialSnapshot={attentionFixture()}
      onOpenLegalEntity={vi.fn()}
      loadPosture={vi.fn().mockResolvedValue(posture)}
      now={new Date("2026-10-07T12:00:00Z")}
    />);

    expect(await screen.findByText("1 OpCo has no current Risk posture. Group risk posture is incomplete.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /Loss events: 5/ }));
    expect(await screen.findByText("2 OpCos · current loss data")).toBeTruthy();
  });

  it("bounds Group Attention to ten OpCos without discarding the complete scope count", async () => {
    const attention = attentionFixture();
    attention.children = Array.from({ length: 12 }, (_, index) => ({
      ...attention.children[0]!,
      legal_entity_id: `entity-${index}`,
      legal_entity_name: `OpCo ${String(index).padStart(2, "0")}`,
    }));
    attention.coverage.authorized_children = 12;
    attention.coverage.included_children = 12;

    render(<GroupOversightWorkspace
      organizationName="Clear Bank"
      homeTab="attention"
      initialSnapshot={attention}
      onOpenLegalEntity={vi.fn()}
      loadPosture={vi.fn().mockResolvedValue(postureFixture())}
    />);

    const table = screen.getByRole("table", { name: "Group OpCo attention" });
    expect(table.querySelectorAll("tbody tr")).toHaveLength(10);
    expect(screen.getByText("2 additional OpCos. Use the organization switcher for the full set.")).toBeTruthy();
    expect(screen.queryByText("OpCo 11")).toBeNull();
  });

  it("keeps Group My work as an OpCo handoff instead of cross-entity aggregation", async () => {
    const onOpenWork = vi.fn();

    render(<GroupOversightWorkspace
      organizationName="Clear Bank"
      initialSnapshot={attentionFixture()}
      onOpenLegalEntity={vi.fn()}
      onOpenWork={onOpenWork}
      loadPosture={vi.fn().mockResolvedValue(postureFixture())}
      now={new Date("2026-10-07T12:00:00Z")}
    />);

    await screen.findByRole("heading", { name: "Group Home" });
    fireEvent.click(screen.getByRole("tab", { name: "My work" }));
    expect(await screen.findByRole("heading", { name: "Assigned work stays within an OpCo" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Open current OpCo My work" }));
    expect(onOpenWork).toHaveBeenCalledTimes(1);
  });
});
