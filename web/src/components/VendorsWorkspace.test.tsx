import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../http";
import { loadFormTemplates } from "../monitoringApi";
import type { FormTemplate } from "../monitoringTypes";
import { completeVendorAssessment, createVendorAssessmentDeficiency, loadCurrentVendorAssessment, loadVendorAssessment, reissueVendorAssessmentRequest, requestVendorAssessmentClarification, retryVendorAssessmentSetup, reviewVendorAssessmentDocument, sendVendorAssessmentRequest, startVendorAssessment, startVendorAssessmentReview, vendorAssessmentDocumentURL } from "../vendorAssessmentApi";
import type { VendorAssessment, VendorAssessmentReviewView } from "../vendorAssessmentTypes";
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
  completeVendorAssessment: vi.fn(),
  createVendorAssessmentDeficiency: vi.fn(),
  loadCurrentVendorAssessment: vi.fn(),
  loadVendorAssessment: vi.fn(),
  reissueVendorAssessmentRequest: vi.fn(),
  requestVendorAssessmentClarification: vi.fn(),
  retryVendorAssessmentSetup: vi.fn(),
  reviewVendorAssessmentDocument: vi.fn(),
  sendVendorAssessmentRequest: vi.fn(),
  startVendorAssessment: vi.fn(),
  startVendorAssessmentReview: vi.fn(),
  vendorAssessmentDocumentURL: vi.fn(),
}));
vi.mock("./VendorWorkPanel", () => ({ VendorWorkPanel: ({ relationshipID }: { relationshipID: string }) => <div data-testid={`vendor-work-relationship-${relationshipID}`}>Vendor requests</div> }));

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

