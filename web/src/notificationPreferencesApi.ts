import { requestJSON } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type NotificationPreferences = {
  tenant_id: string;
  principal_id: string;
  daily_digest_enabled: boolean;
  digest_minute: number;
  time_zone: string;
  quiet_hours_enabled: boolean;
  quiet_start_minute: number;
  quiet_end_minute: number;
  critical_email_required: boolean;
  updated_at?: string;
  version: number;
};

export type NotificationPreferenceUpdate = Pick<
  NotificationPreferences,
  "daily_digest_enabled" | "digest_minute" | "time_zone" | "quiet_hours_enabled" | "quiet_start_minute" | "quiet_end_minute" | "version"
>;

export function loadNotificationPreferences(signal?: AbortSignal): Promise<NotificationPreferences> {
  return requestJSON<NotificationPreferences>(
    apiBase,
    "/api/v1/preferences/notifications",
    signal ? { signal } : undefined,
  );
}

export function saveNotificationPreferences(value: NotificationPreferenceUpdate): Promise<NotificationPreferences> {
  return requestJSON<NotificationPreferences>(apiBase, "/api/v1/preferences/notifications", {
    method: "PUT",
    body: JSON.stringify({
      daily_digest_enabled: value.daily_digest_enabled,
      digest_minute: value.digest_minute,
      time_zone: value.time_zone,
      quiet_hours_enabled: value.quiet_hours_enabled,
      quiet_start_minute: value.quiet_start_minute,
      quiet_end_minute: value.quiet_end_minute,
      expected_version: value.version,
    }),
  });
}
