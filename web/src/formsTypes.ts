import type { CapturePresentation, CaptureSection } from "./types";
import type { CreateFormTemplateInput, FormTemplate as MonitoringFormTemplate, LifecycleStatus } from "./monitoringTypes";

export type FormScoringMode = "NONE" | "RISK" | "COMPLIANCE";

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
  sections: CaptureSection[];
};

export type FormLibraryItem = {
  template: FormTemplate;
  active_version?: number;
  active_status?: LifecycleStatus;
};

export type FormTemplatePage = {
  items: FormLibraryItem[];
  next_cursor?: string;
};

export type FormTemplateQuery = {
  search?: string;
  status?: LifecycleStatus;
  owner?: string;
  program?: string;
  use?: string;
  tag?: string;
  cursor?: string;
  limit?: number;
};

export type SavedFormViewFilter = {
  search?: string;
  status?: LifecycleStatus;
  owner_principal_id?: string;
  program_id?: string;
  use?: string;
  tag?: string;
  limit?: number;
};

export type SavedFormView = {
  id: string;
  name: string;
  filter: SavedFormViewFilter;
  created_at: string;
  updated_at: string;
};

export type StarterTemplate = {
  code: string;
  catalog_version: number;
  published_on: string;
  reference_label: string;
  template: FormTemplate;
};

export type CreateLibraryFormInput = CreateFormTemplateInput & {
  program_id?: string;
  owner_principal_id?: string;
  responsible_team?: string;
  approved_uses?: string[];
  tags?: string[];
  jurisdiction?: string;
  industry?: string;
  sensitivity?: string;
  scoring_mode?: FormScoringMode;
  next_review_at?: string;
};

export type InstantiateStarterTemplateInput = {
  code?: string;
  name?: string;
  purpose?: string;
  program_id?: string;
  owner_principal_id?: string;
  responsible_team?: string;
  jurisdiction?: string;
  industry?: string;
  next_review_at?: string;
};
