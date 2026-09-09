import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadVendorCollection, reconcileVendorEvidence, type VendorCollection } from "../vendorCollectionApi";
import { loadDocuments, type DocumentOccurrence } from "../submittedDocumentApi";
import { VendorEvidenceChecklist } from "./VendorEvidenceChecklist";
import { ApiError } from "../http";

vi.mock("../vendorCollectionApi", () => ({ loadVendorCollection: vi.fn(), reconcileVendorEvidence: vi.fn() }));
vi.mock("../submittedDocumentApi", async (original) => ({ ...await original<typeof import("../submittedDocumentApi")>(), loadDocuments: vi.fn() }));

const source: DocumentOccurrence = {
  id: "occurrence", artifact_id: "original-artifact", request_id: "original-request", submission_id: "original-submission", submission_channel: "MAGIC_LINK", field_id: "iso",
  relationship_id: "relationship", form_template_version: 1, form_title: "Earlier vendor review", field_label: "ISO certificate",
  file_name: "ISO certificate.pdf", file_kind: "PDF", media_type: "application/pdf", size_bytes: 300,
  sha256: "source-digest", artifact_status: "AVAILABLE", uploaded_at: "2026-08-01T10:00:00Z", submitted_at: "2026-08-01T10:30:00Z", current: true,
};
const collection: VendorCollection = {
  assessment_id: "assessment", assessment_version: 4, request_id: "request", request_version: 2, prepared: true, can_reconcile: true,
  observed_at: "2026-09-08T10:00:00Z", vendor_pending_count: 1, bank_pending_count: 1,
  fields: [
    { field_id: "accepted", label: "Accepted certificate", type: "vendor_document", required: true, collection_state: "RECEIVED", vendor_action_required: false, bank_review_state: "VALIDATED" },
    { field_id: "iso", label: "ISO 27001 assurance", type: "vendor_document", required: true, collection_state: "REUSED", vendor_action_required: false, bank_review_state: "PENDING", resolution: { id: "receipt", version: 1, source, reconciled_by: "reviewer", reconciled_at: "2026-09-08T09:00:00Z", rationale: "Covers the same service." } },
    { field_id: "vapt", label: "Vulnerability test report", type: "vendor_document", required: true, collection_state: "MISSING", vendor_action_required: true, bank_review_state: "NOT_REQUIRED" },
    { field_id: "privacy", label: "Privacy assessment", type: "vendor_document", required: true, collection_state: "CONDITION_UNKNOWN", vendor_action_required: false, bank_review_state: "NOT_REQUIRED" },
  ],
};
beforeEach(() => {
  vi.mocked(loadVendorCollection).mockReset().mockResolvedValue(structuredClone(collection));
  vi.mocked(reconcileVendorEvidence).mockReset();
  vi.mocked(loadDocuments).mockReset().mockResolvedValue({ items: [source] });
});

