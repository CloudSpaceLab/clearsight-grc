import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { ProgramsRegister } from "./ProgramsRegister";

const api = vi.hoisted(() => ({ loadProgramSummaries: vi.fn() }));
vi.mock("../api", () => api);
vi.mock("./ProgramCreator", () => ({ ProgramCreateLink: () => null }));
const context = { tenant: { id: "t", name: "Bank" }, legal_entity: { id: "e", name: "Bank" }, actor: { id: "a", name: "Ada", kind: "PERSON" } } as never;
const current = { program: { id: "p1", name: "Access assurance", code: "P1", version: 2 }, program_version: 2, assessed_program_version: 2, projection_version: 1, projection_stale: false, state_generated_at: "2026-11-20T10:00:00Z", overall_state: "CURRENT", state_label: "Up to date", requirement_count: 4, evidence_check_count: 2, open_matter_count: 1 };
beforeEach(() => { vi.clearAllMocks(); window.history.replaceState(null, "", "#/programs"); });

it("shows loaded Program counts without presenting a partial page as the whole population", async () => {
  api.loadProgramSummaries.mockResolvedValue({ items: [current, { ...current, program: { ...current.program, id: "p2", name: "Third party" }, projection_stale: true, open_matter_count: 0 }], next_cursor: "next" });
  render(<ProgramsRegister context={context}/>);
  const counts = await screen.findByRole("region", { name: "Loaded Program summary" });
  expect(within(counts).getByText("2")).toBeTruthy();
  expect(within(counts).getByText("More programs available")).toBeTruthy();
  expect(within(counts).getAllByText("1")).toHaveLength(2);
});

it("shows unknown issue coverage instead of treating missing counts as zero", async () => {
  api.loadProgramSummaries.mockResolvedValue({ items: [{ ...current, open_matter_count: undefined }] });
  render(<ProgramsRegister context={context}/>);
  const counts = await screen.findByRole("region", { name: "Loaded Program summary" });
  expect(within(counts).getByText("Unknown")).toBeTruthy();
});

it("retains the server-side status filter", async () => {
  api.loadProgramSummaries.mockResolvedValue({ items: [] });
  render(<ProgramsRegister context={context}/>);
  fireEvent.change(await screen.findByLabelText("Assessed status"), { target: { value: "AT_RISK" } });
  fireEvent.click(screen.getByRole("button", { name: "Apply filters" }));
  await vi.waitFor(() => expect(api.loadProgramSummaries).toHaveBeenLastCalledWith(expect.objectContaining({ overallState: "AT_RISK" })));
});
