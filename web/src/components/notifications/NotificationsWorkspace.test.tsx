import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { NotificationTab } from "../../appRouting";
import type { InAppNotification, NotificationPage } from "../../notificationApi";
import { loadNotificationPreferences, saveNotificationPreferences } from "../../notificationPreferencesApi";
import { useNotificationInbox } from "../useNotificationInbox";
import { NotificationsWorkspace } from "./NotificationsWorkspace";

vi.mock("../../notificationPreferencesApi", () => ({ loadNotificationPreferences: vi.fn(), saveNotificationPreferences: vi.fn() }));
vi.mock("./NotificationActivity", () => ({ NotificationActivity: ({ canRead }: { canRead: boolean }) => <section aria-label="Activity history">{canRead ? "Recorded actions" : "Activity access is restricted"}</section> }));

const items: InAppNotification[] = [
  { id: "notice-1", kind: "MATTER_OWNER_ASSIGNED", title: "Review service continuity", summary: "A service continuity issue was assigned to you.", subject_type: "MATTER", subject_id: "issue-1", action_path: "#work/matters/issue-1", occurred_at: "2026-10-20T09:00:00Z" },
  { id: "notice-2", kind: "MATTER_COMMENT_MENTIONED", title: "Mention in supplier review", summary: "A colleague mentioned you in a comment.", subject_type: "MATTER", subject_id: "issue-2", action_path: "#work/matters/issue-2", occurred_at: "2026-10-19T11:00:00Z" },
];

function Harness({ loadPage, markRead, onOpenPath = vi.fn(), onOpenWork = vi.fn() }: { loadPage: () => Promise<NotificationPage>; markRead: (id: string) => Promise<InAppNotification>; onOpenPath?: (path: string) => void; onOpenWork?: () => void }) {
  const inbox = useNotificationInbox({ scopeKey: "tenant:entity:actor", loadPage, markRead });
  const [tab, setTab] = useState<NotificationTab>("inbox");
  return <NotificationsWorkspace inbox={inbox} tab={tab} onTab={setTab} actorName="Jordan Ellis" legalEntityID="entity-1" legalEntityName="North Division" canReadActivity={false} onOpenPath={onOpenPath} onOpenWork={onOpenWork}/>;
}

function setup(overrides: { records?: InAppNotification[]; loadPage?: () => Promise<NotificationPage> } = {}) {
  let records = overrides.records ?? items;
  const loadPage = overrides.loadPage ?? vi.fn(async () => ({ items: records, unread_count: records.filter((item) => !item.read_at).length, as_of: "2026-10-20T10:00:00Z" }));
  const markRead = vi.fn(async (id: string) => {
    const updated = { ...records.find((item) => item.id === id)!, read_at: "2026-10-20T10:01:00Z" };
    records = records.map((item) => item.id === id ? updated : item);
    return updated;
  });
  const onOpenPath = vi.fn();
  const onOpenWork = vi.fn();
  const view = render(<Harness loadPage={loadPage} markRead={markRead} onOpenPath={onOpenPath} onOpenWork={onOpenWork}/>);
  return { ...view, loadPage, markRead, onOpenPath, onOpenWork };
}

beforeEach(() => {
  vi.mocked(loadNotificationPreferences).mockResolvedValue({ tenant_id: "tenant", principal_id: "actor", daily_digest_enabled: true, digest_minute: 480, time_zone: "UTC", quiet_hours_enabled: false, quiet_start_minute: 1200, quiet_end_minute: 420, critical_email_required: true, version: 2 });
  vi.mocked(saveNotificationPreferences).mockImplementation(async (value) => ({ ...value, tenant_id: "tenant", principal_id: "actor", critical_email_required: true, version: value.version + 1 }));
});
afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); });

