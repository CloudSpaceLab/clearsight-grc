import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { RCSAWorkspace } from "./RCSAWorkspace";
import { getRCSACycle, listRCSACycles } from "../../rcsaApi";

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

it("renders a restrained cycle register and opens the exact challenge handoff", async () => {
  const onTarget = vi.fn();
  const onOpenMatter = vi.fn();
  const { rerender } = render(<RCSAWorkspace
    organizationName="Meridian Trust Bank"
    legalEntityName="Nigeria"
    onTarget={onTarget}
    onOpenMatter={onOpenMatter}
  />);

  await screen.findByRole("table", { name: "RCSA cycles" });
  expect(screen.getByText("Q3 Technology RCSA")).toBeTruthy();
  expect(screen.getByText("2 Risks")).toBeTruthy();
  expect(screen.getByText("3 controls")).toBeTruthy();
  expect(screen.getByRole("img", { name: /RCSA cycle status for displayed records:.*1 awaiting challenge or resolution/i })).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Review 1 awaiting challenge" }));
  await waitFor(() => expect(listRCSACycles).toHaveBeenCalledWith(expect.objectContaining({ status: "AWAITING_CHALLENGE" }), expect.any(AbortSignal)));

  fireEvent.click(screen.getByRole("button", { name: "Review cycle for Q3 Technology RCSA" }));
  expect(onTarget).toHaveBeenCalledWith("cycle-1");

  rerender(<RCSAWorkspace
    organizationName="Meridian Trust Bank"
    legalEntityName="Nigeria"
    targetID="cycle-1"
    onTarget={onTarget}
    onOpenMatter={onOpenMatter}
  />);
  await screen.findByRole("heading", { name: "Q3 Technology RCSA" });
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
  await waitFor(() => expect(listRCSACycles).toHaveBeenCalled());
  expect(screen.getByText("Some cycles are temporarily unavailable. Available cycles remain unchanged.")).toBeTruthy();
});
