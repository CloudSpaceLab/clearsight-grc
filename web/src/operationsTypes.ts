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

export type SystemActivityCategory = "GRC_WORK" | "FORMS_EVIDENCE" | "VENDOR" | "AI" | "CONFIGURATION" | "SYSTEM" | "OTHER";
export type SystemActivityActorKind = "INTERNAL_USER" | "EXTERNAL_PARTICIPANT" | "SERVICE" | "SYSTEM" | "UNKNOWN";

export type SystemActivityEvent = {
  event_id: string;
  occurred_at: string;
  category: SystemActivityCategory;
  event_type: string;
  action: string;
  outcome: "SUCCEEDED" | "DENIED" | "FAILED" | "CANCELLED" | "PENDING" | "RETRYING";
  actor_kind: SystemActivityActorKind;
  actor_id?: string;
  actor_display_name?: string;
  legal_entity_id?: string;
  object_type: string;
  object_id: string;
  request_id?: string;
  correlation_id?: string;
  source: string;
};

export type SystemActivityPage = {
  items: SystemActivityEvent[];
  next_cursor?: string;
  as_of: string;
};

export type SystemActivityQuery = {
  cursor?: string;
  from?: string;
  to?: string;
  category?: SystemActivityCategory | "";
  eventType?: string;
  objectType?: string;
  objectID?: string;
  actorID?: string;
  legalEntityID?: string;
  limit?: number;
};
