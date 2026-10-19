import { describe, expect, it } from "vitest";
import type { AttentionItem } from "../types";
import { countLoadedWork, matchesWorkFilter } from "./workQueuePresentation";

const referenceTime = Date.UTC(2026, 9, 19, 12);
const base: AttentionItem = {
  id: "request", type: "MATTER", title: "Owner review", why_now: "New submission",
  scope: "Operations", state: "Assigned", evidence: "Revision 3", owner: "Reviewer",
  due_at: "", primary_action: "Review",
};

describe("bounded assigned work measures", () => {
  it("separates overdue from the next seven days and ignores invalid deadlines", () => {
    const rows: AttentionItem[] = [
      { ...base, id: "late", due_at: new Date(referenceTime - 1).toISOString() },
      { ...base, id: "now", due_at: new Date(referenceTime).toISOString() },
      { ...base, id: "seventh", due_at: new Date(referenceTime + 7 * 86400000).toISOString() },
      { ...base, id: "later", due_at: new Date(referenceTime + 7 * 86400000 + 1).toISOString() },
      { ...base, id: "unset", due_at: "0001-01-01T00:00:00Z" },
      { ...base, id: "invalid", due_at: "not-a-date" },
    ];
    const counts = countLoadedWork(rows, referenceTime);
    expect(counts).toEqual({ ALL: 6, OVERDUE: 1, DUE_SOON: 2, APPROVALS: 0, EVIDENCE: 0 });
    expect(matchesWorkFilter(rows[0]!, "DUE_SOON", referenceTime)).toBe(false);
    expect(matchesWorkFilter(rows[2]!, "DUE_SOON", referenceTime)).toBe(true);
    expect(matchesWorkFilter(rows[3]!, "DUE_SOON", referenceTime)).toBe(false);
  });

  it("counts explicit authorization and evidence exceptions, not inferred approvals", () => {
    const rows: AttentionItem[] = [
      { ...base, id: "approval", intervention_class: "AUTHORIZATION" },
      { ...base, id: "decision", intervention_class: "DECISION" },
      { ...base, id: "exception", intervention_class: "EVIDENCE_EXCEPTION" },
      { ...base, id: "request", intervention_class: "REVIEW", action_target_type: "EVIDENCE_REQUEST" },
    ];
    expect(countLoadedWork(rows, referenceTime)).toEqual({
      ALL: 4, OVERDUE: 0, DUE_SOON: 0, APPROVALS: 1, EVIDENCE: 1,
    });
  });
});
