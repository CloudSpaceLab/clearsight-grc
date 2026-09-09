import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { VendorFormSummary, VendorFormRow } from "../vendorFormsApi";
import { VendorComplianceOverview } from "./VendorComplianceOverview";

const api = vi.hoisted(() => ({ loadVendorForms: vi.fn() }));
vi.mock("../vendorFormsApi", async (original) => ({ ...await original<object>(), ...api }));
vi.mock("./forms/ResponseAssessment", () => ({ ResponseAssessment: ({ responseID }: { responseID: string }) => <p>Reviewing {responseID}</p> }));
const summary: VendorFormSummary = { relationship_id: "vendor-1", outstanding_forms: 0, overdue_forms: 0, submitted_forms: 1, awaiting_review: 0, unassessed_forms: 0, assessed_forms: 1, highest_concern: "LOW", outdated_forms: 0, observed_at: "2026-09-09T12:00:00Z" };
const row: VendorFormRow = { request_id: "request-1", relationship_id: "vendor-1", response_id: "response-1", form_template_id: "form-1", form_template_version: 1, title: "Third Party Risk Compliance", response_state: "SUBMITTED", deadline: "2026-09-20T12:00:00Z", updated_at: "2026-09-09T12:00:00Z", required_count: 3, answered_required: 3, missing_fields: [], required_reviews: 0, completed_reviews: 0, current: true, outdated: false, attention_items: [] };
const props = { relationshipID: "vendor-1", serviceName: "Payments", summary, summaryState: "live" as const, assessment: null, assessmentState: "live" as const, refreshKey: 0, onOpenForms: vi.fn(), onOpenDueDiligence: vi.fn(), onRequestForm: vi.fn(), onUpdated: vi.fn() };
beforeEach(() => { vi.clearAllMocks(); api.loadVendorForms.mockResolvedValue({ items: [row], observed_at: summary.observed_at }); });

