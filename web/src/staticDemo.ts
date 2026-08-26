import type { CoverageDecision, DocumentCoverage, DocumentImport, ProposalStatus } from "./documentTypes";
import type { FormTemplate } from "./monitoringTypes";
import type { VendorAssessment, VendorAssessmentReviewView } from "./vendorAssessmentTypes";
import type { VendorRelationshipLink } from "./vendorLinkTypes";
import type { VendorCriticality, VendorPrivacyRole, VendorRelationshipAggregate } from "./vendorTypes";
import type { VendorWorkRequest, VendorWorkResponseView, VendorWorkSendOutcome } from "./vendorWorkTypes";

export const staticDemoEnabled = import.meta.env.VITE_STATIC_DEMO === "true";

export class StaticDemoHTTPError extends Error {
  constructor(readonly status: number, readonly code: string, message: string) {
    super(message);
    this.name = "StaticDemoHTTPError";
  }
}

const now = "2026-08-06T15:30:00Z";
const future = new Date(Date.now() + 72 * 60 * 60 * 1000).toISOString();
const programID = "program-ndpa";
const matterID = "matter-gaid-change";
const evidenceID = "evidence-annual-return";
const vendorRelationshipID = "vendor-relationship-payments";

const program = {
  id: programID, tenant_id: "bank-demo", legal_entity_id: "bank-ng", code: "NDPA", name: "Nigeria Data Protection Programme", type: "PRIVACY", status: "ACTIVE", owning_function: "Data Protection Office", owner_principal_id: "role-dpo", authority_principal_id: "role-cro", jurisdiction: "Nigeria", scope: { legal_entity: "Bank Nigeria", business_lines: ["Retail", "Corporate", "Digital"] }, effective_from: "2025-01-01T00:00:00Z", created_at: "2026-07-01T09:00:00Z", updated_at: now, version: 12,
};
const programSummary = {
  program, state_label: "Evidence incomplete", overall_state: "EVIDENCE_INSUFFICIENT", reasons: [{ code: "EVIDENCE_COVERAGE", summary: "Two annual-return evidence sections still need an accountable owner.", object_type: "EVIDENCE_CONTRACT", object_id: "contract-return" }], reasons_total: 1, reasons_omitted: 0, open_matter_count: 2, requirement_count: 5, safeguard_count: 5, evidence_check_count: 5, program_version: 12, assessed_program_version: 12, projection_version: 3, projection_stale: false, state_generated_at: now,
};
const programDetail = {
  state_label: "Evidence incomplete", program,
  requirements: [
    { id: "req-1", code: "NDPA-RET-01", title: "Maintain accountable data-processing governance", statement: "The bank must maintain approved responsibilities, records and oversight for personal-data processing.", status: "APPROVED", source_anchor: "Nigeria Data Protection Act 2023 · governance duties" },
    { id: "req-2", code: "GAID-RETURN-01", title: "Submit the annual compliance return", statement: "The bank must complete the annual return with source-linked evidence and internal approval before filing.", status: "APPROVED", source_anchor: "GAID 2025 · annual return" },
  ],
  applicability: [{ id: "app-1", requirement_id: "req-1", status: "APPLICABLE", rationale: "The bank processes customer and employee personal data in Nigeria." }],
  control_objectives: [{ id: "obj-1", code: "PRIV-GOV", name: "Accountable privacy governance", outcome: "Responsibilities and decisions remain current and evidenced.", status: "ACTIVE" }],
  control_implementations: [{ id: "control-1", name: "Annual privacy compliance review", description: "DPO-led review with accountable evidence owners and executive sign-off.", implementation_type: "PROCESS", status: "IMPLEMENTED" }],
  requirement_control_links: [{ requirement_id: "req-1", implementation_id: "control-1" }],
  evidence_contracts: [
    { id: "contract-return", code: "GAID-RETURN", name: "Annual return evidence package", claim: "Every required return section has an owner, authoritative source, review status and approval date.", status: "ACTIVE", freshness_minutes: 525600, minimum_coverage: 1 },
    { id: "contract-training", code: "PRIV-TRAIN", name: "Privacy role training", claim: "Assigned privacy responsibilities have current training evidence.", status: "ACTIVE", freshness_minutes: 525600, minimum_coverage: .95 },
  ],
  evidence_assessments: [{ id: "assessment-1", contract_id: "contract-return", conclusion: "PARTIALLY_SUPPORTED", coverage: .8, assessed_at: now }],
  current_state: { id: "state-1", overall_state: "EVIDENCE_INSUFFICIENT", dimensions: { requirements: "CURRENT", controls: "CURRENT", evidence: "EVIDENCE_INSUFFICIENT" }, reasons: programSummary.reasons, open_matter_count: 2, generated_at: now, program_version: 12, projection_version: 3 },
  triggers: [{ id: "trigger-1", type: "REQUIREMENT_CHANGED", observed_at: "2026-08-04T10:00:00Z", source: "NDPC publication monitor" }],
};

let programReviewAcknowledged = false;
function programReviewDigest() {
  const checkpoint = {
    id: "review-ndpa-1", tenant_id: "bank-demo", program_id: programID, principal_id: "role-cro",
    program_version: programReviewAcknowledged ? 12 : 10, projection_version: programReviewAcknowledged ? 3 : 2,
    accepted_at: programReviewAcknowledged ? now : "2026-08-04T09:00:00Z",
  };
  if (programReviewAcknowledged) return {
    program_id: programID, state: "CURRENT", review_required: false, checkpoint,
    current_program_version: 12, current_projection_version: 3, current_overall: "EVIDENCE_INSUFFICIENT", baseline_overall: "EVIDENCE_INSUFFICIENT",
    open_matter_count: 2, open_matter_delta: 0, changes: [], changes_total: 0, changes_omitted: 0,
    current_exceptions: programSummary.reasons, current_exceptions_total: 1, new_exceptions: [], new_exceptions_total: 0, resolved_exceptions: [], resolved_exceptions_total: 0,
  };
  return {
    program_id: programID, state: "CHANGED", review_required: true, checkpoint,
    current_program_version: 12, current_projection_version: 3, current_overall: "EVIDENCE_INSUFFICIENT", baseline_overall: "CURRENT",
    open_matter_count: 2, open_matter_delta: 2,
    changes: [
      { kind: "STATE", summary: "Overall status changed from current to evidence insufficient." },
      { kind: "EVIDENCE", summary: "Evidence for Annual return evidence package was assessed as partially supported.", object_type: "EVIDENCE_CONTRACT", object_id: "contract-return" },
      { kind: "CHANGE", summary: "Observed change: requirement changed.", object_type: "REQUIREMENT", object_id: "req-2" },
    ],
    changes_total: 3, changes_omitted: 0,
    current_exceptions: programSummary.reasons, current_exceptions_total: 1,
    new_exceptions: programSummary.reasons, new_exceptions_total: 1,
    resolved_exceptions: [], resolved_exceptions_total: 0,
  };
}

const matter = {
  id: matterID, tenant_id: "bank-demo", reference: "CHG-2026-0042", type: "REGULATORY_CHANGE", status: "ACTION_IN_PROGRESS", priority: 4, title: "Implement GAID annual-return evidence requirements", summary: "Update the annual return process, evidence ownership and internal approval date.", scope: { journey_code: "REGULATORY_CHANGE", filing_year: 2027 }, known_facts: { official_source: "GAID 2025", filing_deadline: "31 March 2027", affected_process: "Annual privacy compliance return", complete_sections: 8, required_sections: 10 }, missing_facts: ["Owner for the processor register section", "Final DPCO review date"], contradictions: [], owner_principal_id: "role-dpo", required_authority: "AUTHORIZER", due_at: future, created_at: "2026-08-04T10:00:00Z", updated_at: now, version: 8,
};
const matterSummary = { matter, type_label: "Regulatory change", status_label: "Work in progress", next_action: "Complete the remaining evidence ownership updates", program_count: 1, open_action_count: 1, outcome_check_count: 1 };
const matterDetail = {
  type_label: "Regulatory change", status_label: "Work in progress", next_action: "Complete the remaining evidence ownership updates", matter,
  links: [{ id: "link-1", program_id: programID, relationship: "AFFECTS" }],
  decisions: [{ id: "decision-1", type: "IMPLEMENTATION_APPROACH", status: "APPROVED", selected_option: "UPDATE_CURRENT_PROCESS", rationale: "Use the existing annual return process with source-linked owners and an earlier internal approval date.", decided_at: "2026-08-05T11:00:00Z" }],
  actions: [{ id: "action-1", title: "Complete the annual return evidence checklist", description: "Assign the two remaining sections and record the DPCO review date.", status: "IN_PROGRESS", due_at: future }],
  verification_contracts: [{ id: "verify-1", expected_outcome: "All ten return sections have an owner, authoritative source and approved review status.", status: "ACTIVE", observation_period_minutes: 0 }],
  verification_results: [], response_packages: [], closure: { ready: false, reasons: ["One action remains in progress.", "The independent outcome check has not passed."] },
};

