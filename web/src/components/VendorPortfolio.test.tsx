import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import type { VendorRiskWork } from "../vendorRiskWork";
import { VendorPortfolio } from "./VendorPortfolio";
const api = vi.hoisted(() => ({ loadVendorRiskWork: vi.fn() }));
vi.mock("../vendorRiskWork", async original => ({ ...await original<object>(), ...api }));
const records = [{ vendor: { id: "v1", legal_name: "Sample provider" }, relationship: { id: "r1", service_name: "Payment service" } }] as VendorRelationshipAggregate[];
const work = { complete: true, checkedAt: "2026-09-10T12:00:00Z", items: [{ relationshipIDs: ["r1"], record: { matter: { id: "finding-1", type: "VENDOR_DEFICIENCY", title: "Contract audit rights missing", status: "TRIAGE", known_facts: { source_file: "Risk register.xlsx", source_rating: "High", source_assessor: "Blessing", source_owner: "Hakeem" }, due_at: "2026-03-31T00:00:00Z" }, status_label: "Initial review", actions: [{ id: "action-1", title: "Agree the audit addendum", status: "PLANNED", due_at: "2026-03-31T00:00:00Z" }] } }] } as unknown as VendorRiskWork;
beforeEach(() => { vi.clearAllMocks(); api.loadVendorRiskWork.mockResolvedValue(work); });
describe("vendor source-linked portfolio", () => {
  it("shows stored finding/action counts and opens the exact issue", async () => {
    const onOpenMatter = vi.fn();
    render(<VendorPortfolio records={records} hasMore={false} onOpenMatter={onOpenMatter}/>);
    await screen.findByText("Contract audit rights missing");
    expect(within(screen.getByRole("group", { name: "Open findings" })).getByText("1")).toBeTruthy();
    expect(within(screen.getByRole("group", { name: "Overdue actions" })).getByText("1")).toBeTruthy();
    expect(screen.getByText("Source rating: High")).toBeTruthy();
    expect(screen.getByText("Internal assessor: Blessing")).toBeTruthy();
    expect(screen.getByText("Action performer: Hakeem")).toBeTruthy();
    expect(screen.queryByText("Source owner: Hakeem")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Review Contract audit rights missing" }));
    expect(onOpenMatter).toHaveBeenCalledWith("finding-1");
    expect(screen.queryByText("Response review")).toBeNull();
    expect(screen.queryByText("Service criticality")).toBeNull();
  });
  it("keeps unavailable linked work unknown and offers retry", async () => {
    api.loadVendorRiskWork.mockRejectedValueOnce(new Error("offline"));
    render(<VendorPortfolio records={records} hasMore/>);
    expect(await screen.findByText("Linked findings could not be checked.")).toBeTruthy();
    expect(within(screen.getByRole("group", { name: "Open findings" })).getByText("Unknown")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Retry findings" }));
    expect(await screen.findByText("Contract audit rights missing")).toBeTruthy();
  });
  it("keeps partial totals unknown while retaining readable findings", async () => {
    api.loadVendorRiskWork.mockResolvedValue({ ...work, complete: false });
    render(<VendorPortfolio records={records} hasMore/>);
    await screen.findByText("Contract audit rights missing");
    expect(within(screen.getByRole("group", { name: "Open actions" })).getByText("Unknown")).toBeTruthy();
    expect(screen.getByText("Some linked findings could not be checked. Totals unknown.")).toBeTruthy();
  });
});
