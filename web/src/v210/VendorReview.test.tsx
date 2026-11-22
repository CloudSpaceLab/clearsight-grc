import { act, fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { VendorReview } from "./VendorReview";
import { ApiError } from "../http";

const assessment = vi.hoisted(() => ({ loadCurrentVendorAssessment: vi.fn(), loadVendorAssessment: vi.fn(), startVendorAssessment: vi.fn(), startVendorAssessmentReview: vi.fn(), completeVendorAssessment: vi.fn(), sendVendorAssessmentRequest: vi.fn(), retryVendorAssessmentSetup: vi.fn(), reissueVendorAssessmentRequest: vi.fn(), requestVendorAssessmentClarification: vi.fn() }));
const forms = vi.hoisted(() => ({ loadFormTemplatePage: vi.fn(), loadFormTemplateRevision: vi.fn() }));
const authority = vi.hoisted(() => ({ checkVendorAuthority: vi.fn() }));
const collections = vi.hoisted(() => ({ loadVendorCollection: vi.fn(), reconcileVendorEvidence: vi.fn() }));
const submittedDocuments = vi.hoisted(() => ({ loadDocuments: vi.fn() }));
vi.mock("../vendorAssessmentApi", () => assessment);
vi.mock("../formsApi", () => forms);
vi.mock("./vendorAuthority", () => authority);
vi.mock("../vendorCollectionApi", () => collections);
vi.mock("../submittedDocumentApi", async (importOriginal) => ({ ...await importOriginal<typeof import("../submittedDocumentApi")>(), ...submittedDocuments }));
vi.mock("./VendorActivation", () => ({ VendorActivation: () => null }));
const relationship = { vendor: { legal_name: "Atlas Ltd" }, relationship: { id: "rel-1", tenant_id: "bank", legal_entity_id: "entity", version: 3, service_name: "Hosting", status: "PROPOSED", criticality: "IMPORTANT", privacy_role: "PROCESSOR" } } as any;
beforeEach(() => { vi.clearAllMocks(); collections.loadVendorCollection.mockResolvedValue({ assessment_id: "assess-1", assessment_version: 7, fields: [], can_reconcile: false }); forms.loadFormTemplatePage.mockResolvedValue({ items: [{ template: { id: "form-1", name: "Security", code: "F-01" }, active_version: 2, active_status: "ACTIVE" }] }); authority.checkVendorAuthority.mockResolvedValue(true); });

it("opens Documents for the selected vendor and returns to its assessment", async () => {
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: null });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: { id: "assess-1", relationship_id: "rel-1", version: 1, status: "SUBMITTED", review_kind: "ONBOARDING" }, answers: [], documents: [], matters: [], coverage: {} });
  render(<VendorReview relationship={relationship} assessmentId="assess-1" onBack={() => {}}/>);
  const link = await screen.findByRole("link", { name: "View vendor documents" });
  const [path, query] = link.getAttribute("href")!.split("?");
  expect(path).toBe("#/documents");
  expect(new URLSearchParams(query).get("relationship_id")).toBe("rel-1");
  expect(new URLSearchParams(query).get("return_to")).toBe("#/vendors/rel-1/assessments/assess-1");
});

it("prevents assessment actions while linking a document", async () => {
  const current = { id: "assess-1", relationship_id: "rel-1", version: 7, status: "SUBMITTED", review_kind: "ONBOARDING", current_request_id: "req-1" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: current, answers: [], documents: [], matters: [], coverage: {} });
  collections.loadVendorCollection.mockResolvedValue({ assessment_id: "assess-1", assessment_version: 7, request_id: "req-1", request_version: 2, can_reconcile: true, fields: [{ field_id: "field", label: "Audit report", type: "vendor_document", collection_state: "MISSING", bank_review_state: "PENDING" }] });
  submittedDocuments.loadDocuments.mockResolvedValue({ items: [{ id: "doc", relationship_id: "rel-1", current: true, submission_channel: "MAGIC_LINK", submission_id: "s", field_id: "f", artifact_id: "a", artifact_status: "AVAILABLE", file_name: "Report.pdf", submitted_at: "2026-11-20" }] });
  let reject!: (error: Error) => void;
  collections.reconcileVendorEvidence.mockReturnValue(new Promise((_resolve, fail) => { reject = fail; }));
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  const start = await screen.findByRole("button", { name: "Start review" });
  fireEvent.click(await screen.findByRole("button", { name: "Select document for Audit report" }));
  fireEvent.click(await screen.findByLabelText(/Report.pdf/));
  fireEvent.change(screen.getByLabelText("Reason"), { target: { value: "Same service" } });
  fireEvent.click(screen.getByRole("button", { name: "Link document" }));
  expect(start.matches(":disabled")).toBe(true);
  fireEvent.click(start);
  expect(assessment.startVendorAssessmentReview).not.toHaveBeenCalled();
  reject(new Error("connection lost"));
  expect(await screen.findByText(/Review status could not be confirmed/)).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Start review" })).toBeNull();
});