const evidenceRequest = {
  id: evidenceID, tenant_id: "bank-demo", subject_type: "MATTER", subject_id: matterID, title: "Confirm the remaining annual-return evidence owners", purpose: "Complete the evidence ownership record before the DPCO review.", why_you: "You own the affected privacy operations records.", sensitivity: "INTERNAL", audience_type: "INTERNAL", estimated_minutes: 2, deadline: future, known_facts: { filing_year: "2027", completed_sections: "8 of 10", internal_approval_date: "1 March 2027" }, fields: [{ id: "processor_register_owner", label: "Processor register owner", type: "text", required: true, description: "Name the accountable role or position." }, { id: "dpco_review_date", label: "DPCO review date", type: "text", required: true, description: "Enter the approved review date." }], status: "READY", version: 1, created_at: now, updated_at: now,
};

let vendorRelationships: VendorRelationshipAggregate[] = [{
  vendor: {
    id: "vendor-acme-processing", tenant_id: "bank-demo", legal_name: "Acme Processing Limited", trading_name: "Acme Processing",
    registration_ref: "RC-10001", jurisdiction: "Nigeria", source_id: "procurement", external_ref: "vendor-10001", status: "ACTIVE",
    created_at: "2026-07-10T09:00:00Z", updated_at: now, version: 1,
  },
  relationship: {
    id: vendorRelationshipID, tenant_id: "bank-demo", legal_entity_id: "bank-ng", vendor_id: "vendor-acme-processing",
    service_name: "Card transaction processing", business_owner_principal_id: "role-payments-owner", criticality: "IMPORTANT", privacy_role: "PROCESSOR",
    status: "PROPOSED", effective_from: "2026-09-01T00:00:00Z", renewal_at: "2027-09-01T00:00:00Z",
    source_id: "procurement", external_ref: "vendor-10001", created_at: "2026-07-10T09:00:00Z", updated_at: now, version: 1,
  },
}];

const vendorDueDiligenceForm: FormTemplate = {
  id: "form-vendor-due-diligence",
  tenant_id: "bank-demo",
  code: "VENDOR-DUE-DILIGENCE",
  name: "Vendor security and privacy review",
  purpose: "Collect the vendor information and supporting documents required for onboarding review.",
  presentation: { default_mode: "WIZARD", allow_mode_switch: true },
  sections: [
    { id: "contact", title: "Company contact", help: "Confirm who can answer follow-up questions about this submission." },
    { id: "service", title: "Service and data", help: "Describe the service and the bank information it uses." },
    { id: "controls", title: "Security controls", help: "Confirm the controls in operation and provide current supporting documents." },
    { id: "attestation", title: "Submission confirmation", help: "An authorized representative must confirm the response before submission." },
  ],
  fields: [
    { id: "contact_email", section_id: "contact", label: "Security contact email", type: "email", required: true, constraints: { max_length: 254 } },
    { id: "service_description", section_id: "service", label: "Service description", type: "long_text", required: true, constraints: { min_length: 20, max_length: 1200 } },
    { id: "data_classes", section_id: "service", label: "Bank information used", type: "multi_select", required: true, options: ["Customer personal data", "Payment data", "Employee data", "Confidential business data", "No bank information"], constraints: { min_selections: 1, max_selections: 4 } },
    { id: "subprocessors", section_id: "service", label: "Do subcontractors process bank information?", type: "yes_no", required: true },
    { id: "subprocessor_details", section_id: "service", label: "Subcontractor details", type: "long_text", required: true, constraints: { min_length: 10, max_length: 1000 }, condition: { field_id: "subprocessors", operator: "EQUALS", values: ["yes"] } },
    { id: "security_framework", section_id: "controls", label: "Primary security framework", type: "single_select", required: true, options: ["ISO 27001", "SOC 2", "PCI DSS", "NIST CSF", "Other", "None"] },
    { id: "security_document", section_id: "controls", label: "Current independent assurance document", type: "vendor_document", required: true, accepted_formats: ["application/pdf"], constraints: { max_files: 1, max_file_bytes: 25_000_000 } },
    { id: "authorized_attestation", section_id: "attestation", label: "Authorized representative confirmation", type: "attestation", required: true, attestation: "I confirm that this response is complete and accurate to the best of my knowledge." },
  ],
  status: "ACTIVE",
  is_current: true,
  version: 3,
  created_at: "2026-08-01T09:00:00Z",
  updated_at: now,
};

const vendorProgramLink: VendorRelationshipLink = {
  id: "vendor-link-program-payments", tenant_id: "bank-demo", legal_entity_id: "bank-ng", relationship_id: vendorRelationshipID,
  target_type: "PROGRAM", target_id: programID, purpose_code: "CONTROL_ASSURANCE", purpose_label: "Payment-service control assurance",
  state: "ACTIVE", created_by: "role-payments-owner", version: 1, created_at: "2026-08-18T09:00:00Z", updated_at: now,
};
const vendorMatterLink: VendorRelationshipLink = {
  id: "vendor-link-matter-payments", tenant_id: "bank-demo", legal_entity_id: "bank-ng", relationship_id: vendorRelationshipID,
  target_type: "MATTER", target_id: matterID, purpose_code: "REMEDIATION", purpose_label: "Annual-return evidence update",
  state: "ACTIVE", created_by: "role-dpo", version: 1, created_at: "2026-08-19T09:00:00Z", updated_at: now,
};
const vendorWorkLinks = [vendorProgramLink, vendorMatterLink];
let vendorWorkFixture = "";
let vendorWorkRequests: VendorWorkRequest[] = [];

function vendorWorkRecord(link: VendorRelationshipLink, state: VendorWorkRequest["state"], deliveryState: VendorWorkRequest["delivery_state"], version: number): VendorWorkRequest {
  const submitted = state === "RESPONSE_RECEIVED" || state === "UNDER_REVIEW" || state === "ACCEPTED";
  return {
    id: link.target_type === "PROGRAM" ? "vendor-work-program-controls" : "vendor-work-matter-evidence",
    tenant_id: "bank-demo", legal_entity_id: "bank-ng", relationship_id: vendorRelationshipID, relationship_link_id: link.id,
    target_type: link.target_type, target_id: link.target_id, purpose: link.target_type === "PROGRAM" ? "Confirm payment-service controls" : "Complete annual-return evidence",
    instructions: link.target_type === "PROGRAM" ? "Complete the control questions and provide the current independent assurance report." : "Upload the signed evidence schedule and confirm the remaining service-control details.",
    owner_principal_id: link.target_type === "PROGRAM" ? "role-payments-owner" : "role-dpo", reviewer_principal_id: state === "UNDER_REVIEW" || state === "ACCEPTED" ? "role-cro" : undefined,
    form_template_id: vendorDueDiligenceForm.id, form_template_version: vendorDueDiligenceForm.version, presentation: "WIZARD",
    current_request_id: state === "PREPARING" ? undefined : "vendor-work-capture-1", current_invitation_id: state === "PREPARING" ? undefined : "vendor-work-invitation-1", current_capture_sequence: state === "PREPARING" ? 0 : 1,
    submission_id: submitted ? "vendor-work-submission-1" : undefined, state, delivery_state: deliveryState,
    recovery: deliveryState === "RETRY_REQUIRED" ? "Email delivery was not confirmed. Retry delivery to issue a replacement secure link." : undefined,
    review_rationale: state === "ACCEPTED" ? "The response and current assurance report address this request." : undefined,
    due_at: "2026-09-30T23:59:59Z", version, created_at: "2026-08-20T09:00:00Z", updated_at: now,
    response_received_at: submitted ? "2026-08-25T14:20:00Z" : undefined, review_started_at: state === "UNDER_REVIEW" || state === "ACCEPTED" ? "2026-08-25T15:00:00Z" : undefined,
    accepted_at: state === "ACCEPTED" ? "2026-08-26T11:00:00Z" : undefined,
  };
}

