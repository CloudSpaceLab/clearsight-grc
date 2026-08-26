import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../http";
import { loadFormTemplates } from "../monitoringApi";
import type { FormTemplate } from "../monitoringTypes";
import { loadCurrentVendorAssessment, sendVendorAssessmentRequest, startVendorAssessment } from "../vendorAssessmentApi";
import type { VendorAssessment } from "../vendorAssessmentTypes";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import { createVendorRelationship, loadVendorRelationship, loadVendorRelationships, updateVendorRelationship } from "../vendorApi";
import { VendorsWorkspace } from "./VendorsWorkspace";

vi.mock("../vendorApi", () => ({
  createVendorRelationship: vi.fn(),
  loadVendorRelationship: vi.fn(),
  loadVendorRelationships: vi.fn(),
  updateVendorRelationship: vi.fn(),
}));

vi.mock("../monitoringApi", () => ({ loadFormTemplates: vi.fn() }));
vi.mock("../vendorAssessmentApi", () => ({
  loadCurrentVendorAssessment: vi.fn(),
  sendVendorAssessmentRequest: vi.fn(),
  startVendorAssessment: vi.fn(),
}));

const record: VendorRelationshipAggregate = {
  vendor: { id: "vendor-1", tenant_id: "bank", legal_name: "Acme Processing Limited", trading_name: "Acme", registration_ref: "RC-10001", jurisdiction: "Nigeria", source_id: "procurement", external_ref: "vendor-10001", status: "ACTIVE", created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z", version: 1 },
  relationship: { id: "relationship-1", tenant_id: "bank", legal_entity_id: "entity", vendor_id: "vendor-1", service_name: "Card transaction processing", business_owner_principal_id: "owner-1", criticality: "IMPORTANT", privacy_role: "PROCESSOR", status: "PROPOSED", created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z", version: 1 },
};

const activeVendorForm: FormTemplate = {
  id: "form-1", tenant_id: "bank", code: "VENDOR-DUE-DILIGENCE", name: "Vendor security and privacy review",
  purpose: "Collect the information required for the vendor review.", presentation: { default_mode: "WIZARD", allow_mode_switch: true }, sections: [], fields: [],
  status: "ACTIVE", is_current: true, version: 3, created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z",
};

function assessment(status: VendorAssessment["status"]): VendorAssessment {
  return {
    id: "assessment-1", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "relationship-1",
    review_kind: "ONBOARDING", stable_episode_key: "episode-1", status, form_template_id: "form-1", form_template_version: 3,
    review_due_at: "2099-09-30T23:59:59Z", started_by_principal_id: "owner-1", started_at: "2026-08-26T10:00:00Z",
    review_matter_id: status === "SETUP_PENDING" ? undefined : "matter-1", current_request_id: status === "COLLECTING" ? "request-1" : undefined,
    version: 3, created_at: "2026-08-26T10:00:00Z", updated_at: "2026-08-28T12:00:00Z",
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [record] });
  vi.mocked(loadVendorRelationship).mockResolvedValue(record);
  vi.mocked(loadFormTemplates).mockResolvedValue([activeVendorForm]);
  vi.mocked(loadCurrentVendorAssessment).mockRejectedValue(new ApiError(404, "Not found"));
});

