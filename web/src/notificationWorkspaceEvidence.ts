// Synthetic UI evidence only. This module is imported by evidenceMain, never the customer entry.
// Read receipts and preferences here do not prove backend authorization or a completed workflow.
import type { InAppNotification, NotificationPage } from "./notificationApi";
import type { NotificationPreferences } from "./notificationPreferencesApi";
import type { SystemActivityEvent } from "./operationsTypes";

export type NotificationEvidenceCommand = { method: string; path: string; body: unknown; status: number; result: unknown };
export type NotificationEvidenceRead = { path: string; query: string; status: number };
declare global {
  interface Window {
    notificationEvidenceCommands?: NotificationEvidenceCommand[];
    notificationEvidenceReads?: NotificationEvidenceRead[];
  }
}

const variants = new Set(["notifications-loaded", "notifications-empty", "notifications-unavailable", "notifications-read-failed",
  "notifications-preferences-conflict", "notifications-activity-denied", "notifications-activity-unavailable", "notifications-activity-foreign",
  "notifications-more-unavailable"]);
const at = "2026-10-21T12:00:00Z";
export const notificationEvidenceCursor = "sample-notifications/older?batch=2&recipient=sample";
export const activityEvidenceCursor = "sample-activity/older?batch=2&entity=bank-ng";

function seedNotifications(): InAppNotification[] {
  return [
    ["Review the Sample annual-return evidence change", "Review the annual-return evidence requirements before recording the next action.", "MATTER_OWNER_ASSIGNED", "MATTER", "matter-gaid-change", "#work/matters/matter-gaid-change"],
    ["Sample privacy review mentions you", "A reviewer asked you to confirm the customer contact fields.", "MATTER_COMMENT_MENTIONED", "MATTER", "matter-gaid-change", "#work/matters/matter-gaid-change"],
    ["Sample Program review recorded", "The quarterly review is available in the Program record.", "PROGRAM_REVIEW_RECORDED", "PROGRAM", "program-ndpa", "#programs/program-ndpa/overview"],
    ["Sample external reference unavailable", "This delivered update has no supported application destination.", "RECORD_UPDATED", "REFERENCE", "sample-external-reference", "https://example.invalid/notification"],
    ["Sample annual-return evidence updated", "Review the updated evidence requirements in the change record.", "RECORD_UPDATED", "MATTER", "matter-gaid-change", "#work/matters/matter-gaid-change"],
    ["Sample evidence comment mentions you", "Review the comment attached to the evidence request.", "MATTER_COMMENT_MENTIONED", "MATTER", "matter-gaid-change", "#work/matters/matter-gaid-change"],
    ["Sample Program ownership updated", "The Program record contains the latest ownership information.", "RECORD_UPDATED", "PROGRAM", "program-ndpa", "#programs/program-ndpa/overview"],
    ["Sample annual-return review requested", "Review the annual-return change before recording a response.", "MATTER_OWNER_ASSIGNED", "MATTER", "matter-gaid-change", "#work/matters/matter-gaid-change"],
    ["Sample Program evidence requested", "The Program record lists the evidence required for its next review.", "RECORD_UPDATED", "PROGRAM", "program-ndpa", "#programs/program-ndpa/overview"],
    ["Sample review comment mentions you", "A reviewer asked for your response in the annual-return change.", "MATTER_COMMENT_MENTIONED", "MATTER", "matter-gaid-change", "#work/matters/matter-gaid-change"],
  ].map(([title, summary, kind, subject_type, subject_id, action_path], index) => ({
    id: `sample-notification-${index + 1}`, title: title!, summary: summary!, kind: kind!, subject_type: subject_type!, subject_id: subject_id!, action_path: action_path!,
    occurred_at: new Date(Date.parse(at) - index * 3_600_000).toISOString(),
    ...([2, 4, 6].includes(index) ? { read_at: new Date(Date.parse(at) - index * 1_800_000).toISOString() } : {}),
  }));
}

