import { useEffect, useMemo, useRef, useState } from "react";
import { loadNotifications, markNotificationRead, type InAppNotification, type NotificationPage } from "../notificationApi";

export type NotificationInboxOptions = {
  scopeKey?: string;
  loadPage?: typeof loadNotifications;
  markRead?: typeof markNotificationRead;
};

export type NotificationInbox = {
  state: "loading" | "live" | "unavailable";
  items: InAppNotification[];
  unreadCount: number | undefined;
  asOf: string;
  nextCursor: string;
  filter: "all" | "unread";
  setFilter: (filter: "all" | "unread") => void;
  refresh: () => void;
  loadMore: () => Promise<void>;
  loadingMore: boolean;
  loadMoreFailed: boolean;
  readingIDs: readonly string[];
  readError: string;
  markRead: (id: string) => Promise<boolean>;
};

type ReadCommand = { controller: AbortController; promise: Promise<boolean> };
type Owner = { scopeKey?: string; active: boolean; mutation: number; reads: Map<string, ReadCommand> };
type Query = { owner: Owner; filter: NotificationInbox["filter"]; controller?: AbortController; paging: boolean };
type InboxData = Pick<NotificationInbox, "state" | "items" | "unreadCount" | "asOf" | "nextCursor" | "loadingMore" | "loadMoreFailed"> & { query: Query };
type CommandData = Pick<NotificationInbox, "readingIDs" | "readError"> & { owner: Owner };

