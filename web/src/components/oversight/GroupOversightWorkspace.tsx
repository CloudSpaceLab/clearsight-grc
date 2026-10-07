import { useEffect, useMemo, useState } from "react";
import type { HomeTab } from "../../appRouting";
import {
  loadGroupOversight,
  type GroupChild,
  type GroupOversightSnapshot,
} from "../../groupOversightApi";
import {
  loadGroupPosture,
  type GroupPostureBundle,
  type GroupPostureChild,
  type GroupPostureCounts,
} from "../../groupPostureApi";
import {
  homeMetricDetail,
  homeMetricQuality,
  homeMetricTone,
  headlineMetricDefinitions,
  type HomeMetricFilter,
} from "../../homeMetricPresentation";
import type { MetricCompleteness } from "../../metricApi";
import {
  startDateForDays,
  type ReportingPeriod,
  type ReportingPeriodQuery,
} from "../../reportingPeriod";
import {
  Button,
  DataTable,
  EmptyState,
  MetricCard,
  Notice,
  RankedBarList,
  StatusBadge,
  Tabs,
  type DataColumn,
  type RankedBarItem,
  type StatusTone,
} from "../ui";
import { OversightPeriodPicker } from "./OversightPeriodPicker";
import "./group-oversight.css";

type Props = {
  organizationName: string;
  initialSnapshot?: GroupOversightSnapshot;
  metricFilter?: HomeMetricFilter;
  onMetricFilterChange?: (filter: HomeMetricFilter) => void;
  homeTab?: HomeTab;
  onHomeTabChange?: (tab: HomeTab) => void;
  onOpenLegalEntity: (legalEntityID: string) => void;
  onOpenWork?: () => void;
  loadGroup?: (signal?: AbortSignal) => Promise<GroupOversightSnapshot>;
  loadPosture?: typeof loadGroupPosture;
  now?: Date;
};

type LoadState = "loading" | "live" | "unavailable";
type GroupPostureMetric = keyof GroupPostureCounts;

const homeTabs = [
  { id: "oversight", label: "Oversight" },
  { id: "attention", label: "Attention" },
  { id: "my-work", label: "My work" },
] as const;

const postureMetrics: ReadonlyArray<{
  id: GroupPostureMetric;
  label: string;
  detail: string;
}> = [
  { id: "risks_outside_appetite", label: "Outside appetite", detail: "Active Risks beyond approved appetite" },
  { id: "indicator_breaches", label: "Indicator breaches", detail: "High or critical KRI/KCI breaches" },
  { id: "assurance_failures", label: "Assurance failures", detail: "Risks with failed current assurance" },
  { id: "loss_events", label: "Loss events", detail: "Operational Loss events in the selected period" },
];

