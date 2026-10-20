import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { MatterAction } from "../types";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import type { VendorRiskFinding, VendorRiskWork } from "../vendorRiskWork";
import { VendorPortfolio } from "./VendorPortfolio";

const api = vi.hoisted(() => ({ loadVendorRiskWork: vi.fn() }));
vi.mock("../vendorRiskWork", async original => ({ ...await original<object>(), ...api }));

const records = [
  { vendor: { id: "v1", legal_name: "Cloudspace Technologies Ltd" }, relationship: { id: "r1", service_name: "Moneytor GetPaid application" } },
  { vendor: { id: "v2", legal_name: "Sentinel Collections" }, relationship: { id: "r2", service_name: "Loan collections" } },
] as VendorRelationshipAggregate[];

function finding(id: string, relationshipID: string, title: string, dueAt: string | undefined, owner: string | undefined, rating: string, status = "TRIAGE", actionStatus = "PLANNED"): VendorRiskFinding {
  return {
    relationshipIDs: [relationshipID],
    record: {
      matter: { id, type: "VENDOR_DEFICIENCY", title, status, known_facts: { sample: true, source_file: "Risk register.xlsx", source_range: "A4:P4", source_rating: rating, source_assessor: "Blessing", source_owner: owner } },
      status_label: status === "CLOSED" ? "Closed" : "Initial review",
      actions: [{ id: `${id}-action`, title: `Resolve ${title.toLowerCase()}`, description: `Full recommendation for ${title}`, status: actionStatus, due_at: dueAt } as MatterAction],
    },
  } as unknown as VendorRiskFinding;
}

const work = {
  complete: true,
  checkedAt: "2026-09-10T12:00:00Z",
  items: [
    finding("finding-1", "r1", "Contract audit rights missing", "2026-03-31T00:00:00Z", "Hakeem", "Medium"),
    finding("finding-2", "r1", "Independent testing missing", "2026-05-15T00:00:00Z", "Hakeem", "Medium"),
    finding("finding-3", "r1", "Security certification missing", "2026-06-30T00:00:00Z", "Blessing", "High"),
    finding("finding-4", "r2", "Incident reporting gap", "2026-08-01T00:00:00Z", "Ada", "High"),
    finding("finding-5", "r2", "Recovery evidence missing", "2026-08-15T00:00:00Z", "Ada", "Low"),
    finding("finding-6", "r1", "Closed exception", "2026-01-01T00:00:00Z", "Hakeem", "Low", "CLOSED", "IMPLEMENTED"),
  ],
} as VendorRiskWork;

beforeEach(() => {
  vi.clearAllMocks();
  window.sessionStorage.clear();
  api.loadVendorRiskWork.mockResolvedValue(work);
});

describe("vendor exception overview", () => {
  it("shows a compact reconciled summary and an overdue-first operational queue", async () => {
    render(<VendorPortfolio records={records} hasMore={false}/>);
    const first = await within(document.querySelector(".vendor-exception-list") as HTMLElement).findByText("Contract audit rights missing");
    const summary = screen.getByRole("region", { name: "Vendor overview summary" });
    expect(within(summary).getByRole("button", { name: "2 vendor services" })).toBeTruthy();
    expect(within(summary).getByRole("button", { name: "5 open exceptions" })).toBeTruthy();
    expect(within(summary).getByRole("button", { name: "5 open actions" })).toBeTruthy();
    expect(within(summary).getByRole("button", { name: "5 overdue actions" })).toBeTruthy();
    expect(screen.getByRole("region", { name: "Service criticality" })).toBeTruthy();
    expect(screen.getByRole("region", { name: "Exception stages" })).toBeTruthy();
    expect(screen.getByRole("region", { name: "Needs attention" })).toBeTruthy();
    expect(within(screen.getByRole("region", { name: "Exception stages" })).getByText("5 open exceptions")).toBeTruthy();
    expect(document.querySelector(".vendor-metric")).toBeNull();
    expect(screen.getAllByText("Sample data")).toHaveLength(1);
    expect(first.closest("li")?.nextElementSibling?.textContent).toContain("Independent testing missing");
    expect(first.closest("li")?.textContent).toContain("Hakeem");
    expect(first.closest("li")?.textContent).toContain("Resolve contract audit rights missing");
    expect(first.closest("li")?.textContent).toContain("31 Mar 2026");
    expect(screen.queryByText("Full recommendation for Contract audit rights missing")).toBeNull();
    expect(screen.queryByText("A4:P4")).toBeNull();
  });

  it("filters the queue by summary state, vendor, owner and recorded rating", async () => {
    render(<VendorPortfolio records={records} hasMore={false}/>);
    await within(document.querySelector(".vendor-exception-list") as HTMLElement).findByText("Contract audit rights missing");
    const queue = within(document.querySelector(".vendor-exception-list") as HTMLElement);
    fireEvent.change(screen.getByLabelText("Vendor"), { target: { value: "v2" } });
    expect(queue.queryByText("Contract audit rights missing")).toBeNull();
    expect(queue.getByText("Incident reporting gap")).toBeTruthy();
    fireEvent.change(screen.getByLabelText("Owner"), { target: { value: "Ada" } });
    fireEvent.change(screen.getByLabelText("Source rating"), { target: { value: "Low" } });
    expect(queue.queryByText("Incident reporting gap")).toBeNull();
    expect(queue.getByText("Recovery evidence missing")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "All open" }));
    expect(queue.queryByText("Closed exception")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "All exceptions" }));
    fireEvent.change(screen.getByLabelText("Vendor"), { target: { value: "v1" } });
    fireEvent.change(screen.getByLabelText("Owner"), { target: { value: "Hakeem" } });
    fireEvent.change(screen.getByLabelText("Source rating"), { target: { value: "Low" } });
    expect(queue.getByText("Closed exception")).toBeTruthy();
  });

  it("opens the canonical Matter from the single row action", async () => {
    const onOpenMatter = vi.fn();
    render(<VendorPortfolio records={records} hasMore={false} onOpenMatter={onOpenMatter}/>);
    await within(document.querySelector(".vendor-exception-list") as HTMLElement).findByText("Contract audit rights missing");
    fireEvent.click(screen.getByRole("button", { name: "Review exception: Contract audit rights missing" }));
    expect(onOpenMatter).toHaveBeenCalledWith("finding-1");
    fireEvent.click(screen.getByRole("button", { name: "Open attention: Contract audit rights missing" }));
    expect(onOpenMatter).toHaveBeenCalledTimes(2);
  });

  it("keeps unavailable and partial aggregate values explicitly unknown", async () => {
    api.loadVendorRiskWork.mockRejectedValueOnce(new Error("offline"));
    const { unmount } = render(<VendorPortfolio records={records} hasMore/>);
    expect(await screen.findByText("Vendor exceptions could not be checked.")).toBeTruthy();
    expect(screen.getAllByText("Unknown").length).toBeGreaterThanOrEqual(3);
    expect(within(screen.getByRole("region", { name: "Exception stages" })).getByText(/Stage totals unavailable/)).toBeTruthy();
    unmount();
    api.loadVendorRiskWork.mockResolvedValue({ ...work, complete: false });
    render(<VendorPortfolio records={records} hasMore/>);
    await within(document.querySelector(".vendor-exception-list") as HTMLElement).findByText("Contract audit rights missing");
    expect(screen.getByText("Some vendor exceptions could not be checked. Totals unknown.")).toBeTruthy();
    expect(screen.getAllByText("Unknown").length).toBeGreaterThanOrEqual(3);
  });
});