/** One verified inbox owner is shared by the header and the Notifications workspace. */
export function useNotificationInbox({ scopeKey, loadPage = loadNotifications, markRead: persistRead = markNotificationRead }: NotificationInboxOptions): NotificationInbox {
  const owner = useMemo<Owner>(() => ({ scopeKey, active: false, mutation: 0, reads: new Map() }), [scopeKey, loadPage, persistRead]);
  const [selection, setSelection] = useState<{ owner: Owner; filter: NotificationInbox["filter"] }>({ owner, filter: "all" });
  const filter = selection.owner === owner ? selection.filter : "all";
  const [refreshKey, setRefreshKey] = useState(0);
  const query = useMemo<Query>(() => ({ owner, filter, paging: false }), [owner, filter, refreshKey]);
  const [data, setData] = useState<InboxData>(() => emptyData(query));
  const [commands, setCommands] = useState<CommandData>({ owner, readingIDs: [], readError: "" });
  const currentOwner = useRef(owner);
  const currentQuery = useRef(query);
  currentOwner.current = owner;
  currentQuery.current = query;

  // The render identity also hides old data before effect cleanup on an actor/entity change.
  const visible = data.query === query ? data : emptyData(query);
  const visibleCommands = commands.owner === owner ? commands : { readingIDs: [], readError: "" };

  function isOwnerCurrent() {
    return Boolean(owner.scopeKey) && owner.active && currentOwner.current === owner;
  }

  function isQueryCurrent(controller: AbortController) {
    return isOwnerCurrent() && currentQuery.current === query && query.controller === controller && !controller.signal.aborted;
  }

  async function readPage(controller: AbortController, cursor?: string): Promise<NotificationPage | undefined> {
    while (isQueryCurrent(controller)) {
      // A page that overlaps a read command cannot safely supply the global unread count.
      // Wait for those commands and re-read that page instead of guessing its snapshot order.
      await Promise.all([...owner.reads.values()].map((command) => command.promise));
      if (!isQueryCurrent(controller)) return;
      const mutation = owner.mutation;
      try {
        const page = await loadPage({ limit: 25, unreadOnly: filter === "unread", ...(cursor ? { cursor } : {}), signal: controller.signal });
        if (!isQueryCurrent(controller)) return;
        if (mutation !== owner.mutation || owner.reads.size > 0) continue;
        if (!Array.isArray(page?.items) || !Number.isSafeInteger(page.unread_count) || page.unread_count < 0 ||
          typeof page.as_of !== "string" || !Number.isFinite(Date.parse(page.as_of))) throw new Error("Invalid notification page");
        return page;
      } catch (error) {
        if (!isQueryCurrent(controller)) return;
        if (mutation !== owner.mutation || owner.reads.size > 0) continue;
        throw error;
      }
    }
  }

  useEffect(() => {
    owner.active = true;
    return () => {
      owner.active = false;
      for (const command of owner.reads.values()) command.controller.abort();
      owner.reads.clear();
    };
  }, [owner]);

  useEffect(() => {
    const controller = new AbortController();
    query.controller = controller;
    setData(emptyData(query));
    if (scopeKey) {
      void readPage(controller).then((page) => {
        if (!page || !isQueryCurrent(controller)) return;
        setData({ query, state: "live", items: mergeNotifications([], page.items), unreadCount: page.unread_count, asOf: page.as_of, nextCursor: page.next_cursor ?? "", loadingMore: false, loadMoreFailed: false });
      }).catch(() => {
        if (isQueryCurrent(controller)) setData({ ...emptyData(query), state: "unavailable" });
      });
    }
    return () => controller.abort();
  }, [query]);

  function resetQuery() {
    query.controller?.abort();
    setCommands((current) => current.owner === owner ? { ...current, readError: "" } : current);
  }

  function setFilter(nextFilter: NotificationInbox["filter"]) {
    if (!isOwnerCurrent() || nextFilter === filter) return;
    resetQuery();
    setSelection({ owner, filter: nextFilter });
  }

  function refresh() {
    if (!isOwnerCurrent()) return;
    resetQuery();
    setRefreshKey((current) => current + 1);
  }

  async function loadMore() {
    const controller = query.controller;
    if (!controller || !isQueryCurrent(controller) || visible.state !== "live" || !visible.nextCursor || query.paging) return;
    query.paging = true;
    setData((current) => current.query === query ? { ...current, loadingMore: true, loadMoreFailed: false } : current);
    try {
      const page = await readPage(controller, visible.nextCursor);
      if (!page || !isQueryCurrent(controller)) return;
      setData((current) => current.query === query ? {
        ...current,
        items: mergeNotifications(current.items, page.items),
        ...latestUnreadSnapshot(current, page),
        nextCursor: page.next_cursor ?? "",
      } : current);
    } catch {
      if (isQueryCurrent(controller)) setData((current) => current.query === query ? { ...current, loadMoreFailed: true } : current);
    } finally {
      query.paging = false;
      if (isQueryCurrent(controller)) setData((current) => current.query === query ? { ...current, loadingMore: false } : current);
    }
  }

  function markRead(id: string): Promise<boolean> {
    if (!isOwnerCurrent() || currentQuery.current !== query) return Promise.resolve(false);
    const pending = owner.reads.get(id);
    if (pending) return pending.promise;
    const item = visible.items.find((candidate) => candidate.id === id);
    if (!item || visible.state !== "live") return Promise.resolve(false);
    if (item.read_at) return Promise.resolve(true);

    const command: ReadCommand = { controller: new AbortController(), promise: Promise.resolve(false) };
    owner.reads.set(id, command);
    owner.mutation += 1;
    setCommands({ owner, readingIDs: [...owner.reads.keys()], readError: "" });
    command.promise = (async () => {
      try {
        const updated = await persistRead(id, command.controller.signal);
        if (!isOwnerCurrent() || command.controller.signal.aborted) return false;
        if (updated?.id !== id || !updated.read_at) throw new Error("Notification was not marked as read");
        setData((current) => {
          if (current.query !== query) return current;
          return {
            ...current,
            items: current.items.map((candidate) => candidate.id === id ? { ...candidate, read_at: updated.read_at } : candidate),
            // An idempotent receipt does not say whether this command reduced the count.
            unreadCount: undefined,
          };
        });
      } catch {
        if (isOwnerCurrent() && !command.controller.signal.aborted && currentQuery.current === query) {
          setCommands((current) => current.owner === owner ? { ...current, readError: "Could not mark the notification as read. Try again." } : current);
        }
        return false;
      } finally {
        if (owner.reads.get(id) === command) {
          owner.reads.delete(id);
          owner.mutation += 1;
          if (isOwnerCurrent()) setCommands((current) => ({ owner, readingIDs: [...owner.reads.keys()], readError: current.owner === owner ? current.readError : "" }));
        }
      }

      const controller = query.controller;
      if (controller && isQueryCurrent(controller)) {
        try {
          const page = await readPage(controller);
          if (page && isQueryCurrent(controller)) {
            setData((current) => current.query === query ? { ...current, ...latestUnreadSnapshot(current, page) } : current);
          }
        } catch {
          if (isQueryCurrent(controller)) setCommands((current) => ({
            owner,
            readingIDs: current.owner === owner ? current.readingIDs : [],
            readError: "Notification marked as read. Refresh the inbox to check the unread count.",
          }));
        }
      }
      return isOwnerCurrent();
    })();
    return command.promise;
  }

  return {
    state: visible.state,
    items: visible.items,
    unreadCount: visible.unreadCount,
    asOf: visible.asOf,
    nextCursor: visible.nextCursor,
    loadingMore: visible.loadingMore,
    loadMoreFailed: visible.loadMoreFailed,
    readingIDs: visibleCommands.readingIDs,
    readError: visibleCommands.readError,
    filter, setFilter, refresh, loadMore, markRead,
  };
}

function emptyData(query: Query): InboxData {
  return { query, state: query.owner.scopeKey ? "loading" : "unavailable", items: [], unreadCount: undefined, asOf: "", nextCursor: "", loadingMore: false, loadMoreFailed: false };
}

function latestUnreadSnapshot(current: InboxData, page: NotificationPage): Pick<InboxData, "unreadCount" | "asOf"> {
  // Paging and a read-receipt count refresh may finish out of order. Preserve
  // the newest server snapshot without reordering the loaded records or cursor.
  return Date.parse(page.as_of) < Date.parse(current.asOf)
    ? { unreadCount: current.unreadCount, asOf: current.asOf }
    : { unreadCount: page.unread_count, asOf: page.as_of };
}

function mergeNotifications(current: InAppNotification[], next: InAppNotification[]): InAppNotification[] {
  const merged = [...current];
  const positions = new Map(current.map((item, index) => [item.id, index]));
  for (const item of next) {
    const position = positions.get(item.id);
    if (position === undefined) {
      positions.set(item.id, merged.length);
      merged.push(item);
    } else {
      const currentItem = merged[position];
      if (currentItem && !currentItem.read_at && item.read_at) merged[position] = { ...currentItem, read_at: item.read_at };
    }
  }
  return merged;
}
