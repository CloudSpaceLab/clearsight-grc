import { fireEvent, render, screen, within } from "@testing-library/react";
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
    expect(await screen.findByText("No Program status calculation is available.")).toBeTruthy();
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
    expect(screen.getByText("Last assessed at version 7; Program is version 9.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Open Program" }));
    expect(window.location.hash).toBe("#programs/program-stale");
    const statusFacts = within(screen.getByLabelText("Loaded Program status"));
    expect(statusFacts.getByText((_, element) => element?.textContent?.trim() === "0 current")).toBeTruthy();
  });
});