describe("VendorEvidenceChecklist", () => {
  it.each([false, true])("requires explicit demo permission for an unscanned checklist review (allowed=%s)", async (allowed) => {
    const review = vi.fn();
    const file = { field_id: "iso", artifact_id: "artifact", file_name: "Unscanned report.pdf", media_type: "application/pdf", size_bytes: 300, artifact_status: "STORED_UNSCANNED", demo_unscanned_allowed: allowed, status: "SUBMITTED", evidence_class: "VENDOR_SUPPLIED", document_type: "ISO_27001" };
    render(<VendorEvidenceChecklist assessmentID="assessment" assessmentVersion={4} relationshipID="relationship" documents={[file]} onReviewDocument={review}/>);
    const item = await screen.findByRole("article", { name: "ISO 27001 assurance" });
    const button = within(item).getByRole("button", { name: "Review document" });
    expect(button.hasAttribute("disabled")).toBe(!allowed);
    fireEvent.click(button);
    if (allowed) expect(review).toHaveBeenCalledWith(file, "VALIDATE");
    else expect(review).not.toHaveBeenCalled();
  });
  it("separates receipt from acceptance and places pending work before accepted evidence", async () => {
    render(<VendorEvidenceChecklist assessmentID="assessment" assessmentVersion={4} relationshipID="relationship"/>);
    const reused = await screen.findByRole("article", { name: "ISO 27001 assurance" });
    expect(within(reused).getByText("Awaiting review")).toBeTruthy();
    expect(within(reused).getByText(/No upload needed/)).toBeTruthy();
    expect(screen.getByRole("button", { name: "Missing 1" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Awaiting review 1" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Accepted 1" })).toBeTruthy();
    expect(screen.getAllByRole("article").at(-1)?.getAttribute("aria-label")).toBe("Accepted certificate");
    fireEvent.click(screen.getByRole("button", { name: "Missing 1" }));
    expect(screen.getAllByRole("article")).toHaveLength(1);
    expect(screen.getByRole("article", { name: "Vulnerability test report" })).toBeTruthy();
  });

  it("keeps unknown applicability distinct from evidence that passed", async () => {
    vi.mocked(loadVendorCollection).mockResolvedValue({ ...collection, vendor_pending_count: 2, fields: collection.fields.map((field) => field.field_id === "privacy" ? { ...field, vendor_action_required: true } : field) });
    render(<VendorEvidenceChecklist assessmentID="assessment" assessmentVersion={4} relationshipID="relationship"/>);
    const row = await screen.findByRole("article", { name: "Privacy assessment" });
    expect(within(row).getByText("Applicability pending")).toBeTruthy();
    expect(within(row).queryByText("Missing")).toBeNull();
    expect(within(row).queryByRole("button", { name: "Use existing document" })).toBeNull();
    expect(within(row).queryByText("Accepted")).toBeNull();
  });

  it.each([false, true])("saves the exact eligible existing occurrence with a rationale (demo=%s)", async (demo) => {
    vi.mocked(loadDocuments).mockResolvedValue({ items: [{ ...source, artifact_status: demo ? "STORED_UNSCANNED" : "AVAILABLE", demo_unscanned_allowed: demo }] });
    const onChanged = vi.fn();
    const receipt = { id: "new-receipt", version: 1, source, reconciled_by: "reviewer", reconciled_at: "2026-09-08T11:00:00Z", rationale: "Includes the service in this review." };
    vi.mocked(reconcileVendorEvidence).mockResolvedValue({ receipt, collection: { ...collection, assessment_version: 5, request_version: 3, vendor_pending_count: 0, bank_pending_count: 2, fields: collection.fields.map((field) => field.field_id === "vapt" ? { ...field, collection_state: "REUSED", vendor_action_required: false, bank_review_state: "PENDING", resolution: receipt } : field) } });
    render(<VendorEvidenceChecklist assessmentID="assessment" assessmentVersion={4} relationshipID="relationship" onChanged={onChanged}/>);
    const row = await screen.findByRole("article", { name: "Vulnerability test report" });
    fireEvent.click(within(row).getByRole("button", { name: "Use existing document" }));
    fireEvent.click(await screen.findByRole("row", { name: /ISO certificate.pdf/ }));
    fireEvent.click(screen.getByRole("button", { name: "Choose this document" }));
    if (demo) expect(screen.getByText("Unscanned file. Review is enabled in demo mode.")).toBeTruthy();
    fireEvent.change(screen.getByRole("textbox", { name: "Reason for reuse" }), { target: { value: receipt.rationale } });
    fireEvent.click(screen.getByRole("button", { name: "Use this document" }));
    await waitFor(() => expect(reconcileVendorEvidence).toHaveBeenCalledWith("assessment", "vapt", {
      expected_version: 4, request_id: "request", expected_request_version: 2, source_submission_id: "original-submission", source_field_id: "iso", source_artifact_id: "original-artifact", source_response_revision_id: undefined, rationale: receipt.rationale,
    }));
    expect(await screen.findByRole("button", { name: "Missing 0" })).toBeTruthy();
    expect(onChanged).toHaveBeenCalledOnce();
    expect(loadDocuments).toHaveBeenCalledWith(expect.objectContaining({ relationship_id: "relationship" }), expect.any(AbortSignal));
  });

  it("does not label a committed reconciliation as failed when parent refresh fails", async () => {
    vi.mocked(reconcileVendorEvidence).mockResolvedValue({ collection, receipt: collection.fields[1]!.resolution! });
    render(<VendorEvidenceChecklist assessmentID="assessment" assessmentVersion={4} relationshipID="relationship" onChanged={() => Promise.reject(new Error("refresh offline"))}/>);
    fireEvent.click(within(await screen.findByRole("article", { name: "Vulnerability test report" })).getByRole("button", { name: "Use existing document" }));
    fireEvent.click(await screen.findByRole("row", { name: /ISO certificate.pdf/ }));
    fireEvent.click(screen.getByRole("button", { name: "Choose this document" }));
    fireEvent.change(screen.getByRole("textbox", { name: "Reason for reuse" }), { target: { value: "Covers this service." } });
    fireEvent.click(screen.getByRole("button", { name: "Use this document" }));
    expect(await screen.findByText(/Document linked/)).toBeTruthy();
    expect(await screen.findByText(/Reload the vendor review/)).toBeTruthy();
    expect(screen.queryByText(/could not be linked/)).toBeNull();
  });

  it("shows unavailable counts and retry instead of reporting zero pending work", async () => {
    vi.mocked(loadVendorCollection).mockRejectedValueOnce(new Error("offline"));
    render(<VendorEvidenceChecklist assessmentID="assessment" assessmentVersion={4} relationshipID="relationship"/>);
    expect(await screen.findByText("Checklist unavailable.")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Missing 0" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Reload checklist" }));
    expect(await screen.findByRole("button", { name: "Missing 1" })).toBeTruthy();
  });

  it("does not expose reconciliation controls without current reviewer authority", async () => {
    vi.mocked(loadVendorCollection).mockResolvedValue({ ...collection, can_reconcile: false });
    render(<VendorEvidenceChecklist assessmentID="assessment" assessmentVersion={4} relationshipID="relationship"/>);
    await screen.findByRole("article", { name: "Vulnerability test report" });
    expect(screen.queryByRole("button", { name: "Use existing document" })).toBeNull();
  });

  it("identifies a replaced source and keeps it out of accepted evidence", async () => {
    vi.mocked(loadVendorCollection).mockResolvedValue({ ...collection, fields: [{ ...collection.fields[1]!, collection_state: "MISSING", vendor_action_required: true, bank_review_state: "NOT_REQUIRED", resolution: { ...collection.fields[1]!.resolution!, source: { ...source, current: false } } }], vendor_pending_count: 1, bank_pending_count: 0 });
    render(<VendorEvidenceChecklist assessmentID="assessment" assessmentVersion={4} relationshipID="relationship"/>);
    expect(await screen.findByText("Document replaced. Link the current version.")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Accepted 0" })).toBeTruthy();
    expect(screen.queryByText(/No upload needed/)).toBeNull();
  });

  it.each([
    { submission_channel: "STAFF", explanation: /submitted by the vendor/ },
    { current: false, explanation: /has been replaced/ },
    { expires_on: "2025-01-01", explanation: /expired/ },
    { artifact_status: "QUARANTINED", explanation: /not available/ },
    { artifact_status: "QUARANTINED", demo_unscanned_allowed: true, explanation: /not available/ },
    { artifact_status: "STORED_UNSCANNED", demo_preview_available: true, explanation: /not available/ },
    { artifact_status: "STORED_UNSCANNED", demo_unscanned_allowed: true, current: false, explanation: /has been replaced/ },
    { artifact_status: "STORED_UNSCANNED", demo_unscanned_allowed: true, expires_on: "2025-01-01", explanation: /expired/ },
  ])("prevents reuse of an unsuitable source: $explanation", async ({ explanation, ...change }) => {
    vi.mocked(loadDocuments).mockResolvedValue({ items: [{ ...source, ...change }] });
    render(<VendorEvidenceChecklist assessmentID="assessment" assessmentVersion={4} relationshipID="relationship"/>);
    fireEvent.click(within(await screen.findByRole("article", { name: "Vulnerability test report" })).getByRole("button", { name: "Use existing document" }));
    fireEvent.click(await screen.findByRole("row", { name: /ISO certificate.pdf/ }));
    expect(screen.getByText(explanation)).toBeTruthy();
    expect(screen.getByRole("button", { name: "Choose this document" }).hasAttribute("disabled")).toBe(true);
    expect(reconcileVendorEvidence).not.toHaveBeenCalled();
  });

  it("requires a fresh source choice after a version conflict", async () => {
    vi.mocked(reconcileVendorEvidence).mockRejectedValue(new ApiError(409, "Changed"));
    render(<VendorEvidenceChecklist assessmentID="assessment" assessmentVersion={4} relationshipID="relationship"/>);
    fireEvent.click(within(await screen.findByRole("article", { name: "Vulnerability test report" })).getByRole("button", { name: "Use existing document" }));
    fireEvent.click(await screen.findByRole("row", { name: /ISO certificate.pdf/ }));
    fireEvent.click(screen.getByRole("button", { name: "Choose this document" }));
    fireEvent.change(screen.getByRole("textbox", { name: "Reason for reuse" }), { target: { value: "Covers this service." } });
    fireEvent.click(screen.getByRole("button", { name: "Use this document" }));
    expect(await screen.findByText(/request or evidence changed/)).toBeTruthy();
    expect(screen.getByRole("button", { name: "Use this document" }).hasAttribute("disabled")).toBe(true);
    fireEvent.click(screen.getByRole("button", { name: "Reload checklist" }));
    await waitFor(() => expect(loadVendorCollection).toHaveBeenCalledTimes(2));
    expect(screen.queryByRole("dialog")).toBeNull();
  });
});
