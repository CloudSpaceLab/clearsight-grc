export type ReportDefinitionStatus = "DRAFT" | "PENDING_REVIEW" | "REVIEWED" | "ACTIVE" | "RETIRED";
export type ReportRunStatus = "QUEUED" | "RUNNING" | "READY" | "FAILED";
export type ReportDataset = "PROCESSING_ACTIVITIES" | "PROCESSING_ACTIVITY_EXCEPTIONS" | "PROGRAMS" | "MATTER_EXCEPTIONS" | "VENDORS";
export type ReportScopeKind = "LEGAL_ENTITY" | "PROGRAM" | "MATTER";
export type ReportFormat = "CSV" | "NDJSON" | "XLSX";
export type ReportDefinitionAction = "submit" | "review" | "activate" | "reject" | "retire";

export type ReportFilterExpression = {
  kind: "group" | "condition" | string;
  field?: string;
  operator: "and" | "or" | "is" | "is_not" | "contains" | string;
  value?: string;
  children?: ReportFilterExpression[];
};

export type ReportFilterFieldDefinition = {
  field: string;
  label: string;
  dataset: ReportDataset;
  operators: string[];
  indexed: boolean;
};

export type ReportFilterFieldResponse = {
  fields: ReportFilterFieldDefinition[];
};

export type ReportDefinition = {
  id: string;
  tenant_id: string;
  legal_entity_id: string;
  code: string;
  name: string;
  description: string;
  dataset: ReportDataset;
  scope_kind: ReportScopeKind;
  scope_ref?: string;
  format: ReportFormat;
  filter?: ReportFilterExpression;
  status: ReportDefinitionStatus;
  current_version: number;
  effective: boolean;
  checksum: string;
  maker_id: string;
  checker_id?: string;
  reviewer_id?: string;
  reviewer_note?: string;
  effective_from?: string;
  effective_until?: string;
  submitted_at?: string;
  approved_at?: string;
  retired_at?: string;
  created_at: string;
  updated_at: string;
  version: number;
};

export type ReportDefinitionRevision = {
  definition_id: string;
  tenant_id: string;
  legal_entity_id: string;
  version: number;
  base_version: number;
  dataset: ReportDataset;
  scope_kind: ReportScopeKind;
  scope_ref?: string;
  format: ReportFormat;
  filter: ReportFilterExpression;
  checksum: string;
  maker_id: string;
  created_at: string;
  reviewed_by?: string;
  reviewed_at?: string;
  approved_by?: string;
  approved_at?: string;
  decision: string;
  decision_note: string;
};

export type ReportSourceBoundary = {
  captured_at: string;
  projection_version: string;
  source_high_water: Record<string, string>;
  population: number;
  population_complete: boolean;
};

export type ReportRun = {
  id: string;
  tenant_id: string;
  legal_entity_id: string;
  definition_id: string;
  definition_version: number;
  definition_code: string;
  definition_checksum: string;
  scope_kind: ReportScopeKind;
  scope_ref?: string;
  requested_by_ref: string;
  as_of: string;
  filter: ReportFilterExpression;
  dataset: ReportDataset;
  format: ReportFormat;
  status: ReportRunStatus;
  attempt_count: number;
  row_count: number;
  data_object_key?: string;
  data_sha256?: string;
  manifest_object_key?: string;
  manifest_sha256?: string;
  failure_code?: string;
  created_at: string;
  completed_at?: string;
  expires_at: string;
  source_boundary: ReportSourceBoundary;
};

export type ReportDefinitionInput = {
  code: string;
  name: string;
  description?: string;
  dataset: ReportDataset;
  scope_kind: ReportScopeKind;
  scope_ref?: string;
  format: ReportFormat;
  filter?: ReportFilterExpression;
  effective_from?: string;
  effective_until?: string;
};

export type ReportDefinitionTransitionInput = {
  expected_version: number;
  checksum_seen: string;
  note?: string;
  effective_from?: string;
};

export type ReportRunInput = {
  definition_id: string;
  expected_definition_version: number;
};
