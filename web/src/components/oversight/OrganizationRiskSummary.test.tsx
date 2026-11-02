import { useLayoutEffect, useRef } from "react";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
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

it.each([false, true])("shows an accessible exact organization ranking and scope actions with embedded=%s", async (embedded) => {
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
    embedded={embedded}
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
  const list = await screen.findByRole("list", { name: "Outside-appetite risks by organization area" });
  expect(list.getAttribute("data-layout")).toBe(embedded ? "inline" : "stacked");
  expect(screen.getByText("Other areas")).toBeTruthy();
  expect(screen.getByText("Unattributed")).toBeTruthy();
  expect(screen.queryByText("Legal")).toBeNull();
  expect(screen.queryByText("People")).toBeNull();

  const technology = screen.getByRole("button", { name: "Open Technology" });
  const label = technology.querySelector(".cs-ranked-bars__label");
  const value = technology.querySelector(".cs-ranked-bars__value");
  const track = technology.querySelector(".cs-ranked-bars__track");
  expect(label?.textContent).toBe("Technology");
  expect(value?.textContent).toBe("3");
  expect(track?.getAttribute("aria-hidden")).toBe("true");
  expect(label?.parentElement === technology).toBe(embedded);
  expect(value?.parentElement === technology).toBe(embedded);
  expect(screen.getByText("No organization area recorded")).toBeTruthy();
  fireEvent.click(technology);
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

it("qualifies zero identified breaches when assessment coverage is incomplete", async () => {
  const partial = bundle();
  partial.items[0]!.value = 0;
  partial.items[0]!.completeness = "PARTIAL";
  partial.items[0]!.unknown = 2;
  const loadBreakdown = vi.fn().mockResolvedValue({
    source_id: partial.source_id,
    metric_id: "risks_outside_appetite",
    definition_revision: "enterprise-domain-v1",
    count: 0,
    items: [],
  });

  render(<OrganizationRiskSummary bundle={partial} loadBreakdown={loadBreakdown}/>);

  expect(await screen.findByRole("heading", { name: "No identified appetite breaches" })).toBeTruthy();
  expect(screen.getByText("Risk coverage is incomplete; some assessments may be missing.")).toBeTruthy();
  expect(screen.queryByText("No current risks exceed approved appetite.")).toBeNull();
});

it("does not describe an empty assessed population as complete risk coverage", async () => {
  const unassessed = bundle();
  unassessed.items[0]!.value = 0;
  unassessed.items[0]!.population = 0;
  const loadBreakdown = vi.fn().mockResolvedValue({
    source_id: unassessed.source_id, metric_id: "risks_outside_appetite",
    definition_revision: "enterprise-domain-v1", count: 0, items: [],
  });

  render(<OrganizationRiskSummary bundle={unassessed} loadBreakdown={loadBreakdown}/>);

  expect(await screen.findByRole("heading", { name: "No identified appetite breaches" })).toBeTruthy();
  expect(screen.getByText("Risk coverage is incomplete; some assessments may be missing.")).toBeTruthy();
  expect(screen.queryByText("No current risks exceed approved appetite.")).toBeNull();
});

it.each([
  "removed bundle", "different source", "different definition", "different count", "different scope", "missing metric",
])("never commits the previous breakdown after %s", async (change) => {
  const initial = bundle();
  const breakdown: OrganizationMetricBreakdown = {
    source_id: initial.source_id, metric_id: initial.items[0]!.id,
    definition_revision: initial.items[0]!.definition_revision, count: 12,
    items: [{ key: "old-area", label: "Previous area", kind: "ORGANIZATION_SCOPE", value: 12 }],
  };
  const loadBreakdown = vi.fn().mockResolvedValueOnce(breakdown)
    .mockImplementation(() => new Promise<OrganizationMetricBreakdown>(() => {}));
  const commits: string[] = [];
  function Observed({ current, scope }: { current: DomainMetricBundle | null; scope?: string }) {
    const container = useRef<HTMLDivElement>(null);
    // Capture the committed DOM before passive effects can clear stale state.
    useLayoutEffect(() => { commits.push(container.current?.textContent ?? ""); }, [current, scope]);
    return <div ref={container}><OrganizationRiskSummary bundle={current} organizationScopeID={scope}
      loadBreakdown={loadBreakdown}/></div>;
  }

  const view = render(<Observed current={initial}/>);
  expect(await screen.findByText("Previous area")).toBeTruthy();
  const next = bundle();
  if (change === "different source") next.source_id = "source-next";
  if (change === "different definition") next.items[0]!.definition_revision = "domain-v2";
  if (change === "different count") next.items[0]!.value = 8;
  if (change === "missing metric") next.items = [];
  commits.length = 0;
  view.rerender(<Observed current={change === "removed bundle" ? null : next}
    scope={change === "different scope" ? "area-next" : undefined}/>);

  expect(commits.length).toBeGreaterThan(0);
  expect(commits.every((text) => !text.includes("Previous area"))).toBe(true);
  expect(screen.queryByText("Previous area")).toBeNull();
});

it("ignores an aborted slow breakdown after a newer scope has loaded", async () => {
  const source = bundle();
  const breakdown = (scope: string): OrganizationMetricBreakdown => ({
    source_id: source.source_id, metric_id: source.items[0]!.id,
    definition_revision: source.items[0]!.definition_revision, count: 12, scope_id: scope,
    items: [{ key: scope, label: scope, kind: "ORGANIZATION_SCOPE", value: 12 }],
  });
  let resolvePrevious!: (value: OrganizationMetricBreakdown) => void;
  const previous = new Promise<OrganizationMetricBreakdown>((resolve) => { resolvePrevious = resolve; });
  const loadBreakdown = vi.fn().mockReturnValueOnce(previous).mockResolvedValue(breakdown("Current area"));
  const view = render(<OrganizationRiskSummary bundle={source} organizationScopeID="Previous area"
    loadBreakdown={loadBreakdown}/>);
  const previousSignal = loadBreakdown.mock.calls[0]?.[4] as AbortSignal;

  view.rerender(<OrganizationRiskSummary bundle={source} organizationScopeID="Current area"
    loadBreakdown={loadBreakdown}/>);
  expect(previousSignal.aborted).toBe(true);
  expect(await screen.findByText("Current area")).toBeTruthy();
  await act(async () => { resolvePrevious(breakdown("Previous area")); });

  expect(screen.queryByText("Previous area")).toBeNull();
  expect(screen.getByText("Current area")).toBeTruthy();
});
