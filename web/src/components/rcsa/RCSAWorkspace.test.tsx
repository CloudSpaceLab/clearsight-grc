import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { RCSAWorkspace } from "./RCSAWorkspace";
import { getRCSACycle, listRCSACycles } from "../../rcsaApi";
import type { RCSACycleDetail } from "../../rcsaTypes";

vi.mock("../../rcsaApi", () => ({
  listRCSACycles: vi.fn(),
  getRCSACycle: vi.fn(),
}));

const cycle = {
  id: "cycle-1",
  tenant_id: "bank",
  legal_entity_id: "entity-a",
  code: "RCSA-Q3",
  name: "Q3 Technology RCSA",
  trigger_kind: "SCHEDULED" as const,
  first_line_owner_principal_id: "owner-1",
  status: "AWAITING_CHALLENGE" as const,
  population_checksum: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  first_line_distribution_id: "distribution-1",
  first_line_response_revision_id: "response-1",
  challenge_matter_id: "matter-1",
  version: 4,
  created_at: "2026-07-01T08:00:00Z",
  updated_at: "2026-10-05T08:00:00Z",
};

beforeEach(() => {
  vi.mocked(listRCSACycles).mockReset();
  vi.mocked(getRCSACycle).mockReset();
  vi.mocked(listRCSACycles).mockResolvedValue({
    complete: true,
    items: [{
      cycle,
      risk_count: 2,
      control_count: 3,
      first_line_owner_display_name: "Technology Risk Owner",
      handoff: { stage: "CHALLENGE", label: "Independent challenge in progress", target_type: "MATTER", target_id: "matter-1" },
    }],
  });
  vi.mocked(getRCSACycle).mockResolvedValue({
    cycle,
    complete: true,
    first_line_owner_display_name: "Technology Risk Owner",
    assessment_period_start: "2026-07-01T00:00:00Z",
    assessment_period_end: "2026-09-30T23:59:59Z",
    first_line_request_id: "request-1",
    handoff: { stage: "CHALLENGE", label: "Independent challenge in progress", target_type: "MATTER", target_id: "matter-1" },
    phase: { stage: "INDEPENDENT_CHALLENGE", label: "Independent challenge", detail: "Independent review of the first-line assessment is in progress." },
    risks: [
      { cycle_id: cycle.id, risk_id: "risk-1", risk_version: 3, code: "TECH-01", name: "Service interruption", category: "Technology" },
      { cycle_id: cycle.id, risk_id: "risk-2", risk_version: 2, code: "TECH-02", name: "Privileged access misuse", category: "Cyber" },
    ],
    controls: [{
      cycle_id: cycle.id, risk_id: "risk-1", risk_version: 3, risk_control_link_id: "link-1",
      definition_id: "control-1", definition_code: "BCP-01", definition_name: "Service recovery test",
      program_id: "program-1", implementation_id: "implementation-1", implementation_version: 2,
      implementation_name: "Quarterly recovery exercise", implementation_status: "IMPLEMENTED",
    }],
  });
});

it("renders assessment cards with frozen population facts and opens the exact challenge handoff", async () => {
  const onTarget = vi.fn();
  const onOpenMatter = vi.fn();
  const { rerender } = render(<RCSAWorkspace
    organizationName="Meridian Trust Bank"
    legalEntityName="Nigeria"
    embedded
    onTarget={onTarget}
    onOpenMatter={onOpenMatter}
  />);

  const list = await screen.findByRole("list", { name: "RCSA cycles" });
  expect(screen.getByRole("heading", { name: "Risk & control self-assessments", level: 2 })).toBeTruthy();
  expect(screen.queryByRole("heading", { level: 1 })).toBeNull();
  expect(within(list).getByRole("heading", { name: "Q3 Technology RCSA", level: 3 })).toBeTruthy();
  const population = within(list).getByLabelText("Q3 Technology RCSA frozen population");
  expect(population.textContent).toBe("Risks2Controls3");
  expect(within(list).queryByRole("progressbar")).toBeNull();
  expect(screen.getByText("1 cycle shown")).toBeTruthy();
  fireEvent.click(screen.getByText("Cycle status"));
  expect(screen.getByRole("img", { name: /RCSA cycle status for displayed records:.*1 awaiting challenge or resolution/i })).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Review 1 awaiting challenge" }));
  await waitFor(() => expect(listRCSACycles).toHaveBeenCalledWith(expect.objectContaining({ status: "AWAITING_CHALLENGE" }), expect.any(AbortSignal)));

  fireEvent.click(screen.getByRole("button", { name: "Review cycle for Q3 Technology RCSA" }));
  expect(onTarget).toHaveBeenCalledWith("cycle-1");

  rerender(<RCSAWorkspace
    organizationName="Meridian Trust Bank"
    legalEntityName="Nigeria"
    embedded
    targetID="cycle-1"
    onTarget={onTarget}
    onOpenMatter={onOpenMatter}
  />);
  await screen.findByRole("dialog", { name: "Q3 Technology RCSA RCSA cycle" });
  expect(screen.getByText("1 Jul 2026 – 30 Sep 2026")).toBeTruthy();
  expect(screen.getByRole("table", { name: "RCSA frozen Risks" })).toBeTruthy();
  expect(screen.getByRole("table", { name: "RCSA frozen Controls" })).toBeTruthy();
  expect(screen.getByRole("region", { name: "RCSA cycle path" })).toBeTruthy();
  expect(screen.getByText("First-line collection")).toBeTruthy();
  expect(screen.getAllByText("Independent challenge").length).toBeGreaterThan(0);
  expect(screen.getByText("Risk acceptance")).toBeTruthy();
  expect(screen.getByText("Remediation verification")).toBeTruthy();

  fireEvent.click(screen.getByRole("button", { name: "Open challenge work" }));
  expect(onOpenMatter).toHaveBeenCalledWith("matter-1", "cycle-1");
});

