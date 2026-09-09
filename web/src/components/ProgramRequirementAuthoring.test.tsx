import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import type { ProgramAggregate } from "../types";
import { addProgramRequirement, createProgram, loadProgramSetupCandidates } from "../continuityCommands";
import { ProgramSetupWorkspace } from "./ProgramSetupWorkspace";
import { ProgramRequirementsPanel } from "./ProgramRequirementsPanel";

vi.mock("../continuityCommands", () => ({ addProgramRequirement: vi.fn(), createProgram: vi.fn(), loadProgramSetupCandidates: vi.fn() }));
vi.mock("../programOperationsApi", () => ({ loadProgramOperations: vi.fn().mockResolvedValue({ operations: [] }), addProgramRequirement: vi.fn(), supersedeProgramRequirement: vi.fn(), determineProgramApplicability: vi.fn() }));
vi.mock("./MonitoringSetup", () => ({ MonitoringSetup: () => null }));

const aggregate = { program: { id: "delivery", name: "Delivery records", version: 1 }, requirements: [], applicability: [] } as unknown as ProgramAggregate;

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(createProgram).mockResolvedValue(aggregate);
  vi.mocked(addProgramRequirement).mockResolvedValue(aggregate);
  vi.mocked(loadProgramSetupCandidates).mockResolvedValue({ owner_candidates: [{ id: "owner", display_name: "Owner", kind: "PERSON", role: "Owner" }], approval_authority_candidates: [{ id: "approver", display_name: "Approver", kind: "PERSON", role: "Approver" }], has_more: false, generated_at: "2026-09-09T00:00:00Z" });
});

it("captures the actual obligated party and task during nonbank Program setup", async () => {
  render(<ProgramSetupWorkspace actorPrincipalID="signed-in-person" canConfigureSources={false} onCreated={vi.fn()} onClose={vi.fn()}/>);
  await screen.findByRole("option", { name: "Owner · Owner" });
  fireEvent.change(screen.getByLabelText("Program name"), { target: { value: "Delivery records" } });
  fireEvent.change(screen.getByLabelText("Code"), { target: { value: "DELIVERY" } });
  fireEvent.change(screen.getByLabelText("Owning function"), { target: { value: "Logistics" } });
  fireEvent.change(screen.getByLabelText("Scope"), { target: { value: "Carrier delivery records" } });
  fireEvent.click(screen.getByRole("button", { name: "Create Program" }));
  await screen.findByRole("heading", { name: "Requirements" });
  for (const label of ["Who must act?", "What must they do?", "What does it apply to?"]) {
    expect((screen.getByLabelText(label, { exact: false }) as HTMLInputElement).value).toBe("");
    expect((screen.getByLabelText(label, { exact: false }) as HTMLInputElement).required).toBe(true);
  }
  fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Retain delivery records" } });
  fireEvent.change(screen.getByLabelText("Code"), { target: { value: "RETENTION" } });
  fireEvent.change(screen.getByLabelText("Requirement"), { target: { value: "The carrier should retain delivery records." } });
  fireEvent.click(screen.getByRole("button", { name: /Obligation strength/ }));
  fireEvent.click(await screen.findByRole("option", { name: "Should" }));
  fireEvent.change(screen.getByLabelText("Who must act?", { exact: false }), { target: { value: "The carrier" } });
  fireEvent.change(screen.getByLabelText("What must they do?", { exact: false }), { target: { value: "retain" } });
  fireEvent.change(screen.getByLabelText("What does it apply to?", { exact: false }), { target: { value: "delivery records" } });
  fireEvent.click(screen.getByRole("button", { name: "Add requirement" }));
  await waitFor(() => expect(addProgramRequirement).toHaveBeenCalledWith("delivery", 1, expect.objectContaining({ actor: "The carrier", action: "retain", object: "delivery records", modality: "SHOULD" })));
  expect((screen.getByLabelText("Who must act?", { exact: false }) as HTMLInputElement).value).toBe("");
});

it("starts required meaning fields empty and preserves an existing sector-specific obligation on replacement", () => {
  const requirement = { id: "existing", code: "EX", title: "Existing bank requirement", statement: "The bank must retain records.", actor: "The bank", action: "retain", object: "records", status: "APPROVED" };
  render(<ProgramRequirementsPanel aggregate={{ ...aggregate, requirements: [requirement] }} operations={[
    { command: "program.requirement.add", label: "Add", responsibility: "ACCOUNTABLE_OWNER", can_act: true, reason: "" },
    { command: "program.requirement.supersede", subresource_id: "existing", label: "Replace", responsibility: "ACCOUNTABLE_OWNER", can_act: true, reason: "" },
  ]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
  fireEvent.click(screen.getByRole("button", { name: "Add requirement" }));
  expect((screen.getByLabelText("Obligation strength") as HTMLSelectElement).value).toBe("");
  expect((screen.getByLabelText("Who must act?", { exact: false }) as HTMLInputElement).value).toBe("");
  for (const label of ["Who must act?", "What must they do?", "What does it apply to?"]) expect((screen.getByLabelText(label, { exact: false }) as HTMLInputElement).required).toBe(true);
  fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  fireEvent.click(screen.getByRole("button", { name: "Replace requirement" }));
  expect((screen.getByLabelText("Who must act?", { exact: false }) as HTMLInputElement).value).toBe("The bank");
});
