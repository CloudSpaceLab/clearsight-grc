import { useEffect, useRef, useState } from "react";
import { ApiError } from "../../http";
import type { SystemActivityEvent, SystemActivityPage, SystemActivityQuery } from "../../operationsTypes";
import { loadSystemActivity } from "../../systemActivityApi";
import { WorkItemIcon } from "../WorkItemIcon";
import { Button, EmptyState, IconButton, Notice, SelectField, StatusBadge, type StatusTone } from "../ui";

type ActivityPeriod = "7" | "30" | "90";
type Props = {
  canRead: boolean;
  legalEntityID?: string;
  legalEntityName: string;
  load?: typeof loadSystemActivity;
};
type HistoryState = {
  status: "loading" | "ready" | "unavailable" | "restricted";
  items: SystemActivityEvent[];
  nextCursor?: string;
  asOf?: string;
  loadingMore: boolean;
  error?: string;
};

const periodOptions = [
  { id: "7", label: "Last 7 days" },
  { id: "30", label: "Last 30 days" },
  { id: "90", label: "Last 90 days" },
] as const;
const loadingState: HistoryState = { status: "loading", items: [], loadingMore: false };
const pageSize = 25;
const dayMilliseconds = 24 * 60 * 60 * 1000;
const dateFormat = new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "long", year: "numeric", timeZone: "UTC" });
const timeFormat = new Intl.DateTimeFormat("en-GB", { hour: "2-digit", minute: "2-digit", hourCycle: "h23", timeZone: "UTC" });

export function NotificationActivity({ canRead, legalEntityID, legalEntityName, load = loadSystemActivity }: Props) {
  const [period, setPeriod] = useState<ActivityPeriod>("30");
  const [refresh, setRefresh] = useState(0);
  const entityID = legalEntityID?.trim();
  const hasEntity = Boolean(entityID && entityID !== "*");

  return <section className="notification-activity" aria-label="Activity history">
    <div className="notification-activity__top">
      <div><h2>Activity</h2><p>Recorded actions and outcomes for {legalEntityName}.</p></div>
      {canRead && hasEntity && <div className="notification-activity__filters">
        <SelectField<ActivityPeriod>
          label="Activity period"
          value={period}
          placeholder="Choose a period"
          options={periodOptions}
          allowsEmpty={false}
          isLabelHidden
          onChange={(value) => { if (value) setPeriod(value); }}
        />
        <IconButton aria-label="Refresh activity" title="Refresh activity" onPress={() => setRefresh((value) => value + 1)}><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden="true"><path d="M20 4v6h-6M4 20v-6h6M5 9a7 7 0 0 1 12-4l3 5M4 14l3 5a7 7 0 0 0 12-4"/></svg></IconButton>
      </div>}
    </div>
    {!canRead ? <RestrictedActivity/> : !hasEntity ? <EmptyState
      population="Activity history"
      title="Choose a legal entity"
      description="Select one legal entity to view its recorded activity. Group activity is not available in this view."
    /> : <ActivityHistory
      // A new scope or period must hide the former history immediately, before
      // passive effect cleanup can invalidate its outstanding responses.
      key={`${entityID}:${period}:${refresh}`}
      legalEntityID={entityID!}
      legalEntityName={legalEntityName}
      period={period}
      load={load}
      onRetry={() => setRefresh((value) => value + 1)}
    />}
  </section>;
}

