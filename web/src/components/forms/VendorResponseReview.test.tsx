import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { VendorRelationshipAggregate } from "../../vendorTypes";
import type { VendorAssessmentReviewView } from "../../vendorAssessmentTypes";
import { VendorResponseReview } from "./VendorResponseReview";

const relationship = { vendor: { id: "vendor-1", tenant_id: "bank", legal_name: "Acme Limited", trading_name: "", registration_ref: "RC-1", jurisdiction: "Nigeria", source_id: "", external_ref: "", registered_address: "Old address", website_domain: "acme.example", status: "ACTIVE", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-08-01T00:00:00Z", version: 4 }, relationship: { id: "relationship-1", tenant_id: "bank", legal_entity_id: "entity", vendor_id: "vendor-1", service_name: "Payments", business_owner_principal_id: "owner-1", criticality: "IMPORTANT", privacy_role: "PROCESSOR", status: "ACTIVE", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-08-01T00:00:00Z", version: 2 } } as VendorRelationshipAggregate;
const review = { assessment: { id: "assessment-1", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "relationship-1", review_kind: "PERIODIC", source_trigger: "annual-2026", stable_episode_key: "episode", status: "UNDER_REVIEW", form_template_id: "form-1", form_template_version: 3, review_due_at: "2026-09-30T00:00:00Z", started_by_principal_id: "owner-1", started_at: "2026-08-01T00:00:00Z", submission_id: "submission-1", version: 8, created_at: "2026-08-01T00:00:00Z", updated_at: "2026-08-29T00:00:00Z" }, requests: [], response: { submission_id: "submission-1", request_id: "request-1", submitted_at: "2026-08-28T00:00:00Z", answer_count: 1, artifact_count: 0 }, answers: [{ field_id: "registered_address", label: "Registered address", type: "LONG_TEXT", required: true, visibility: "VISIBLE", baseline: { target_key: "VENDOR.IDENTITY.REGISTERED_ADDRESS", subject_type: "VENDOR_RELATIONSHIP", subject_id: "relationship-1", record_id: "vendor-1", record_version: 4, display_value: "Old address", source_label: "Validated vendor record", observed_or_confirmed_at: "2026-08-01T00:00:00Z" }, value: { text: "New address" }, provenance: { origin: "RESPONDENT_CORRECTED" } }], coverage: { visible_fields: 1, answered_fields: 1, required_fields: 1, answered_required: 1, ratio: 1 }, documents: [], matters: [] } as VendorAssessmentReviewView;

describe("VendorResponseReview", () => {
  it("requires a decision and rationale before applying the exact response revision", async () => {
    const onApply = vi.fn().mockResolvedValue({ receipt: { id: "receipt-1", assessment_id: "assessment-1", response_revision_id: "submission-1", vendor_id: "vendor-1", actor_principal_id: "reviewer-1", accepted_field_ids: ["registered_address"], rejected_field_ids: [], decisions: [], prior_vendor_version: 4, result_vendor_version: 5, result_assessment_version: 9, applied_at: "2026-08-29T12:00:00Z" }, review });
    render(<VendorResponseReview relationship={relationship} review={review} onApply={onApply}/>);
    const apply = screen.getByRole("button", { name: "Apply reviewed changes" }) as HTMLButtonElement;
    expect(apply.disabled).toBe(true);
    fireEvent.click(screen.getByLabelText("Accept submitted value"));
    fireEvent.change(screen.getByLabelText("Decision rationale"), { target: { value: "Vendor evidence supports the new office." } });
    expect(apply.disabled).toBe(false);
    fireEvent.click(apply);
    await waitFor(() => expect(onApply).toHaveBeenCalledWith("assessment-1", "submission-1", expect.objectContaining({ expected_assessment_version: 8, expected_submission_revision: 1 })));
    expect(await screen.findByRole("heading", { name: "Reviewed changes recorded" })).toBeTruthy();
  });

  it("blocks application when the current held version differs from the frozen request", () => {
    render(<VendorResponseReview relationship={{ ...relationship, vendor: { ...relationship.vendor, version: 5 } }} review={review} onApply={vi.fn()}/>);
    expect(screen.getByText("1 held record has changed")).toBeTruthy();
    expect((screen.getByRole("button", { name: "Apply reviewed changes" }) as HTMLButtonElement).disabled).toBe(true);
  });

  it("restores the immutable application receipt after the review is reloaded", () => {
    const receipt = { id: "receipt-1", assessment_id: "assessment-1", response_revision_id: "submission-1", vendor_id: "vendor-1", actor_principal_id: "reviewer-1", accepted_field_ids: ["registered_address"], rejected_field_ids: [], decisions: [], prior_vendor_version: 4, result_vendor_version: 5, result_assessment_version: 9, applied_at: "2026-08-29T12:00:00Z" };
    render(<VendorResponseReview relationship={relationship} review={{ ...review, application_receipt: receipt }} onApply={vi.fn()}/>);
    expect(screen.getByRole("heading", { name: "Reviewed changes recorded" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Apply reviewed changes" })).toBeNull();
  });
});
