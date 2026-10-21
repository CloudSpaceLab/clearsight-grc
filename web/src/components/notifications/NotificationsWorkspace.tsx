import { useEffect, useMemo, useRef, useState } from "react";
import type { NotificationTab } from "../../appRouting";
import type { InAppNotification } from "../../notificationApi";
import { NotificationPreferences } from "../NotificationPreferences";
import { safeNotificationPath } from "../NotificationCenter";
import type { NotificationInbox } from "../useNotificationInbox";
import { NavigationIcon } from "../NavigationIcon";
import { Button, EmptyState, IconButton, Notice, SearchField, SelectableRecord, StatusBadge, Tabs } from "../ui";
import { NotificationActivity } from "./NotificationActivity";
import "./notifications-workspace.css";

type Props = {
  inbox: NotificationInbox;
  tab: NotificationTab;
  onTab: (tab: NotificationTab) => void;
  legalEntityID?: string;
  legalEntityName: string;
  actorName: string;
  canReadActivity: boolean;
  onOpenWork: () => void;
  onOpenPath?: (path: string) => void;
};

const tabs = [
  { id: "inbox", label: "Inbox" },
  { id: "activity", label: "Activity" },
  { id: "preferences", label: "Preferences" },
] as const;

export function NotificationsWorkspace({ inbox, tab, onTab, legalEntityID, legalEntityName, actorName, canReadActivity, onOpenWork, onOpenPath = openPath }: Props) {
  return <section className="notifications-workspace" aria-label="Notifications workspace">
    <header className="notifications-workspace__header">
      <div>
        <nav aria-label="Notification breadcrumb"><Button variant="quiet" size="compact" onPress={onOpenWork}>Work</Button><span aria-hidden="true">›</span><span>Notifications</span></nav>
        <h1>Notifications</h1>
        <p>Updates delivered to you.</p>
      </div>
      <IconButton aria-label="Notification preferences" variant="quiet" onPress={() => onTab("preferences")}><NavigationIcon view="configure"/></IconButton>
      <svg className="notifications-workspace__waves" viewBox="0 0 720 150" preserveAspectRatio="none" aria-hidden="true"><path d="M0 145C150 145 220 25 334 79s180-66 294-21S691 103 720 38V150H0"/><path d="M0 146C152 130 184 111 292 29S480 155 589 61S687 113 720 100V150H0"/></svg>
    </header>
    <Tabs ariaLabel="Notification views" variant="surface" items={tabs} selectedKey={tab} onSelectionChange={onTab} retainVisitedPanels>
      {(selectedTab) => selectedTab === "inbox"
        ? <NotificationInboxView inbox={inbox} actorName={actorName} legalEntityName={legalEntityName} onOpenWork={onOpenWork} onOpenPath={onOpenPath}/>
        : selectedTab === "activity"
          ? <NotificationActivity canRead={canReadActivity} legalEntityID={legalEntityID} legalEntityName={legalEntityName}/>
          : <NotificationSettings/>}
    </Tabs>
  </section>;
}