describe("Notifications workspace", () => {
  it("selects context without a command and opens the exact canonical record separately", async () => {
    const { markRead, onOpenPath } = setup();
    const list = await screen.findByRole("list", { name: "Notification items" });
    fireEvent.click(within(list).getByRole("button", { name: /Mention in supplier review/ }));
    const detail = screen.getByRole("complementary", { name: "Selected notification" });
    expect(within(detail).getByRole("heading", { name: "Mention in supplier review" })).toBeTruthy();
    expect(markRead).not.toHaveBeenCalled();
    expect(onOpenPath).not.toHaveBeenCalled();
    fireEvent.click(within(detail).getByRole("button", { name: "Open record" }));
    expect(onOpenPath).toHaveBeenCalledExactlyOnceWith("#work/matters/issue-2");
    await waitFor(() => expect(markRead).toHaveBeenCalledTimes(1));
    expect(markRead.mock.calls[0]?.[0]).toBe("notice-2");
  });

  it("retains the selected read receipt after it leaves the unread list", async () => {
    const { markRead } = setup();
    await screen.findByRole("list", { name: "Notification items" });
    fireEvent.click(screen.getByRole("button", { name: "Unread (2)" }));
    await screen.findByRole("list", { name: "Notification items" });
    const detail = screen.getByRole("complementary", { name: "Selected notification" });
    fireEvent.click(within(detail).getByRole("button", { name: "Mark as read" }));
    await within(detail).findByRole("button", { name: "Marked as read" });
    expect(markRead).toHaveBeenCalledTimes(1);
    expect(within(detail).getByRole("heading", { name: "Review service continuity" })).toBeTruthy();
    expect(within(screen.getByRole("list", { name: "Notification items" })).queryByRole("button", { name: /Review service continuity/ })).toBeNull();
    expect(await screen.findByRole("button", { name: "Unread (1)" })).toBeTruthy();
  });

  it("finds only loaded rows without issuing a different server query or retaining unrelated context", async () => {
    const { loadPage } = setup();
    await screen.findByRole("list", { name: "Notification items" });
    fireEvent.change(screen.getByRole("searchbox", { name: "Find in loaded notifications" }), { target: { value: "supplier" } });
    const list = screen.getByRole("list", { name: "Notification items" });
    expect(within(list).getAllByRole("button")).toHaveLength(1);
    expect(within(screen.getByRole("complementary", { name: "Selected notification" })).getByRole("heading", { name: "Mention in supplier review" })).toBeTruthy();
    expect(loadPage).toHaveBeenCalledTimes(1);
    fireEvent.change(screen.getByRole("searchbox", { name: "Find in loaded notifications" }), { target: { value: "no matching text" } });
    expect(screen.getByRole("heading", { name: "No loaded notifications match" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Select a notification" })).toBeTruthy();
  });

  it("can record a read receipt for an unsafe destination while refusing its navigation", async () => {
    const { markRead, onOpenPath } = setup({ records: [{ ...items[0]!, action_path: "https://untrusted.example/redirect" }] });
    await screen.findByRole("list", { name: "Notification items" });
    const detail = screen.getByRole("complementary", { name: "Selected notification" });
    expect(within(detail).getByRole("button", { name: "Open record" }).getAttribute("data-disabled")).not.toBeNull();
    fireEvent.click(within(detail).getByRole("button", { name: "Mark as read" }));
    await within(detail).findByRole("button", { name: "Marked as read" });
    expect(markRead).toHaveBeenCalledTimes(1);
    expect(onOpenPath).not.toHaveBeenCalled();
  });

  it("retains an unsaved email draft across local views and preserves the supported preference version", async () => {
    setup();
    fireEvent.click(screen.getByRole("tab", { name: "Preferences" }));
    const zone = await screen.findByRole("textbox", { name: "Time zone" });
    fireEvent.change(zone, { target: { value: "Africa/Lagos" } });
    fireEvent.click(screen.getByRole("tab", { name: "Inbox" }));
    expect(screen.queryByRole("textbox", { name: "Time zone" })).toBeNull();
    fireEvent.click(screen.getByRole("tab", { name: "Activity" }));
    expect(screen.getByText("Activity access is restricted")).toBeTruthy();
    fireEvent.click(screen.getByRole("tab", { name: "Preferences" }));
    expect((screen.getByRole("textbox", { name: "Time zone" }) as HTMLInputElement).value).toBe("Africa/Lagos");
    fireEvent.click(screen.getByRole("button", { name: "Save email preferences" }));
    await waitFor(() => expect(saveNotificationPreferences).toHaveBeenCalledWith(expect.objectContaining({ time_zone: "Africa/Lagos", version: 2 })));
    expect(loadNotificationPreferences).toHaveBeenCalledTimes(1);
  });

  it("moves focus to mobile inline context on selection and back to the chosen row", async () => {
    const original = window.matchMedia.bind(window);
    vi.stubGlobal("matchMedia", (query: string) => query === "(max-width: 900px)" ? { ...original(query), matches: true } : original(query));
    const scroll = vi.spyOn(HTMLElement.prototype, "scrollIntoView").mockImplementation(() => {});
    setup();
    const list = await screen.findByRole("list", { name: "Notification items" });
    const selectedRow = within(list).getByRole("button", { name: /Mention in supplier review/ });
    fireEvent.click(selectedRow);
    await waitFor(() => expect(document.activeElement).toBe(screen.getByRole("complementary", { name: "Selected notification" })));
    fireEvent.click(screen.getByRole("button", { name: "Back to notifications" }));
    await waitFor(() => expect(document.activeElement).toBe(selectedRow));
    expect(scroll).toHaveBeenCalledTimes(2);
  });

  it("keeps unavailable history distinct from a clear inbox and can recover", async () => {
    const loadPage = vi.fn().mockRejectedValueOnce(new Error("offline")).mockResolvedValueOnce({ items: [], unread_count: 0, as_of: "2026-10-20T10:00:00Z" });
    setup({ loadPage });
    expect(await screen.findByRole("heading", { name: "Notifications unavailable" })).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "No notifications loaded" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Retry notifications" }));
    expect(await screen.findByRole("heading", { name: "No notifications loaded" })).toBeTruthy();
  });
});
