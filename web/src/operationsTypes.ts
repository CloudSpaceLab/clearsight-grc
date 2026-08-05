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
