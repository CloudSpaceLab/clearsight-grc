import { beforeEach, describe, expect, it, vi } from "vitest";
import type { MatterAggregate } from "./types";
import { loadVendorRiskWork, summarizeVendorRiskWork } from "./vendorRiskWork";
const api = vi.hoisted(() => ({ loadVendorRelationshipLinks: vi.fn(), loadMatter: vi.fn() }));
vi.mock("./vendorLinkApi", () => ({ loadVendorRelationshipLinks: api.loadVendorRelationshipLinks }));
vi.mock("./api", () => ({ loadMatter: api.loadMatter }));
const finding = { matter: { id: "f1", type: "VENDOR_DEFICIENCY", title: "Audit rights missing", status: "ACTIONS_IN_PROGRESS", updated_at: "2026-09-01T00:00:00Z" }, actions: [
  { id: "a1", title: "Sign addendum", status: "IN_PROGRESS", due_at: "2026-03-31T00:00:00Z" },
  { id: "a2", title: "Provide certificate", status: "IMPLEMENTED", due_at: "2026-03-31T00:00:00Z" },
] } as MatterAggregate;
beforeEach(() => { vi.resetAllMocks(); api.loadVendorRelationshipLinks.mockResolvedValue({ items: [{ target_type: "MATTER", target_id: "f1", state: "ACTIVE" }] }); api.loadMatter.mockResolvedValue(finding); });
describe("linked vendor risk work", () => {
  it("deduplicates shared findings and separates implemented actions from open work", async () => {
    const work = await loadVendorRiskWork(["r1", "r2"]);
    expect(api.loadMatter).toHaveBeenCalledTimes(1);
    expect(work.items[0]!.relationshipIDs).toEqual(["r1", "r2"]);
    expect(summarizeVendorRiskWork(work, Date.parse("2026-09-10"))).toEqual({ findings: 1, openActions: 1, overdueActions: 1, implementedActions: 1 });
  });
  it("marks incomplete links and denied exact reads as unknown", async () => {
    api.loadVendorRelationshipLinks.mockResolvedValue({ items: [{ target_type: "MATTER", target_id: "f1", state: "ACTIVE" }], next_cursor: "more" });
    api.loadMatter.mockRejectedValue(new Error("not found"));
    const work = await loadVendorRiskWork(["r1"]);
    expect(work.complete).toBe(false);
    expect(summarizeVendorRiskWork(work).findings).toBeNull();
  });
  it("excludes closed findings and does not invent a deadline", () => {
    const work = { complete: true, checkedAt: "2026-09-10T00:00:00Z", items: [{ relationshipIDs: ["r1"], record: { ...finding, matter: { ...finding.matter, status: "CLOSED" } } }, { relationshipIDs: ["r1"], record: { ...finding, matter: { ...finding.matter, id: "f2" }, actions: [{ id: "a3", title: "Review test", description: "Review supplied evidence", status: "PLANNED" }] } }] };
    expect(summarizeVendorRiskWork(work)).toEqual({ findings: 1, openActions: 1, overdueActions: 0, implementedActions: 0 });
  });
});