export function GroupOversightWorkspace({
  organizationName,
  initialSnapshot,
  metricFilter = "all",
  onMetricFilterChange,
  homeTab,
  onHomeTabChange,
  onOpenLegalEntity,
  onOpenWork,
  loadGroup = loadGroupOversight,
  loadPosture = loadGroupPosture,
  now,
}: Props) {
  const nowMs = now?.getTime();
  const reportingDate = formatUTCDate(nowMs === undefined ? new Date() : new Date(nowMs));
  const [attention, setAttention] = useState<GroupOversightSnapshot | undefined>(initialSnapshot);
  const [attentionState, setAttentionState] = useState<LoadState>(initialSnapshot ? "live" : "loading");
  const [attentionRetry, setAttentionRetry] = useState(0);
  const [posture, setPosture] = useState<GroupPostureBundle | null>(null);
  const [postureState, setPostureState] = useState<LoadState>("loading");
  const [postureRetry, setPostureRetry] = useState(0);
  const [basisOpen, setBasisOpen] = useState(false);
  const [localFilter, setLocalFilter] = useState<HomeMetricFilter>(metricFilter);
  const [localHomeTab, setLocalHomeTab] = useState<HomeTab>(homeTab ?? (metricFilter === "all" ? "oversight" : "attention"));
  const [postureMetric, setPostureMetric] = useState<GroupPostureMetric>("risks_outside_appetite");
  const [period, setPeriod] = useState<ReportingPeriod>(() => reportingPeriod(reportingDate, 30));
  const selectedAttention = onMetricFilterChange ? metricFilter : localFilter;
  const selectedHomeTab = onHomeTabChange ? (homeTab ?? "oversight") : localHomeTab;

  useEffect(() => { setLocalFilter(metricFilter); }, [metricFilter]);
  useEffect(() => {
    if (homeTab) setLocalHomeTab(homeTab);
    else if (metricFilter !== "all") setLocalHomeTab("attention");
  }, [homeTab, metricFilter]);

  useEffect(() => {
    setPeriod((current) => ({
      ...current,
      end_date: reportingDate,
      start_date: startDateForDays(reportingDate, daysInPeriod(current)),
    }));
  }, [reportingDate]);

  useEffect(() => {
    if (initialSnapshot) {
      setAttention(initialSnapshot);
      setAttentionState("live");
      return;
    }
    const controller = new AbortController();
    setAttentionState("loading");
    void loadGroup(controller.signal).then((next) => {
      if (controller.signal.aborted) return;
      setAttention(next);
      setAttentionState("live");
    }).catch(() => {
      if (controller.signal.aborted) return;
      setAttentionState("unavailable");
    });
    return () => controller.abort();
  }, [attentionRetry, initialSnapshot, loadGroup]);

  useEffect(() => {
    if (selectedHomeTab !== "oversight") return;
    const controller = new AbortController();
    setPostureState("loading");
    void loadPosture(
      { start_date: period.start_date, end_date: period.end_date },
      controller.signal,
    ).then((next) => {
      if (controller.signal.aborted) return;
      setPosture(next);
      setPostureState("live");
    }).catch(() => {
      if (controller.signal.aborted) return;
      setPosture(null);
      setPostureState("unavailable");
    });
    return () => controller.abort();
  }, [loadPosture, period.end_date, period.start_date, postureRetry, selectedHomeTab]);

  function selectHomeTab(tab: HomeTab) {
    if (onHomeTabChange) onHomeTabChange(tab);
    else {
      setLocalHomeTab(tab);
      if (tab !== "attention") setLocalFilter("all");
    }
  }

  function chooseAttentionMetric(filter: HomeMetricFilter) {
    const next = selectedAttention === filter ? "all" : filter;
    if (onMetricFilterChange) onMetricFilterChange(next);
    else setLocalFilter(next);
    requestAnimationFrame(() => document.getElementById("group-attention-opcos")?.focus());
  }

  function applyPeriod(next: ReportingPeriodQuery) {
    setPeriod({
      start_date: next.start_date,
      end_date: next.end_date,
      mode: "CURRENT_WINDOW",
      max_days: 365,
    });
  }

  const attentionRows = useMemo(() => {
    const items = [...(attention?.children ?? [])];
    return items.sort((left, right) => {
      const leftValue = groupAttentionMetricValue(left, selectedAttention);
      const rightValue = groupAttentionMetricValue(right, selectedAttention);
      if (leftValue !== rightValue) return rightValue - leftValue;
      if (left.state === "MISSING" && right.state !== "MISSING") return 1;
      if (right.state === "MISSING" && left.state !== "MISSING") return -1;
      return left.legal_entity_name.localeCompare(right.legal_entity_name);
    });
  }, [attention?.children, selectedAttention]);

  const postureRows = useMemo<RankedBarItem[]>(() => {
    if (!posture) return [];
    const children = posture.children
      .filter((child) => postureMetric === "loss_events" || child.risk_state !== "MISSING")
      .sort((left, right) => {
        const delta = groupPostureValue(right, postureMetric) - groupPostureValue(left, postureMetric);
        if (delta !== 0) return delta;
        if (postureMetric !== "loss_events") {
          if (left.risk_state === "STALE" && right.risk_state !== "STALE") return 1;
          if (right.risk_state === "STALE" && left.risk_state !== "STALE") return -1;
        }
        return left.legal_entity_name.localeCompare(right.legal_entity_name);
      });

    const visible: RankedBarItem[] = children.slice(0, 8).map((child) => ({
      id: child.legal_entity_id,
      label: child.legal_entity_name,
      value: groupPostureValue(child, postureMetric),
      meta: postureRowMeta(child, postureMetric),
      actionLabel: `Open ${child.legal_entity_name}`,
    }));
    if (children.length > 8) {
      visible.push({
        id: "other-opcos",
        label: "Other OpCos",
        value: children.slice(8).reduce((sum, child) => sum + groupPostureValue(child, postureMetric), 0),
        meta: `${children.length - 8} additional OpCos`,
        isDisabled: true,
      });
    }
    return visible;
  }, [posture, postureMetric]);

  const attentionColumns = useMemo<readonly DataColumn<GroupChild>[]>(() => groupAttentionColumns(), []);
  const basisColumns = useMemo<readonly DataColumn<GroupChild>[]>(() => groupAttentionBasisColumns(), []);

  return <section className="group-oversight-page" aria-labelledby="group-oversight-heading">
    <header className="topbar group-oversight-header">
      <div>
        <span className="eyebrow">{organizationName}</span>
        <h1 id="group-oversight-heading">Group Home</h1>
        <p>Authorized OpCo aggregates only. Open an OpCo before viewing record detail.</p>
      </div>
      {selectedHomeTab === "oversight" && <OversightPeriodPicker
        period={period}
        freshness={postureState === "live" && posture && groupPostureCoverage(posture, postureMetric).complete ? "CURRENT" : "STALE"}
        generatedAt={posture?.generated_at ?? new Date(nowMs ?? Date.now()).toISOString()}
        error={postureState === "unavailable" ? "Group CRO posture is unavailable." : undefined}
        onApply={applyPeriod}
      />}
    </header>

    <Tabs ariaLabel="Group Home views" compactLabel="Group Home view" items={homeTabs} selectedKey={selectedHomeTab} onSelectionChange={selectHomeTab}>
      {(tab) => <div className="group-home-panel">
        {tab === "oversight" && <GroupPostureView
          value={posture}
          state={postureState}
          metric={postureMetric}
          period={period}
          onMetricChange={setPostureMetric}
          rows={postureRows}
          onOpenLegalEntity={onOpenLegalEntity}
          onRetry={() => setPostureRetry((current) => current + 1)}
        />}
        {tab === "attention" && <GroupAttentionView
          value={attention}
          state={attentionState}
          selected={selectedAttention}
          rows={attentionRows}
          columns={attentionColumns}
          onMetricChange={chooseAttentionMetric}
          onOpenLegalEntity={onOpenLegalEntity}
          onRetry={() => setAttentionRetry((current) => current + 1)}
        />}
        {tab === "my-work" && <EmptyState
          population="Group scope"
          title="Assigned work stays within an OpCo"
          description="Open an OpCo to work its queue, or return to your current OpCo work list."
          action={onOpenWork ? <Button onPress={onOpenWork}>Open current OpCo My work</Button> : undefined}
        />}
      </div>}
    </Tabs>

    {attention && <details className="oversight-data-freshness group-data-basis" onToggle={(event) => setBasisOpen(event.currentTarget.open)}>
      <summary>Attention data basis · {attention.coverage.authorized_children} authorized {attention.coverage.authorized_children === 1 ? "OpCo" : "OpCos"}</summary>
      {basisOpen && <div>
        <p>Attention uses the retained Group snapshot. CRO posture retains its own per-OpCo source identity.</p>
        <dl className="group-data-basis__group">
          <div><dt>Group revision</dt><dd><code>{attention.revision_id}</code></dd></div>
          <div><dt>Generated</dt><dd><time dateTime={attention.generated_at}>{formatGroupTime(attention.generated_at)}</time></dd></div>
          <div><dt>Projection</dt><dd>{attention.projection_version}</dd></div>
        </dl>
        <DataTable
          ariaLabel="Group Attention data basis"
          rows={attention.children}
          rowKey={(item) => item.legal_entity_id}
          rowName={(item) => `${item.legal_entity_name}, ${groupChildStateLabel(item.state)}`}
          columns={basisColumns}
        />
      </div>}
    </details>}
  </section>;
}

