export type NotificationDeliveryOutcome = "SUCCEEDED" | "FAILED" | "CANCELLED" | "PENDING" | "RETRYING";

export type NotificationDeliveryHistoryItem = {
  event_id: string;
  occurred_at: string;
  event_type: string;
  action: string;
  outcome: NotificationDeliveryOutcome;
};