function NotificationInboxView({ inbox, actorName, legalEntityName, onOpenWork, onOpenPath }: Pick<Props, "inbox" | "actorName" | "legalEntityName" | "onOpenWork"> & { onOpenPath: (path: string) => void }) {
  const [query, setQuery] = useState("");
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [selectedID, setSelectedID] = useState<string>();
  const [revealSelection, setRevealSelection] = useState(false);
  const detailRef = useRef<HTMLElement>(null);
  const listRef = useRef<HTMLElement>(null);
  const normalizedQuery = query.trim().toLocaleLowerCase();
  const shown = useMemo(() => inbox.items.filter((item) =>
    (inbox.filter === "all" || !item.read_at) &&
    (!normalizedQuery || `${item.title} ${item.summary} ${kindLabel(item.kind)}`.toLocaleLowerCase().includes(normalizedQuery))), [inbox.items, inbox.filter, normalizedQuery]);
  // Keep a selected read receipt visible even when it leaves the Unread list.
  const selected = inbox.items.find((item) => item.id === selectedID) ?? shown[0];

  useEffect(() => { setSelectedID(undefined); }, [inbox.filter, normalizedQuery]);
  useEffect(() => {
    if (!revealSelection || !selected) return;
    setRevealSelection(false);
    if (window.matchMedia("(max-width: 900px)").matches) {
      detailRef.current?.scrollIntoView({ block: "start" });
      detailRef.current?.focus({ preventScroll: true });
    }
  }, [revealSelection, selected]);

  function select(item: InAppNotification) {
    setSelectedID(item.id);
    setRevealSelection(true);
  }

  function backToList() {
    listRef.current?.scrollIntoView({ block: "start" });
    const current = listRef.current?.querySelector<HTMLButtonElement>('[aria-pressed="true"]')
      ?? listRef.current?.querySelector<HTMLButtonElement>(".cs-selectable-record");
    (current ?? listRef.current)?.focus({ preventScroll: true });
  }

  function markSelectedRead(item: InAppNotification) {
    setSelectedID(item.id);
    void inbox.markRead(item.id);
  }

  function openRecord(item: InAppNotification) {
    if (!safeNotificationPath(item.action_path)) return;
    if (!item.read_at) void inbox.markRead(item.id);
    onOpenPath(item.action_path);
  }

  return <div className="notification-inbox">
    <div className="notification-inbox__toolbar">
      <div className="notification-inbox__views" role="group" aria-label="Inbox read filter">
        <Button variant={inbox.filter === "all" ? "primary" : "secondary"} aria-pressed={inbox.filter === "all"} onPress={() => inbox.setFilter("all")}>All</Button>
        <Button variant={inbox.filter === "unread" ? "primary" : "secondary"} aria-pressed={inbox.filter === "unread"} onPress={() => inbox.setFilter("unread")}>Unread{inbox.unreadCount === undefined ? "" : ` (${inbox.unreadCount})`}</Button>
      </div>
      <div className="notification-inbox__actions">
        <IconButton aria-label="Refresh" title="Refresh notifications" onPress={inbox.refresh} isDisabled={inbox.state === "loading"}><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden="true"><path d="M20 4v6h-6M4 20v-6h6M5 9a7 7 0 0 1 12-4l3 5M4 14l3 5a7 7 0 0 0 12-4"/></svg></IconButton>
        <div className="notification-inbox__filter-toggle"><IconButton aria-label={filtersOpen ? "Hide notification filters" : "Show notification filters"} aria-expanded={filtersOpen} aria-controls="notification-filters" onPress={() => setFiltersOpen((value) => !value)}><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden="true"><path d="M3 5h18l-7 8v6l-4 2v-8L3 5z"/></svg></IconButton></div>
      </div>
    </div>
    <div className="notification-inbox__workspace">
      <section ref={listRef} className="notification-inbox__list-panel" aria-label="Notification list" tabIndex={-1}>
        <div id="notification-filters" className="notification-inbox__filters" data-expanded={filtersOpen}>
          <SearchField label="Find in loaded notifications" value={query} onChange={setQuery} placeholder="Find a delivered update"/>
          <p>Searches the notifications loaded below. Load more to include older updates.</p>
        </div>
        <div className="notification-inbox__list-heading"><div><h2>{inbox.filter === "unread" ? "Unread notifications" : "Your inbox"}</h2><p>Assignments, mentions and record updates.</p></div>{inbox.state === "live" && <StatusBadge tone="neutral">{shown.length} shown</StatusBadge>}</div>
        {inbox.state === "loading" && <div className="notification-inbox__loading" role="status" aria-busy="true">Loading notifications…</div>}
        {inbox.state === "unavailable" && <EmptyState population="Your inbox" title="Notifications unavailable" description="Assigned work is unchanged. Retry this delivery view." role="alert" action={<Button onPress={inbox.refresh}>Retry notifications</Button>}/>}
        {inbox.state === "live" && shown.length === 0 && <EmptyState population="Your inbox" title={normalizedQuery ? "No loaded notifications match" : inbox.filter === "unread" ? "No unread notifications loaded" : "No notifications loaded"} description={normalizedQuery ? "Try another phrase or load more notifications." : inbox.nextCursor ? "Load more to review older updates." : "New assignments and mentions will appear here."} action={normalizedQuery ? <Button onPress={() => setQuery("")}>Clear search</Button> : undefined}/>}
        {inbox.state === "live" && shown.length > 0 && <ul className="notification-inbox__list" aria-label="Notification items">
          {shown.map((item) => <li key={item.id} data-unread={!item.read_at || undefined}>
            <SelectableRecord variant="list" title={item.title} description={item.summary} metadata={kindLabel(item.kind)} isSelected={selected?.id === item.id}
              startContent={<span className="notification-symbol"><NotificationSymbol item={item}/></span>}
              endContent={<span className="notification-inbox__row-end"><time dateTime={validTime(item.occurred_at)}>{formatTime(item.occurred_at, true)}</time><StatusBadge tone={item.read_at ? "neutral" : "info"}>{item.read_at ? "Read" : "Unread"}</StatusBadge></span>}
              onPress={() => select(item)}/>
          </li>)}
        </ul>}
        {inbox.loadMoreFailed && <Notice tone="warning">More notifications could not be loaded. The current list remains available.</Notice>}
        {inbox.state === "live" && <div className="notification-inbox__list-footer"><span>{inbox.items.length} loaded{inbox.nextCursor ? " · More available" : ""}</span>{inbox.nextCursor && <Button onPress={() => void inbox.loadMore()} isLoading={inbox.loadingMore}>Load more</Button>}<Button variant="quiet" onPress={onOpenWork}>Open assigned work</Button></div>}
      </section>
      <aside ref={detailRef} className="notification-detail" aria-label="Selected notification" tabIndex={-1}>
        <div className="notification-detail__back"><Button variant="quiet" onPress={backToList}>Back to notifications</Button></div>
        {selected && inbox.state === "live" ? <>
          <header className="notification-detail__header"><span className="notification-symbol"><NotificationSymbol item={selected}/></span><div><p className="eyebrow">{kindLabel(selected.kind)}</p><h2>{selected.title}</h2><p>{formatTime(selected.occurred_at)}</p></div></header>
          <div className="notification-detail__read"><StatusBadge tone={selected.read_at ? "neutral" : "info"}>{selected.read_at ? "Read" : "Unread"}</StatusBadge><Button size="compact" onPress={() => markSelectedRead(selected)} isDisabled={Boolean(selected.read_at)} isLoading={inbox.readingIDs.includes(selected.id)}>{selected.read_at ? "Marked as read" : "Mark as read"}</Button></div>
          {inbox.readError && <Notice tone="warning">{inbox.readError}</Notice>}
          <div className="notification-detail__body">
            <h3>About this update</h3><p>{selected.summary}</p>
            <dl className="notification-detail__facts">
              <div><dt>Recipient</dt><dd>{actorName}</dd></div><div><dt>Legal entity</dt><dd>{legalEntityName}</dd></div>
              <div><dt>Record type</dt><dd>{subjectLabel(selected.subject_type)}</dd></div><div><dt>Record reference</dt><dd>{selected.subject_id || "Not recorded"}</dd></div>
            </dl>
            <section className="notification-detail__action" aria-label="Current record"><h3>Current record</h3><p>Review the latest details and available actions in the record.</p><Button variant="primary" isDisabled={!safeNotificationPath(selected.action_path)} onPress={() => openRecord(selected)}>Open record</Button>{!safeNotificationPath(selected.action_path) && <p>The record cannot be opened from this notification.</p>}</section>
            <section className="notification-detail__history" aria-label="Notification history"><h3>Notification history</h3><ol><li><strong>Update recorded</strong><time dateTime={validTime(selected.occurred_at)}>{formatTime(selected.occurred_at)}</time></li>{selected.read_at && <li><strong>Marked as read</strong><time dateTime={validTime(selected.read_at)}>{formatTime(selected.read_at)}</time></li>}</ol></section>
          </div>
          <p className="notification-detail__footnote">Reading an update does not complete or approve the work.</p>
        </> : <EmptyState population="Notification context" title="Select a notification" description="Choose an update to view its details and open the related record." compact/>}
      </aside>
    </div>
    {inbox.state === "live" && <p className="notification-inbox__as-of">Inbox checked {formatTime(inbox.asOf)}. Read status is separate from the current state of the work.</p>}
  </div>;
}

