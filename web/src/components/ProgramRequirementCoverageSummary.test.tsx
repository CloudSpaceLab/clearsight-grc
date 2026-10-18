import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ProgramAggregate } from "../types";
import { ProgramRequirementCoverageSummary } from "./ProgramRequirementCoverageSummary";

const current: ProgramAggregate = {
  program: {
    id: "program-1", tenant_id: "tenant-1", code: "P1", name: "Security", version: 4,
    calculation_policy: {last_revision: 1, active: {
      number: 1, checksum: "approved", maker_id: "owner", proposed_at: "2026-10-01T00:00:00Z",
      approver_id: "reviewer", approved_at: "2026-10-02T00:00:00Z",
      definition: {revision: "v1", minimum_coverage_bps: 6000,
        categories: [{code: "REQUIREMENTS", weight_bps: 10000}]},
    }},
  },
  current_state: {
    id: "state-1", generated_at: "2026-10-12T12:00:00Z", program_version: 4, projection_version: 3,
    dimensions: {}, reasons: [], open_matter_count: 0,
    calculation: {
      state: "PROVISIONAL", policy_revision: "v1", coverage_bps: 5000,
      eligible_count: 4, assessed_count: 2, unknown_count: 1, excluded_count: 0,
      categories: [{
        code: "REQUIREMENTS", weight_bps: 10000, coverage_bps: 5000,
        eligible_count: 4, assessed_count: 2, unknown_count: 1, excluded_count: 0,
      }],
    },
  },
  requirements: [], applicability: [], evidence_contracts: [], evidence_assessments: [],
  control_objectives: [], control_implementations: [], requirement_control_links: [],
  triggers: [], state_label: "Evidence incomplete",
} as unknown as ProgramAggregate;

describe("approved Program Requirement coverage", () => {
  it("shows a real assessed/eligible denominator and distinguishes unknown values", () => {
    const onNavigate = vi.fn();
    render(<ProgramRequirementCoverageSummary aggregate={current} onNavigate={onNavigate}/>);
    expect(screen.getByRole("heading", {name: "Requirement coverage"})).toBeTruthy();
    expect(screen.getByText("2 of 4")).toBeTruthy();
    expect(screen.getByText("Coverage incomplete")).toBeTruthy();
    const progress = screen.getByRole("progressbar", {name: "Assessed requirement coverage"});
    expect(progress.getAttribute("value")).toBe("2");
    expect(progress.getAttribute("max")).toBe("4");
    expect(screen.getByText("Support and gap breakdowns remain unavailable without separately verified assessments.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", {name: "View requirements"}));
    expect(onNavigate).toHaveBeenCalledWith("requirements-controls");
  });

  it("does not fabricate coverage without an active approved calculation", () => {
    render(<ProgramRequirementCoverageSummary aggregate={{
      ...current,
      program: {...current.program, calculation_policy: undefined},
    }} onNavigate={vi.fn()}/>);
    expect(screen.getByText("Coverage not assessed")).toBeTruthy();
    expect(screen.queryByRole("progressbar")).toBeNull();
    expect(screen.queryByText("2 of 4")).toBeNull();
  });

  it("labels an old Program projection as out-of-date without presenting it as current", () => {
    render(<ProgramRequirementCoverageSummary aggregate={{
      ...current, program: {...current.program, version: 5},
    }} onNavigate={vi.fn()}/>);
    expect(screen.getByText("Out-of-date assessment")).toBeTruthy();
    expect(screen.getByText("2 of 4")).toBeTruthy();
  });
});