it("opens the matching document review directly from the checklist", async () => {
  const current = { id: "assess-1", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel-1", version: 7, status: "UNDER_REVIEW", review_kind: "ONBOARDING", current_request_id: "req-1" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: current, answers: [], documents: [{ field_id: "field", request_id: "req-1", artifact_id: "doc", file_name: "Audit.pdf", artifact_status: "AVAILABLE", status: "SUBMITTED", document_type: "AUDIT" }], matters: [], coverage: {} });
  collections.loadVendorCollection.mockResolvedValue({ assessment_id: "assess-1", assessment_version: 7, request_id: "req-1", request_version: 2, fields: [{ field_id: "field", label: "Audit report", type: "vendor_document", collection_state: "RECEIVED", bank_review_state: "PENDING" }] });
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Review Audit report" }));
  expect(await screen.findByRole("heading", { name: "Review Audit.pdf" })).toBeTruthy();
  expect(screen.getByLabelText("Document type")).toHaveProperty("value", "AUDIT");
});

it.each([{ request: "old-request", version: 7, allowed: true }, { request: "req-1", version: 6, allowed: true }, { request: "req-1", version: 7, allowed: false }])("withholds checklist review for stale collection or unavailable authority", async ({ request, version, allowed }) => {
  const current = { id: "assess-1", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel-1", version: 7, status: "UNDER_REVIEW", review_kind: "ONBOARDING", current_request_id: "req-1" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: current, answers: [], documents: [{ field_id: "field", request_id: "req-1", artifact_id: "doc", file_name: "Audit.pdf", artifact_status: "AVAILABLE", status: "SUBMITTED", document_type: "AUDIT" }], matters: [], coverage: {} });
  collections.loadVendorCollection.mockResolvedValue({ assessment_id: "assess-1", assessment_version: version, request_id: request, request_version: 2, fields: [{ field_id: "field", label: "Audit report", type: "vendor_document", collection_state: "RECEIVED", bank_review_state: "PENDING" }] });
  authority.checkVendorAuthority.mockResolvedValue(allowed);
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  expect(await screen.findByText("Audit report")).toBeTruthy();
  await vi.waitFor(() => expect(authority.checkVendorAuthority).toHaveBeenCalled());
  expect(screen.queryByRole("button", { name: "Review Audit report" })).toBeNull();
});

it("starts onboarding against the selected form and current relationship version", async () => {
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: null });
  assessment.startVendorAssessment.mockResolvedValue({ id: "assess-1", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel-1", version: 1, status: "SETUP_PENDING", review_kind: "ONBOARDING", source_trigger: "INITIAL", form_template_id: "form-1", form_template_version: 2, review_due_at: new Date("2027-02-18T12:00").toISOString(), scope_kind: "FULL" });
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Start onboarding assessment" }));
  fireEvent.change(await screen.findByLabelText("Assessment form"), { target: { value: "form-1" } });
  fireEvent.change(screen.getByLabelText("Review due"), { target: { value: "2027-02-18T12:00" } });
  fireEvent.click(screen.getByRole("button", { name: "Start assessment" }));
  await vi.waitFor(() => expect(assessment.startVendorAssessment).toHaveBeenCalledWith("rel-1", expect.objectContaining({ relationship_version: 3, form_template_id: "form-1", form_template_version: 2, review_kind: "ONBOARDING" })));
});

it("starts an event-driven reassessment with an explicit reference", async () => {
  const managed = { ...relationship, relationship: { ...relationship.relationship, status: "ACTIVE" } };
  const completed = { id: "old", relationship_id: "rel-1", version: 7, status: "COMPLETED", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: completed });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: completed, answers: [], documents: [], matters: [], coverage: {} });
  assessment.startVendorAssessment.mockResolvedValue({ id: "new", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel-1", version: 1, status: "SETUP_PENDING", review_kind: "TRIGGERED", source_trigger: "incident-42", form_template_id: "form-1", form_template_version: 2, review_due_at: new Date("2027-02-18T12:00").toISOString(), scope_kind: "FULL" });
  render(<VendorReview relationship={managed} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Start reassessment" }));
  fireEvent.change(screen.getByLabelText("Review type"), { target: { value: "TRIGGERED" } });
  fireEvent.change(screen.getByLabelText("Review reference"), { target: { value: "incident-42" } });
  fireEvent.change(await screen.findByLabelText("Assessment form"), { target: { value: "form-1" } });
  fireEvent.change(screen.getByLabelText("Review due"), { target: { value: "2027-02-18T12:00" } });
  fireEvent.click(screen.getByRole("button", { name: "Start assessment" }));
  await vi.waitFor(() => expect(assessment.startVendorAssessment).toHaveBeenCalledWith("rel-1", expect.objectContaining({ relationship_version: 3, review_kind: "TRIGGERED", source_trigger: "incident-42", form_template_id: "form-1", form_template_version: 2 })));
});

it("starts a focused reassessment using fields from the selected form revision", async () => {
  const managed = { ...relationship, relationship: { ...relationship.relationship, status: "ACTIVE" } };
  const completed = { id: "old", relationship_id: "rel-1", version: 7, status: "COMPLETED", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: completed });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: completed, answers: [], documents: [], matters: [], coverage: {} });
  forms.loadFormTemplateRevision.mockResolvedValue({ id: "form-1", tenant_id: "bank", version: 2, status: "ACTIVE", is_current: true, fields: [{ id: "q1", label: "Access review" }, { id: "q2", label: "Audit report" }] });
  assessment.startVendorAssessment.mockResolvedValue({ id: "new", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel-1", version: 1, status: "SETUP_PENDING", review_kind: "PERIODIC", source_trigger: "annual-2027", form_template_id: "form-1", form_template_version: 2, review_due_at: new Date("2027-02-18T12:00").toISOString(), scope_kind: "FOCUSED", selected_field_ids: ["q2"] });
  render(<VendorReview relationship={managed} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Start reassessment" }));
  fireEvent.change(screen.getByLabelText("Review reference"), { target: { value: "annual-2027" } });
  fireEvent.change(await screen.findByLabelText("Assessment form"), { target: { value: "form-1" } });
  fireEvent.change(screen.getByLabelText("Assessment scope"), { target: { value: "FOCUSED" } });
  fireEvent.click(await screen.findByLabelText("Audit report"));
  fireEvent.change(screen.getByLabelText("Review due"), { target: { value: "2027-02-18T12:00" } });
  fireEvent.click(screen.getByRole("button", { name: "Start assessment" }));
  await vi.waitFor(() => expect(assessment.startVendorAssessment).toHaveBeenCalledWith("rel-1", expect.objectContaining({ scope_kind: "FOCUSED", selected_field_ids: ["q2"], form_template_id: "form-1", form_template_version: 2 })));
});

it("clears loaded form choices when access is revoked during pagination", async () => {
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: null });
  forms.loadFormTemplatePage.mockResolvedValueOnce({ items: [{ template: { id: "form-1", name: "Security", code: "F-01" }, active_version: 2, active_status: "ACTIVE" }], next_cursor: "next" }).mockRejectedValueOnce(new ApiError(403, "Forbidden"));
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Start onboarding assessment" }));
  fireEvent.change(await screen.findByLabelText("Assessment form"), { target: { value: "form-1" } });
  fireEvent.click(screen.getByRole("button", { name: "Load more forms" }));
  expect(await screen.findByText("Assessment forms unavailable.")).toBeTruthy();
  expect(screen.queryByRole("option", { name: "Security · v2" })).toBeNull();
  expect(screen.queryByRole("button", { name: "Start assessment" })).toBeNull();
});

it("stops repeated form pagination cursors", async () => {
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: null });
  forms.loadFormTemplatePage.mockResolvedValueOnce({ items: [], next_cursor: "next" }).mockResolvedValueOnce({ items: [], next_cursor: "next" });
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Start onboarding assessment" }));
  fireEvent.click(await screen.findByRole("button", { name: "Load more forms" }));
  expect(await screen.findByText("More forms unavailable.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Load more forms" })).toBeNull();
});

it("deduplicates form choices within a later page", async () => {
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: null });
  const item = { template: { id: "form-2", name: "Continuity", code: "F-02" }, active_version: 3, active_status: "ACTIVE" };
  forms.loadFormTemplatePage.mockResolvedValueOnce({ items: [], next_cursor: "next" }).mockResolvedValueOnce({ items: [item, item] });
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Start onboarding assessment" }));
  fireEvent.click(await screen.findByRole("button", { name: "Load more forms" }));
  expect(await screen.findAllByRole("option", { name: "Continuity · v3" })).toHaveLength(1);
});

it.each([{ version: 1, status: "ACTIVE", isCurrent: true }, { version: 2, status: "RETIRED", isCurrent: true }, { version: 2, status: "ACTIVE", isCurrent: false }])("fails closed for a mismatched, retired or superseded form revision ($version, $status, $isCurrent)", async ({ version, status, isCurrent }) => {
  const managed = { ...relationship, relationship: { ...relationship.relationship, status: "ACTIVE" } };
  const completed = { id: "old", relationship_id: "rel-1", version: 7, status: "COMPLETED", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: completed });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: completed, answers: [], documents: [], matters: [], coverage: {} });
  forms.loadFormTemplateRevision.mockResolvedValue({ id: "form-1", tenant_id: "bank", version, status, is_current: isCurrent, fields: [{ id: "q1", label: "Access review" }] });
  render(<VendorReview relationship={managed} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Start reassessment" }));
  fireEvent.change(await screen.findByLabelText("Assessment form"), { target: { value: "form-1" } });
  fireEvent.change(screen.getByLabelText("Assessment scope"), { target: { value: "FOCUSED" } });
  expect(await screen.findByText(/Questions unavailable/)).toBeTruthy();
  expect(screen.queryByLabelText("Access review")).toBeNull();
  expect(screen.getByRole("button", { name: "Start assessment" }).matches(":disabled")).toBe(true);
  expect(assessment.startVendorAssessment).not.toHaveBeenCalled();
});

