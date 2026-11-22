import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { ProgramAggregate } from "../types";
import { ProgramCoverage } from "../v210/ProgramCoverage";

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
      eligible_count: 4, assessed_count: 2, unknown_count: 2, excluded_count: 0,
      categories: [{
        code: "REQUIREMENTS", weight_bps: 10000, coverage_bps: 5000,
        eligible_count: 4, assessed_count: 2, unknown_count: 2, excluded_count: 0,
      }],
    },
  },
  requirements: [], applicability: [], evidence_contracts: [], evidence_assessments: [],
  control_objectives: [], control_implementations: [], requirement_control_links: [],
  triggers: [], state_label: "Evidence incomplete",
} as unknown as ProgramAggregate;

describe("Program requirement coverage", () => {
  it("shows the assessed requirement count and percentage with the real denominator", () => {
    render(<ProgramCoverage aggregate={current}/>);
    expect(screen.getByRole("heading", {name: "Requirement coverage"})).toBeTruthy();
    expect(screen.getByText("2 of 4 requirements assessed")).toBeTruthy();
    expect(screen.getByText("50%")).toBeTruthy();
    expect(screen.getByText("2 unknown")).toBeTruthy();
    const progress = screen.getByRole("progressbar", {name: "Assessed requirements"});
    expect(progress.getAttribute("value")).toBe("2");
    expect(progress.getAttribute("max")).toBe("4");
    expect(screen.getByText("Control area results unavailable.")).toBeTruthy();
    expect(screen.queryByText(/supported 2/i)).toBeNull();
  });
  it("does not fabricate coverage without an approved calculation", () => {
    render(<ProgramCoverage aggregate={{ ...current, program: {...current.program, calculation_policy: undefined} }}/>);
    expect(screen.getByText("Coverage unavailable.")).toBeTruthy();
    expect(screen.queryByRole("progressbar")).toBeNull();
    expect(screen.queryByText("50%")).toBeNull();
  });
  it("marks an older Program snapshot as out of date", () => {
    render(<ProgramCoverage aggregate={{ ...current, program: {...current.program, version: 5} }}/>);
    expect(screen.getByText("Out of date")).toBeTruthy();
    expect(screen.getByText("2 of 4 requirements assessed")).toBeTruthy();
  });
  it("withholds a percentage for an empty or inconsistent denominator", () => {
    const calc = current.current_state!.calculation!;
    render(<ProgramCoverage aggregate={{ ...current, current_state: { ...current.current_state!, calculation: { ...calc, categories: [{ ...calc.categories[0]!, eligible_count: 0, assessed_count: 0 }] } } }}/>);
    expect(screen.getByText("Coverage unavailable.")).toBeTruthy();
    expect(screen.queryByRole("progressbar")).toBeNull();
  });
  it("uses requirement counts rather than totals across unrelated categories", () => {
    const calc = current.current_state!.calculation!;
    render(<ProgramCoverage aggregate={{ ...current, current_state: { ...current.current_state!, calculation: { ...calc, eligible_count: 10, assessed_count: 9, categories: [...calc.categories, { ...calc.categories[0]!, code: "VENDORS", eligible_count: 6, assessed_count: 6, unknown_count: 0 }] } } }}/>);
    expect(screen.getByText("2 of 4 requirements assessed")).toBeTruthy();
    expect(screen.getByText("50%")).toBeTruthy();
    expect(screen.queryByText("90%")).toBeNull();
  });
  it("marks a snapshot using an older approved policy as out of date", () => {
    const calc = current.current_state!.calculation!;
    render(<ProgramCoverage aggregate={{ ...current, current_state: { ...current.current_state!, calculation: { ...calc, policy_revision: "old" } } }}/>);
    expect(screen.getByText("Out of date")).toBeTruthy();
  });
  it("does not treat missing snapshot freshness or inconsistent counts as current", () => {
    const calc = current.current_state!.calculation!;
    const view = render(<ProgramCoverage aggregate={{ ...current, current_state: { ...current.current_state!, generated_at: "" } }}/>);
    expect(screen.getByText("Coverage unavailable.")).toBeTruthy();
    view.rerender(<ProgramCoverage aggregate={{ ...current, current_state: { ...current.current_state!, calculation: { ...calc, categories: [{ ...calc.categories[0]!, assessed_count: 5 }] } } }}/>);
    expect(screen.getByText("Coverage unavailable.")).toBeTruthy();
    expect(screen.queryByRole("progressbar")).toBeNull();
  });
  it("withholds coverage when eligible requirements are unaccounted for", () => {
    const calc = current.current_state!.calculation!;
    render(<ProgramCoverage aggregate={{ ...current, current_state: { ...current.current_state!, calculation: { ...calc, categories: [{ ...calc.categories[0]!, unknown_count: 0 }] } } }}/>);
    expect(screen.getByText("Coverage unavailable.")).toBeTruthy();
    expect(screen.queryByRole("progressbar")).toBeNull();
  });
});