describe("vendor compliance overview", () => {
  it("shows submitted gaps without describing the form as incomplete and opens its exact review", async () => {
    api.loadVendorForms.mockResolvedValue({ items: [{ ...row, attention_items: [{ field_id: "iso", label: "ISO 27001 certificate", state: "MISSING", source: "RESPONSE" }] }], observed_at: summary.observed_at });
    render(<VendorComplianceOverview {...props}/>);
    expect(await screen.findByText("ISO 27001 certificate")).toBeTruthy();
    expect(screen.getByText("Submitted")).toBeTruthy();
    expect(screen.getByText("Action required")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Review Third Party Risk Compliance" }));
    expect(await screen.findByText("Reviewing response-1")).toBeTruthy();
  });
  it("shows no assessment for an empty vendor instead of a favourable compliance claim", async () => {
    api.loadVendorForms.mockResolvedValue({ items: [], observed_at: summary.observed_at });
    render(<VendorComplianceOverview {...props} summary={{ ...summary, submitted_forms: 0, assessed_forms: 0, highest_concern: undefined }}/>);
    expect(await screen.findByText("No forms requested")).toBeTruthy();
    expect(screen.getByText("Not assessed")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Request form" }));
    expect(props.onRequestForm).toHaveBeenCalledOnce();
  });
  it("keeps received evidence separate from missing items", async () => {
    api.loadVendorForms.mockResolvedValue({ items: [{ ...row, response_state: "NO_VENDOR_ACTION", response_id: undefined, held_required: 3, answered_required: 0 }], observed_at: summary.observed_at });
    render(<VendorComplianceOverview {...props}/>);
    expect(await screen.findByText("3 documents received")).toBeTruthy();
    expect(screen.queryByText("Missing")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Review evidence for Third Party Risk Compliance" }));
    expect(props.onOpenDueDiligence).toHaveBeenCalledOnce();
  });
  it("shows outdated submitted evidence and does not use the response deadline as freshness", async () => {
    api.loadVendorForms.mockResolvedValue({ items: [{ ...row, outdated: true, attention_items: [{ field_id: "pci", label: "PCI DSS attestation", state: "EXPIRED", source: "RESPONSE" }] }], observed_at: summary.observed_at });
    render(<VendorComplianceOverview {...props} summary={{ ...summary, outdated_forms: 1 }}/>);
    expect(await screen.findByText("PCI DSS attestation")).toBeTruthy();
    expect(screen.getByText("Expired")).toBeTruthy();
    expect(screen.queryByText("Satisfactory")).toBeNull();
  });
  it("retains a page boundary and loads the next page only on request", async () => {
    api.loadVendorForms.mockResolvedValueOnce({ items: [row], next_cursor: "cursor-2", observed_at: summary.observed_at }).mockResolvedValueOnce({ items: [{ ...row, request_id: "request-2", title: "Privacy review" }], observed_at: summary.observed_at });
    render(<VendorComplianceOverview {...props}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Load more forms" }));
    expect(await screen.findByText("Privacy review")).toBeTruthy();
    expect(api.loadVendorForms).toHaveBeenLastCalledWith("vendor-1", { limit: 25, cursor: "cursor-2" });
  });
  it("shows unavailable data and allows retry without substituting zero counts", async () => {
    api.loadVendorForms.mockRejectedValueOnce(new Error("offline"));
    render(<VendorComplianceOverview {...props} summary={undefined} summaryState="unavailable"/>);
    expect(await screen.findByText("Compliance status unavailable")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    await waitFor(() => expect(api.loadVendorForms).toHaveBeenCalledTimes(2));
    expect(props.onUpdated).toHaveBeenCalledOnce();
  });
  it("shows an automatic failure while independent review is pending", async () => {
    api.loadVendorForms.mockResolvedValue({ items: [{ ...row, required_reviews: 1, completed_reviews: 0, attention_items: [{ rule_id: "audit-clause", label: "SLA audit rights", state: "GAP", source: "RESPONSE" }] }], observed_at: summary.observed_at });
    render(<VendorComplianceOverview {...props} summary={{ ...summary, awaiting_review: 1, unassessed_forms: 1, assessed_forms: 0, highest_concern: undefined }}/>);
    expect(await screen.findByText("SLA audit rights")).toBeTruthy();
    expect(screen.getByText("Not met")).toBeTruthy();
    expect(screen.getByText("Based on submitted answers")).toBeTruthy();
    expect(screen.getByText("1 check awaiting review")).toBeTruthy();
    expect(screen.getByText("Action required")).toBeTruthy();
  });
  it("does not turn unknown freshness into a favourable current result", async () => {
    api.loadVendorForms.mockResolvedValue({ items: [{ ...row, outdated: null }], observed_at: summary.observed_at });
    render(<VendorComplianceOverview {...props}/>);
    await screen.findByText("Third Party Risk Compliance");
    expect(screen.getByText("Unknown")).toBeTruthy();
    expect(screen.queryByText("Low concern")).toBeNull();
  });
  it("does not describe an unscored form as awaiting an assigned reviewer", async () => {
    render(<VendorComplianceOverview {...props} summary={{ ...summary, unassessed_forms: 1, assessed_forms: 0, highest_concern: undefined }}/>);
    await screen.findByText("Third Party Risk Compliance");
    expect(screen.getByText("Not assessed")).toBeTruthy();
  });
  it("shows one requirement when automatic and reviewed findings agree", async () => {
    api.loadVendorForms.mockResolvedValue({ items: [{ ...row, attention_items: [{ rule_id: "audit", label: "SLA audit rights", state: "GAP", source: "RESPONSE" }, { rule_id: "audit", label: "SLA audit rights", state: "GAP", source: "REVIEW" }] }], observed_at: summary.observed_at });
    render(<VendorComplianceOverview {...props}/>);
    await screen.findByText("SLA audit rights");
    expect(screen.getAllByText("SLA audit rights")).toHaveLength(1);
    expect(screen.getByText("Includes reviewed findings")).toBeTruthy();
  });
});
