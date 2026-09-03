export type LifecycleStatus = "DRAFT" | "PENDING_APPROVAL" | "ACTIVE" | "REJECTED" | "PAUSED" | "RETIRED";
export type RiskBand = "LOW" | "MODERATE" | "HIGH" | "CRITICAL" | "NOT_ASSESSED";

export type FormScoring = {
  id?: string;
  required?: boolean;
  weight: number;
  answer_scores: Record<string, number>;
  critical_answers?: string[];
};

export type FormTemplateField = {
  id: string;
  label: string;
  type: "text" | "short_text" | "long_text" | "date" | "number" | "single_select" | "photo" | "file" | "signature";
  required: boolean;
  description?: string;
  options?: string[];
  accepted_formats?: string[];
  scoring?: FormScoring;
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
  code: string;
  name: string;
  purpose: string;
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