describe("VendorsWorkspace", () => {
  it("shows the scoped vendor register and record details", async () => {
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria"/>);
    expect(await screen.findByRole("heading", { name: "Vendors" })).toBeTruthy();
    fireEvent.click(await screen.findByRole("button", { name: /Acme Processing Limited/ }));
    expect(screen.getByText("Card transaction processing")).toBeTruthy();
    expect(screen.getByText("owner-1")).toBeTruthy();
    expect(screen.getByText("Version 1")).toBeTruthy();
    expect(await screen.findByRole("heading", { name: "Due diligence" })).toBeTruthy();
  });

  it("treats a scoped missing assessment as not started and selects only the current active vendor form", async () => {
    vi.mocked(loadFormTemplates).mockResolvedValue([
      { ...activeVendorForm, id: "draft-form", status: "DRAFT", version: 4 },
      { ...activeVendorForm, id: "other-form", code: "ACCESS-REVIEW", name: "Access review" },
      activeVendorForm,
    ]);
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    expect(await screen.findByText("No due diligence review has been started for this vendor relationship.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Start due diligence" }));
    expect(screen.getByText("Vendor security and privacy review")).toBeTruthy();
    expect(screen.queryByText("Access review")).toBeNull();
    expect(loadCurrentVendorAssessment).toHaveBeenCalledWith("relationship-1");
    expect(screen.getAllByRole("button").filter((button) => button.classList.contains("primary-button") && !(button as HTMLButtonElement).disabled)).toHaveLength(1);
  });

  it("shows an assessment load failure instead of presenting a new assessment", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockRejectedValue(new ApiError(503, "Unavailable"));
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    expect((await screen.findByRole("alert")).textContent).toContain("Due diligence is unavailable");
    expect(screen.queryByRole("button", { name: "Start due diligence" })).toBeNull();
  });

  it("reloads approved forms without leaving the selected vendor", async () => {
    vi.mocked(loadFormTemplates)
      .mockRejectedValueOnce(new ApiError(503, "Unavailable"))
      .mockResolvedValueOnce([activeVendorForm]);
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    fireEvent.click(await screen.findByRole("button", { name: "Reload forms" }));
    expect(await screen.findByRole("button", { name: "Start due diligence" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Acme Processing Limited" })).toBeTruthy();
    expect(loadFormTemplates).toHaveBeenCalledTimes(2);
  });

  it("starts due diligence with the selected relationship and exact current form", async () => {
    vi.mocked(startVendorAssessment).mockResolvedValue(assessment("SETUP_PENDING"));
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    await screen.findByText("No due diligence review has been started for this vendor relationship.");
    fireEvent.click(screen.getByRole("button", { name: "Start due diligence" }));
    fireEvent.change(screen.getByLabelText("Review due date"), { target: { value: "2099-09-30" } });
    fireEvent.click(screen.getByRole("button", { name: "Start due diligence" }));

    await waitFor(() => expect(startVendorAssessment).toHaveBeenCalledWith("relationship-1", {
      relationship_version: 1, form_template_id: "form-1", form_template_version: 3, review_due_at: "2099-09-30T23:59:59.000Z",
    }));
    expect(await screen.findByText("Setup in progress")).toBeTruthy();
  });

  it("refreshes setup status and translates a terminal failure code into a safe recovery", async () => {
    vi.mocked(loadCurrentVendorAssessment)
      .mockResolvedValueOnce({ assessment: assessment("SETUP_PENDING"), setup: { assessment_id: "assessment-1", state: "LEASED" } })
      .mockResolvedValueOnce({ assessment: assessment("SETUP_PENDING"), setup: { assessment_id: "assessment-1", state: "FAILED", failure_code: "MATTER_CREATE_FAILED" } });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    fireEvent.click(await screen.findByRole("button", { name: "View setup status" }));
    expect(await screen.findByText("The review work item could not be created. Ask an administrator to retry assessment setup.")).toBeTruthy();
    expect(loadCurrentVendorAssessment).toHaveBeenCalledTimes(2);
  });

  it("opens the exact current evidence request when collection is in progress", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("COLLECTING") });
    const onOpenRequest = vi.fn();
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1" onOpenRequest={onOpenRequest}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Review request status" }));
    expect(onOpenRequest).toHaveBeenCalledWith("request-1");
  });

  it("sends the vendor request and preserves a truthful partial-delivery recovery", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("READY_TO_SEND") });
    vi.mocked(sendVendorAssessmentRequest).mockResolvedValue({
      assessment: { ...assessment("COLLECTING"), current_request_id: "request-1" },
      request: { id: "request-1", status: "READY", deadline: "2099-09-20T23:59:59Z" },
      state: "LINK_CREATED_EMAIL_NOT_SENT", recovery: "Copy the secure link or retry delivery.", capture_url: "https://capture.example.test/?capture_invite=secret",
    });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1" onOpenRequest={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Send due diligence request" }));
    fireEvent.change(screen.getByLabelText("Vendor contact email"), { target: { value: "security@vendor.example" } });
    fireEvent.change(screen.getByLabelText("Response due date"), { target: { value: "2099-09-20" } });
    fireEvent.click(screen.getByRole("button", { name: "Send due diligence request" }));

    expect((await screen.findByRole("alert")).textContent).toContain("Email delivery did not complete");
    expect(sendVendorAssessmentRequest).toHaveBeenCalledWith("assessment-1", expect.objectContaining({ audience: "security@vendor.example", expected_version: 3 }));
    expect(screen.queryByText("security@vendor.example")).toBeNull();
  });

  it("states the exact empty population and next action", async () => {
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [] });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria"/>);
    expect(await screen.findByText("No vendor relationships found for Clear Bank Nigeria.")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Add vendor" })).toBeTruthy();
  });

  it("creates a vendor relationship without browser-supplied identity", async () => {
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [] });
    vi.mocked(createVendorRelationship).mockResolvedValue(record);
    const onTarget = vi.fn();
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onTarget={onTarget}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Add vendor" }));
    fireEvent.change(screen.getByLabelText("Legal name"), { target: { value: "Acme Processing Limited" } });
    fireEvent.change(screen.getByLabelText("Service supplied"), { target: { value: "Card transaction processing" } });
    fireEvent.change(screen.getByLabelText("Criticality"), { target: { value: "IMPORTANT" } });
    fireEvent.change(screen.getByLabelText("Privacy role"), { target: { value: "PROCESSOR" } });
    fireEvent.click(screen.getByRole("button", { name: "Add vendor relationship" }));
    await waitFor(() => expect(createVendorRelationship).toHaveBeenCalled());
    const call = vi.mocked(createVendorRelationship).mock.calls[0];
    if (!call) throw new Error("createVendorRelationship was not called");
    expect(call[0]).toEqual(expect.not.objectContaining({ tenant_id: expect.anything(), legal_entity_id: expect.anything(), actor_id: expect.anything() }));
    expect(onTarget).toHaveBeenCalledWith("relationship-1");
    expect(await screen.findByText("Vendor relationship added.")).toBeTruthy();
  });

  it("preserves entered values when a concurrent update wins", async () => {
    vi.mocked(updateVendorRelationship).mockRejectedValue(new ApiError(409, "This vendor relationship changed."));
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor relationship" }));
    expect(screen.queryByLabelText("Legal name")).toBeNull();
    expect(screen.getByText("These details are shared across the bank and cannot be changed from this service relationship.")).toBeTruthy();
    const service = screen.getByLabelText("Service supplied") as HTMLInputElement;
    fireEvent.change(service, { target: { value: "Card processing and settlement" } });
    fireEvent.click(screen.getByRole("button", { name: "Save vendor relationship" }));
    expect(await screen.findByText("This record changed. Your entries are still here; reload the record before saving again.")).toBeTruthy();
    expect(service.value).toBe("Card processing and settlement");
  });

  it("updates only relationship-scoped fields", async () => {
    vi.mocked(updateVendorRelationship).mockResolvedValue({ ...record, relationship: { ...record.relationship, service_name: "Card settlement", version: 2 } });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor relationship" }));
    fireEvent.change(screen.getByLabelText("Service supplied"), { target: { value: "Card settlement" } });
    fireEvent.click(screen.getByRole("button", { name: "Save vendor relationship" }));
    await waitFor(() => expect(updateVendorRelationship).toHaveBeenCalled());
    const call = vi.mocked(updateVendorRelationship).mock.calls[0];
    if (!call) throw new Error("updateVendorRelationship was not called");
    expect(call[1]).toEqual({
      expected_version: 1, service_name: "Card settlement", criticality: "IMPORTANT", privacy_role: "PROCESSOR",
      effective_from: undefined, renewal_at: undefined,
    });
    expect(call[1]).toEqual(expect.not.objectContaining({ legal_name: expect.anything(), registration_ref: expect.anything() }));
  });
});