function syncVendorWorkFixture(fixture: string) {
  if (vendorWorkFixture === fixture) return;
  vendorWorkFixture = fixture;
  if (fixture === "vendor-work-submitted") vendorWorkRequests = [vendorWorkRecord(vendorMatterLink, "RESPONSE_RECEIVED", "DELIVERED", 4)];
  else if (fixture === "vendor-work-partial-delivery") vendorWorkRequests = [vendorWorkRecord(vendorProgramLink, "AWAITING_VENDOR", "RETRY_REQUIRED", 3)];
  else if (fixture === "vendor-work-accepted") vendorWorkRequests = [vendorWorkRecord(vendorMatterLink, "ACCEPTED", "DELIVERED", 6)];
  else vendorWorkRequests = [];
}

function vendorWorkResponse(work: VendorWorkRequest): VendorWorkResponseView {
  const requestID = work.current_request_id!;
  return {
    work,
    request: { request_id: requestID, status: "SUBMITTED", deadline: work.due_at, form_template_id: work.form_template_id, form_template_version: work.form_template_version, presentation: { default_mode: "WIZARD", allow_mode_switch: true } },
    response: { submission_id: work.submission_id!, request_id: requestID, submitted_at: work.response_received_at! },
    answers: [
      { field_id: "service_description", label: "Service description", type: "LONG_TEXT", required: true, visibility: "VISIBLE", value: { text: "Card transaction processing, settlement routing and operational support for the bank." }, provenance: { origin: "SOURCE_PREFILLED", source_receipt: { source_id: "Vendor register", observed_at: "2026-08-24T09:00:00Z" } } },
      { field_id: "data_classes", label: "Bank information used", type: "MULTI_SELECT", required: true, visibility: "VISIBLE", value: { values: ["Customer personal data", "Payment data"] }, provenance: { origin: "RESPONDENT_ENTERED" } },
      { field_id: "subprocessors", label: "Do subcontractors process bank information?", type: "YES_NO", required: true, visibility: "VISIBLE", value: { text: "Yes" }, provenance: { origin: "RESPONDENT_ENTERED" } },
      { field_id: "subprocessor_details", label: "Subcontractor details", type: "LONG_TEXT", required: true, visibility: "CONDITIONALLY_OMITTED" },
      { field_id: "control_owner", label: "Control owner", type: "SHORT_TEXT", required: true, visibility: "VISIBLE" },
    ],
    documents: [
      { field_id: "security_document", artifact_id: "artifact-vendor-iso27001", file_name: "acme-iso-27001-certificate.pdf", media_type: "application/pdf", size_bytes: 684220, artifact_status: "AVAILABLE", evidence_class: "VENDOR_SUPPLIED", document_type: "ISO_27001_CERTIFICATE", reference: "ISO-27001-ACME-2026", issued_by: "Accredited certification body", issued_on: "2026-03-01", expires_on: "2027-03-01" },
      { field_id: "security_document", artifact_id: "artifact-vendor-test-report", file_name: "acme-penetration-test-report.pdf", media_type: "application/pdf", size_bytes: 921430, artifact_status: "QUARANTINED", evidence_class: "VENDOR_SUPPLIED", document_type: "PENETRATION_TEST_REPORT" },
    ],
  };
}

let vendorAssessment: VendorAssessment | null = null;

function submittedVendorAssessment(): VendorAssessment {
  return {
    id: "vendor-assessment-payments-2026",
    tenant_id: "bank-demo",
    legal_entity_id: "bank-ng",
    relationship_id: vendorRelationshipID,
    review_kind: "ONBOARDING",
    source_trigger: "INITIAL",
    stable_episode_key: "vendor-relationship-payments:ONBOARDING:2026",
    status: "SUBMITTED",
    form_template_id: vendorDueDiligenceForm.id,
    form_template_version: vendorDueDiligenceForm.version,
    current_request_id: "vendor-request-payments-2026",
    submission_id: "vendor-submission-payments-2026",
    review_matter_id: "matter-vendor-review-payments",
    review_due_at: "2026-09-25T23:59:59Z",
    started_by_principal_id: "role-payments-owner",
    started_at: "2026-08-20T09:00:00Z",
    submitted_at: "2026-08-25T14:20:00Z",
    version: 4,
    created_at: "2026-08-20T09:00:00Z",
    updated_at: "2026-08-25T14:20:00Z",
  };
}

function fixtureVendorAssessment(fixture: string): VendorAssessment | null {
  const submitted = submittedVendorAssessment();
  switch (fixture) {
    case "vendor-ready":
    case "vendor-partial-delivery":
      return { ...submitted, status: "READY_TO_SEND", current_request_id: undefined, submission_id: undefined, submitted_at: undefined, version: 2, updated_at: "2026-08-20T09:05:00Z" };
    case "vendor-collecting":
      return { ...submitted, status: "COLLECTING", submission_id: undefined, submitted_at: undefined, version: 3, updated_at: "2026-08-21T10:00:00Z" };
    case "vendor-submitted": return submitted;
    case "vendor-completed":
      return { ...submitted, status: "COMPLETED", reviewer_principal_id: "role-cro", review_started_at: "2026-08-25T15:00:00Z", completed_at: "2026-08-26T11:00:00Z", conclusion: "SATISFACTORY_WITH_CONDITIONS", conclusion_rationale: "Proceed after the access-control action is complete.", conclusion_uncertainty: "The next resilience exercise remains due.", version: 6, updated_at: "2026-08-26T11:00:00Z" };
    default: return null;
  }
}

function submittedVendorReview(assessment: VendorAssessment): VendorAssessmentReviewView {
  return {
    assessment,
    requests: [{ request_id: assessment.current_request_id!, purpose: "INITIAL", sequence: 1, origin_sequence: 1, status: "SUBMITTED", deadline: "2026-09-12T23:59:59Z", form_template_id: assessment.form_template_id, form_template_version: assessment.form_template_version }],
    response: { submission_id: assessment.submission_id!, request_id: assessment.current_request_id!, submitted_at: assessment.submitted_at!, answer_count: 7, artifact_count: 1 },
    answers: [
      { field_id: "contact_email", label: "Security contact email", type: "EMAIL", required: true, visibility: "VISIBLE", value: { text: "security@acme.example" }, provenance: { source: "Vendor response" } },
      { field_id: "data_classes", label: "Bank information used", type: "MULTI_SELECT", required: true, visibility: "VISIBLE", value: { values: ["Customer personal data", "Payment data"] }, provenance: { source: "Vendor response" } },
      { field_id: "subprocessors", label: "Do subcontractors process bank information?", type: "YES_NO", required: true, visibility: "VISIBLE", value: { text: "Yes" }, provenance: { source: "Vendor response" } },
      { field_id: "security_framework", label: "Primary security framework", type: "SINGLE_SELECT", required: true, visibility: "VISIBLE", value: { text: "ISO 27001" }, provenance: { source: "Vendor response" } },
      { field_id: "subprocessor_details", label: "Subcontractor details", type: "LONG_TEXT", required: true, visibility: "VISIBLE", value: { text: "Payment-routing infrastructure is provided by a contracted hosting provider in the stated service scope." }, provenance: { source: "Vendor response" } },
    ],
    coverage: { visible_fields: 7, answered_fields: 7, required_fields: 4, answered_required: 4, ratio: 1 },
    documents: [{ field_id: "security_document", artifact_id: "artifact-vendor-iso27001", file_name: "acme-iso-27001-certificate.pdf", media_type: "application/pdf", size_bytes: 684_220, artifact_status: "AVAILABLE", evidence_class: "VENDOR_SUPPLIED", document_type: "ISO_27001_CERTIFICATE", reference: "ISO-27001-ACME-2026", issued_by: "Accredited certification body", issued_on: "2026-03-01", expires_on: "2027-03-01" }],
    provisional_score: { score: 82, coverage: 1, rule_results: [] },
    matters: [],
  };
}