it("opens the first-line evidence request with the originating cycle", async () => {
  const firstLineCycle = {
    ...cycle,
    status: "ASSESSMENT_OPEN" as const,
    first_line_response_revision_id: undefined,
    challenge_matter_id: undefined,
    version: 2,
  };
  vi.mocked(getRCSACycle).mockResolvedValueOnce({
    cycle: firstLineCycle,
    complete: true,
    first_line_owner_display_name: "Technology Risk Owner",
    first_line_request_id: "request-1",
    handoff: { stage: "FIRST_LINE", label: "First-line assessment in progress", target_type: "EVIDENCE_REQUEST", target_id: "request-1" },
    phase: { stage: "COLLECTION", label: "First-line collection", detail: "First-line assessment is in progress." },
    risks: [],
    controls: [],
  });
  const onOpenEvidence = vi.fn();

  render(<RCSAWorkspace
    organizationName="Meridian Trust Bank"
    legalEntityName="Nigeria"
    targetID="cycle-1"
    onTarget={vi.fn()}
    onOpenEvidence={onOpenEvidence}
  />);

  fireEvent.click(await screen.findByRole("button", { name: "Open first-line assessment" }));
  expect(onOpenEvidence).toHaveBeenCalledWith("request-1", "cycle-1");
});

it("shows partial authority resolution without inventing a complete population", async () => {
  vi.mocked(listRCSACycles).mockResolvedValueOnce({ complete: false, items: [] });
  render(<RCSAWorkspace organizationName="Bank" legalEntityName="Nigeria" onTarget={() => {}}/>);
  expect(await screen.findByText("Some cycles unavailable. Retry to confirm the full register.")).toBeTruthy();
  expect(screen.queryByText("No cycles in this view")).toBeNull();
  expect(screen.queryByText("No RCSA cycles are available to you in this legal entity.")).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Retry" }));
  expect(await screen.findByRole("list", { name: "RCSA cycles" })).toBeTruthy();
  expect(screen.queryByText("Some cycles unavailable. Retry to confirm the full register.")).toBeNull();
});

it("uses the server cursor and resets it when the cycle status changes", async () => {
  vi.mocked(listRCSACycles).mockResolvedValue({ complete: true, items: [], next_cursor: "cycle-page-2" });
  render(<RCSAWorkspace onTarget={vi.fn()}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Next page" }));
  await waitFor(() => expect(listRCSACycles).toHaveBeenCalledWith({ status: undefined, cursor: "cycle-page-2", limit: 25 }, expect.any(AbortSignal)));
  fireEvent.click(screen.getByRole("button", { name: /Status/ }));
  fireEvent.click(await screen.findByRole("option", { name: "Completed" }));
  await waitFor(() => expect(listRCSACycles).toHaveBeenLastCalledWith({ status: "COMPLETED", cursor: undefined, limit: 25 }, expect.any(AbortSignal)));
  expect(screen.queryByRole("button", { name: "Previous page" })).toBeNull();
});

it("clears the old cycle on route changes and ignores a late detail response", async () => {
  const second = deferred<RCSACycleDetail>();
  vi.mocked(getRCSACycle).mockImplementation((id) => id === "cycle-2" ? second.promise : Promise.resolve(detailFor(id)));
  const props = { onTarget: vi.fn() };
  const { rerender } = render(<RCSAWorkspace {...props} targetID="cycle-1"/>);
  await screen.findByRole("dialog", { name: "Assessment cycle-1 RCSA cycle" });

  rerender(<RCSAWorkspace {...props} targetID="cycle-2"/>);
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(screen.getByText("Loading RCSA cycle…")).toBeTruthy();
  rerender(<RCSAWorkspace {...props} targetID="cycle-3"/>);
  await screen.findByRole("dialog", { name: "Assessment cycle-3 RCSA cycle" });
  await act(async () => { second.resolve(detailFor("cycle-2")); });
  expect(screen.getByRole("dialog", { name: "Assessment cycle-3 RCSA cycle" })).toBeTruthy();
  expect(screen.queryByRole("dialog", { name: "Assessment cycle-2 RCSA cycle" })).toBeNull();

  rerender(<RCSAWorkspace {...props}/>);
  expect(screen.queryByRole("dialog")).toBeNull();
});

it("does not display a different cycle returned for the selected route", async () => {
  vi.mocked(getRCSACycle).mockResolvedValue(detailFor("cycle-other"));
  render(<RCSAWorkspace targetID="cycle-1" onTarget={vi.fn()}/>);
  expect(await screen.findByText("This RCSA cycle could not be loaded.")).toBeTruthy();
  expect(screen.queryByRole("dialog")).toBeNull();
});

function detailFor(id: string): RCSACycleDetail {
  return {
    cycle: { ...cycle, id, name: `Assessment ${id}` }, complete: true, risks: [], controls: [],
    handoff: { stage: "CHALLENGE", label: "Independent challenge in progress" },
    phase: { stage: "INDEPENDENT_CHALLENGE", label: "Independent challenge", detail: "Independent review is in progress." },
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((accept) => { resolve = accept; });
  return { promise, resolve };
}
