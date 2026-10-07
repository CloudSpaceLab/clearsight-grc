import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import type { DomainMetricBundle } from "../../metricApi";
import { RiskMovement } from "./RiskMovement";

function bundle(value = 14): DomainMetricBundle {
  return {
    generated_at: "2026-10-07T11:55:00Z",
    posture_as_of: "2026-10-07T11:55:00Z",
    scope_id: "technology",
    scope_kind: "ORGANIZATION_SCOPE",
    source_id: "8f770000-0000-4000-8000-000000000001",
    source_revision: "enterprise-domain-scoped-v1",
    definition_revision: "enterprise-domain-v1",
    items: [{
      id: "risks_outside_appetite",
      label: "Outside appetite",
      value,
      unit: "COUNT",
      condition: value > 0 ? "ATTENTION" : "CLEAR",
      freshness: "CURRENT",
      completeness: "COMPLETE",
      population: 30,
      excluded: 0,
      unknown: 0,
      generated_at: "2026-10-07T11:55:00Z",
      source_revision: "enterprise-domain-scoped-v1",
      definition_revision: "enterprise-domain-v1",
      basis: "CURRENT_POSTURE",
      drill: { workspace: "risks", filter: "outside-appetite", consistency: "SOURCE_SNAPSHOT" },
    }],
  };
}

it("compares current scoped Risk posture to the exact historical day", async () => {
  const loadOrganizationTrend = vi.fn().mockResolvedValue({
    metric_id: "risks_outside_appetite",
    definition_revision: "enterprise-domain-v1",
    organization_scope_id: "technology",
    start: "2026-09-07T00:00:00Z",
    end: "2026-10-07T12:00:00Z",
    resolution: "DAY",
    points: [
      { date: "2026-09-07", at: "2026-09-07T23:55:00Z", value: 9, source_revision: "enterprise-domain-v1", source_complete: true },
      { date: "2026-10-06", at: "2026-10-06T23:55:00Z", value: 11, source_revision: "enterprise-domain-v1", source_complete: true },
    ],
  });
  const loadLegalEntityTrend = vi.fn();

  render(<RiskMovement
    bundle={bundle()}
    organizationScopeID="technology"
    loadOrganizationTrend={loadOrganizationTrend}
    loadLegalEntityTrend={loadLegalEntityTrend}
    now={new Date("2026-10-07T12:00:00Z")}
  />);

  await waitFor(() => expect(loadOrganizationTrend).toHaveBeenCalledWith(
    "risks_outside_appetite",
    "technology",
    "2026-09-07",
    "2026-10-07",
    expect.any(AbortSignal),
  ));
  expect(await screen.findByText("+5 worse vs 30 days ago")).toBeTruthy();
  expect(screen.getByText("14 now")).toBeTruthy();
  expect(screen.getByRole("img", { name: /Outside-appetite risk movement/ })).toBeTruthy();
  expect(loadLegalEntityTrend).not.toHaveBeenCalled();
});

it("does not claim direction without an exact complete baseline", async () => {
  const loadOrganizationTrend = vi.fn().mockResolvedValue({
    metric_id: "risks_outside_appetite",
    definition_revision: "enterprise-domain-v1",
    organization_scope_id: "technology",
    start: "2026-09-07T00:00:00Z",
    end: "2026-10-07T12:00:00Z",
    resolution: "DAY",
    points: [
      { date: "2026-09-08", at: "2026-09-08T23:55:00Z", value: 9, source_revision: "enterprise-domain-v1", source_complete: true },
    ],
  });

  render(<RiskMovement
    bundle={bundle()}
    organizationScopeID="technology"
    loadOrganizationTrend={loadOrganizationTrend}
    now={new Date("2026-10-07T12:00:00Z")}
  />);

  expect(await screen.findByText("No comparable history")).toBeTruthy();
  expect(screen.queryByRole("img", { name: /Outside-appetite risk movement/ })).toBeNull();
});

it("forces a daily legal-entity series for the seven-day comparison", async () => {
  const loadLegalEntityTrend = vi.fn().mockResolvedValue({
    metric_id: "risks_outside_appetite",
    definition_revision: "enterprise-domain-v1",
    start: "2026-09-29T00:00:00Z",
    end: "2026-10-07T12:00:00Z",
    resolution: "DAY",
    points: [
      {
        at: "2026-09-30T23:55:00Z", value: 12, freshness: "CURRENT", completeness: "COMPLETE",
        population: 30, excluded: 0, unknown: 0, source_revision: "enterprise-domain-v1",
      },
    ],
    direction: "UNKNOWN",
    comparison_quality: "MISSING",
  });

  render(<RiskMovement
    bundle={{ ...bundle(), scope_id: "bank-ng", scope_kind: "LEGAL_ENTITY" }}
    loadLegalEntityTrend={loadLegalEntityTrend}
    now={new Date("2026-10-07T12:00:00Z")}
  />);

  fireEvent.click(screen.getByRole("button", { name: "7d" }));
  await waitFor(() => expect(loadLegalEntityTrend).toHaveBeenLastCalledWith(
    "risks_outside_appetite",
    "2026-09-29",
    "2026-10-07",
    expect.any(AbortSignal),
  ));
  expect(await screen.findByText("+2 worse vs 7 days ago")).toBeTruthy();
});
