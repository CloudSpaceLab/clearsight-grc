import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { loadProgram, loadProgramSummaries } from "../api";
import { ProgramsWorkspace } from "./ProgramsWorkspace";

vi.mock("../api", () => ({
  loadProgram: vi.fn(),
  loadProgramSummaries: vi.fn(),
}));

describe("Program projection truth", () => {
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

    expect(await screen.findByRole("heading", { name: "1 loaded program is still being set up or reassessed" })).toBeTruthy();
    expect(screen.getByText("Updating status")).toBeTruthy();
    expect(screen.getByText("Last assessed at version 7; Program is version 9.")).toBeTruthy();
    expect(screen.getByText("0 current")).toBeTruthy();
  });
});
