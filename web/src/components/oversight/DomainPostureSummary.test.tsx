import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { DomainPostureSummary } from "./DomainPostureSummary";
import type { DomainMetricBundle, LossPeriodBundle } from "../../metricApi";

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

function mixedLossBundle(): LossPeriodBundle {
  return {
    generated_at: "2026-10-06T18:00:00Z",
    period_start: "2026-09-07T00:00:00Z",
    period_end: "2026-10-06T18:00:00Z",
    scope_id: "scope-risk",
    scope_kind: "ORGANIZATION_SCOPE",
    source_id: "8f720000-0000-4000-8000-000000000101",
    source_revision: "operational-loss-ledger-v1",
    definition_revision: "operational-loss-period-v1",
    event_count: 0,
    contributing_loss_count: 2,
    unattributed_event_count: 0,
    mixed_currencies: true,
    currencies: [
      {
        currency: "NGN",
        gross: { minor_units: "0", currency: "NGN" },
        recovery: { minor_units: "10000", currency: "NGN" },
        reversal: { minor_units: "0", currency: "NGN" },
        net: { minor_units: "-10000", currency: "NGN" },
        loss_event_count: 0,
        recovery_event_count: 1,
        reversal_event_count: 0,
      },
      {
        currency: "USD",
        gross: { minor_units: "0", currency: "USD" },
        recovery: { minor_units: "500", currency: "USD" },
        reversal: { minor_units: "0", currency: "USD" },
        net: { minor_units: "-500", currency: "USD" },
        loss_event_count: 0,
        recovery_event_count: 1,
        reversal_event_count: 0,
      },
    ],
    organization_breakdown: [],
    flow_resolution: "DAY",
    flow_points: [],
    comparison: {
      period_start: "2026-08-08T00:00:00Z",
      period_end: "2026-09-06T23:59:59Z",
      event_count: 0,
      contributing_loss_count: 0,
      mixed_currencies: false,
      currencies: [],
      event_delta: 0,
      direction: "UNKNOWN",
      comparison_quality: "LIMITED",
    },
  };
}

it("keeps mixed-currency recovery contributors drillable even with zero new events", async () => {
  const loadLossMembers = vi.fn().mockResolvedValue({
    source_id: "8f720000-0000-4000-8000-000000000101",
    metric_id: "operational_loss_net",
    definition_revision: "operational-loss-period-v1",
    count: 2,
    items: [
      {
        member_id: "8f720000-0000-4000-8000-000000000110",
        target_type: "LOSS",
        target_id: "8f720000-0000-4000-8000-000000000120",
        target_title: "Recovery-only NGN Loss",
        state: "RECOVERY",
        accessible: true,
      },
      {
        member_id: "8f720000-0000-4000-8000-000000000111",
        target_type: "LOSS",
        target_id: "8f720000-0000-4000-8000-000000000121",
        target_title: "Recovery-only USD Loss",
        state: "RECOVERY",
        accessible: true,
      },
    ],
  });

  render(<DomainPostureSummary
    bundle={bundle()}
    state="live"
    lossBundle={mixedLossBundle()}
    lossState="live"
    organizationScopeID="scope-risk"
    loadLossMembers={loadLossMembers}
  />);

  fireEvent.click(screen.getByRole("button", { name: /Net operational loss: 0 events/ }));

  await waitFor(() => expect(loadLossMembers).toHaveBeenCalledWith(
    "operational_loss_net",
    "8f720000-0000-4000-8000-000000000101",
    "operational-loss-period-v1",
    "scope-risk",
    undefined,
    50,
    expect.any(AbortSignal),
  ));
  expect(await screen.findByText("2 exact records in this metric snapshot.")).toBeTruthy();
});

