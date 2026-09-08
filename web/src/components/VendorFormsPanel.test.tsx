import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { VendorFormsPanel } from "./VendorFormsPanel";
const api = vi.hoisted(() => ({ loadVendorForms: vi.fn(), loadCompletedResponses: vi.fn() }));
vi.mock("../vendorFormsApi", () => api);
vi.mock("../formsDistributionApi", async (original) => ({ ...await original<typeof import("../formsDistributionApi")>(), loadCompletedResponses: api.loadCompletedResponses }));
vi.mock("./forms/ResponseAssessment", () => ({ ResponseAssessment: ({ responseID }: { responseID: string }) => <section aria-label={`Assess ${responseID}`}/> }));
vi.mock("./documents/DocumentBrowser", () => ({ DocumentBrowser: () => null }));
const row = { request_id: "request", relationship_id: "r1", response_id: "submitted-2", form_template_id: "form", form_template_version: 4, title: "Security evidence", response_state: "SUBMITTED", assessment_state: "AWAITING_REVIEW", required_reviews: 2, completed_reviews: 0, required_count: 3, answered_required: 3, current: true, deadline: "2099-09-08T12:00:00Z", updated_at: "2026-09-08T12:00:00Z", submitted_at: "2026-09-08T12:00:00Z", missing_fields: [] };
beforeEach(() => { vi.clearAllMocks(); api.loadVendorForms.mockResolvedValue({ items: [row], observed_at: "2026-09-08T12:00:00Z" }); });
describe("vendor forms and responses", () => {
  it("identifies a partly replaced response without describing its old score as current", async () => {
    api.loadVendorForms.mockResolvedValue({ items: [{ ...row, response_currency: "PARTIALLY_REPLACED" }], observed_at: row.updated_at });
    render(<VendorFormsPanel relationshipID="r1" serviceName="Payments" onRequestForm={() => {}}/>);
    expect(await screen.findByText(/Partly replaced response/)).toBeTruthy();
    expect(screen.queryByText(/Current request/)).toBeNull();
    expect(screen.getByText(/remaining fields require review/i)).toBeTruthy();
  });
  it("pages the exact vendor's response history and opens the selected historical revision", async () => {
    api.loadCompletedResponses.mockResolvedValueOnce({ items: [{ id: "revision-current", title: "Current security evidence", revision: 2, current: true, form_template_version: 4, completed_at: "2026-09-08T12:00:00Z" }], next_cursor: "history-next" }).mockResolvedValueOnce({ items: [{ id: "revision-historical", title: "Earlier security evidence", revision: 1, current: false, form_template_version: 4, completed_at: "2026-09-01T12:00:00Z" }] });
    render(<VendorFormsPanel relationshipID="r1" serviceName="Payments" onRequestForm={() => {}}/>);
    fireEvent.click(screen.getByRole("button", { name: "View response history" }));
    expect(await screen.findByText("Current security evidence")).toBeTruthy();
    expect(api.loadCompletedResponses).toHaveBeenLastCalledWith({ subject_type: "VENDOR_RELATIONSHIP", subject_id: "r1", current_only: false, sort: "COMPLETED_DESC", limit: 25, cursor: undefined });
    fireEvent.click(screen.getByRole("button", { name: "Load earlier responses" }));
    expect(await screen.findByText("Earlier security evidence")).toBeTruthy();
    expect(api.loadCompletedResponses).toHaveBeenLastCalledWith(expect.objectContaining({ subject_id: "r1", cursor: "history-next" }));
    fireEvent.click(screen.getByRole("button", { name: "Review Earlier security evidence revision 1" }));
    expect(screen.getByRole("region", { name: "Assess revision-historical" })).toBeTruthy();
  });
  it("separates a submitted response from required bank review and opens the exact response", async () => {
    render(<VendorFormsPanel relationshipID="r1" serviceName="Payments" onRequestForm={() => {}}/>);
    expect(await screen.findByText("Security evidence")).toBeTruthy();
    expect(screen.getByText("2 required fields awaiting review")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Review Security evidence response" }));
    expect(screen.getByRole("region", { name: "Assess submitted-2" })).toBeTruthy();
    expect(api.loadVendorForms).toHaveBeenCalledWith("r1", expect.objectContaining({ limit: 25 }));
  });
  it("shows only saved progress labels for unsubmitted work and sends filters to the server", async () => {
    api.loadVendorForms.mockResolvedValue({ items: [{ ...row, response_id: undefined, response_state: "IN_PROGRESS", required_count: null, answered_required: null, missing_fields: [{ id: "report", label: "Vulnerability report" }] }], observed_at: "2026-09-08T12:00:00Z" });
    render(<VendorFormsPanel relationshipID="r1" serviceName="Payments" onRequestForm={() => {}}/>);
    expect(await screen.findByText("Saved response progress unknown")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Review Security evidence response" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: /Form work to show/ }));
    fireEvent.click(screen.getByRole("option", { name: "Awaiting bank review" }));
    await waitFor(() => expect(api.loadVendorForms).toHaveBeenLastCalledWith("r1", expect.objectContaining({ filter: "AWAITING_REVIEW" })));
  });
});
