export type RiskStatus = "DRAFT" | "ACTIVE" | "RETIRED";
export type RiskAssessmentKind = "INHERENT" | "CURRENT" | "RESIDUAL" | "TARGET" | "STRESSED" | "ACCEPTED";
export type RiskAppetitePosition = "WITHIN" | "APPROACHING" | "BREACHED" | "UNKNOWN";
export type RiskAppetiteStatus = "ACTIVE" | "RETIRED";

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
};
