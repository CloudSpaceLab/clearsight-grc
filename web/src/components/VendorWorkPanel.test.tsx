import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadFormTemplates } from "../monitoringApi";
import { loadVendorRelationship } from "../vendorApi";
import { loadVendorRelationshipLinks } from "../vendorLinkApi";
import {
  acceptVendorWork, cancelVendorWork, loadVendorWork, loadVendorWorkResponse, prepareVendorWork, requestVendorWorkChanges,
  retryVendorWorkDelivery, sendVendorWork, startVendorWorkReview,
} from "../vendorWorkApi";
import type { VendorWorkRequest } from "../vendorWorkTypes";
import { VendorWorkPanel } from "./VendorWorkPanel";

vi.mock("../monitoringApi", () => ({ loadFormTemplates: vi.fn() }));
vi.mock("../vendorApi", () => ({ loadVendorRelationship: vi.fn() }));
vi.mock("../vendorLinkApi", () => ({ loadVendorRelationshipLinks: vi.fn() }));
vi.mock("../vendorWorkApi", () => ({
  acceptVendorWork: vi.fn(), cancelVendorWork: vi.fn(), loadVendorWork: vi.fn(), loadVendorWorkResponse: vi.fn(), prepareVendorWork: vi.fn(),
  requestVendorWorkChanges: vi.fn(), retryVendorWorkDelivery: vi.fn(), sendVendorWork: vi.fn(), startVendorWorkReview: vi.fn(),
  vendorWorkDocumentURL: (relationshipID: string, workID: string, requestID: string, artifactID: string) => `/api/v1/vendors/${relationshipID}/work/${workID}/requests/${requestID}/documents/${artifactID}/open`,
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

const addressForm = {
  ...form, id: "form-address", code: "VENDOR-ADDRESS-VERIFICATION", name: "Verify vendor address", version: 1,
};
const certificationForm = {
  ...form, id: "form-certifications", code: "VENDOR-CERTIFICATION-REFRESH", name: "Submit current vendor certifications", version: 1,
};

const work: VendorWorkRequest = {
  id: "work-1", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "relationship-1", relationship_link_id: "link-1",
  target_type: "PROGRAM", target_id: "program-1", request_kind: "GENERAL", purpose: "Confirm annual resilience controls", instructions: "Complete the form and attach the current report.",
  owner_principal_id: "owner", form_template_id: "form-1", form_template_version: 4, presentation: "WIZARD", current_request_id: "request-1",
  current_capture_sequence: 1, state: "PREPARING", delivery_state: "NOT_SENT", due_at: "2026-09-30T17:00:00Z", version: 2,
  created_at: "2026-08-26T10:00:00Z", updated_at: "2026-08-26T10:01:00Z",
};

const response = {
  work: { ...work, state: "RESPONSE_RECEIVED" as const, version: 4 },
  request: { request_id: "request-1", status: "SUBMITTED", deadline: work.due_at, form_template_id: "form-1", form_template_version: 4, presentation: { default_mode: "WIZARD" as const, allow_mode_switch: true } },
  response: { submission_id: "submission-1", request_id: "request-1", submitted_at: "2026-09-20T12:00:00Z" },
  answers: [
    { field_id: "control_confirmed", label: "Are the controls operating?", type: "yes_no", required: true, visibility: "VISIBLE" as const, value: { text: "Yes" }, provenance: { origin: "SOURCE_PREFILLED", source_receipt: { source_id: "vendor-register", observed_at: "2026-09-18T09:00:00Z" } } },
    { field_id: "missing_owner", label: "Control owner", type: "short_text", required: true, visibility: "VISIBLE" as const },
    { field_id: "conditional_note", label: "Exception details", type: "long_text", required: true, visibility: "CONDITIONALLY_OMITTED" as const },
    { field_id: "regions", label: "Service regions", type: "multi_select", required: false, visibility: "VISIBLE" as const, value: { values: ["Nigeria", "Ghana"] }, provenance: { origin: "RESPONDENT_ENTERED" } },
  ],
  documents: [
    { field_id: "test_report", artifact_id: "artifact-available", file_name: "current-test-report.pdf", media_type: "application/pdf", size_bytes: 64000, artifact_status: "AVAILABLE", evidence_class: "VENDOR_SUPPLIED", document_type: "SECURITY_TEST" },
    { field_id: "test_report", artifact_id: "artifact-quarantined", file_name: "quarantined-report.pdf", media_type: "application/pdf", size_bytes: 32000, artifact_status: "QUARANTINED", evidence_class: "VENDOR_SUPPLIED", document_type: "SECURITY_TEST" },
  ],
};

async function chooseRequestOption(label: string, option: string) {
  const dialog = screen.getByRole("dialog", { name: "Request vendor work" });
  fireEvent.click(within(dialog).getByRole("button", { name: new RegExp(label, "i") }));
  fireEvent.click(await screen.findByRole("option", { name: option }));
  await waitFor(() => expect(screen.queryByRole("listbox")).toBeNull());
}

describe("VendorWorkPanel", () => {
  beforeEach(() => {
    vi.mocked(loadVendorRelationshipLinks).mockReset().mockResolvedValue({ items: [link] });
    vi.mocked(loadVendorRelationship).mockReset().mockResolvedValue(relationship);
    vi.mocked(loadFormTemplates).mockReset().mockResolvedValue([form]);
    vi.mocked(loadVendorWork).mockReset().mockResolvedValue({ items: [] });
    vi.mocked(loadVendorWorkResponse).mockReset().mockImplementation(async (_relationshipID, workID) => ({ ...response, work: { ...response.work, id: workID } }));
    vi.mocked(prepareVendorWork).mockReset();
    vi.mocked(sendVendorWork).mockReset();
    vi.mocked(startVendorWorkReview).mockReset();
    vi.mocked(requestVendorWorkChanges).mockReset();
    vi.mocked(acceptVendorWork).mockReset();
    vi.mocked(cancelVendorWork).mockReset();
    vi.mocked(retryVendorWorkDelivery).mockReset();
  });

  it("presents certification collection and acceptance as separate steps", async () => {
    vi.mocked(loadFormTemplates).mockResolvedValue([form, addressForm, certificationForm]);
    vi.mocked(loadVendorWork).mockResolvedValue({ items: [
      { ...work, id: "cert-pending", request_kind: "CERTIFICATION_REFRESH", state: "AWAITING_VENDOR", delivery_state: "DELIVERED" } as VendorWorkRequest,
      { ...work, id: "cert-response", request_kind: "CERTIFICATION_REFRESH", state: "RESPONSE_RECEIVED", delivery_state: "DELIVERED", version: 4 } as VendorWorkRequest,
      { ...work, id: "accepted", request_kind: "CERTIFICATION_REFRESH", state: "ACCEPTED", delivery_state: "DELIVERED", review_rationale: "The certification evidence is current and complete." } as VendorWorkRequest,
    ] });
    render(<VendorWorkPanel targetType="MATTER" targetID="matter-1"/>);

    expect(await screen.findByText("Certification evidence received")).toBeTruthy();
    expect(screen.getByText("Evidence accepted")).toBeTruthy();
    expect(screen.queryByText("Matter resolved")).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Request vendor work" }));
    await chooseRequestOption("Request type", "ISO 27001 and PCI DSS evidence");
    expect(screen.getByText(/asked for current ISO 27001 and PCI DSS evidence/i)).toBeTruthy();
    expect(screen.getByText(/submission does not mean the bank accepted it/i)).toBeTruthy();
    expect(screen.getByLabelText(/Vendor contact email/)).toBeTruthy();
    expect(screen.getByRole("button", { name: "Send certification request" })).toBeTruthy();
  });

  it("prepares and sends vendor work with typed inputs and a real rendering choice", async () => {
    vi.mocked(prepareVendorWork).mockResolvedValue(work);
    vi.mocked(sendVendorWork).mockResolvedValue({ work: { ...work, state: "AWAITING_VENDOR", delivery_state: "DELIVERED", version: 3 }, state: "DELIVERED" });
    render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);

    await screen.findByText("No vendor requests have been recorded for this Program.");
    fireEvent.click(screen.getByRole("button", { name: "Request vendor work" }));
    await chooseRequestOption("Vendor relationship", "Acme Processing Limited — Card transaction processing");
    fireEvent.change(screen.getByLabelText(/Request purpose/), { target: { value: work.purpose } });
    fireEvent.change(screen.getByLabelText(/Instructions for the vendor/), { target: { value: work.instructions } });
    await chooseRequestOption("Collection form", "Vendor control confirmation · version 4");
    await chooseRequestOption("Form layout", "Wizard");
    expect((screen.getByLabelText(/Vendor contact/) as HTMLInputElement).type).toBe("email");
    fireEvent.change(screen.getByLabelText(/Vendor contact/), { target: { value: "assurance@vendor.example" } });
    fireEvent.change(screen.getByLabelText(/Due date/), { target: { value: "2026-09-30" } });

    expect(screen.getByText("2 fields · 2 required · 1 document upload")).toBeTruthy();
    expect(screen.getByText(/Known vendor and service details will be shown with this request/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Prepare and send request" }));

    await waitFor(() => expect(prepareVendorWork).toHaveBeenCalledWith("relationship-1", expect.objectContaining({
      relationship_link_id: "link-1", request_kind: "GENERAL", form_template_id: "form-1", form_template_version: 4, presentation: "WIZARD",
      vendor_audience: "assurance@vendor.example", due_at: "2026-09-30T23:59:59.000Z",
    })));
    expect(sendVendorWork).toHaveBeenCalledWith("relationship-1", "work-1", { expected_version: 2, vendor_audience: "assurance@vendor.example", invitation_ttl_minutes: 10080 });
    expect(await screen.findByText("Waiting for vendor")).toBeTruthy();
  });

  it("keeps an in-flight request open and prevents duplicate preparation", async () => {
    let finishPrepare!: (value: VendorWorkRequest) => void;
    vi.mocked(prepareVendorWork).mockImplementation(() => new Promise((resolve) => { finishPrepare = resolve; }));
    vi.mocked(sendVendorWork).mockResolvedValue({ work: { ...work, state: "AWAITING_VENDOR", delivery_state: "DELIVERED", version: 3 }, state: "DELIVERED" });
    render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);

    await screen.findByText("No vendor requests have been recorded for this Program.");
    fireEvent.click(screen.getByRole("button", { name: "Request vendor work" }));
    await chooseRequestOption("Vendor relationship", "Acme Processing Limited — Card transaction processing");
    fireEvent.change(screen.getByLabelText(/Request purpose/), { target: { value: work.purpose } });
    fireEvent.change(screen.getByLabelText(/Instructions for the vendor/), { target: { value: work.instructions } });
    await chooseRequestOption("Collection form", "Vendor control confirmation · version 4");
    fireEvent.change(screen.getByLabelText(/Vendor contact/), { target: { value: "assurance@vendor.example" } });
    fireEvent.change(screen.getByLabelText(/Due date/), { target: { value: "2026-09-30" } });
    const submit = screen.getByRole("button", { name: "Prepare and send request" });
    fireEvent.click(submit);
    fireEvent.click(submit);

    await waitFor(() => expect(prepareVendorWork).toHaveBeenCalledTimes(1));
    const close = screen.getByRole("button", { name: "Vendor request is being sent" });
    expect((close as HTMLButtonElement).disabled).toBe(true);
    fireEvent.keyDown(screen.getByRole("dialog", { name: "Request vendor work" }), { key: "Escape" });
    fireEvent.mouseDown(document.querySelector(".cs-sheet__overlay") as HTMLElement);
    expect(screen.getByRole("dialog", { name: "Request vendor work" })).toBeTruthy();

    finishPrepare(work);
    expect(await screen.findByText("Waiting for vendor")).toBeTruthy();
    expect(sendVendorWork).toHaveBeenCalledTimes(1);
  });

  it("starts linked Program or issue work from the selected vendor relationship", async () => {
    vi.mocked(prepareVendorWork).mockResolvedValue(work);
    vi.mocked(sendVendorWork).mockResolvedValue({ work: { ...work, state: "AWAITING_VENDOR", delivery_state: "DELIVERED", version: 3 }, state: "DELIVERED" });
    render(<VendorWorkPanel relationshipID="relationship-1"/>);

    await screen.findByText("No vendor requests have been recorded for this vendor relationship.");
    expect(loadVendorRelationshipLinks).toHaveBeenCalledWith({ relationship_id: "relationship-1", limit: 50 });
    fireEvent.click(screen.getByRole("button", { name: "Request vendor work" }));
    await chooseRequestOption("Related Program or issue", "Program · Service support");
    fireEvent.change(screen.getByLabelText(/Request purpose/), { target: { value: work.purpose } });
    fireEvent.change(screen.getByLabelText(/Instructions for the vendor/), { target: { value: work.instructions } });
    await chooseRequestOption("Collection form", "Vendor control confirmation · version 4");
    fireEvent.change(screen.getByLabelText(/Vendor contact/), { target: { value: "assurance@vendor.example" } });
    fireEvent.change(screen.getByLabelText(/Due date/), { target: { value: "2026-09-30" } });
    fireEvent.click(screen.getByRole("button", { name: "Prepare and send request" }));
    await waitFor(() => expect(prepareVendorWork).toHaveBeenCalledWith("relationship-1", expect.objectContaining({ relationship_link_id: "link-1" })));
  });

  it("keeps entered values when preparation fails", async () => {
    vi.mocked(prepareVendorWork).mockRejectedValue(new Error("unavailable"));
    render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);
    await screen.findByText("No vendor requests have been recorded for this Program.");
    fireEvent.click(screen.getByRole("button", { name: "Request vendor work" }));
    await chooseRequestOption("Vendor relationship", "Acme Processing Limited — Card transaction processing");
    fireEvent.change(screen.getByLabelText(/Request purpose/), { target: { value: work.purpose } });
    fireEvent.change(screen.getByLabelText(/Instructions for the vendor/), { target: { value: work.instructions } });
    await chooseRequestOption("Collection form", "Vendor control confirmation · version 4");
    fireEvent.change(screen.getByLabelText(/Vendor contact/), { target: { value: "assurance@vendor.example" } });
    fireEvent.change(screen.getByLabelText(/Due date/), { target: { value: "2026-09-30" } });
    fireEvent.click(screen.getByRole("button", { name: "Prepare and send request" }));

    expect((await screen.findByRole("alert")).textContent).toContain("The vendor request could not be prepared");
    expect((screen.getByLabelText(/Request purpose/) as HTMLInputElement).value).toBe(work.purpose);
    expect((screen.getByLabelText(/Vendor contact/) as HTMLInputElement).value).toBe("assurance@vendor.example");
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
    await waitFor(() => expect(loadVendorWorkResponse).toHaveBeenCalledWith("relationship-1", "work-current"));
    expect(within(current).getByText("Vendor response")).toBeTruthy();
    expect(startVendorWorkReview).not.toHaveBeenCalled();
    fireEvent.click(within(current).getByRole("button", { name: "Begin review" }));
    await waitFor(() => expect(startVendorWorkReview).toHaveBeenCalledWith("relationship-1", "work-current", { expected_version: 4 }));
    fireEvent.click(within(current).getByRole("button", { name: "Open collection request" }));
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
    await chooseRequestOption("Vendor relationship", "Acme Processing Limited — Card transaction processing");
    fireEvent.change(screen.getByLabelText(/Request purpose/), { target: { value: work.purpose } });
    fireEvent.change(screen.getByLabelText(/Instructions for the vendor/), { target: { value: work.instructions } });
    await chooseRequestOption("Collection form", "Vendor control confirmation · version 4");
    fireEvent.change(screen.getByLabelText(/Vendor contact/), { target: { value: "assurance@vendor.example" } });
    fireEvent.change(screen.getByLabelText(/Due date/), { target: { value: "2026-09-30" } });
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

  it("labels incomplete setup separately from delivery recovery", async () => {
    const incomplete = { ...work, current_request_id: undefined, delivery_state: "RETRY_REQUIRED" as const, recovery: "Retry sending this vendor request." };
    vi.mocked(loadVendorWork).mockResolvedValue({ items: [incomplete] });
    vi.mocked(retryVendorWorkDelivery).mockResolvedValue({ work: { ...work, state: "AWAITING_VENDOR", delivery_state: "DELIVERED", version: 3 }, state: "DELIVERED" });
    render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);

    const card = await screen.findByTestId("vendor-work-work-1");
    expect(within(card).getByText("The collection request was not created. Complete setup to create and send it.")).toBeTruthy();
    expect(within(card).queryByText("Retry sending this vendor request.")).toBeNull();
    fireEvent.change(within(card).getByLabelText("Vendor contact"), { target: { value: "assurance@vendor.example" } });
    fireEvent.click(within(card).getByRole("button", { name: "Complete setup" }));

    await waitFor(() => expect(retryVendorWorkDelivery).toHaveBeenCalledWith("relationship-1", "work-1", { expected_version: 2, vendor_audience: "assurance@vendor.example", invitation_ttl_minutes: 10080 }));
  });

  it("requires review reasons for changes, acceptance and cancellation", async () => {
    vi.mocked(loadVendorWork).mockResolvedValue({ items: [{ ...work, state: "UNDER_REVIEW", delivery_state: "DELIVERED", version: 5 }] });
    vi.mocked(loadVendorWorkResponse).mockResolvedValue({ ...response, work: { ...response.work, state: "UNDER_REVIEW", version: 5 }, documents: response.documents.filter((document) => document.artifact_status === "AVAILABLE") });
    vi.mocked(acceptVendorWork).mockResolvedValue({ ...work, state: "ACCEPTED", delivery_state: "DELIVERED", version: 6 });
    vi.mocked(requestVendorWorkChanges).mockResolvedValue({ work: { ...work, state: "CHANGES_REQUESTED", delivery_state: "DELIVERED", version: 7 }, state: "DELIVERED" });
    vi.mocked(cancelVendorWork).mockResolvedValue({ ...work, state: "CANCELLED", delivery_state: "DELIVERED", version: 6 });
    render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);
    const card = await screen.findByTestId("vendor-work-work-1");

    fireEvent.click(within(card).getByRole("button", { name: "Review response" }));
    await within(card).findByText("Vendor response");
    fireEvent.click(within(card).getByRole("button", { name: "Accept response" }));
    expect((within(card).getByRole("button", { name: "Confirm acceptance" }) as HTMLButtonElement).disabled).toBe(true);
    fireEvent.change(within(card).getByLabelText("Acceptance basis"), { target: { value: "The submitted controls and document address this request." } });
    fireEvent.click(within(card).getByRole("button", { name: "Confirm acceptance" }));
    await waitFor(() => expect(acceptVendorWork).toHaveBeenCalledWith("relationship-1", "work-1", { expected_version: 5, rationale: "The submitted controls and document address this request." }));
  });

  it("distinguishes missing and conditional answers, formats typed values, and shows provenance", async () => {
    vi.mocked(loadVendorWork).mockResolvedValue({ items: [{ ...work, state: "RESPONSE_RECEIVED", version: 4 }] });
    render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);
    const card = await screen.findByTestId("vendor-work-work-1");
    fireEvent.click(within(card).getByRole("button", { name: "Review response" }));

    expect(await within(card).findByText("Vendor response: Yes")).toBeTruthy();
    expect(within(card).getByText("Required response missing")).toBeTruthy();
    expect(within(card).getByText("Not requested because its condition was not met")).toBeTruthy();
    expect(within(card).getByText("Vendor response: Nigeria, Ghana")).toBeTruthy();
    expect(within(card).getByText(/Prefilled from vendor-register/)).toBeTruthy();
    expect(within(card).getByText("Entered by the vendor")).toBeTruthy();
  });

  it("opens only available documents through the server-provided safe URL", async () => {
    vi.mocked(loadVendorWork).mockResolvedValue({ items: [{ ...work, state: "UNDER_REVIEW", version: 5 }] });
    const open = vi.spyOn(window, "open").mockImplementation(() => null);
    render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);
    const card = await screen.findByTestId("vendor-work-work-1");
    fireEvent.click(within(card).getByRole("button", { name: "Review response" }));
    const available = await within(card).findByRole("article", { name: "current-test-report.pdf" });
    const quarantined = within(card).getByRole("article", { name: "quarantined-report.pdf" });
    fireEvent.click(within(available).getByRole("button", { name: "Open document" }));

    expect(open).toHaveBeenCalledWith("/api/v1/vendors/relationship-1/work/work-1/requests/request-1/documents/artifact-available/open", "_blank", "noopener,noreferrer");
    expect(within(quarantined).queryByRole("button", { name: "Open document" })).toBeNull();
    expect(within(quarantined).getByText("This document is quarantined. Request a clean replacement before review.")).toBeTruthy();
    expect((within(card).getByRole("button", { name: "Accept response" }) as HTMLButtonElement).disabled).toBe(true);
    expect(within(card).getByText("Acceptance is unavailable while a submitted document is pending inspection, quarantined or unavailable. Wait for inspection or request a replacement.")).toBeTruthy();
    expect(card.textContent).not.toContain("artifact-quarantined");
  });

  it("keeps review unstarted when the response projection is unavailable", async () => {
    vi.mocked(loadVendorWork).mockResolvedValue({ items: [{ ...work, state: "RESPONSE_RECEIVED", version: 4 }] });
    vi.mocked(loadVendorWorkResponse).mockRejectedValue(new Error("unavailable"));
    render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);
    const card = await screen.findByTestId("vendor-work-work-1");
    fireEvent.click(within(card).getByRole("button", { name: "Review response" }));

    expect((await within(card).findByRole("alert")).textContent).toContain("The submitted response could not be loaded");
    expect(within(card).getByRole("button", { name: "Try again" })).toBeTruthy();
    expect(startVendorWorkReview).not.toHaveBeenCalled();
    expect(within(card).queryByRole("button", { name: "Accept response" })).toBeNull();
  });

  it("removes the reviewed response when a clarification becomes current", async () => {
    vi.mocked(loadVendorWork).mockResolvedValue({ items: [{ ...work, state: "UNDER_REVIEW", delivery_state: "DELIVERED", version: 5 }] });
    vi.mocked(requestVendorWorkChanges).mockResolvedValue({ work: { ...work, state: "CHANGES_REQUESTED", current_request_id: "request-2", submission_id: undefined, current_capture_sequence: 2, delivery_state: "DELIVERED", version: 6 }, state: "DELIVERED" });
    render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);
    const card = await screen.findByTestId("vendor-work-work-1");
    fireEvent.click(within(card).getByRole("button", { name: "Review response" }));
    expect(await within(card).findByText("Vendor response: Yes")).toBeTruthy();
    fireEvent.click(within(card).getByRole("button", { name: "Request changes" }));
    fireEvent.change(within(card).getByLabelText("What the vendor must change"), { target: { value: "Upload a clean replacement." } });
    fireEvent.click(within(card).getByLabelText("Are the controls operating?"));
    fireEvent.change(within(card).getByLabelText("Vendor contact"), { target: { value: "assurance@vendor.example" } });
    fireEvent.change(within(card).getByLabelText("Revised due date"), { target: { value: "2099-09-30" } });
    fireEvent.click(within(card).getByRole("button", { name: "Send change request" }));

    await waitFor(() => expect(requestVendorWorkChanges).toHaveBeenCalled());
    await waitFor(() => expect(within(card).queryByLabelText("Vendor response")).toBeNull());
  });

  it("removes a loaded response when the current submission changes", async () => {
    vi.mocked(loadVendorWork)
      .mockResolvedValueOnce({ items: [{ ...work, state: "RESPONSE_RECEIVED", version: 4 }], next_cursor: "updated-response" })
      .mockResolvedValueOnce({ items: [{ ...work, state: "RESPONSE_RECEIVED", submission_id: "submission-2", version: 5 }] });
    render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);
    const card = await screen.findByTestId("vendor-work-work-1");
    fireEvent.click(within(card).getByRole("button", { name: "Review response" }));
    expect(await within(card).findByText("Vendor response: Yes")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Load more vendor requests" }));

    await waitFor(() => expect(loadVendorWork).toHaveBeenLastCalledWith({ target_type: "PROGRAM", target_id: "program-1", cursor: "updated-response", limit: 20 }));
    await waitFor(() => expect(within(card).queryByLabelText("Vendor response")).toBeNull());
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

  it("loads additional linked vendors before preparing a request", async () => {
    const secondLink = { ...link, id: "link-2", relationship_id: "relationship-2" };
    const secondRelationship = { ...relationship, relationship: { ...relationship.relationship, id: "relationship-2", service_name: "Payment reconciliation" } };
    vi.mocked(loadVendorRelationshipLinks)
      .mockResolvedValueOnce({ items: [link], next_cursor: "next-link" })
      .mockResolvedValueOnce({ items: [secondLink] });
    vi.mocked(loadVendorRelationship).mockImplementation(async (id) => id === "relationship-2" ? secondRelationship : relationship);
    render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);
    await screen.findByText("No vendor requests have been recorded for this Program.");
    fireEvent.click(screen.getByRole("button", { name: "Request vendor work" }));

    fireEvent.click(screen.getByRole("button", { name: "Load more linked vendors" }));

    fireEvent.click(within(screen.getByRole("dialog", { name: "Request vendor work" })).getByRole("button", { name: /Vendor relationship/i }));
    expect(await screen.findByRole("option", { name: "Acme Processing Limited — Payment reconciliation" })).toBeTruthy();
    expect(loadVendorRelationshipLinks).toHaveBeenLastCalledWith({ target_type: "PROGRAM", target_id: "program-1", cursor: "next-link", limit: 50 });
  });

  it("ignores preparation completed after the target changes", async () => {
    let finishPrepare!: (value: VendorWorkRequest) => void;
    vi.mocked(prepareVendorWork).mockImplementation(() => new Promise((resolve) => { finishPrepare = resolve; }));
    const view = render(<VendorWorkPanel targetType="PROGRAM" targetID="program-1"/>);
    await screen.findByText("No vendor requests have been recorded for this Program.");
    fireEvent.click(screen.getByRole("button", { name: "Request vendor work" }));
    await chooseRequestOption("Vendor relationship", "Acme Processing Limited — Card transaction processing");
    fireEvent.change(screen.getByLabelText(/Request purpose/), { target: { value: work.purpose } });
    fireEvent.change(screen.getByLabelText(/Instructions for the vendor/), { target: { value: work.instructions } });
    await chooseRequestOption("Collection form", "Vendor control confirmation · version 4");
    fireEvent.change(screen.getByLabelText(/Vendor contact/), { target: { value: "assurance@vendor.example" } });
    fireEvent.change(screen.getByLabelText(/Due date/), { target: { value: "2026-09-30" } });
    fireEvent.click(screen.getByRole("button", { name: "Prepare and send request" }));

    view.rerender(<VendorWorkPanel targetType="MATTER" targetID="matter-2"/>);
    await waitFor(() => expect(loadVendorWork).toHaveBeenLastCalledWith({ target_type: "MATTER", target_id: "matter-2", limit: 20 }));
    finishPrepare(work);
    await waitFor(() => expect(sendVendorWork).not.toHaveBeenCalled());
    expect(screen.queryByTestId("vendor-work-work-1")).toBeNull();
  });
});
