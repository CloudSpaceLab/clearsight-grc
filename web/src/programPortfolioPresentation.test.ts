import { describe, expect, it } from "vitest";
import type { ProgramListSummary } from "./programPortfolioPresentation";
import { needsProgramAssessment, summarizeProgramPortfolio } from "./programPortfolioPresentation";

function program(id: string, options: {
  state?: ProgramListSummary["overall_state"];
  issues?: number;
  assessedVersion?: number;
  version?: number;
  stale?: boolean;
  status?: string;
} = {}): ProgramListSummary {
  return {
    program: { id, name: id, code: id, status: options.status ?? "ACTIVE", version: options.version ?? 3, owning_function: "Compliance" },
    state_label: options.state ?? "CURRENT",
    overall_state: options.state ?? "CURRENT",
    reasons: [], reasons_total: 0, reasons_omitted: 0,
    open_matter_count: options.issues,
    requirement_count: 5, evidence_check_count: 2, safeguard_count: 1,
    program_version: options.version ?? 3,
    assessed_program_version: options.assessedVersion ?? 3,
    projection_version: 7, projection_stale: options.stale ?? false,
  } as ProgramListSummary;
}

describe("loaded Program portfolio presentation", () => {
  it("separates outdated and unassessed Programs from current and adverse assessments", () => {
    const view = summarizeProgramPortfolio([
      program("current", { issues: 0 }),
      program("overdue", { state: "OVERDUE", issues: 9 }),
      program("unassessed", { assessedVersion: 0, state: "CURRENT", issues: 7 }),
      program("stale", { assessedVersion: 2, issues: 5 }),
      program("not-applicable", { state: "NOT_APPLICABLE", issues: 0 }),
      program("draft", { status: "DRAFT", issues: 1 }),
    ]);
    expect({ attention: view.attention, current: view.current, setup: view.setup, notApplicable: view.notApplicable })
      .toEqual({ attention: 1, current: 1, setup: 3, notApplicable: 1 });
    expect(view.segments.reduce((sum, segment) => sum + segment.count, 0)).toBe(6);
    expect(view.knownIssuePrograms.map((item) => item.id)).toEqual(["overdue"]);
    expect(view.excludedFromIssueComparison).toBe(3);
  });

  it("does not allow a missing or mismatched assessment to inherit a current status", () => {
    expect(needsProgramAssessment(program("missing", { assessedVersion: 0 }))).toBe(true);
    expect(needsProgramAssessment(program("stale", { stale: true }))).toBe(true);
    expect(needsProgramAssessment(program("different", { assessedVersion: 2 }))).toBe(true);
    expect(needsProgramAssessment(program("current"))).toBe(false);
  });

  it("ranks only known nonnegative counts and retains a clear boundary for unknown data", () => {
    const result = summarizeProgramPortfolio([
      program("small", { issues: 2 }),
      program("unknown"),
      program("invalid", { issues: -1 }),
      program("big", { issues: 17 }),
      program("medium", { issues: 6 }),
      program("stale-large", { stale: true, issues: 200 }),
      program("minor", { issues: 1 }),
      program("hidden-fifth", { issues: 1 }),
    ]);
    expect(result.knownIssuePrograms.map((item) => item.id)).toEqual(["big", "medium", "small", "hidden-fifth"]);
    expect(result.excludedFromIssueComparison).toBe(3);
  });
});
