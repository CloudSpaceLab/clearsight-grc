import type { VendorCollection, VendorCollectionResolution } from "./vendorCollectionApi";
import type { DocumentOccurrence } from "./submittedDocumentApi";
import type { VendorAssessment, VendorAssessmentReviewView } from "./vendorAssessmentTypes";

// Imported only by evidenceMain. Sample decisions never enter the production API.
export function installVendorCollectionEvidence() {
  const fixture = new URLSearchParams(window.location.search).get("fixture") ?? "";
  if (!fixture.startsWith("vendor-collection")) return;
  const original = globalThis.fetch.bind(globalThis);
  const now = "2026-09-08T10:00:00Z";
  const relationshipID = "vendor-relationship-payments";
  const unscannedFixture = fixture === "vendor-collection-unscanned-allowed" || fixture === "vendor-collection-unscanned-blocked";
  let assessment: VendorAssessment = {
    id: "sample-collection-assessment", tenant_id: "bank", legal_entity_id: "entity", relationship_id: relationshipID,
    review_kind: "ONBOARDING", source_trigger: "INITIAL", stable_episode_key: "sample-collection",
    status: fixture.includes("prepare") ? "READY_TO_SEND" : "UNDER_REVIEW", current_request_id: fixture.includes("prepare") ? undefined : "sample-collection-request",
    form_template_id: "sample-collection-form", form_template_version: 1, review_due_at: "2099-09-30T23:59:59Z",
    started_by_principal_id: "sample-owner", started_at: now, review_matter_id: "sample-review", version: 4, created_at: now, updated_at: now,
  };
  const source: DocumentOccurrence = {
    id: "sample-iso-occurrence", artifact_id: "sample-iso-artifact", request_id: "sample-earlier-request", submission_id: "sample-earlier-submission", submission_channel: "MAGIC_LINK", field_id: "iso",
    relationship_id: relationshipID, form_template_version: 1, form_title: "Sample · Earlier security review", field_label: "ISO 27001 assurance",
    file_name: "ISO 27001 certificate.pdf", file_kind: "PDF", media_type: "application/pdf", size_bytes: 1024, sha256: "sample-digest",
    artifact_status: unscannedFixture ? "STORED_UNSCANNED" : "AVAILABLE", ...(unscannedFixture ? { demo_unscanned_allowed: fixture.endsWith("allowed") } : {}), uploaded_at: "2026-08-03T09:00:00Z", submitted_at: "2026-08-03T10:00:00Z", current: true, expires_on: "2099-08-03T00:00:00Z",
  };
  const resolution: VendorCollectionResolution = { id: "sample-link", version: 1, source, reconciled_by: "Sample · Security reviewer", reconciled_at: now, rationale: "Certificate covers the vendor entity and the payment service under review." };
  let collection: VendorCollection = {
    assessment_id: assessment.id, assessment_version: assessment.version, request_id: assessment.current_request_id, request_version: 2,
    prepared: Boolean(assessment.current_request_id), can_reconcile: !fixture.includes("readonly"), observed_at: now, vendor_pending_count: 2, bank_pending_count: 1,
    audience_hint: "s***@example.test", deadline: "2099-09-20T23:59:59Z",
    fields: [
      { field_id: "vapt", label: "Vulnerability test report", type: "vendor_document", required: true, collection_state: "MISSING", vendor_action_required: true, bank_review_state: "NOT_REQUIRED" },
      { field_id: "iso", label: "ISO 27001 assurance", type: "vendor_document", required: true, collection_state: "REUSED", vendor_action_required: false, bank_review_state: "PENDING", resolution },
      { field_id: "continuity", label: "Business continuity assurance", type: "vendor_document", required: true, collection_state: "RECEIVED", vendor_action_required: false, bank_review_state: "VALIDATED" },
      { field_id: "service", label: "Service description", type: "long_text", required: true, collection_state: "RECEIVED", vendor_action_required: false, bank_review_state: "NOT_REQUIRED" },
      { field_id: "transfer", label: "International transfer assessment", type: "vendor_document", required: true, collection_state: "CONDITION_UNKNOWN", vendor_action_required: true, bank_review_state: "NOT_REQUIRED" },
    ],
  };
  if (fixture.includes("empty")) collection = { ...collection, fields: [], vendor_pending_count: 0, bank_pending_count: 0 };
  if (fixture === "vendor-collection-unscanned-blocked") collection = { ...collection, vendor_pending_count: 3, bank_pending_count: 0, fields: collection.fields.map((field) => field.field_id === "iso" ? { ...field, collection_state: "MISSING", vendor_action_required: true, bank_review_state: "NOT_REQUIRED" } : field) };
  if (fixture.includes("replaced")) collection = { ...collection, vendor_pending_count: 3, bank_pending_count: 0, fields: collection.fields.map((field) => field.field_id === "iso" ? { ...field, collection_state: "MISSING", vendor_action_required: true, bank_review_state: "NOT_REQUIRED", resolution: { ...resolution, source: { ...source, current: false } } } : field) };
  if (fixture.includes("long")) collection.fields[0]!.label = "Independent vulnerability assessment and penetration test report for the payment processing service and its supporting infrastructure";
  function review(): VendorAssessmentReviewView {
    return { assessment, requests: [], response: { submission_id: "sample-partial-submission", request_id: "sample-collection-request", submitted_at: now, answer_count: 2, artifact_count: 1 }, answers: [{ field_id: "service", label: "Service description", type: "long_text", required: true, visibility: "VISIBLE", value: { text: "Card payment processing and settlement." } }], coverage: { visible_fields: 5, answered_fields: 2, required_fields: 5, answered_required: 2, ratio: .4 },
      documents: collection.fields.filter((field) => field.resolution || field.field_id === "continuity").map((field) => ({
        field_id: field.field_id, artifact_id: field.resolution?.source.artifact_id ?? "sample-continuity", request_id: field.resolution?.source.request_id ?? "sample-collection-request",
        file_name: field.resolution?.source.file_name ?? "Business continuity certificate.pdf", media_type: "application/pdf", size_bytes: 1024,
        artifact_status: field.resolution?.source.artifact_status ?? "AVAILABLE", demo_unscanned_allowed: field.resolution?.source.demo_unscanned_allowed, status: field.bank_review_state === "VALIDATED" ? "VALIDATED" : "SUBMITTED", evidence_class: "VENDOR_SUPPLIED", document_type: field.field_id,
      })), matters: [] };
  }
  let errorAttempts = 0;
  let conflictAttempts = 0;
  globalThis.fetch = async (input, init) => {
    const raw = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    const url = new URL(raw, window.location.origin), path = url.pathname;
    const method = init?.method ?? "GET";
    const json = (value: unknown, status = 200) => new Response(JSON.stringify(value), { status, headers: { "Content-Type": "application/json" } });
    if (path === "/api/v1/vendors/form-summaries") return json({ items: (url.searchParams.get("relationship_ids") ?? "").split(",").filter(Boolean).map((id) => ({ relationship_id: id, outstanding_forms: id === relationshipID ? 1 : 0, overdue_forms: 0, submitted_forms: id === relationshipID ? 1 : 0, awaiting_review: id === relationshipID ? 1 : 0, unassessed_forms: 0, assessed_forms: 0, observed_at: now })) });
    if (path === `/api/v1/vendors/${relationshipID}/forms`) return json({ items: [{ request_id: "sample-collection-request", relationship_id: relationshipID, form_template_id: "sample-collection-form", form_template_version: 1, title: "Sample · Vendor due diligence", response_state: "SUBMITTED", deadline: collection.deadline, updated_at: now, submitted_at: now, required_count: 5, answered_required: 2, missing_fields: collection.fields.filter((field) => field.collection_state === "MISSING").map((field) => ({ id: field.field_id, label: field.label })), assessment_state: "IN_REVIEW", required_reviews: 2, completed_reviews: 1, current: true, recipient_hint: collection.audience_hint }], observed_at: now });
    if (path === `/api/v1/vendors/${relationshipID}/activation`) return json({ eligible: false, policy: { id: "sample-policy", policy_number: 1, version: 1, effective_from: now, status: "ACTIVE" }, gates: [{ code: "CURRENT_ASSESSMENT", satisfied: false, explanation: "Sample · Due diligence review is incomplete." }] });
    if (path === `/api/v1/vendors/${relationshipID}/assessments/current`) return json({ assessment });
    if (path === `/api/v1/vendor-assessments/${assessment.id}`) return json(review());
    if (path === `/api/v1/vendor-assessments/${assessment.id}/collection`) {
      if (fixture.includes("loading")) return new Promise<Response>(() => undefined);
      if (fixture.includes("error") && errorAttempts++ === 0) return json({ error: { message: "Sample · Request checklist could not be loaded." } }, 503);
      return json(collection);
    }
    if (path.endsWith("/prepare-request") && path.includes(assessment.id)) {
      const body = JSON.parse(String(init?.body));
      assessment = { ...assessment, current_request_id: "sample-collection-request", version: assessment.version + 1 };
      collection = { ...collection, prepared: true, request_id: assessment.current_request_id, deadline: body.deadline, assessment_version: assessment.version };
      return json({ assessment, request: { id: assessment.current_request_id, status: "READY", deadline: collection.deadline }, state: "PREPARED" });
    }
    if (path.includes(`/vendor-assessments/${assessment.id}/collection/`) && path.endsWith("/reconcile") && method === "POST") {
      const body = JSON.parse(String(init?.body)), fieldID = decodeURIComponent(path.split("/").at(-2)!);
      if (fixture.includes("conflict") && conflictAttempts++ === 0) {
        assessment = { ...assessment, version: assessment.version + 1 };
        collection = { ...collection, assessment_version: assessment.version };
        return json({ error: { message: "Sample · Request changed." } }, 409);
      }
      if (body.expected_version !== assessment.version) return json({ error: { message: "Sample · Assessment changed. Reload the checklist." } }, 409);
      const receipt = { ...resolution, id: `sample-link-${fieldID}`, rationale: body.rationale, source: { ...source, file_name: "Security test report.pdf", artifact_id: body.source_artifact_id } };
      assessment = { ...assessment, version: assessment.version + 1 };
      collection = { ...collection, assessment_version: assessment.version, request_version: collection.request_version! + 1, vendor_pending_count: 1, bank_pending_count: 2, fields: collection.fields.map((field) => field.field_id === fieldID ? { ...field, collection_state: "REUSED", vendor_action_required: false, bank_review_state: "PENDING", resolution: receipt } : field) };
      return json({ collection, receipt });
    }
    if (path.includes(`/vendor-assessments/${assessment.id}/documents/`) && path.endsWith("/validate")) {
      const body = JSON.parse(String(init?.body)), artifact = path.split("/").at(-2);
      assessment = { ...assessment, version: assessment.version + 1 };
      collection = { ...collection, assessment_version: assessment.version, bank_pending_count: Math.max(0, collection.bank_pending_count - 1), fields: collection.fields.map((field) => (body.field_id ? field.field_id === body.field_id : field.resolution?.source.artifact_id === artifact) ? { ...field, bank_review_state: body.decision === "VALIDATE" ? "VALIDATED" : "REJECTED" } : field) };
      return json(review());
    }
    if (path === "/api/v1/forms/documents") return json({ items: [{ ...source, id: "sample-test-report", field_id: "test", artifact_id: "sample-test-artifact", file_name: "Security test report.pdf", field_label: "Vulnerability testing" }, source] });
    return original(input, init);
  };
}