function GroupPostureView({
  value,
  state,
  metric,
  period,
  onMetricChange,
  rows,
  onOpenLegalEntity,
  onRetry,
}: {
  value: GroupPostureBundle | null;
  state: LoadState;
  metric: GroupPostureMetric;
  period: ReportingPeriod;
  onMetricChange: (metric: GroupPostureMetric) => void;
  rows: RankedBarItem[];
  onOpenLegalEntity: (legalEntityID: string) => void;
  onRetry: () => void;
}) {
  if (state === "unavailable" && !value) {
    return <EmptyState
      population="Authorized Group CRO posture"
      title="Group CRO posture unavailable"
      description="Refresh or open an OpCo."
      action={<Button variant="secondary" onPress={onRetry}>Try again</Button>}
      role="alert"
    />;
  }

  return <>
    {value && <GroupPostureCoverageNotice value={value} metric={metric}/>} 
    <div className="oversight-counts" aria-label="Group CRO posture" aria-busy={state === "loading" || undefined}>
      {postureMetrics.map((definition) => {
        const metricValue = value?.counts[definition.id];
        const active = metric === definition.id;
        const quality = value ? groupPostureMetricQuality(value, definition.id) : "unknown";
        return <MetricCard
          key={definition.id}
          label={definition.label}
          value={metricValue ?? "—"}
          detail={definition.detail}
          meta={value ? groupPostureMetricMeta(value, definition.id, period) : undefined}
          tone={metricValue === undefined ? "neutral" : groupPostureTone(definition.id, metricValue, quality)}
          quality={quality}
          qualityLabel={value ? undefined : state === "loading" ? "Loading" : "Unavailable"}
          actionLabel={active ? "Selected" : "Compare OpCos"}
          isSelected={active}
          ariaControls="group-posture-opcos"
          onPress={() => {
            onMetricChange(definition.id);
            requestAnimationFrame(() => document.getElementById("group-posture-opcos")?.focus());
          }}
        />;
      })}
    </div>

    <section id="group-posture-opcos" className="group-opcos" aria-labelledby="group-posture-opcos-heading" tabIndex={-1}>
      <div className="section-header">
        <div>
          <span className="eyebrow">Legal entities</span>
          <h2 id="group-posture-opcos-heading">{postureMetricLabel(metric)} by OpCo</h2>
          <p>{value ? groupPostureCoverageText(value, metric) : "Loading authorized OpCos…"}</p>
        </div>
      </div>
      {rows.length
        ? <RankedBarList
          ariaLabel={`${postureMetricLabel(metric)} by OpCo`}
          items={rows}
          onAction={(item) => {
            if (item.id !== "other-opcos") onOpenLegalEntity(item.id);
          }}
        />
        : state === "loading"
          ? <p className="group-oversight-status" role="status">Loading Group posture…</p>
          : <EmptyState population="Authorized Group legal entities" title="No Group posture available" description="No authorized OpCo returned the selected CRO measure."/>}
    </section>
  </>;
}

