export type ProjectionHealth = {
  tenant_id: string;
  projection: string;
  display_name: string;
  state: "CURRENT" | "UPDATE_PENDING" | "DELAYED" | "NEEDS_ATTENTION" | "NOT_CONFIGURED";
  pending: number;
  failed: number;
  oldest_pending?: string;
  last_completed?: string;
  last_error?: string;
  lag_seconds: number;
  updated_at: string;
};

export type ReconcileResult = {
  tenant_id: string;
  checked: number;
  queued: number;
  already_queued: number;
  current: number;
};

export type BackgroundJob = { id: string; queue: string; kind: string; state: string; attempts: number; failure_code?: string; terminal_at?: string };
export type BackgroundJobSnapshot = { queues: Array<{ queue: string; pending: number; running: number; terminal: number; highest_attempts: number; oldest_pending?: string }>; jobs: BackgroundJob[] };
export type JobRecoveryReceipt = { job_id: string; queue: string; previous_attempts: number; state: string; retried_at: string };