export function createNotificationWorkspaceEvidence(fixture: string | null, commands: NotificationEvidenceCommand[], reads: NotificationEvidenceRead[], readContext: () => Promise<Response>) {
  if (!fixture || !variants.has(fixture)) return undefined;
  const items = fixture === "notifications-empty" ? [] : seedNotifications();
  let unreadCount = items.length ? 7 : 0;
  let inboxAsOf = at;
  let preferences: NotificationPreferences = {
    tenant_id: "bank-demo", principal_id: "sample-notification-recipient", daily_digest_enabled: true, digest_minute: 480,
    time_zone: "Africa/Lagos", quiet_hours_enabled: true, quiet_start_minute: 1260, quiet_end_minute: 420,
    critical_email_required: true, updated_at: at, version: 3,
  };
  return async (input: RequestInfo | URL, init?: RequestInit): Promise<Response | undefined> => {
    const raw = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    const url = new URL(raw, window.location.origin);
    const method = (init?.method ?? (input instanceof Request ? input.method : "GET")).toUpperCase();
    const path = url.pathname;
    let body: unknown;
    const reply = (result: unknown, status = 200) => {
      if (method === "GET") reads.push({ path, query: url.search, status });
      else commands.push({ method, path, body: clone(body), status, result: clone(result) });
      return json(result, status);
    };
    if (method === "GET" && path === "/api/v1/context") {
      const response = await readContext();
      if (!response.ok) return response;
      const context = await response.json();
      // Match the verified fixture context; callers never supply an actor in a command.
      preferences = { ...preferences, principal_id: context.actor?.id ?? preferences.principal_id, tenant_id: context.tenant?.id ?? preferences.tenant_id };
      return json({ ...context, capabilities: { ...context.capabilities, platform_operations_read: fixture !== "notifications-activity-denied" } });
    }
    if (method === "GET" && path === "/api/v1/notifications") {
      if (url.searchParams.get("limit") !== "25" || !["true", "false"].includes(url.searchParams.get("unread_only") ?? "")
        || [...url.searchParams.keys()].some((key) => !["limit", "unread_only", "cursor"].includes(key))) {
        return reply(error("notification_query_invalid", "Sample inbox query is not supported."), 400);
      }
      if (fixture === "notifications-unavailable") return reply(error("notifications_unavailable", "Sample notifications are temporarily unavailable."), 503);
      const cursor = url.searchParams.get("cursor");
      if (cursor && cursor !== notificationEvidenceCursor) return reply(error("notification_cursor_invalid", "Reload the sample inbox."), 400);
      if (cursor && fixture === "notifications-more-unavailable") return reply(error("notifications_unavailable", "Older sample notifications are temporarily unavailable."), 503);
      const pageItems = items.slice(cursor ? 5 : 0, cursor ? 10 : 5).filter((item) => url.searchParams.get("unread_only") !== "true" || !item.read_at);
      const page: NotificationPage = { items: pageItems, unread_count: unreadCount, as_of: inboxAsOf,
        ...(!cursor && items.length > 5 ? { next_cursor: notificationEvidenceCursor } : {}) };
      return reply(page);
    }
    if (method === "GET" && path === "/api/v1/preferences/notifications") return reply(preferences);
    if (method === "GET" && path === "/api/v1/system-activity") {
      const from = Date.parse(url.searchParams.get("from") ?? "");
      const to = Date.parse(url.searchParams.get("to") ?? "");
      const days = (to - from) / 86_400_000;
      const cursor = url.searchParams.get("cursor");
      if (url.searchParams.get("legal_entity_id") !== "bank-ng" || url.searchParams.get("limit") !== "25"
        || ![7, 30, 90].includes(days) || !Number.isFinite(to)
        || (cursor && cursor !== activityEvidenceCursor)
        || [...url.searchParams.keys()].some((key) => !["legal_entity_id", "limit", "from", "to", "cursor"].includes(key))) {
        return reply(error("activity_query_invalid", "Select the sample legal entity and supported activity period."), 400);
      }
      if (fixture === "notifications-activity-denied") return reply(error("permission_denied", "Sample activity access is restricted."), 403);
      if (fixture === "notifications-activity-unavailable") return reply(error("activity_unavailable", "Sample activity is temporarily unavailable."), 503);
      const events = fixture === "notifications-empty" ? [] : activityEvents(to, fixture === "notifications-activity-foreign");
      return reply({ items: events.slice(cursor ? 3 : 0, cursor ? 5 : 3), as_of: new Date(to).toISOString(),
        ...(!cursor && events.length > 3 ? { next_cursor: activityEvidenceCursor } : {}) });
    }
    const read = /^\/api\/v1\/notifications\/([^/]+)\/read$/.exec(path);
    if (method !== "POST" && method !== "PUT" || !read && path !== "/api/v1/preferences/notifications") return undefined;
    try {
      const text = init?.body !== undefined && init.body !== null ? await new Response(init.body).text()
        : input instanceof Request ? await input.clone().text() : "";
      body = JSON.parse(text);
    } catch { return reply(error("notification_request_invalid", "Sample request is invalid."), 400); }
    if (!record(body)) return reply(error("notification_request_invalid", "Sample request is invalid."), 400);
    if (read && method === "POST") {
      if (Object.keys(body).length) return reply(error("notification_request_invalid", "The read receipt does not accept actor or record fields."), 400);
      const item = items.find((entry) => entry.id === decodeURIComponent(read[1]!));
      if (!item) return reply(error("notification_not_found", "Sample notification is unavailable."), 404);
      if (fixture === "notifications-read-failed") return reply(error("notification_read_failed", "Sample read status could not be saved."), 503);
      if (!item.read_at) { item.read_at = "2026-10-21T12:05:00Z"; inboxAsOf = item.read_at; unreadCount -= 1; }
      return reply(item);
    }
    if (path === "/api/v1/preferences/notifications" && method === "PUT") {
      const keys = ["daily_digest_enabled", "digest_minute", "time_zone", "quiet_hours_enabled", "quiet_start_minute", "quiet_end_minute", "expected_version"];
      if (Object.keys(body).length !== keys.length || Object.keys(body).some((key) => !keys.includes(key))
        || typeof body.daily_digest_enabled !== "boolean" || typeof body.quiet_hours_enabled !== "boolean"
        || ![body.digest_minute, body.quiet_start_minute, body.quiet_end_minute].every((minute) => Number.isInteger(minute) && Number(minute) >= 0 && Number(minute) < 1440)
        || typeof body.time_zone !== "string" || !validZone(body.time_zone)) {
        return reply(error("notification_preferences_invalid", "Sample email preferences are invalid."), 400);
      }
      if (body.expected_version !== preferences.version || fixture === "notifications-preferences-conflict") {
        return reply(error("version_conflict", "Email preferences changed. Reload before saving again."), 409);
      }
      preferences = { ...preferences, daily_digest_enabled: body.daily_digest_enabled, digest_minute: Number(body.digest_minute), time_zone: body.time_zone,
        quiet_hours_enabled: body.quiet_hours_enabled, quiet_start_minute: Number(body.quiet_start_minute), quiet_end_minute: Number(body.quiet_end_minute),
        updated_at: "2026-10-21T12:06:00Z", version: preferences.version + 1 };
      return reply(preferences);
    }
    return undefined;
  };
}

