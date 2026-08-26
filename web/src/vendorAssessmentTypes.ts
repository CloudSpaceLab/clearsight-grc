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

export type ReissueVendorAssessmentRequestInput = {
  expected_version: number;
  audience: string;
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

export type RetryVendorAssessmentSetupInput = { expected_version: number };

export type CancelVendorAssessmentInput = {
  expected_version: number;
  reason: string;
};

export type VendorAssessmentSetupRetryOutcome = {
  assessment: VendorAssessment;
  setup: VendorAssessmentSetupStatus;
};

export type VendorAssessmentResponseSummary = {
  submission_id: string;
  request_id: string;
  submitted_at: string;
  answer_count: number;
  artifact_count: number;
};

export type VendorAssessmentReviewRequest = {
  request_id: string;
  purpose: string;
  sequence: number;
  origin_sequence: number;
  status: string;
  deadline: string;
  form_template_id: string;
  form_template_version: number;
};

export type VendorAssessmentAnswerValue = {
  text?: string;
  values?: string[];
  artifact_ids?: string[];
  document?: {
    artifact_id: string;
    document_type: string;
    reference?: string;
    issued_by?: string;
    issued_on?: string;
    expires_on?: string;
  };
};

export type VendorAssessmentReviewAnswer = {
  field_id: string;
  label: string;
  type: string;
  required: boolean;
  visibility: "VISIBLE" | "CONDITIONALLY_OMITTED";
  value?: VendorAssessmentAnswerValue;
  provenance?: {
    origin?: "SOURCE_PREFILLED" | "RESPONDENT_ENTERED" | "RESPONDENT_CORRECTED" | string;
    source?: string;
    binding_id?: string;
    binding_version?: number;
    source_value?: { kind: string; text?: string };
    source_receipt?: {
      source_id: string;
      observed_at: string;
      [key: string]: unknown;
    };
    validations?: {
      state?: string;
      binding_name?: string;
      source_id?: string;
      failure_code?: string;
      receipt?: { observed_at?: string; [key: string]: unknown };
      [key: string]: unknown;
    }[];
  };
};

export type VendorAssessmentReviewCoverage = {
  visible_fields: number;
  answered_fields: number;
  required_fields: number;
  answered_required: number;
  ratio: number;
};

export type VendorAssessmentDocument = {
  field_id: string;
  artifact_id: string;
  file_name: string;
  media_type: string;
  size_bytes: number;
  artifact_status: string;
  status?: "SUBMITTED" | "VALIDATED" | "REJECTED" | "EXPIRED" | string;
  evidence_class: "VENDOR_SUPPLIED" | "BANK_VALIDATED" | "OFFICIAL_SOURCE" | string;
  document_type: string;
  reference?: string;
  issued_by?: string;
  issued_on?: string;
  expires_on?: string;
};

export type VendorAssessmentFinding = {
  matter_id: string;
  type: string;
  title: string;
  status: string;
};

export type VendorAssessmentProvisionalScore = {
  score?: number;
  coverage: number;
  critical_failures?: { field_id: string; outcome: string; points: number; critical?: boolean }[];
  rule_results: { field_id: string; outcome: string; points: number; critical?: boolean }[];
};

export type VendorAssessmentReviewView = {
  assessment: VendorAssessment;
  requests: VendorAssessmentReviewRequest[];
  response?: VendorAssessmentResponseSummary;
  answers: VendorAssessmentReviewAnswer[];
  coverage: VendorAssessmentReviewCoverage;
  documents: VendorAssessmentDocument[];
  provisional_score?: VendorAssessmentProvisionalScore;
  matters: VendorAssessmentFinding[];
};

export type StartVendorAssessmentReviewInput = { expected_version: number };

export type VendorAssessmentClarificationInput = {
  expected_version: number;
  request_fields: string[];
  message: string;
  audience: string;
  deadline: string;
  invitation_ttl_minutes: number;
};

export type VendorAssessmentClarificationOutcome = {
  assessment: VendorAssessment;
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
  trigger_key: string;
  title: string;
  summary: string;
  due_at: string;
};

export type VendorAssessmentDeficiencyOutcome = {
  assessment: VendorAssessment;
  matter: {
    type_label?: string;
    status_label?: string;
    next_action?: string;
    matter: {
      id: string;
      reference: string;
      type?: string;
      status: string;
      title: string;
      summary?: string;
      due_at?: string;
      version?: number;
    };
  };
};
