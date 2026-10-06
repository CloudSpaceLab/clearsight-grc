import type { RCSACycleDetail, RCSACyclePage } from "./rcsaTypes";

const cycle = {
  id: "cycle-rcsa-1",
  tenant_id: "bank-demo",
  legal_entity_id: "bank-ng",
  code: "RCSA-Q3-TECH",
  name: "Q3 Technology RCSA",
  trigger_kind: "SCHEDULED" as const,
  first_line_owner_principal_id: "role-cro",
  status: "AWAITING_CHALLENGE" as const,
  population_checksum: "2d253353020ea1489ea2e9cb759aa1d57f8cb7d6a3bed2d59568d58d1fb1f316",
  first_line_distribution_id: "distribution-rcsa-1",
  first_line_response_revision_id: "response-rcsa-1",
  challenge_matter_id: "matter-gaid-change",
  version: 4,
  created_at: "2026-07-01T08:00:00Z",
  updated_at: "2026-10-05T08:00:00Z",
};

export const sampleRCSAPage: RCSACyclePage = {
  complete: true,
  items: [{
    cycle,
    risk_count: 3,
    control_count: 4,
    first_line_owner_display_name: "Morgan Ellis",
    handoff: {
      stage: "CHALLENGE",
      label: "Complete independent challenge",
      target_type: "MATTER",
      target_id: "matter-gaid-change",
    },
  }, {
    cycle: {
      ...cycle,
      id: "cycle-rcsa-2",
      code: "RCSA-Q4-OPS",
      name: "Q4 Operations RCSA",
      trigger_kind: "CHANGE",
      status: "ASSESSMENT_OPEN",
      first_line_distribution_id: "distribution-rcsa-2",
      first_line_response_revision_id: undefined,
      challenge_matter_id: undefined,
      version: 2,
      created_at: "2026-10-01T08:00:00Z",
      updated_at: "2026-10-04T11:30:00Z",
    },
    risk_count: 2,
    control_count: 2,
    first_line_owner_display_name: "Operations Risk",
    handoff: { stage: "FIRST_LINE", label: "Complete first-line assessment" },
  }],
};

export const sampleRCSADetail: RCSACycleDetail = {
  cycle,
  complete: true,
  first_line_owner_display_name: "Morgan Ellis",
  assessment_period_start: "2026-07-01T00:00:00Z",
  assessment_period_end: "2026-09-30T23:59:59Z",
  first_line_request_id: "request-rcsa-1",
  handoff: {
    stage: "CHALLENGE",
    label: "Complete independent challenge",
    target_type: "MATTER",
    target_id: "matter-gaid-change",
  },
  risks: [
    { cycle_id: cycle.id, risk_id: "risk-rcsa-1", risk_version: 3, code: "TECH-001", name: "Digital service interruption", category: "Technology" },
    { cycle_id: cycle.id, risk_id: "risk-rcsa-2", risk_version: 2, code: "CYB-012", name: "Privileged access misuse", category: "Cyber" },
    { cycle_id: cycle.id, risk_id: "risk-rcsa-3", risk_version: 5, code: "OPS-027", name: "Change implementation failure", category: "Operational" },
  ],
  controls: [
    {
      cycle_id: cycle.id, risk_id: "risk-rcsa-1", risk_version: 3, risk_control_link_id: "risk-control-1",
      definition_id: "control-1", definition_code: "BCP-01", definition_name: "Service recovery test",
      program_id: "program-1", implementation_id: "implementation-1", implementation_version: 2,
      implementation_name: "Quarterly recovery exercise", implementation_status: "IMPLEMENTED",
    },
    {
      cycle_id: cycle.id, risk_id: "risk-rcsa-2", risk_version: 2, risk_control_link_id: "risk-control-2",
      definition_id: "control-2", definition_code: "IAM-04", definition_name: "Privileged access review",
      program_id: "program-2", implementation_id: "implementation-2", implementation_version: 4,
      implementation_name: "Monthly privileged access review", implementation_status: "IMPLEMENTED",
    },
  ],
};