it("does not confirm a focused assessment with different selected questions", async () => {
  const managed = { ...relationship, relationship: { ...relationship.relationship, status: "ACTIVE" } };
  const completed = { id: "old", relationship_id: "rel-1", version: 7, status: "COMPLETED", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: completed });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: completed, answers: [], documents: [], matters: [], coverage: {} });
  forms.loadFormTemplateRevision.mockResolvedValue({ id: "form-1", tenant_id: "bank", version: 2, status: "ACTIVE", is_current: true, fields: [{ id: "q1", label: "Access review" }, { id: "q2", label: "Audit report" }] });
  assessment.startVendorAssessment.mockResolvedValue({ id: "new", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel-1", version: 1, status: "SETUP_PENDING", review_kind: "PERIODIC", source_trigger: "annual-2027", form_template_id: "form-1", form_template_version: 2, review_due_at: new Date("2027-02-18T12:00").toISOString(), scope_kind: "FOCUSED", selected_field_ids: ["q1"] });
  render(<VendorReview relationship={managed} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Start reassessment" }));
  fireEvent.change(screen.getByLabelText("Review reference"), { target: { value: "annual-2027" } });
  fireEvent.change(await screen.findByLabelText("Assessment form"), { target: { value: "form-1" } });
  fireEvent.change(screen.getByLabelText("Assessment scope"), { target: { value: "FOCUSED" } });
  fireEvent.click(await screen.findByLabelText("Audit report"));
  fireEvent.change(screen.getByLabelText("Review due"), { target: { value: "2027-02-18T12:00" } });
  fireEvent.click(screen.getByRole("button", { name: "Start assessment" }));
  expect(await screen.findByText(/Review status could not be confirmed/)).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Start reassessment" })).toBeNull();
});

it("does not confirm a reassessment for another relationship", async () => {
  const managed = { ...relationship, relationship: { ...relationship.relationship, status: "ACTIVE" } };
  const completed = { id: "old", relationship_id: "rel-1", version: 7, status: "COMPLETED", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: completed });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: completed, answers: [], documents: [], matters: [], coverage: {} });
  assessment.startVendorAssessment.mockResolvedValue({ id: "new", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "other", version: 1, status: "SETUP_PENDING", review_kind: "PERIODIC", source_trigger: "annual-2027", form_template_id: "form-1", form_template_version: 2, review_due_at: new Date("2027-02-18T12:00").toISOString(), scope_kind: "FULL" });
  render(<VendorReview relationship={managed} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Start reassessment" }));
  fireEvent.change(screen.getByLabelText("Review reference"), { target: { value: "annual-2027" } });
  fireEvent.change(await screen.findByLabelText("Assessment form"), { target: { value: "form-1" } });
  fireEvent.change(screen.getByLabelText("Review due"), { target: { value: "2027-02-18T12:00" } });
  fireEvent.click(screen.getByRole("button", { name: "Start assessment" }));
  expect(await screen.findByText(/Review status could not be confirmed/)).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Start reassessment" })).toBeNull();
});

it("does not start reassessment when authority is unavailable", async () => {
  const managed = { ...relationship, relationship: { ...relationship.relationship, status: "ACTIVE" } };
  const completed = { id: "old", relationship_id: "rel-1", version: 7, status: "COMPLETED", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: completed });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: completed, answers: [], documents: [], matters: [], coverage: {} });
  authority.checkVendorAuthority.mockResolvedValue(false);
  render(<VendorReview relationship={managed} onBack={() => {}}/>);
  expect(await screen.findByText(/Assessment action unavailable/)).toBeTruthy();
  expect(authority.checkVendorAuthority).toHaveBeenCalledWith("start", "rel-1");
  expect(screen.queryByRole("button", { name: "Start reassessment" })).toBeNull();
  expect(assessment.startVendorAssessment).not.toHaveBeenCalled();
});

it("opens only the selected assessment and starts review at its recorded version", async () => {
  const current = { id: "assess-1", relationship_id: "rel-1", version: 7, status: "SUBMITTED", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: current, answers: [{ field_id: "q1", label: "Access reviewed?", visibility: "VISIBLE", value: { text: "Yes" } }], documents: [], matters: [], coverage: {} });
  assessment.startVendorAssessmentReview.mockResolvedValue({ ...current, version: 8, status: "UNDER_REVIEW" });
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  expect(await screen.findByText("Access reviewed?")).toBeTruthy();
  fireEvent.click(await screen.findByRole("button", { name: "Start review" }));
  await vi.waitFor(() => expect(assessment.startVendorAssessmentReview).toHaveBeenCalledWith("assess-1", { expected_version: 7 }));
});

it("shows assessment findings without offering an unavailable creation action", async () => {
  const current = { id: "assess-1", relationship_id: "rel-1", review_matter_id: "review-1", version: 8, status: "UNDER_REVIEW", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: current, answers: [], documents: [], matters: [{ matter_id: "f1", title: "Update the access report", type: "VENDOR_DEFICIENCY", status: "OPEN" }], coverage: {} });
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  expect(await screen.findByText("Update the access report")).toBeTruthy();
  expect(screen.getByText("Finding creation unavailable.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Add finding" })).toBeNull();
});

it("keeps an uncertain assessment submission from being repeated", async () => {
  const current = { id: "assess-1", relationship_id: "rel-1", version: 7, status: "SUBMITTED", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: current, answers: [], documents: [], matters: [], coverage: {} });
  assessment.startVendorAssessmentReview.mockRejectedValue(new Error("connection lost"));
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Start review" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Review status could not be confirmed");
  expect(screen.queryByRole("button", { name: "Start review" })).toBeNull();
});

it("does not show an assessment linked to a different vendor relationship", async () => {
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: null });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: { id: "assess-2", relationship_id: "different", version: 1, status: "SUBMITTED" }, answers: [{ field_id: "q1", label: "Private answer", visibility: "VISIBLE" }], documents: [] });
  render(<VendorReview relationship={relationship} assessmentId="assess-2" onBack={() => {}}/>);
  expect(await screen.findByText("Assessment unavailable.")).toBeTruthy();
  expect(screen.queryByText("Private answer")).toBeNull();
});

it("makes a historical assessment read-only", async () => {
  const historical = { id: "older", relationship_id: "rel-1", version: 4, status: "SUBMITTED", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: { ...historical, id: "current" } });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: historical, answers: [], documents: [], matters: [], coverage: {} });
  render(<VendorReview relationship={relationship} assessmentId="older" onBack={() => {}}/>);
  expect(await screen.findByText("Earlier assessment")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Start review" })).toBeNull();
});

