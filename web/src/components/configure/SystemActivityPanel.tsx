import { useCallback, useEffect, useMemo, useState } from "react";
import type { SystemActivityActorKind, SystemActivityCategory, SystemActivityEvent, SystemActivityQuery } from "../../operationsTypes";
import { loadSystemActivity } from "../../systemActivityApi";

type Mode = "activity" | "audit";
type LoadState = "loading" | "live" | "unavailable";

type Filters = {
  actor: string;
  actorKind: SystemActivityActorKind | "";
  category: SystemActivityCategory | "";
  eventType: string;
  objectType: string;
  objectID: string;
  legalEntityID: string;
  from: string;
  to: string;
};

const emptyFilters: Filters = {
  actor: "",
  actorKind: "",
  category: "",
  eventType: "",
  objectType: "",
  objectID: "",
  legalEntityID: "",
  from: "",
  to: "",
};

function toRFC3339(value: string): string | undefined {
  if (!value) return undefined;
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? undefined : parsed.toISOString();
}

function actorLabel(event: SystemActivityEvent) {
  if (event.actor_display_name) return event.actor_display_name;
  if (event.actor_id) return event.actor_id;
  if (event.actor_kind === "SYSTEM") return "System worker";
  if (event.actor_kind === "SERVICE") return "Service identity";
  return "Actor not recorded";
}

function objectLabel(event: SystemActivityEvent) {
  return `${event.object_type.replaceAll("_", " ")} · ${event.object_id}`;
}

function categoryLabel(value: string) {
  return value.replaceAll("_", " ").toLowerCase().replace(/^./, (letter) => letter.toUpperCase());
}