function review(status: VendorAssessment["status"]): VendorAssessmentReviewView {
  return {
    assessment: {
      ...assessment(status),
      submission_id: "submission-1",
      submitted_at: "2026-08-28T11:00:00Z",
      review_started_at: status === "SUBMITTED" ? undefined : "2026-08-28T12:00:00Z",
      completed_at: status === "COMPLETED" ? "2026-08-29T12:00:00Z" : undefined,
      conclusion: status === "COMPLETED" ? "SATISFACTORY_WITH_CONDITIONS" : undefined,
      conclusion_rationale: status === "COMPLETED" ? "Proceed after the recorded access-control action is complete." : undefined,
      conclusion_uncertainty: status === "COMPLETED" ? "The next resilience exercise remains due." : undefined,
    },
    requests: [{ request_id: "request-1", purpose: "INITIAL", sequence: 1, origin_sequence: 1, status: "SUBMITTED", deadline: "2026-08-27T23:59:59Z", form_template_id: "form-1", form_template_version: 3 }],
    response: { submission_id: "submission-1", request_id: "request-1", submitted_at: "2026-08-28T11:00:00Z", answer_count: 2, artifact_count: 1 },
    answers: [
      { field_id: "security-testing", label: "Independent security testing", type: "YES_NO", required: true, visibility: "VISIBLE", value: { text: "Yes" }, provenance: { origin: "SOURCE_PREFILLED", binding_id: "binding-1", binding_version: 4, source_receipt: { source_id: "procurement", observed_at: "2026-08-27T09:30:00Z" } } },
      { field_id: "hidden-follow-up", label: "Follow-up detail", type: "LONG_TEXT", required: true, visibility: "CONDITIONALLY_OMITTED" },
    ],
    coverage: { visible_fields: 1, answered_fields: 1, required_fields: 1, answered_required: 1, ratio: 1 },
    documents: [{ field_id: "security-report", artifact_id: "artifact-1", file_name: "independent-security-test.pdf", media_type: "application/pdf", size_bytes: 64000, artifact_status: "AVAILABLE", evidence_class: "VENDOR_SUPPLIED", document_type: "SECURITY_TEST", expires_on: "2027-08-01" }],
    provisional_score: { score: 86, coverage: 1, rule_results: [] },
    matters: [{ matter_id: "matter-1", type: "VENDOR_DEFICIENCY", status: "OPEN", title: "Access-control evidence gap" }],
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
    expect(screen.getByTestId("vendor-work-relationship-relationship-1")).toBeTruthy();
  });

  it("names each relationship with its vendor and service", async () => {
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria"/>);

    expect(await screen.findByRole("button", { name: "Acme Processing Limited, Card transaction processing" })).toBeTruthy();
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
    expect(await screen.findByText("The review work item could not be created. Retry assessment setup; no duplicate review will be created.")).toBeTruthy();
    expect(loadCurrentVendorAssessment).toHaveBeenCalledTimes(2);
  });

  it("retries failed setup with the current version and shows the queued setup state", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("SETUP_PENDING"), setup: { assessment_id: "assessment-1", state: "FAILED", attempts: 3, failure_code: "MATTER_CREATE_FAILED" } });
    vi.mocked(retryVendorAssessmentSetup).mockResolvedValue({
      assessment: { ...assessment("SETUP_PENDING"), version: 4 },
      setup: { assessment_id: "assessment-1", state: "READY", attempts: 0, next_attempt_at: "2026-08-26T10:10:00Z" },
    });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    fireEvent.click(await screen.findByRole("button", { name: "Retry due diligence setup" }));

    await waitFor(() => expect(retryVendorAssessmentSetup).toHaveBeenCalledWith("assessment-1", { expected_version: 3 }));
    expect(await screen.findByText("Assessment setup queued. Review setup will continue in the background.")).toBeTruthy();
    expect(screen.getByText("Setup in progress")).toBeTruthy();
    expect(screen.getByRole("button", { name: "View setup status" })).toBeTruthy();
    expect(screen.getAllByRole("button").filter((button) => button.classList.contains("primary-button") && !(button as HTMLButtonElement).disabled)).toHaveLength(1);
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

  it("reissues the current request without replacing the request-status action", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("COLLECTING") });
    vi.mocked(reissueVendorAssessmentRequest).mockResolvedValue({
      assessment: { ...assessment("COLLECTING"), version: 4 }, request: { id: "request-1", status: "READY" }, state: "DELIVERED",
    });
    const onOpenRequest = vi.fn();
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1" onOpenRequest={onOpenRequest}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Send replacement link" }));
    fireEvent.change(screen.getByLabelText("Vendor contact email"), { target: { value: "security@vendor.example" } });
    fireEvent.click(screen.getByRole("button", { name: "Send replacement link" }));

    await waitFor(() => expect(reissueVendorAssessmentRequest).toHaveBeenCalledWith("assessment-1", {
      expected_version: 3, audience: "security@vendor.example", invitation_ttl_minutes: 1440,
    }));
    expect(await screen.findByText("Replacement link sent. Previous access to this request has ended.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Review request status" }));
    expect(onOpenRequest).toHaveBeenCalledWith("request-1");
  });

  it("loads the submitted response and starts review with the current assessment version", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("SUBMITTED") });
    vi.mocked(loadVendorAssessment).mockResolvedValue(review("SUBMITTED"));
    vi.mocked(startVendorAssessmentReview).mockResolvedValue({ ...assessment("UNDER_REVIEW"), version: 4 });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    expect(await screen.findByText(/Independent security testing/)).toBeTruthy();
    expect(screen.getByText("1 of 1 required answers received")).toBeTruthy();
    expect(screen.getByText("Provisional score: 86 of 100 · Form version 3")).toBeTruthy();
    expect(screen.getByText("Prefilled from procurement · Observed 27 Aug 2026")).toBeTruthy();
    expect(screen.getByText("Scan complete · Vendor supplied evidence")).toBeTruthy();
    fireEvent.click(await screen.findByRole("button", { name: "Review vendor response" }));

    await waitFor(() => expect(startVendorAssessmentReview).toHaveBeenCalledWith("assessment-1", { expected_version: 3 }));
    expect(await screen.findByRole("button", { name: "Record assessment conclusion" })).toBeTruthy();
  });

  it("records the reviewer conclusion without changing the vendor relationship", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("UNDER_REVIEW") });
    vi.mocked(loadVendorAssessment).mockResolvedValue(review("UNDER_REVIEW"));
    vi.mocked(completeVendorAssessment).mockResolvedValue(review("COMPLETED").assessment);
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    fireEvent.click(await screen.findByRole("button", { name: "Record assessment conclusion" }));
    fireEvent.change(screen.getByLabelText("Conclusion"), { target: { value: "SATISFACTORY_WITH_CONDITIONS" } });
    fireEvent.change(screen.getByLabelText("Assessment basis"), { target: { value: "Proceed after the recorded access-control action is complete." } });
    fireEvent.click(screen.getByRole("button", { name: "Record assessment conclusion" }));

    await waitFor(() => expect(completeVendorAssessment).toHaveBeenCalledWith("assessment-1", expect.objectContaining({
      expected_version: 3,
      conclusion: "SATISFACTORY_WITH_CONDITIONS",
      rationale: "Proceed after the recorded access-control action is complete.",
    })));
    expect(await screen.findByText("Assessment conclusion recorded. The vendor relationship status has not changed.")).toBeTruthy();
  });

  it("sends clarification through the secure outcome contract and clears the raw audience", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("UNDER_REVIEW") });
    vi.mocked(loadVendorAssessment).mockResolvedValue(review("UNDER_REVIEW"));
    vi.mocked(requestVendorAssessmentClarification).mockResolvedValue({
      assessment: { ...assessment("COLLECTING"), version: 4, current_request_id: "request-2" },
      state: "DELIVERED",
    });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1" onOpenRequest={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Request clarification" }));
    fireEvent.click(screen.getByLabelText("Independent security testing"));
    fireEvent.change(screen.getByLabelText("What the vendor must provide"), { target: { value: "Provide the current independent test report." } });
    fireEvent.change(screen.getByLabelText("Vendor contact email"), { target: { value: "security@vendor.example" } });
    fireEvent.change(screen.getByLabelText("Response due date"), { target: { value: "2099-09-12" } });
    fireEvent.click(screen.getByRole("button", { name: "Send clarification request" }));

    await waitFor(() => expect(requestVendorAssessmentClarification).toHaveBeenCalledWith("assessment-1", expect.objectContaining({
      expected_version: 3, request_fields: ["security-testing"], audience: "security@vendor.example", invitation_ttl_minutes: 1440,
    })));
    expect(await screen.findByText("Clarification sent. The assessment will return to review when the vendor responds.")).toBeTruthy();
    expect(screen.queryByText("security@vendor.example")).toBeNull();
  });

  it("reloads canonical findings after the deficiency command instead of adding one locally", async () => {
    const initial = review("UNDER_REVIEW");
    const refreshed = { ...initial, assessment: { ...initial.assessment, version: 4 }, matters: [...initial.matters, { matter_id: "matter-2", type: "VENDOR_DEFICIENCY", status: "OPEN", title: "Current security test required" }] };
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("UNDER_REVIEW") });
    vi.mocked(loadVendorAssessment).mockResolvedValueOnce(initial).mockResolvedValueOnce(refreshed);
    vi.mocked(createVendorAssessmentDeficiency).mockResolvedValue({
      assessment: refreshed.assessment,
      matter: { matter: { id: "matter-2", reference: "MAT-002", title: "Current security test required", status: "OPEN" } },
    });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    fireEvent.click(await screen.findByRole("button", { name: "Record finding" }));
    fireEvent.change(screen.getByLabelText("Finding reference"), { target: { value: "security-test-report" } });
    fireEvent.change(screen.getByLabelText("Finding title"), { target: { value: "Current security test required" } });
    fireEvent.change(screen.getByLabelText("Finding details"), { target: { value: "The submitted report is no longer current for this review." } });
    fireEvent.change(screen.getByLabelText("Action due date"), { target: { value: "2099-09-20" } });
    fireEvent.click(screen.getByRole("button", { name: "Record finding" }));

    await waitFor(() => expect(createVendorAssessmentDeficiency).toHaveBeenCalledWith("assessment-1", expect.objectContaining({ expected_version: 3, trigger_key: "security-test-report" })));
    expect(await screen.findByText("Current security test required")).toBeTruthy();
    expect(loadVendorAssessment).toHaveBeenCalledTimes(2);
  });

  it("renders the refreshed review returned by a document decision", async () => {
    const initial = review("UNDER_REVIEW");
    const refreshed = { ...initial, assessment: { ...initial.assessment, version: 4 }, documents: initial.documents.map((document) => ({ ...document, status: "VALIDATED", evidence_class: "BANK_VALIDATED" })) };
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("UNDER_REVIEW") });
    vi.mocked(loadVendorAssessment).mockResolvedValue(initial);
    vi.mocked(reviewVendorAssessmentDocument).mockResolvedValue(refreshed);
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    fireEvent.click(await screen.findByRole("button", { name: "Validate document" }));
    fireEvent.change(screen.getByLabelText("Evidence class"), { target: { value: "BANK_VALIDATED" } });
    fireEvent.click(screen.getByRole("button", { name: "Record validation" }));

    await waitFor(() => expect(reviewVendorAssessmentDocument).toHaveBeenCalledWith("assessment-1", "artifact-1", {
      expected_version: 3, decision: "VALIDATE", document_type: "SECURITY_TEST", evidence_class: "BANK_VALIDATED", valid_until: "2027-08-01",
    }));
    expect(await screen.findByText("Validated · Bank validated evidence")).toBeTruthy();
  });

  it("opens an authorized assessment document in a separate browser context", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("UNDER_REVIEW") });
    vi.mocked(loadVendorAssessment).mockResolvedValue(review("UNDER_REVIEW"));
    vi.mocked(vendorAssessmentDocumentURL).mockReturnValue("/api/v1/vendor-assessments/assessment-1/requests/request-1/documents/artifact-1/open");
    const openDocument = vi.spyOn(window, "open").mockImplementation(() => null);
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    fireEvent.click(await screen.findByRole("button", { name: "Open document" }));

    expect(vendorAssessmentDocumentURL).toHaveBeenCalledWith("assessment-1", "request-1", "artifact-1");
    expect(openDocument).toHaveBeenCalledWith("/api/v1/vendor-assessments/assessment-1/requests/request-1/documents/artifact-1/open", "_blank", "noopener,noreferrer");
    openDocument.mockRestore();
  });

  it("shows a completed assessment as a read-only review record", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: review("COMPLETED").assessment });
    vi.mocked(loadVendorAssessment).mockResolvedValue(review("COMPLETED"));
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    expect(await screen.findByText("Proceed after the recorded access-control action is complete.")).toBeTruthy();
    expect(screen.getByText("The next resilience exercise remains due.")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "View completed assessment" })).toBeNull();
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
