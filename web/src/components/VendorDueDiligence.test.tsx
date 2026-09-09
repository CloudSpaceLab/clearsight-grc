import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import axe from "axe-core";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import type { VendorAssessment, VendorAssessmentReviewView } from "../vendorAssessmentTypes";
import { VendorDueDiligence } from "./VendorDueDiligence";
import { loadVendorCollection } from "../vendorCollectionApi";
vi.mock("../vendorCollectionApi", () => ({ loadVendorCollection: vi.fn().mockResolvedValue({ assessment_id: "assessment-1", assessment_version: 3, prepared: false, can_reconcile: false, observed_at: "2026-09-08T10:00:00Z", fields: [], vendor_pending_count: 0, bank_pending_count: 0 }) }));

const relationship: VendorRelationshipAggregate = {
  vendor: { id: "vendor-1", tenant_id: "bank", legal_name: "Acme Processing Limited", jurisdiction: "Nigeria", status: "ACTIVE", created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z", version: 1 },
  relationship: { id: "relationship-1", tenant_id: "bank", legal_entity_id: "entity", vendor_id: "vendor-1", service_name: "Card transaction processing", business_owner_principal_id: "owner-1", criticality: "IMPORTANT", privacy_role: "PROCESSOR", status: "PROPOSED", created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z", version: 4 },
};

function assessment(status: VendorAssessment["status"]): VendorAssessment {
  return {
    id: "assessment-1", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "relationship-1",
    review_kind: "ONBOARDING", source_trigger: "INITIAL", stable_episode_key: "episode-1", status, form_template_id: "form-1", form_template_version: 3,
    review_due_at: "2026-09-30T17:00:00Z", started_by_principal_id: "owner-1", started_at: "2026-08-26T10:00:00Z",
    review_matter_id: status === "SETUP_PENDING" ? undefined : "matter-1", current_request_id: status === "COLLECTING" ? "request-1" : undefined,
    submitted_at: status === "SUBMITTED" || status === "UNDER_REVIEW" || status === "COMPLETED" ? "2026-08-28T11:00:00Z" : undefined,
    review_started_at: status === "UNDER_REVIEW" || status === "COMPLETED" ? "2026-08-28T12:00:00Z" : undefined,
    completed_at: status === "COMPLETED" ? "2026-08-29T12:00:00Z" : undefined,
    conclusion: status === "COMPLETED" ? "SATISFACTORY_WITH_CONDITIONS" : undefined,
    conclusion_rationale: status === "COMPLETED" ? "Required evidence was reviewed and one finding remains subject to an agreed action." : undefined,
    version: 3, created_at: "2026-08-26T10:00:00Z", updated_at: "2026-08-28T12:00:00Z",
  };
}

const form = { id: "form-1", version: 3, name: "Vendor security and privacy review", presentation: "WIZARD" as const };

async function chooseOption(label: string, option: string) {
  fireEvent.click(screen.getByRole("button", { name: new RegExp(label) }));
  fireEvent.click(await screen.findByRole("option", { name: option }));
}

function primaryActions() {
  return screen.queryAllByRole("button").filter((button) => button.classList.contains("cs-button--primary") && !button.hasAttribute("disabled"));
}

describe("VendorDueDiligence", () => {
  it("shows only uncovered document requirements below a mixed checklist", async () => {
    const current = { ...assessment("UNDER_REVIEW"), current_request_id: "request-1" };
    const document = { field_id: "covered", artifact_id: "shared-file", file_name: "Covered.pdf", media_type: "application/pdf", size_bytes: 100, artifact_status: "AVAILABLE", status: "SUBMITTED", evidence_class: "VENDOR_SUPPLIED", document_type: "SECURITY_TEST" };
    const review: VendorAssessmentReviewView = { assessment: current, requests: [], answers: [], coverage: { visible_fields: 0, answered_fields: 0, required_fields: 0, answered_required: 0, ratio: 1 }, documents: [document, { ...document, field_id: "uncovered", file_name: "Supporting.pdf" }], matters: [] };
    vi.mocked(loadVendorCollection).mockResolvedValueOnce({ assessment_id: current.id, assessment_version: current.version, request_id: "request-1", request_version: 5, prepared: true, can_reconcile: false, observed_at: "2026-09-09T10:00:00Z", fields: [{ field_id: "covered", label: "Covered requirement", type: "vendor_document", required: true, collection_state: "RECEIVED", vendor_action_required: false, bank_review_state: "PENDING" }], vendor_pending_count: 0, bank_pending_count: 1 });
    render(<VendorDueDiligence relationship={relationship} assessment={current} review={review} form={form} onPrepare={vi.fn()}/>);
    await screen.findByRole("article", { name: "Covered requirement" });
    const supporting = screen.getByRole("heading", { name: "Supporting documents" }).parentElement!;
    await waitFor(() => expect(within(supporting).queryByText("Covered.pdf")).toBeNull());
    expect(within(supporting).getByText("Supporting.pdf")).toBeTruthy();
  });
  it("sends a prepared request using the latest matching collection version", async () => {
    vi.mocked(loadVendorCollection).mockResolvedValueOnce({ assessment_id: "assessment-1", assessment_version: 7, request_id: "request-1", request_version: 5, prepared: true, can_reconcile: false, deadline: "2099-09-30T16:00:00Z", observed_at: "2026-09-08T10:00:00Z", fields: [], vendor_pending_count: 1, bank_pending_count: 0 });
    const onSend = vi.fn().mockResolvedValue({ assessment: { ...assessment("COLLECTING"), version: 8 }, request: { id: "request-1", deadline: "2099-09-30T16:00:00Z" }, state: "DELIVERED" });
    render(<VendorDueDiligence relationship={relationship} assessment={{ ...assessment("READY_TO_SEND"), current_request_id: "request-1", review_due_at: "2099-09-30T23:59:59.000Z" }} form={form} onPrepare={vi.fn()} onSend={onSend}/>);
    await screen.findByRole("button", { name: "Refresh checklist" });
    fireEvent.click(screen.getByRole("button", { name: "Send due diligence request" }));
    fireEvent.click(screen.getByRole("button", { name: "Send due diligence request" }));
    await waitFor(() => expect(onSend).toHaveBeenCalledWith(expect.objectContaining({ expected_version: 7, audience: "", deadline: "2099-09-30T16:00:00Z" })));
  });

  it("shows an existing document receipt without labelling it a missing respondent answer", async () => {
    const review = { assessment: assessment("UNDER_REVIEW"), requests: [], answers: [{ field_id: "security-report", label: "Security report", type: "vendor_document", required: true, visibility: "VISIBLE", collection_resolution: { id: "receipt", version: 1, source: { file_name: "Existing security report.pdf" }, reconciled_by: "reviewer", reconciled_at: "2026-09-08T10:00:00Z", rationale: "Service covered." } }], coverage: { visible_fields: 1, answered_fields: 0, required_fields: 1, answered_required: 0, ratio: 0 }, documents: [], matters: [] } as VendorAssessmentReviewView;
    render(<VendorDueDiligence relationship={relationship} assessment={review.assessment} review={review} form={form}/>);
    const answer = screen.getByText(/^Security report/, { selector: "dt" }).parentElement!;
    expect(within(answer).getByText("Document linked: Existing security report.pdf")).toBeTruthy();
    expect(within(answer).queryByText("Required response missing")).toBeNull();
    expect(within(answer).queryByText(/Vendor response:/)).toBeNull();
    expect((await axe.run(answer.closest("dl")!, { runOnly: { type: "rule", values: ["definition-list", "dlitem"] } })).violations).toEqual([]);
  });

  it("offers a rejection decision through the checklist review action and submits the exact field", async () => {
    const document = { field_id: "second-report", artifact_id: "shared-artifact", file_name: "Service assurance.pdf", media_type: "application/pdf", size_bytes: 64000, artifact_status: "AVAILABLE", status: "SUBMITTED", evidence_class: "VENDOR_SUPPLIED", document_type: "SECURITY_TEST" };
    const review: VendorAssessmentReviewView = { assessment: assessment("UNDER_REVIEW"), requests: [], answers: [], coverage: { visible_fields: 0, answered_fields: 0, required_fields: 0, answered_required: 0, ratio: 1 }, documents: [document], matters: [] };
    vi.mocked(loadVendorCollection).mockResolvedValueOnce({ assessment_id: "assessment-1", assessment_version: 3, request_id: "request-1", request_version: 5, prepared: true, can_reconcile: true, observed_at: "2026-09-08T10:00:00Z", fields: [{ field_id: "second-report", label: "Service assurance", type: "vendor_document", required: true, collection_state: "REUSED", vendor_action_required: false, bank_review_state: "PENDING" }], vendor_pending_count: 0, bank_pending_count: 1 });
    const onReviewDocument = vi.fn().mockResolvedValue({ ...review, assessment: { ...review.assessment, version: 4 } });
    render(<VendorDueDiligence relationship={relationship} assessment={review.assessment} review={review} form={form} onPrepare={vi.fn()} onReviewDocument={onReviewDocument}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Review document" }));
    expect(screen.getByRole("dialog", { name: "Review document" })).toBeTruthy();
    await chooseOption("Decision", "Reject");
    fireEvent.click(screen.getByRole("button", { name: "Record rejection" }));
    await waitFor(() => expect(onReviewDocument).toHaveBeenCalledWith("assessment-1", "shared-artifact", expect.objectContaining({ field_id: "second-report", decision: "REJECT", evidence_class: "VENDOR_SUPPLIED" })));
  });

  it("prepares the request without sending and preserves the contact for the send step", async () => {
    const prepared = { ...assessment("READY_TO_SEND"), current_request_id: "request-1", version: 4 };
    const onPrepare = vi.fn().mockResolvedValue({ assessment: prepared, request: { id: "request-1", status: "PREPARED", deadline: "2099-09-30T23:59:59.000Z" }, state: "PREPARED" });
    const onSend = vi.fn().mockResolvedValue({ assessment: { ...prepared, status: "COLLECTING", version: 5 }, request: { id: "request-1", status: "READY" }, state: "DELIVERED" });
    render(<VendorDueDiligence relationship={relationship} assessment={{ ...assessment("READY_TO_SEND"), review_due_at: "2099-09-30T23:59:59.000Z" }} form={form} onPrepare={onPrepare} onSend={onSend}/>);
    fireEvent.click(screen.getByRole("button", { name: "Prepare request" }));
    fireEvent.change(screen.getByLabelText("Vendor contact email", { exact: false }), { target: { value: "security@vendor.example" } });
    fireEvent.change(screen.getByLabelText("Response due date", { exact: false }), { target: { value: "2099-09-30" } });
    fireEvent.click(screen.getByRole("button", { name: "Prepare request" }));
    await waitFor(() => expect(onPrepare).toHaveBeenCalledWith({ expected_version: 3, audience: "security@vendor.example", deadline: "2099-09-30T23:59:59.000Z" }));
    expect(onSend).not.toHaveBeenCalled();
    expect(await screen.findByText(/Request prepared/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Send due diligence request" }));
    expect(screen.getByText(/prepared contact/)).toBeTruthy();
    expect(screen.queryByRole("textbox", { name: "Vendor contact email" })).toBeNull();
    expect(loadVendorCollection).toHaveBeenCalledWith("assessment-1", expect.any(AbortSignal));
  });
  it("offers governed form setup when no active form is available", () => {
    const onSetUpForm = vi.fn();
    const onOpenForms = vi.fn();
    render(<VendorDueDiligence relationship={relationship} assessment={null} onSetUpForm={onSetUpForm} onOpenForms={onOpenForms}/>);
    expect(screen.getByText(/No active due-diligence form was found in this legal entity/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Use a starter template" }));
    expect(onSetUpForm).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole("button", { name: "Open Forms" }));
    expect(onOpenForms).toHaveBeenCalledOnce();
    expect(primaryActions()).toHaveLength(1);
  });

  it("starts from a relationship-scoped preview with one primary action", async () => {
    const onStart = vi.fn().mockResolvedValue(assessment("SETUP_PENDING"));
    render(<VendorDueDiligence relationship={relationship} assessment={null} form={form} defaultReviewDueDate="2026-09-30" onStart={onStart}/>);

    expect(screen.getByRole("heading", { name: "Due diligence" })).toBeTruthy();
    expect(screen.getByText("Acme Processing Limited")).toBeTruthy();
    expect(screen.getAllByText("Card transaction processing")).toHaveLength(2);
    expect(primaryActions()).toHaveLength(1);
    fireEvent.click(screen.getByRole("button", { name: "Start due diligence" }));

    expect((screen.getByLabelText("Review due date", { exact: false }) as HTMLInputElement).value).toBe("2026-09-30");
    expect(screen.getByText("Vendor security and privacy review")).toBeTruthy();
    expect(primaryActions()).toHaveLength(1);
    fireEvent.click(screen.getByRole("button", { name: "Start due diligence" }));

    await waitFor(() => expect(onStart).toHaveBeenCalledWith({
      relationship_version: 4,
      form_template_id: "form-1",
      form_template_version: 3,
      review_due_at: "2026-09-30T23:59:59.000Z",
    }));
  });

  it("starts an event-driven reassessment with the review reference", async () => {
    const managedRelationship = { ...relationship, relationship: { ...relationship.relationship, status: "RESTRICTED" as const } };
    const onStart = vi.fn().mockResolvedValue({ ...assessment("SETUP_PENDING"), review_kind: "TRIGGERED", source_trigger: "change-2099-0042" });
    render(<VendorDueDiligence relationship={managedRelationship} assessment={assessment("COMPLETED")} form={form} defaultReviewDueDate="2099-09-30" onStart={onStart}/>);

    fireEvent.click(screen.getByRole("button", { name: "Start reassessment" }));
    await chooseOption("Review type", "Event or change");
    fireEvent.change(screen.getByLabelText("Review reference", { exact: false }), { target: { value: "change-2099-0042" } });
    fireEvent.click(screen.getByRole("button", { name: "Start reassessment" }));

    await waitFor(() => expect(onStart).toHaveBeenCalledWith({
      relationship_version: 4,
      review_kind: "TRIGGERED",
      source_trigger: "change-2099-0042",
      form_template_id: "form-1",
      form_template_version: 3,
      review_due_at: "2099-09-30T23:59:59.000Z",
      scope_kind: "FULL",
      selected_field_ids: [],
    }));
  });

  it("starts a focused reassessment for selected held records", async () => {
    const managedRelationship = { ...relationship, relationship: { ...relationship.relationship, status: "ACTIVE" as const } };
    const focusedForm = { ...form, fields: [{ id: "registered-address", label: "Registered address", type: "LONG_TEXT", collection_intent: "CONFIRM_OR_CORRECT" as const, target_key: "VENDOR.IDENTITY.REGISTERED_ADDRESS" }] };
    const onStart = vi.fn().mockResolvedValue(assessment("SETUP_PENDING"));
    render(<VendorDueDiligence relationship={managedRelationship} assessment={assessment("COMPLETED")} form={focusedForm} defaultReviewDueDate="2099-09-30" onStart={onStart}/>);

    fireEvent.click(screen.getByRole("button", { name: "Start reassessment" }));
    fireEvent.change(screen.getByLabelText("Review reference", { exact: false }), { target: { value: "address-refresh-2099" } });
    fireEvent.click(screen.getByLabelText("Selected held records only", { exact: false }));
    expect((screen.getByLabelText(/Registered address/) as HTMLInputElement).checked).toBe(true);
    fireEvent.click(screen.getByRole("button", { name: "Start reassessment" }));

    await waitFor(() => expect(onStart).toHaveBeenCalledWith(expect.objectContaining({
      scope_kind: "FOCUSED",
      selected_field_ids: ["registered-address"],
      source_trigger: "address-refresh-2099",
    })));
  });

  it.each([
    ["SETUP_PENDING", "View setup status"],
    ["READY_TO_SEND", "Send due diligence request"],
    ["COLLECTING", "Review request status"],
    ["SUBMITTED", "Review vendor response"],
    ["UNDER_REVIEW", "Record assessment conclusion"],
  ] as const)("offers one dominant action for %s", (status, label) => {
    render(<VendorDueDiligence
      relationship={relationship}
      assessment={assessment(status)}
      form={form}
      onRefresh={vi.fn()}
      onSend={vi.fn()}
      onOpenRequest={vi.fn()}
      onStartReview={vi.fn()}
      onComplete={vi.fn()}
    />);

    expect((screen.getByRole("button", { name: label }) as HTMLButtonElement).disabled).toBe(false);
    expect(primaryActions()).toHaveLength(1);
  });

  it("keeps a completed assessment read-only", () => {
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("COMPLETED")} form={form}/>);
    expect(screen.getByText("The assessment conclusion is recorded. The vendor relationship status remains separate.")).toBeTruthy();
    expect(primaryActions()).toHaveLength(0);
  });

  it("retries failed setup without presenting the assessment as ready", async () => {
    const onRetrySetup = vi.fn().mockResolvedValue({ assessment: { ...assessment("SETUP_PENDING"), version: 4 }, setup: { assessment_id: "assessment-1", state: "READY", attempts: 0 } });
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("SETUP_PENDING")} form={form} setupFailure="The review record could not be prepared after three attempts." onRetrySetup={onRetrySetup}/>);

    expect(screen.getByRole("alert").textContent).toContain("The review record could not be prepared after three attempts.");
    fireEvent.click(screen.getByRole("button", { name: "Retry due diligence setup" }));
    expect(onRetrySetup).toHaveBeenCalledWith("assessment-1", 3);
    expect(await screen.findByText("Assessment setup queued. Review setup will continue in the background.")).toBeTruthy();
    expect(primaryActions()).toHaveLength(1);
  });

  it("clears and stops displaying the recipient after the send attempt", async () => {
    const sent = { assessment: { ...assessment("COLLECTING"), current_request_id: "request-1" }, request: { id: "request-1", status: "READY", deadline: "2026-09-20T17:00:00Z" }, state: "DELIVERED" as const };
    const onSend = vi.fn().mockResolvedValue(sent);
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("READY_TO_SEND")} form={form} onSend={onSend} onOpenRequest={vi.fn()}/>);

    fireEvent.click(screen.getByRole("button", { name: "Send due diligence request" }));
    fireEvent.change(screen.getByLabelText("Vendor contact email", { exact: false }), { target: { value: "security@vendor.example" } });
    fireEvent.change(screen.getByLabelText("Response due date", { exact: false }), { target: { value: "2026-09-20" } });
    fireEvent.click(screen.getByRole("button", { name: "Send due diligence request" }));

    await screen.findByText(/The request was sent\. The response is due 20 Sept? 2026\./);
    expect(screen.queryByText("security@vendor.example")).toBeNull();
    expect(screen.queryByDisplayValue("security@vendor.example")).toBeNull();
    expect(onSend).toHaveBeenCalledWith({ expected_version: 3, audience: "security@vendor.example", deadline: "2026-09-20T23:59:59.000Z", invitation_ttl_minutes: 1440 });
  });

  it("offers safe recovery when email delivery did not complete", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    render(<VendorDueDiligence
      relationship={relationship}
      assessment={assessment("COLLECTING")}
      form={form}
      requestOutcome={{ assessment: assessment("COLLECTING"), request: { id: "request-1", status: "READY" }, state: "LINK_CREATED_EMAIL_NOT_SENT", recovery: "Copy the secure link or retry email delivery.", capture_url: "https://capture.example.test/?capture_invite=secret" }}
      onOpenRequest={vi.fn()}
    />);

    expect(screen.getByRole("alert").textContent).toContain("Email delivery did not complete");
    expect(screen.queryByText(/capture_invite/)).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Copy secure link" }));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("https://capture.example.test/?capture_invite=secret"));
    expect(await screen.findByText("Secure link copied.")).toBeTruthy();
    expect(primaryActions()).toHaveLength(1);
  });

  it("keeps request status primary and sends another independently expiring link as a focused secondary action", async () => {
    const delivered = { assessment: { ...assessment("COLLECTING"), version: 4 }, request: { id: "request-1", status: "READY" }, state: "DELIVERED" as const };
    const onReissue = vi.fn().mockResolvedValue(delivered);
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("COLLECTING")} form={form} onOpenRequest={vi.fn()} onReissue={onReissue}/>);

    expect(screen.getByRole("button", { name: "Review request status" }).classList.contains("cs-button--primary")).toBe(true);
    expect(screen.getByRole("button", { name: "Send another link" }).classList.contains("cs-button--secondary")).toBe(true);
    expect(primaryActions()).toHaveLength(1);
    fireEvent.click(screen.getByRole("button", { name: "Send another link" }));
    fireEvent.change(screen.getByLabelText("Vendor contact email", { exact: false }), { target: { value: "security@vendor.example" } });
    await chooseOption("New link valid for", "7 days");
    fireEvent.click(screen.getByRole("button", { name: "Send another link" }));

    await waitFor(() => expect(onReissue).toHaveBeenCalledWith({ expected_version: 3, audience: "security@vendor.example", invitation_ttl_minutes: 10080 }));
    expect(await screen.findByText("Another link was sent. Earlier links remain available until their printed expiry unless you cancel the request.")).toBeTruthy();
    expect(screen.queryByText("security@vendor.example")).toBeNull();
    expect(screen.queryByDisplayValue("security@vendor.example")).toBeNull();
    expect(primaryActions()).toHaveLength(1);
  });

  it("clears the replacement recipient after a failed attempt", async () => {
    const onReissue = vi.fn().mockRejectedValue(new Error("unavailable"));
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("COLLECTING")} form={form} onOpenRequest={vi.fn()} onReissue={onReissue}/>);

    fireEvent.click(screen.getByRole("button", { name: "Send another link" }));
    fireEvent.change(screen.getByLabelText("Vendor contact email", { exact: false }), { target: { value: "security@vendor.example" } });
    fireEvent.click(screen.getByRole("button", { name: "Send another link" }));

    expect(await screen.findByText("The new link was not sent. Re-enter the vendor contact email before trying again.")).toBeTruthy();
    expect(screen.queryByText("security@vendor.example")).toBeNull();
    expect(screen.queryByDisplayValue("security@vendor.example")).toBeNull();
  });

  it("clears an invalid replacement recipient after validation", async () => {
    const onReissue = vi.fn();
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("COLLECTING")} form={form} onOpenRequest={vi.fn()} onReissue={onReissue}/>);

    fireEvent.click(screen.getByRole("button", { name: "Send another link" }));
    fireEvent.change(screen.getByLabelText("Vendor contact email", { exact: false }), { target: { value: "not-an-email" } });
    fireEvent.click(screen.getByRole("button", { name: "Send another link" }));

    expect(await screen.findByText("Enter a valid vendor contact email before sending another link.")).toBeTruthy();
    expect((screen.getByLabelText("Vendor contact email", { exact: false }) as HTMLInputElement).value).toBe("");
    expect(onReissue).not.toHaveBeenCalled();
  });

  it("offers controlled copy recovery when replacement email delivery fails", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    const failed = {
      assessment: { ...assessment("COLLECTING"), version: 4 }, request: { id: "request-1", status: "READY" },
      state: "LINK_CREATED_EMAIL_NOT_SENT" as const, recovery: "Copy the replacement secure link or retry email delivery.",
      capture_url: "https://capture.example.test/?capture_invite=replacement-secret",
    };
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("COLLECTING")} form={form} onOpenRequest={vi.fn()} onReissue={vi.fn().mockResolvedValue(failed)}/>);

    fireEvent.click(screen.getByRole("button", { name: "Send another link" }));
    fireEvent.change(screen.getByLabelText("Vendor contact email", { exact: false }), { target: { value: "security@vendor.example" } });
    fireEvent.click(screen.getByRole("button", { name: "Send another link" }));

    fireEvent.click(await screen.findByRole("button", { name: "Copy new link" }));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("https://capture.example.test/?capture_invite=replacement-secret"));
    expect(screen.queryByText(/replacement-secret/)).toBeNull();
    expect(primaryActions()).toHaveLength(1);
  });

  it("keeps clarification and findings secondary to the review conclusion", () => {
	const openMatter = vi.fn();
    const review: VendorAssessmentReviewView = {
      assessment: assessment("UNDER_REVIEW"),
      requests: [],
      response: { submission_id: "submission-1", request_id: "request-1", submitted_at: "2026-08-28T11:00:00Z", answer_count: 14, artifact_count: 2 },
      answers: [],
      coverage: { visible_fields: 14, answered_fields: 14, required_fields: 10, answered_required: 10, ratio: 1 },
      documents: [{ field_id: "security-report", artifact_id: "artifact-1", file_name: "independent-security-test.pdf", media_type: "application/pdf", size_bytes: 64000, artifact_status: "AVAILABLE", evidence_class: "VENDOR_SUPPLIED", document_type: "SECURITY_TEST" }],
      matters: [{ matter_id: "finding-1", type: "VENDOR_DEFICIENCY", title: "Independent test renewal is due", status: "ACTION_AGREED" }],
    };
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("UNDER_REVIEW")} review={review} form={form} onRequestClarification={vi.fn()} onCreateDeficiency={vi.fn()} onReviewDocument={vi.fn()} onComplete={vi.fn()} onOpenMatter={openMatter}/>);

    const reviewRegion = screen.getByRole("region", { name: "Vendor response review" });
    expect(within(reviewRegion).getByText(/14 answers · 2 documents/)).toBeTruthy();
    expect(within(reviewRegion).getByText("independent-security-test.pdf")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Request clarification" }).classList.contains("cs-button--secondary")).toBe(true);
    expect(screen.getByRole("button", { name: "Record assessment conclusion" }).classList.contains("cs-button--primary")).toBe(true);
	fireEvent.click(screen.getByRole("button", { name: "Open finding" }));
	expect(openMatter).toHaveBeenCalledWith("finding-1");
    expect(primaryActions()).toHaveLength(1);
  });

  it("requires an explicit conclusion and basis without selecting from the provisional score", async () => {
    const review: VendorAssessmentReviewView = {
      assessment: assessment("UNDER_REVIEW"), requests: [], answers: [],
      coverage: { visible_fields: 0, answered_fields: 0, required_fields: 0, answered_required: 0, ratio: 1 },
      documents: [], provisional_score: { score: 100, coverage: 1, rule_results: [] }, matters: [],
    };
    const onComplete = vi.fn();
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("UNDER_REVIEW")} review={review} form={form} onComplete={onComplete}/>);

    fireEvent.click(screen.getByRole("button", { name: "Record assessment conclusion" }));

    const conclusion = screen.getByRole("button", { name: /Conclusion/ });
    const submit = screen.getByRole("button", { name: "Record assessment conclusion" }) as HTMLButtonElement;
    expect(conclusion.textContent).toContain("Select a conclusion");
    expect(submit.disabled).toBe(true);

    await chooseOption("Conclusion", "Satisfactory");
    expect(submit.disabled).toBe(true);
    fireEvent.change(screen.getByLabelText("Assessment basis", { exact: false }), { target: { value: "The submitted evidence supports the stated controls." } });
    expect(submit.disabled).toBe(false);
  });

  it("shows corrections, omissions, critical responses, validation limits and freshness", () => {
    const review: VendorAssessmentReviewView = {
      assessment: assessment("UNDER_REVIEW"), requests: [],
      response: { submission_id: "submission-1", request_id: "request-1", submitted_at: "2026-08-28T11:00:00Z", answer_count: 2, artifact_count: 0 },
      answers: [
        {
          field_id: "service-criticality", label: "Service criticality", type: "SINGLE_SELECT", required: true, visibility: "VISIBLE", value: { text: "Critical" },
          provenance: {
            origin: "RESPONDENT_CORRECTED", source_value: { kind: "STRING", text: "Important" },
            source_receipt: { source_id: "Procurement register", observed_at: "2026-08-27T09:30:00Z" },
            validations: [{ state: "STALE", binding_name: "Approved vendor register", source_id: "Procurement register", receipt: { observed_at: "2026-08-20T09:30:00Z" } }],
          },
        },
        { field_id: "insurance", label: "Cyber insurance", type: "YES_NO", required: true, visibility: "VISIBLE" },
        { field_id: "insurance-limit", label: "Insurance limit", type: "NUMBER", required: true, visibility: "CONDITIONALLY_OMITTED" },
      ],
      coverage: { visible_fields: 2, answered_fields: 1, required_fields: 2, answered_required: 1, ratio: 0.5 },
      documents: [],
      provisional_score: {
        score: 35, coverage: 0.5,
        critical_failures: [{ field_id: "service-criticality", outcome: "Critical service", points: 0, critical: true }],
        rule_results: [],
      },
      matters: [],
    };
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("UNDER_REVIEW")} review={review} form={form} onComplete={vi.fn()}/>);

    const criticality = screen.getByText(/^Service criticality/, { selector: "dt" }).parentElement!;
    expect(within(criticality).getByText("Vendor response: Critical")).toBeTruthy();
    expect(within(criticality).getByText("Source value: Important")).toBeTruthy();
    expect(within(criticality).getByText("Critical response: Critical service")).toBeTruthy();
    expect(within(criticality).getByText("Validation out of date · Approved vendor register · Checked 20 Aug 2026")).toBeTruthy();
    expect(screen.getByText(/^Cyber insurance/, { selector: "dt" }).parentElement!.textContent).toContain("Required response missing");
    expect(screen.getByText(/^Insurance limit/, { selector: "dt" }).parentElement!.textContent).toContain("Not requested because its condition was not met");
  });

  it("opens only an available assessment document before its review actions", () => {
    const available = { field_id: "security-report", artifact_id: "artifact-1", file_name: "security-report.pdf", media_type: "application/pdf", size_bytes: 64000, artifact_status: "AVAILABLE", status: "SUBMITTED", evidence_class: "VENDOR_SUPPLIED" as const, document_type: "SECURITY_TEST" };
    const quarantined = { ...available, artifact_id: "artifact-2", file_name: "quarantined-report.pdf", artifact_status: "QUARANTINED" };
    const review: VendorAssessmentReviewView = {
      assessment: assessment("UNDER_REVIEW"), requests: [],
      response: { submission_id: "submission-1", request_id: "request-7", submitted_at: "2026-08-28T11:00:00Z", answer_count: 0, artifact_count: 2 },
      answers: [], coverage: { visible_fields: 0, answered_fields: 0, required_fields: 0, answered_required: 0, ratio: 1 },
      documents: [available, quarantined], matters: [],
    };
    const onOpenDocument = vi.fn();
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("UNDER_REVIEW")} review={review} form={form} onOpenDocument={onOpenDocument} onReviewDocument={vi.fn()} onComplete={vi.fn()}/>);

    const availableDocument = screen.getByRole("article", { name: "security-report.pdf" });
    const availableActions = within(availableDocument).getAllByRole("button");
    expect(availableActions.map((button) => button.textContent)).toEqual(["Open document", "Validate document", "Reject document"]);
    fireEvent.click(availableActions[0]!);
    expect(onOpenDocument).toHaveBeenCalledWith("assessment-1", "request-7", "artifact-1");

    const quarantinedDocument = screen.getByRole("article", { name: "quarantined-report.pdf" });
    expect((within(quarantinedDocument).getByRole("button", { name: "Open document" }) as HTMLButtonElement).disabled).toBe(true);
    expect(within(quarantinedDocument).getByText("This document is quarantined. Wait for a clean replacement before reviewing it.")).toBeTruthy();
    expect((within(quarantinedDocument).getByRole("button", { name: "Reject document" }) as HTMLButtonElement).disabled).toBe(true);
  });

  it("requests targeted clarification, clears the audience and exposes only a returned fallback link", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    const review: VendorAssessmentReviewView = {
      assessment: assessment("UNDER_REVIEW"), requests: [],
      response: { submission_id: "submission-1", request_id: "request-1", submitted_at: "2026-08-28T11:00:00Z", answer_count: 1, artifact_count: 0 },
      answers: [{ field_id: "security-testing", label: "Independent security testing", type: "YES_NO", required: true, visibility: "VISIBLE", value: { text: "Yes" } }],
      coverage: { visible_fields: 1, answered_fields: 1, required_fields: 1, answered_required: 1, ratio: 1 }, documents: [], matters: [],
    };
    const outcome = {
      assessment: { ...assessment("COLLECTING"), version: 4, current_request_id: "request-2" },
      state: "LINK_CREATED_EMAIL_NOT_SENT" as const,
      recovery: "Copy the secure link or retry email delivery.",
      capture_url: "https://capture.example.test/?capture_invite=clarification-secret",
    };
    const onRequestClarification = vi.fn().mockResolvedValue(outcome);
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("UNDER_REVIEW")} review={review} form={form} onRequestClarification={onRequestClarification} onOpenRequest={vi.fn()} onComplete={vi.fn()}/>);

    fireEvent.click(screen.getByRole("button", { name: "Request clarification" }));
    fireEvent.click(screen.getByRole("checkbox", { name: "Independent security testing" }));
    fireEvent.change(screen.getByLabelText("What the vendor must provide", { exact: false }), { target: { value: "Provide the current independent security test report." } });
    fireEvent.change(screen.getByLabelText("Vendor contact email", { exact: false }), { target: { value: "security@vendor.example" } });
    fireEvent.change(screen.getByLabelText("Response due date", { exact: false }), { target: { value: "2026-09-12" } });
    await chooseOption("Secure link valid for", "1 hour");
    fireEvent.click(screen.getByRole("button", { name: "Send clarification request" }));

    await waitFor(() => expect(onRequestClarification).toHaveBeenCalledWith("assessment-1", {
      expected_version: 3,
      request_fields: ["security-testing"],
      message: "Provide the current independent security test report.",
      audience: "security@vendor.example",
      deadline: "2026-09-12T23:59:59.000Z",
      invitation_ttl_minutes: 60,
    }));
    expect(screen.queryByText("security@vendor.example")).toBeNull();
    expect(screen.queryByDisplayValue("security@vendor.example")).toBeNull();
    expect(screen.queryByText(/clarification-secret/)).toBeNull();
    fireEvent.click(await screen.findByRole("button", { name: "Copy clarification link" }));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith(outcome.capture_url));
    expect(primaryActions()).toHaveLength(1);
  });

  it("clears the clarification audience after an unsuccessful attempt", async () => {
    const review: VendorAssessmentReviewView = {
      assessment: assessment("UNDER_REVIEW"), requests: [],
      answers: [{ field_id: "security-testing", label: "Independent security testing", type: "YES_NO", required: true, visibility: "VISIBLE", value: { text: "Yes" } }],
      coverage: { visible_fields: 1, answered_fields: 1, required_fields: 1, answered_required: 1, ratio: 1 }, documents: [], matters: [],
    };
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("UNDER_REVIEW")} review={review} form={form} onRequestClarification={vi.fn().mockRejectedValue(new Error("unavailable"))} onComplete={vi.fn()}/>);

    fireEvent.click(screen.getByRole("button", { name: "Request clarification" }));
    fireEvent.click(screen.getByRole("checkbox", { name: "Independent security testing" }));
    fireEvent.change(screen.getByLabelText("What the vendor must provide", { exact: false }), { target: { value: "Provide the current report." } });
    fireEvent.change(screen.getByLabelText("Vendor contact email", { exact: false }), { target: { value: "security@vendor.example" } });
    fireEvent.change(screen.getByLabelText("Response due date", { exact: false }), { target: { value: "2026-09-12" } });
    fireEvent.click(screen.getByRole("button", { name: "Send clarification request" }));

    expect(await screen.findByText("The clarification request was not sent. Re-enter the vendor contact email before trying again.")).toBeTruthy();
    expect((screen.getByLabelText("Vendor contact email", { exact: false }) as HTMLInputElement).value).toBe("");
  });

  it("records a bounded canonical finding without adding a local finding", async () => {
    const review: VendorAssessmentReviewView = {
      assessment: assessment("UNDER_REVIEW"), requests: [], answers: [],
      coverage: { visible_fields: 0, answered_fields: 0, required_fields: 0, answered_required: 0, ratio: 1 }, documents: [], matters: [],
    };
    const onCreateDeficiency = vi.fn().mockResolvedValue({
      assessment: { ...assessment("UNDER_REVIEW"), version: 4 },
      matter: { matter: { id: "finding-1", reference: "MAT-002", title: "Current security test required", status: "OPEN" } },
    });
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("UNDER_REVIEW")} review={review} form={form} onCreateDeficiency={onCreateDeficiency} onComplete={vi.fn()}/>);

    fireEvent.click(screen.getByRole("button", { name: "Record finding" }));
    const reference = screen.getByLabelText("Finding reference", { exact: false }) as HTMLInputElement;
    expect(reference.maxLength).toBe(80);
    fireEvent.change(reference, { target: { value: "security-test-report" } });
    fireEvent.change(screen.getByLabelText("Finding title", { exact: false }), { target: { value: "Current security test required" } });
    fireEvent.change(screen.getByLabelText("Finding details", { exact: false }), { target: { value: "The submitted report is no longer current for this review." } });
    fireEvent.change(screen.getByLabelText("Action due date", { exact: false }), { target: { value: "2026-09-20" } });
    fireEvent.click(screen.getByRole("button", { name: "Record finding" }));

    await waitFor(() => expect(onCreateDeficiency).toHaveBeenCalledWith("assessment-1", {
      expected_version: 3, trigger_key: "security-test-report", title: "Current security test required", summary: "The submitted report is no longer current for this review.", due_at: "2026-09-20T23:59:59.000Z",
    }));
    expect(await screen.findByText("Finding MAT-002 is open and linked to this assessment.")).toBeTruthy();
    expect(screen.getByText("No findings are linked to this assessment.")).toBeTruthy();
    expect(primaryActions()).toHaveLength(1);
  });

  it("records the exact document decision metadata through a focused panel", async () => {
    const document = { field_id: "security-report", artifact_id: "artifact-1", file_name: "independent-security-test.pdf", media_type: "application/pdf", size_bytes: 64000, artifact_status: "AVAILABLE", status: "SUBMITTED", evidence_class: "VENDOR_SUPPLIED" as const, document_type: "SOC_2_TYPE_II", expires_on: "2027-05-31" };
    const review: VendorAssessmentReviewView = {
      assessment: assessment("UNDER_REVIEW"), requests: [], answers: [],
      coverage: { visible_fields: 0, answered_fields: 0, required_fields: 0, answered_required: 0, ratio: 1 }, documents: [document], matters: [],
    };
    const onReviewDocument = vi.fn().mockResolvedValue({ ...review, assessment: { ...assessment("UNDER_REVIEW"), version: 4 }, documents: [{ ...document, status: "VALIDATED", evidence_class: "BANK_VALIDATED" }] });
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("UNDER_REVIEW")} review={review} form={form} onReviewDocument={onReviewDocument} onComplete={vi.fn()}/>);

    fireEvent.click(screen.getByRole("button", { name: "Validate document" }));
    expect((screen.getByLabelText("Document type", { exact: false }) as HTMLInputElement).maxLength).toBe(128);
    await chooseOption("Evidence class", "Validated by reviewer");
    fireEvent.change(screen.getByLabelText("Valid until", { exact: false }), { target: { value: "2027-05-31" } });
    fireEvent.click(screen.getByRole("button", { name: "Record validation" }));

    await waitFor(() => expect(onReviewDocument).toHaveBeenCalledWith("assessment-1", "artifact-1", {
      expected_version: 3, field_id: "security-report", decision: "VALIDATE", document_type: "SOC_2_TYPE_II", evidence_class: "BANK_VALIDATED", valid_until: "2027-05-31",
    }));
    expect(await screen.findByText("Document accepted.")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Reject document" }));
    fireEvent.click(screen.getByRole("button", { name: "Record rejection" }));
    await waitFor(() => expect(onReviewDocument).toHaveBeenLastCalledWith("assessment-1", "artifact-1", {
      expected_version: 4, field_id: "security-report", decision: "REJECT", document_type: "SOC_2_TYPE_II", evidence_class: "VENDOR_SUPPLIED", valid_until: "2027-05-31",
    }));
    expect(await screen.findByText("Document rejected.")).toBeTruthy();
  });

  it("shows answer provenance and document security status without exposing internal identifiers", () => {
    const review: VendorAssessmentReviewView = {
      assessment: assessment("UNDER_REVIEW"),
      requests: [],
      response: { submission_id: "submission-1", request_id: "request-1", submitted_at: "2026-08-28T11:00:00Z", answer_count: 1, artifact_count: 1 },
      answers: [{
        field_id: "critical-service", label: "Service criticality", type: "SINGLE_SELECT", required: true, visibility: "VISIBLE", value: { text: "Important" },
        provenance: { origin: "RESPONDENT_CORRECTED", binding_id: "binding-sensitive", binding_version: 7, source_receipt: { source_id: "procurement", observed_at: "2026-08-27T09:30:00Z" } },
      }],
      coverage: { visible_fields: 1, answered_fields: 1, required_fields: 1, answered_required: 1, ratio: 1 },
      documents: [{ field_id: "security-report", artifact_id: "artifact-sensitive", file_name: "security-report.pdf", media_type: "application/pdf", size_bytes: 64000, artifact_status: "QUARANTINED", evidence_class: "VENDOR_SUPPLIED", document_type: "SECURITY_TEST" }],
      matters: [],
    };
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("UNDER_REVIEW")} review={review} form={form} onComplete={vi.fn()}/>);

    expect(screen.getByText("Updated by the vendor · Source value observed 27 Aug 2026")).toBeTruthy();
    expect(screen.getByText("Quarantined by security scan · Vendor supplied evidence")).toBeTruthy();
    expect(screen.queryByText("binding-sensitive")).toBeNull();
    expect(screen.queryByText("artifact-sensitive")).toBeNull();
    expect(primaryActions()).toHaveLength(1);
  });

  it("constrains response and next-review dates to the server review window", () => {
    const { rerender } = render(<VendorDueDiligence relationship={relationship} assessment={assessment("READY_TO_SEND")} form={form} onSend={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", { name: "Send due diligence request" }));
    const responseDue = screen.getByLabelText("Response due date", { exact: false }) as HTMLInputElement;
    expect(responseDue.min).not.toBe("");
    expect(responseDue.max).toBe("2026-09-30");

    rerender(<VendorDueDiligence relationship={relationship} assessment={assessment("UNDER_REVIEW")} form={form} onComplete={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", { name: "Record assessment conclusion" }));
    expect((screen.getByLabelText("Recommended next review", { exact: false }) as HTMLInputElement).min).not.toBe("");
  });

  it("shows scoped loading, unavailable and source-warning states", () => {
    const { rerender } = render(<VendorDueDiligence relationship={relationship} assessment={null} viewState="loading" form={form}/>);
    expect(screen.getByText("Loading due diligence for Card transaction processing…")).toBeTruthy();

    rerender(<VendorDueDiligence relationship={relationship} assessment={null} viewState="unavailable" form={form} onRefresh={vi.fn()}/>);
    expect(screen.getByRole("alert").textContent).toContain("Due diligence is unavailable");
    expect(screen.getByRole("button", { name: "Try again" })).toBeTruthy();

    rerender(<VendorDueDiligence relationship={relationship} assessment={assessment("READY_TO_SEND")} form={form} sourceStatus={{ state: "STALE", detail: "Procurement data was last received on 20 Aug 2026." }} onSend={vi.fn()}/>);
    expect(screen.getByRole("status").textContent).toContain("Procurement data was last received on 20 Aug 2026.");
    expect((screen.getByRole("button", { name: "Send due diligence request" }) as HTMLButtonElement).disabled).toBe(false);
  });

  it("offers an explicit restart for cancelled onboarding without presenting it as a first assessment", () => {
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("CANCELLED")} form={form} onStart={vi.fn()}/>);

    expect(screen.getByText("This onboarding assessment was cancelled. The relationship remains available for a new review.")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Start due diligence" })).toBeNull();
    expect(screen.getByRole("button", { name: "Restart due diligence" })).toBeTruthy();
    expect(primaryActions()).toHaveLength(1);
  });

  it("requires a reason before cancelling an active assessment", async () => {
    const onCancelAssessment = vi.fn().mockResolvedValue({ ...assessment("READY_TO_SEND"), status: "CANCELLED", version: 3, cancellation_reason: "The service is no longer being procured." });
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("READY_TO_SEND")} form={form} onSend={vi.fn()} onCancelAssessment={onCancelAssessment}/>);

    fireEvent.click(screen.getByRole("button", { name: "Cancel assessment" }));
    const confirm = screen.getByRole("button", { name: "Cancel assessment" });
    expect((confirm as HTMLButtonElement).disabled).toBe(true);
    fireEvent.change(screen.getByLabelText("Reason for cancellation", { exact: false }), { target: { value: "The service is no longer being procured." } });
    fireEvent.click(confirm);

    await waitFor(() => expect(onCancelAssessment).toHaveBeenCalledWith("assessment-1", { expected_version: 3, reason: "The service is no longer being procured." }));
    expect(await screen.findByText("Assessment cancelled. The vendor relationship was not changed.")).toBeTruthy();
  });
});