export function SystemActivityPanel({ mode }: { mode: Mode }) {
  const [filters, setFilters] = useState<Filters>(() => mode === "activity"
    ? { ...emptyFilters, from: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString().slice(0, 16) }
    : emptyFilters);
  const [applied, setApplied] = useState<Filters>(filters);
  const [items, setItems] = useState<SystemActivityEvent[]>([]);
  const [nextCursor, setNextCursor] = useState("");
  const [asOf, setAsOf] = useState("");
  const [state, setState] = useState<LoadState>("loading");

  const query = useMemo<SystemActivityQuery>(() => ({
    from: toRFC3339(applied.from),
    to: toRFC3339(applied.to),
    category: applied.category,
    eventType: applied.eventType.trim() || undefined,
    objectType: applied.objectType.trim() || undefined,
    objectID: applied.objectID.trim() || undefined,
    actor: applied.actor.trim() || undefined,
    actorKind: applied.actorKind,
    legalEntityID: applied.legalEntityID.trim() || undefined,
    limit: 50,
  }), [applied]);

  const load = useCallback(async (cursor = "", append = false) => {
    setState("loading");
    try {
      const page = await loadSystemActivity({ ...query, cursor: cursor || undefined });
      setItems((current) => append ? [...current, ...page.items] : page.items);
      setNextCursor(page.next_cursor ?? "");
      setAsOf(page.as_of);
      setState("live");
    } catch {
      if (!append) setItems([]);
      setState("unavailable");
    }
  }, [query]);

  useEffect(() => { void load(); }, [load]);

  function applyFilters(event: React.FormEvent) {
    event.preventDefault();
    setApplied(filters);
  }

  function clearFilters() {
    const next = mode === "activity"
      ? { ...emptyFilters, from: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString().slice(0, 16) }
      : emptyFilters;
    setFilters(next);
    setApplied(next);
  }

  return <section className="configure-card" aria-labelledby={`${mode}-events-heading`}>
    <div className="configure-card-heading">
      <div>
        <h3 id={`${mode}-events-heading`}>{mode === "activity" ? "Recent activity" : "Audit log"}</h3>
        <p>{mode === "activity"
          ? "A bounded view of recently committed system and business activity. Open the owning record to act on it."
          : "Reconstruct recorded activity by actor, object, event type and date range without exposing event payloads."}</p>
      </div>
      <div className="form-actions">
        {asOf && <span className="muted-copy">Current to {new Date(asOf).toLocaleString()}</span>}
        <button className="secondary-button" type="button" onClick={() => void load()}>Refresh</button>
      </div>
    </div>

    <form className="configure-filter-grid" onSubmit={applyFilters}>
      <label><span>Actor</span><input value={filters.actor} onChange={(event) => setFilters({ ...filters, actor: event.target.value })} placeholder="Name or safe identifier" /></label>
      <label><span>Actor type</span><select value={filters.actorKind} onChange={(event) => setFilters({ ...filters, actorKind: event.target.value as Filters["actorKind"] })}>
        <option value="">All actors</option><option value="INTERNAL_USER">Internal user</option><option value="EXTERNAL_PARTICIPANT">Vendor / external participant</option><option value="SERVICE">Service</option><option value="SYSTEM">System</option><option value="UNKNOWN">Not recorded</option>
      </select></label>
      <label><span>Category</span><select value={filters.category} onChange={(event) => setFilters({ ...filters, category: event.target.value as Filters["category"] })}>
        <option value="">All activity</option><option value="GRC_WORK">GRC work</option><option value="FORMS_EVIDENCE">Forms & evidence</option><option value="VENDOR">Vendors</option><option value="AI">AI</option><option value="CONFIGURATION">Configuration</option><option value="SYSTEM">System</option><option value="OTHER">Other</option>
      </select></label>
      <label><span>Event type</span><input value={filters.eventType} onChange={(event) => setFilters({ ...filters, eventType: event.target.value })} placeholder="e.g. MATTER_STATE_CHANGED" /></label>
      {mode === "audit" && <>
        <label><span>Object type</span><input value={filters.objectType} onChange={(event) => setFilters({ ...filters, objectType: event.target.value })} placeholder="Matter, vendor, policy…" /></label>
        <label><span>Object ID</span><input value={filters.objectID} onChange={(event) => setFilters({ ...filters, objectID: event.target.value })} placeholder="Exact identifier" /></label>
        <label><span>Legal entity</span><input value={filters.legalEntityID} onChange={(event) => setFilters({ ...filters, legalEntityID: event.target.value })} placeholder="Exact identifier" /></label>
      </>}
      <label><span>From</span><input type="datetime-local" value={filters.from} onChange={(event) => setFilters({ ...filters, from: event.target.value })} /></label>
      <label><span>To</span><input type="datetime-local" value={filters.to} onChange={(event) => setFilters({ ...filters, to: event.target.value })} /></label>
      <div className="form-actions configure-filter-actions"><button className="secondary-button" type="button" onClick={clearFilters}>Clear</button><button className="primary-button" type="submit">Apply filters</button></div>
    </form>

    {state === "unavailable" && <div className="configure-empty-state"><strong>Activity could not be loaded.</strong><p>The recorded system state was not changed. Retry when the service is available.</p><button className="secondary-button" type="button" onClick={() => void load()}>Retry</button></div>}
    {state === "loading" && items.length === 0 && <p role="status" className="muted-copy">Loading recorded activity…</p>}
    {state === "live" && items.length === 0 && <div className="configure-empty-state"><strong>No matching activity.</strong><p>Try a wider time range or fewer filters.</p></div>}
    {items.length > 0 && <div className="configure-record-list system-activity-list" aria-live="polite">{items.map((event) => <article key={event.event_id} className="system-activity-row">
      <time dateTime={event.occurred_at}>{new Date(event.occurred_at).toLocaleString()}</time>
      <div><strong>{actorLabel(event)}</strong><span>{categoryLabel(event.actor_kind)}</span></div>
      <div><strong>{event.action}</strong><span>{categoryLabel(event.category)} · {event.event_type}</span></div>
      <div><strong>{objectLabel(event)}</strong><span>{event.outcome}</span></div>
    </article>)}</div>}
    {nextCursor && <div className="form-actions"><button className="secondary-button" type="button" disabled={state === "loading"} onClick={() => void load(nextCursor, true)}>Load older activity</button></div>}
  </section>;
}