function ActivityHistory({ legalEntityID, legalEntityName, period, load, onRetry }: {
  legalEntityID: string;
  legalEntityName: string;
  period: ActivityPeriod;
  load: typeof loadSystemActivity;
  onRetry: () => void;
}) {
  const [query] = useState<SystemActivityQuery>(() => {
    const to = Date.now();
    return { legalEntityID, limit: pageSize, from: new Date(to - Number(period) * dayMilliseconds).toISOString(), to: new Date(to).toISOString() };
  });
  const [history, setHistory] = useState<HistoryState>(loadingState);
  const request = useRef(0);
  const busy = useRef(false);

  useEffect(() => {
    const sequence = ++request.current;
    busy.current = true;
    setHistory(loadingState);
    void load(query).then((page) => {
      if (request.current !== sequence) return;
      requireScope(page, legalEntityID);
      setHistory({ status: "ready", items: uniqueEvents(page.items), nextCursor: page.next_cursor, asOf: page.as_of, loadingMore: false });
    }).catch((error: unknown) => {
      if (request.current !== sequence) return;
      setHistory({ status: isDenied(error) ? "restricted" : "unavailable", items: [], loadingMore: false, error: historyError(error) });
    }).finally(() => {
      if (request.current === sequence) busy.current = false;
    });
    return () => { request.current += 1; busy.current = false; };
  }, [load, query, legalEntityID]);

  async function loadOlder() {
    if (busy.current || history.status !== "ready" || !history.nextCursor) return;
    busy.current = true;
    const sequence = ++request.current;
    setHistory((current) => ({ ...current, loadingMore: true, error: undefined }));
    try {
      // Preserve both date bounds and the opaque server cursor for this history.
      const page = await load({ ...query, cursor: history.nextCursor });
      if (request.current !== sequence) return;
      requireScope(page, legalEntityID);
      setHistory((current) => ({
        status: "ready",
        items: uniqueEvents([...current.items, ...page.items]),
        nextCursor: page.next_cursor,
        asOf: page.as_of,
        loadingMore: false,
      }));
    } catch (error) {
      if (request.current !== sequence) return;
      if (isDenied(error)) {
        setHistory({ status: "restricted", items: [], loadingMore: false });
      } else {
        setHistory((current) => ({ ...current, loadingMore: false, error: historyError(error, true) }));
      }
    } finally {
      if (request.current === sequence) busy.current = false;
    }
  }

  if (history.status === "restricted") return <RestrictedActivity/>;
  const groups = groupByDay(history.items);
  return <div className="notification-activity__columns">
    <div className="notification-activity__panel" aria-busy={history.status === "loading" || history.loadingMore}>
      <div className="notification-activity__summary">
        <strong>{history.status === "ready" ? `${history.items.length} ${history.items.length === 1 ? "event" : "events"} loaded` : history.status === "loading" ? "Loading activity" : "Activity unavailable"}</strong>
        <span>Last {period} days · Newest first</span>
      </div>
      {history.status === "loading" ? <EmptyState
        population="Activity history"
        title="Loading activity history"
        description={`Checking recorded activity for ${legalEntityName}.`}
      /> : history.status === "unavailable" ? <EmptyState
        population="Activity history"
        title="Activity could not be loaded"
        description={history.error ?? "Refresh activity to try again."}
        role="alert"
        action={<Button onPress={onRetry}>Retry activity</Button>}
      /> : <>
        {history.items.length === 0 ? <EmptyState
          population="Activity history"
          title="No activity in this period"
          description={`No recorded activity was returned for ${legalEntityName} in the last ${period} days. Choose another period or refresh activity.`}
        /> : <div className="notification-activity__feed">
          {groups.map((group) => <section className="notification-activity__day" key={group.date} aria-label={group.label}>
            <h3>{group.label}</h3>
            <ol aria-label={`Activity on ${group.label}`}>
              {group.items.map((event) => <ActivityRecord key={event.event_id} event={event}/>)}
            </ol>
          </section>)}
        </div>}
        {(history.error || history.nextCursor) && <div className="notification-activity__paging">
          {history.error && <Notice tone="error">{history.error}</Notice>}
          {history.nextCursor && <Button isLoading={history.loadingMore} onPress={() => void loadOlder()}>Load older activity</Button>}
        </div>}
      </>}
    </div>
    <aside className="notification-activity__aside" aria-label="Activity context">
      <span className="notification-activity__eyebrow">Activity records</span>
      <h3>Recorded actions</h3>
      <p>Each event shows its actor, record and recorded outcome. A successful action does not establish that the related work is complete.</p>
      <dl className="notification-activity__proof">
        <div><dt>Selected scope</dt><dd>{legalEntityName}</dd></div>
        <div><dt>Period</dt><dd>Last {period} days</dd></div>
        <div><dt>History through</dt><dd>{formatTimestamp(query.to)}</dd></div>
        <div><dt>Last checked</dt><dd>{history.asOf ? formatTimestamp(history.asOf) : "Not yet available"}</dd></div>
      </dl>
      <p className="notification-activity__hint">Record versions are not recorded in this history. Check the related record for its current version and available actions.</p>
    </aside>
  </div>;
}

