export type RiskStatus = "DRAFT" | "ACTIVE" | "RETIRED";
export type RiskAssessmentKind = "INHERENT" | "CURRENT" | "RESIDUAL" | "TARGET" | "STRESSED" | "ACCEPTED";
export type RiskAppetitePosition = "WITHIN" | "APPROACHING" | "BREACHED" | "UNKNOWN";
export type RiskAppetiteStatus = "ACTIVE" | "RETIRED";

export type RiskControlImplementationStatus = "PLANNED" | "IN_PROGRESS" | "IMPLEMENTED" | "INACTIVE" | "RETIRED";
export type RiskControlEvidenceConclusion = "SUPPORTED" | "PARTIALLY_SUPPORTED" | "UNSUPPORTED" | "CONTRADICTED" | "INDETERMINATE" | "EXPIRED";

export type RiskControlLink = {
  id: string;
  risk_id: string;
  risk_version: number;
  catalog_link_id: string;
  linked_by?: string;
  created_at: string;
};

export type RiskControlDefinition = {
  id: string;
  code: string;
  name: string;
  objective: string;
  description: string;
  category: string;
  status: "ACTIVE" | "RETIRED";
  version: number;
};

export type RiskControlEvidence = {
  contract_id: string;
  name: string;
  conclusion?: RiskControlEvidenceConclusion;
  assessed_at?: string;
  valid_until?: string;
};

export type RiskControlDetail = {
  link: RiskControlLink;
  definition: RiskControlDefinition;
  program_id: string;
  program_name: string;
  implementation_id: string;
  objective_id: string;
  implementation_name: string;
  implementation_type: string;
  implementation_status: RiskControlImplementationStatus;
  implementation_version: number;
  owner_display_name?: string;
  owner_assigned: boolean;
  evidence: RiskControlEvidence[];
};

export type RiskRecord = {
  id: string;
  tenant_id: string;
  legal_entity_id: string;
  code: string;
  name: string;
  category: string;
  statement: string;
  cause: string;
  event: string;
  impact: string;
  scope: Record<string, unknown>;
  owner_principal_id?: string;
  status: RiskStatus;
  version: number;
  created_at: string;
  updated_at: string;
};

export type RiskAssessment = {
  id: string;
  risk_id: string;
  risk_version: number;
  kind: RiskAssessmentKind;
  method_code: string;
  method_version: string;
  dimensions: Record<string, unknown>;
  assumptions: Record<string, unknown>;
  evidence_references: unknown[];
  confidence?: number;
  assessed_by?: string;
  appetite_statement_id?: string;
  appetite_position: RiskAppetitePosition;
  appetite_rationale: string;
  assessed_at: string;
  created_at: string;
};

export type RiskAppetiteStatement = {
  id: string;
  risk_id: string;
  risk_version: number;
  version: number;
  statement: string;
  rule: Record<string, unknown>;
  rationale: string;
  owner_principal_id?: string;
  authority_principal_id?: string;
  status: RiskAppetiteStatus;
  effective_from: string;
  effective_until?: string;
  created_at: string;
};

export type RiskSummary = {
  risk: RiskRecord;
  latest_assessment?: RiskAssessment;
  active_appetite?: RiskAppetiteStatement;
};

export type RiskPage = {
  items: RiskSummary[];
  next_cursor?: string;
};

export type RiskAggregate = {
  risk: RiskRecord;
  assessments: RiskAssessment[];
  appetite: RiskAppetiteStatement[];
  active_appetite?: RiskAppetiteStatement;
  controls?: RiskControlLink[];
  control_details?: RiskControlDetail[];
  control_details_complete?: boolean;
};
