import { describe, expect, it } from "vitest";
import { isSourceWorkStatus, workSourceFilters } from "./workSourceFilters";

const now = Date.UTC(2026, 9, 26, 12, 0, 0);

describe("actor source work filters", () => {
  it("maps the visible active states to canonical workflow predicates", () => {
    expect(isSourceWorkStatus("In progress")).toBe(true);
    expect(isSourceWorkStatus("Awaiting review")).toBe(false);
    expect(workSourceFilters("In progress", "ALL", now)).toEqual({ status: "IN_PROGRESS" });
    expect(workSourceFilters("Completed", "ALL", now)).toEqual({});
  });

  it("preserves overdue and inclusive 7/30-day windows with an exclusive SQL boundary", () => {
    expect(workSourceFilters(undefined, "OVERDUE", now)).toEqual({
      due_before: "2026-10-26T12:00:00.000Z",
    });
    expect(workSourceFilters("Ready", "7", now)).toEqual({
      status: "READY",
      due_before: "2026-11-02T12:00:00.001Z",
    });
    expect(workSourceFilters(undefined, "30", now)).toEqual({
      due_before: "2026-11-25T12:00:00.001Z",
    });
    expect(workSourceFilters(undefined, "ALL", now)).toEqual({});
  });

  it("does not fabricate due bounds when the clock is unavailable", () => {
    expect(workSourceFilters("Blocked", "7", NaN)).toEqual({ status: "BLOCKED" });
  });
});
