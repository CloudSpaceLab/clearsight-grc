import type { MonitoringNativeMeasurement, RiskBand } from "./monitoringTypes";

export type IndicatorKind = "KRI" | "KCI";
export type IndicatorState = "NORMAL" | "WATCH" | "BREACH" | "UNKNOWN";

export type IndicatorIntervention = {
  matter_id: string;
  reference: string;
  status: string;
  priority: number;
  created_at: string;
};

export type IndicatorRiskReference = {
  id: string;
  name: string;
};

export type IndicatorDetailModel = {
  kind: IndicatorKind;
  program_id: string;
  program_name: string;
  check_id: string;
  check_code: string;
  check_name: string;
  claim: string;
  check_status: "DRAFT" | "PENDING_APPROVAL" | "ACTIVE" | "REJECTED" | "PAUSED" | "RETIRED";
  check_version: number;
  input_kind: "FORM" | "SOURCE";
  owner_display_name?: string;
  reviewer_display_name?: string;
  native_measurement?: MonitoringNativeMeasurement;
  state: IndicatorState;
  reason: string;
  score?: number;
  band?: RiskBand;
  coverage?: number;
  minimum_coverage: number;
  freshness_minutes: number;
  result_id?: string;
  evaluated_at?: string;
  intervention?: IndicatorIntervention;
  risks?: IndicatorRiskReference[];
};
