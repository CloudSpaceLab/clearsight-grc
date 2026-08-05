import type { Matter, Program, ProgramState, StateReason } from "./types";

export type ProgramSummary = {
  program: Program;
  state_label: string;
  overall_state: ProgramState;
  reasons: StateReason[];
  open_matter_count: number;
  requirement_count: number;
  safeguard_count: number;
  evidence_check_count: number;
  state_generated_at?: string;
};

export type MatterSummary = {
  matter: Matter;
  type_label: string;
  status_label: string;
  next_action: string;
  program_count: number;
  open_action_count: number;
  outcome_check_count: number;
  latest_outcome?: string;
  latest_outcome_at?: string;
};

export type SummaryPage<T> = {
  items: T[];
  next_cursor?: string;
  generated_at: string;
};

export type SummaryQuery = {
  q?: string;
  status?: string;
  cursor?: string;
  limit?: number;
};
