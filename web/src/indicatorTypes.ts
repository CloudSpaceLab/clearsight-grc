import type { RiskIndicatorDetail, RiskIndicatorKind, RiskStatus } from "./riskTypes";

export type IndicatorRiskReference = {
  id: string;
  code: string;
  name: string;
  organization_scope_id?: string;
  status: RiskStatus;
};

export type IndicatorPopulationItem = {
  indicator: RiskIndicatorDetail;
  risks: IndicatorRiskReference[];
  risk_count: number;
  risks_truncated?: boolean;
  kind_conflict?: boolean;
};

export type IndicatorPopulationPage = {
  items: IndicatorPopulationItem[];
  truncated?: boolean;
  complete: boolean;
  organization_scope_id?: string;
};

export type IndicatorPopulationParams = {
  kind?: RiskIndicatorKind;
  organizationScopeID?: string;
  checkID?: string;
  limit?: number;
};
