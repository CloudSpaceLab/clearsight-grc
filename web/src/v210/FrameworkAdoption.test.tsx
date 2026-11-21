import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { FrameworkAdoption } from "./FrameworkAdoption";

const api = vi.hoisted(() => ({ loadProgramSummaries: vi.fn(), loadProgram: vi.fn() }));
const operations = vi.hoisted(() => ({ loadProgramOperations: vi.fn(), proposeProgramFrameworkAdoption: vi.fn(), decideProgramFrameworkAdoption: vi.fn() }));
vi.mock("../api", () => api);
vi.mock("../programOperationsApi", () => operations);
const edition = { id: "e1", publisher_code: "CBN", framework_code: "SEC", edition_code: "2026", title: "Security standard", jurisdiction: "GLOBAL", source_published_on: "2026-01-01", source_url: "https://example.org/standard", license_basis: "CITATIONS_ONLY" };
const program = { program: { id: "p1", name: "Security program", version: 4 }, framework_adoptions: [] };
beforeEach(() => {
  vi.resetAllMocks(); api.loadProgramSummaries.mockResolvedValue({ items: [{ program: { id: "p1", name: "Security program" } }] });
  api.loadProgram.mockResolvedValue(program);
  operations.loadProgramOperations.mockResolvedValue({ program_id: "p1", program_version: 4, authority_available: true, operations: [{ command: "program.framework.propose", can_act: true }] });
});

async function chooseProgram() {
  fireEvent.change(await screen.findByLabelText("Program"), { target: { value: "p1" } });
  await screen.findByLabelText("Reason for adoption");
}

it("proposes the selected published edition using the current program version", async () => {
  operations.proposeProgramFrameworkAdoption.mockResolvedValue({ program: { ...program.program, version: 5 }, framework_adoptions: [{ id: "a1", edition_id: "e1", status: "PENDING" }] });
  render(<FrameworkAdoption edition={edition} onCancel={() => {}}/>);
  await chooseProgram();
  fireEvent.change(screen.getByLabelText("Reason for adoption"), { target: { value: "Annual review" } });
  fireEvent.click(screen.getByRole("button", { name: "Propose adoption" }));
  expect(await screen.findByText("Adoption proposed. Approval is required.")).toBeTruthy();
  expect(operations.proposeProgramFrameworkAdoption).toHaveBeenCalledWith("p1", 4, "e1", "Annual review");
});

it("does not offer adoption when authority is missing or the program version changed", async () => {
  operations.loadProgramOperations.mockResolvedValue({ program_id: "p1", program_version: 5, authority_available: true, operations: [{ command: "program.framework.propose", can_act: true }] });
  render(<FrameworkAdoption edition={edition} onCancel={() => {}}/>);
  fireEvent.change(await screen.findByLabelText("Program"), { target: { value: "p1" } });
  expect(await screen.findByText("Program changed. Reload before proposing adoption.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Propose adoption" })).toBeNull();
});

it("does not repeat a proposal after an uncertain response", async () => {
  operations.proposeProgramFrameworkAdoption.mockRejectedValue(new Error("connection lost"));
  render(<FrameworkAdoption edition={edition} onCancel={() => {}}/>);
  await chooseProgram();
  fireEvent.change(screen.getByLabelText("Reason for adoption"), { target: { value: "Annual review" } });
  fireEvent.click(screen.getByRole("button", { name: "Propose adoption" }));
  expect(await screen.findByText("Adoption status unknown. Check the program before trying again.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Propose adoption" })).toBeNull();
});

it("allows an authorized reviewer to approve the exact pending edition", async () => {
  api.loadProgram.mockResolvedValue({ ...program, framework_adoptions: [{ id: "a1", edition_id: "e1", status: "PENDING", publisher_code: "CBN", framework_code: "SEC", edition_code: "2026", rationale: "Annual review", proposed_by: "maker", proposed_at: "2026-01-01" }] });
  operations.loadProgramOperations.mockResolvedValue({ program_id: "p1", program_version: 4, authority_available: true, operations: [{ command: "program.framework.approve", can_act: true }] });
  operations.decideProgramFrameworkAdoption.mockResolvedValue({ program: { ...program.program, version: 5 }, framework_adoptions: [{ id: "a1", edition_id: "e1", status: "ACTIVE" }] });
  render(<FrameworkAdoption edition={edition} onCancel={() => {}}/>);
  fireEvent.change(await screen.findByLabelText("Program"), { target: { value: "p1" } });
  fireEvent.change(await screen.findByLabelText("Review reason"), { target: { value: "Source verified" } });
  expect(screen.queryByText(/maker/)).toBeNull();
  expect(screen.getByText("Proposed 2026-01-01")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Approve adoption" }));
  expect(await screen.findByText("Adoption approved.")).toBeTruthy();
  expect(operations.decideProgramFrameworkAdoption).toHaveBeenCalledWith("p1", "a1", 4, "approve", "Source verified");
});

it("blocks a review when the pending edition identity does not match", async () => {
  api.loadProgram.mockResolvedValue({ ...program, framework_adoptions: [{ id: "a1", edition_id: "e1", status: "PENDING", publisher_code: "OTHER", framework_code: "SEC", edition_code: "2026", rationale: "Annual review", proposed_by: "maker", proposed_at: "2026-01-01" }] });
  operations.loadProgramOperations.mockResolvedValue({ program_id: "p1", program_version: 4, authority_available: true, operations: [{ command: "program.framework.approve", can_act: true }] });
  render(<FrameworkAdoption edition={edition} onCancel={() => {}}/>);
  fireEvent.change(await screen.findByLabelText("Program"), { target: { value: "p1" } });
  expect(await screen.findByText("Edition details do not match this proposal.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Approve adoption" })).toBeNull();
  expect(operations.decideProgramFrameworkAdoption).not.toHaveBeenCalled();
});
