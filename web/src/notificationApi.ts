import { requestJSON } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type InAppNotification = {
  id: string;
  kind: string;
  title: string;
  summary: string;
  subject_type: string;
  subject_id: string;
  action_path: string;
  occurred_at: string;
  read_at?: string;
};

export type NotificationPage = {
  items: InAppNotification[];
  next_cursor?: string;
  unread_count: number;
  as_of: string;
};

export function loadNotifications({ cursor, limit = 25, unreadOnly = false }: { cursor?: string; limit?: number; unreadOnly?: boolean } = {}): Promise<NotificationPage> {
  const params = new URLSearchParams({ limit: String(limit), unread_only: String(unreadOnly) });
  if (cursor) params.set("cursor", cursor);
  return requestJSON<NotificationPage>(apiBase, `/api/v1/notifications?${params.toString()}`);
}

export function markNotificationRead(id: string): Promise<InAppNotification> {
  return requestJSON<InAppNotification>(apiBase, `/api/v1/notifications/${encodeURIComponent(id)}/read`, { method: "POST", body: "{}" });
}
