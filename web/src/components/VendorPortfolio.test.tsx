import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { VendorFormSummary } from "../vendorFormsApi";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import { VendorPortfolio } from "./VendorPortfolio";

const records = ["CRITICAL", "IMPORTANT", "STANDARD"].map((criticality, index) => ({
  vendor: { id: `v${index}`, legal_name: `Vendor ${index}` },
  relationship: { id: `r${index}`, criticality },
})) as VendorRelationshipAggregate[];
const summaries = new Map(records.map((record, index) => [record.relationship.id, {
  relationship_id: record.relationship.id, outstanding_forms: index, overdue_forms: index === 2 ? 1 : 0,
  awaiting_review: 1, submitted_forms: 2, assessed_forms: 1, unassessed_forms: 1,
  observed_at: `2026-09-0${index + 1}T12:00:00Z`,
} satisfies VendorFormSummary]));

describe("Vendor portfolio metrics", () => {
  it("aggregates only loaded relationships and filters overdue work", () => {
    const onFilter = vi.fn();
    render(<VendorPortfolio records={records} summaries={new Map([...summaries, ["other", { ...summaries.get("r1")!, overdue_forms: 99 }]])} summaryState="live" hasMore onFilter={onFilter}/>);
    expect(within(screen.getByRole("group", { name: "Outstanding forms" })).getByText("3")).toBeTruthy();
    expect(within(screen.getByRole("group", { name: "Overdue forms" })).getByText("1")).toBeTruthy();
    expect(screen.getByText("3 loaded services · More services available")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Review overdue forms" }));
    expect(onFilter).toHaveBeenCalledWith("OVERDUE");
    expect(screen.getByText("3 assessed / 6 submitted")).toBeTruthy();
  });
  it.each(["loading", "unavailable", "live"] as const)("does not turn missing %s summaries into zero totals", (summaryState) => {
    render(<VendorPortfolio records={records} summaries={new Map([["r0", summaries.get("r0")!]])} summaryState={summaryState} hasMore={false} onFilter={vi.fn()}/>);
    expect(within(screen.getByRole("group", { name: "Outstanding forms" })).getByText("Unknown")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Review overdue forms" })).toBeNull();
    expect(screen.getByText("1 of 3 service summaries available")).toBeTruthy();
  });
  it("does not use retained counts while summaries refresh", () => {
    render(<VendorPortfolio records={records} summaries={summaries} summaryState="loading" hasMore={false} onFilter={vi.fn()}/>);
    expect(within(screen.getByRole("group", { name: "Awaiting review" })).getByText("Unknown")).toBeTruthy();
  });
  it("keeps an empty population explicit without a compliance score", () => {
    render(<VendorPortfolio records={[]} summaries={new Map()} summaryState="live" hasMore={false} onFilter={vi.fn()}/>);
    expect(screen.getByText("0 loaded services")).toBeTruthy();
    expect(screen.getByText("No submitted forms in this population.")).toBeTruthy();
    expect(screen.queryByText(/compliant/i)).toBeNull();
  });
});
