import { describe, expect, it } from "vitest";
import type { MatterAction } from "./types";
import type { VendorRiskFinding } from "./vendorRiskWork";
import { presentVendorExceptions } from "./vendorExceptionPresentation";

const now = Date.parse("2026-09-10T12:00:00Z");

function finding(id: string, title: string, status: string, actions: Partial<MatterAction>[], facts: Record<string, unknown> = {}): VendorRiskFinding {
  return {
    relationshipIDs: [`relationship-${id}`],
    record: {
      matter: { id, type: "VENDOR_DEFICIENCY", title, status, known_facts: facts },
      actions: actions.map((action, index) => ({ id: `${id}-action-${index}`, title: `Action ${index + 1}`, description: "", status: "PLANNED", ...action })),
    },
  } as unknown as VendorRiskFinding;
}

const fixtures = [
  finding("routine", "Routine open", "TRIAGE", [{ due_at: "2027-01-15T00:00:00Z" }], { source_owner: "Ada", source_rating: "Low" }),
  finding("due-soon", "Due soon", "ASSESSMENT", [{ due_at: "2026-09-20T00:00:00Z" }], { source_owner: "Blessing", source_rating: "Medium" }),
  finding("incomplete", "Assignment missing", "TRIAGE", [{ due_at: "2026-10-31T00:00:00Z" }], { source_rating: "Medium" }),
  finding("blocked", "Blocked response", "ACTION_IN_PROGRESS", [{ status: "BLOCKED", due_at: "2026-10-01T00:00:00Z" }], { source_owner: "Hakeem", source_rating: "High" }),
  finding("overdue-later", "Overdue later", "ACTION_IN_PROGRESS", [{ due_at: "2026-08-31T00:00:00Z" }], { source_owner: "Hakeem", source_rating: "High" }),
  finding("overdue-earlier", "Overdue earlier", "ACTION_IN_PROGRESS", [
    { due_at: "2026-07-01T00:00:00Z" },
    { status: "IN_PROGRESS", due_at: "2026-07-20T00:00:00Z" },
  ], { source_owner: "Hakeem", source_rating: "High" }),
  finding("no-deadline", "Deadline missing", "TRIAGE", [{}], { source_owner: "Ada", source_rating: "Medium" }),
  finding("closed", "Closed exception", "CLOSED", [{ status: "IMPLEMENTED", due_at: "2026-06-01T00:00:00Z" }], { source_owner: "Ada", source_rating: "Low" }),
];

describe("vendor exception presentation", () => {
  it("orders attention by operational urgency and earliest deadline", () => {
    const rows = presentVendorExceptions(fixtures, "ATTENTION", now);
    expect(rows.map(row => [row.item.record.matter.id, row.band])).toEqual([
      ["overdue-earlier", "OVERDUE"],
      ["overdue-later", "OVERDUE"],
      ["blocked", "BLOCKED"],
      ["incomplete", "INCOMPLETE"],
      ["no-deadline", "INCOMPLETE"],
      ["due-soon", "DUE_SOON"],
    ]);
    expect(rows[0]).toMatchObject({ openActionCount: 2, overdueActionCount: 2, owner: "Hakeem", sourceRating: "High" });
    expect(rows[0]?.nextAction?.due_at).toBe("2026-07-01T00:00:00Z");
  });

  it("keeps each filter boundary explicit", () => {
    expect(presentVendorExceptions(fixtures, "OVERDUE", now).map(row => row.item.record.matter.id)).toEqual(["overdue-earlier", "overdue-later"]);
    expect(presentVendorExceptions(fixtures, "OPEN", now)).toHaveLength(7);
    expect(presentVendorExceptions(fixtures, "ALL", now)).toHaveLength(8);
    expect(presentVendorExceptions(fixtures, "ALL", now).at(-1)?.band).toBe("CLOSED");
  });

  it("does not turn a missing deadline into an overdue deadline", () => {
    const row = presentVendorExceptions(fixtures, "OPEN", now).find(value => value.item.record.matter.id === "no-deadline");
    expect(row).toMatchObject({ band: "INCOMPLETE", overdueActionCount: 0 });
    expect(row?.nextAction?.due_at).toBeUndefined();
  });
});
