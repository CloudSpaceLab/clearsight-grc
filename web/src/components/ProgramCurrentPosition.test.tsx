import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
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
    expect(screen.getByText("Status reasons are unavailable.")).toBeTruthy();
    expect(screen.queryByText(/No status exceptions|Invalid Date/)).toBeNull();
  });

  it("does not invent an assessment version when freshness metadata is absent", () => {
    render(<ProgramCurrentPosition aggregate={{ ...aggregate, current_state: { ...current, program_version: undefined } as unknown as typeof current }} operations={operations} digest={digest}/>);
    expect(screen.getByRole("heading", { name: "Unknown" })).toBeTruthy();
    expect(screen.getByText(/The assessment version is unavailable/)).toBeTruthy();
    expect(screen.queryByText(/version 0|latest Program version/)).toBeNull();
  });
});
