import { requestJSON } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type NotificationHistoryEvent = {
  event_id: string;
  kind: string;
  occurred_at: string;
  in_app_deliveries: number;
  email_status?: string;
  email_attempts?: number;
};

export type NotificationHistory = {
  items: NotificationHistoryEvent[];
  as_of: string;
};

export type NotificationHistorySubject = "RISK" | "LOSS";

export function loadNotificationHistory(subject: NotificationHistorySubject, id: string, signal?: AbortSignal): Promise<NotificationHistory> {
  const collection = subject === "RISK" ? "risks" : "losses";
  return requestJSON<NotificationHistory>(
    apiBase,
    `/api/v1/${collection}/${encodeURIComponent(id)}/notification-history`,
    signal ? { signal } : undefined,
  );
}