function ActivityRecord({ event }: { event: SystemActivityEvent }) {
  const date = validDate(event.occurred_at);
  return <li className="notification-activity__record" data-activity-event-id={event.event_id}>
    <span className="notification-activity__marker" aria-hidden="true"><WorkItemIcon type={event.object_type}/></span>
    <div className="notification-activity__record-main">
      <div className="notification-activity__record-heading">
        <h4>{event.action || humanize(event.event_type) || "Activity recorded"}</h4>
        <StatusBadge tone={outcomeTone(event.outcome)}>{humanize(event.outcome)}</StatusBadge>
      </div>
      <p className="notification-activity__actor"><strong>{actorLabel(event)}</strong> · {actorKindLabel(event.actor_kind)} · {date ? <time dateTime={event.occurred_at}>{timeFormat.format(date)} UTC</time> : "Time not recorded"}</p>
      <dl className="notification-activity__facts">
        <div><dt>Record</dt><dd>{objectLabel(event.object_type)} · {event.object_id || "Reference not recorded"}</dd></div>
      </dl>
      <p className="notification-activity__reference">Event {event.event_id} · {humanize(event.source) || "Source not recorded"}</p>
    </div>
  </li>;
}

function RestrictedActivity() {
  return <EmptyState population="Activity history" title="Activity access is restricted" description="Your permissions do not allow this activity history. Your personal notifications remain available in the Inbox."/>;
}

class ScopeMismatch extends Error {}

function requireScope(page: SystemActivityPage, legalEntityID: string) {
  // A missing or different scope invalidates the page; never turn a broad
  // response into an apparently authorized history by filtering its rows.
  if (page.items.some((event) => event.legal_entity_id !== legalEntityID)) throw new ScopeMismatch();
}

function historyError(error: unknown, older = false) {
  if (error instanceof ScopeMismatch) return "Activity could not be confirmed for the selected legal entity. Refresh activity to try again.";
  return older ? "Older activity could not be loaded. The events already loaded are still shown. Try loading older activity again." : "Recorded activity is unavailable. Retry activity to try again.";
}

function isDenied(error: unknown) { return error instanceof ApiError && (error.status === 401 || error.status === 403); }
function uniqueEvents(items: SystemActivityEvent[]) { return [...new Map(items.map((event) => [event.event_id, event])).values()]; }
function humanize(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/^./, (letter) => letter.toUpperCase()); }
function validDate(value?: string) { if (!value) return undefined; const date = new Date(value); return Number.isNaN(date.getTime()) ? undefined : date; }
function formatTimestamp(value?: string) { const date = validDate(value); return date ? `${dateFormat.format(date)}, ${timeFormat.format(date)} UTC` : "Not recorded"; }
function actorLabel(event: SystemActivityEvent) { return event.actor_display_name || event.actor_id || (event.actor_kind === "SYSTEM" ? "System" : event.actor_kind === "SERVICE" ? "Service identity" : "Actor not recorded"); }
function actorKindLabel(kind: SystemActivityEvent["actor_kind"]) {
  switch (kind) {
    case "INTERNAL_USER": return "Internal user";
    case "EXTERNAL_PARTICIPANT": return "External participant";
    case "SERVICE": return "Service";
    case "SYSTEM": return "System event";
    default: return "Actor type not recorded";
  }
}
function objectLabel(type: string) { return type === "MATTER" ? "Issue or change" : humanize(type) || "Record type not recorded"; }
function outcomeTone(outcome: SystemActivityEvent["outcome"]): StatusTone {
  switch (outcome) {
    case "SUCCEEDED": return "success";
    case "FAILED": return "error";
    case "DENIED": return "warning";
    case "PENDING":
    case "RETRYING": return "info";
    default: return "neutral";
  }
}
function groupByDay(items: SystemActivityEvent[]) {
  const groups = new Map<string, { date: string; label: string; items: SystemActivityEvent[] }>();
  for (const event of items) {
    const date = validDate(event.occurred_at);
    const key = date?.toISOString().slice(0, 10) ?? "undated";
    const group = groups.get(key) ?? { date: key, label: date ? dateFormat.format(date) : "Date not recorded", items: [] };
    group.items.push(event);
    groups.set(key, group);
  }
  return [...groups.values()];
}