function GroupAttentionView({
  value,
  state,
  selected,
  rows,
  columns,
  onMetricChange,
  onOpenLegalEntity,
  onRetry,
}: {
  value?: GroupOversightSnapshot;
  state: LoadState;
  selected: HomeMetricFilter;
  rows: GroupChild[];
  columns: readonly DataColumn<GroupChild>[];
  onMetricChange: (filter: HomeMetricFilter) => void;
  onOpenLegalEntity: (legalEntityID: string) => void;
  onRetry: () => void;
}) {
  if (state === "unavailable" && !value) {
    return <EmptyState
      population="Authorized Group Attention"
      title="Group Attention unavailable"
      description="Refresh or return to the current legal entity."
      action={<Button variant="secondary" onPress={onRetry}>Try again</Button>}
      role="alert"
    />;
  }
  const completeness = value ? groupAttentionCompleteness(value) : "UNKNOWN";
  return <>
    {value && <GroupAttentionCoverageNotice value={value}/>}
    <div className="oversight-counts" aria-label="Group attention metrics" aria-busy={state === "loading" || undefined}>
      {headlineMetricDefinitions.map((definition) => {
        if (!value) return <MetricCard key={definition.id} label={definition.label} value="—" detail={definition.detail} quality="unknown" qualityLabel={state === "loading" ? "Loading" : "Unavailable"}/>;
        const metricValue = groupHeadlineValue(value, definition.id);
        const filter = groupFilterForMetric(definition.id);
        const active = selected === filter;
        return <MetricCard
          key={definition.id}
          label={definition.label}
          value={metricValue}
          detail={homeMetricDetail(definition.id)}
          meta={groupAttentionMetricMeta(value)}
          tone={homeMetricTone({ id: definition.id, condition: metricValue > 0 ? "ATTENTION" : "CLEAR" })}
          quality={homeMetricQuality({ freshness: value.freshness, completeness })}
          actionLabel={active ? "Show all OpCos" : "Compare OpCos"}
          isSelected={active}
          ariaControls="group-attention-opcos"
          onPress={() => onMetricChange(filter)}
        />;
      })}
    </div>

    <section id="group-attention-opcos" className="group-opcos" aria-labelledby="group-attention-opcos-heading" tabIndex={-1}>
      <div className="section-header">
        <div>
          <span className="eyebrow">Legal entities</span>
          <h2 id="group-attention-opcos-heading">{selected === "all" ? "OpCo attention" : groupFilterLabel(selected)}</h2>
          <p>{value ? `${value.coverage.included_children} of ${value.coverage.authorized_children} OpCos contributing` : "Loading authorized OpCos…"}</p>
        </div>
      </div>
      {value?.children.length
        ? <DataTable
          ariaLabel="Group OpCo attention"
          rows={rows}
          rowKey={(item) => item.legal_entity_id}
          rowName={(item) => `${item.legal_entity_name}, ${groupChildStateLabel(item.state)}`}
          columns={columns}
          onRowAction={(item) => onOpenLegalEntity(item.legal_entity_id)}
          rowActionLabel="Open OpCo"
          isLoading={state === "loading"}
        />
        : state === "loading"
          ? <p className="group-oversight-status" role="status">Loading Group attention…</p>
          : <EmptyState population="Authorized Group legal entities" title="No OpCo attention available" description="No authorized legal entity returned an Attention snapshot."/>}
    </section>
  </>;
}

