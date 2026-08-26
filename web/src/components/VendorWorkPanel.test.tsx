import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadFormTemplates } from "../monitoringApi";
import { loadVendorRelationship } from "../vendorApi";
import { loadVendorRelationshipLinks } from "../vendorLinkApi";
import {
  acceptVendorWork, cancelVendorWork, loadVendorWork, prepareVendorWork, requestVendorWorkChanges,
  retryVendorWorkDelivery, sendVendorWork, startVendorWorkReview,
} from "../vendorWorkApi";
import type { VendorWorkRequest } from "../vendorWorkTypes";
import { VendorWorkPanel } from "./VendorWorkPanel";

vi.mock("../monitoringApi", () => ({ loadFormTemplates: vi.fn() }));
vi.mock("../vendorApi", () => ({ loadVendorRelationship: vi.fn() }));
vi.mock("../vendorLinkApi", () => ({ loadVendorRelationshipLinks: vi.fn() }));
vi.mock("../vendorWorkApi", () => ({
  acceptVendorWork: vi.fn(), cancelVendorWork: vi.fn(), loadVendorWork: vi.fn(), prepareVendorWork: vi.fn(),
  requestVendorWorkChanges: vi.fn(), retryVendorWorkDelivery: vi.fn(), sendVendorWork: vi.fn(), startVendorWorkReview: vi.fn(),
}));