it("sends the pinned request to the specified recipient and shows the delivery outcome", async () => {
  const current = { id: "assess-1", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel-1", form_template_id: "form-1", form_template_version: 2, version: 7, status: "READY_TO_SEND", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.sendVendorAssessmentRequest.mockResolvedValue({ assessment: { ...current, version: 9, status: "COLLECTING", current_request_id: "request-1" }, request: { id: "request-1", tenant_id: "bank", legal_entity_id: "entity", subject_type: "VENDOR_RELATIONSHIP", subject_id: "rel-1", form_template_id: "form-1", form_template_version: 2, origin: { type: "THIRD_PARTY_ASSESSMENT", id: "assess-1", version: 1 }, deadline: new Date("2027-02-17T12:00").toISOString() }, state: "LINK_CREATED_EMAIL_NOT_SENT", capture_url: "https://secret.test/token" });
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Send assessment request" }));
  fireEvent.change(screen.getByLabelText("Recipient email"), { target: { value: "vendor@example.com" } });
  fireEvent.change(screen.getByLabelText("Response deadline"), { target: { value: "2027-02-17T12:00" } });
  fireEvent.click(screen.getByRole("button", { name: "Send request" }));
  expect(await screen.findByText("Request created; email delivery not confirmed.")).toBeTruthy();
  expect(assessment.sendVendorAssessmentRequest).toHaveBeenCalledWith("assess-1", expect.objectContaining({ expected_version: 7, audience: "vendor@example.com" }));
  expect(document.body.textContent).not.toContain("secret.test");
});

it("withholds review when the authority route cannot be verified", async () => {
  const current = { id: "assess-1", relationship_id: "rel-1", version: 7, status: "SUBMITTED", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: current, answers: [], documents: [], matters: [], coverage: {} });
  authority.checkVendorAuthority.mockRejectedValue(new Error("unavailable"));
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  expect(await screen.findByText("Assessment action unavailable. Check your role or reload the assessment.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Start review" })).toBeNull();
  expect(assessment.startVendorAssessmentReview).not.toHaveBeenCalled();
});

it("submits the review conclusion using the current assessment version", async () => {
  const current = { id: "assess-1", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "rel-1", version: 8, status: "UNDER_REVIEW", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: current, answers: [], documents: [], matters: [], coverage: {} });
  assessment.completeVendorAssessment.mockResolvedValue({ ...current, version: 9, status: "COMPLETED", conclusion: "INCONCLUSIVE", conclusion_rationale: "The scope needs clarification.", reviewer_principal_id: "reviewer", completed_at: "2026-11-22T10:00:00Z" });
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Complete review" }));
  fireEvent.change(screen.getByLabelText("Conclusion"), { target: { value: "INCONCLUSIVE" } });
  fireEvent.change(screen.getByLabelText("Rationale"), { target: { value: "The scope needs clarification." } });
  fireEvent.click(screen.getByRole("button", { name: "Complete review" }));
  await vi.waitFor(() => expect(assessment.completeVendorAssessment).toHaveBeenCalledWith("assess-1", { expected_version: 8, conclusion: "INCONCLUSIVE", rationale: "The scope needs clarification." }));
  await vi.waitFor(() => expect(assessment.loadCurrentVendorAssessment).toHaveBeenCalledTimes(2));
});

it("opens an earlier exact assessment when there is no current assessment", async () => {
  const { ApiError } = await import("../http");
  const historical = { id: "older", relationship_id: "rel-1", version: 4, status: "COMPLETED", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockRejectedValue(new ApiError(404, "Not found", "vendor_assessment_not_found"));
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: historical, answers: [{ field_id: "d1", label: "Contract", visibility: "VISIBLE", value: { artifact_ids: ["file1"] } }], documents: [], matters: [], coverage: {} });
  render(<VendorReview relationship={relationship} assessmentId="older" onBack={() => {}}/>);
  expect(await screen.findByText("Earlier assessment")).toBeTruthy();
  expect(screen.getByText("Document submitted")).toBeTruthy();
});

it("shows a definite completion rejection without reporting an uncertain result", async () => {
  const { ApiError } = await import("../http");
  const current = { id: "assess-1", relationship_id: "rel-1", version: 8, status: "UNDER_REVIEW", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: current, answers: [], documents: [], matters: [], coverage: {} });
  assessment.completeVendorAssessment.mockRejectedValue(new ApiError(409, "Review the required documents.", "vendor_assessment_review_incomplete"));
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Complete review" }));
  fireEvent.change(screen.getByLabelText("Conclusion"), { target: { value: "INCONCLUSIVE" } });
  fireEvent.change(screen.getByLabelText("Rationale"), { target: { value: "Documents remain incomplete." } });
  fireEvent.click(screen.getByRole("button", { name: "Complete review" }));
  expect(await screen.findByText("Review the required documents.")).toBeTruthy();
  expect(screen.queryByText(/status could not be confirmed/)).toBeNull();
});

it("finds active assessment forms beyond the first page", async () => {
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: null });
  forms.loadFormTemplatePage.mockResolvedValueOnce({ items: [], next_cursor: "next" }).mockResolvedValueOnce({ items: [{ template: { id: "form-2", name: "Continuity", code: "F-02" }, active_version: 3, active_status: "ACTIVE" }] });
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Start onboarding assessment" }));
  fireEvent.click(await screen.findByRole("button", { name: "Load more forms" }));
  expect(await screen.findByRole("option", { name: "Continuity · v3" })).toBeTruthy();
  expect(forms.loadFormTemplatePage).toHaveBeenLastCalledWith(expect.objectContaining({ cursor: "next", limit: 50 }));
});

it("requires a reload after an assessment version conflict", async () => {
  const { ApiError } = await import("../http");
  const current = { id: "assess-1", relationship_id: "rel-1", version: 7, status: "SUBMITTED", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: current, answers: [], documents: [], matters: [], coverage: {} });
  assessment.startVendorAssessmentReview.mockRejectedValue(new ApiError(409, "Assessment changed. Reload it.", "vendor_assessment_changed"));
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Start review" }));
  expect(await screen.findByText("Assessment changed. Reload it.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Start review" })).toBeNull();
  expect(screen.getByRole("button", { name: "Reload assessment" })).toBeTruthy();
});

it("identifies the linked bank document and the reconciliation receipt", async () => {
  const current = { id: "a", relationship_id: "rel-1", version: 7, status: "SUBMITTED", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: current, answers: [{ field_id: "q", label: "Assurance report", visibility: "VISIBLE", collection_resolution: { id: "receipt-1", version: 2, source: { file_name: "Bank report.pdf" }, reconciled_by: "risk-officer", reconciled_at: "2026-11-20T12:00:00Z", rationale: "Current assurance" } }], documents: [], matters: [], coverage: {} });
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  expect(await screen.findByText(/Bank report.pdf/)).toBeTruthy();
  expect(screen.getByText(/risk-officer/)).toBeTruthy();
  expect(screen.getByText(/receipt-1/)).toBeTruthy();
  expect(screen.getByText(/Current assurance/)).toBeTruthy();
  expect(screen.queryByText("No answer recorded")).toBeNull();
});

it("retries only failed setup with owner authority and the current assessment version", async () => {
  const current = { id: "assess-1", relationship_id: "rel-1", version: 7, status: "SETUP_PENDING", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current, setup: { assessment_id: "assess-1", state: "FAILED" } });
  assessment.retryVendorAssessmentSetup.mockResolvedValue({ assessment: { ...current, version: 8 }, setup: { assessment_id: "assess-1", state: "READY" } });
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Retry assessment setup" }));
  expect(await screen.findByText("Assessment setup queued.")).toBeTruthy();
  expect(assessment.retryVendorAssessmentSetup).toHaveBeenCalledWith("assess-1", { expected_version: 7 });
  expect(authority.checkVendorAuthority).toHaveBeenCalledWith("setup", "assess-1");
});

