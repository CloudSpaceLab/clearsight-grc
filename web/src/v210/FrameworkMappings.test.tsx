import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { FrameworkMappings } from "./FrameworkMappings";
import type { ProgramAggregate } from "../types";

const api = vi.hoisted(() => ({ loadFrameworkClauseCandidates: vi.fn(), loadProgramFrameworkMappings: vi.fn(), proposeProgramFrameworkMapping: vi.fn(), decideProgramFrameworkMapping: vi.fn() }));
vi.mock("../programFrameworkCandidatesApi", () => api);
const aggregate = { program: { id: "p1", version: 4 }, framework_adoptions: [{ id: "a1", edition_id: "e1", framework_code: "SEC", edition_code: "2026", status: "ACTIVE" }], requirements: [{ id: "r1", code: "R1", title: "Review access" }] } as unknown as ProgramAggregate;
const operations = [{ command: "program.framework.mapping.propose", can_act: true, label: "Propose link", responsibility: "OWNER", reason: "" }];
beforeEach(() => {
  vi.resetAllMocks();
  api.loadFrameworkClauseCandidates.mockResolvedValue({ program_id: "p1", program_version: 4, adoption_id: "a1", edition_id: "e1", next_after_code: "", candidates: [{ clause_id: "c1", clause_code: "A1", source_anchor: "Section 1", status: "REVIEW_CANDIDATE", candidates: [{ requirement_id: "r1", requirement_code: "R1", title: "Review access" }] }] });
  api.loadProgramFrameworkMappings.mockResolvedValue({ adoption_id: "a1", program_version: 4, items: [], next_after_id: "" });
});

it("proposes a link from a verified suggested match using the current version", async () => {
  api.proposeProgramFrameworkMapping.mockResolvedValue({ program: { program: { id: "p1", version: 5 } }, mapping: { id: "m1", clause_id: "c1", requirement_id: "r1", adoption_id: "a1", status: "PENDING" } });
  render(<FrameworkMappings aggregate={aggregate} operations={operations} onUpdated={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Link A1 to Review access" }));
  fireEvent.change(screen.getByLabelText("Reason for link"), { target: { value: "Same control" } });
  fireEvent.click(screen.getByRole("button", { name: "Propose link" }));
  expect(await screen.findByText("Link proposed. Approval is required.")).toBeTruthy();
  expect(api.proposeProgramFrameworkMapping).toHaveBeenCalledWith("p1", "a1", 4, "c1", "r1", "EXACT", "Same control");
});

it("does not offer links when the program version differs", async () => {
  api.loadProgramFrameworkMappings.mockResolvedValue({ adoption_id: "a1", program_version: 5, items: [], next_after_id: "" });
  render(<FrameworkMappings aggregate={aggregate} operations={operations} onUpdated={() => {}}/>);
  expect(await screen.findByText("Program changed. Reload before linking requirements.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: /Link A1/ })).toBeNull();
});

it("does not offer a duplicate link while other links remain unloaded", async () => {
  api.loadProgramFrameworkMappings.mockResolvedValue({ adoption_id: "a1", program_version: 4, items: [], next_after_id: "cursor" });
  render(<FrameworkMappings aggregate={aggregate} operations={operations} onUpdated={() => {}}/>);
  expect(await screen.findByRole("button", { name: "Load more links" })).toBeTruthy();
  expect(screen.queryByRole("button", { name: /Link A1/ })).toBeNull();
});

it("allows only the authorized reviewer to approve a pending link", async () => {
  api.loadProgramFrameworkMappings.mockResolvedValue({ adoption_id: "a1", program_version: 4, items: [{ id: "m1", clause_id: "c1", clause_code: "A1", requirement_id: "r1", adoption_id: "a1", status: "PENDING" }], next_after_id: "" });
  api.decideProgramFrameworkMapping.mockResolvedValue({ program: { program: { id: "p1", version: 5 } }, mapping: { id: "m1", adoption_id: "a1", status: "APPROVED" } });
  const review = [{ command: "program.framework.mapping.approve", can_act: true, label: "Approve link", responsibility: "AUTHORIZER", reason: "" }];
  render(<FrameworkMappings aggregate={aggregate} operations={review} onUpdated={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Review link for A1" }));
  fireEvent.change(screen.getByLabelText("Review reason"), { target: { value: "Source confirmed" } });
  fireEvent.click(screen.getByRole("button", { name: "Approve link" }));
  expect(await screen.findByText("Link approved.")).toBeTruthy();
  expect(api.decideProgramFrameworkMapping).toHaveBeenCalledWith("p1", "a1", "m1", 4, "approve", "Source confirmed");
});

it("does not retry a link after an uncertain response", async () => {
  api.proposeProgramFrameworkMapping.mockRejectedValue(new Error("connection lost"));
  render(<FrameworkMappings aggregate={aggregate} operations={operations} onUpdated={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Link A1 to Review access" }));
  fireEvent.change(screen.getByLabelText("Reason for link"), { target: { value: "Same control" } });
  fireEvent.click(screen.getByRole("button", { name: "Propose link" }));
  expect(await screen.findByText(/Link status unknown/)).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Propose link" })).toBeNull();
});
