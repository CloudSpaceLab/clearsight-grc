export type JourneyStep = {
  code: string;
  label: string;
  complete: boolean;
};

export type JourneyActionTarget = "PROGRAM" | "MATTER" | "EVIDENCE_REQUEST";

export type BankJourney = {
  code: "NDPA_CONTINUOUS" | "REGULATORY_CHANGE" | "AUTHORITY_REQUEST" | "FINDING_REMEDIATION";
  title: string;
  summary: string;
  status: string;
  status_label: string;
  next_action: string;
  owner: string;
  owner_principal_id?: string;
  program_id?: string;
  matter_id?: string;
  evidence_request_id?: string;
  action_target_type?: JourneyActionTarget;
  action_target_id?: string;
  action_label?: string;
  action_available: boolean;
  action_unavailable_reason?: string;
  due_at?: string;
  completed_steps: number;
  total_steps: number;
  steps: JourneyStep[];
  source_names: string[];
  sensitive: boolean;
  sample: boolean;
  updated_at?: string;
};

export type BankJourneysResponse = {
  items: BankJourney[];
  generated_at: string;
  sample: boolean;
};
