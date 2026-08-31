import type { CapturePresentation } from "./types";
import type { CreateFormTemplateInput, FormScoringMode, FormTemplate as MonitoringFormTemplate, FormTemplateField, FormTemplateSection, LifecycleStatus } from "./monitoringTypes";
import type { DocumentSourceAnchor } from "./documentTypes";

export type { FormScoringMode } from "./monitoringTypes";

export type FormTemplate = MonitoringFormTemplate & {
  owner_principal_id?: string;
  responsible_team?: string;
  approved_uses?: string[];
  tags?: string[];
  jurisdiction?: string;
  industry?: string;
  sensitivity: string;
  scoring_mode: FormScoringMode;
  next_review_at?: string;
  starter_catalog_code?: string;
  starter_catalog_version?: number;
  presentation: CapturePresentation;
  sections: FormTemplateSection[];
};

export type FormLibraryOperation = {
  command: "forms.template.revise" | "forms.template.transition";
  label: string;
  responsibility: string;
  can_act: boolean;
  reason: string;
  allowed_targets?: LifecycleStatus[];
};

export type FormLibraryItem = {
  template: FormTemplate;
  active_version?: number;
  active_status?: LifecycleStatus;
  authority_available?: boolean;
  operations?: FormLibraryOperation[];
};

export type FormFilterField = "status" | "owner" | "program" | "use" | "tag";
export type FormFilterCondition = {
  kind: "condition";
  field: FormFilterField;
  operator: "is";
  value: string;
};
export type FormFilterGroup = {
  kind: "group";
  operator: "and" | "or";
  children: FormFilterExpression[];
};
export type FormFilterExpression = FormFilterCondition | FormFilterGroup;
export type FormLibraryFacets = { status?: Partial<Record<LifecycleStatus, number>> };
export type FormTemplatePage = {
  items: FormLibraryItem[];
  next_cursor?: string;
  total?: number;
  facets?: FormLibraryFacets;
};
export type ReusableFormTemplateRef = { id: string; name: string; code: string; version: number };

export type FormTemplateQuery = {
  search?: string; status?: LifecycleStatus; owner?: string; program?: string;
  use?: string; tag?: string; filter?: FormFilterExpression; sort?: "UPDATED_DESC" | "UPDATED_ASC"; cursor?: string; limit?: number;
};

export type SavedFormViewFilter = {
  search?: string; status?: LifecycleStatus; owner_principal_id?: string; program_id?: string;
  use?: string; tag?: string; expression?: FormFilterExpression; limit?: number;
  sort?: "UPDATED_DESC" | "UPDATED_ASC";
};
export type SavedFormView = { id: string; name: string; filter: SavedFormViewFilter; created_at: string; updated_at: string };

export type StarterTemplate = {
  code: string; catalog_version: number; published_on: string; reference_label: string; template: FormTemplate;
};

export type CreateLibraryFormInput = CreateFormTemplateInput & {
  program_id?: string; owner_principal_id?: string; responsible_team?: string; approved_uses?: string[];
  tags?: string[]; jurisdiction?: string; industry?: string; sensitivity?: string; scoring_mode?: FormScoringMode; next_review_at?: string;
};

export type InstantiateStarterTemplateInput = {
  code?: string; name?: string; purpose?: string; program_id?: string; owner_principal_id?: string;
  responsible_team?: string; jurisdiction?: string; industry?: string; next_review_at?: string;
};

export type FormProposalStatus = "GENERATING" | "REVIEW_REQUIRED" | "ACCEPTED" | "REJECTED" | "FAILED";
export type FormProposalSourceKind = "DOCUMENT" | "AI";
export type FormProposalChangeKind = "ADD_FIELD" | "UPDATE_FIELD" | "REMOVE_FIELD";
export type FormProposalContract = {
  scoring_mode: FormScoringMode;
  presentation: CapturePresentation;
  sections: FormTemplateSection[];
  fields: FormTemplateField[];
};
export type FormProposalFieldChange = {
  id: string;
  kind: FormProposalChangeKind;
  field: FormTemplateField;
  anchor: DocumentSourceAnchor;
  confidence: number;
  unresolved?: string[];
};
export type FormProposalUnresolvedItem = {
  code: string;
  message: string;
  field_change_id?: string;
  anchor?: DocumentSourceAnchor;
};
export type FormAIProvenance = {
  workload_id: string;
  policy_ref?: string;
  gateway_request_id?: string;
  gateway_response_id?: string;
  route_id?: string;
  model_alias: string;
  prompt_version: string;
  snapshot_sha256: string;
  source_document_sha256?: string;
  source_element_refs?: string[];
  validation_results: string[];
};
export type FormProposalProvenance = {
  proposal_version: string;
  source_document_id: string;
  source_sha256: string;
  source_version: number;
  parser_version?: string;
  adapter_version?: string;
  extraction_status: string;
  tabular_parser_version?: string;
  ai?: FormAIProvenance;
};
export type FormTemplateProposal = {
  id: string;
  source_kind: FormProposalSourceKind;
  source_document_id?: string;
  source_document_version?: number;
  source_sha256?: string;
  base_template_id?: string;
  base_template_version?: number;
  status: FormProposalStatus;
  proposed_contract: FormProposalContract;
  field_changes: FormProposalFieldChange[];
  unresolved_items: FormProposalUnresolvedItem[];
  provenance: FormProposalProvenance;
  failure_code?: string;
  failure_message?: string;
  created_by: string;
  reviewed_by?: string;
  accepted_change_ids?: string[];
  result_template_id?: string;
  result_template_version?: number;
  created_at: string;
  updated_at: string;
  reviewed_at?: string;
  version: number;
};

export type RequestAIFormProposalInput = {
  objective: string;
  source_document_id?: string;
  expected_source_document_version?: number;
  source_element_refs?: string[];
};