function groupAttentionColumns(): readonly DataColumn<GroupChild>[] {
  return [
    entityColumn(),
    {
      id: "critical",
      header: "Critical & high",
      kind: "number",
      render: (item) => item.state === "MISSING" ? "—" : item.counts.critical_high,
      accessibleText: (item) => item.state === "MISSING" ? "Unavailable" : String(item.counts.critical_high),
    },
    {
      id: "overdue",
      header: "Overdue",
      kind: "number",
      render: (item) => item.state === "MISSING" ? "—" : item.counts.overdue,
      accessibleText: (item) => item.state === "MISSING" ? "Unavailable" : String(item.counts.overdue),
    },
    {
      id: "routing",
      header: "Routing gaps",
      kind: "number",
      render: (item) => item.state === "MISSING" ? "—" : item.counts.routing_failures,
      accessibleText: (item) => item.state === "MISSING" ? "Unavailable" : String(item.counts.routing_failures),
    },
    {
      id: "outcomes",
      header: "Outcome failures",
      kind: "number",
      render: (item) => item.state === "MISSING" ? "—" : item.counts.outcome_failures,
      accessibleText: (item) => item.state === "MISSING" ? "Unavailable" : String(item.counts.outcome_failures),
    },
    {
      id: "quality",
      header: "Data",
      kind: "status",
      render: (item) => <StatusBadge tone={groupChildTone(item.state)}>{groupChildStateLabel(item.state)}</StatusBadge>,
      accessibleText: (item) => groupChildStateLabel(item.state),
    },
  ];
}

function groupAttentionBasisColumns(): readonly DataColumn<GroupChild>[] {
  return [
    entityColumn(),
    {
      id: "data",
      header: "Data",
      kind: "status",
      render: (item) => <StatusBadge tone={groupChildTone(item.state)}>{groupChildStateLabel(item.state)}</StatusBadge>,
      accessibleText: (item) => groupChildStateLabel(item.state),
    },
    {
      id: "snapshot",
      header: "Snapshot revision",
      mobileLayout: "full-width",
      render: (item) => item.state === "MISSING"
        ? <span>No snapshot contributed.</span>
        : <span className="group-data-basis__snapshot"><code>{item.child_snapshot_id || "Unavailable"}</code><small>{formatHighWater(item.source_high_water)}</small></span>,
      accessibleText: (item) => item.state === "MISSING" ? "No snapshot contributed." : `${item.child_snapshot_id || "Unavailable"}. ${formatHighWater(item.source_high_water)}`,
    },
  ];
}

