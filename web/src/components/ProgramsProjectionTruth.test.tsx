import { render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { loadProgram, loadProgramSummaries } from "../api";
import { ProgramsWorkspace } from "./ProgramsWorkspace";
import type { ProgramSummary } from "../summaryTypes";

vi.mock("../api", () => ({
  loadProgram: vi.fn(),
  loadProgramSummaries: vi.fn(),
}));

describe("Program projection truth", () => {
  it("keeps absent assessment and issue counts unknown even if the stale flag is absent", async () => {
    vi.mocked(loadProgramSummaries).mockResolvedValue({ generated_at: "2026-09-09T00:00:00Z", items: [{ program: { id: "unknown", name: "Delivery records", code: "DELIVERY", version: 2, status: "ACTIVE" }, overall_state: "CURRENT", state_label: "Up to date", program_version: 2, reasons: [], requirement_count: 1, evidence_check_count: 0, open_matter_count: 0 } as unknown as ProgramSummary] });
    render(<ProgramsWorkspace/>);
    expect(await screen.findByText("Unknown", { selector: ".program-state strong" })).toBeTruthy();
    expect(screen.getByText("Unknown", { selector: ".program-state strong" })).toBeTruthy();
    expect(screen.getByText("Unknown", { selector: ".program-counts b" })).toBeTruthy();
    expect(screen.queryByText("Up to date", { selector: ".program-state strong" })).toBeNull();
  });
  it("does not present a stale last-known CURRENT snapshot as current", async () => {
    vi.mocked(loadProgramSummaries).mockResolvedValue({
      generated_at: "2026-08-07T14:00:00Z",
      items: [{
        program: {
          id: "program-stale",
          tenant_id: "bank-demo",
          code: "PRIVACY",
          name: "Privacy compliance",
          type: "REGULATORY",
          status: "ACTIVE",
          owning_function: "Compliance",
          scope: {},
          effective_from: "2026-01-01T00:00:00Z",
          created_at: "2026-01-01T00:00:00Z",
          updated_at: "2026-08-07T14:00:00Z",
          version: 9,
        },
        state_label: "Up to date",
        overall_state: "CURRENT",
        reasons: [],
        reasons_total: 0,
        reasons_omitted: 0,
        open_matter_count: 0,
        requirement_count: 12,
        safeguard_count: 8,
        evidence_check_count: 6,
        program_version: 9,
        assessed_program_version: 7,
        projection_version: 14,
        projection_stale: true,
        state_generated_at: "2026-08-07T13:55:00Z",
      }],
    });
    vi.mocked(loadProgram).mockRejectedValue(new Error("not opened"));

    render(<ProgramsWorkspace/>);

    expect(await screen.findByRole("heading", { name: "1 loaded program needs setup, review or a current assessment" })).toBeTruthy();
    expect(screen.getByText("Out of date")).toBeTruthy();
    const rowLink = screen.getByRole("link", { name: /Privacy compliance/ });
    expect(rowLink.getAttribute("href")).toBe("#programs/program-stale");
    expect(screen.getByRole("img", { name: /Loaded Program status:.*0 current/i })).toBeTruthy();
    expect(screen.getByRole("region", { name: "Loaded Program portfolio" })).toBeTruthy();
  });
  it("groups Program counts into distinct scannable facts without treating unknowns as zero", async () => {
    vi.mocked(loadProgramSummaries).mockResolvedValue({
      generated_at: "2026-10-09T12:00:00Z",
      items: [{
        program: {
          id: "program-privacy",
          name: "Data protection",
          code: "NDPA-2023",
          status: "ACTIVE",
          owning_function: "Data Protection Office",
          jurisdiction: "Nigeria",
          version: 5,
        },
        state_label: "Evidence incomplete",
        overall_state: "EVIDENCE_INSUFFICIENT",
        reasons: [],
        requirement_count: 6,
        evidence_check_count: 5,
        open_matter_count: 8,
        program_version: 5,
        assessed_program_version: 5,
        projection_stale: false,
      } as unknown as ProgramSummary],
    });

    render(<ProgramsWorkspace/>);
    const row = await screen.findByRole("link", { name: /Data protection/ });
    const counts = within(row).getByLabelText("Program indicators");
    expect(within(counts).getByText("6")).toBeTruthy();
    expect(within(counts).getByText("5")).toBeTruthy();
    expect(within(counts).getByText("8")).toBeTruthy();
    expect(within(counts).getByText("Open issues")).toBeTruthy();
    expect(within(counts).getByText("8").closest(".program-counts__attention")).toBeTruthy();
    expect(within(row).getByText("Supporting information needs review")).toBeTruthy();
    expect(within(row).getByText(/Open program/)).toBeTruthy();
    expect(screen.getByRole("img", { name: /1 follow-up, 0 current, 0 needs assessment/i })).toBeTruthy();
    expect(within(screen.getByLabelText("Open issues by Program in loaded current assessments")).getByText("Data protection")).toBeTruthy();
  });

});
