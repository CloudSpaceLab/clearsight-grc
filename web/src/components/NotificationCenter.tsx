import { useEffect, useState } from "react";
import { loadNotifications, markNotificationRead, type InAppNotification, type NotificationPage } from "../notificationApi";
import { NotificationPreferences } from "./NotificationPreferences";
import { Button, EmptyState, FocusedSheet, IconButton, Notice, StatusBadge } from "./ui";
import "../notification-center.css";

type LoadState = "loading" | "live" | "unavailable";

export function NotificationCenter({
  loadPage = loadNotifications,
  markRead = markNotificationRead,
  onOpenPath = openNotificationPath,
}: {
  loadPage?: typeof loadNotifications;
  markRead?: typeof markNotificationRead;
  onOpenPath?: (path: string) => void;
}) {
  const [state, setState] = useState<LoadState>("loading");
  const [page, setPage] = useState<NotificationPage | null>(null);
  const [items, setItems] = useState<InAppNotification[]>([]);
  const [open, setOpen] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [loadMoreFailed, setLoadMoreFailed] = useState(false);

  async function loadFirstPage() {
    setState("loading");
    try {
      const next = await loadPage({ limit: 25 });
      setPage(next);
      setItems(next.items);
      setState("live");
    } catch {
      setPage(null);
      setItems([]);
      setState("unavailable");
    }
  }

  useEffect(() => { void loadFirstPage(); }, []);

  async function loadMore() {
    if (!page?.next_cursor || loadingMore) return;
    setLoadingMore(true);
    setLoadMoreFailed(false);
    try {
      const next = await loadPage({ cursor: page.next_cursor, limit: 25 });
      setItems((current) => mergeNotifications(current, next.items));
      setPage(next);
    } catch {
      setLoadMoreFailed(true);
    } finally {
      setLoadingMore(false);
    }
  }

  function openItem(item: InAppNotification) {
    if (!safeNotificationPath(item.action_path)) return;
    if (!item.read_at) {
      const readAt = new Date().toISOString();
      setItems((current) => current.map((candidate) => candidate.id === item.id ? { ...candidate, read_at: readAt } : candidate));
      setPage((current) => current ? { ...current, unread_count: Math.max(0, current.unread_count - 1) } : current);
      void markRead(item.id).then((updated) => {
        setItems((current) => current.map((candidate) => candidate.id === updated.id ? updated : candidate));
      }).catch(() => {
        setItems((current) => current.map((candidate) => candidate.id === item.id ? { ...candidate, read_at: undefined } : candidate));
        setPage((current) => current ? { ...current, unread_count: current.unread_count + 1 } : current);
      });
    }
    setOpen(false);
    onOpenPath(item.action_path);
  }

  const unread = page?.unread_count ?? 0;
  const buttonLabel = state === "unavailable"
    ? "Notifications unavailable"
    : unread === 0 ? "Notifications" : `Notifications, ${unread} unread`;

  return <>
    <span className="notification-center__launcher">
      <IconButton aria-label={buttonLabel} variant="quiet" onPress={() => setOpen(true)}>
        <BellGlyph/>
      </IconButton>
      {state === "live" && unread > 0 && <span className="notification-center__count" aria-hidden="true">{unread > 99 ? "99+" : unread}</span>}
    </span>
    {open && <FocusedSheet label="Notifications" onClose={() => setOpen(false)} panelClassName="notification-center__sheet">
      <div className="notification-center">
        <header className="cs-sheet-heading">
          <span className="eyebrow">Attention</span>
          <h2>Notifications</h2>
          <p>Recent work changes delivered to your signed-in account.</p>
        </header>

        {state === "loading" && <div className="workspace-loading" aria-live="polite" aria-busy="true">Loading notifications…</div>}
        {state === "unavailable" && <EmptyState
          population="Notifications"
          title="Notifications are unavailable"
          description="Assigned work is unchanged. Retry this delivery view."
          action={<Button onPress={() => void loadFirstPage()}>Try again</Button>}
          role="alert"
        />}
        {state === "live" && items.length === 0 && <EmptyState
          population="Notifications"
          title="No recent notifications"
          description="New assignments and mentions will appear here."
        />}
        {state === "live" && items.length > 0 && <div className="notification-center__list" role="list">
          {items.map((item) => <article className={`notification-center__item${item.read_at ? "" : " is-unread"}`} key={item.id} role="listitem">
            <div className="notification-center__item-copy">
              <div className="notification-center__item-heading">
                <strong>{item.title}</strong>
                {!item.read_at && <StatusBadge tone="info">Unread</StatusBadge>}
              </div>
              <p>{item.summary}</p>
              <time dateTime={item.occurred_at}>{formatNotificationTime(item.occurred_at)}</time>
            </div>
            <Button variant={item.read_at ? "secondary" : "primary"} onPress={() => openItem(item)} isDisabled={!safeNotificationPath(item.action_path)}>Open</Button>
          </article>)}
        </div>}
        {state === "live" && loadMoreFailed && <Notice tone="warning">More notifications could not be loaded. The current list remains available.</Notice>}
        {state === "live" && page?.next_cursor && <div className="notification-center__more">
          <Button onPress={() => void loadMore()} isLoading={loadingMore}>Load more</Button>
        </div>}
      </div>
    </FocusedSheet>}
  </>;
}

function mergeNotifications(current: InAppNotification[], next: InAppNotification[]) {
  const seen = new Set(current.map((item) => item.id));
  return [...current, ...next.filter((item) => !seen.has(item.id))];
}

function safeNotificationPath(value: string) {
  return /^#(?:oversight|work|programs|vendors|ropa|forms|people|configure|imports|reports)(?:[/?#]|$)/.test(value) && !/[\r\n]/.test(value);
}

function openNotificationPath(path: string) {
  if (safeNotificationPath(path)) window.location.hash = path.slice(1);
}

function formatNotificationTime(value: string) {
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) return "Time unavailable";
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(parsed));
}


function BellGlyph() {
  return <svg className="compact-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9"/>
    <path d="M10 21h4"/>
  </svg>;
}
