import { useState } from "react";
import type { FormTemplate } from "./monitoringTypes";
import type { ResponseAssessmentDetail } from "./formAssessmentApi";
import type { VendorFormRow, VendorRequestInput } from "./vendorFormsApi";
import { FormBuilder } from "./components/FormBuilder";
import { ResponseAssessment } from "./components/forms/ResponseAssessment";
import { Notice } from "./components/ui";

const now = "2026-09-08T12:00:00Z";
export const assessmentEvidenceTemplate: FormTemplate = {
  id: "sample-assessment-form", tenant_id: "tenant-demo", code: "SAMPLE-VENDOR-SECURITY", name: "Sample · Vendor security evidence", purpose: "Review payment-service control and test evidence.", version: 4, status: "DRAFT", is_current: true, created_at: now, updated_at: now, scoring_mode: "RISK", presentation: { default_mode: "CLASSIC", allow_mode_switch: true }, sections: [{ id: "security", title: "Security evidence" }],
  fields: [
    { id: "test-report", section_id: "security", label: "Independent vulnerability test report", type: "file", required: true, accepted_formats: ["application/pdf"], assessment: { mode: "MANUAL", required: true, weight: 50, reviewer_role: "RISK_REVIEWER", rubric: [{ id: "sufficient", label: "Payment service covered; no unresolved high findings", points: 0 }, { id: "gap", label: "Scope or remediation evidence incomplete", points: 80 }] } },
    { id: "access-control", section_id: "security", label: "Are privileged access reviews complete?", type: "yes_no", required: true, options: ["Yes", "No"], scoring: { id: "access-control", weight: 50, answer_scores: { Yes: 0, No: 80 } }, assessment: { mode: "AUTOMATIC", required: false, weight: 50 } },
    { id: "context", section_id: "security", label: "Service changes since the previous review", type: "long_text", required: false, assessment: { mode: "NONE", required: false, weight: 100 } },
  ],
};
function sampleAssessment(responseID: string): ResponseAssessmentDetail {
  const assessed = responseID.includes("certificate");
  const current = !responseID.includes("historical");
  return { response_id: responseID, form_template_id: assessmentEvidenceTemplate.id, form_template_version: 4, version: assessed ? 1 : 0, current, may_review: current, state: assessed ? "ASSESSED" : "AWAITING_REVIEW", required_count: 1, reviewed_required_count: assessed ? 1 : 0, reviewed_count: assessed ? 1 : 0,
    fields: assessmentEvidenceTemplate.fields.map((field) => ({ field, may_review: true, answer: field.id === "test-report" ? { artifact_ids: ["sample-vulnerability-report"] } : { text: field.id === "access-control" ? "No" : "The provider added a payment processing region in July 2026." }, decision: assessed && field.id === "test-report" ? { id: "sample-certificate-judgement", field_id: field.id, outcome_id: "gap", points: 80, rationale: "The submitted report excludes the payment processing service. Request evidence covering this service and its remediation owners.", reviewer_id: "Sample Risk Reviewer", assessed_at: now } : undefined })),
    automatic_score: { state: "FINAL", final: true, mode: "RISK", direction: "HIGH_IS_POOR", raw_score: 80, adverse_score: 80, band: "HIGH", coverage: 1, calculated_at: now, profile_version: "sample-risk-v4", contribution_results: [{ id: "access-control", outcome: "PASSED", points: 80, weight: 50 }], rule_results: [] },
    assessed_score: { state: assessed ? "FINAL" : "PROVISIONAL", final: assessed, mode: "RISK", direction: "HIGH_IS_POOR", raw_score: 80, adverse_score: 80, band: "HIGH", coverage: assessed ? 1 : 0.5, calculated_at: now },
  };
}
const rowBase = { relationship_id: "vendor-relationship-payments", form_template_id: assessmentEvidenceTemplate.id, form_template_version: 4, deadline: "2026-09-18T16:00:00Z", updated_at: now, current: true, required_reviews: 0, completed_reviews: 0, missing_fields: [] };
const sampleRows: VendorFormRow[] = [
  { ...rowBase, request_id: "sample-superseded-request", title: "Sample · Previous vendor assurance request", response_state: "SUPERSEDED", current: false, required_count: 4, answered_required: 0, missing_fields: [{ id: "historical-question", label: "Historical assurance document" }] },
  { ...rowBase, request_id: "sample-missing-request", title: "Sample · Resilience evidence refresh", purpose: "Confirm the recovery exercise and remediation dates.", response_state: "IN_PROGRESS", recipient_hint: "r***@example.test", required_count: 5, answered_required: 3, deadline: "2026-09-05T16:00:00Z", missing_fields: [{ id: "recovery-test", label: "Latest recovery exercise report" }, { id: "remediation", label: "Remediation owner and due date" }] },
  { ...rowBase, request_id: "sample-review-request", response_id: "sample-response-review", title: "Sample · Payment-service security evidence", purpose: "Review current control and vulnerability testing evidence.", response_state: "SUBMITTED", recipient_hint: "s***@example.test", required_count: 3, answered_required: 3, submitted_at: now, assessment_state: "AWAITING_REVIEW", required_reviews: 1, score: sampleAssessment("sample-response-review").automatic_score, assessed_score: sampleAssessment("sample-response-review").assessed_score },
  { ...rowBase, request_id: "sample-certificate-request", response_id: "sample-response-certificate", title: "Sample · Certification review", response_state: "SUBMITTED", recipient_hint: "c***@example.test", required_count: 2, answered_required: 2, submitted_at: "2026-09-07T10:00:00Z", assessment_state: "ASSESSED", required_reviews: 1, completed_reviews: 1, score: sampleAssessment("sample-response-certificate").automatic_score, assessed_score: { ...sampleAssessment("sample-response-certificate").automatic_score!, band: "HIGH" } },
];

