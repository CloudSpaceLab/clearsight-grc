import { describe, expect, it } from "vitest";
import { matterDeadlinePresentation, matterPriorityLabel, matterPriorityTone } from "./matterPresentation";

describe("matter attention presentation", () => {
  it("keeps critical and high issues on the same critical tone used by Home", () => {
    expect(matterPriorityLabel(5)).toBe("Critical");
    expect(matterPriorityLabel(4)).toBe("High");
    expect(matterPriorityTone(5)).toBe("error");
    expect(matterPriorityTone(4)).toBe("error");
    expect(matterPriorityTone(3)).toBe("warning");
  });

  it("keeps overdue work on warning rather than consuming critical tone", () => {
    const now = new Date("2026-10-05T10:00:00Z").valueOf();
    expect(matterDeadlinePresentation("2026-10-04T10:00:00Z", now)).toMatchObject({
      state: "OVERDUE",
      label: "Overdue",
      tone: "warning",
    });
    expect(matterDeadlinePresentation("2026-10-06T10:00:00Z", now)).toMatchObject({
      state: "DUE",
      label: "Due",
      tone: "neutral",
    });
  });
});
