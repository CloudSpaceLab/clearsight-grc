export type ProcessingActivityStatus = "NEW" | "OPEN" | "CLOSED";

export type DataCategory = {
  category: string;
  sensitivity: "UNCLASSIFIED" | "DIRECT_PERSONAL" | "INDIRECT_PERSONAL" | "SENSITIVE_BY_NATURE" | "SENSITIVE_BY_LAW";
};

export type TransferBasis =
  | "ADEQUACY"
  | "APPROVED_INSTRUMENT"
  | "RECOGNISED_LAWFUL_BASIS"
  | "CONSENT"
  | "STANDARD_CONTRACT_CLAUSES"
  | "BINDING_CORPORATE_RULES"
  | "CERTIFICATION"
  | "NOT_APPLICABLE";

export type Recipient = {
  recipient: string;
  recipient_kind: "INTERNAL" | "EXTERNAL" | "AUTHORITY";
  country_code?: string;
  is_cross_border: boolean;
  transfer_basis: TransferBasis;
};

export type System = {
  system_name: string;
  system_kind: "APPLICATION" | "DATABASE" | "FILE" | "MANUAL" | "THIRD_PARTY";
};

export type Review = {
  id: string;
  created_at: string;
  due_date: string;
  completed_at?: string;
  outcome?: "CONFIRMED" | "REVISED" | "WITHDRAWN";
  reviewer_principal_id?: string;
};

export type ProcessingActivity = {
  id: string;
  tenant_id: string;
  legal_entity_id: string;
  code: string;
  name: string;
  description: string;
  status: ProcessingActivityStatus;
  purpose: string;
  lawful_basis: string;
  controller: string;
  processor: string;
  automated_decision_making: boolean;
  data_subject_categories: string;
  personal_data_categories: string;
  security_measures: string;
  retention_period: string;
  start_date?: string;
  end_date?: string;
  next_review_date?: string;
  owner_principal_id?: string;
  required_authority_principal_id?: string;
  program_id?: string;
  version: number;
  created_at: string;
  updated_at: string;
  data_categories?: DataCategory[];
  recipients?: Recipient[];
  systems?: System[];
  reviews?: Review[];
};

export type RegisterCounts = {
  total: number;
  new: number;
  open: number;
  closed: number;
  review_overdue: number;
  missing_lawful_basis: number;
  missing_owner: number;
  no_data_subjects: number;
  retired: number;
};

export type Coverage = {
  population: number;
  excluded?: number | null;
  unknown?: number | null;
};

export type RegisterSummary = {
  generated_at: string;
  projection_version: string;
  freshness: "CURRENT" | "STALE";
  source_high_water: string;
  coverage: Coverage;
  counts: RegisterCounts;
};

export type ProcessingActivityEvent = {
  id: string;
  tenant_id: string;
  legal_entity_id: string;
  aggregate_type: "PROCESSING_ACTIVITY";
  aggregate_id: string;
  aggregate_version: number;
  type: string;
  payload: Record<string, unknown>;
  actor_type: string;
  actor_id?: string;
  occurred_at: string;
};

export type ProcessingActivityHistoryResponse = {
  events: ProcessingActivityEvent[];
  has_more: boolean;
};

// The register client keeps a small compatibility shape for older responses
// while the current ActivityPage contract is stable snake_case JSON.
export type ProcessingActivityPageWire = {
  rows?: ProcessingActivity[];
  next_cursor?: string;
  has_more?: boolean;
  Rows?: ProcessingActivity[];
  NextCursor?: string;
  HasMore?: boolean;
};

export type ProcessingActivityPage = {
  rows: ProcessingActivity[];
  next_cursor?: string;
  has_more: boolean;
};

export type ActivityPage = ProcessingActivityPage;

export type ProcessingActivityResponse = {
  state_label: string;
  activity: ProcessingActivity;
  closure_blockers?: string[];
};