export function installNotificationWorkspaceEvidence() {
  const fixture = new URLSearchParams(window.location.search).get("fixture");
  if (!fixture || !variants.has(fixture)) return;
  const previous = globalThis.fetch.bind(globalThis);
  window.notificationEvidenceCommands = [];
  window.notificationEvidenceReads = [];
  const handle = createNotificationWorkspaceEvidence(fixture, window.notificationEvidenceCommands, window.notificationEvidenceReads,
    () => previous("/api/v1/context"))!;
  globalThis.fetch = async (input, init) => await handle(input, init) ?? previous(input, init);
}

function activityEvents(to: number, foreign: boolean): SystemActivityEvent[] {
  return [
    ["MATTER_OWNER_ASSIGNED", "Assigned Sample annual-return evidence review", "MATTER", "matter-gaid-change", "SUCCEEDED", "INTERNAL_USER", "Sample Amina Yusuf", "WORKFLOW"],
    ["PROGRAM_REVIEW_RECORDED", "Recorded Sample Program review", "PROGRAM", "program-ndpa", "SUCCEEDED", "INTERNAL_USER", "Sample Emeka Okoro", "PROGRAMS"],
    ["AUTHORITY_REQUEST_DENIED", "Denied Sample approval request", "MATTER", "sample-restricted-change", "DENIED", "INTERNAL_USER", "Sample Ruth Mensah", "AUTHORITY"],
    ["EVIDENCE_REQUESTED", "Requested Sample supplier evidence", "EVIDENCE", "sample-evidence-request", "PENDING", "SYSTEM", "Sample workflow service", "WORKFLOW"],
    ["NOTIFICATION_SENT", "Delivered Sample review update", "NOTIFICATION", "sample-notification-1", "SUCCEEDED", "SERVICE", "Sample notification service", "NOTIFICATIONS"],
  ].map(([event_type, action, object_type, object_id, outcome, actor_kind, actor_display_name, source], index) => ({
    event_id: `sample-activity-event-${index + 1}`, occurred_at: new Date(to - (index + 1) * 3_600_000).toISOString(),
    category: index === 4 ? "SYSTEM" : "GRC_WORK", event_type: event_type!, action: action!, outcome: outcome as SystemActivityEvent["outcome"],
    actor_kind: actor_kind as SystemActivityEvent["actor_kind"], actor_id: `sample-activity-actor-${index + 1}`, actor_display_name,
    legal_entity_id: foreign && index === 1 ? "sample-other-entity" : "bank-ng", object_type: object_type!, object_id: object_id!, source: source!,
  }));
}
function record(value: unknown): value is Record<string, unknown> { return Boolean(value) && typeof value === "object" && !Array.isArray(value); }
function validZone(value: string) { try { new Intl.DateTimeFormat("en", { timeZone: value }).format(); return true; } catch { return false; } }
function clone<T>(value: T): T { return value === undefined ? value : structuredClone(value); }
function error(code: string, message: string) { return { error: { code, message } }; }
function json(value: unknown, status = 200) { return new Response(JSON.stringify(value), { status, headers: { "Content-Type": "application/json" } }); }
