import { describe, expect, it } from "vitest";
import { matterContextKind, matterDeadlinePresentation, matterPriorityLabel, matterPriorityTone } from "./matterPresentation";

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


describe("Matter domain context", () => {
  const base = { type: "OTHER", source_type: undefined, source_id: undefined, trigger_type: undefined };

  it("uses canonical loss source identity rather than the title", () => {
    expect(matterContextKind({ ...base, type: "OPERATIONAL_LOSS", source_type: "OPERATIONAL_LOSS", source_id: "loss-1" })).toBe("OPERATIONAL_LOSS");
    expect(matterContextKind({ ...base, type: "OPERATIONAL_LOSS", source_type: "DOCUMENT_IMPORT", source_id: "loss-1" })).toBe("GENERIC");
  });

  it("recognizes exact monitoring-backed indicator intervention work", () => {
    expect(matterContextKind({ ...base, type: "KRI_BREACH", source_type: "MONITORING_RESULT", source_id: "result-1" })).toBe("INDICATOR");
    expect(matterContextKind({ ...base, type: "CONTROL_GAP", source_type: "MONITORING_RESULT", source_id: "result-1", trigger_type: "MONITORING_RESULT_ADVERSE" })).toBe("INDICATOR");
  });

  it("keeps unsupported or unlinked types on the complete generic Matter workspace", () => {
    expect(matterContextKind({ ...base, type: "INCIDENT" })).toBe("GENERIC");
    expect(matterContextKind({ ...base, type: "KRI_BREACH", source_type: "MONITORING_RESULT" })).toBe("GENERIC");
  });
});
