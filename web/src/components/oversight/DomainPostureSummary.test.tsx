import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { DomainPostureSummary } from "./DomainPostureSummary";
import type { DomainMetricBundle } from "../../metricApi";

function bundle(): DomainMetricBundle {
  const metric = (id: string, label: string, value: number, workspace: string) => ({
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
    generated_at: "2026-10-06T18:00:00Z",
    source_revision: "enterprise-domain-scoped-v1",
    definition_revision: "enterprise-domain-v1",
    basis: "CURRENT_POSTURE" as const,
    drill: { workspace, filter: id, consistency: "SOURCE_SNAPSHOT" as const },
  });
  return {
    generated_at: "2026-10-06T18:00:00Z",
    posture_as_of: "2026-10-06T18:00:00Z",
    scope_id: "scope-risk",
    scope_kind: "ORGANIZATION_SCOPE",
    source_id: "8f720000-0000-4000-8000-000000000001",
    source_revision: "enterprise-domain-scoped-v1",
    definition_revision: "enterprise-domain-v1",
    items: [
      metric("risks_outside_appetite", "Outside appetite", 1, "risks"),
      metric("indicator_breaches", "Indicator breaches", 0, "risks"),
      metric("assurance_failures", "Assurance failures", 0, "risks"),
      metric("losses_without_issue", "Losses without issue", 1, "losses"),
    ],
  };
}

it("opens exact scoped Risk members from the canonical posture card", async () => {
  const loadMembers = vi.fn().mockResolvedValue({
    source_id: "8f720000-0000-4000-8000-000000000001",
    metric_id: "risks_outside_appetite",
    definition_revision: "enterprise-domain-v1",
    count: 1,
    items: [{
      member_id: "8f720000-0000-4000-8000-000000000010",
      target_type: "RISK",
      target_id: "8f720000-0000-4000-8000-000000000020",
      target_title: "Payments availability risk",
      state: "BREACHED",
      accessible: true,
    }],
  });
  const onOpenRisk = vi.fn();

  render(<DomainPostureSummary
    bundle={bundle()}
    state="live"
    organizationScopeID="scope-risk"
    loadMembers={loadMembers}
    onOpenRisk={onOpenRisk}
  />);

  fireEvent.click(screen.getByRole("button", { name: /Outside appetite: 1/ }));

  await waitFor(() => expect(loadMembers).toHaveBeenCalledWith(
    "risks_outside_appetite",
    "8f720000-0000-4000-8000-000000000001",
    "enterprise-domain-v1",
    "scope-risk",
    undefined,
    50,
    expect.any(AbortSignal),
  ));
  expect(await screen.findByText("Payments availability risk")).toBeTruthy();

  fireEvent.click(screen.getByRole("button", { name: /Open record for Payments availability risk/ }));
  expect(onOpenRisk).toHaveBeenCalledWith("8f720000-0000-4000-8000-000000000020");
});

it("fails closed when exact membership does not match the card value", async () => {
  const loadMembers = vi.fn().mockResolvedValue({
    source_id: "8f720000-0000-4000-8000-000000000001",
    metric_id: "risks_outside_appetite",
    definition_revision: "enterprise-domain-v1",
    count: 2,
    items: [],
  });

  render(<DomainPostureSummary bundle={bundle()} state="live" loadMembers={loadMembers}/>);

  fireEvent.click(screen.getByRole("button", { name: /Outside appetite: 1/ }));
  expect(await screen.findByText(/Exact snapshot detail is unavailable/)).toBeTruthy();
});