function entityColumn(): DataColumn<GroupChild> {
  return {
    id: "entity",
    header: "OpCo",
    mobileLayout: "full-width",
    render: (item) => <span className="group-opco__identity"><strong>{item.legal_entity_name}</strong><small>{item.jurisdiction || item.legal_entity_code || "Legal entity"}</small></span>,
    accessibleText: (item) => `${item.legal_entity_name}, ${item.jurisdiction || item.legal_entity_code || "Legal entity"}`,
  };
}

function GroupPostureCoverageNotice({ value, metric }: { value: GroupPostureBundle; metric: GroupPostureMetric }) {
  const coverage = groupPostureCoverage(value, metric);
  const label = metric === "loss_events" ? "Loss data" : "Risk posture";
  if (coverage.missing_children > 0) {
    return <Notice tone="warning">{coverage.missing_children} {coverage.missing_children === 1 ? "OpCo has" : "OpCos have"} no current {label}. Group {label.toLowerCase()} is incomplete.</Notice>;
  }
  if (coverage.stale_children > 0 || coverage.partial_children > 0) {
    return <Notice tone="warning">Some OpCo {label.toLowerCase()} is stale or incomplete.</Notice>;
  }
  return <p className="group-oversight-quality">{coverage.authorized_children} OpCos · current {label.toLowerCase()}</p>;
}

function GroupAttentionCoverageNotice({ value }: { value: GroupOversightSnapshot }) {
  if (value.coverage.missing_children > 0) {
    return <Notice tone="warning">{value.coverage.missing_children} {value.coverage.missing_children === 1 ? "OpCo has" : "OpCos have"} no current Attention snapshot. Group totals are incomplete.</Notice>;
  }
  if (value.coverage.stale_children > 0) {
    return <Notice tone="warning">{value.coverage.stale_children} {value.coverage.stale_children === 1 ? "OpCo has" : "OpCos have"} stale Attention data.</Notice>;
  }
  return <p className="group-oversight-quality">{value.coverage.authorized_children} OpCos · {value.record_coverage.population} issues checked</p>;
}

function groupPostureCoverage(value: GroupPostureBundle, metric: GroupPostureMetric) {
  return metric === "loss_events" ? value.loss_coverage : value.risk_coverage;
}

function groupPostureMetricQuality(value: GroupPostureBundle, metric: GroupPostureMetric) {
  const coverage = groupPostureCoverage(value, metric);
  if (coverage.missing_children > 0) return "unknown" as const;
  if (coverage.stale_children > 0 || coverage.partial_children > 0) return "partial" as const;
  return "current" as const;
}

function groupPostureMetricMeta(value: GroupPostureBundle, metric: GroupPostureMetric, period: ReportingPeriod) {
  const coverage = groupPostureCoverage(value, metric);
  if (metric === "loss_events") {
    return `${coverage.included_children} OpCos · ${daysInPeriod(period)} days`;
  }
  return `${coverage.included_children} contributing · ${coverage.missing_children} missing`;
}

function groupPostureCoverageText(value: GroupPostureBundle, metric: GroupPostureMetric) {
  const coverage = groupPostureCoverage(value, metric);
  return `${coverage.included_children} of ${coverage.authorized_children} OpCos contributing`;
}

function groupPostureTone(
  metric: GroupPostureMetric,
  value: number,
  quality: ReturnType<typeof groupPostureMetricQuality>,
): StatusTone {
  if (quality !== "current") return "neutral";
  if (value === 0) return "success";
  if (metric === "indicator_breaches" || metric === "loss_events") return "warning";
  return "error";
}

function groupPostureValue(item: GroupPostureChild, metric: GroupPostureMetric) {
  return item.counts[metric];
}

function postureRowMeta(item: GroupPostureChild, metric: GroupPostureMetric) {
  if (metric === "loss_events") return item.jurisdiction || item.legal_entity_code || "OpCo";
  return [item.jurisdiction || item.legal_entity_code || "OpCo", groupRiskStateLabel(item.risk_state)].join(" · ");
}