const todayItems = [
  { id: "today-change", type: "REGULATORY_CHANGE", title: matter.title, why_now: "The source change is approved and two evidence sections still need owners before the internal review.", scope: "Nigeria Data Protection · Regulatory change", state: "Work in progress", evidence: "8 of 10 sections complete", owner: "Data Protection Office", due_at: future, primary_action: "Complete the evidence ownership update", action_target_type: "MATTER", action_target_id: matterID },
  { id: "today-evidence", type: "EVIDENCE_REQUEST", title: evidenceRequest.title, why_now: evidenceRequest.why_you, scope: "Annual privacy return · Evidence", state: "Response required", evidence: "Known facts prefilled", owner: "Privacy Operations", due_at: future, primary_action: "Provide the two missing details", action_target_type: "EVIDENCE_REQUEST", action_target_id: evidenceID },
  { id: "today-program", type: "PROGRAM", title: "Review the Nigeria Data Protection Programme", why_now: "The latest status is evidence incomplete, not current.", scope: "Nigeria · Privacy", state: "Evidence incomplete", evidence: "5 evidence checks", owner: "Data Protection Officer", due_at: future, primary_action: "Review status reasons", action_target_type: "PROGRAM", action_target_id: programID, intervention_class: "REVIEW", authority: { responsibility: "REVIEWER", materiality: 2 } },
];

let document: DocumentImport = {
  id: "document-gaid", tenant_id: "bank-demo", legal_entity_id: "bank-ng", file_name: "gaid-annual-return-notice.docx", media_type: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", purpose: "Assess updated annual return requirements", source_type: "REGULATORY", size_bytes: 186420, sha256: "c6c882f6f0a23c3b7aa213a7ddab2b62d2fa86289d5378a430da356228b551d8", storage_key: "document-imports/bank-demo/document-gaid/gaid-annual-return-notice.docx", artifact_status: "STORED_UNSCANNED", extraction_status: "EXTRACTED", extraction_method: "DOCX_XML_V1", analysis_status: "REVIEW_REQUIRED", analysis_method: "DETERMINISTIC_RULES_V1", limitations: ["This sample uses reference content; no customer or bank document is included.", "Review each proposal before use. The results are not legal advice or a compliance conclusion."], sections: [{ id: "section-return", sequence: 1, title: "Annual return requirements", text: "The institution must maintain an accountable owner and authoritative source for every required annual return section. The completed return shall be reviewed before filing." }], proposals: [{ id: "proposal-owner", kind: "REQUIREMENT_CANDIDATE", title: "Possible requirement", statement: "The institution must maintain an accountable owner and authoritative source for every required annual return section.", confidence: .86, anchor: { section_id: "section-return", quote: "The institution must maintain an accountable owner and authoritative source for every required annual return section." }, status: "PENDING_REVIEW" }], created_by: "role-cro", created_at: now, updated_at: now, version: 1,
};

let documentCoverage: DocumentCoverage = {
  id: "coverage-gaid", tenant_id: "bank-demo", legal_entity_id: "bank-ng", document_id: document.id, document_sha256: document.sha256,
  status: "READY", analyzer_version: "STRUCTURED_OBLIGATION_V1", matcher_version: "EXPLAINABLE_MATCHER_V1", scoring_policy_version: "COVERAGE_POLICY_V1",
  program_snapshot_hash: "static-program-snapshot-v1", version: 1, assessed_at: now, updated_at: now, next_cursor: "", limitations: ["Extracted obligations require human review before coverage is verified."],
  metrics: {
    estimated_verified: { numerator: 0, denominator: 2 }, verified: { numerator: 0, denominator: 2 },
    requirement_mapped: { numerator: 1, denominator: 2 }, control_implemented: { numerator: 1, denominator: 2 }, evidence_supported: { numerator: 0, denominator: 2 },
  },
  candidates: [{
    id: "coverage-owner", fingerprint: "gaid-owner", eligible: true,
    statement: "The institution must maintain an accountable owner and authoritative source for every required annual return section.",
    anchor: { section_id: "section-return", page: 3, quote: "The institution must maintain an accountable owner and authoritative source for every required annual return section." },
    modality: "MUST", actor: "the institution", action: "maintain", object: "an accountable owner and authoritative source", citations: ["GAID 2025 annual return"], dates: [], topics: ["governance", "records"], uncertainty: [], jurisdiction: "Nigeria", regulator: "Nigeria Data Protection Commission", program_type: "PRIVACY", classification: "MAPPED_NO_CURRENT_EVIDENCE",
    matches: [{
      id: "match-owner", program_id: programID, program_code: "NDPA", program_name: "Nigeria Data Protection Programme", program_version: 12,
      requirement_id: "req-1", requirement_code: "NDPA-RET-01", requirement_title: "Maintain accountable data-processing governance", requirement_version: 1,
      score: .89, band: "STRONG", rationale: "The obligation and approved requirement share the same accountable-owner, source-record and Nigerian privacy scope.", conflicts: [],
      components: [{ name: "SEMANTIC", weight: .35, score: .94, reason: "Accountable ownership and authoritative records align." }, { name: "SCOPE", weight: .3, score: 1, reason: "Both apply to the same Nigerian legal entity and privacy Program." }, { name: "MODALITY", weight: .2, score: 1, reason: "Both are mandatory obligations." }, { name: "CITATION", weight: .15, score: .55, reason: "The source families differ but address the same governance duty." }],
      coverage: { requirement_id: "req-1", applicability: "APPLICABLE", applicable: true, control_implemented: true, evidence_supported: false, complete: false, control_ids: ["control-1"], evidence_contract_ids: ["contract-training"], reasons: ["EVIDENCE_NOT_ASSESSED"] },
    }],
  }, {
    id: "coverage-review", fingerprint: "gaid-review", eligible: true,
    statement: "The completed return shall be reviewed before filing.", anchor: { section_id: "section-return", page: 3, quote: "The completed return shall be reviewed before filing." },
    modality: "MUST", actor: "the institution", action: "review", object: "the completed return", citations: ["GAID 2025 annual return"], dates: ["before filing"], topics: ["review", "filing"], uncertainty: [], jurisdiction: "Nigeria", regulator: "Nigeria Data Protection Commission", program_type: "PRIVACY", classification: "GAP", matches: [],
  }],
  suggestions: [{ id: "suggestion-review", candidate_id: "coverage-review", type: "ADD_REQUIREMENT", status: "PROPOSED", title: "Add the pre-filing review requirement", rationale: "The privacy Program is in scope, but no approved requirement captures this review-before-filing duty.", program_id: programID }],
  matters: [{ candidate_id: "coverage-owner", matter_id: matterID, reference: matter.reference, type: matter.type, status: matter.status, title: matter.title, summary: matter.summary, score: .82 }],
};

const todayGuide = { code: "executive-first-run", surface: "TODAY", profile: "executive", role: "Executive risk or compliance leader", version: 1, title: "Executive review", description: "Review priority work, Program status and supporting evidence.", illustration: "guided-orbit", steps: [
  { id: "brief", title: "Review priority work", description: "Today shows work assigned to you, due dates and data freshness.", action: "Open Today", view: "today", target: "today-brief" },
  { id: "attention", title: "Review a priority item", description: "Open the first Program, issue or evidence request in the queue.", action: "Review first item", view: "today", target: "attention-list", intent: "open-first-attention" },
  { id: "programs", title: "Check Program status", description: "Programs show status, requirements, controls, evidence and open issues.", action: "Open Programs", view: "programs", target: "programs-workspace" },
  { id: "finish", title: "Review status details", description: "Check the status reason, source, owner and next action.", action: "Done", view: "programs", target: "programs-workspace" },
] };

const vendorsGuide = { code: "vendor-operations-first-run", surface: "VENDORS", required_capability: "VENDORS", profile: "vendor-operations", role: "Vendor relationship owner", role_codes: ["BUSINESS_OWNER"], priority: 100, version: 1, title: "Manage vendor relationships", description: "Record the service, collect missing information and route vendor work for review.", illustration: "guided-orbit", steps: [
  { id: "register", title: "Review the vendor register", description: "Check the supplied service, owner and current relationship state.", action: "Review vendors", view: "vendors", target: "vendor-register" },
  { id: "due-diligence", title: "Collect due diligence", description: "Use known bank records first, then request only missing information.", action: "Open Vendors", view: "vendors", target: "vendors-workspace" },
  { id: "work", title: "Request vendor action", description: "Send a focused form, document, signature or upload request when the vendor must act.", action: "Open Vendors", view: "vendors", target: "vendors-workspace" },
  { id: "finish", title: "Confirm the outcome", description: "Completion and upload remain separate from review and outcome confirmation.", action: "Done", view: "vendors", target: "vendors-workspace" },
] };

