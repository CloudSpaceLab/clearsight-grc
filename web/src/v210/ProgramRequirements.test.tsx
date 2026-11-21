import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { ProgramRequirements } from "./ProgramRequirements";
import type { ProgramAggregate } from "../types";

const api = vi.hoisted(() => ({ loadProgramOperations: vi.fn(), addProgramRequirement: vi.fn(), determineProgramApplicability: vi.fn(), supersedeProgramRequirement: vi.fn() }));
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

it("requires and saves the reason for a not-applicable decision", async () => {
  api.determineProgramApplicability.mockResolvedValue({ ...aggregate, program: { id: "p1", version: 5 }, applicability: [{ id: "a1", requirement_id: "r1", status: "NOT_APPLICABLE" }] });
  render(<ProgramRequirements aggregate={aggregate} onUpdated={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Set applicability for Review access" }));
  fireEvent.change(screen.getByLabelText("Does this apply?"), { target: { value: "NOT_APPLICABLE" } });
  expect((screen.getByRole("button", { name: "Save applicability" }) as HTMLButtonElement).disabled).toBe(true);
  fireEvent.change(screen.getByLabelText("Why is this not applicable?"), { target: { value: "The service has no privileged accounts." } });
  fireEvent.click(screen.getByRole("button", { name: "Save applicability" }));
  expect(await screen.findByText("Requirement updated.")).toBeTruthy();
  expect(api.determineProgramApplicability).toHaveBeenCalledWith("p1", 4, expect.objectContaining({ requirementID: "r1", status: "NOT_APPLICABLE", rationale: "The service has no privileged accounts." }));
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

it("adds a source-anchored requirement with the actual obligated party and task", async () => {
  api.loadProgramOperations.mockResolvedValue({ program_id: "p1", program_version: 4, authority_available: true, operations: [{ command: "program.requirement.add", can_act: true }] });
  const created = { ...requirement, id: "r2", code: "DELIVERY", title: "Retain records", statement: "The carrier should retain delivery records.", source_anchor: "Contract section 4", modality: "SHOULD", actor: "The carrier", action: "retain", object: "delivery records", status: "DRAFT" };
  api.addProgramRequirement.mockImplementation(async (_id: string, _version: number, input: { effectiveFrom: string }) => ({ ...aggregate, program: { id: "p1", version: 5 }, requirements: [requirement, { ...created, effective_from: input.effectiveFrom }] }));
  const updated = vi.fn(); render(<ProgramRequirements aggregate={aggregate} onUpdated={updated}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Add requirement" }));
  for (const label of ["Who must act?", "What must they do?", "What does it apply to?"]) expect((screen.getByLabelText(label) as HTMLInputElement).value).toBe("");
  expect((screen.getByLabelText("Obligation strength") as HTMLSelectElement).value).toBe("");
  fireEvent.change(screen.getByLabelText("Requirement code"), { target: { value: "DELIVERY" } });
  fireEvent.change(screen.getByLabelText("Requirement title"), { target: { value: "Retain records" } });
  fireEvent.change(screen.getByLabelText("Requirement text"), { target: { value: "The carrier should retain delivery records." } });
  fireEvent.change(screen.getByLabelText("Source and section"), { target: { value: "Contract section 4" } });
  fireEvent.change(screen.getByLabelText("Obligation strength"), { target: { value: "SHOULD" } });
  fireEvent.change(screen.getByLabelText("Who must act?"), { target: { value: "The carrier" } });
  fireEvent.change(screen.getByLabelText("What must they do?"), { target: { value: "retain" } });
  fireEvent.change(screen.getByLabelText("What does it apply to?"), { target: { value: "delivery records" } });
  fireEvent.click(screen.getByRole("button", { name: "Save requirement" }));
  expect(await screen.findByText("Requirement draft saved. Review required.")).toBeTruthy();
  expect(api.addProgramRequirement).toHaveBeenCalledWith("p1", 4, expect.objectContaining({ code: "DELIVERY", title: "Retain records", sourceAnchor: "Contract section 4", modality: "SHOULD", actor: "The carrier", action: "retain", object: "delivery records" }));
  expect(updated).toHaveBeenCalledTimes(1);
});

it("does not offer requirement creation without current authority", async () => {
  api.loadProgramOperations.mockResolvedValue({ program_id: "p1", program_version: 4, authority_available: true, operations: [{ command: "program.requirement.add", can_act: false }] });
  render(<ProgramRequirements aggregate={aggregate} onUpdated={() => {}}/>);
  await screen.findByText("Review privileged access.");
  expect(screen.queryByRole("button", { name: "Add requirement" })).toBeNull();
});

it("retains a not-applicable reason only while that decision is selected", async () => {
  render(<ProgramRequirements aggregate={aggregate} onUpdated={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Set applicability for Review access" }));
  fireEvent.change(screen.getByLabelText("Does this apply?"), { target: { value: "NOT_APPLICABLE" } });
  fireEvent.change(screen.getByLabelText("Why is this not applicable?"), { target: { value: "No privileged accounts" } });
  fireEvent.change(screen.getByLabelText("Does this apply?"), { target: { value: "APPLICABLE" } });
  expect(screen.queryByLabelText("Why is this not applicable?")).toBeNull();
  fireEvent.change(screen.getByLabelText("Does this apply?"), { target: { value: "NOT_APPLICABLE" } });
  expect((screen.getByLabelText("Why is this not applicable?") as HTMLTextAreaElement).value).toBe("");
  fireEvent.change(screen.getByLabelText("Does this apply?"), { target: { value: "PARTIALLY_APPLICABLE" } });
  expect((screen.getByLabelText("Reason for decision") as HTMLTextAreaElement).required).toBe(true);
});

it("returns focus to Add requirement after cancelling", async () => {
  api.loadProgramOperations.mockResolvedValue({ program_id: "p1", program_version: 4, authority_available: true, operations: [{ command: "program.requirement.add", can_act: true }] });
  render(<ProgramRequirements aggregate={aggregate} onUpdated={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Add requirement" }));
  expect(document.activeElement).toBe(screen.getByLabelText("Requirement title"));
  fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  expect(document.activeElement).toBe(screen.getByRole("button", { name: "Add requirement" }));
});

async function fillNewRequirement() {
  fireEvent.click(await screen.findByRole("button", { name: "Add requirement" }));
  for (const [label, value] of [["Requirement code", "D1"], ["Requirement title", "Keep records"], ["Requirement text", "The carrier should retain records."], ["Source and section", "Contract 4"], ["Obligation strength", "SHOULD"], ["Who must act?", "The carrier"], ["What must they do?", "retain"], ["What does it apply to?", "records"]]) fireEvent.change(screen.getByLabelText(label!), { target: { value } });
}

it("does not confirm or repeat an unverified requirement receipt", async () => {
  api.loadProgramOperations.mockResolvedValue({ program_id: "p1", program_version: 4, authority_available: true, operations: [{ command: "program.requirement.add", can_act: true }] });
  api.addProgramRequirement.mockResolvedValue({ ...aggregate, program: { id: "p1", version: 5 } });
  const updated = vi.fn(); render(<ProgramRequirements aggregate={aggregate} onUpdated={updated}/>);
  await fillNewRequirement();
  fireEvent.click(screen.getByRole("button", { name: "Save requirement" }));
  expect(await screen.findByText(/Requirement status unknown/)).toBeTruthy();
  expect(updated).not.toHaveBeenCalled();
  expect(screen.queryByRole("button", { name: "Add requirement" })).toBeNull();
  expect(api.addProgramRequirement).toHaveBeenCalledTimes(1);
});

it("does not confirm a requirement saved with a different effective date", async () => {
  api.loadProgramOperations.mockResolvedValue({ program_id: "p1", program_version: 4, authority_available: true, operations: [{ command: "program.requirement.add", can_act: true }] });
  api.addProgramRequirement.mockResolvedValue({ ...aggregate, program: { id: "p1", version: 5 }, requirements: [...aggregate.requirements, { ...requirement, id: "r2", code: "D1", title: "Keep records", statement: "The carrier should retain records.", source_anchor: "Contract 4", modality: "SHOULD", actor: "The carrier", action: "retain", object: "records", effective_from: "2000-01-01T00:00:00Z" }] });
  const updated = vi.fn(); render(<ProgramRequirements aggregate={aggregate} onUpdated={updated}/>);
  await fillNewRequirement();
  fireEvent.click(screen.getByRole("button", { name: "Save requirement" }));
  expect(await screen.findByText(/Requirement status unknown/)).toBeTruthy();
  expect(updated).not.toHaveBeenCalled();
});

it("identifies draft requirements as awaiting independent review", () => {
  render(<ProgramRequirements aggregate={{ ...aggregate, requirements: [{ ...requirement, status: "DRAFT" }] }} onUpdated={() => {}}/>);
  expect(screen.getByText("Draft · Review required")).toBeTruthy();
  expect(screen.queryByRole("button", { name: /Set applicability/ })).toBeNull();
});

it("ignores a requirement receipt after the selected Program changes", async () => {
  api.loadProgramOperations.mockImplementation(async (id: string) => ({ program_id: id, program_version: 4, authority_available: true, operations: [{ command: "program.requirement.add", can_act: true }] }));
  let finish!: (value: ProgramAggregate) => void;
  api.addProgramRequirement.mockReturnValue(new Promise<ProgramAggregate>((resolve) => { finish = resolve; }));
  const updated = vi.fn(); const view = render(<ProgramRequirements aggregate={aggregate} onUpdated={updated}/>);
  await fillNewRequirement();
  fireEvent.click(screen.getByRole("button", { name: "Save requirement" }));
  view.rerender(<ProgramRequirements aggregate={{ ...aggregate, program: { ...aggregate.program, id: "p2" } }} onUpdated={updated}/>);
  finish({ ...aggregate, program: { ...aggregate.program, version: 5 }, requirements: [...aggregate.requirements, { ...requirement, id: "r2", code: "D1", title: "Keep records", statement: "The carrier should retain records.", source_anchor: "Contract 4", modality: "SHOULD", actor: "The carrier", action: "retain", object: "records" }] });
  await screen.findByRole("button", { name: "Add requirement" });
  expect(updated).not.toHaveBeenCalled();
  expect(screen.queryByText("Requirement updated.")).toBeNull();
});
