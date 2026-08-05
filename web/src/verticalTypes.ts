export type JourneyStep = {
  code: string;
  label: string;
  complete: boolean;
};

export type BankJourney = {
  code: "NDPA_CONTINUOUS" | "REGULATORY_CHANGE" | "AUTHORITY_REQUEST" | "FINDING_REMEDIATION";
  title: string;
  summary: string;
  status: string;
  status_label: string;
  next_action: string;
  owner: string;
  program_id?: string;
  matter_id?: string;
  evidence_request_id?: string;
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