it("does not retry setup when the authority route is unavailable", async () => {
  const current = { id: "assess-1", relationship_id: "rel-1", version: 7, status: "SETUP_PENDING", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current, setup: { assessment_id: "assess-1", state: "FAILED" } });
  authority.checkVendorAuthority.mockResolvedValue(false);
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  expect(await screen.findByText(/Assessment action unavailable/)).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Retry assessment setup" })).toBeNull();
  expect(assessment.retryVendorAssessmentSetup).not.toHaveBeenCalled();
});

it("reissues an invitation for the existing recipient without exposing its token", async () => {
  const current = { id: "assess-1", relationship_id: "rel-1", current_request_id: "request-1", version: 7, status: "COLLECTING", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.reissueVendorAssessmentRequest.mockResolvedValue({ assessment: { ...current, version: 9 }, request: { id: "request-1" }, state: "DELIVERED", delivery: { status: "DELIVERED", delivered_at: "2026-11-22T10:00:00Z" }, capture_url: "https://example.test/#form_access=secret" });
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Replace invitation" }));
  fireEvent.change(screen.getByLabelText("Recipient email"), { target: { value: "vendor@example.com" } });
  fireEvent.click(screen.getByRole("button", { name: "Send replacement" }));
  expect(await screen.findByText("Replacement sent.")).toBeTruthy();
  expect(assessment.reissueVendorAssessmentRequest).toHaveBeenCalledWith("assess-1", { expected_version: 7, audience: "vendor@example.com", invitation_ttl_minutes: 10080 });
  expect(authority.checkVendorAuthority).toHaveBeenCalledWith("reissue", "assess-1");
  expect(document.body.textContent).not.toContain("secret");
});

it("requires reload after any unconfirmed replacement outcome", async () => {
  const current = { id: "assess-1", relationship_id: "rel-1", current_request_id: "request-1", version: 7, status: "COLLECTING", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.reissueVendorAssessmentRequest.mockRejectedValue(new Error("connection lost"));
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Replace invitation" }));
  fireEvent.change(screen.getByLabelText("Recipient email"), { target: { value: "vendor@example.com" } });
  fireEvent.click(screen.getByRole("button", { name: "Send replacement" }));
  expect(await screen.findByText(/status could not be confirmed/i)).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Send replacement" })).toBeNull();
});

it("does not offer retry for setup belonging to another assessment", async () => {
  const current = { id: "assess-1", relationship_id: "rel-1", version: 7, status: "SETUP_PENDING", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current, setup: { assessment_id: "other", state: "FAILED" } });
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  expect(await screen.findByText(/Setup pending/)).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Retry assessment setup" })).toBeNull();
});

it("does not claim replacement succeeded for a mismatched request", async () => {
  const current = { id: "assess-1", relationship_id: "rel-1", current_request_id: "request-1", version: 7, status: "COLLECTING", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.reissueVendorAssessmentRequest.mockResolvedValue({ assessment: { ...current, version: 9 }, request: { id: "different" }, state: "DELIVERED" });
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Replace invitation" }));
  fireEvent.change(screen.getByLabelText("Recipient email"), { target: { value: "vendor@example.com" } });
  fireEvent.click(screen.getByRole("button", { name: "Send replacement" }));
  expect(await screen.findByText(/status could not be confirmed/i)).toBeTruthy();
  expect(screen.queryByText("Replacement sent.")).toBeNull();
});

it("requests clarification for selected visible answers at the current version", async () => {
  const current = { id: "assess-1", relationship_id: "rel-1", current_request_id: "request-1", submission_id: "submission-1", review_due_at: "2099-03-01T18:00:00Z", version: 7, status: "UNDER_REVIEW", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: current, answers: [{ field_id: "q1", label: "Is access reviewed?", visibility: "VISIBLE" }, { field_id: "q2", label: "Hidden question", visibility: "CONDITIONALLY_OMITTED" }], documents: [], matters: [], coverage: {} });
  assessment.requestVendorAssessmentClarification.mockResolvedValue({ assessment: { ...current, version: 9, status: "COLLECTING", current_request_id: "request-2" }, state: "DELIVERED", delivery: { status: "DELIVERED", delivered_at: "2026-11-22T10:00:00Z" }, capture_url: "https://example.test/#form_access=secret" });
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Request clarification" }));
  fireEvent.click(screen.getByLabelText("Is access reviewed?"));
  expect(screen.queryByLabelText("Hidden question")).toBeNull();
  fireEvent.change(screen.getByLabelText("Recipient email"), { target: { value: "vendor@example.com" } });
  fireEvent.change(screen.getByLabelText("Request details"), { target: { value: "Please confirm the review date." } });
  fireEvent.change(screen.getByLabelText("Response deadline"), { target: { value: "2099-02-20T12:00" } });
  fireEvent.click(screen.getByRole("button", { name: "Send clarification" }));
  expect(await screen.findByText("Clarification sent.")).toBeTruthy();
  expect(assessment.requestVendorAssessmentClarification).toHaveBeenCalledWith("assess-1", expect.objectContaining({ expected_version: 7, request_fields: ["q1"], audience: "vendor@example.com", message: "Please confirm the review date.", invitation_ttl_minutes: 10080 }));
  expect(authority.checkVendorAuthority).toHaveBeenCalledWith("clarification", "assess-1");
  expect(document.body.textContent).not.toContain("secret");
});

it("withholds clarification without its reviewer authority", async () => {
  const current = { id: "assess-1", relationship_id: "rel-1", current_request_id: "request-1", submission_id: "submission-1", version: 7, status: "UNDER_REVIEW", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: current, answers: [{ field_id: "q1", label: "Access?", visibility: "VISIBLE" }], documents: [], matters: [], coverage: {} });
  authority.checkVendorAuthority.mockImplementation(async (action) => action !== "clarification");
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  expect(await screen.findByText("Access?")).toBeTruthy();
  await vi.waitFor(() => expect(authority.checkVendorAuthority).toHaveBeenCalledWith("clarification", "assess-1"));
  expect(screen.queryByRole("button", { name: "Request clarification" })).toBeNull();
});

it("does not retry a clarification when delivery cannot be confirmed", async () => {
  const { ApiError } = await import("../http");
  const current = { id: "assess-1", relationship_id: "rel-1", current_request_id: "request-1", submission_id: "submission-1", review_due_at: "2099-03-01T18:00:00Z", version: 7, status: "UNDER_REVIEW", review_kind: "ONBOARDING" };
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: current, answers: [{ field_id: "q1", label: "Access?", visibility: "VISIBLE" }], documents: [], matters: [], coverage: {} });
  assessment.requestVendorAssessmentClarification.mockRejectedValue(new ApiError(409, "Changed", "version_conflict"));
  render(<VendorReview relationship={relationship} onBack={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Request clarification" }));
  fireEvent.click(screen.getByLabelText("Access?"));
  fireEvent.change(screen.getByLabelText("Recipient email"), { target: { value: "vendor@example.com" } });
  fireEvent.change(screen.getByLabelText("Request details"), { target: { value: "Confirm the current review date." } });
  fireEvent.change(screen.getByLabelText("Response deadline"), { target: { value: "2099-02-20T12:00" } });
  fireEvent.click(screen.getByRole("button", { name: "Send clarification" }));
  expect(await screen.findByText(/status could not be confirmed/i)).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Send clarification" })).toBeNull();
  expect(assessment.requestVendorAssessmentClarification).toHaveBeenCalledTimes(1);
});