const relationship = {
  vendor: { id: "vendor-1", tenant_id: "bank", legal_name: "Acme Processing Limited", status: "ACTIVE" as const, created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z", version: 1 },
  relationship: { id: "relationship-1", tenant_id: "bank", legal_entity_id: "entity", vendor_id: "vendor-1", service_name: "Card transaction processing", business_owner_principal_id: "owner", criticality: "IMPORTANT" as const, privacy_role: "PROCESSOR" as const, status: "ACTIVE" as const, created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z", version: 3 },
};

const link = { id: "link-1", relationship_id: "relationship-1", target_type: "PROGRAM" as const, target_id: "program-1", purpose_code: "SERVICE_SUPPORT", purpose_label: "Service support", state: "ACTIVE" as const, version: 1 };
const form = {
  id: "form-1", tenant_id: "bank", code: "VENDOR-CONTROL", name: "Vendor control confirmation", purpose: "Confirm current service controls.",
  status: "ACTIVE" as const, is_current: true, version: 4, created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z",
  presentation: { default_mode: "AUTOMATIC" as const, allow_mode_switch: true },
  sections: [{ id: "service", title: "Service controls", description: "Current controls for the service." }],
  fields: [
    { id: "control_confirmed", section_id: "service", label: "Are the controls operating?", type: "yes_no" as const, required: true },
    { id: "test_report", section_id: "service", label: "Current test report", type: "file" as const, required: true, accepted_formats: ["application/pdf"] },
  ],
};

const work: VendorWorkRequest = {
  id: "work-1", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "relationship-1", relationship_link_id: "link-1",
  target_type: "PROGRAM", target_id: "program-1", purpose: "Confirm annual resilience controls", instructions: "Complete the form and attach the current report.",
  owner_principal_id: "owner", form_template_id: "form-1", form_template_version: 4, presentation: "WIZARD", current_request_id: "request-1",
  current_capture_sequence: 1, state: "PREPARING", delivery_state: "NOT_SENT", due_at: "2026-09-30T17:00:00Z", version: 2,
  created_at: "2026-08-26T10:00:00Z", updated_at: "2026-08-26T10:01:00Z",
};

describe("VendorWorkPanel", () => {
  beforeEach(() => {
    vi.mocked(loadVendorRelationshipLinks).mockReset().mockResolvedValue({ items: [link] });
    vi.mocked(loadVendorRelationship).mockReset().mockResolvedValue(relationship);
    vi.mocked(loadFormTemplates).mockReset().mockResolvedValue([form]);
    vi.mocked(loadVendorWork).mockReset().mockResolvedValue({ items: [] });
    vi.mocked(prepareVendorWork).mockReset();
    vi.mocked(sendVendorWork).mockReset();
    vi.mocked(startVendorWorkReview).mockReset();
    vi.mocked(requestVendorWorkChanges).mockReset();
    vi.mocked(acceptVendorWork).mockReset();
    vi.mocked(cancelVendorWork).mockReset();
    vi.mocked(retryVendorWorkDelivery).mockReset();
  });

  it("prepares and sends vendor work with typed inputs and a real rendering choice", async () => {
    vi.mocked(prepareVendorWork).mockResolvedValue(work);
    vi.mocked(sendVendorWork).mockResolvedValue({ work: { ...work, state: "AWAITING_VENDOR", delivery_state: "DELIVERED", version: 3 }, state: "DELIVERED" });
    render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);

    await screen.findByText("No vendor requests have been recorded for this Program.");
    fireEvent.click(screen.getByRole("button", { name: "Request vendor work" }));
    fireEvent.change(screen.getByLabelText("Vendor relationship"), { target: { value: "link-1" } });
    fireEvent.change(screen.getByLabelText("Request purpose"), { target: { value: work.purpose } });
    fireEvent.change(screen.getByLabelText("Instructions for the vendor"), { target: { value: work.instructions } });
    fireEvent.change(screen.getByLabelText("Collection form"), { target: { value: "form-1:4" } });
    fireEvent.change(screen.getByLabelText("Form layout"), { target: { value: "WIZARD" } });
    expect((screen.getByLabelText("Vendor contact") as HTMLInputElement).type).toBe("email");
    fireEvent.change(screen.getByLabelText("Vendor contact"), { target: { value: "assurance@vendor.example" } });
    fireEvent.change(screen.getByLabelText("Due date"), { target: { value: "2026-09-30" } });

    expect(screen.getByText("2 fields · 2 required · 1 document upload")).toBeTruthy();
    expect(screen.getByText("Known vendor and service details will be shown with this request.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Prepare and send request" }));

    await waitFor(() => expect(prepareVendorWork).toHaveBeenCalledWith("relationship-1", expect.objectContaining({
      relationship_link_id: "link-1", form_template_id: "form-1", form_template_version: 4, presentation: "WIZARD",
      vendor_audience: "assurance@vendor.example", due_at: "2026-09-30T23:59:59.000Z",
    })));
    expect(sendVendorWork).toHaveBeenCalledWith("relationship-1", "work-1", { expected_version: 2, vendor_audience: "assurance@vendor.example", invitation_ttl_minutes: 10080 });
    expect(await screen.findByText("Waiting for vendor")).toBeTruthy();
  });

  it("keeps entered values when preparation fails", async () => {
    vi.mocked(prepareVendorWork).mockRejectedValue(new Error("unavailable"));
    render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);
    await screen.findByText("No vendor requests have been recorded for this Program.");
    fireEvent.click(screen.getByRole("button", { name: "Request vendor work" }));
    fireEvent.change(screen.getByLabelText("Vendor relationship"), { target: { value: "link-1" } });
    fireEvent.change(screen.getByLabelText("Request purpose"), { target: { value: work.purpose } });
    fireEvent.change(screen.getByLabelText("Instructions for the vendor"), { target: { value: work.instructions } });
    fireEvent.change(screen.getByLabelText("Collection form"), { target: { value: "form-1:4" } });
    fireEvent.change(screen.getByLabelText("Vendor contact"), { target: { value: "assurance@vendor.example" } });
    fireEvent.change(screen.getByLabelText("Due date"), { target: { value: "2026-09-30" } });
    fireEvent.click(screen.getByRole("button", { name: "Prepare and send request" }));

    expect((await screen.findByRole("alert")).textContent).toContain("The vendor request could not be prepared");
    expect((screen.getByLabelText("Request purpose") as HTMLInputElement).value).toBe(work.purpose);
    expect((screen.getByLabelText("Vendor contact") as HTMLInputElement).value).toBe("assurance@vendor.example");
  });

  it("shows the current request action and keeps completed requests in history", async () => {
    vi.mocked(loadVendorWork).mockResolvedValue({ items: [
      { ...work, id: "work-current", state: "RESPONSE_RECEIVED", delivery_state: "DELIVERED", version: 4 },
      { ...work, id: "work-history", state: "ACCEPTED", delivery_state: "DELIVERED", review_rationale: "Reviewed and accepted.", version: 7 },
    ] });
    vi.mocked(startVendorWorkReview).mockResolvedValue({ ...work, id: "work-current", state: "UNDER_REVIEW", delivery_state: "DELIVERED", version: 5 });
    const onOpenRequest = vi.fn();
    render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1" onOpenRequest={onOpenRequest}/>);

    const current = await screen.findByTestId("vendor-work-work-current");
    expect(within(current).getByText("Response received")).toBeTruthy();
    fireEvent.click(within(current).getByRole("button", { name: "Review response" }));
    await waitFor(() => expect(startVendorWorkReview).toHaveBeenCalledWith("relationship-1", "work-current", { expected_version: 4 }));
    expect(onOpenRequest).toHaveBeenCalledWith("request-1");
    expect(await within(current).findByText("Under review")).toBeTruthy();
    expect(screen.getByText("Request history")).toBeTruthy();
    expect(screen.getByText("Accepted")).toBeTruthy();
  });

  it("exposes a secure link only when delivery returned one", async () => {
    vi.mocked(prepareVendorWork).mockResolvedValue(work);
    vi.mocked(sendVendorWork).mockResolvedValue({ work: { ...work, state: "AWAITING_VENDOR", delivery_state: "LINK_CREATED_EMAIL_NOT_SENT", version: 3 }, state: "LINK_CREATED_EMAIL_NOT_SENT", capture_url: "https://capture.example.test/respond?invitation=secret" });
    render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);
    await screen.findByText("No vendor requests have been recorded for this Program.");
    fireEvent.click(screen.getByRole("button", { name: "Request vendor work" }));
    fireEvent.change(screen.getByLabelText("Vendor relationship"), { target: { value: "link-1" } });
    fireEvent.change(screen.getByLabelText("Request purpose"), { target: { value: work.purpose } });
    fireEvent.change(screen.getByLabelText("Instructions for the vendor"), { target: { value: work.instructions } });
    fireEvent.change(screen.getByLabelText("Collection form"), { target: { value: "form-1:4" } });
    fireEvent.change(screen.getByLabelText("Vendor contact"), { target: { value: "assurance@vendor.example" } });
    fireEvent.change(screen.getByLabelText("Due date"), { target: { value: "2026-09-30" } });
    fireEvent.click(screen.getByRole("button", { name: "Prepare and send request" }));

    expect(await screen.findByRole("button", { name: "Copy secure link" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Prepare and send request" })).toBeNull();
  });

  it("supports delivery recovery without recreating the request", async () => {
    vi.mocked(loadVendorWork).mockResolvedValue({ items: [{ ...work, delivery_state: "RETRY_REQUIRED", recovery: "Retry sending this vendor request." }] });
    vi.mocked(retryVendorWorkDelivery).mockResolvedValue({ work: { ...work, state: "AWAITING_VENDOR", delivery_state: "DELIVERED", version: 3 }, state: "DELIVERED" });
    render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);
    const card = await screen.findByTestId("vendor-work-work-1");
    fireEvent.change(within(card).getByLabelText("Vendor contact"), { target: { value: "assurance@vendor.example" } });
    fireEvent.click(within(card).getByRole("button", { name: "Retry delivery" }));
    await waitFor(() => expect(retryVendorWorkDelivery).toHaveBeenCalledWith("relationship-1", "work-1", { expected_version: 2, vendor_audience: "assurance@vendor.example", invitation_ttl_minutes: 10080 }));
  });

  it("requires review reasons for changes, acceptance and cancellation", async () => {
    vi.mocked(loadVendorWork).mockResolvedValue({ items: [{ ...work, state: "UNDER_REVIEW", delivery_state: "DELIVERED", version: 5 }] });
    vi.mocked(acceptVendorWork).mockResolvedValue({ ...work, state: "ACCEPTED", delivery_state: "DELIVERED", version: 6 });
    vi.mocked(requestVendorWorkChanges).mockResolvedValue({ work: { ...work, state: "CHANGES_REQUESTED", delivery_state: "DELIVERED", version: 7 }, state: "DELIVERED" });
    vi.mocked(cancelVendorWork).mockResolvedValue({ ...work, state: "CANCELLED", delivery_state: "DELIVERED", version: 6 });
    render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);
    const card = await screen.findByTestId("vendor-work-work-1");

    fireEvent.click(within(card).getByRole("button", { name: "Accept response" }));
    expect((within(card).getByRole("button", { name: "Confirm acceptance" }) as HTMLButtonElement).disabled).toBe(true);
    fireEvent.change(within(card).getByLabelText("Acceptance basis"), { target: { value: "The submitted controls and document address this request." } });
    fireEvent.click(within(card).getByRole("button", { name: "Confirm acceptance" }));
    await waitFor(() => expect(acceptVendorWork).toHaveBeenCalledWith("relationship-1", "work-1", { expected_version: 5, rationale: "The submitted controls and document address this request." }));
  });

  it("loads additional request history with the bounded cursor", async () => {
    vi.mocked(loadVendorWork)
      .mockResolvedValueOnce({ items: [{ ...work, id: "work-current", state: "AWAITING_VENDOR" }], next_cursor: "next-work" })
      .mockResolvedValueOnce({ items: [{ ...work, id: "work-history", state: "CANCELLED", cancellation_reason: "No longer required." }] });
    render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);
    await screen.findByTestId("vendor-work-work-current");

    fireEvent.click(screen.getByRole("button", { name: "Load more vendor requests" }));

    expect(await screen.findByTestId("vendor-work-work-history")).toBeTruthy();
    expect(loadVendorWork).toHaveBeenLastCalledWith({ target_type: "PROGRAM", target_id: "program-1", cursor: "next-work", limit: 20 });
  });

  it("ignores preparation completed after the target changes", async () => {
    let finishPrepare!: (value: VendorWorkRequest) => void;
    vi.mocked(prepareVendorWork).mockImplementation(() => new Promise((resolve) => { finishPrepare = resolve; }));
    const view = render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);
    await screen.findByText("No vendor requests have been recorded for this Program.");
    fireEvent.click(screen.getByRole("button", { name: "Request vendor work" }));
    fireEvent.change(screen.getByLabelText("Vendor relationship"), { target: { value: "link-1" } });
    fireEvent.change(screen.getByLabelText("Request purpose"), { target: { value: work.purpose } });
    fireEvent.change(screen.getByLabelText("Instructions for the vendor"), { target: { value: work.instructions } });
    fireEvent.change(screen.getByLabelText("Collection form"), { target: { value: "form-1:4" } });
    fireEvent.change(screen.getByLabelText("Vendor contact"), { target: { value: "assurance@vendor.example" } });
    fireEvent.change(screen.getByLabelText("Due date"), { target: { value: "2026-09-30" } });
    fireEvent.click(screen.getByRole("button", { name: "Prepare and send request" }));

    view.rerender(<VendorWorkPanel targetType="MATTER" targetID="matter-2"/>);
    await waitFor(() => expect(loadVendorWork).toHaveBeenLastCalledWith({ target_type: "MATTER", target_id: "matter-2", limit: 20 }));
    finishPrepare(work);
    await waitFor(() => expect(sendVendorWork).not.toHaveBeenCalled());
    expect(screen.queryByTestId("vendor-work-work-1")).toBeNull();
  });
});