export async function staticDemoRequest<T>(path: string, init?: RequestInit): Promise<T> {
  if (!staticDemoEnabled) throw new Error("Static demo transport is disabled");
  const url = new URL(path, "https://clearsight.demo");
  const pathname = url.pathname;
  const method = (init?.method ?? "GET").toUpperCase();
  const fixture = activeFixture();
  syncVendorWorkFixture(fixture);

  if (fixture === "today-loading" && pathname === "/api/v1/today") await delay(1800);
  if (fixture === "today-unavailable" && pathname === "/api/v1/today") throw new StaticDemoHTTPError(503, "today_unavailable", "Today's work is unavailable.");
  if (fixture === "evidence-requests-unavailable" && pathname === "/api/v1/evidence/requests") throw new StaticDemoHTTPError(503, "evidence_unavailable", "Evidence requests are temporarily unavailable.");
  if (fixture === "authority-forbidden" && pathname === "/api/v1/authority/resolve") throw new StaticDemoHTTPError(403, "permission_denied", "Authority inspection is restricted.");
  if (fixture === "capture-not-found" && pathname === `/api/v1/evidence/requests/${evidenceID}`) throw new StaticDemoHTTPError(404, "request_not_found", "The request is no longer available.");
  if (fixture === "capture-conflict" && pathname.includes(`/api/v1/evidence/requests/${evidenceID}/submissions`) && method === "POST") throw new StaticDemoHTTPError(409, "version_conflict", "The request changed while you were working.");

  if (pathname === "/api/v1/context") {
    const productionUnavailable = fixture === "today-unavailable";
    const noConfig = fixture === "no-config-access";
    return clone({ tenant: { id: "bank-demo", name: "Meridian Trust Bank" }, legal_entity: { id: "bank-ng", name: "Meridian Trust Bank Nigeria" }, actor: { id: "role-cro", name: "Chief Risk Officer", kind: "PERSON", role_codes: ["CRO", "EXECUTIVE"], assurance_level: "MFA", authentication: "STATIC_DEMO", session_id: "pages-demo" }, mode: "static-stakeholder-demo", demo_mode: !productionUnavailable, capabilities: { document_import: true, reference_journeys: !productionUnavailable, config_read: !noConfig, config_write: !noConfig, platform_operations_read: !noConfig, platform_operations_write: !noConfig } }) as T;
  }
  if (pathname === "/api/v1/onboarding/guide") {
    const surface = url.searchParams.get("surface")?.trim().toUpperCase() ?? "TODAY";
    if (surface === "VENDORS") return clone(vendorsGuide) as T;
    if (surface === "TODAY") return clone(todayGuide) as T;
    throw new StaticDemoHTTPError(404, "not_found", "Guide not found.");
  }
  if (pathname === "/api/v1/today") return clone({ items: fixture === "today-empty" ? [] : todayItems, generated_at: now }) as T;
  if (pathname === "/api/v1/compliance/readiness") return clone({ tenant_id: "bank-demo", status: "AT_RISK", baseline_known: false, generated_at: now, dimensions: { current: 0, aging: 1, at_risk: 1, unknown: 1, blocked_routing: 0, pending_human: 1 }, active_drifts: [{ id: "drift-1", subject_type: "PROGRAM", subject_id: programID, dimension: "EVIDENCE", severity: 4, summary: "Two annual-return evidence sections are incomplete.", required_action: "Assign owners and complete DPCO review.", detected_at: now }], recommended_actions: ["Complete the two missing evidence ownership records.", "Confirm the final DPCO review date."] }) as T;
  if (pathname === "/api/v1/program-summaries") return clone({ items: matches(url, programSummary.program.name, programSummary.program.code) ? [programSummary] : [], generated_at: now }) as T;
  if (pathname === "/api/v1/form-templates" && method === "GET") {
    if (fixture === "vendor-source-degraded") throw new StaticDemoHTTPError(503, "vendor_forms_unavailable", "Approved due-diligence forms could not be loaded.");
    return clone({ items: [vendorDueDiligenceForm], next_cursor: "" }) as T;
  }
  if (pathname === "/api/v1/vendor-links" && method === "GET") {
    const targetType = url.searchParams.get("target_type");
    const targetID = url.searchParams.get("target_id");
    return clone({ items: vendorWorkLinks.filter((link) => link.target_type === targetType && link.target_id === targetID), next_cursor: "" }) as T;
  }
  if (pathname === "/api/v1/vendor-work" && method === "GET") {
    const relationshipID = url.searchParams.get("relationship_id");
    const targetType = url.searchParams.get("target_type");
    const targetID = url.searchParams.get("target_id");
    const items = vendorWorkRequests.filter((work) => relationshipID ? work.relationship_id === relationshipID : work.target_type === targetType && work.target_id === targetID);
    return clone({ items, next_cursor: "" }) as T;
  }
  const prepareVendorWorkMatch = pathname.match(/^\/api\/v1\/vendors\/([^/]+)\/work\/prepare$/);
  if (prepareVendorWorkMatch && method === "POST") {
    const relationshipID = decodeURIComponent(prepareVendorWorkMatch[1]!);
    const input = parseBody(init) as { relationship_link_id?: string; purpose?: string; instructions?: string; form_template_id?: string; form_template_version?: number; presentation?: VendorWorkRequest["presentation"]; vendor_audience?: string; due_at?: string };
    const link = vendorWorkLinks.find((item) => item.id === input.relationship_link_id && item.relationship_id === relationshipID && item.state === "ACTIVE");
    if (!link || !input.purpose?.trim() || !input.instructions?.trim() || input.form_template_id !== vendorDueDiligenceForm.id || input.form_template_version !== vendorDueDiligenceForm.version || !input.vendor_audience?.includes("@") || !input.due_at) throw new StaticDemoHTTPError(422, "vendor_work_invalid", "Confirm the linked vendor, purpose, form, vendor contact and due date.");
    const work = { ...vendorWorkRecord(link, "PREPARING", "NOT_SENT", 1), purpose: input.purpose.trim(), instructions: input.instructions.trim(), presentation: input.presentation ?? "AUTOMATIC", due_at: input.due_at };
    vendorWorkRequests = [work, ...vendorWorkRequests.filter((item) => item.id !== work.id)];
    return clone(work) as T;
  }
  const vendorWorkResponseMatch = pathname.match(/^\/api\/v1\/vendors\/([^/]+)\/work\/([^/]+)\/response$/);
  if (vendorWorkResponseMatch && method === "GET") {
    const relationshipID = decodeURIComponent(vendorWorkResponseMatch[1]!);
    const workID = decodeURIComponent(vendorWorkResponseMatch[2]!);
    const work = vendorWorkRequests.find((item) => item.id === workID && item.relationship_id === relationshipID);
    if (!work?.submission_id || !work.current_request_id) throw new StaticDemoHTTPError(404, "vendor_work_response_not_found", "The submitted vendor response is not available for this request.");
    return clone(vendorWorkResponse(work)) as T;
  }
  const vendorWorkCommandMatch = pathname.match(/^\/api\/v1\/vendors\/([^/]+)\/work\/([^/]+)\/(send|retry|review\/start|changes|accept|cancel)$/);
  if (vendorWorkCommandMatch && method === "POST") {
    const relationshipID = decodeURIComponent(vendorWorkCommandMatch[1]!);
    const workID = decodeURIComponent(vendorWorkCommandMatch[2]!);
    const command = vendorWorkCommandMatch[3]!;
    const index = vendorWorkRequests.findIndex((item) => item.id === workID && item.relationship_id === relationshipID);
    if (index < 0) throw new StaticDemoHTTPError(404, "vendor_work_not_found", "The vendor request is not available in this legal entity.");
    const current = vendorWorkRequests[index]!;
    const input = parseBody(init) as { expected_version?: number; vendor_audience?: string; invitation_ttl_minutes?: number; message?: string; field_ids?: string[]; due_at?: string; rationale?: string; reason?: string };
    if (input.expected_version !== current.version) throw new StaticDemoHTTPError(409, "vendor_work_changed", "The vendor request changed before this action was recorded.");
    let updated: VendorWorkRequest;
    let outcome: VendorWorkSendOutcome | undefined;
    if (command === "send" || command === "retry") {
      if (!input.vendor_audience?.includes("@") || !input.invitation_ttl_minutes) throw new StaticDemoHTTPError(422, "vendor_work_invalid", "Enter a valid vendor contact and secure-link lifetime.");
      const partial = command === "send" && fixture === "vendor-work-partial-delivery";
      updated = { ...current, state: "AWAITING_VENDOR", delivery_state: partial ? "RETRY_REQUIRED" : "DELIVERED", recovery: partial ? "Email delivery was not confirmed. Retry delivery to issue a replacement secure link." : undefined, current_request_id: current.current_request_id ?? "vendor-work-capture-1", current_invitation_id: "vendor-work-invitation-replacement", current_capture_sequence: current.current_capture_sequence + 1, version: current.version + 1, updated_at: now };
      outcome = { work: updated, state: updated.delivery_state, delivery: { status: partial ? "FAILED" : "DELIVERED" }, recovery: updated.recovery, ...(partial ? { capture_url: "https://capture.example.test/?capture_invite=sample-vendor-work-recovery" } : {}) };
    } else if (command === "review/start") {
      if (current.state !== "RESPONSE_RECEIVED") throw new StaticDemoHTTPError(409, "vendor_work_action_unavailable", "A submitted response is required before review starts.");
      updated = { ...current, state: "UNDER_REVIEW", reviewer_principal_id: "role-cro", review_started_at: now, version: current.version + 1, updated_at: now };
    } else if (command === "changes") {
      if (current.state !== "UNDER_REVIEW" || !input.message?.trim() || !input.field_ids?.length || !input.vendor_audience?.includes("@") || !input.due_at) throw new StaticDemoHTTPError(422, "vendor_work_invalid", "Identify what the vendor must change, the affected fields, vendor contact and revised due date.");
      updated = { ...current, state: "CHANGES_REQUESTED", delivery_state: "DELIVERED", due_at: input.due_at, current_invitation_id: "vendor-work-invitation-changes", current_capture_sequence: current.current_capture_sequence + 1, version: current.version + 1, updated_at: now };
      outcome = { work: updated, state: "DELIVERED", delivery: { status: "DELIVERED" } };
    } else if (command === "accept") {
      if (current.state !== "UNDER_REVIEW" || !input.rationale?.trim()) throw new StaticDemoHTTPError(422, "vendor_work_invalid", "Record the basis for accepting this response.");
      updated = { ...current, state: "ACCEPTED", review_rationale: input.rationale.trim(), accepted_at: now, version: current.version + 1, updated_at: now };
    } else {
      if (!input.reason?.trim()) throw new StaticDemoHTTPError(422, "vendor_work_invalid", "Record why this vendor request is being cancelled.");
      updated = { ...current, state: "CANCELLED", cancellation_reason: input.reason.trim(), cancelled_at: now, version: current.version + 1, updated_at: now };
    }
    vendorWorkRequests = vendorWorkRequests.map((item, itemIndex) => itemIndex === index ? updated : item);
    return clone(outcome ?? updated) as T;
  }
  if (pathname === "/api/v1/vendors" && method === "GET") {
    const query = (url.searchParams.get("search") ?? "").trim().toLowerCase();
    const items = query ? vendorRelationships.filter((item) => `${item.vendor.legal_name} ${item.vendor.trading_name} ${item.relationship.service_name}`.toLowerCase().includes(query)) : vendorRelationships;
    return clone({ items, next_cursor: "" }) as T;
  }
  if (pathname === "/api/v1/vendors" && method === "POST") {
    const input = parseBody(init) as Record<string, string>;
    const created: VendorRelationshipAggregate = {
      vendor: { id: "vendor-static-new", tenant_id: "bank-demo", legal_name: input.legal_name ?? "Unnamed vendor", trading_name: input.trading_name ?? "", registration_ref: input.registration_ref ?? "", jurisdiction: input.jurisdiction ?? "", source_id: input.source_id ?? "", external_ref: input.external_ref ?? "", status: "ACTIVE", created_at: now, updated_at: now, version: 1 },
      relationship: { id: "vendor-relationship-static-new", tenant_id: "bank-demo", legal_entity_id: "bank-ng", vendor_id: "vendor-static-new", service_name: input.service_name ?? "Service not recorded", business_owner_principal_id: "role-cro", criticality: (input.criticality ?? "STANDARD") as VendorCriticality, privacy_role: (input.privacy_role ?? "NONE") as VendorPrivacyRole, status: "PROPOSED", effective_from: input.effective_from, renewal_at: input.renewal_at, source_id: input.source_id ?? "", external_ref: input.external_ref ?? "", created_at: now, updated_at: now, version: 1 },
    };
    vendorRelationships = [created, ...vendorRelationships];
    return clone(created) as T;
  }
  if (pathname === `/api/v1/vendors/${vendorRelationshipID}/assessments/current` && method === "GET") {
    const fixtureAssessment = fixtureVendorAssessment(fixture);
    if (fixtureAssessment && vendorAssessment?.status !== fixtureAssessment.status) vendorAssessment = fixtureAssessment;
    if (!vendorAssessment) throw new StaticDemoHTTPError(404, "vendor_assessment_not_found", "No due-diligence assessment has been started for this vendor relationship.");
    return clone({ assessment: vendorAssessment, setup: { assessment_id: vendorAssessment.id, state: "COMPLETED", attempts: 1, updated_at: vendorAssessment.updated_at } }) as T;
  }
  if (pathname === `/api/v1/vendors/${vendorRelationshipID}/assessments` && method === "POST") {
    const input = parseBody(init) as { relationship_version?: number; review_kind?: VendorAssessment["review_kind"]; source_trigger?: string; restart_assessment_id?: string; form_template_id?: string; form_template_version?: number; review_due_at?: string };
    if (input.relationship_version !== vendorRelationships[0]?.relationship.version) throw new StaticDemoHTTPError(409, "vendor_version_conflict", "The vendor relationship changed before due diligence was started.");
    if (input.form_template_id !== vendorDueDiligenceForm.id || input.form_template_version !== vendorDueDiligenceForm.version) throw new StaticDemoHTTPError(409, "vendor_assessment_form_inactive", "Select the current approved due-diligence form.");
    vendorAssessment = {
      id: "vendor-assessment-payments-2026",
      tenant_id: "bank-demo",
      legal_entity_id: "bank-ng",
      relationship_id: vendorRelationshipID,
      review_kind: input.review_kind ?? "ONBOARDING",
      source_trigger: input.restart_assessment_id ? `RESTART:${input.restart_assessment_id}` : input.source_trigger ?? "INITIAL",
      stable_episode_key: `${vendorRelationshipID}:${input.review_kind ?? "ONBOARDING"}:${input.restart_assessment_id ?? input.source_trigger ?? "INITIAL"}`,
      status: "READY_TO_SEND",
      form_template_id: vendorDueDiligenceForm.id,
      form_template_version: vendorDueDiligenceForm.version,
      review_matter_id: "matter-vendor-review-payments",
      review_due_at: input.review_due_at ?? future,
      started_by_principal_id: "role-cro",
      started_at: now,
      version: 2,
      created_at: now,
      updated_at: now,
    };
    return clone(vendorAssessment) as T;
  }
  if (vendorAssessment && pathname === `/api/v1/vendor-assessments/${vendorAssessment.id}/send-request` && method === "POST") {
    const input = parseBody(init) as { expected_version?: number; audience?: string; deadline?: string; invitation_ttl_minutes?: number };
    if (input.expected_version !== vendorAssessment.version) throw new StaticDemoHTTPError(409, "vendor_assessment_changed", "The due-diligence assessment changed before the request was sent.");
    if (!input.audience?.includes("@") || !input.deadline || !input.invitation_ttl_minutes) throw new StaticDemoHTTPError(422, "vendor_assessment_invalid", "Enter a valid vendor contact, response deadline and secure-link lifetime.");
    vendorAssessment = { ...vendorAssessment, status: "COLLECTING", current_request_id: "vendor-request-payments-2026", version: vendorAssessment.version + 1, updated_at: now };
    const outcome = {
      assessment: vendorAssessment,
      request: { id: vendorAssessment.current_request_id, status: "READY", deadline: input.deadline, updated_at: now },
      state: fixture === "vendor-partial-delivery" ? "LINK_CREATED_EMAIL_NOT_SENT" : "DELIVERED",
      delivery: fixture === "vendor-partial-delivery" ? { status: "FAILED", recipient_hint: maskEmail(input.audience), failure_code: "DELIVERY_UNAVAILABLE" } : { status: "DELIVERED", recipient_hint: maskEmail(input.audience), delivered_at: now },
      ...(fixture === "vendor-partial-delivery" ? { capture_url: "https://capture.example.test/?capture_invite=sample-recovery-token", recovery: "Copy the secure link or retry delivery." } : {}),
    };
    return clone(outcome) as T;
  }
  if (vendorAssessment && pathname === `/api/v1/vendor-assessments/${vendorAssessment.id}` && method === "GET") {
    if (!vendorAssessment.submission_id) throw new StaticDemoHTTPError(409, "vendor_assessment_action_unavailable", "A submitted vendor response is required before review.");
    return clone(submittedVendorReview(vendorAssessment)) as T;
  }
  if (vendorAssessment && pathname === `/api/v1/vendor-assessments/${vendorAssessment.id}/review/start` && method === "POST") {
    const input = parseBody(init) as { expected_version?: number };
    if (input.expected_version !== vendorAssessment.version) throw new StaticDemoHTTPError(409, "vendor_assessment_changed", "The assessment changed before the review was started.");
    if (vendorAssessment.status !== "SUBMITTED") throw new StaticDemoHTTPError(409, "vendor_assessment_action_unavailable", "The vendor response is not ready to enter review.");
    vendorAssessment = { ...vendorAssessment, status: "UNDER_REVIEW", reviewer_principal_id: "role-cro", review_started_at: now, version: vendorAssessment.version + 1, updated_at: now };
    return clone(vendorAssessment) as T;
  }
  if (vendorAssessment && pathname === `/api/v1/vendor-assessments/${vendorAssessment.id}/complete` && method === "POST") {
    const input = parseBody(init) as { expected_version?: number; conclusion?: VendorAssessment["conclusion"]; rationale?: string; uncertainty?: string; next_review_recommended_at?: string };
    if (input.expected_version !== vendorAssessment.version) throw new StaticDemoHTTPError(409, "vendor_assessment_changed", "The assessment changed before the conclusion was recorded.");
    if (vendorAssessment.status !== "UNDER_REVIEW") throw new StaticDemoHTTPError(409, "vendor_assessment_action_unavailable", "Start the vendor review before recording a conclusion.");
    if (!input.conclusion || !input.rationale?.trim()) throw new StaticDemoHTTPError(422, "vendor_assessment_invalid", "Select a conclusion and record its assessment basis.");
    vendorAssessment = { ...vendorAssessment, status: "COMPLETED", conclusion: input.conclusion, conclusion_rationale: input.rationale.trim(), conclusion_uncertainty: input.uncertainty?.trim(), next_review_recommended_at: input.next_review_recommended_at, completed_at: now, version: vendorAssessment.version + 1, updated_at: now };
    return clone(vendorAssessment) as T;
  }
  if (pathname.startsWith("/api/v1/vendors/") && method === "GET") {
    const id = decodeURIComponent(pathname.slice("/api/v1/vendors/".length));
    const found = vendorRelationships.find((item) => item.relationship.id === id);
    if (!found) throw new StaticDemoHTTPError(404, "vendor_not_found", "The vendor relationship is not available in this legal entity.");
    return clone(found) as T;
  }
  if (pathname.startsWith("/api/v1/vendors/") && method === "POST") {
    const id = decodeURIComponent(pathname.slice("/api/v1/vendors/".length));
    const index = vendorRelationships.findIndex((item) => item.relationship.id === id);
    if (index < 0) throw new StaticDemoHTTPError(404, "vendor_not_found", "The vendor relationship is not available in this legal entity.");
    const input = parseBody(init) as Record<string, string | number>;
    const current = vendorRelationships[index]!;
    if (input.expected_version !== current.relationship.version) throw new StaticDemoHTTPError(409, "vendor_version_conflict", "The vendor relationship changed while it was being edited.");
    const updated: VendorRelationshipAggregate = { vendor: { ...current.vendor, legal_name: String(input.legal_name ?? current.vendor.legal_name), trading_name: String(input.trading_name ?? ""), registration_ref: String(input.registration_ref ?? ""), jurisdiction: String(input.jurisdiction ?? ""), updated_at: now, version: current.vendor.version + 1 }, relationship: { ...current.relationship, service_name: String(input.service_name ?? current.relationship.service_name), criticality: String(input.criticality ?? current.relationship.criticality) as VendorCriticality, privacy_role: String(input.privacy_role ?? current.relationship.privacy_role) as VendorPrivacyRole, effective_from: input.effective_from ? String(input.effective_from) : undefined, renewal_at: input.renewal_at ? String(input.renewal_at) : undefined, updated_at: now, version: current.relationship.version + 1 } };
    vendorRelationships = vendorRelationships.map((item, itemIndex) => itemIndex === index ? updated : item);
    return clone(updated) as T;
  }
  if (pathname === `/api/v1/programs/${programID}`) return clone(programDetail) as T;
  if (pathname === `/api/v1/programs/${programID}/transition` && method === "POST") {
    const input = parseBody(init) as { expected_version?: number; to?: string; rationale?: string };
    if (input.expected_version !== program.version) throw new StaticDemoHTTPError(409, "version_conflict", "The Program changed before the status update was recorded.");
    const target = input.to ?? "";
    const transitions: Record<string, string[]> = { DRAFT: ["ACTIVE", "RETIRED"], ACTIVE: ["PAUSED", "RETIRED"], PAUSED: ["ACTIVE", "RETIRED"] };
    if (!(transitions[program.status] ?? []).includes(target)) throw new StaticDemoHTTPError(409, "transition_invalid", `The Program cannot move from ${program.status} to ${target || "an empty status"}.`);
    if (!input.rationale?.trim()) throw new StaticDemoHTTPError(400, "rationale_required", "A rationale is required for the Program status change.");
    program.status = target;
    program.version += 1;
    program.updated_at = now;
    programSummary.program_version = program.version;
    programSummary.projection_stale = true;
    return clone({ ...programDetail, program }) as T;
  }
  if (pathname === `/api/v1/programs/${programID}/review-digest` && method === "GET") return clone(programReviewDigest()) as T;
  if (pathname === `/api/v1/programs/${programID}/reviews` && method === "POST") {
    const input = parseBody(init) as { expected_program_version?: number; expected_projection_version?: number };
    if (input.expected_program_version !== 12 || input.expected_projection_version !== 3) throw new StaticDemoHTTPError(409, "version_conflict", "The Program changed while it was being reviewed.");
    programReviewAcknowledged = true;
    return clone(programReviewDigest()) as T;
  }
  if (pathname === "/api/v1/matter-summaries") return clone({ items: matches(url, matter.title, matter.reference) ? [matterSummary] : [], generated_at: now }) as T;
  if (pathname === `/api/v1/matters/${matterID}`) return clone(matterDetail) as T;
  if (pathname === "/api/v1/evidence/sources") return clone({ items: [{ id: "source-ndpc", tenant_id: "bank-demo", legal_entity_id: "bank-ng", code: "NDPC-PUBLICATIONS", name: "NDPC official publications", type: "REGULATORY", authority_class: "AUTHORITATIVE", expected_freshness_minutes: 1440, last_observed_at: now, last_success_at: now, health: "CURRENT", status: "ACTIVE", version: 3 }, { id: "source-iam", tenant_id: "bank-demo", legal_entity_id: "bank-ng", code: "IAM-ENTITLEMENTS", name: "Identity and access records", type: "SYSTEM", authority_class: "SYSTEM_OF_RECORD", expected_freshness_minutes: 60, last_observed_at: now, last_success_at: "2026-08-06T14:30:00Z", health: "DEGRADED", status: "ACTIVE", version: 8 }] }) as T;
  if (pathname === "/api/v1/evidence/requests") return clone({ items: [fixture === "capture-terminal" ? { ...evidenceRequest, status: "EXPIRED" } : fixture === "long-content" ? { ...evidenceRequest, title: "Confirm the accountable owner for the processor register covering the Nigeria annual-return process across retail, corporate, digital and delegated processing operations", purpose: "Confirm the smallest unresolved ownership fact while preserving the full legal-entity, filing-year, source and review context needed by the DPCO without requiring the respondent to reconstruct the wider compliance programme." } : evidenceRequest] }) as T;
  if (pathname === `/api/v1/evidence/requests/${evidenceID}`) return clone(fixture === "capture-terminal" ? { ...evidenceRequest, status: "EXPIRED" } : evidenceRequest) as T;
  if (pathname.includes(`/api/v1/evidence/requests/${evidenceID}/submissions`) && method === "POST") return clone({ request_id: evidenceID, status: "SUBMITTED", submitted_at: now }) as T;
  if (pathname === "/api/v1/authority/resolve") {
    const input = parseBody(init) as { responsibility?: string };
    const reviewer = input.responsibility === "REVIEWER";
    const principal = reviewer ? { id: "role-dpco", display_name: "Data Protection Compliance Officer", kind: "POSITION", role: "DPCO reviewer" } : { id: "role-cro", display_name: "Chief Risk Officer", kind: "POSITION", role: "CRO" };
    const candidates = reviewer ? [principal, { id: "role-deputy-dpco", display_name: "Deputy Data Protection Compliance Officer", kind: "POSITION", role: "Deputy DPCO reviewer" }] : [principal];
    return clone({ principal, candidate_principals: candidates, strategy: candidates.length > 1 ? "ANY_OF" : "SINGLE", rule_id: reviewer ? "rule-privacy-review" : "rule-critical-risk", policy_version: "risk-authority-v4", explanation: reviewer ? "The current privacy review policy permits either active independent DPCO reviewer for this Program." : "The current material authority policy resolves the CRO for this scoped decision." }) as T;
  }
  if (pathname === "/api/v1/authority/integrity") return clone({ findings: [] }) as T;
  if (pathname === "/api/v1/authority/policies") {
    if (fixture === "configure-partial") throw new StaticDemoHTTPError(503, "policy_unavailable", "Routing policies are temporarily unavailable.");
    return clone({ items: [{ id: "policy-risk", code: "RISK-AUTH", name: "Risk and compliance decision authority", status: "ACTIVE", version: 4, effective_from: "2026-07-01T00:00:00Z" }] }) as T;
  }
  if (pathname === "/api/v1/workflow/tasks") return clone({ items: [{ id: "task-dpco", tenant_id: "bank-demo", workflow_id: "workflow-return", step_key: "DPCO_REVIEW", responsibility: "REVIEWER", principal_id: "role-dpco", title: "Confirm the final DPCO review date", status: "READY", due_at: future, context: { program: "Nigeria Data Protection Programme" }, version: 1 }] }) as T;
  if (pathname === "/api/v1/operations/projections") return clone({ items: [{ tenant_id: "bank-demo", projection: "program_state", display_name: "Program status", state: "CURRENT", pending: 0, failed: 0, lag_seconds: 3, last_completed: now, updated_at: now }] }) as T;
  if (pathname === "/api/v1/operations/projections/reconcile") return clone({ tenant_id: "bank-demo", checked: 1, queued: 0, already_queued: 0, current: 1 }) as T;
  if (pathname === "/api/v1/compliance/automation-policies") return clone({ items: [] }) as T;
  if (pathname === "/api/v1/bank-journeys") return clone({ items: [], sample: true }) as T;
  if (pathname === "/api/v1/document-imports") return method === "POST" ? clone(document) as T : clone({ items: [document] }) as T;
  if (pathname === `/api/v1/document-imports/${document.id}/coverage` && method === "GET") return clone(documentCoverage) as T;
  if (pathname === `/api/v1/document-imports/${document.id}/coverage/review` && method === "POST") {
    const input = parseBody(init) as { expected_version?: number; decisions?: Array<{ candidate_id?: string; decision?: CoverageDecision; match_id?: string; reason?: string }> };
    if (input.expected_version !== documentCoverage.version) throw new StaticDemoHTTPError(409, "version_conflict", "The coverage assessment changed while it was being reviewed.");
    const decisions = input.decisions ?? [];
    documentCoverage = {
      ...documentCoverage, version: documentCoverage.version + 1, updated_at: now,
      candidates: documentCoverage.candidates.map((candidate) => {
        const decision = decisions.find((item) => item.candidate_id === candidate.id);
        return decision?.decision ? { ...candidate, classification: decision.decision === "NOT_APPLICABLE" ? "NOT_APPLICABLE" : candidate.classification, review: { decision: decision.decision, match_id: decision.match_id, reason: decision.reason, reviewer_id: "role-cro", reviewed_at: now } } : candidate;
      }),
    };
    return clone(documentCoverage) as T;
  }
  if (pathname === `/api/v1/document-imports/${document.id}/coverage/recompare` && method === "POST") return clone({ status: "QUEUED" }) as T;
  if (pathname.includes(`/api/v1/document-imports/${document.id}/coverage/suggestions/`) && pathname.endsWith("/apply") && method === "POST") {
    const input = parseBody(init) as { expected_version?: number };
    if (input.expected_version !== documentCoverage.version) throw new StaticDemoHTTPError(409, "version_conflict", "The coverage assessment changed while the recommendation was being applied.");
    documentCoverage = { ...documentCoverage, version: documentCoverage.version + 1, updated_at: now, suggestions: documentCoverage.suggestions.map((suggestion) => ({ ...suggestion, status: "APPLIED", applied_type: "REQUIREMENT", applied_id: "req-gaid-review-draft" })) };
    return clone({ assessment: documentCoverage, object_type: "REQUIREMENT", object_id: "req-gaid-review-draft" }) as T;
  }
  if (pathname === `/api/v1/document-imports/${document.id}`) return clone(document) as T;
  if (pathname.includes(`/api/v1/document-imports/${document.id}/proposals/`) && method === "POST") {
    const body = parseBody(init) as { status?: ProposalStatus; note?: string };
    document = { ...document, version: document.version + 1, updated_at: now, proposals: document.proposals.map((item) => ({ ...item, status: body.status ?? item.status, reviewed_by: "role-cro", reviewed_at: now, review_note: body.note })) };
    return clone(document) as T;
  }
  if (pathname === "/api/v1/onboarding/state") {
    const code = url.searchParams.get("guide_code") ?? "executive-first-run";
    const key = `clearsight-static-guide:${code}`;
    const fallback = { tenant_id: "bank-demo", principal_id: "role-cro", guide_code: code, guide_version: 1, current_step: 0, completed: false, dismissed: false, version: 0 };
    if (method === "PUT") {
      const input = parseBody(init) as Record<string, unknown>;
      const current = JSON.parse(localStorage.getItem(key) ?? JSON.stringify(fallback)) as Record<string, unknown>;
      const value = { ...current, current_step: input.current_step, completed: input.completed, dismissed: input.dismissed, version: Number(current.version ?? 0) + 1, updated_at: now };
      localStorage.setItem(key, JSON.stringify(value));
      return clone(value) as T;
    }
    return clone(JSON.parse(localStorage.getItem(key) ?? JSON.stringify(fallback))) as T;
  }
  throw new StaticDemoHTTPError(501, "fixture_not_implemented", `Static stakeholder demo does not implement ${method} ${pathname}`);
}

function activeFixture() { return typeof window === "undefined" ? "" : new URLSearchParams(window.location.search).get("fixture") ?? ""; }
function delay(ms: number) { return new Promise((resolve) => window.setTimeout(resolve, ms)); }
function matches(url: URL, ...values: string[]) { const query = (url.searchParams.get("q") ?? "").trim().toLowerCase(); const status = url.searchParams.get("status") ?? ""; if (status && ![program.status, matter.status, "OPEN"].includes(status)) return false; return !query || values.some((value) => value.toLowerCase().includes(query)); }
function parseBody(init?: RequestInit) { if (typeof init?.body !== "string") return {}; try { return JSON.parse(init.body) as unknown; } catch { return {}; } }
function maskEmail(value: string) { const [local, domain] = value.split("@"); return `${local?.slice(0, 1) || "*"}***@${domain || "vendor"}`; }
function clone<T>(value: T): T { return typeof structuredClone === "function" ? structuredClone(value) : JSON.parse(JSON.stringify(value)) as T; }
