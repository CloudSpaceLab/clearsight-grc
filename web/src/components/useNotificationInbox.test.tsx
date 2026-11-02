import { StrictMode, type ReactNode } from "react";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { InAppNotification, NotificationPage } from "../notificationApi";
import { useNotificationInbox, type NotificationInbox } from "./useNotificationInbox";

function item(id = "notification-1", overrides: Partial<InAppNotification> = {}): InAppNotification {
  return {
    id,
    kind: "MATTER_OWNER_ASSIGNED",
    title: "Review account access",
    summary: "An access review was assigned to you.",
    subject_type: "MATTER",
    subject_id: "matter-1",
    action_path: "#work/matters/matter-1",
    occurred_at: "2026-10-02T09:00:00Z",
    ...overrides,
  };
}

function page(overrides: Partial<NotificationPage> = {}): NotificationPage {
  return { items: [item()], unread_count: 7, as_of: "2026-10-02T09:05:00Z", ...overrides };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}

const scope = "tenant-1:entity-1:actor-1";
const readAt = "2026-10-02T09:06:00Z";

describe("useNotificationInbox", () => {
  it("does not query or manufacture an unread count before scope verification", async () => {
    const loadPage = vi.fn().mockResolvedValue(page());
    const markRead = vi.fn();
    const { result } = renderHook(() => useNotificationInbox({ loadPage, markRead }));

    expect(result.current.state).toBe("unavailable");
    expect(result.current.items).toEqual([]);
    expect(result.current.unreadCount).toBeUndefined();
    act(() => { result.current.refresh(); result.current.setFilter("unread"); });
    await act(async () => {
      await result.current.loadMore();
      expect(await result.current.markRead("notification-1")).toBe(false);
    });
    expect(loadPage).not.toHaveBeenCalled();
    expect(markRead).not.toHaveBeenCalled();
  });

  it("keeps the server unread total and deduplicates keyset pages without estimating from visible rows", async () => {
    const loadPage = vi.fn()
      .mockResolvedValueOnce(page({ next_cursor: "page-2" }))
      .mockResolvedValueOnce(page({
        items: [item(), item("notification-2"), item("notification-2")],
        unread_count: 11,
        as_of: "2026-10-02T09:07:00Z",
      }));
    const { result } = renderHook(() => useNotificationInbox({ scopeKey: scope, loadPage }));
    await waitFor(() => expect(result.current.state).toBe("live"));
    expect(result.current.unreadCount).toBe(7);
    expect(result.current.items).toHaveLength(1);
    expect(loadPage).toHaveBeenNthCalledWith(1, { limit: 25, unreadOnly: false, signal: expect.any(AbortSignal) });

    await act(async () => { await result.current.loadMore(); });
    expect(loadPage).toHaveBeenNthCalledWith(2, { cursor: "page-2", limit: 25, unreadOnly: false, signal: expect.any(AbortSignal) });
    expect(result.current.items.map((row) => row.id)).toEqual(["notification-1", "notification-2"]);
    expect(result.current.unreadCount).toBe(11);
    expect(result.current.asOf).toBe("2026-10-02T09:07:00Z");
    expect(result.current.nextCursor).toBe("");
  });

  it("distinguishes unavailable from a confirmed empty inbox and refreshes after failure", async () => {
    const loadPage = vi.fn().mockRejectedValueOnce(new Error("unavailable")).mockResolvedValueOnce(page({ items: [], unread_count: 0 }));
    const { result } = renderHook(() => useNotificationInbox({ scopeKey: scope, loadPage }));
    await waitFor(() => expect(result.current.state).toBe("unavailable"));
    expect(result.current.unreadCount).toBeUndefined();
    expect(result.current.asOf).toBe("");

    act(() => result.current.refresh());
    expect(result.current.state).toBe("loading");
    await waitFor(() => expect(result.current.state).toBe("live"));
    expect(result.current.items).toEqual([]);
    expect(result.current.unreadCount).toBe(0);
    expect(result.current.asOf).toBe("2026-10-02T09:05:00Z");
  });

  it("does not display a missing unread total as zero", async () => {
    const loadPage = vi.fn().mockResolvedValue({ items: [], as_of: "2026-10-02T09:05:00Z" });
    const { result } = renderHook(() => useNotificationInbox({ scopeKey: scope, loadPage }));
    await waitFor(() => expect(result.current.state).toBe("unavailable"));
    expect(result.current.unreadCount).toBeUndefined();
  });

  it.each(["", "not-a-time", undefined])("does not claim a current inbox count without a valid snapshot time: %s", async (as_of) => {
    const loadPage = vi.fn().mockResolvedValue({ ...page(), as_of });
    const { result } = renderHook(() => useNotificationInbox({ scopeKey: scope, loadPage }));
    await waitFor(() => expect(result.current.state).toBe("unavailable"));
    expect(result.current.unreadCount).toBeUndefined();
    expect(result.current.asOf).toBe("");
  });

  it("preserves the current page and cursor on paging failure, then retries that cursor once", async () => {
    const retry = deferred<NotificationPage>();
    const loadPage = vi.fn()
      .mockResolvedValueOnce(page({ next_cursor: "page-2" }))
      .mockRejectedValueOnce(new Error("network"))
      .mockReturnValueOnce(retry.promise);
    const { result } = renderHook(() => useNotificationInbox({ scopeKey: scope, loadPage }));
    await waitFor(() => expect(result.current.state).toBe("live"));
    await act(async () => { await result.current.loadMore(); });
    expect(result.current.loadMoreFailed).toBe(true);
    expect(result.current.items.map((row) => row.id)).toEqual(["notification-1"]);
    expect(result.current.unreadCount).toBe(7);
    expect(result.current.nextCursor).toBe("page-2");

    let loading!: Promise<void>;
    act(() => { loading = result.current.loadMore(); void result.current.loadMore(); });
    await waitFor(() => expect(loadPage).toHaveBeenCalledTimes(3));
    expect(result.current.loadingMore).toBe(true);
    await act(async () => { retry.resolve(page({ items: [item("notification-2")] })); await loading; });
    expect(result.current.items).toHaveLength(2);
    expect(result.current.loadMoreFailed).toBe(false);
    expect(result.current.loadingMore).toBe(false);
    expect(loadPage.mock.calls.slice(1).map(([request]) => request.cursor)).toEqual(["page-2", "page-2"]);
  });

  it("clears protected data during the first render of a new scope and ignores an old response", async () => {
    const oldPage = deferred<NotificationPage>();
    const nextScope = deferred<NotificationPage>();
    const loadPage = vi.fn().mockResolvedValueOnce(page({ next_cursor: "page-2" })).mockReturnValueOnce(oldPage.promise).mockReturnValueOnce(nextScope.promise);
    const renders: { scopeKey: string; inbox: NotificationInbox }[] = [];
    const { result, rerender } = renderHook(({ scopeKey }) => {
      const inbox = useNotificationInbox({ scopeKey, loadPage });
      renders.push({ scopeKey, inbox });
      return inbox;
    }, { initialProps: { scopeKey: scope } });
    await waitFor(() => expect(result.current.state).toBe("live"));
    let loading!: Promise<void>;
    act(() => { loading = result.current.loadMore(); });
    await waitFor(() => expect(loadPage).toHaveBeenCalledTimes(2));
    expect(result.current.items).toHaveLength(1);
    const oldSignal = loadPage.mock.calls[1]![0].signal as AbortSignal;

    rerender({ scopeKey: "tenant-1:entity-2:actor-1" });
    const firstNewRender = renders.find((render) => render.scopeKey === "tenant-1:entity-2:actor-1")!.inbox;
    expect(firstNewRender.items).toEqual([]);
    expect(firstNewRender.unreadCount).toBeUndefined();
    expect(firstNewRender.asOf).toBe("");
    expect(firstNewRender.filter).toBe("all");
    expect(oldSignal.aborted).toBe(true);
    await waitFor(() => expect(loadPage).toHaveBeenCalledTimes(3));
    await act(async () => { nextScope.resolve(page({ items: [item("entity-2-row")], unread_count: 2 })); });
    await waitFor(() => expect(result.current.state).toBe("live"));
    await act(async () => { oldPage.resolve(page({ items: [item("private-old-row")], unread_count: 99 })); await loading; });
    expect(result.current.items.map((row) => row.id)).toEqual(["entity-2-row"]);
    expect(result.current.unreadCount).toBe(2);
  });

  it("aborts page and read requests on scope revocation and ignores their eventual results", async () => {
    const more = deferred<NotificationPage>();
    const read = deferred<InAppNotification>();
    const loadPage = vi.fn().mockResolvedValueOnce(page({ next_cursor: "page-2" })).mockReturnValueOnce(more.promise);
    const markRead = vi.fn().mockReturnValue(read.promise);
    const { result, rerender } = renderHook(({ scopeKey }: { scopeKey?: string }) => useNotificationInbox({ scopeKey, loadPage, markRead }), { initialProps: { scopeKey: scope as string | undefined } });
    await waitFor(() => expect(result.current.state).toBe("live"));
    let loading!: Promise<void>;
    let reading!: Promise<boolean>;
    act(() => { loading = result.current.loadMore(); });
    await waitFor(() => expect(loadPage).toHaveBeenCalledTimes(2));
    act(() => { reading = result.current.markRead("notification-1"); });
    expect(result.current.readingIDs).toEqual(["notification-1"]);

    rerender({ scopeKey: undefined });
    expect(result.current.state).toBe("unavailable");
    expect(result.current.items).toEqual([]);
    expect(result.current.unreadCount).toBeUndefined();
    expect(result.current.readingIDs).toEqual([]);
    expect((loadPage.mock.calls[1]![0].signal as AbortSignal).aborted).toBe(true);
    expect((markRead.mock.calls[0]![1] as AbortSignal).aborted).toBe(true);
    await act(async () => {
      more.resolve(page({ items: [item("private-old-row")] }));
      read.resolve(item("notification-1", { read_at: readAt }));
      await loading;
      expect(await reading).toBe(false);
    });
    expect(result.current.items).toEqual([]);
    expect(result.current.unreadCount).toBeUndefined();
    expect(loadPage).toHaveBeenCalledTimes(2);
  });

  it("resets pagination on the server unread filter, ignores an older page, and refreshes the selected filter", async () => {
    const oldMore = deferred<NotificationPage>();
    const loadPage = vi.fn()
      .mockResolvedValueOnce(page({ next_cursor: "all-page-2" }))
      .mockReturnValueOnce(oldMore.promise)
      .mockResolvedValueOnce(page({ items: [item("unread-row")], unread_count: 3, next_cursor: "unread-page-2" }))
      .mockResolvedValueOnce(page({ items: [item("refreshed-unread")], unread_count: 2 }));
    const { result } = renderHook(() => useNotificationInbox({ scopeKey: scope, loadPage }));
    await waitFor(() => expect(result.current.state).toBe("live"));
    let loading!: Promise<void>;
    act(() => { loading = result.current.loadMore(); });
    await waitFor(() => expect(loadPage).toHaveBeenCalledTimes(2));
    act(() => result.current.setFilter("unread"));
    expect(result.current.items).toEqual([]);
    expect(result.current.nextCursor).toBe("");
    await waitFor(() => expect(result.current.state).toBe("live"));
    expect(loadPage).toHaveBeenNthCalledWith(3, { limit: 25, unreadOnly: true, signal: expect.any(AbortSignal) });
    await act(async () => { oldMore.resolve(page({ items: [item("old-all-row")], unread_count: 99 })); await loading; });
    expect(result.current.items.map((row) => row.id)).toEqual(["unread-row"]);
    expect(result.current.nextCursor).toBe("unread-page-2");
    expect(result.current.unreadCount).toBe(3);

    act(() => result.current.refresh());
    await waitFor(() => expect(result.current.items[0]?.id).toBe("refreshed-unread"));
    expect(result.current.filter).toBe("unread");
    expect(loadPage).toHaveBeenNthCalledWith(4, { limit: 25, unreadOnly: true, signal: expect.any(AbortSignal) });
  });

  it("waits for read confirmation, coalesces duplicate commands, and keeps the selected row after reading", async () => {
    const read = deferred<InAppNotification>();
    const loadPage = vi.fn().mockResolvedValueOnce(page()).mockResolvedValueOnce(page()).mockResolvedValue(page({ unread_count: 6 }));
    const markRead = vi.fn().mockReturnValue(read.promise);
    const { result } = renderHook(() => useNotificationInbox({ scopeKey: scope, loadPage, markRead }));
    await waitFor(() => expect(result.current.state).toBe("live"));
    act(() => result.current.setFilter("unread"));
    await waitFor(() => expect(result.current.state).toBe("live"));
    let first!: Promise<boolean>;
    let second!: Promise<boolean>;
    act(() => { first = result.current.markRead("notification-1"); second = result.current.markRead("notification-1"); });
    expect(first).toBe(second);
    expect(markRead).toHaveBeenCalledTimes(1);
    expect(markRead).toHaveBeenCalledWith("notification-1", expect.any(AbortSignal));
    expect(result.current.readingIDs).toEqual(["notification-1"]);
    expect(result.current.items[0]!.read_at).toBeUndefined();
    expect(result.current.unreadCount).toBe(7);

    await act(async () => { read.resolve(item("notification-1", { read_at: readAt })); expect(await first).toBe(true); });
    expect(result.current.items).toHaveLength(1);
    expect(result.current.items[0]!.read_at).toBe(readAt);
    expect(result.current.unreadCount).toBe(6);
    expect(result.current.readingIDs).toEqual([]);
    await act(async () => {
      expect(await result.current.markRead("notification-1")).toBe(true);
      expect(await result.current.markRead("not-loaded")).toBe(false);
    });
    expect(markRead).toHaveBeenCalledTimes(1);
  });

  it("preserves unread data after a failed read and retries without decrementing twice", async () => {
    const loadPage = vi.fn().mockResolvedValueOnce(page()).mockResolvedValue(page({ unread_count: 6 }));
    const markRead = vi.fn().mockRejectedValueOnce(new Error("unavailable")).mockResolvedValueOnce(item("notification-1", { read_at: readAt }));
    const { result } = renderHook(() => useNotificationInbox({ scopeKey: scope, loadPage, markRead }));
    await waitFor(() => expect(result.current.state).toBe("live"));
    await act(async () => { expect(await result.current.markRead("notification-1")).toBe(false); });
    expect(result.current.unreadCount).toBe(7);
    expect(result.current.items[0]!.read_at).toBeUndefined();
    expect(result.current.readError).toBe("Could not mark the notification as read. Try again.");
    await act(async () => { expect(await result.current.markRead("notification-1")).toBe(true); });
    expect(result.current.unreadCount).toBe(6);
    expect(result.current.readError).toBe("");
  });

  it("rejects a read response for a different notification", async () => {
    const loadPage = vi.fn().mockResolvedValue(page());
    const markRead = vi.fn().mockResolvedValue(item("different-id", { read_at: readAt }));
    const { result } = renderHook(() => useNotificationInbox({ scopeKey: scope, loadPage, markRead }));
    await waitFor(() => expect(result.current.state).toBe("live"));
    await act(async () => { expect(await result.current.markRead("notification-1")).toBe(false); });
    expect(result.current.items[0]!.read_at).toBeUndefined();
    expect(result.current.unreadCount).toBe(7);
    expect(result.current.readError).not.toBe("");
  });

  it("reconciles a page overlapping a read command and never restores its stale unread metadata", async () => {
    const olderPage = deferred<NotificationPage>();
    const countPage = deferred<NotificationPage>();
    const reconciledPage = deferred<NotificationPage>();
    const loadPage = vi.fn().mockResolvedValueOnce(page({ next_cursor: "page-2" })).mockReturnValueOnce(olderPage.promise).mockReturnValueOnce(countPage.promise).mockReturnValueOnce(reconciledPage.promise);
    const markRead = vi.fn().mockResolvedValue(item("notification-1", { read_at: readAt }));
    const { result } = renderHook(() => useNotificationInbox({ scopeKey: scope, loadPage, markRead }));
    await waitFor(() => expect(result.current.state).toBe("live"));
    let loading!: Promise<void>;
    let reading!: Promise<boolean>;
    act(() => { loading = result.current.loadMore(); });
    await waitFor(() => expect(loadPage).toHaveBeenCalledTimes(2));
    act(() => { reading = result.current.markRead("notification-1"); });
    await waitFor(() => expect(loadPage).toHaveBeenCalledTimes(3));
    expect(result.current.unreadCount).toBeUndefined();
    await act(async () => { countPage.resolve(page({ unread_count: 6 })); expect(await reading).toBe(true); });
    expect(result.current.unreadCount).toBe(6);
    await act(async () => { olderPage.resolve(page({ items: [item(), item("notification-2")], unread_count: 7 })); });
    await waitFor(() => expect(loadPage).toHaveBeenCalledTimes(4));
    expect(result.current.unreadCount).toBe(6);
    expect(result.current.items).toHaveLength(1);
    await act(async () => {
      reconciledPage.resolve(page({ items: [item(), item("notification-2")], unread_count: 6 }));
      await loading;
    });
    expect(result.current.items.map((row) => row.id)).toEqual(["notification-1", "notification-2"]);
    expect(result.current.items[0]!.read_at).toBe(readAt);
    expect(result.current.unreadCount).toBe(6);
    expect(loadPage.mock.calls.slice(1).map(([request]) => request.cursor)).toEqual(["page-2", undefined, "page-2"]);
  });

  it("uses a fresh server count when a different session already recorded the read", async () => {
    const count = deferred<NotificationPage>();
    const loadPage = vi.fn()
      .mockResolvedValueOnce(page({ next_cursor: "page-2" }))
      .mockResolvedValueOnce(page({ items: [item("notification-2")], unread_count: 5, as_of: "2026-10-02T09:07:00Z" }))
      .mockReturnValueOnce(count.promise);
    const markRead = vi.fn().mockResolvedValue(item("notification-1", { read_at: readAt }));
    const { result } = renderHook(() => useNotificationInbox({ scopeKey: scope, loadPage, markRead }));
    await waitFor(() => expect(result.current.state).toBe("live"));
    await act(async () => { await result.current.loadMore(); });
    expect(result.current.items[0]!.read_at).toBeUndefined();
    expect(result.current.unreadCount).toBe(5);
    let reading!: Promise<boolean>;
    act(() => { reading = result.current.markRead("notification-1"); });
    await waitFor(() => expect(result.current.items[0]!.read_at).toBe(readAt));
    expect(result.current.unreadCount).toBeUndefined();
    await act(async () => { count.resolve(page({ unread_count: 5, as_of: "2026-10-02T09:08:00Z" })); expect(await reading).toBe(true); });
    expect(result.current.unreadCount).toBe(5);
    expect(result.current.items.map((row) => row.id)).toEqual(["notification-1", "notification-2"]);
    expect(result.current.asOf).toBe("2026-10-02T09:08:00Z");
  });

  it.each(["count refresh", "older page"] as const)("does not replace a newer unread snapshot when a stale %s finishes last", async (last) => {
    const count = deferred<NotificationPage>();
    const older = deferred<NotificationPage>();
    const loadPage = vi.fn().mockResolvedValueOnce(page({ next_cursor: "page-2" })).mockReturnValueOnce(count.promise).mockReturnValueOnce(older.promise);
    const markRead = vi.fn().mockResolvedValue(item("notification-1", { read_at: readAt }));
    const { result } = renderHook(() => useNotificationInbox({ scopeKey: scope, loadPage, markRead }));
    await waitFor(() => expect(result.current.state).toBe("live"));
    let reading!: Promise<boolean>;
    let loading!: Promise<void>;
    act(() => { reading = result.current.markRead("notification-1"); });
    await waitFor(() => expect(loadPage).toHaveBeenCalledTimes(2));
    act(() => { loading = result.current.loadMore(); });
    await waitFor(() => expect(loadPage).toHaveBeenCalledTimes(3));
    const newer = { unread_count: 5, as_of: "2026-10-02T09:08:00Z" };
    const stale = { unread_count: 6, as_of: "2026-10-02T09:07:00Z" };
    if (last === "count refresh") {
      await act(async () => { older.resolve(page({ items: [item("notification-2")], ...newer })); await loading; });
      await act(async () => { count.resolve(page(stale)); await reading; });
    } else {
      await act(async () => { count.resolve(page(newer)); await reading; });
      await act(async () => { older.resolve(page({ items: [item("notification-2")], ...stale })); await loading; });
    }
    expect(result.current.unreadCount).toBe(5);
    expect(result.current.asOf).toBe(newer.as_of);
    expect(result.current.items.map((row) => row.id)).toEqual(["notification-1", "notification-2"]);
    expect(result.current.items[0]!.read_at).toBe(readAt);
    expect(result.current.nextCursor).toBe("");
  });

  it("keeps a confirmed read when refreshing its unread count fails", async () => {
    const loadPage = vi.fn().mockResolvedValueOnce(page()).mockRejectedValueOnce(new Error("unavailable"));
    const markRead = vi.fn().mockResolvedValue(item("notification-1", { read_at: readAt }));
    const { result } = renderHook(() => useNotificationInbox({ scopeKey: scope, loadPage, markRead }));
    await waitFor(() => expect(result.current.state).toBe("live"));
    await act(async () => { expect(await result.current.markRead("notification-1")).toBe(true); });
    expect(result.current.items[0]!.read_at).toBe(readAt);
    expect(result.current.unreadCount).toBeUndefined();
    expect(result.current.state).toBe("live");
    expect(result.current.readError).toBe("Notification marked as read. Refresh the inbox to check the unread count.");
  });

  it("waits for an in-flight read before refreshing a changed filter", async () => {
    const read = deferred<InAppNotification>();
    const loadPage = vi.fn().mockResolvedValueOnce(page()).mockResolvedValueOnce(page({ items: [], unread_count: 6 }));
    const markRead = vi.fn().mockReturnValue(read.promise);
    const { result } = renderHook(() => useNotificationInbox({ scopeKey: scope, loadPage, markRead }));
    await waitFor(() => expect(result.current.state).toBe("live"));
    let reading!: Promise<boolean>;
    act(() => { reading = result.current.markRead("notification-1"); });
    act(() => result.current.setFilter("unread"));
    await act(async () => {});
    expect(loadPage).toHaveBeenCalledTimes(1);
    expect(result.current.state).toBe("loading");
    expect(result.current.unreadCount).toBeUndefined();
    await act(async () => { read.resolve(item("notification-1", { read_at: readAt })); expect(await reading).toBe(true); });
    await waitFor(() => expect(result.current.state).toBe("live"));
    expect(result.current.filter).toBe("unread");
    expect(result.current.items).toEqual([]);
    expect(result.current.unreadCount).toBe(6);
    expect(loadPage).toHaveBeenLastCalledWith({ limit: 25, unreadOnly: true, signal: expect.any(AbortSignal) });
  });

  it("remains usable when React replays mount effects", async () => {
    const loadPage = vi.fn().mockResolvedValue(page());
    const { result } = renderHook(() => useNotificationInbox({ scopeKey: scope, loadPage }), { wrapper: ({ children }: { children: ReactNode }) => <StrictMode>{children}</StrictMode> });
    await waitFor(() => expect(result.current.state).toBe("live"));
    expect(result.current.items[0]!.id).toBe("notification-1");
    expect(result.current.unreadCount).toBe(7);
    expect((loadPage.mock.calls.at(-1)![0].signal as AbortSignal).aborted).toBe(false);
  });
});
