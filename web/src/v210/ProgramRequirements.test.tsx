import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { ProgramRequirements } from "./ProgramRequirements";
import type { ProgramAggregate } from "../types";

const api = vi.hoisted(() => ({ loadProgramOperations: vi.fn(), determineProgramApplicability: vi.fn(), supersedeProgramRequirement: vi.fn() }));
vi.mock("../programOperationsApi", () => api);
const requirement = { id: "r1", code: "R1", title: "Review access", statement: "Review privileged access.", status: "APPROVED", source_id: "s1", source_anchor: "Section 1", modality: "MUST", actor: "Owner", action: "Review", object: "Access", effective_from: "2025-01-01T00:00:00Z" };
const aggregate = { program: { id: "p1", version: 4 }, requirements: [requirement], applicability: [] } as unknown as ProgramAggregate;
beforeEach(() => { vi.resetAllMocks(); api.loadProgramOperations.mockResolvedValue({ program_id: "p1", program_version: 4, authority_available: true, operations: [{ command: "program.applicability.decide", can_act: true }, { command: "program.requirement.supersede", subresource_id: "r1", can_act: true }] }); });

it("reveals the not applicable reason only after that answer is selected", async () => {
  render(<ProgramRequirements aggregate={aggregate} onUpdated={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Set applicability for Review access" }));
  expect(screen.queryByLabelText("Why is this not applicable?")).toBeNull();
  fireEvent.change(screen.getByLabelText("Does this apply?"), { target: { value: "NOT_APPLICABLE" } });
  expect(screen.getByLabelText("Why is this not applicable?")).toBeTruthy();
  expect((screen.getByRole("button", { name: "Save applicability" }) as HTMLButtonElement).disabled).toBe(true);
  fireEvent.change(screen.getByLabelText("Does this apply?"), { target: { value: "APPLICABLE" } });
  expect(screen.queryByLabelText("Why is this not applicable?")).toBeNull();
});

it("records an applicable decision without a reason using the current program version", async () => {
  api.determineProgramApplicability.mockResolvedValue({ ...aggregate, program: { id: "p1", version: 5 }, applicability: [{ id: "a1", requirement_id: "r1", status: "APPLICABLE" }] });
  render(<ProgramRequirements aggregate={aggregate} onUpdated={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Set applicability for Review access" }));
  fireEvent.change(screen.getByLabelText("Does this apply?"), { target: { value: "APPLICABLE" } });
  fireEvent.click(screen.getByRole("button", { name: "Save applicability" }));
  expect(await screen.findByText("Requirement updated.")).toBeTruthy();
  expect(api.determineProgramApplicability).toHaveBeenCalledWith("p1", 4, expect.objectContaining({ requirementID: "r1", status: "APPLICABLE", rationale: "" }));
});

it("returns focus to the requirement action when cancelled", async () => {
  render(<ProgramRequirements aggregate={aggregate} onUpdated={() => {}}/>);
  const trigger = await screen.findByRole("button", { name: "Set applicability for Review access" });
  fireEvent.click(trigger);
  fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  expect(document.activeElement).toBe(screen.getByRole("button", { name: "Set applicability for Review access" }));
});

it("saves the edited wording as a new requirement version with source metadata", async () => {
  api.supersedeProgramRequirement.mockResolvedValue({ ...aggregate, program: { id: "p1", version: 5 }, requirements: [{ ...requirement, status: "SUPERSEDED" }, { ...requirement, id: "r2", title: "Review privileged access", status: "APPROVED" }] });
  const updated = vi.fn(); render(<ProgramRequirements aggregate={aggregate} onUpdated={updated}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Edit Review access" }));
  fireEvent.change(screen.getByLabelText("Requirement title"), { target: { value: "Review privileged access" } });
  fireEvent.change(screen.getByLabelText("Reason for change"), { target: { value: "Updated scope" } });
  fireEvent.change(screen.getByLabelText("Effective date"), { target: { value: "2026-01-01" } });
  fireEvent.click(screen.getByRole("button", { name: "Save requirement" }));
  expect(await screen.findByText("Requirement updated.")).toBeTruthy();
  expect(api.supersedeProgramRequirement).toHaveBeenCalledWith("p1", "r1", 4, expect.objectContaining({ title: "Review privileged access", sourceID: "s1", sourceAnchor: "Section 1", actor: "Owner", rationale: "Updated scope" }));
  expect(updated).toHaveBeenCalledTimes(1);
});

it("does not display edit controls for a stale authority response", async () => {
  api.loadProgramOperations.mockResolvedValue({ program_id: "p1", program_version: 3, authority_available: true, operations: [{ command: "program.applicability.decide", can_act: true }] });
  render(<ProgramRequirements aggregate={aggregate} onUpdated={() => {}}/>);
  expect(await screen.findByText("Program changed. Reload before editing requirements.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: /Set applicability/ })).toBeNull();
});
