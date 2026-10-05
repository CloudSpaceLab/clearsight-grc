export type NotificationHistoryItem = {
  channel: "IN_APP" | "ATTENTION_CRITICAL" | string;
  kind: string;
  status: string;
  notice_sequence?: number;
  occurred_at: string;
};

export type NotificationDeliveryStatusCount = {
  delivery_class: string;
  status: string;
  count: number;
};

export type NotificationDeliveryFailure = {
  delivery_class: string;
  status: string;
  failure_code?: string;
  attempt_count: number;
  attempted_at: string;
};

export type NotificationDeliveryHealth = {
  as_of: string;
  window_start: string;
  counts: NotificationDeliveryStatusCount[];
  recent_failures: NotificationDeliveryFailure[];
};
