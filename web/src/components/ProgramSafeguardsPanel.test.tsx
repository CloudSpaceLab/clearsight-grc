import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import {
  addProgramControlImplementation,
  assignProgramControlImplementation,
  reviseProgramControlImplementation,
  retireProgramRequirementControlLink,
  transitionProgramControlImplementation,
} from "../programOperationsApi";
import type { ProgramAggregate } from "../types";
import { ProgramSafeguardsPanel } from "./ProgramSafeguardsPanel";

vi.mock("../programOperationsApi", () => ({
  addProgramControlImplementation: vi.fn(),
  addProgramControlObjective: vi.fn(),
  linkProgramRequirementControl: vi.fn(),
  reviseProgramControlImplementation: vi.fn(),
  assignProgramControlImplementation: vi.fn(),
  transitionProgramControlImplementation: vi.fn(),
  retireProgramRequirementControlLink: vi.fn(),
}));

beforeEach(() => vi.clearAllMocks());

it("shows the stored safeguard owner label without exposing its principal ID", () => {
  const aggregate = {
    program: { id: "program-1", version: 4 },
    requirements: [],
    control_objectives: [{ id: "objective-1", code: "OBJ-1", name: "Reliable filing", outcome: "Returns are filed on time.", status: "ACTIVE" }],
    control_implementations: [{ id: "safeguard-1", objective_id: "objective-1", name: "Annual return checklist", description: "Confirm every filing section.", owner_principal_id: "owner-private", status: "IMPLEMENTED" }],
    requirement_control_links: [],
  } as unknown as ProgramAggregate;
  render(<ProgramSafeguardsPanel aggregate={aggregate} operations={[]} responsibleParties={[{ scope: "SAFEGUARD", subresource_id: "safeguard-1", responsibility: "PERFORMER", display_name: "Ada Okafor", kind: "PERSON" }]} onUpdated={vi.fn()} onReload={vi.fn()}/>);

  expect(screen.getByText(/Ada Okafor · Implemented/)).toBeTruthy();
  expect(screen.queryByText(/owner-private/)).toBeNull();
});

it("removes a current requirement coverage link through its exact governed operation", async () => {
  const aggregate = {
    program: { id: "program-1", version: 8 },
    requirements: [{ id: "requirement-1", code: "CAR-01", title: "File the annual return", status: "APPROVED" }],
    control_objectives: [{ id: "objective-1", code: "OBJ-1", name: "Reliable filing", outcome: "Returns are filed on time.", status: "ACTIVE" }],
    control_implementations: [{ id: "safeguard-1", objective_id: "objective-1", name: "Annual return checklist", description: "Confirm every filing section.", status: "IMPLEMENTED" }],
    requirement_control_links: [{ id: "link-1", requirement_id: "requirement-1", implementation_id: "safeguard-1" }],
  } as unknown as ProgramAggregate;
  const updated = { ...aggregate, program: { ...aggregate.program, version: 9 }, requirement_control_links: [] };
  vi.mocked(retireProgramRequirementControlLink).mockResolvedValue(updated);
  const onUpdated = vi.fn();
  render(<ProgramSafeguardsPanel aggregate={aggregate} operations={[{
    command: "program.safeguard.unlink", subresource_id: "link-1", label: "Remove coverage link",
    responsibility: "OWNER", can_act: true, reason: "You own this Program.",
  }]} onUpdated={onUpdated} onReload={vi.fn()}/>);

  expect(screen.getByText("File the annual return → Annual return checklist")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Remove File the annual return coverage link" }));
  fireEvent.change(screen.getByLabelText("Reason for removing this coverage link"), { target: { value: "The replacement safeguard now provides this coverage." } });
  fireEvent.click(screen.getByRole("button", { name: "Remove coverage link" }));

  expect(retireProgramRequirementControlLink).toHaveBeenCalledWith("program-1", "link-1", 8, "The replacement safeguard now provides this coverage.");
  await waitFor(() => expect(onUpdated).toHaveBeenCalledWith(updated));
});

it("creates safeguards as planned work and exposes governed maintenance actions", async () => {
  const aggregate = {
    program: { id: "program-1", version: 4 }, requirements: [],
    control_objectives: [{ id: "objective-1", code: "OBJ-1", name: "Reliable filing", outcome: "Returns are filed on time.", status: "ACTIVE" }],
    control_implementations: [{ id: "safeguard-1", objective_id: "objective-1", name: "Annual return checklist", description: "Confirm every filing section.", implementation_type: "CHECKLIST", owner_principal_id: "owner-1", scope: { description: "Annual return" }, status: "PLANNED", effective_from: "2026-08-01T00:00:00Z", version: 1 }],
    requirement_control_links: [],
  } as unknown as ProgramAggregate;
  const operations = [
    { command: "program.safeguard.define", label: "Define safeguards", responsibility: "OWNER", can_act: true, reason: "You can define safeguards.", candidates: [{ id: "owner-1", display_name: "Ada Okafor", kind: "PERSON", role: "Control owner" }] },
    { command: "program.safeguard.update", subresource_id: "safeguard-1", label: "Edit safeguard", responsibility: "OWNER", can_act: true, reason: "You can edit this safeguard.", assigned_to: { id: "program-owner", display_name: "Program Owner", kind: "PERSON", role: "Program owner" } },
    { command: "program.safeguard.assign", subresource_id: "safeguard-1", label: "Change owner", responsibility: "OWNER", can_act: true, reason: "You can assign this safeguard.", candidates: [{ id: "owner-2", display_name: "Chidi Bello", kind: "PERSON", role: "Control owner" }] },
    { command: "program.safeguard.transition", subresource_id: "safeguard-1", label: "Change status", responsibility: "PERFORMER", can_act: true, reason: "You operate this safeguard.", allowed_targets: ["IN_PROGRESS", "RETIRED"] },
  ];
  vi.mocked(addProgramControlImplementation).mockResolvedValue(aggregate);
  vi.mocked(reviseProgramControlImplementation).mockResolvedValue(aggregate);
  vi.mocked(assignProgramControlImplementation).mockResolvedValue(aggregate);
  vi.mocked(transitionProgramControlImplementation).mockResolvedValue(aggregate);
  render(<ProgramSafeguardsPanel aggregate={aggregate} operations={operations} onUpdated={vi.fn()} onReload={vi.fn()}/>);

  fireEvent.click(screen.getByRole("button", { name: "Add safeguard" }));
  fireEvent.change(screen.getByLabelText("Safeguard name"), { target: { value: "Filing review" } });
  fireEvent.change(screen.getByLabelText("How the safeguard works"), { target: { value: "Review every filing section." } });
  fireEvent.click(screen.getByRole("button", { name: "Save safeguard" }));
  expect(addProgramControlImplementation).toHaveBeenCalledWith("program-1", 4, expect.objectContaining({ status: "PLANNED" }));

  expect(screen.getByRole("button", { name: "Edit Annual return checklist" })).toBeTruthy();
  expect(screen.getByRole("button", { name: "Change Annual return checklist owner" })).toBeTruthy();
  expect(screen.getByRole("button", { name: "Change Annual return checklist status" })).toBeTruthy();
});
