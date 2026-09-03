import type { CaptureFieldConstraints, CapturePresentation, CaptureSection, CaptureVisibilityCondition } from "./types";

export type LifecycleStatus = "DRAFT" | "PENDING_APPROVAL" | "ACTIVE" | "REJECTED" | "PAUSED" | "RETIRED";
export type RiskBand = "LOW" | "MODERATE" | "HIGH" | "CRITICAL" | "NOT_ASSESSED";
export type FormScoringMode = "NONE" | "RISK" | "COMPLIANCE";
export type FormScoreDirection = "HIGH_IS_POOR" | "LOW_IS_POOR";
export type FormConcernBand = "LOW" | "MODERATE" | "HIGH" | "CRITICAL";
export type FormScoreMissing = "INDETERMINATE" | "EXCLUDE" | "ZERO";
export type FormScorePredicateOperator = "EQUALS" | "NOT_EQUALS" | "IN" | "NOT_IN" | "CONTAINS" | "CONTAINS_ANY" | "CONTAINS_ALL" | "GREATER_THAN" | "GREATER_OR_EQUAL" | "LESS_THAN" | "LESS_OR_EQUAL" | "NUMBER_BETWEEN" | "DATE_BEFORE" | "DATE_ON_OR_AFTER" | "DATE_BETWEEN" | "ANSWERED" | "UNANSWERED" | "AND" | "OR" | "NOT";
export type FormScorePredicate = { field_id?: string; operator: FormScorePredicateOperator; values?: string[]; children?: FormScorePredicate[] };
export type FormScoreContribution = { id: string; label: string; weight: number; predicate: FormScorePredicate; match_points: number; non_match_points: number; missing: FormScoreMissing; required?: boolean };
export type FormScoreRuleEffect = { kind: "CONTRIBUTION" | "FLOOR" | "CAP" | "DISQUALIFY"; value?: number; weight?: number };
export type FormScoreRule = { id: string; label: string; predicate: FormScorePredicate; effect: FormScoreRuleEffect };
export type FormScoreBandRange = { band: FormConcernBand; from: number; through: number };
export type FormScoreProfile = { version: string; mode: Exclude<FormScoringMode, "NONE">; direction: FormScoreDirection; contributions: FormScoreContribution[]; rules?: FormScoreRule[]; bands: FormScoreBandRange[] };
export type FormCollectionIntent = "CAPTURE" | "CONFIRM_OR_CORRECT" | "REPLACE_HELD_DOCUMENT";
export type FormBrowserCachePolicy = "ALLOWED" | "NO_BROWSER_CACHE";
export type FormRecordTarget = { key: string; required_subject_type: string };

export type FormFieldType =
  | "short_text" | "long_text" | "email" | "telephone" | "url"
  | "integer" | "decimal" | "percentage" | "currency" | "date"
  | "yes_no" | "single_select" | "multi_select" | "checkbox" | "attestation"
  | "file" | "photo" | "signature" | "vendor_document";

export type FormScoring = {
  id?: string;
  required?: boolean;
  weight: number;
  answer_scores: Record<string, number>;
  critical_answers?: string[];
};

export type FormTemplateSection = CaptureSection & {
  weight?: number;
  condition?: CaptureVisibilityCondition;
};

export type FormTemplateField = {
  id: string;
  section_id?: string;
  label: string;
  type: FormFieldType | "text" | "number";
  required: boolean;
  description?: string;
  options?: string[];
  accepted_formats?: string[];
  attestation?: string;
  constraints?: CaptureFieldConstraints;
  condition?: CaptureVisibilityCondition;
  scoring?: FormScoring;
  collection_intent?: FormCollectionIntent;
  record_target?: FormRecordTarget;
  browser_cache_policy?: FormBrowserCachePolicy;
};

export type CreateFormTemplateInput = {
  code: string;
  name: string;
  purpose: string;
  scoring_mode?: FormScoringMode;
  score_profile?: FormScoreProfile;
  presentation: CapturePresentation;
  sections: FormTemplateSection[];
  fields: FormTemplateField[];
};

export type Lifecycle = {
  status: LifecycleStatus;
  is_current: boolean;
  version: number;
  created_by?: string;
  submitted_by?: string;
  approved_by?: string;
  rejected_by?: string;
  effective_from?: string;
  effective_until?: string;
  created_at: string;
  updated_at: string;
};

export type FormTemplate = Lifecycle & {
  id: string;
  tenant_id: string;
  legal_entity_id?: string;
  program_id?: string;
  code: string;
  name: string;
  purpose: string;
  scoring_mode?: FormScoringMode;
  score_profile?: FormScoreProfile;
  presentation?: CapturePresentation;
  sections?: FormTemplateSection[];
  fields: FormTemplateField[];
};

export type CollectionPolicy = {
  validity_months: number;
  renewal_window_days: number;
  reminder_count: number;
};

export type MonitoringCheck = Lifecycle & {
  id: string;
  tenant_id: string;
  program_id: string;
  code: string;
  name: string;
  claim: string;
  input_kind: "FORM" | "SOURCE";
  form_template_id?: string;
  form_template_version?: number;
  collection_policy?: CollectionPolicy;
  binding_id?: string;
  binding_version?: number;
  thresholds: { moderate_from: number; high_from: number; critical_from: number };
  freshness_minutes: number;
  minimum_coverage: number;
  owner_principal_id?: string;
  reviewer_principal_id?: string;
  failure_action: "REVIEW" | "RECOMMEND_MATTER";
};

export type CollectionCurrencyState = "NO_RESPONSE_SUBMITTED" | "CURRENT" | "RENEWAL_DUE" | "RESPONSE_POTENTIALLY_EXPIRED" | "AWAITING_RESPONSE" | "RENEWAL_BLOCKED";

export type CollectionSummary = {
  monitoring_check_id: string;
  latest_request_id?: string;
  latest_submission_id?: string;
  latest_submission_at?: string;
  respondent_label?: string;
  recipient_hint?: string;
  expires_at: string;
  renewal_opens_at: string;
  currency_state: CollectionCurrencyState;
  active_request_deadline?: string;
  reminders_sent: number;
  reminder_count: number;
  delivery_state: "NOT_DISPATCHED" | "ASSIGNED" | "DELIVERED" | "BLOCKED" | "FAILED";
  last_error_safe?: string;
  projection_generated_at: string;
  projection_source_version: number;
};

export type MonitoringResult = {
  id: string;
  monitoring_check_id: string;
  monitoring_check_version: number;
  evaluated_at: string;
  evaluation: {
    score?: number;
    band: RiskBand;
    coverage: number;
    critical_failures?: Array<{ rule_id?: string; field_id: string; outcome: "PASS" | "FAIL" | "INDETERMINATE"; points: number; critical?: boolean; reason: string }>;
    rule_results?: Array<{ rule_id?: string; field_id: string; outcome: "PASS" | "FAIL" | "INDETERMINATE"; points: number; critical?: boolean; reason: string }>;
  };
};
