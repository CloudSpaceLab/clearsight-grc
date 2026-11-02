import { afterEach, describe, expect, it, vi } from "vitest";
import { activityEvidenceCursor, createNotificationWorkspaceEvidence, installNotificationWorkspaceEvidence, notificationEvidenceCursor,
  type NotificationEvidenceCommand, type NotificationEvidenceRead } from "./notificationWorkspaceEvidence";
import type { NotificationPage } from "./notificationApi";
import type { NotificationPreferences } from "./notificationPreferencesApi";

const context = { actor: { id: "sample-actor" }, legal_entity: { id: "bank-ng" }, capabilities: { config_read: true, platform_operations_read: true } };
const first = "/api/v1/notifications?limit=25&unread_only=false";
const activity = "/api/v1/system-activity?legal_entity_id=bank-ng&limit=25&from=2026-09-21T12%3A00%3A00.000Z&to=2026-10-21T12%3A00%3A00.000Z";
function setup(fixture = "notifications-loaded") {
  const commands: NotificationEvidenceCommand[] = [];
  const reads: NotificationEvidenceRead[] = [];
  const readContext = vi.fn(async () => new Response(JSON.stringify(context)));
  const handle = createNotificationWorkspaceEvidence(fixture, commands, reads, readContext)!;
  async function get<T>(path: string): Promise<T> { const response = await handle(path); expect(response?.status).toBe(200); return response!.json(); }
  async function command(path: string, method: string, body: unknown) { return handle(path, { method, body: JSON.stringify(body) }); }
  return { handle, commands, reads, readContext, get, command };
}

afterEach(() => { vi.unstubAllGlobals(); window.history.replaceState({}, "", "/"); delete window.notificationEvidenceCommands; delete window.notificationEvidenceReads; });

