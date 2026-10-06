import type { NotificationDeliveryHistoryItem } from "./notificationTypes";

export type LossStatus = "ACTIVE" | "VOIDED";
export type LossRecoveryStatus = "NONE" | "PARTIAL" | "FULL";
export type LossRecoveryKind = "RECOVERY" | "REVERSAL";
export type LossEventType =
  | "INTERNAL_FRAUD"
  | "EXTERNAL_FRAUD"
  | "EMPLOYMENT_PRACTICES"
  | "CLIENT_PRODUCTS_BUSINESS_PRACTICES"
  | "DAMAGE_TO_PHYSICAL_ASSETS"
  | "BUSINESS_DISRUPTION_SYSTEM_FAILURES"
  | "EXECUTION_DELIVERY_PROCESS_MANAGEMENT"
  | "OTHER";

export type LossRecord = {
  id: string;
  tenant_id: string;
  legal_entity_id: string;
  organization_scope_id?: string;
  code: string;
  title: string;
  event_type: LossEventType;
  cause: string;
  description: string;
  gross_amount_minor: number;
  currency: string;
  occurred_at: string;
  discovered_at: string;
  risk_id?: string;
  matter_id?: string;
  owner_principal_id?: string;
  status: LossStatus;
  version: number;
  created_at: string;
  updated_at: string;
};

export type LossRecovery = {
  id: string;
  loss_id: string;
  loss_version: number;
  kind: LossRecoveryKind;
  amount_minor: number;
  currency: string;
  reference: string;
  recovered_at: string;
  actor_id?: string;
  created_at: string;
};

export type LossTotals = {
  gross_amount_minor: number;
  recovered_amount_minor: number;
  net_loss_minor: number;
  currency: string;
  recovery_status: LossRecoveryStatus;
};

export type LossSummary = {
  loss: LossRecord;
  totals: LossTotals;
};

export type LossPage = {
  items: LossSummary[];
  next_cursor?: string;
};

export type LossAggregate = {
  loss: LossRecord;
  recoveries: LossRecovery[];
  totals: LossTotals;
  notification_history?: NotificationDeliveryHistoryItem[];
  notification_history_complete?: boolean;
};

export type LossInterventionResponse = {
  loss: LossRecord;
  matter: {
    id: string;
    reference: string;
    status: string;
    matter_type?: string;
  };
};