it('does not confirm a completion receipt for a different assessment or conclusion',async()=>{const current={id:'assess-1',tenant_id:'bank',legal_entity_id:'entity',relationship_id:'rel-1',version:8,status:'UNDER_REVIEW',review_kind:'ONBOARDING'};assessment.loadCurrentVendorAssessment.mockResolvedValue({assessment:current});assessment.loadVendorAssessment.mockResolvedValue({assessment:current,answers:[],documents:[],matters:[],coverage:{}});assessment.completeVendorAssessment.mockResolvedValue({...current,id:'other',version:9,status:'COMPLETED',conclusion:'SATISFACTORY',conclusion_rationale:'The scope needs clarification.'});render(<VendorReview relationship={relationship} onBack={()=>{}}/>);fireEvent.click(await screen.findByRole('button',{name:'Complete review'}));fireEvent.change(screen.getByLabelText('Conclusion'),{target:{value:'INCONCLUSIVE'}});fireEvent.change(screen.getByLabelText('Rationale'),{target:{value:'The scope needs clarification.'}});fireEvent.click(screen.getByRole('button',{name:'Complete review'}));expect(await screen.findByText(/Review status could not be confirmed/)).toBeTruthy();expect(screen.queryByLabelText('Conclusion')).toBeNull();});
it('submits only once while completion is pending',async()=>{const current={id:'assess-1',tenant_id:'bank',legal_entity_id:'entity',relationship_id:'rel-1',version:8,status:'UNDER_REVIEW',review_kind:'ONBOARDING'};assessment.loadCurrentVendorAssessment.mockResolvedValue({assessment:current});assessment.loadVendorAssessment.mockResolvedValue({assessment:current,answers:[],documents:[],matters:[],coverage:{}});assessment.completeVendorAssessment.mockReturnValue(new Promise(()=>{}));render(<VendorReview relationship={relationship} onBack={()=>{}}/>);fireEvent.click(await screen.findByRole('button',{name:'Complete review'}));fireEvent.change(screen.getByLabelText('Conclusion'),{target:{value:'INCONCLUSIVE'}});fireEvent.change(screen.getByLabelText('Rationale'),{target:{value:'The scope needs clarification.'}});const save=screen.getByRole('button',{name:'Complete review'});fireEvent.submit(save.closest('form')!);fireEvent.submit(save.closest('form')!);expect(assessment.completeVendorAssessment).toHaveBeenCalledTimes(1);});

it.each([{tenant_id:'other'},{legal_entity_id:'other'},{relationship_id:'other'},{version:10},{status:'UNDER_REVIEW'},{conclusion:'SATISFACTORY'},{conclusion_rationale:'Different'},{conclusion_uncertainty:'Different'},{reviewer_principal_id:''},{completed_at:''}])('requires a matching scoped completion receipt: %o',async(change)=>{const current={id:'assess-1',tenant_id:'bank',legal_entity_id:'entity',relationship_id:'rel-1',version:8,status:'UNDER_REVIEW',review_kind:'ONBOARDING'};assessment.loadCurrentVendorAssessment.mockResolvedValue({assessment:current});assessment.loadVendorAssessment.mockResolvedValue({assessment:current,answers:[],documents:[],matters:[],coverage:{}});assessment.completeVendorAssessment.mockResolvedValue({...current,version:9,status:'COMPLETED',conclusion:'INCONCLUSIVE',conclusion_rationale:'Reviewed',reviewer_principal_id:'reviewer',completed_at:'2026-11-22T10:00:00Z',...change});render(<VendorReview relationship={relationship} onBack={()=>{}}/>);fireEvent.click(await screen.findByRole('button',{name:'Complete review'}));fireEvent.change(screen.getByLabelText('Conclusion'),{target:{value:'INCONCLUSIVE'}});fireEvent.change(screen.getByLabelText('Rationale'),{target:{value:'Reviewed'}});fireEvent.click(screen.getByRole('button',{name:'Complete review'}));expect(await screen.findByText(/Review status could not be confirmed/)).toBeTruthy();});

it('ignores a late completion receipt after changing vendor scope',async()=>{const current={id:'assess-1',tenant_id:'bank',legal_entity_id:'entity',relationship_id:'rel-1',version:8,status:'UNDER_REVIEW',review_kind:'ONBOARDING'};assessment.loadCurrentVendorAssessment.mockResolvedValue({assessment:current});assessment.loadVendorAssessment.mockResolvedValue({assessment:current,answers:[],documents:[],matters:[],coverage:{}});let resolve!:(value:unknown)=>void;assessment.completeVendorAssessment.mockReturnValue(new Promise(done=>{resolve=done}));const view=render(<VendorReview relationship={relationship} onBack={()=>{}}/>);fireEvent.click(await screen.findByRole('button',{name:'Complete review'}));fireEvent.change(screen.getByLabelText('Conclusion'),{target:{value:'INCONCLUSIVE'}});fireEvent.change(screen.getByLabelText('Rationale'),{target:{value:'Reviewed'}});fireEvent.click(screen.getByRole('button',{name:'Complete review'}));assessment.loadCurrentVendorAssessment.mockResolvedValue({assessment:null});view.rerender(<VendorReview relationship={{...relationship,relationship:{...relationship.relationship,id:'rel-2'}}} onBack={()=>{}}/>);await screen.findByText('No assessment recorded for this relationship.');const reads=assessment.loadCurrentVendorAssessment.mock.calls.length;await act(async()=>{resolve({...current,version:9,status:'COMPLETED',conclusion:'INCONCLUSIVE',conclusion_rationale:'Reviewed',reviewer_principal_id:'reviewer',completed_at:'2026-11-22T10:00:00Z'});});expect(assessment.loadCurrentVendorAssessment).toHaveBeenCalledTimes(reads);expect(screen.queryByText(/Review status could not be confirmed/)).toBeNull();});

it.each([401,403,404])('clears the completion form and assessment after %s access loss',async(status)=>{const current={id:'assess-1',tenant_id:'bank',legal_entity_id:'entity',relationship_id:'rel-1',version:8,status:'UNDER_REVIEW',review_kind:'ONBOARDING'};assessment.loadCurrentVendorAssessment.mockResolvedValue({assessment:current});assessment.loadVendorAssessment.mockResolvedValue({assessment:current,answers:[{field_id:'f',label:'Confidential answer',visibility:'VISIBLE',value:{text:'Private answer'}}],documents:[],matters:[],coverage:{}});assessment.completeVendorAssessment.mockRejectedValue(new ApiError(status,'Denied'));render(<VendorReview relationship={relationship} onBack={()=>{}}/>);fireEvent.click(await screen.findByRole('button',{name:'Complete review'}));fireEvent.change(screen.getByLabelText('Conclusion'),{target:{value:'INCONCLUSIVE'}});fireEvent.change(screen.getByLabelText('Rationale'),{target:{value:'Reviewed'}});fireEvent.click(screen.getByRole('button',{name:'Complete review'}));expect(await screen.findByText('Assessment unavailable.')).toBeTruthy();expect(screen.queryByLabelText('Conclusion')).toBeNull();expect(screen.queryByText('Private answer')).toBeNull();fireEvent.click(screen.getByRole('button',{name:'Try again'}));await vi.waitFor(()=>expect(assessment.loadCurrentVendorAssessment).toHaveBeenCalledTimes(2));});

