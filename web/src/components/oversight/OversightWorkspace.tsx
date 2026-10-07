import { useEffect, useMemo, useState } from "react";
import { loadHomeMetricMembers, loadHomeMetrics, type HomeMetricBundle, type HomeMetricMemberPage } from "../../metricApi";
import { homeMetricDetail, homeMetricFilter, homeMetricMeta, homeMetricQuality, homeMetricTone, headlineMetricDefinitions, type HomeMetricFilter } from "../../homeMetricPresentation";
import { loadOversight, type OversightSnapshot } from "../../oversightApi";
import type { HomeTab } from "../../appRouting";
import type { ReportingPeriod, ReportingPeriodQuery } from "../../reportingPeriod";
import { Button, DataTable, EmptyState, MetricCard, Notice, Tabs } from "../ui";
import type { AttentionItem } from "../../types";
import { OversightPeriodPicker } from "./OversightPeriodPicker";
import { MetricMemberDrill } from "./MetricMemberDrill";
import "../../oversight.css";

type DetailView = "pressure" | "outlook" | "performance";
export type OversightMetricFilter = HomeMetricFilter;
type TodayState = "loading" | "live" | "unavailable";

type OversightWorkspaceProps = {
  refreshToken?: string;
  organizationName: string;
  legalEntityName: string;
  organizationScopeID?: string;
  organizationScopeName?: string;
  onOpenMatter: (id: string) => void;
  onOpenProgram?: (id: string) => void;
  loadSnapshot?: (period?: ReportingPeriodQuery, organizationScopeID?: string) => Promise<OversightSnapshot>;
  loadMetrics?: (period?: ReportingPeriodQuery, organizationScopeID?: string) => Promise<HomeMetricBundle>;
  loadMetricMembers?: typeof loadHomeMetricMembers;
  metricFilter?: OversightMetricFilter;
  onMetricFilterChange?: (filter: OversightMetricFilter) => void;
  todayItems?: AttentionItem[];
  todayState?: TodayState;
  onOpenTodayItem?: (item: AttentionItem) => void;
  homeTab?: HomeTab;
  onHomeTabChange?: (tab: HomeTab) => void;
  onOpenWork?: () => void;
};