describe("isolated Notifications browser evidence", () => {
  it("keeps server unread totals distinct from loaded rows and preserves the opaque cursor", async () => {
    const { get, handle, reads } = setup();
    const initial = await get<NotificationPage>(first);
    expect(initial.items).toHaveLength(5);
    expect(initial.items.filter((item) => !item.read_at)).toHaveLength(3);
    expect(initial.unread_count).toBe(7);
    expect(initial.next_cursor).toBe(notificationEvidenceCursor);
    const next = await get<NotificationPage>(`${first}&cursor=${encodeURIComponent(initial.next_cursor!)}`);
    expect(next.items.map((item) => item.id)).toEqual(["sample-notification-6", "sample-notification-7", "sample-notification-8", "sample-notification-9", "sample-notification-10"]);
    expect(next.next_cursor).toBeUndefined();
    expect([...initial.items, ...next.items].filter((item) => !item.read_at)).toHaveLength(initial.unread_count);
    const unread = await get<NotificationPage>(first.replace("false", "true"));
    expect(unread.items.map((item) => item.id)).toEqual(["sample-notification-1", "sample-notification-2", "sample-notification-4"]);
    expect((await handle(`${first}&cursor=made-up`))?.status).toBe(400);
    expect((await handle(`${first}&actor_id=other`))?.status).toBe(400);
    expect(reads).toHaveLength(5);
  });

  it("persists one read receipt without completing work or accepting supplied identity", async () => {
    const { command, get, commands } = setup();
    const path = "/api/v1/notifications/sample-notification-1/read";
    expect((await command(path, "POST", { actor_id: "other" }))?.status).toBe(400);
    const reply = await command(path, "POST", {});
    expect(reply?.status).toBe(200);
    expect(await reply!.json()).toMatchObject({ id: "sample-notification-1", read_at: "2026-10-21T12:05:00Z", action_path: "#work/matters/matter-gaid-change" });
    await command(path, "POST", {});
    expect(await get<NotificationPage>(first)).toMatchObject({ unread_count: 6, as_of: "2026-10-21T12:05:00Z" });
    expect((await get<NotificationPage>(first.replace("false", "true"))).items.some((item) => item.id === "sample-notification-1")).toBe(false);
    expect(commands.map(({ status }) => status)).toEqual([400, 200, 200]);
    expect(commands.every(({ path: actual }) => actual === path)).toBe(true);
    expect((await setup().get<NotificationPage>(first)).unread_count).toBe(7);
  });

  it("retains unread state on read failure and loaded rows on older-page failure", async () => {
    const failed = setup("notifications-read-failed");
    expect((await failed.command("/api/v1/notifications/sample-notification-1/read", "POST", {}))?.status).toBe(503);
    const page = await failed.get<NotificationPage>(first);
    expect(page.unread_count).toBe(7);
    expect(page.items[0]!.read_at).toBeUndefined();
    const paging = setup("notifications-more-unavailable");
    expect((await paging.get<NotificationPage>(first)).items).toHaveLength(5);
    expect((await paging.handle(`${first}&cursor=${encodeURIComponent(notificationEvidenceCursor)}`))?.status).toBe(503);
    expect((await paging.get<NotificationPage>(first)).items).toHaveLength(5);
  });

  it("distinguishes an empty inbox from a failed read", async () => {
    expect(await setup("notifications-empty").get<NotificationPage>(first)).toEqual({ items: [], unread_count: 0, as_of: "2026-10-21T12:00:00Z" });
    expect((await setup("notifications-unavailable").handle(first))?.status).toBe(503);
  });

  it("validates versioned preferences and never changes the required critical-email policy", async () => {
    const { command, get } = setup();
    const path = "/api/v1/preferences/notifications";
    const value = await get<NotificationPreferences>(path);
    const body = { daily_digest_enabled: true, digest_minute: 585, time_zone: "Africa/Lagos", quiet_hours_enabled: true,
      quiet_start_minute: 1330, quiet_end_minute: 380, expected_version: value.version };
    expect((await command(path, "PUT", { ...body, principal_id: "other" }))?.status).toBe(400);
    expect((await command(path, "PUT", { ...body, digest_minute: 1440 }))?.status).toBe(400);
    const response = await command(path, "PUT", body);
    expect(response?.status).toBe(200);
    expect(await response!.json()).toMatchObject({ digest_minute: 585, quiet_start_minute: 1330, quiet_end_minute: 380, critical_email_required: true, version: 4 });
    expect((await command(path, "PUT", body))?.status).toBe(409);
    const conflict = setup("notifications-preferences-conflict");
    expect((await conflict.command(path, "PUT", body))?.status).toBe(409);
    expect((await conflict.get<NotificationPreferences>(path)).digest_minute).toBe(480);
  });

  it("requires Activity entity and period, preserves pagination, and exposes a foreign-scope failure fixture", async () => {
    const { get, handle } = setup();
    const page = await get<{ items: Array<{ legal_entity_id: string }>; next_cursor: string }>(activity);
    expect(page.items).toHaveLength(3);
    expect(page.items.every((item) => item.legal_entity_id === "bank-ng")).toBe(true);
    expect(page.next_cursor).toBe(activityEvidenceCursor);
    expect((await get<{ items: unknown[] }>(`${activity}&cursor=${encodeURIComponent(page.next_cursor)}`)).items).toHaveLength(2);
    expect((await handle(activity.replace("bank-ng", "other")))?.status).toBe(400);
    expect((await handle(activity.replace("2026-09-21", "2026-09-20")))?.status).toBe(400);
    const foreign = await setup("notifications-activity-foreign").get<{ items: Array<{ legal_entity_id: string }> }>(activity);
    expect(foreign.items[1]!.legal_entity_id).toBe("sample-other-entity");
    expect((await setup("notifications-activity-unavailable").handle(activity))?.status).toBe(503);
  });

  it("restricts Activity independently from inbox delivery and Settings permission", async () => {
    const denied = setup("notifications-activity-denied");
    expect(await denied.get<typeof context>("/api/v1/context")).toEqual({ ...context, capabilities: { ...context.capabilities, platform_operations_read: false } });
    expect((await denied.handle(activity))?.status).toBe(403);
    expect((await denied.get<NotificationPage>(first)).items).toHaveLength(5);
    expect(denied.readContext).toHaveBeenCalledOnce();
  });

  it("installs only for the named fixture family and delegates unrelated paths without mutation", async () => {
    const previous = vi.fn(async () => new Response(JSON.stringify({ delegated: true })));
    vi.stubGlobal("fetch", previous);
    window.history.replaceState({}, "", "/?fixture=report-definitions");
    installNotificationWorkspaceEvidence();
    expect(fetch).toBe(previous);
    window.history.replaceState({}, "", "/?fixture=notifications-loaded");
    installNotificationWorkspaceEvidence();
    expect(await (await fetch(first)).json()).toMatchObject({ unread_count: 7 });
    expect(await (await fetch("/api/v1/unrelated")).json()).toEqual({ delegated: true });
    expect(previous).toHaveBeenCalledWith("/api/v1/unrelated", undefined);
    expect(window.notificationEvidenceCommands).toEqual([]);
  });
});