it('does not report a sent assessment when the returned request belongs to another vendor',async()=>{const current={id:'assess-1',tenant_id:'bank',legal_entity_id:'entity',relationship_id:'rel-1',version:7,status:'READY_TO_SEND',review_kind:'ONBOARDING',form_template_id:'form-1',form_template_version:2};assessment.loadCurrentVendorAssessment.mockResolvedValue({assessment:current});assessment.sendVendorAssessmentRequest.mockResolvedValue({assessment:{...current,version:9,status:'COLLECTING',current_request_id:'request-1'},request:{id:'request-1',tenant_id:'bank',legal_entity_id:'entity',subject_type:'VENDOR_RELATIONSHIP',subject_id:'other',form_template_id:'form-1',form_template_version:2,origin:{type:'THIRD_PARTY_ASSESSMENT',id:'assess-1',version:1},deadline:new Date('2027-02-17T12:00').toISOString()},state:'DELIVERED',delivery:{status:'DELIVERED',delivered_at:'2026-11-22T10:00:00Z'}});render(<VendorReview relationship={relationship} onBack={()=>{}}/>);fireEvent.click(await screen.findByRole('button',{name:'Send assessment request'}));fireEvent.change(screen.getByLabelText('Recipient email'),{target:{value:'vendor@example.com'}});fireEvent.change(screen.getByLabelText('Response deadline'),{target:{value:'2027-02-17T12:00'}});fireEvent.click(screen.getByRole('button',{name:'Send request'}));expect(await screen.findByText(/Request status could not be confirmed/)).toBeTruthy();expect(screen.queryByText('Request sent.')).toBeNull();});

it.each([{state:'DELIVERED',updated:{status:'READY_TO_SEND'}},{state:'DELIVERED',request:{id:'other'}},{state:'DELIVERED',request:{form_template_version:3}},{state:'DELIVERED',request:{origin:{type:'THIRD_PARTY_ASSESSMENT',id:'other',version:1}}},{state:'DELIVERED',request:{deadline:'2027-02-18T12:00:00Z'}},{state:'UNKNOWN'},{state:'DELIVERY_UNKNOWN'}])('rejects inconsistent send receipt: %o',async(change)=>{const current={id:'assess-1',tenant_id:'bank',legal_entity_id:'entity',relationship_id:'rel-1',version:7,status:'READY_TO_SEND',review_kind:'ONBOARDING',form_template_id:'form-1',form_template_version:2};const req={id:'request-1',tenant_id:'bank',legal_entity_id:'entity',subject_type:'VENDOR_RELATIONSHIP',subject_id:'rel-1',form_template_id:'form-1',form_template_version:2,origin:{type:'THIRD_PARTY_ASSESSMENT',id:'assess-1',version:1},deadline:new Date('2027-02-17T12:00').toISOString()};assessment.loadCurrentVendorAssessment.mockResolvedValue({assessment:current});assessment.sendVendorAssessmentRequest.mockResolvedValue({assessment:{...current,version:9,status:'COLLECTING',current_request_id:'request-1',...change.updated},request:{...req,...change.request},state:change.state,delivery:{status:'DELIVERED',delivered_at:'2026-11-22T10:00:00Z'}});render(<VendorReview relationship={relationship} onBack={()=>{}}/>);fireEvent.click(await screen.findByRole('button',{name:'Send assessment request'}));fireEvent.change(screen.getByLabelText('Recipient email'),{target:{value:'vendor@example.com'}});fireEvent.change(screen.getByLabelText('Response deadline'),{target:{value:'2027-02-17T12:00'}});fireEvent.click(screen.getByRole('button',{name:'Send request'}));expect(await screen.findByText(/Request status could not be confirmed/)).toBeTruthy();expect(screen.queryByRole('button',{name:'Send request'})).toBeNull();});
it('recognizes a prepared request when invitation creation is unavailable',async()=>{const current={id:'assess-1',tenant_id:'bank',legal_entity_id:'entity',relationship_id:'rel-1',version:7,status:'READY_TO_SEND',review_kind:'ONBOARDING',form_template_id:'form-1',form_template_version:2};assessment.loadCurrentVendorAssessment.mockResolvedValue({assessment:current});assessment.sendVendorAssessmentRequest.mockResolvedValue({assessment:{...current,version:8,current_request_id:'request-1'},request:{id:'request-1',tenant_id:'bank',legal_entity_id:'entity',subject_type:'VENDOR_RELATIONSHIP',subject_id:'rel-1',form_template_id:'form-1',form_template_version:2,origin:{type:'THIRD_PARTY_ASSESSMENT',id:'assess-1',version:1},deadline:new Date('2027-02-17T12:00').toISOString()},state:'REQUEST_READY_INVITATION_NOT_ISSUED'});render(<VendorReview relationship={relationship} onBack={()=>{}}/>);fireEvent.click(await screen.findByRole('button',{name:'Send assessment request'}));fireEvent.change(screen.getByLabelText('Recipient email'),{target:{value:'vendor@example.com'}});fireEvent.change(screen.getByLabelText('Response deadline'),{target:{value:'2027-02-17T12:00'}});fireEvent.click(screen.getByRole('button',{name:'Send request'}));expect(await screen.findByText('Request created; access not issued.')).toBeTruthy();});
it.each([401,403,404])('clears the assessment when sending returns %s',async(status)=>{const current={id:'assess-1',tenant_id:'bank',legal_entity_id:'entity',relationship_id:'rel-1',version:7,status:'READY_TO_SEND',review_kind:'ONBOARDING',form_template_id:'form-1',form_template_version:2};assessment.loadCurrentVendorAssessment.mockResolvedValue({assessment:current});assessment.sendVendorAssessmentRequest.mockRejectedValue(new ApiError(status,'Denied'));render(<VendorReview relationship={relationship} onBack={()=>{}}/>);fireEvent.click(await screen.findByRole('button',{name:'Send assessment request'}));fireEvent.change(screen.getByLabelText('Recipient email'),{target:{value:'vendor@example.com'}});fireEvent.change(screen.getByLabelText('Response deadline'),{target:{value:'2027-02-17T12:00'}});fireEvent.click(screen.getByRole('button',{name:'Send request'}));expect(await screen.findByText('Assessment unavailable.')).toBeTruthy();expect(screen.queryByLabelText('Recipient email')).toBeNull();});

it.each([409,422])('requires assessment reload after an unconfirmed send error %s',async(status)=>{const current={id:'assess-1',tenant_id:'bank',legal_entity_id:'entity',relationship_id:'rel-1',version:7,status:'READY_TO_SEND',review_kind:'ONBOARDING',form_template_id:'form-1',form_template_version:2};assessment.loadCurrentVendorAssessment.mockResolvedValue({assessment:current});assessment.sendVendorAssessmentRequest.mockRejectedValue(new ApiError(status,'Request unavailable'));render(<VendorReview relationship={relationship} onBack={()=>{}}/>);fireEvent.click(await screen.findByRole('button',{name:'Send assessment request'}));fireEvent.change(screen.getByLabelText('Recipient email'),{target:{value:'vendor@example.com'}});fireEvent.change(screen.getByLabelText('Response deadline'),{target:{value:'2027-02-17T12:00'}});fireEvent.click(screen.getByRole('button',{name:'Send request'}));expect(await screen.findByText(/Request status could not be confirmed/)).toBeTruthy();expect(screen.queryByRole('button',{name:'Send request'})).toBeNull();});

