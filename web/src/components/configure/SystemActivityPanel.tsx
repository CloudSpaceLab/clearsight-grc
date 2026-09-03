import { type FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import type { SystemActivityActorKind, SystemActivityCategory, SystemActivityEvent, SystemActivityQuery } from "../../operationsTypes";
import { loadSystemActivity } from "../../systemActivityApi";
import { Button, DataTable, EmptyState, FilterBar, Notice, SearchField, SelectField, StatusBadge, TextField, type DataColumn, type StatusTone } from "../ui";

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

const actorKindOptions = [
  { id: "INTERNAL_USER", label: "Internal user" },
  { id: "EXTERNAL_PARTICIPANT", label: "Vendor / external participant" },
  { id: "SERVICE", label: "Service" },
  { id: "SYSTEM", label: "System" },
  { id: "UNKNOWN", label: "Not recorded" },
] satisfies ReadonlyArray<{ id: SystemActivityActorKind; label: string }>;

const categoryOptions = [
  { id: "GRC_WORK", label: "GRC work" },
  { id: "FORMS_EVIDENCE", label: "Forms & evidence" },
  { id: "VENDOR", label: "Vendors" },
  { id: "AI", label: "AI" },
  { id: "CONFIGURATION", label: "Configuration" },
  { id: "SYSTEM", label: "System" },
  { id: "OTHER", label: "Other" },
] satisfies ReadonlyArray<{ id: SystemActivityCategory; label: string }>;

function localDateTimeInput(value: Date): string {
  const pad = (part: number) => String(part).padStart(2, "0");
  return `${value.getFullYear()}-${pad(value.getMonth() + 1)}-${pad(value.getDate())}T${pad(value.getHours())}:${pad(value.getMinutes())}`;
}

function initialFilters(mode: Mode): Filters {
  if (mode === "audit") return { ...emptyFilters };
  return { ...emptyFilters, from: localDateTimeInput(new Date(Date.now() - 24 * 60 * 60 * 1000)) };
}

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

function label(value: string) {
  return value.replaceAll("_", " ").toLowerCase().replace(/^./, (letter) => letter.toUpperCase());
}

function outcomeTone(outcome: SystemActivityEvent["outcome"]): StatusTone {
  switch (outcome) {
    case "SUCCEEDED": return "success";
    case "FAILED": return "error";
    case "DENIED": return "warning";
    case "PENDING":
    case "RETRYING": return "info";
    case "CANCELLED": return "neutral";
    default: return "unknown";
  }
}

const columns: readonly DataColumn<SystemActivityEvent>[] = [
  {
    id: "time",
    header: "Time",
    render: (event) => <time dateTime={event.occurred_at}>{new Date(event.occurred_at).toLocaleString()}</time>,
    accessibleText: (event) => new Date(event.occurred_at).toLocaleString(),
  },
  {
    id: "actor",
    header: "Actor",
    render: (event) => <span className="system-activity-cell"><strong>{actorLabel(event)}</strong><small>{label(event.actor_kind)}</small></span>,
    accessibleText: (event) => `${actorLabel(event)}, ${label(event.actor_kind)}`,
  },
  {
    id: "activity",
    header: "Activity",
    render: (event) => <span className="system-activity-cell"><strong>{event.action}</strong><small>{label(event.category)} · {event.event_type}</small></span>,
    accessibleText: (event) => `${event.action}, ${label(event.category)}, ${event.event_type}`,
  },
  {
    id: "object",
    header: "Object",
    render: (event) => <span className="system-activity-cell"><strong>{event.object_type.replaceAll("_", " ")}</strong><small>{event.object_id}</small></span>,
    accessibleText: objectLabel,
  },
  {
    id: "outcome",
    header: "Outcome",
    kind: "status",
    render: (event) => <StatusBadge tone={outcomeTone(event.outcome)}>{label(event.outcome)}</StatusBadge>,
    accessibleText: (event) => label(event.outcome),
  },
];

export function SystemActivityPanel({ mode }: { mode: Mode }) {
  const [filters, setFilters] = useState<Filters>(() => initialFilters(mode));
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

  function applyFilters(event: FormEvent) {
    event.preventDefault();
    setApplied(filters);
  }

  function clearFilters() {
    const next = initialFilters(mode);
    setFilters(next);
    setApplied(next);
  }

  const title = mode === "activity" ? "Recent activity" : "Audit log";
  const description = mode === "activity"
    ? "A bounded view of recently committed system and business activity. Open the owning record to act on it."
    : "Reconstruct recorded activity by actor, object, event type and date range without exposing event payloads.";

  return <section className="configure-card system-activity-panel" aria-labelledby={`${mode}-events-heading`}>
    <div className="configure-card-heading">
      <div><h3 id={`${mode}-events-heading`}>{title}</h3><p>{description}</p></div>
      <div className="system-activity-toolbar">
        {asOf && <span className="muted-copy">Current to {new Date(asOf).toLocaleString()}</span>}
        <Button variant="secondary" size="compact" isLoading={state === "loading" && items.length > 0} onPress={() => void load()}>Refresh</Button>
      </div>
    </div>

    <FilterBar
      label={`${title} filters`}
      resultCount={state === "live" ? items.length : undefined}
      resultLabel={(count) => `${count} loaded ${count === 1 ? "event" : "events"}`}
      onClear={clearFilters}
      fields={<form className="system-activity-filters" onSubmit={applyFilters}>
        <SearchField label="Actor" value={filters.actor} onChange={(actor) => setFilters((current) => ({ ...current, actor }))} placeholder="Actor name or safe identifier" isLoading={state === "loading"}/>
        <SelectField label="Actor type" value={filters.actorKind || undefined} placeholder="All actors" options={actorKindOptions} onChange={(actorKind) => setFilters((current) => ({ ...current, actorKind: actorKind ?? "" }))}/>
        <SelectField label="Category" value={filters.category || undefined} placeholder="All activity" options={categoryOptions} onChange={(category) => setFilters((current) => ({ ...current, category: category ?? "" }))}/>
        <TextField label="Event type" value={filters.eventType} onChange={(eventType) => setFilters((current) => ({ ...current, eventType }))} placeholder="e.g. MATTER_STATE_CHANGED"/>
        {mode === "audit" && <>
          <TextField label="Object type" value={filters.objectType} onChange={(objectType) => setFilters((current) => ({ ...current, objectType }))} placeholder="Matter, vendor, policy…"/>
          <TextField label="Object ID" value={filters.objectID} onChange={(objectID) => setFilters((current) => ({ ...current, objectID }))} placeholder="Exact identifier"/>
          <TextField label="Legal entity" value={filters.legalEntityID} onChange={(legalEntityID) => setFilters((current) => ({ ...current, legalEntityID }))} placeholder="Exact identifier"/>
        </>}
        <TextField label="From" type="datetime-local" value={filters.from} onChange={(from) => setFilters((current) => ({ ...current, from }))}/>
        <TextField label="To" type="datetime-local" value={filters.to} onChange={(to) => setFilters((current) => ({ ...current, to }))}/>
        <div className="system-activity-filter-action"><Button type="submit" variant="primary" size="compact">Apply filters</Button></div>
      </form>}
    />

    {state === "unavailable" && <Notice tone="error"><span>Activity could not be loaded. Recorded system state was not changed.</span> <Button variant="secondary" size="compact" onPress={() => void load()}>Retry</Button></Notice>}
    {state === "loading" && items.length === 0 && <Notice>Loading recorded activity…</Notice>}
    {state === "live" && items.length === 0 && <EmptyState population="Recorded activity matching the current filters" title="No matching activity" description="Try a wider time range or fewer filters."/>}
    {items.length > 0 && <DataTable
      ariaLabel={mode === "activity" ? "Recent system activity" : "Audit log events"}
      rows={items}
      rowKey={(event) => event.event_id}
      rowName={(event) => `${event.action} by ${actorLabel(event)}`}
      columns={columns}
      isLoading={state === "loading"}
      pagination={nextCursor ? {
        label: "Activity history pagination",
        nextLabel: "Load older activity",
        onNext: () => void load(nextCursor, true),
        isLoading: state === "loading",
      } : undefined}
    />}
  </section>;
}
