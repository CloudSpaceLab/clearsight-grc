import type { CaptureFieldConstraints, CapturePresentation, CaptureSection, CaptureVisibilityCondition } from "./types";

export type LifecycleStatus = "DRAFT" | "PENDING_APPROVAL" | "ACTIVE" | "REJECTED" | "PAUSED" | "RETIRED";
export type RiskBand = "LOW" | "MODERATE" | "HIGH" | "CRITICAL" | "NOT_ASSESSED";

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
};

export type CreateFormTemplateInput = {
  code: string;
  name: string;
  purpose: string;
  presentation: CapturePresentation;
  sections: CaptureSection[];
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
  presentation?: CapturePresentation;
  sections?: CaptureSection[];
  fields: FormTemplateField[];
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
  binding_id?: string;
  binding_version?: number;
  thresholds: { moderate_from: number; high_from: number; critical_from: number };
  freshness_minutes: number;
  minimum_coverage: number;
  owner_principal_id?: string;
  reviewer_principal_id?: string;
  failure_action: "REVIEW" | "RECOMMEND_MATTER";
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
