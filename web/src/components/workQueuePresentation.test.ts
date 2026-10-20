import { describe, expect, it } from "vitest";
import type { AttentionItem } from "../types";
import { countLoadedWork, filterLoadedWork, matchesDueWindow, matchesWorkFilter } from "./workQueuePresentation";

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

  it("filters 7/30-day windows against the server snapshot and excludes unknown deadlines", () => {
    const rows: AttentionItem[] = [
      { ...base, id: "overdue", due_at: new Date(referenceTime - 86400000).toISOString() },
      { ...base, id: "seventh", due_at: new Date(referenceTime + 7 * 86400000).toISOString() },
      { ...base, id: "thirtieth", due_at: new Date(referenceTime + 30 * 86400000).toISOString() },
      { ...base, id: "later", due_at: new Date(referenceTime + 30 * 86400000 + 1).toISOString() },
      { ...base, id: "unset", due_at: "" },
    ];
    expect(filterLoadedWork(rows, "ALL", "", undefined, referenceTime, "7").map((item) => item.id))
      .toEqual(["overdue", "seventh"]);
    expect(filterLoadedWork(rows, "ALL", "", undefined, referenceTime, "30").map((item) => item.id))
      .toEqual(["overdue", "seventh", "thirtieth"]);
    expect(filterLoadedWork(rows, "ALL", "", undefined, referenceTime, "OVERDUE").map((item) => item.id))
      .toEqual(["overdue"]);
    expect(matchesDueWindow(rows[4]!, "7", referenceTime)).toBe(false);
    expect(filterLoadedWork(rows, "ALL", "", undefined, referenceTime).map((item) => item.id))
      .toHaveLength(5);
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
  it("searches only loaded records and combines exact status with actor-facing work intent", () => {
    const rows: AttentionItem[] = [
      { ...base, id: "owner", title: "Review overdue vendor evidence", scope: "North branch", state: "Blocked", intervention_class: "REVIEW" },
      { ...base, id: "approval", title: "Authorize vendor evidence", scope: "South branch", state: "Ready", intervention_class: "AUTHORIZATION" },
      { ...base, id: "hidden", title: "Review payment loss", scope: "North branch", state: "Ready", intervention_class: "REVIEW" },
    ];
    expect(filterLoadedWork(rows, "ALL", "VENDOR evidence", undefined, referenceTime).map((item) => item.id))
      .toEqual(["owner", "approval"]);
    expect(filterLoadedWork(rows, "ALL", "north", "Blocked", referenceTime).map((item) => item.id))
      .toEqual(["owner"]);
    expect(filterLoadedWork(rows, "APPROVALS", "vendor", "Blocked", referenceTime)).toEqual([]);
    expect(filterLoadedWork(rows, "ALL", "  ", undefined, referenceTime)).toHaveLength(3);
    expect(rows).toHaveLength(3);
  });

});
