export type VendorAssessmentStatus =
  | "SETUP_PENDING"
  | "READY_TO_SEND"
  | "COLLECTING"
  | "SUBMITTED"
  | "UNDER_REVIEW"
  | "COMPLETED"
  | "CANCELLED";

export type VendorAssessmentConclusion =
  | "SATISFACTORY"
  | "SATISFACTORY_WITH_CONDITIONS"
  | "UNSATISFACTORY"
  | "INCONCLUSIVE";

export type VendorAssessment = {
  id: string;
  tenant_id: string;
  legal_entity_id: string;
  relationship_id: string;
  review_kind: "ONBOARDING";
  stable_episode_key: string;
  status: VendorAssessmentStatus;
  form_template_id: string;
  form_template_version: number;
  current_request_id?: string;
  submission_id?: string;
  review_matter_id?: string;
  review_due_at: string;
  started_by_principal_id: string;
  started_at: string;
  submitted_at?: string;
  review_started_at?: string;
  completed_at?: string;
  reviewer_principal_id?: string;
  conclusion?: VendorAssessmentConclusion;
  conclusion_uncertainty?: string;
  conclusion_rationale?: string;
  next_review_recommended_at?: string;
  cancellation_reason?: string;
  version: number;
  created_at: string;
  updated_at: string;
};

export type VendorAssessmentFormOption = {
  id: string;
  version: number;
  name: string;
  presentation: "CLASSIC" | "WIZARD" | "AUTOMATIC";
};

export type StartVendorAssessmentInput = {
  relationship_version: number;
  form_template_id: string;
  form_template_version: number;
  review_due_at: string;
};

export type SendVendorAssessmentRequestInput = {
  expected_version: number;
  audience: string;
  deadline: string;
  invitation_ttl_minutes: number;
};

export type VendorAssessmentRequestSummary = {
  id: string;
  status: string;
  deadline?: string;
  updated_at?: string;
};

export type VendorAssessmentSendState =
  | "REQUEST_READY_INVITATION_NOT_ISSUED"
  | "LINK_CREATED_EMAIL_NOT_SENT"
  | "DELIVERED";

export type VendorAssessmentSendOutcome = {
  assessment: VendorAssessment;
  request: VendorAssessmentRequestSummary;
  state: VendorAssessmentSendState;
  recovery?: string;
  capture_url?: string;
  delivery?: {
    status: string;
    recipient_hint?: string;
    delivered_at?: string;
    failure_code?: string;
  };
};

export type VendorAssessmentSetupStatus = {
  assessment_id?: string;
  state: "READY" | "LEASED" | "COMPLETED" | "FAILED" | string;
  attempts?: number;
  next_attempt_at?: string;
  lease_until?: string;
  terminal_at?: string;
  failure_code?: string;
  updated_at?: string;
};

export type CurrentVendorAssessment = {
  assessment: VendorAssessment | null;
  setup?: VendorAssessmentSetupStatus;
};

export type VendorAssessmentResponseSummary = {
  request_id: string;
  submitted_at: string;
  answer_count: number;
  artifact_count: number;
  provisional_score?: number;
  provisional_band?: string;
};

export type VendorAssessmentDocument = {
  artifact_id: string;
  file_name: string;
  status: "SUBMITTED" | "VALIDATED" | "REJECTED" | "EXPIRED" | string;
  evidence_class?: "VENDOR_SUPPLIED" | "BANK_VALIDATED" | "OFFICIAL_SOURCE";
  valid_until?: string;
};

export type VendorAssessmentFinding = {
  matter_id: string;
  title: string;
  state: string;
};

export type VendorAssessmentReviewView = {
  assessment: VendorAssessment;
  response?: VendorAssessmentResponseSummary;
  documents: VendorAssessmentDocument[];
  findings: VendorAssessmentFinding[];
};

export type StartVendorAssessmentReviewInput = { expected_version: number };

export type VendorAssessmentClarificationInput = {
  expected_version: number;
  request_fields: string[];
  message: string;
  deadline: string;
};

export type CompleteVendorAssessmentInput = {
  expected_version: number;
  conclusion: VendorAssessmentConclusion;
  rationale: string;
  uncertainty?: string;
  next_review_recommended_at?: string;
};

export type ReviewVendorAssessmentDocumentInput = {
  expected_version: number;
  decision: "VALIDATE" | "REJECT";
  document_type: string;
  evidence_class: "VENDOR_SUPPLIED" | "BANK_VALIDATED" | "OFFICIAL_SOURCE";
  valid_until?: string;
};

export type CreateVendorAssessmentDeficiencyInput = {
  expected_version: number;
  title: string;
  summary: string;
  due_at?: string;
};
