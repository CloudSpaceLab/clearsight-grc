import { useEffect, useMemo, useState } from "react";
import { loadOversight, type OversightSnapshot } from "../../oversightApi";
import { Button, DataTable, EmptyState, Tabs } from "../ui";
import type { AttentionItem } from "../../types";
import "../../oversight.css";

type DetailView = "pressure" | "outlook" | "performance";
export type OversightMetricFilter = "all" | "critical-high" | "overdue" | "routing-gaps" | "outcome-failures";
type TodayState = "loading" | "live" | "unavailable";

export function OversightWorkspace({ organizationName, legalEntityName, onOpenMatter, loadSnapshot = loadOversight, metricFilter = "all", onMetricFilterChange, todayItems = [], todayState = "loading", onOpenTodayItem }: { organizationName: string; legalEntityName: string; onOpenMatter: (id: string) => void; loadSnapshot?: () => Promise<OversightSnapshot>; metricFilter?: OversightMetricFilter; onMetricFilterChange?: (filter: OversightMetricFilter) => void; todayItems?: AttentionItem[]; todayState?: TodayState; onOpenTodayItem?: (item: AttentionItem) => void }) {
  const [snapshot, setSnapshot] = useState<OversightSnapshot | null>(null);
  const [state, setState] = useState<"loading" | "live" | "unavailable">("loading");
  const [view, setView] = useState<DetailView>("pressure");
  const [localMetricFilter, setLocalMetricFilter] = useState<OversightMetricFilter>(metricFilter);
  const selectedMetricFilter = onMetricFilterChange ? metricFilter : localMetricFilter;

  useEffect(() => { setLocalMetricFilter(metricFilter); }, [metricFilter]);

  function selectMetric(filter: OversightMetricFilter) {
    if (onMetricFilterChange) onMetricFilterChange(filter);
    else setLocalMetricFilter(filter);
    if (filter !== "all") {
      requestAnimationFrame(() => {
        const attention = document.getElementById("oversight-attention");
        attention?.scrollIntoView?.({ behavior: window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth", block: "start" });
        attention?.focus();
      });
    }
  }

  async function load() {
    setState("loading");
    try {
      setSnapshot(await loadSnapshot());
      setState("live");
    } catch {
      setSnapshot(null);
      setState("unavailable");
    }
  }

  useEffect(() => { void load(); }, []);

  if (state === "loading") return <section className="oversight-workspace" aria-busy="true"><header className="oversight-header"><div><span className="eyebrow">{organizationName} · {legalEntityName}</span><h1>Risk and delivery oversight</h1><p>Loading the latest oversight snapshot…</p></div></header></section>;
  if (state === "unavailable" || !snapshot) return <section className="oversight-workspace"><div className="oversight-unavailable"><span className="eyebrow">{legalEntityName}</span><h1>Oversight information is unavailable</h1><p>No current snapshot could be loaded. Check projection operations or retry after the next processing cycle.</p><Button onPress={() => void load()}>Retry oversight</Button></div><OversightToday items={todayItems} state={todayState} onOpenItem={onOpenTodayItem}/></section>;

  const coverage = `${snapshot.coverage.population} issues checked · ${formatKnown(snapshot.coverage.excluded)} excluded · ${formatKnown(snapshot.coverage.unknown)} unknown`;
  const interventions = filterInterventions(snapshot.interventions, selectedMetricFilter);
  return <section className="oversight-workspace">
    <header className="oversight-header">
      <div><span className="eyebrow">{organizationName} · {legalEntityName}</span><h1>Risk and delivery oversight</h1><p>Review issues requiring intervention, resolution outlook and operating workload for this legal entity.</p></div>
      <div className={`oversight-freshness ${snapshot.freshness.toLowerCase()}`}><strong>{snapshot.freshness === "CURRENT" ? "Current snapshot" : "Snapshot needs refresh"}</strong><span>Generated {formatDateTime(snapshot.generated_at)}</span></div>
    </header>

    <div className="oversight-scope-line"><span>{coverage}</span><span>{formatDate(snapshot.period_start)} – {formatDate(snapshot.period_end)}</span><span>{snapshot.projection_version}</span></div>
    <details className="oversight-data-freshness">
      <summary>Data freshness</summary>
      <div><p>This snapshot was generated {formatDateTime(snapshot.generated_at)} from projection {snapshot.projection_version}.</p><p>{historyQualityLabel(snapshot)}</p><dl>{orderedHighWater(snapshot.source_high_water).map(([source, at]) => <div key={source}><dt>{humanize(source)}</dt><dd>{formatDateTime(at)}</dd></div>)}</dl></div>
    </details>

    <div className="oversight-counts" aria-label="Issues requiring oversight">
      <Metric label="Critical and high" value={snapshot.counts.critical_high} tone="critical" detail="Open priority 4–5 issues" filter="critical-high" active={selectedMetricFilter === "critical-high"} onSelect={selectMetric}/>
      <Metric label="Overdue" value={snapshot.counts.overdue} tone="warning" detail="Open issues past their due date" filter="overdue" active={selectedMetricFilter === "overdue"} onSelect={selectMetric}/>
      <Metric label="Routing gaps" value={snapshot.counts.routing_failures} tone="warning" detail="Active work without a resolved recipient" filter="routing-gaps" active={selectedMetricFilter === "routing-gaps"} onSelect={selectMetric}/>
      <Metric label="Outcome failures" value={snapshot.counts.outcome_failures} tone="critical" detail="Latest outcome check failed or inconclusive" filter="outcome-failures" active={selectedMetricFilter === "outcome-failures"} onSelect={selectMetric}/>
    </div>

    <OversightToday items={todayItems} state={todayState} onOpenItem={onOpenTodayItem}/>

    <section id="oversight-attention" className="oversight-attention" aria-labelledby="oversight-attention-heading" tabIndex={-1}>
      <div className="section-header"><div><span className="eyebrow">What needs attention now</span><h2 id="oversight-attention-heading">{selectedMetricFilter === "all" ? "Priority interventions" : metricFilterLabel(selectedMetricFilter)}</h2><p>{selectedMetricFilter === "all" ? "Ranked by overdue state, priority and current deadline." : "Ranked intervention records matching the selected measure."}</p></div><div className="oversight-inline-counts"><span>{snapshot.counts.due_soon} due soon</span><span>{snapshot.counts.unassigned} unassigned</span></div></div>
      {selectedMetricFilter !== "all" && <p className="oversight-result-count" aria-live="polite">{interventions.length} ranked {interventions.length === 1 ? "issue" : "issues"} shown for {metricFilterLabel(selectedMetricFilter).toLowerCase()}.</p>}
      {interventions.length ? <div className="oversight-intervention-list">{interventions.map((item) => <article key={`${item.target_type}-${item.target_id}`}>
        <div className={`oversight-priority p${item.priority}`}><span>P{item.priority}</span></div>
        <div><div className="oversight-intervention-title"><strong>{item.title}</strong><span>{humanize(item.category)}</span></div><p>{item.reason}</p><small>{item.owner_name || "No owner recorded"}{item.due_at ? ` · Due ${formatDate(item.due_at)}` : " · No due date recorded"} · {humanize(item.state)}</small></div>
        <div className="oversight-action"><Button size="compact" onPress={() => onOpenMatter(item.target_id)} aria-label={`Review ${item.title}`}>{item.next_action}</Button></div>
      </article>)}</div> : <EmptyState population={`${snapshot.coverage.population} issues checked in ${legalEntityName}`} title={selectedMetricFilter === "all" ? "No issue meets the current intervention criteria" : "No ranked intervention matches this measure"} description="Review the freshness and coverage above before treating this result as complete."/>}
    </section>

    <div className="oversight-analysis"><Tabs ariaLabel="Oversight analysis" items={detailViews} selectedKey={view} onSelectionChange={setView}>{(selected) => <div className="oversight-detail">
      {selected === "pressure" && <RiskPressure snapshot={snapshot}/>}
      {selected === "outlook" && <ResolutionOutlook snapshot={snapshot}/>}
      {selected === "performance" && <OperatingPerformance snapshot={snapshot}/>}
    </div>}</Tabs></div>
  </section>;
}

function Metric({ label, value, detail, tone, filter, active, onSelect }: { label: string; value: number; detail: string; tone: string; filter: OversightMetricFilter; active: boolean; onSelect: (filter: OversightMetricFilter) => void }) {
  const action = active ? "Show all priority interventions" : `Show ${label.toLowerCase()} interventions`;
  return <button type="button" className={`oversight-metric ${tone}`} aria-pressed={active} aria-controls="oversight-attention" onClick={() => onSelect(active ? "all" : filter)}>
    <span>{label}</span><strong>{value}</strong><small>{detail}</small><em>{action}</em>
  </button>;
}

function OversightToday({ items, state, onOpenItem }: { items: AttentionItem[]; state: TodayState; onOpenItem?: (item: AttentionItem) => void }) {
  const visible = items.slice(0, 4);
  return <section className="oversight-today" aria-labelledby="oversight-today-heading">
    <div className="section-header"><div><span className="eyebrow">Assigned work</span><h2 id="oversight-today-heading">Your assigned work</h2><p>Assigned decisions, evidence and exceptions requiring your current responsibility.</p></div></div>
    {state === "loading" ? <p className="oversight-today-status" aria-live="polite" aria-busy="true">Loading assigned work…</p> : state === "unavailable" ? <p className="oversight-today-status">Assigned work is unavailable. Refresh before relying on the current queue.</p> : visible.length ? <div className="oversight-today-list">{visible.map((item) => <button type="button" className="oversight-today-item" key={item.id} onClick={() => onOpenItem?.(item)} disabled={!onOpenItem} aria-label={`Open ${item.title}`}>
      <span className="oversight-today-item__main"><strong>{item.title}</strong><small>{item.why_now}</small></span><span className="oversight-today-item__meta"><span>{item.owner}</span><time>{formatTodayDue(item.due_at)}</time></span>
    </button>)}</div> : <p className="oversight-today-status">No assigned work or permitted operational exceptions are open for you.</p>}
  </section>;
}

function RiskPressure({ snapshot }: { snapshot: OversightSnapshot }) {
  const max = useMemo(() => Math.max(1, ...snapshot.pressure.map((item) => item.critical + item.high + item.other)), [snapshot.pressure]);
  return <div className="oversight-two-column"><article className="oversight-panel"><div className="section-header"><div><h2>Risk pressure by issue type</h2><p>Open issues split by recorded priority.</p></div></div>{snapshot.pressure.length ? <><div className="oversight-bars" aria-hidden="true">{snapshot.pressure.map((item) => { const total = item.critical + item.high + item.other; return <div key={item.category}><span>{humanize(item.category)}</span><div><i className="critical" style={{ width: `${item.critical / max * 100}%` }}/><i className="high" style={{ width: `${item.high / max * 100}%` }}/><i className="other" style={{ width: `${item.other / max * 100}%` }}/></div><strong>{total}</strong></div>; })}</div><DataTable ariaLabel="Risk pressure by issue type" rows={snapshot.pressure} rowKey={(item) => item.category} rowName={(item) => humanize(item.category)} columns={pressureColumns}/></> : <EmptyMeasure/>}</article><Aging snapshot={snapshot}/></div>;
}

function Aging({ snapshot }: { snapshot: OversightSnapshot }) {
  const total = Math.max(1, snapshot.aging.reduce((sum, item) => sum + item.count, 0));
  return <article className="oversight-panel"><div className="section-header"><div><h2>Open issue aging</h2><p>Elapsed time since each open issue was recorded.</p></div></div><div className="oversight-aging">{snapshot.aging.map((item) => <div key={item.label}><div><strong>{item.label}</strong><span>{item.count}</span></div><progress max={total} value={item.count}>{item.count} of {total}</progress></div>)}</div></article>;
}

function ResolutionOutlook({ snapshot }: { snapshot: OversightSnapshot }) {
  return <div className="oversight-two-column"><article className="oversight-panel oversight-estimates"><div className="section-header"><div><h2>Historical resolution ranges</h2><p>Ranges use completed issues of the same type; they are not promised completion dates.</p></div></div>{snapshot.estimates.length ? snapshot.estimates.map((item) => <div key={item.category}><div><strong>{humanize(item.category)}</strong><span>{item.confidence.toLowerCase()} confidence · {item.sample_size} completed</span></div><p><b>{formatDuration(item.lower_hours)}–{formatDuration(item.upper_hours)}</b><span>Median {formatDuration(item.median_hours)}</span></p><small>{item.estimated_by}</small></div>) : <EmptyState population="Comparable completed issues in this legal entity" title="Not enough completed work for a resolution range" description="At least five comparable completed issues are required before a range is shown."/>}</article><Aging snapshot={snapshot}/></div>;
}

function OperatingPerformance({ snapshot }: { snapshot: OversightSnapshot }) {
  return <article className="oversight-panel"><div className="section-header"><div><h2>Workload and completion context</h2><p>Measures show work conditions and delivery history; they are not an employee ranking.</p></div></div>{snapshot.performance.length ? <DataTable ariaLabel="Owner workload and completion context" rows={snapshot.performance} rowKey={(item) => item.owner_id} rowName={(item) => item.owner_name} columns={performanceColumns}/> : <EmptyState population="Issues with an accountable owner and recorded lifecycle dates" title="No owner-level measures are available" description="Completion and workload measures appear after issues have an accountable owner and recorded lifecycle dates."/>}</article>;
}

function EmptyMeasure() { return <EmptyState population="Open issues in this legal entity" title="No open issues in this measure" description="The current snapshot contains no rows for this breakdown."/>; }

function historyQualityLabel(snapshot: OversightSnapshot) {
  const quality = snapshot.history_quality;
  return `${quality.complete_lifecycle} of ${quality.completed_population} completed issues have complete lifecycle events · ${quality.excluded_from_durations} excluded because an opened or closed event is missing · employee handling time follows each recorded owner assignment; reassignment, return, blocked and reopen counts remain visible separately`;
}

const detailViews = [
  { id: "pressure", label: "Risk pressure" },
  { id: "outlook", label: "Resolution outlook" },
  { id: "performance", label: "Operating performance" },
] as const;

const pressureColumns = [
  { id: "category", header: "Issue type", render: (item: OversightSnapshot["pressure"][number]) => humanize(item.category), accessibleText: (item: OversightSnapshot["pressure"][number]) => humanize(item.category) },
  { id: "critical", header: "Critical", kind: "number" as const, render: (item: OversightSnapshot["pressure"][number]) => item.critical, accessibleText: (item: OversightSnapshot["pressure"][number]) => String(item.critical) },
  { id: "high", header: "High", kind: "number" as const, render: (item: OversightSnapshot["pressure"][number]) => item.high, accessibleText: (item: OversightSnapshot["pressure"][number]) => String(item.high) },
  { id: "other", header: "Other", kind: "number" as const, render: (item: OversightSnapshot["pressure"][number]) => item.other, accessibleText: (item: OversightSnapshot["pressure"][number]) => String(item.other) },
  { id: "overdue", header: "Overdue", kind: "number" as const, render: (item: OversightSnapshot["pressure"][number]) => item.overdue, accessibleText: (item: OversightSnapshot["pressure"][number]) => String(item.overdue) },
];

const performanceColumns = [
  { id: "owner", header: "Person", render: (item: OversightSnapshot["performance"][number]) => <strong>{item.owner_name}</strong>, accessibleText: (item: OversightSnapshot["performance"][number]) => item.owner_name },
  { id: "load", header: "Current load", kind: "number" as const, render: (item: OversightSnapshot["performance"][number]) => item.current_load, accessibleText: (item: OversightSnapshot["performance"][number]) => String(item.current_load) },
  { id: "completed", header: "Completed", kind: "number" as const, render: (item: OversightSnapshot["performance"][number]) => <>{item.completed}<span>{item.completed} completed · {item.measurement_samples} measured</span></>, accessibleText: (item: OversightSnapshot["performance"][number]) => `${item.completed} completed; ${item.measurement_samples} measured` },
  { id: "cycle", header: "Active handling time", render: (item: OversightSnapshot["performance"][number]) => <>{item.median_hours == null ? "Unknown" : `${formatDuration(item.median_hours)} median`}<span>{item.p75_hours == null ? "p75 unknown" : `${formatDuration(item.p75_hours)} p75`} · {formatDuration(item.blocked_hours)} blocked</span></>, accessibleText: (item: OversightSnapshot["performance"][number]) => item.median_hours == null ? "Unknown" : `${formatDuration(item.median_hours)} median; ${item.p75_hours == null ? "p75 unknown" : `${formatDuration(item.p75_hours)} p75`}; ${formatDuration(item.blocked_hours)} blocked` },
  { id: "sla", header: "SLA met", render: (item: OversightSnapshot["performance"][number]) => item.sla_attainment == null ? "Unknown" : formatPercent(item.sla_attainment), accessibleText: (item: OversightSnapshot["performance"][number]) => item.sla_attainment == null ? "Unknown" : formatPercent(item.sla_attainment) },
  { id: "history", header: "Workflow history", render: (item: OversightSnapshot["performance"][number]) => <>{workflowHistory(item)}</>, accessibleText: (item: OversightSnapshot["performance"][number]) => workflowHistory(item).replaceAll(" · ", "; ") },
];
function orderedHighWater(values: Record<string, string>) {
  const order = ["matters", "actions", "workflow_tasks", "verification_results", "continuity_events"];
  return Object.entries(values).sort(([left], [right]) => {
    const leftIndex = order.indexOf(left); const rightIndex = order.indexOf(right);
    return (leftIndex < 0 ? order.length : leftIndex) - (rightIndex < 0 ? order.length : rightIndex) || left.localeCompare(right);
  });
}
function workflowHistory(item: OversightSnapshot["performance"][number]) { return `${item.blocked} blocked · ${item.reopened} reopened · ${formatOptional(item.reassigned)} reassigned · ${formatOptional(item.returned)} returned`; }
function formatOptional(value?: number) { return value == null ? "Unknown" : value.toLocaleString(); }
function formatKnown(value?: number) { return value == null ? "unknown" : value.toLocaleString(); }
function formatPercent(value: number) { return `${(value * 100).toFixed(1)}%`; }
function formatDate(value: string) { return new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric" }).format(new Date(value)); }
function formatDateTime(value: string) { return new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", hour: "numeric", minute: "2-digit" }).format(new Date(value)); }
function formatDuration(hours: number) { return hours < 48 ? `${Math.round(hours)}h` : `${(hours / 24).toFixed(hours % 24 === 0 ? 0 : 1)}d`; }
function humanize(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()); }

function filterInterventions(items: OversightSnapshot["interventions"], filter: OversightMetricFilter) {
  if (filter === "all") return items;
  return items.filter((item) => {
    if (filter === "critical-high") return item.priority >= 4;
    if (filter === "overdue") return Boolean(item.due_at && Date.parse(item.due_at) < Date.now());
    if (filter === "routing-gaps") return !item.owner_id && !item.owner_name;
    return /outcome|verification|inconclusive|failed/i.test(`${item.category} ${item.state} ${item.reason}`);
  });
}

function metricFilterLabel(filter: Exclude<OversightMetricFilter, "all">) {
  switch (filter) {
    case "critical-high": return "Critical and high interventions";
    case "overdue": return "Overdue interventions";
    case "routing-gaps": return "Routing gap interventions";
    case "outcome-failures": return "Outcome failure interventions";
  }
}

function formatTodayDue(value: string) {
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) return "No deadline";
  return `Due ${new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric" }).format(new Date(parsed))}`;
}
