import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import type { DomainMetricBundle, OrganizationMetricBreakdown } from "../../metricApi";
import { OrganizationRiskSummary } from "./OrganizationRiskSummary";

function bundle(): DomainMetricBundle {
  return {
    generated_at: "2026-10-06T18:00:00Z",
    posture_as_of: "2026-10-06T18:00:00Z",
    scope_id: "bank-ng",
    scope_kind: "LEGAL_ENTITY",
    source_id: "8f740000-0000-4000-8000-000000000001",
    source_revision: "enterprise-domain-v1",
    definition_revision: "enterprise-domain-v1",
    items: [{
      id: "risks_outside_appetite",
      label: "Outside appetite",
      value: 12,
      unit: "COUNT",
      condition: "ATTENTION",
      freshness: "CURRENT",
      completeness: "COMPLETE",
      population: 40,
      excluded: 0,
      unknown: 0,
      generated_at: "2026-10-06T18:00:00Z",
      source_revision: "enterprise-domain-v1",
      definition_revision: "enterprise-domain-v1",
      basis: "CURRENT_POSTURE",
      drill: { workspace: "risks", filter: "outside-appetite", consistency: "SOURCE_SNAPSHOT" },
    }],
  };
}

it("shows a bounded exact organization ranking and switches through the authoritative scope callback", async () => {
  const breakdown: OrganizationMetricBreakdown = {
    source_id: "8f740000-0000-4000-8000-000000000001",
    metric_id: "risks_outside_appetite",
    definition_revision: "enterprise-domain-v1",
    count: 12,
    items: [
      { key: "scope:a", scope_id: "a", label: "Technology", kind: "ORGANIZATION_SCOPE", value: 3 },
      { key: "scope:b", scope_id: "b", label: "Operations", kind: "ORGANIZATION_SCOPE", value: 2 },
      { key: "scope:c", scope_id: "c", label: "Retail", kind: "ORGANIZATION_SCOPE", value: 1 },
      { key: "scope:d", scope_id: "d", label: "Finance", kind: "ORGANIZATION_SCOPE", value: 1 },
      { key: "scope:e", scope_id: "e", label: "Corporate", kind: "ORGANIZATION_SCOPE", value: 1 },
      { key: "scope:f", scope_id: "f", label: "Treasury", kind: "ORGANIZATION_SCOPE", value: 1 },
      { key: "scope:g", scope_id: "g", label: "Legal", kind: "ORGANIZATION_SCOPE", value: 1 },
      { key: "scope:h", scope_id: "h", label: "People", kind: "ORGANIZATION_SCOPE", value: 1 },
      { key: "unattributed", label: "Unattributed", kind: "UNATTRIBUTED", value: 1 },
    ],
  };
  const loadBreakdown = vi.fn().mockResolvedValue(breakdown);
  const onOpenScope = vi.fn();

  render(<OrganizationRiskSummary
    bundle={bundle()}
    loadBreakdown={loadBreakdown}
    onOpenScope={onOpenScope}
  />);

  await waitFor(() => expect(loadBreakdown).toHaveBeenCalledWith(
    "risks_outside_appetite",
    "8f740000-0000-4000-8000-000000000001",
    "enterprise-domain-v1",
    undefined,
    expect.any(AbortSignal),
  ));
  expect(await screen.findByRole("list", { name: "Outside-appetite risks by organization area" })).toBeTruthy();
  expect(screen.getByText("Other areas")).toBeTruthy();
  expect(screen.getByText("Unattributed")).toBeTruthy();
  expect(screen.queryByText("Legal")).toBeNull();
  expect(screen.queryByText("People")).toBeNull();

  fireEvent.click(screen.getByRole("button", { name: "Open Technology" }));
  expect(onOpenScope).toHaveBeenCalledWith("a");
});

it("fails closed when the organization breakdown does not match the exact metric count", async () => {
  const loadBreakdown = vi.fn().mockResolvedValue({
    source_id: "8f740000-0000-4000-8000-000000000001",
    metric_id: "risks_outside_appetite",
    definition_revision: "enterprise-domain-v1",
    count: 11,
    items: [],
  });

  render(<OrganizationRiskSummary bundle={bundle()} loadBreakdown={loadBreakdown}/>);

  expect(await screen.findByText("Risk concentration is unavailable for this scope.")).toBeTruthy();
  expect(screen.queryByRole("list", { name: "Outside-appetite risks by organization area" })).toBeNull();
});

it("shows a compact empty state without editorial labels", async () => {
  const emptyBundle = bundle();
  emptyBundle.items[0]!.value = 0;
  const loadBreakdown = vi.fn().mockResolvedValue({
    source_id: emptyBundle.source_id,
    metric_id: "risks_outside_appetite",
    definition_revision: "enterprise-domain-v1",
    count: 0,
    items: [],
  });

  render(<OrganizationRiskSummary bundle={emptyBundle} loadBreakdown={loadBreakdown}/>);

  expect(await screen.findByRole("heading", { name: "No risks outside appetite" })).toBeTruthy();
  expect(screen.getByRole("heading", { name: "Risk concentration" })).toBeTruthy();
  expect(screen.queryByText("Where risk is concentrated")).toBeNull();
  expect(screen.queryByText("Current outside-appetite risk population")).toBeNull();
});
