import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import type { VendorAssessment, VendorAssessmentReviewView } from "../vendorAssessmentTypes";
import { VendorDueDiligence } from "./VendorDueDiligence";

const relationship: VendorRelationshipAggregate = {
  vendor: { id: "vendor-1", tenant_id: "bank", legal_name: "Acme Processing Limited", jurisdiction: "Nigeria", status: "ACTIVE", created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z", version: 1 },
  relationship: { id: "relationship-1", tenant_id: "bank", legal_entity_id: "entity", vendor_id: "vendor-1", service_name: "Card transaction processing", business_owner_principal_id: "owner-1", criticality: "IMPORTANT", privacy_role: "PROCESSOR", status: "PROPOSED", created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z", version: 4 },
};

function assessment(status: VendorAssessment["status"]): VendorAssessment {
  return {
    id: "assessment-1", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "relationship-1",
    review_kind: "ONBOARDING", stable_episode_key: "episode-1", status, form_template_id: "form-1", form_template_version: 3,
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

function primaryActions() {
  return screen.queryAllByRole("button").filter((button) => button.classList.contains("primary-button") && !button.hasAttribute("disabled"));
}

describe("VendorDueDiligence", () => {
  it("starts from a relationship-scoped preview with one primary action", async () => {
    const onStart = vi.fn().mockResolvedValue(assessment("SETUP_PENDING"));
    render(<VendorDueDiligence relationship={relationship} assessment={null} form={form} defaultReviewDueDate="2026-09-30" onStart={onStart}/>);

    expect(screen.getByRole("heading", { name: "Due diligence" })).toBeTruthy();
    expect(screen.getByText("Acme Processing Limited")).toBeTruthy();
    expect(screen.getAllByText("Card transaction processing")).toHaveLength(2);
    expect(primaryActions()).toHaveLength(1);
    fireEvent.click(screen.getByRole("button", { name: "Start due diligence" }));

    expect((screen.getByLabelText("Review due date") as HTMLInputElement).value).toBe("2026-09-30");
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
    fireEvent.change(screen.getByLabelText("Vendor contact email"), { target: { value: "security@vendor.example" } });
    fireEvent.change(screen.getByLabelText("Response due date"), { target: { value: "2026-09-20" } });
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

  it("keeps request status primary and sends a replacement link as a focused secondary action", async () => {
    const delivered = { assessment: { ...assessment("COLLECTING"), version: 4 }, request: { id: "request-1", status: "READY" }, state: "DELIVERED" as const };
    const onReissue = vi.fn().mockResolvedValue(delivered);
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("COLLECTING")} form={form} onOpenRequest={vi.fn()} onReissue={onReissue}/>);

    expect(screen.getByRole("button", { name: "Review request status" }).classList.contains("primary-button")).toBe(true);
    expect(screen.getByRole("button", { name: "Send replacement link" }).classList.contains("secondary-button")).toBe(true);
    expect(primaryActions()).toHaveLength(1);
    fireEvent.click(screen.getByRole("button", { name: "Send replacement link" }));
    fireEvent.change(screen.getByLabelText("Vendor contact email"), { target: { value: "security@vendor.example" } });
    fireEvent.change(screen.getByLabelText("Replacement link valid for"), { target: { value: "10080" } });
    fireEvent.click(screen.getByRole("button", { name: "Send replacement link" }));

    await waitFor(() => expect(onReissue).toHaveBeenCalledWith({ expected_version: 3, audience: "security@vendor.example", invitation_ttl_minutes: 10080 }));
    expect(await screen.findByText("Replacement link sent. Previous access to this request has ended.")).toBeTruthy();
    expect(screen.queryByText("security@vendor.example")).toBeNull();
    expect(screen.queryByDisplayValue("security@vendor.example")).toBeNull();
    expect(primaryActions()).toHaveLength(1);
  });

  it("clears the replacement recipient after a failed attempt", async () => {
    const onReissue = vi.fn().mockRejectedValue(new Error("unavailable"));
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("COLLECTING")} form={form} onOpenRequest={vi.fn()} onReissue={onReissue}/>);

    fireEvent.click(screen.getByRole("button", { name: "Send replacement link" }));
    fireEvent.change(screen.getByLabelText("Vendor contact email"), { target: { value: "security@vendor.example" } });
    fireEvent.click(screen.getByRole("button", { name: "Send replacement link" }));

    expect(await screen.findByText("The replacement link was not sent. Re-enter the vendor contact email before trying again.")).toBeTruthy();
    expect(screen.queryByText("security@vendor.example")).toBeNull();
    expect(screen.queryByDisplayValue("security@vendor.example")).toBeNull();
  });

  it("clears an invalid replacement recipient after validation", async () => {
    const onReissue = vi.fn();
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("COLLECTING")} form={form} onOpenRequest={vi.fn()} onReissue={onReissue}/>);

    fireEvent.click(screen.getByRole("button", { name: "Send replacement link" }));
    fireEvent.change(screen.getByLabelText("Vendor contact email"), { target: { value: "not-an-email" } });
    fireEvent.click(screen.getByRole("button", { name: "Send replacement link" }));

    expect(await screen.findByText("Enter a valid vendor contact email before sending the replacement link.")).toBeTruthy();
    expect((screen.getByLabelText("Vendor contact email") as HTMLInputElement).value).toBe("");
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

    fireEvent.click(screen.getByRole("button", { name: "Send replacement link" }));
    fireEvent.change(screen.getByLabelText("Vendor contact email"), { target: { value: "security@vendor.example" } });
    fireEvent.click(screen.getByRole("button", { name: "Send replacement link" }));

    fireEvent.click(await screen.findByRole("button", { name: "Copy replacement link" }));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("https://capture.example.test/?capture_invite=replacement-secret"));
    expect(screen.queryByText(/replacement-secret/)).toBeNull();
    expect(primaryActions()).toHaveLength(1);
  });

  it("keeps clarification and findings secondary to the review conclusion", () => {
    const review: VendorAssessmentReviewView = {
      assessment: assessment("UNDER_REVIEW"),
      requests: [],
      response: { submission_id: "submission-1", request_id: "request-1", submitted_at: "2026-08-28T11:00:00Z", answer_count: 14, artifact_count: 2 },
      answers: [],
      coverage: { visible_fields: 14, answered_fields: 14, required_fields: 10, answered_required: 10, ratio: 1 },
      documents: [{ field_id: "security-report", artifact_id: "artifact-1", file_name: "independent-security-test.pdf", media_type: "application/pdf", size_bytes: 64000, artifact_status: "AVAILABLE", evidence_class: "VENDOR_SUPPLIED", document_type: "SECURITY_TEST" }],
      matters: [{ matter_id: "finding-1", type: "VENDOR_DEFICIENCY", title: "Independent test renewal is due", status: "ACTION_AGREED" }],
    };
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("UNDER_REVIEW")} review={review} form={form} onRequestClarification={vi.fn()} onCreateDeficiency={vi.fn()} onReviewDocument={vi.fn()} onComplete={vi.fn()}/>);

    const reviewRegion = screen.getByRole("region", { name: "Vendor response review" });
    expect(within(reviewRegion).getByText(/14 answers · 2 documents/)).toBeTruthy();
    expect(within(reviewRegion).getByText("independent-security-test.pdf")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Request clarification" }).classList.contains("secondary-button")).toBe(true);
    expect(screen.getByRole("button", { name: "Record assessment conclusion" }).classList.contains("primary-button")).toBe(true);
    expect(primaryActions()).toHaveLength(1);
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
    const responseDue = screen.getByLabelText("Response due date") as HTMLInputElement;
    expect(responseDue.min).not.toBe("");
    expect(responseDue.max).toBe("2026-09-30");

    rerender(<VendorDueDiligence relationship={relationship} assessment={assessment("UNDER_REVIEW")} form={form} onComplete={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", { name: "Record assessment conclusion" }));
    expect((screen.getByLabelText("Recommended next review") as HTMLInputElement).min).not.toBe("");
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

  it("does not offer a false restart action for a cancelled onboarding episode", () => {
    render(<VendorDueDiligence relationship={relationship} assessment={assessment("CANCELLED")} form={form} onStart={vi.fn()}/>);

    expect(screen.getByText("This assessment was cancelled. A new onboarding review has not been started.")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Start due diligence" })).toBeNull();
    expect(primaryActions()).toHaveLength(0);
  });
});