export function OversightWorkspace({
  refreshToken,
  organizationName,
  legalEntityName,
  organizationScopeID,
  organizationScopeName,
  onOpenMatter,
  onOpenProgram,
  loadSnapshot = loadOversight,
  loadMetrics = loadHomeMetrics,
  loadMetricMembers = loadHomeMetricMembers,
  metricFilter = "all",
  onMetricFilterChange,
  todayItems = [],
  todayState = "loading",
  onOpenTodayItem,
  homeTab,
  onHomeTabChange,
  onOpenWork,
}: OversightWorkspaceProps) {
  const [snapshot, setSnapshot] = useState<OversightSnapshot | null>(null);
  const [state, setState] = useState<"loading" | "live" | "unavailable">("loading");
  const [metrics, setMetrics] = useState<HomeMetricBundle | null>(null);
  const [metricState, setMetricState] = useState<"loading" | "live" | "unavailable">("loading");
  const [view, setView] = useState<DetailView>("pressure");
  const [localMetricFilter, setLocalMetricFilter] = useState<OversightMetricFilter>(metricFilter);
  const [periodState, setPeriodState] = useState<"idle" | "changing">("idle");
  const [periodError, setPeriodError] = useState("");
  const [activePeriod, setActivePeriod] = useState<ReportingPeriodQuery>();
  const [memberPage, setMemberPage] = useState<HomeMetricMemberPage | null>(null);
  const [memberState, setMemberState] = useState<"idle" | "loading" | "live" | "unavailable">("idle");
  const [memberCursors, setMemberCursors] = useState<string[]>([]);
  const [memberRetry, setMemberRetry] = useState(0);
  const [localHomeTab, setLocalHomeTab] = useState<HomeTab>(homeTab ?? (metricFilter === "all" ? "oversight" : "attention"));
  const selectedHomeTab = onHomeTabChange ? (homeTab ?? "oversight") : localHomeTab;
  const selectedMetricFilter = onMetricFilterChange ? metricFilter : localMetricFilter;
  const selectedMetric = selectedMetricFilter === "all"
    ? undefined
    : metrics?.items.find((item) => homeMetricFilter(item.drill.filter) === selectedMetricFilter);
  const exactMetricSourceID = selectedMetric?.drill.consistency === "SOURCE_SNAPSHOT" ? metrics?.source_id : undefined;
  const exactMetricID = exactMetricSourceID ? selectedMetric?.id : undefined;
  const exactDefinitionRevision = exactMetricSourceID ? selectedMetric?.definition_revision : undefined;
  const exactMetricValue = exactMetricSourceID ? selectedMetric?.value : undefined;
  const exactDrillIdentity = exactMetricSourceID && exactMetricID && exactDefinitionRevision
    ? `${exactMetricSourceID}:${exactMetricID}:${exactDefinitionRevision}`
    : "";
  const exactDrillActive = exactDrillIdentity !== "";
  const memberCursor = memberCursors[memberCursors.length - 1];

  useEffect(() => { setLocalMetricFilter(metricFilter); }, [metricFilter]);
  useEffect(() => {
    if (homeTab) setLocalHomeTab(homeTab);
    else if (metricFilter !== "all") setLocalHomeTab("attention");
  }, [homeTab, metricFilter]);

  useEffect(() => {
    setMemberCursors([]);
    setMemberPage(null);
    setMemberState(exactDrillIdentity ? "loading" : "idle");
  }, [exactDrillIdentity]);

  useEffect(() => {
    if (selectedHomeTab !== "attention" || !exactDrillIdentity || !exactMetricSourceID || !exactMetricID || !exactDefinitionRevision || exactMetricValue === undefined) return;
    const controller = new AbortController();
    setMemberState("loading");
    void loadMetricMembers(
      exactMetricID,
      exactMetricSourceID,
      exactDefinitionRevision,
      organizationScopeID,
      memberCursor,
      50,
      controller.signal,
    ).then((page) => {
      if (controller.signal.aborted) return;
      const valid = page.source_id === exactMetricSourceID
        && page.metric_id === exactMetricID
        && page.definition_revision === exactDefinitionRevision
        && page.count === exactMetricValue;
      if (!valid) {
        setMemberPage(null);
        setMemberState("unavailable");
        return;
      }
      setMemberPage(page);
      setMemberState("live");
    }).catch((error: unknown) => {
      if (controller.signal.aborted || isAbortError(error)) return;
      setMemberPage(null);
      setMemberState("unavailable");
    });
    return () => controller.abort();
  }, [
    exactDrillIdentity,
    exactMetricSourceID,
    exactMetricID,
    exactDefinitionRevision,
    exactMetricValue,
    organizationScopeID,
    memberCursor,
    memberRetry,
    loadMetricMembers,
    selectedHomeTab,
  ]);

  function selectMetric(filter: OversightMetricFilter) {
    if (onMetricFilterChange) onMetricFilterChange(filter);
    else setLocalMetricFilter(filter);
    if (selectedHomeTab !== "attention") {
      if (onHomeTabChange) onHomeTabChange("attention");
      else setLocalHomeTab("attention");
    }
    if (filter !== "all") {
      requestAnimationFrame(() => {
        const attention = document.getElementById("oversight-attention");
        attention?.scrollIntoView?.({ behavior: window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth", block: "start" });
        attention?.focus();
      });
    }
  }

  async function load(period = activePeriod) {
    setState("loading");
    setMetricState("loading");
    const [snapshotResult, metricResult] = await Promise.allSettled([
      loadSnapshot(period, organizationScopeID),
      loadMetrics(period, organizationScopeID),
    ]);
    if (snapshotResult.status === "fulfilled") {
      setSnapshot(snapshotResult.value);
      setState("live");
    } else {
      setSnapshot(null);
      setState("unavailable");
    }
    if (metricResult.status === "fulfilled") {
      setMetrics(metricResult.value);
      setMetricState("live");
    } else {
      setMetrics(null);
      setMetricState("unavailable");
    }
  }

  useEffect(() => { void load(); }, [organizationScopeID, refreshToken]);

  async function changePeriod(period: ReportingPeriodQuery) {
    if (periodState === "changing") return;
    setPeriodState("changing");
    setPeriodError("");
    const [snapshotResult, metricResult] = await Promise.allSettled([
      loadSnapshot(period, organizationScopeID),
      loadMetrics(period, organizationScopeID),
    ]);
    if (snapshotResult.status === "fulfilled" && metricResult.status === "fulfilled"
      && sameReportingPeriod(snapshotResult.value.reporting_period, metricResult.value.reporting_period)
      && sameOrganizationScope(snapshotResult.value, metricResult.value, organizationScopeID)) {
      setSnapshot(snapshotResult.value);
      setMetrics(metricResult.value);
      setState("live");
      setMetricState("live");
      setActivePeriod(period);
    } else {
      setPeriodError("The reporting period could not be changed. Current data is still shown.");
    }
    setPeriodState("idle");
  }

  function selectHomeTab(tab: HomeTab) {
    if (onHomeTabChange) onHomeTabChange(tab);
    else {
      setLocalHomeTab(tab);
      if (tab !== "attention") setLocalMetricFilter("all");
    }
  }

  const headlineMetrics = <HomeMetricStrip metrics={metrics} state={metricState} selected={selectedMetricFilter} onSelect={selectMetric}/>;
  const coverage = snapshot
    ? organizationScopeID
      ? `${snapshot.coverage.population} issues · ${formatKnown(snapshot.coverage.excluded)} excluded · unassigned area excluded`
      : `${snapshot.coverage.population} issues checked · ${formatKnown(snapshot.coverage.excluded)} excluded · ${formatKnown(snapshot.coverage.unknown)} unknown`
    : "";
  const interventions = snapshot ? filterInterventions(snapshot.interventions, selectedMetricFilter) : [];
  const visibleInterventions = interventions.slice(0, 10);
  const periodSource = snapshot
    ? { reporting_period: snapshot.reporting_period, freshness: snapshot.freshness, generated_at: snapshot.generated_at }
    : metrics
      ? { reporting_period: metrics.reporting_period, freshness: metrics.freshness, generated_at: metrics.generated_at }
      : undefined;

  return <section className="oversight-workspace">
    <header className="oversight-header">
      <div>
        <span className="eyebrow">{organizationName} · {legalEntityName}{organizationScopeName ? ` · ${organizationScopeName}` : ""}</span>
        <h1>Home</h1>
        <p>{homeTabDescription(selectedHomeTab, organizationScopeName)}</p>
      </div>
      {selectedHomeTab !== "my-work" && periodSource && <OversightPeriodPicker
        period={periodSource.reporting_period}
        freshness={periodSource.freshness}
        generatedAt={periodSource.generated_at}
        isChanging={periodState === "changing"}
        error={periodError}
        onApply={(period) => void changePeriod(period)}
      />}
    </header>

    {selectedHomeTab !== "my-work" && snapshot && <>
      <div className="oversight-scope-line"><span>{coverage}</span></div>
      <details className="oversight-data-freshness">
        <summary>Data freshness</summary>
        <div>
          <p>This snapshot was generated {formatDateTime(snapshot.generated_at)} from projection {snapshot.projection_version}.</p>
          <p>{historyQualityLabel(snapshot)}</p>
          <dl>{orderedHighWater(snapshot.source_high_water).map(([source, at]) => <div key={source}><dt>{humanize(source)}</dt><dd>{formatDateTime(at)}</dd></div>)}</dl>
        </div>
      </details>
    </>}

    <div className="oversight-home-tabs">
      <Tabs ariaLabel="Home views" compactLabel="Home view" items={homeIntentTabs} selectedKey={selectedHomeTab} onSelectionChange={selectHomeTab}>
        {(tab) => <div className="oversight-home-panel">
          {tab === "oversight" && <>
            {state === "loading" && <p className="oversight-today-status" role="status" aria-busy="true">Loading oversight…</p>}
            {state === "unavailable" && <><Notice tone="warning">Oversight information is unavailable.</Notice><div className="workspace-recovery-actions"><Button onPress={() => void load()}>Retry Home data</Button></div></>}
            {state === "live" && snapshot && <div className="oversight-analysis"><Tabs ariaLabel="Oversight analysis" items={detailViews} selectedKey={view} onSelectionChange={setView}>{(selected) => <div className="oversight-detail">
              {selected === "pressure" && <RiskPressure snapshot={snapshot}/>}
              {selected === "outlook" && <ResolutionOutlook snapshot={snapshot}/>}
              {selected === "performance" && <OperatingPerformance snapshot={snapshot}/>}
            </div>}</Tabs></div>}
          </>}

          {tab === "attention" && <>
            {headlineMetrics}
            {state === "loading" && <p className="oversight-today-status" role="status" aria-busy="true">Loading priority interventions…</p>}
            {state === "unavailable" && <Notice tone="warning">Priority intervention detail is unavailable. Headline metrics may still be current.</Notice>}
            {snapshot && <section id="oversight-attention" className="oversight-attention" aria-labelledby="oversight-attention-heading" tabIndex={-1}>
              <div className="section-header">
                <div>
                  <span className="eyebrow">Attention</span>
                  <h2 id="oversight-attention-heading">{selectedMetricFilter === "all" ? "Priority interventions" : metricFilterLabel(selectedMetricFilter)}</h2>
                  <p>{selectedMetricFilter === "all" ? "Highest-priority current issues. Open Work for the complete queue." : exactDrillActive ? "Exact retained records counted in this metric snapshot." : "Current intervention records matching the selected measure."}</p>
                </div>
                <div className="oversight-attention-actions">
                  <div className="oversight-inline-counts"><span>{snapshot.counts.due_soon} due soon</span><span>{snapshot.counts.unassigned} unassigned</span></div>
                  {onOpenWork && <Button variant="secondary" size="compact" onPress={onOpenWork}>Open Work</Button>}
                </div>
              </div>
              {selectedMetricFilter !== "all" && exactDrillActive
                ? <MetricMemberDrill
                  label={selectedMetric?.label ?? metricFilterLabel(selectedMetricFilter)}
                  state={memberState === "idle" ? "loading" : memberState}
                  page={memberPage}
                  hasPrevious={memberCursors.length > 0}
                  onPrevious={() => setMemberCursors((value) => value.slice(0, -1))}
                  onNext={() => {
                    if (!memberPage?.next_cursor) return;
                    setMemberCursors((value) => [...value, memberPage.next_cursor!]);
                  }}
                  onRetry={() => setMemberRetry((value) => value + 1)}
                  onOpenMatter={onOpenMatter}
                  onOpenProgram={onOpenProgram}
                />
                : <>
                  {selectedMetricFilter !== "all" && <p className="oversight-result-count" aria-live="polite">{visibleInterventions.length} ranked {visibleInterventions.length === 1 ? "issue" : "issues"} shown for {metricFilterLabel(selectedMetricFilter).toLowerCase()}.</p>}
                  {visibleInterventions.length ? <div className="oversight-intervention-list">{visibleInterventions.map((item) => <article key={`${item.target_type}-${item.target_id}`}>
                    <div className={`oversight-priority p${item.priority}`}><span>P{item.priority}</span></div>
                    <div><div className="oversight-intervention-title"><strong>{item.title}</strong><span>{humanize(item.category)}</span></div><p>{item.reason}</p><small>{item.owner_name || "No owner recorded"}{item.due_at ? ` · Due ${formatDate(item.due_at)}` : " · No due date recorded"} · {humanize(item.state)}</small></div>
                    <div className="oversight-action"><Button size="compact" onPress={() => onOpenMatter(item.target_id)} aria-label={`Review ${item.title}`}>{item.next_action}</Button></div>
                  </article>)}</div> : state === "live" && <EmptyState population={organizationScopeID ? `${snapshot.coverage.population} attributed issues in ${organizationScopeName || "this scope"}` : `${snapshot.coverage.population} issues checked in ${legalEntityName}`} title={selectedMetricFilter === "all" ? "No issue meets the current intervention criteria" : "No ranked intervention matches this measure"} description="Review the freshness and coverage above before treating this result as complete."/>}
                </>}
            </section>}
          </>}

          {tab === "my-work" && <OversightToday items={todayItems} state={todayState} onOpenItem={onOpenTodayItem} onOpenWork={onOpenWork}/>}
        </div>}
      </Tabs>
    </div>
  </section>;
}

function sameOrganizationScope(snapshot: OversightSnapshot, metrics: HomeMetricBundle, requested?: string) {
  const expected = requested ?? "";
  return (snapshot.organization_scope_id ?? "") === expected
    && (metrics.scope_kind === "ORGANIZATION_SCOPE" ? metrics.scope_id : "") === expected;
}

function sameReportingPeriod(left: ReportingPeriod, right: ReportingPeriod) {
  return left.start_date === right.start_date && left.end_date === right.end_date && left.mode === right.mode;
}

function HomeMetricStrip({ metrics, state, selected, onSelect }: { metrics: HomeMetricBundle | null; state: "loading" | "live" | "unavailable"; selected: OversightMetricFilter; onSelect: (filter: OversightMetricFilter) => void }) {
  return <div className="oversight-counts" aria-label={state === "unavailable" ? "Risk metrics unavailable" : "Risk metrics"} aria-busy={state === "loading" || undefined}>
    {headlineMetricDefinitions.map((definition) => {
      const metric = metrics?.items.find((item) => item.id === definition.id);
      if (!metric) return <MetricCard key={definition.id} label={definition.label} value="—" detail={definition.detail} quality="unknown" qualityLabel={state === "loading" ? "Loading" : "Unavailable"}/>;
      const filter = homeMetricFilter(metric.drill.filter);
      const active = filter !== "all" && selected === filter;
      return <MetricCard
        key={metric.id}
        label={metric.label}
        value={metric.value}
        detail={homeMetricDetail(metric.id)}
        meta={homeMetricMeta(metric)}
        tone={homeMetricTone(metric)}
        quality={homeMetricQuality(metric)}
        actionLabel={filter === "all" ? undefined : active ? "Show all priority interventions" : metric.drill.consistency === "SOURCE_SNAPSHOT" ? "Review exact snapshot records" : "Review current related interventions"}
        isSelected={active}
        ariaControls={filter === "all" ? undefined : "oversight-attention"}
        onPress={filter === "all" ? undefined : () => onSelect(active ? "all" : filter)}
      />;
    })}
  </div>;
}

function OversightToday({ items, state, onOpenItem, onOpenWork }: { items: AttentionItem[]; state: TodayState; onOpenItem?: (item: AttentionItem) => void; onOpenWork?: () => void }) {
  const visible = items.slice(0, 8);
  return <section className="oversight-today" aria-labelledby="oversight-today-heading">
    <div className="section-header"><div><span className="eyebrow">My work</span><h2 id="oversight-today-heading">Your assigned work</h2><p>Assigned decisions, evidence and exceptions requiring your action.</p></div>{onOpenWork && <Button variant="secondary" size="compact" onPress={onOpenWork}>Open Work</Button>}</div>
    {state === "loading" ? <p className="oversight-today-status" aria-live="polite" aria-busy="true">Loading assigned work…</p> : state === "unavailable" ? <p className="oversight-today-status">Assigned work is unavailable. Refresh before relying on the current queue.</p> : visible.length ? <div className="oversight-today-list">{visible.map((item) => <button type="button" className="oversight-today-item" key={item.id} onClick={() => onOpenItem?.(item)} disabled={!onOpenItem} aria-label={`Open ${item.title}`}>
      <span className="oversight-today-item__main"><strong>{item.title}</strong><small>{item.why_now}</small></span><span className="oversight-today-item__meta"><span>{item.owner}</span><time>{formatTodayDue(item.due_at)}</time></span>
    </button>)}</div> : <p className="oversight-today-status">No assigned work or permitted operational exceptions are open for you.</p>}
  </section>;
}

function RiskPressure({ snapshot }: { snapshot: OversightSnapshot }) {
  const max = useMemo(() => Math.max(1, ...snapshot.pressure.map((item) => item.critical + item.high + item.other)), [snapshot.pressure]);
  return <div className="oversight-two-column"><article className="oversight-panel"><div className="section-header"><div><h2>Risk pressure by issue type</h2><p>Open issues split by recorded priority.</p></div></div>{snapshot.pressure.length ? <><div className="oversight-bars" aria-hidden="true">{snapshot.pressure.map((item) => { const total = item.critical + item.high + item.other; return <div key={item.category}><span>{humanize(item.category)}</span><div><i className="critical" style={{ width: `${item.critical / max * 100}%` }}/><i className="high" style={{ width: `${item.high / max * 100}%` }}/><i className="other" style={{ width: `${item.other / max * 100}%` }}/></div><strong>{total}</strong></div>; })}</div><div className="oversight-pressure-table"><DataTable ariaLabel="Risk pressure by issue type" rows={snapshot.pressure} rowKey={(item) => item.category} rowName={(item) => humanize(item.category)} columns={pressureColumns}/></div></> : <EmptyMeasure/>}</article><Aging snapshot={snapshot}/></div>;
}

function Aging({ snapshot }: { snapshot: OversightSnapshot }) {
  const total = Math.max(1, snapshot.aging.reduce((sum, item) => sum + item.count, 0));
  return <article className="oversight-panel"><div className="section-header"><div><h2>Open issue aging</h2><p>Elapsed time since each open issue was recorded.</p></div></div><div className="oversight-aging">{snapshot.aging.map((item) => <div key={item.label}><div><strong>{item.label}</strong><span>{item.count}</span></div><progress max={total} value={item.count}>{item.count} of {total}</progress></div>)}</div></article>;
}

function ResolutionOutlook({ snapshot }: { snapshot: OversightSnapshot }) {
  return <div className="oversight-two-column"><article className="oversight-panel oversight-estimates"><div className="section-header"><div><h2>Historical resolution ranges</h2><p>Ranges use completed issues of the same type; they are not promised completion dates.</p></div></div>{snapshot.estimates.length ? snapshot.estimates.map((item) => <div key={item.category}><div><strong>{humanize(item.category)}</strong><span>{item.confidence.toLowerCase()} confidence · {item.sample_size} completed</span></div><p><b>{formatDuration(item.lower_hours)}–{formatDuration(item.upper_hours)}</b><span>Median {formatDuration(item.median_hours)}</span></p><small>{item.estimated_by}</small></div>) : <EmptyState population="Comparable completed issues in this legal entity" title="Not enough completed work for a resolution range" description="At least five comparable completed issues are required before a range is shown."/>}</article><Aging snapshot={snapshot}/></div>;
}

function OperatingPerformance({ snapshot }: { snapshot: OversightSnapshot }) {
  return <article className="oversight-panel"><div className="section-header"><div><h2>Workload and completion context</h2><p>Current workload is shown with completion and handling history from the selected period; this is not an employee ranking.</p></div></div>{snapshot.performance.length ? <DataTable ariaLabel="Owner workload and completion context" rows={snapshot.performance} rowKey={(item) => item.owner_id} rowName={(item) => item.owner_name} columns={performanceColumns}/> : <EmptyState population="Issues with an accountable owner and recorded lifecycle dates" title="No owner-level measures are available" description="Completion and workload measures appear after issues have an accountable owner and recorded lifecycle dates."/>}</article>;
}

function EmptyMeasure() { return <EmptyState population="Open issues in this legal entity" title="No open issues in this measure" description="The current snapshot contains no rows for this breakdown."/>; }

function historyQualityLabel(snapshot: OversightSnapshot) {
  const quality = snapshot.history_quality;
  return `${quality.complete_lifecycle} of ${quality.completed_population} completed issues have complete lifecycle events · ${quality.excluded_from_durations} excluded because an opened or closed event is missing · employee handling time follows each recorded owner assignment; reassignment, return, blocked and reopen counts remain visible separately`;
}

function homeTabDescription(tab: HomeTab, organizationScopeName?: string) {
  const scope = organizationScopeName ? ` in ${organizationScopeName}` : "";
  if (tab === "attention") return `Current conditions requiring intervention${scope}.`;
  if (tab === "my-work") return "Assigned decisions, reviews and evidence requiring your action.";
  return `Current risk posture and operating context${scope}.`;
}

const homeIntentTabs = [
  { id: "oversight", label: "Oversight" },
  { id: "attention", label: "Attention" },
  { id: "my-work", label: "My work" },
] as const;

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


function isAbortError(error: unknown) {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}