function NotificationSettings() {
  return <section className="notification-settings" aria-label="Notification preferences">
    <header><h2>Preferences</h2><p>Choose when to receive email updates.</p></header>
    <div className="notification-settings__columns">
      <section className="notification-settings__in-app" aria-labelledby="notification-in-app-heading"><h3 id="notification-in-app-heading">In-app notifications</h3><p>Your inbox brings together updates delivered to your account.</p>
        <dl><div><dt>Assignments and mentions</dt><dd>Shown when they are delivered to you.</dd></div><div><dt>Read status</dt><dd>Manage each update in your inbox.</dd></div><div><dt>Category preferences</dt><dd>Individual categories cannot be switched off here.</dd></div></dl>
        <Notice tone="info">Critical emails continue during quiet hours and when the daily digest is off.</Notice>
      </section>
      <NotificationPreferences/>
    </div>
  </section>;
}

function NotificationSymbol({ item }: { item: InAppNotification }) {
  const view = item.subject_type === "MATTER" ? "work" : item.subject_type === "RISK" ? "risks" : item.subject_type === "LOSS" ? "losses" : item.subject_type === "PROGRAM" ? "programs" : undefined;
  return view ? <NavigationIcon view={view}/> : <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden="true"><path d="M18 8a6 6 0 0 0-12 0c0 8-3 8-3 10h18c0-2-3-2-3-10M10 22h4"/></svg>;
}

function kindLabel(kind: string) {
  const labels: Record<string, string> = { MATTER_OWNER_ASSIGNED: "Work assignment", MATTER_COMMENT_MENTIONED: "Mention in a comment" };
  return labels[kind] ?? subjectLabel(kind);
}

function subjectLabel(value: string) {
  return value ? value.replaceAll("_", " ").toLocaleLowerCase().replace(/^./, (letter) => letter.toLocaleUpperCase()) : "Not recorded";
}

function validTime(value: string) { return Number.isFinite(Date.parse(value)) ? value : undefined; }
function formatTime(value: string, compact = false) {
  return validTime(value) ? new Date(value).toLocaleString(undefined, compact ? { month: "short", day: "numeric" } : { dateStyle: "medium", timeStyle: "short" }) : "Time unavailable";
}
function openPath(path: string) { window.location.hash = path; }
