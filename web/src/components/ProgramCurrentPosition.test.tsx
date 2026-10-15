import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ProgramAggregate } from "../types";
import type { ProgramOperations } from "../programOperationsApi";
import type { ProgramReviewDigest } from "../programReviewApi";
import { ProgramCurrentPosition } from "./ProgramCurrentPosition";

const aggregate = { program: { id: "program-1", version: 3 }, state_label: "Up to date", requirements: [] } as unknown as ProgramAggregate;
const operations = { operations: [] } as unknown as ProgramOperations;
const digest = { review_required: false } as ProgramReviewDigest;
const current = { id: "state-1", program_version: 3, generated_at: "2026-09-09T09:00:00Z", reasons: [], open_matter_count: 0 } as unknown as NonNullable<ProgramAggregate["current_state"]>;

describe("Program calculation truth", () => {
  it("keeps an absent calculation and issue population unknown", () => {
    render(<ProgramCurrentPosition aggregate={aggregate} operations={operations} digest={digest}/>);
    expect(screen.getByRole("heading", { name: "Unknown" })).toBeTruthy();
    expect(screen.getByText("Open issues").parentElement?.textContent).toBe("Open issues Unknown");
    expect(screen.queryByText(/No status exceptions|being recalculated|latest Program changes/)).toBeNull();
  });

  it.each([2, 4])("labels a mismatched calculation version %s as out of date without inferring worker activity", (version) => {
    render(<ProgramCurrentPosition aggregate={{ ...aggregate, current_state: { ...current, program_version: version } }} operations={operations} digest={digest}/>);
    expect(screen.getByRole("heading", { name: "Out of date" })).toBeTruthy();
    expect(screen.getByText("Open issues").parentElement?.textContent).toBe("Open issues 0 (last calculation)");
    expect(screen.queryByText(/No status exceptions.*latest|being recalculated|Updating status/)).toBeNull();
  });

  it("distinguishes a recorded zero from an absent count on a current calculation", () => {
    const { rerender } = render(<ProgramCurrentPosition aggregate={{ ...aggregate, current_state: current }} operations={operations} digest={digest}/>);
    expect(screen.getByText("Open issues").parentElement?.textContent).toBe("Open issues 0");
    rerender(<ProgramCurrentPosition aggregate={{ ...aggregate, current_state: { ...current, open_matter_count: undefined } as unknown as typeof current }} operations={operations} digest={digest}/>);
    expect(screen.getByText("Open issues").parentElement?.textContent).toBe("Open issues Unknown");
  });

  it("does not turn missing reasons or an invalid timestamp into a clean latest calculation", () => {
    render(<ProgramCurrentPosition aggregate={{ ...aggregate, current_state: { ...current, reasons: undefined, generated_at: "invalid" } as unknown as typeof current }} operations={operations} digest={digest}/>);
    expect(screen.getByText("Status reasons unavailable.")).toBeTruthy();
    expect(screen.queryByText(/No status exceptions|Invalid Date/)).toBeNull();
  });

  it("does not invent an assessment version when freshness metadata is absent", () => {
    render(<ProgramCurrentPosition aggregate={{ ...aggregate, current_state: { ...current, program_version: undefined } as unknown as typeof current }} operations={operations} digest={digest}/>);
    expect(screen.getByRole("heading", { name: "Unknown" })).toBeTruthy();
    expect(screen.getByText(/Assessment version unavailable/)).toBeTruthy();
    expect(screen.queryByText(/version 0|latest Program version/)).toBeNull();
  });

  it("shows verified requirement compliance without collapsing zero into missing", () => {
    const verified = {
      ...aggregate,
      program: {
        ...aggregate.program,
        calculation_policy: { last_revision: 1, active: {
          number: 1, checksum: "verified", maker_id: "owner", approver_id: "reviewer",
          proposed_at: "2026-09-08T00:00:00Z", approved_at: "2026-09-09T08:00:00Z",
          definition: { revision: "program-v1", minimum_coverage_bps: 10000, categories: [{ code: "REQUIREMENTS", weight_bps: 10000 }] },
        } },
      },
      current_state: { ...current, calculation: {
        policy_revision: "program-v1", state: "FINAL" as const, score_bps: 0,
        coverage_bps: 10000, eligible_count: 1, assessed_count: 1,
        unknown_count: 0, excluded_count: 0, categories: [],
      } },
    } as ProgramAggregate;
    const { rerender } = render(<ProgramCurrentPosition aggregate={verified} operations={operations} digest={digest}/>);
    expect(screen.getByText("Requirement compliance").parentElement?.textContent).toBe("Requirement compliance 0% · 100% coverage");

    rerender(<ProgramCurrentPosition aggregate={{ ...verified, current_state: {
      ...verified.current_state!, calculation: { ...verified.current_state!.calculation!, state: "PROVISIONAL", score_bps: undefined, coverage_bps: 5000 },
    } }} operations={operations} digest={digest}/>);
    expect(screen.getByText("Requirement compliance").parentElement?.textContent).toBe("Requirement compliance Not assessed · 50% coverage");

    rerender(<ProgramCurrentPosition aggregate={{ ...verified, program: { ...verified.program, version: 4 } }} operations={operations} digest={digest}/>);
    expect(screen.queryByText("Requirement compliance")).toBeNull();
  });

  it("does not show a requirements-only percentage for a different policy", () => {
    const value = {
      ...aggregate,
      program: { ...aggregate.program, calculation_policy: { last_revision: 2, active: {
        number: 2, checksum: "verified", maker_id: "owner", approver_id: "reviewer",
        proposed_at: "2026-09-08T00:00:00Z", approved_at: "2026-09-09T08:00:00Z",
        definition: { revision: "program-v2", minimum_coverage_bps: 10000, categories: [
          { code: "REQUIREMENTS", weight_bps: 7000 }, { code: "VENDORS", weight_bps: 3000 },
        ] },
      } } },
      current_state: { ...current, calculation: {
        policy_revision: "program-v1", state: "FINAL" as const, score_bps: 10000,
        coverage_bps: 10000, eligible_count: 1, assessed_count: 1,
        unknown_count: 0, excluded_count: 0, categories: [],
      } },
    } as ProgramAggregate;
    render(<ProgramCurrentPosition aggregate={value} operations={operations} digest={digest}/>);
    expect(screen.queryByText("Requirement compliance")).toBeNull();
  });

  it("names an expired Program assessment without implying that a vendor response is stale", () => {
    const onNavigate = vi.fn();
    render(<ProgramCurrentPosition aggregate={{ ...aggregate, current_state: { ...current, reasons: [{ code: "EVIDENCE_EXPIRED", summary: "Evidence is out of date for Annual control review.", object_type: "EVIDENCE_CONTRACT", object_id: "contract-1" }] } }} operations={operations} digest={digest} onNavigate={onNavigate}/>);
    expect(screen.getByText("1 · Assessments need renewal")).toBeTruthy();
    expect(screen.getByText("Validity has ended for these internal assessments.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Review assessments" }));
    expect(onNavigate).toHaveBeenCalledWith("evidence-results");
  });
});
