import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { VendorDocuments } from "./VendorDocuments";
const authority = vi.hoisted(() => ({ checkVendorAuthority: vi.fn() }));
const assessment = vi.hoisted(() => ({ reviewVendorAssessmentDocument: vi.fn(), vendorAssessmentDocumentURL: vi.fn() }));
vi.mock("./vendorAuthority", () => authority);
vi.mock("../vendorAssessmentApi", () => assessment);

it("records an independently authorized decision against the assessment version", async () => {
  authority.checkVendorAuthority.mockResolvedValue(true);
  assessment.reviewVendorAssessmentDocument.mockResolvedValue({ assessment: { id: "a", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel", status: "UNDER_REVIEW", version: 9 }, documents: [{ field_id: "f", artifact_id: "doc", request_id: "req", status: "VALIDATED", document_type: "AUDIT", evidence_class: "VENDOR_SUPPLIED" }] });
  const changed = vi.fn();
  render(<VendorDocuments assessment={{ id: "a", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel", version: 8, status: "UNDER_REVIEW" } as any} historical={false} documents={[{ artifact_id: "doc", field_id: "f", request_id: "req", file_name: "Audit.pdf", media_type: "application/pdf", size_bytes: 1, artifact_status: "AVAILABLE", document_type: "AUDIT", status: "SUBMITTED", evidence_class: "VENDOR_SUPPLIED" }]} onChanged={changed} onFailed={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Review Audit.pdf" }));
  fireEvent.change(screen.getByLabelText("Decision"), { target: { value: "VALIDATE" } });
  fireEvent.change(screen.getByLabelText("Document source"), { target: { value: "VENDOR_SUPPLIED" } });
  fireEvent.click(screen.getByRole("button", { name: "Save document review" }));
  await vi.waitFor(() => expect(assessment.reviewVendorAssessmentDocument).toHaveBeenCalledWith("a", "doc", { expected_version: 8, field_id: "f", decision: "VALIDATE", document_type: "AUDIT", evidence_class: "VENDOR_SUPPLIED" }));
  expect(changed).toHaveBeenCalled();
});

it("does not confirm a checklist decision for a different document", async () => {
  authority.checkVendorAuthority.mockResolvedValue(true);
  assessment.reviewVendorAssessmentDocument.mockResolvedValue({ assessment: { id: "a", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel", status: "UNDER_REVIEW", version: 9 }, documents: [{ field_id: "f", artifact_id: "other", request_id: "req", status: "VALIDATED", document_type: "AUDIT", evidence_class: "VENDOR_SUPPLIED" }] });
  const changed = vi.fn(); const failed = vi.fn();
  render(<VendorDocuments assessment={{ id: "a", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel", version: 8, status: "UNDER_REVIEW" } as any} historical={false} documents={[{ artifact_id: "doc", field_id: "f", request_id: "req", file_name: "Audit.pdf", media_type: "application/pdf", size_bytes: 1, artifact_status: "AVAILABLE", document_type: "AUDIT", status: "SUBMITTED", evidence_class: "VENDOR_SUPPLIED" }]} onChanged={changed} onFailed={failed}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Review Audit.pdf" }));
  fireEvent.change(screen.getByLabelText("Decision"), { target: { value: "VALIDATE" } });
  fireEvent.change(screen.getByLabelText("Document source"), { target: { value: "VENDOR_SUPPLIED" } });
  fireEvent.click(screen.getByRole("button", { name: "Save document review" }));
  await vi.waitFor(() => expect(failed).toHaveBeenCalled());
  expect(changed).not.toHaveBeenCalled();
});

it.each([{ document_type: "OTHER", evidence_class: "VENDOR_SUPPLIED", expires_on: "2027-03-01" }, { document_type: "AUDIT", evidence_class: "OFFICIAL_SOURCE", expires_on: "2027-03-01" }, { document_type: "AUDIT", evidence_class: "VENDOR_SUPPLIED", expires_on: "2027-04-01" }])("does not confirm a document decision with different metadata", async (metadata) => {
  authority.checkVendorAuthority.mockResolvedValue(true);
  assessment.reviewVendorAssessmentDocument.mockResolvedValue({ assessment: { id: "a", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel", status: "UNDER_REVIEW", version: 9 }, documents: [{ field_id: "f", artifact_id: "doc", request_id: "req", status: "VALIDATED", ...metadata }] });
  const changed = vi.fn(); const failed = vi.fn();
  render(<VendorDocuments assessment={{ id: "a", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel", version: 8, status: "UNDER_REVIEW" } as any} historical={false} documents={[{ artifact_id: "doc", field_id: "f", request_id: "req", file_name: "Audit.pdf", media_type: "application/pdf", size_bytes: 1, artifact_status: "AVAILABLE", document_type: "AUDIT", status: "SUBMITTED", evidence_class: "VENDOR_SUPPLIED" }]} onChanged={changed} onFailed={failed}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Review Audit.pdf" }));
  fireEvent.change(screen.getByLabelText("Decision"), { target: { value: "VALIDATE" } });
  fireEvent.change(screen.getByLabelText("Document source"), { target: { value: "VENDOR_SUPPLIED" } });
  fireEvent.change(screen.getByLabelText("Valid until"), { target: { value: "2027-03-01" } });
  fireEvent.click(screen.getByRole("button", { name: "Save document review" }));
  await vi.waitFor(() => expect(failed).toHaveBeenCalled());
  expect(changed).not.toHaveBeenCalled();
});

it("locks checklist review while the command is pending", async () => {
  authority.checkVendorAuthority.mockResolvedValue(true);
  let resolve!: (value: unknown) => void;
  assessment.reviewVendorAssessmentDocument.mockReturnValue(new Promise((done) => { resolve = done; }));
  const pending = vi.fn();
  render(<VendorDocuments assessment={{ id: "a", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel", version: 8, status: "UNDER_REVIEW" } as any} historical={false} documents={[{ artifact_id: "doc", field_id: "f", request_id: "req", file_name: "Audit.pdf", media_type: "application/pdf", size_bytes: 1, artifact_status: "AVAILABLE", document_type: "AUDIT", status: "SUBMITTED", evidence_class: "VENDOR_SUPPLIED" }]} onChanged={() => {}} onFailed={() => {}} onPendingChange={pending}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Review Audit.pdf" }));
  fireEvent.change(screen.getByLabelText("Decision"), { target: { value: "VALIDATE" } });
  fireEvent.change(screen.getByLabelText("Document source"), { target: { value: "VENDOR_SUPPLIED" } });
  const form = screen.getByRole("button", { name: "Save document review" }).closest("form")!;
  fireEvent.submit(form); fireEvent.submit(form);
  expect(assessment.reviewVendorAssessmentDocument).toHaveBeenCalledTimes(1);
  expect(pending).toHaveBeenCalledWith(true);
  resolve({ assessment: { id: "a", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel", status: "UNDER_REVIEW", version: 9 }, documents: [{ field_id: "f", artifact_id: "doc", request_id: "req", status: "VALIDATED", document_type: "AUDIT", evidence_class: "VENDOR_SUPPLIED" }] });
  await vi.waitFor(() => expect(pending).toHaveBeenCalledWith(false));
});

it("does not offer a document decision without an authority route", async () => {
  authority.checkVendorAuthority.mockResolvedValue(false);
  render(<VendorDocuments assessment={{ id: "a", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel", version: 8, status: "UNDER_REVIEW" } as any} historical={false} documents={[{ artifact_id: "doc", field_id: "f", request_id: "req", file_name: "Audit.pdf", media_type: "application/pdf", size_bytes: 1, artifact_status: "AVAILABLE", document_type: "AUDIT", status: "SUBMITTED", evidence_class: "VENDOR_SUPPLIED" }]} onChanged={() => {}} onFailed={() => {}}/>);
  expect(await screen.findByText("Document review unavailable. Check your role.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Review Audit.pdf" })).toBeNull();
});

it("withholds review for superseded and unknown document states", async () => {
  authority.checkVendorAuthority.mockResolvedValue(true);
  const base = { field_id: "f", request_id: "req", media_type: "application/pdf", size_bytes: 1, artifact_status: "AVAILABLE", document_type: "AUDIT", evidence_class: "VENDOR_SUPPLIED" };
  render(<VendorDocuments assessment={{ id: "a", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel", version: 8, status: "UNDER_REVIEW" } as any} historical={false} documents={[{...base, artifact_id: "old", file_name: "Old.pdf", status: "SUPERSEDED"}, {...base, artifact_id: "unknown", file_name: "Unknown.pdf"}]} onChanged={() => {}} onFailed={() => {}}/>);
  expect(await screen.findByText("Superseded")).toBeTruthy();
  await vi.waitFor(() => expect(authority.checkVendorAuthority).toHaveBeenCalled());
  expect(screen.queryByRole("button", { name: "Review Old.pdf" })).toBeNull();
  expect(screen.queryByRole("button", { name: "Review Unknown.pdf" })).toBeNull();
});

it("shows document provenance and records the selected expiry", async () => {
  authority.checkVendorAuthority.mockResolvedValue(true);
  assessment.reviewVendorAssessmentDocument.mockResolvedValue({ assessment: { id: "a", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel", status: "UNDER_REVIEW", version: 9 }, documents: [{ field_id: "f", artifact_id: "doc", status: "VALIDATED", document_type: "AUDIT", evidence_class: "VENDOR_SUPPLIED", expires_on: "2027-03-01" }] });
  render(<VendorDocuments assessment={{ id: "a", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel", version: 8, status: "UNDER_REVIEW" } as any} historical={false} documents={[{ artifact_id: "doc", field_id: "f", file_name: "Audit.pdf", media_type: "application/pdf", size_bytes: 1, artifact_status: "AVAILABLE", document_type: "AUDIT", status: "SUBMITTED", evidence_class: "VENDOR_SUPPLIED", reference: "Ref 123", issued_by: "Acme Audit", issued_on: "2026-01-01", expires_on: "2027-01-01" }]} onChanged={() => {}} onFailed={() => {}}/>);
  expect(screen.getByText(/Ref 123/)).toBeTruthy();
  expect(screen.getByText(/Acme Audit/)).toBeTruthy();
  expect(screen.getByText(/2026/)).toBeTruthy();
  expect(screen.getByText(/2027/)).toBeTruthy();
  fireEvent.click(await screen.findByRole("button", { name: "Review Audit.pdf" }));
  fireEvent.change(screen.getByLabelText("Decision"), { target: { value: "VALIDATE" } });
  fireEvent.change(screen.getByLabelText("Document source"), { target: { value: "VENDOR_SUPPLIED" } });
  fireEvent.change(screen.getByLabelText("Valid until"), { target: { value: "2027-03-01" } });
  fireEvent.click(screen.getByRole("button", { name: "Save document review" }));
  await vi.waitFor(() => expect(assessment.reviewVendorAssessmentDocument).toHaveBeenCalledWith("a", "doc", expect.objectContaining({ valid_until: "2027-03-01" })));
});

it("discloses a demo unscanned exception before download or review", async () => {
  authority.checkVendorAuthority.mockResolvedValue(true);
  render(<VendorDocuments assessment={{ id: "a", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel", version: 8, status: "UNDER_REVIEW" } as any} historical={false} documents={[{ artifact_id: "doc", field_id: "f", request_id: "req", file_name: "Audit.pdf", media_type: "application/pdf", size_bytes: 1, artifact_status: "STORED_UNSCANNED", demo_unscanned_allowed: true, document_type: "AUDIT", status: "SUBMITTED", evidence_class: "VENDOR_SUPPLIED" }]} onChanged={() => {}} onFailed={() => {}}/>);
  expect(await screen.findByText("Demo file. No antivirus scan was performed.")).toBeTruthy();
  expect(await screen.findByRole("button", { name: "Review Audit.pdf" })).toBeTruthy();
  expect(screen.getByRole("button", { name: "Download Audit.pdf" })).toBeTruthy();
});
