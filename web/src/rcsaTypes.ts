export type RCSAStatus = "DRAFT" | "ASSESSMENT_OPEN" | "AWAITING_CHALLENGE" | "COMPLETED" | "CANCELLED";
export type RCSATriggerKind = "SCHEDULED" | "CHANGE" | "MANUAL";

export type RCSACycle = {
  id: string;
  tenant_id: string;
  legal_entity_id: string;
  code: string;
  name: string;
  trigger_kind: RCSATriggerKind;
  first_line_owner_principal_id: string;
  status: RCSAStatus;
  population_checksum: string;
  first_line_distribution_id?: string;
  first_line_response_revision_id?: string;
  challenge_matter_id?: string;
  version: number;
  created_at: string;
  updated_at: string;
};

export type RCSARiskSnapshot = {
  cycle_id: string;
  risk_id: string;
  risk_version: number;
  code: string;
  name: string;
  category?: string;
};

export type RCSAControlSnapshot = {
  cycle_id: string;
  risk_id: string;
  risk_version: number;
  risk_control_link_id: string;
  definition_id: string;
  definition_code: string;
  definition_name: string;
  program_id: string;
  implementation_id: string;
  implementation_version: number;
  implementation_name: string;
  implementation_status: string;
};

export type RCSAHandoff = {
  stage: "SETUP" | "FIRST_LINE" | "CHALLENGE" | "COMPLETE" | "CANCELLED" | "UNKNOWN";
  label: string;
  target_type?: "EVIDENCE_REQUEST" | "MATTER";
  target_id?: string;
};

export type RCSACycleSummary = {
  cycle: RCSACycle;
  risk_count: number;
  control_count: number;
  first_line_owner_display_name?: string;
  handoff: RCSAHandoff;
};

export type RCSACyclePage = {
  items: RCSACycleSummary[];
  next_cursor?: string;
  complete: boolean;
};

export type RCSAChallengeContext = {
  matter_id?: string;
  matter_status?: string;
  decision_status?: string;
  decision_option?: "ACCEPT_FIRST_LINE" | "REQUIRE_CHANGES" | "DEFICIENCY_CONFIRMED" | string;
  open_action_count: number;
  implemented_action_count: number;
  blocked_action_count: number;
  active_verification_count: number;
  passed_verification_count: number;
  failed_verification_count: number;
  inconclusive_verification_count: number;
};

export type RCSACycleDetail = {
  cycle: RCSACycle;
  risks: RCSARiskSnapshot[];
  controls: RCSAControlSnapshot[];
  first_line_owner_display_name?: string;
  assessment_period_start?: string;
  assessment_period_end?: string;
  first_line_request_id?: string;
  challenge_context?: RCSAChallengeContext;
  challenge_context_complete: boolean;
  handoff: RCSAHandoff;
  complete: boolean;
};