export function installVendorAssessmentEvidence() {
  const params = new URLSearchParams(window.location.search);
  if (!params.get("fixture")?.startsWith("vendor-form-assessment") && params.get("fixture") !== "field-assessment-builder" && params.get("fixture") !== "field-assessment-review") return;
  const previousFetch = globalThis.fetch.bind(globalThis);
  const assessments = new Map<string, ResponseAssessmentDetail>();
  const attempts = new Map<string, number>();
  globalThis.fetch = async (input, init) => {
    const raw = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    const url = new URL(raw, window.location.origin), method = init?.method ?? "GET";
    const json = (value: unknown, status = 200) => new Response(JSON.stringify(value), { status, headers: { "Content-Type": "application/json" } });
    const path = url.pathname;
    const findingLabels = ["Recovery exercise evidence missing", "Support escalation contacts outdated", "Service reporting schedule incomplete", "Access review evidence missing", "Data retention schedule unconfirmed"];
    const actionLabels = ["Provide the recovery exercise results", "Confirm the support escalation contacts", "Agree the service reporting schedule", "Complete the access review", "Confirm the data retention schedule"];
    if (/^\/api\/v1\/vendors\/[^/]+\/links$/.test(path)) {
      if (params.get("fixture") === "vendor-form-assessment-error") return json({ error: { message: "Sample linked findings unavailable" } }, 503);
      const relationship = decodeURIComponent(path.split("/")[4]!);
      return json({ items: relationship === "sample-relationship-hosting" || params.get("fixture") === "vendor-form-assessment-empty" ? [] : findingLabels.map((_, index) => ({ id: `sample-finding-link-${index}`, relationship_id: relationship, target_type: "MATTER", target_id: `sample-portfolio-finding-${index}`, state: "ACTIVE", purpose_code: "SOURCE_REGISTER_FINDING" })) });
    }
    const finding = /^\/api\/v1\/matters\/sample-portfolio-finding-([0-4])$/.exec(path);
    if (finding) {
      const index = Number(finding[1]);
      return json({ matter: { id: `sample-portfolio-finding-${index}`, type: "VENDOR_DEFICIENCY", title: findingLabels[index], status: "TRIAGE", known_facts: { sample: true, source_file: "Example vendor register.xlsx", source_range: `Example assessment · Finding ${index + 1}`, source_owner: "Morgan Ellis", source_rating: "High", source_period: "Q1 2026" }, due_at: "2026-03-31T16:00:00Z", updated_at: now }, type_label: "Vendor finding", status_label: "Initial review", next_action: "Review finding", actions: [{ id: `sample-portfolio-action-${index}`, title: actionLabels[index], description: "Synthetic example action", status: "PLANNED", due_at: "2026-03-31T16:00:00Z" }], links: [], decisions: [], verification_contracts: [], verification_results: [], response_packages: [], closure: { ready: false, reasons: [] } });
    }
    if (path === "/api/v1/access/overview") return json({ roles: [{ id: "sample-reviewer-role", code: "RISK_REVIEWER", name: "Risk reviewer", capabilities: [] }], can_configure: true });
    if (path === "/api/v1/forms/templates" && method === "GET") return json({ items: [{ template: assessmentEvidenceTemplate, active_version: 4, active_status: "ACTIVE" }] });
    if (path === "/api/v1/forms/responses" && method === "GET" && url.searchParams.get("subject_type") === "VENDOR_RELATIONSHIP") {
      const historical = url.searchParams.has("cursor"), relationshipID = url.searchParams.get("subject_id");
      return json({ items: [{ id: historical ? "sample-response-certificate-historical" : "sample-response-certificate", distribution_id: "sample-certificate-request", form_template_id: assessmentEvidenceTemplate.id, form_template_version: 4, title: historical ? "Sample · Earlier certification review" : "Sample · Certification review", subject_type: "VENDOR_RELATIONSHIP", subject_id: relationshipID, revision: historical ? 1 : 2, current: !historical, state: "FINAL", score: sampleAssessment("sample-response-certificate").automatic_score, completed_at: historical ? "2026-08-01T10:00:00Z" : now }], next_cursor: historical ? undefined : "sample-history-next" });
    }
    if (path === "/api/v1/vendors" && method === "GET") {
      const baseline = await previousFetch(input, init);
      const body = await baseline.json();
      const first = body.items?.[0];
      if (!first) return json(body);
      return json({ ...body, items: [first, { ...first, vendor: { ...first.vendor, id: "sample-vendor-hosting", legal_name: "Sample · Northshore Hosting Limited", trading_name: "Northshore" }, relationship: { ...first.relationship, id: "sample-relationship-hosting", vendor_id: "sample-vendor-hosting", service_name: "Recovery hosting" } }] });
    }
    if (path === "/api/v1/vendors/form-summaries" || /^\/api\/v1\/vendors\/[^/]+\/forms$/.test(path)) {
      if (params.get("fixture") === "vendor-form-assessment-error") return json({ error: { message: "Sample · Vendor form records are temporarily unavailable. Retry to check this service." } }, 503);
      const empty = params.get("fixture") === "vendor-form-assessment-empty";
      const partial = params.get("fixture") === "vendor-form-assessment-partial";
      if (path.endsWith("form-summaries")) return json({ items: (url.searchParams.get("relationship_ids") ?? "").split(",").map((id) => ({ relationship_id: id, outstanding_forms: empty ? 0 : 1, overdue_forms: empty ? 0 : 1, submitted_forms: empty ? 0 : 2, awaiting_review: empty ? 0 : 1, unassessed_forms: empty ? 0 : 1, assessed_forms: empty ? 0 : 1, highest_concern: empty ? undefined : "HIGH", partially_replaced_forms: partial ? 1 : 0, observed_at: now })) });
      const selectedFilter = url.searchParams.get("filter");
      const rows = sampleRows.filter((row) => !selectedFilter || selectedFilter === "AWAITING_VENDOR" && row.response_state !== "SUBMITTED" || selectedFilter === "AWAITING_REVIEW" && row.assessment_state === "AWAITING_REVIEW" || selectedFilter === "OVERDUE" && row.request_id === "sample-missing-request" || selectedFilter === "NOT_ASSESSED" && row.assessment_state !== "ASSESSED" || (selectedFilter === "WITH_RISKS" || selectedFilter === "HIGH_RISK") && row.response_state === "SUBMITTED");
      const relationshipID = decodeURIComponent(path.split("/")[4]!);
      return json({ items: empty ? [] : rows.map((row) => ({ ...row, response_currency: partial && row.request_id === "sample-certificate-request" ? "PARTIALLY_REPLACED" : "CURRENT", relationship_id: relationshipID, response_id: row.response_id ? `${row.response_id}:${relationshipID}` : undefined })), observed_at: now });
    }
    const match = /^\/api\/v1\/forms\/responses\/([^/]+)\/assessment$/.exec(path);
    if (match) {
      const id = decodeURIComponent(match[1]!);
      const detail = assessments.get(id) ?? sampleAssessment(id);
      if (method === "POST") {
        const body = JSON.parse(String(init?.body ?? "{}"));
        if (body.expected_version !== detail.version) return json({ error: { message: "Sample · Another reviewer changed this assessment. Reload before saving." } }, 409);
        detail.version++; detail.state = "ASSESSED"; detail.reviewed_required_count = 1; detail.reviewed_count = 1;
        detail.fields = detail.fields.map((item) => { const decision = body.decisions.find((value: { field_id: string }) => value.field_id === item.field.id); const outcome = item.field.assessment?.rubric?.find((value) => value.id === decision?.outcome_id); return decision && outcome ? { ...item, decision: { id: `sample-decision-${detail.version}`, field_id: item.field.id, outcome_id: outcome.id, points: outcome.points, rationale: decision.rationale, reviewer_id: "Sample Risk Reviewer", assessed_at: now } } : item; });
        detail.assessed_score = { ...detail.automatic_score!, coverage: 1 }; assessments.set(id, detail);
      }
      return json(detail);
    }
    if (path === "/api/v1/vendors/form-requests" && method === "POST") {
      const body = JSON.parse(String(init?.body ?? "{}")) as VendorRequestInput;
      return json({ batch_id: body.batch_id, items: body.targets.map((target, index) => { const key = `${body.batch_id}:${target.relationship_id}`, count = attempts.get(key) ?? 0; attempts.set(key, count + 1); return { relationship_id: target.relationship_id, status: index === 1 && count === 0 ? "PREPARED" : "CREATED", distribution_id: `sample-distribution-${target.relationship_id}`, error: index === 1 && count === 0 ? "Sample · Delivery setup needs retry." : undefined }; }) });
    }
    return previousFetch(input, init);
  };
}

export function FieldAssessmentEvidencePage({ review = false }: { review?: boolean }) {
  const [template, setTemplate] = useState(assessmentEvidenceTemplate);
  return <main><div id="cs-overlay-root" className="cs-overlay-root" aria-live="off"/><Notice tone="info">Sample data · Vendor field assessment review fixture, 8 September 2026.</Notice>{review ? <ResponseAssessment responseID="sample-response-review"/> : <FormBuilder initialValue={template} onSaved={setTemplate} onCancel={() => { window.location.hash = "forms"; }} saveDraft={async (input) => ({ ...template, ...input, version: template.version + 1 })}/>}</main>;
}