it('does not claim a clarification was sent when delivery is unknown',async()=>{const current={id:'assess-1',relationship_id:'rel-1',current_request_id:'request-1',submission_id:'submission-1',review_due_at:'2099-03-01T18:00:00Z',version:7,status:'UNDER_REVIEW',review_kind:'ONBOARDING'};assessment.loadCurrentVendorAssessment.mockResolvedValue({assessment:current});assessment.loadVendorAssessment.mockResolvedValue({assessment:current,answers:[{field_id:'q1',label:'Access?',visibility:'VISIBLE'}],documents:[],matters:[],coverage:{}});collections.loadVendorCollection.mockResolvedValue({assessment_id:'assess-1',assessment_version:7,fields:[],can_reconcile:false});assessment.requestVendorAssessmentClarification.mockResolvedValue({assessment:{...current,version:8},state:'DELIVERY_UNKNOWN'});render(<VendorReview relationship={relationship} onBack={()=>{}}/>);fireEvent.click(await screen.findByRole('button',{name:'Request clarification'}));fireEvent.click(screen.getByLabelText('Access?'));fireEvent.change(screen.getByLabelText('Recipient email'),{target:{value:'vendor@example.com'}});fireEvent.change(screen.getByLabelText('Request details'),{target:{value:'Confirm the review date.'}});fireEvent.change(screen.getByLabelText('Response deadline'),{target:{value:'2099-02-20T12:00'}});fireEvent.click(screen.getByRole('button',{name:'Send clarification'}));expect(await screen.findByText(/status could not be confirmed/i)).toBeTruthy();expect(screen.queryByText('Clarification sent.')).toBeNull();});

it('does not claim delivery when a delivered state has no delivery receipt',async()=>{const current={id:'assess-1',tenant_id:'bank',legal_entity_id:'entity',relationship_id:'rel-1',version:7,status:'READY_TO_SEND',review_kind:'ONBOARDING',form_template_id:'form-1',form_template_version:2};assessment.loadCurrentVendorAssessment.mockResolvedValue({assessment:current});assessment.sendVendorAssessmentRequest.mockResolvedValue({assessment:{...current,version:9,status:'COLLECTING',current_request_id:'request-1'},request:{id:'request-1',tenant_id:'bank',legal_entity_id:'entity',subject_type:'VENDOR_RELATIONSHIP',subject_id:'rel-1',form_template_id:'form-1',form_template_version:2,origin:{type:'THIRD_PARTY_ASSESSMENT',id:'assess-1',version:1},deadline:new Date('2027-02-17T12:00').toISOString()},state:'DELIVERED'});render(<VendorReview relationship={relationship} onBack={()=>{}}/>);fireEvent.click(await screen.findByRole('button',{name:'Send assessment request'}));fireEvent.change(screen.getByLabelText('Recipient email'),{target:{value:'vendor@example.com'}});fireEvent.change(screen.getByLabelText('Response deadline'),{target:{value:'2027-02-17T12:00'}});fireEvent.click(screen.getByRole('button',{name:'Send request'}));expect(await screen.findByText(/Request status could not be confirmed/)).toBeTruthy();expect(screen.queryByText('Request sent.')).toBeNull();});

it('reports a request sent only with a matching delivered receipt',async()=>{const current={id:'assess-1',tenant_id:'bank',legal_entity_id:'entity',relationship_id:'rel-1',version:7,status:'READY_TO_SEND',review_kind:'ONBOARDING',form_template_id:'form-1',form_template_version:2};assessment.loadCurrentVendorAssessment.mockResolvedValue({assessment:current});assessment.sendVendorAssessmentRequest.mockResolvedValue({assessment:{...current,version:9,status:'COLLECTING',current_request_id:'request-1'},request:{id:'request-1',tenant_id:'bank',legal_entity_id:'entity',subject_type:'VENDOR_RELATIONSHIP',subject_id:'rel-1',form_template_id:'form-1',form_template_version:2,origin:{type:'THIRD_PARTY_ASSESSMENT',id:'assess-1',version:1},deadline:new Date('2027-02-17T12:00').toISOString()},state:'DELIVERED',delivery:{status:'DELIVERED',delivered_at:'2026-11-22T10:00:00Z'}});render(<VendorReview relationship={relationship} onBack={()=>{}}/>);fireEvent.click(await screen.findByRole('button',{name:'Send assessment request'}));fireEvent.change(screen.getByLabelText('Recipient email'),{target:{value:'vendor@example.com'}});fireEvent.change(screen.getByLabelText('Response deadline'),{target:{value:'2027-02-17T12:00'}});fireEvent.click(screen.getByRole('button',{name:'Send request'}));expect(await screen.findByText('Request sent.')).toBeTruthy();});

it('does not claim a clarification was sent without a delivered receipt',async()=>{const current={id:'assess-1',relationship_id:'rel-1',current_request_id:'request-1',submission_id:'submission-1',review_due_at:'2099-03-01T18:00:00Z',version:7,status:'UNDER_REVIEW',review_kind:'ONBOARDING'};assessment.loadCurrentVendorAssessment.mockResolvedValue({assessment:current});assessment.loadVendorAssessment.mockResolvedValue({assessment:current,answers:[{field_id:'q1',label:'Access?',visibility:'VISIBLE'}],documents:[],matters:[],coverage:{}});assessment.requestVendorAssessmentClarification.mockResolvedValue({assessment:{...current,version:9,status:'COLLECTING',current_request_id:'request-2'},state:'DELIVERED'});render(<VendorReview relationship={relationship} onBack={()=>{}}/>);fireEvent.click(await screen.findByRole('button',{name:'Request clarification'}));fireEvent.click(screen.getByLabelText('Access?'));fireEvent.change(screen.getByLabelText('Recipient email'),{target:{value:'vendor@example.com'}});fireEvent.change(screen.getByLabelText('Request details'),{target:{value:'Confirm the review date.'}});fireEvent.change(screen.getByLabelText('Response deadline'),{target:{value:'2099-02-20T12:00'}});fireEvent.click(screen.getByRole('button',{name:'Send clarification'}));expect(await screen.findByText(/Request status could not be confirmed/)).toBeTruthy();expect(screen.queryByText('Clarification sent.')).toBeNull();});
it('does not claim a replacement was sent without a delivered receipt',async()=>{const current={id:'assess-1',relationship_id:'rel-1',current_request_id:'request-1',version:7,status:'COLLECTING',review_kind:'ONBOARDING'};assessment.loadCurrentVendorAssessment.mockResolvedValue({assessment:current});assessment.reissueVendorAssessmentRequest.mockResolvedValue({assessment:{...current,version:9},request:{id:'request-1'},state:'DELIVERED'});render(<VendorReview relationship={relationship} onBack={()=>{}}/>);fireEvent.click(await screen.findByRole('button',{name:'Replace invitation'}));fireEvent.change(screen.getByLabelText('Recipient email'),{target:{value:'vendor@example.com'}});fireEvent.click(screen.getByRole('button',{name:'Send replacement'}));expect(await screen.findByText(/status could not be confirmed/i)).toBeTruthy();expect(screen.queryByText('Replacement sent.')).toBeNull();});