function postureMetricLabel(metric: GroupPostureMetric) {
  if (metric === "risks_outside_appetite") return "Outside appetite";
  if (metric === "indicator_breaches") return "Indicator breaches";
  if (metric === "assurance_failures") return "Assurance failures";
  return "Loss events";
}

function groupAttentionCompleteness(value: GroupOversightSnapshot): MetricCompleteness {
  if (value.coverage.missing_children > 0 || value.record_coverage.unknown === undefined) return "UNKNOWN";
  if (value.coverage.stale_children > 0 || value.record_coverage.unknown > 0) return "PARTIAL";
  return "COMPLETE";
}

function groupAttentionMetricMeta(value: GroupOversightSnapshot) {
  return `${value.record_coverage.population} checked · ${knownCount(value.record_coverage.excluded)} excluded · ${knownCount(value.record_coverage.unknown)} unknown`;
}

function groupHeadlineValue(value: GroupOversightSnapshot, id: string) {
  if (id === "critical_high_open") return value.counts.critical_high;
  if (id === "overdue_open") return value.counts.overdue;
  if (id === "routing_gaps") return value.counts.routing_failures;
  return value.counts.outcome_failures;
}

function groupFilterForMetric(id: string): Exclude<HomeMetricFilter, "all"> {
  if (id === "critical_high_open") return "critical-high";
  if (id === "overdue_open") return "overdue";
  if (id === "routing_gaps") return "routing-gaps";
  return "outcome-failures";
}

function groupAttentionMetricValue(item: GroupChild, filter: HomeMetricFilter) {
  if (item.state === "MISSING" || filter === "all") return 0;
  if (filter === "critical-high") return item.counts.critical_high;
  if (filter === "overdue") return item.counts.overdue;
  if (filter === "routing-gaps") return item.counts.routing_failures;
  return item.counts.outcome_failures;
}

function groupFilterLabel(filter: HomeMetricFilter) {
  if (filter === "critical-high") return "Critical and high by OpCo";
  if (filter === "overdue") return "Overdue by OpCo";
  if (filter === "routing-gaps") return "Routing gaps by OpCo";
  if (filter === "outcome-failures") return "Outcome failures by OpCo";
  return "OpCo attention";
}

function groupRiskStateLabel(state: GroupPostureChild["risk_state"]) {
  if (state === "AVAILABLE") return "Current";
  if (state === "STALE") return "Stale";
  return "No Risk posture";
}

function groupChildStateLabel(state: GroupChild["state"]) {
  if (state === "AVAILABLE") return "Current";
  if (state === "STALE") return "Stale";
  return "No snapshot";
}

function groupChildTone(state: GroupChild["state"]): StatusTone {
  if (state === "AVAILABLE") return "success";
  if (state === "STALE") return "warning";
  return "neutral";
}

function reportingPeriod(endDate: string, days: number): ReportingPeriod {
  return {
    start_date: startDateForDays(endDate, days),
    end_date: endDate,
    mode: "CURRENT_WINDOW",
    max_days: 365,
  };
}

function daysInPeriod(value: ReportingPeriod) {
  const start = Date.parse(`${value.start_date}T00:00:00Z`);
  const end = Date.parse(`${value.end_date}T00:00:00Z`);
  return Math.round((end - start) / 86_400_000) + 1;
}

function formatUTCDate(value: Date) {
  return value.toISOString().slice(0, 10);
}

function formatGroupTime(value: string) {
  const parsed = Date.parse(value);
  return Number.isFinite(parsed)
    ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(parsed))
    : "Unavailable";
}

function formatHighWater(values: Record<string, string> | undefined) {
  const entries = Object.entries(values ?? {}).sort(([left], [right]) => left.localeCompare(right));
  if (!entries.length) return "Not recorded";
  return entries.map(([source, at]) => `${humanizeSource(source)} ${formatGroupTime(at)}`).join(" · ");
}

function humanizeSource(value: string) {
  return value.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function knownCount(value: number | undefined) {
  return value === undefined ? "unknown" : String(value);
}
