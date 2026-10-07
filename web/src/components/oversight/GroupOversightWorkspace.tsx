import { useEffect, useMemo, useState } from "react";
import type { HomeTab } from "../../appRouting";
import {
  loadGroupOversight,
  type GroupChild,
  type GroupDomainPosture,
  type GroupOversightSnapshot,
} from "../../groupOversightApi";
import {
  homeMetricDetail,
  homeMetricQuality,
  homeMetricTone,
  headlineMetricDefinitions,
  type HomeMetricFilter,
} from "../../homeMetricPresentation";
import type { MetricCompleteness } from "../../metricApi";
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
};

type LoadState = "loading" | "live" | "unavailable";
type GroupPostureMetric = keyof GroupDomainPosture;

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
}: Props) {
  const [value, setValue] = useState<GroupOversightSnapshot | undefined>(initialSnapshot);
  const [state, setState] = useState<LoadState>(initialSnapshot ? "live" : "loading");
  const [retry, setRetry] = useState(0);
  const [basisOpen, setBasisOpen] = useState(false);
  const [localFilter, setLocalFilter] = useState<HomeMetricFilter>(metricFilter);
  const [localHomeTab, setLocalHomeTab] = useState<HomeTab>(homeTab ?? (metricFilter === "all" ? "oversight" : "attention"));
  const [postureMetric, setPostureMetric] = useState<GroupPostureMetric>("risks_outside_appetite");
  const selectedAttention = onMetricFilterChange ? metricFilter : localFilter;
  const selectedHomeTab = onHomeTabChange ? (homeTab ?? "oversight") : localHomeTab;

  useEffect(() => { setLocalFilter(metricFilter); }, [metricFilter]);
  useEffect(() => {
    if (homeTab) setLocalHomeTab(homeTab);
    else if (metricFilter !== "all") setLocalHomeTab("attention");
  }, [homeTab, metricFilter]);

  useEffect(() => {
    if (initialSnapshot) {
      setValue(initialSnapshot);
      setState("live");
      return;
    }
    const controller = new AbortController();
    setState("loading");
    void loadGroup(controller.signal).then((next) => {
      if (controller.signal.aborted) return;
      setValue(next);
      setState("live");
    }).catch(() => {
      if (controller.signal.aborted) return;
      setState("unavailable");
    });
    return () => controller.abort();
  }, [initialSnapshot, loadGroup, retry]);

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

  const attentionRows = useMemo(() => {
    const items = [...(value?.children ?? [])];
    return items.sort((left, right) => {
      const leftValue = groupAttentionMetricValue(left, selectedAttention);
      const rightValue = groupAttentionMetricValue(right, selectedAttention);
      if (leftValue !== rightValue) return rightValue - leftValue;
      if (left.state === "MISSING" && right.state !== "MISSING") return 1;
      if (right.state === "MISSING" && left.state !== "MISSING") return -1;
      return left.legal_entity_name.localeCompare(right.legal_entity_name);
    });
  }, [selectedAttention, value?.children]);

  const postureRows = useMemo<RankedBarItem[]>(() => {
    const available = (value?.children ?? [])
      .filter((child) => child.domain_state !== "MISSING")
      .sort((left, right) => {
        const delta = groupPostureValue(right, postureMetric) - groupPostureValue(left, postureMetric);
        if (delta !== 0) return delta;
        if (left.domain_state === "STALE" && right.domain_state !== "STALE") return 1;
        if (right.domain_state === "STALE" && left.domain_state !== "STALE") return -1;
        return left.legal_entity_name.localeCompare(right.legal_entity_name);
      });

    const visible = available.slice(0, 8).map((child) => ({
      id: child.legal_entity_id,
      label: child.legal_entity_name,
      value: groupPostureValue(child, postureMetric),
      meta: [child.jurisdiction || child.legal_entity_code || "OpCo", groupChildStateLabel(child.domain_state)].join(" · "),
      actionLabel: `Open ${child.legal_entity_name}`,
    }));
    if (available.length > 8) {
      visible.push({
        id: "other-opcos",
        label: "Other OpCos",
        value: available.slice(8).reduce((sum, child) => sum + groupPostureValue(child, postureMetric), 0),
        meta: `${available.length - 8} additional OpCos`,
        isDisabled: true,
      });
    }
    return visible;
  }, [postureMetric, value?.children]);

  const attentionColumns: readonly DataColumn<GroupChild>[] = [
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

  const basisColumns: readonly DataColumn<GroupChild>[] = [
    entityColumn(),
    {
      id: "attention-data",
      header: "Attention data",
      kind: "status",
      render: (item) => <StatusBadge tone={groupChildTone(item.state)}>{groupChildStateLabel(item.state)}</StatusBadge>,
      accessibleText: (item) => groupChildStateLabel(item.state),
    },
    {
      id: "posture-data",
      header: "CRO posture",
      kind: "status",
      render: (item) => <StatusBadge tone={groupChildTone(item.domain_state)}>{groupChildStateLabel(item.domain_state)}</StatusBadge>,
      accessibleText: (item) => groupChildStateLabel(item.domain_state),
    },
    {
      id: "attention-source",
      header: "Attention source",
      mobileLayout: "full-width",
      render: (item) => item.state === "MISSING"
        ? <span>No snapshot contributed.</span>
        : <SourceBasis sourceID={item.child_snapshot_id} generatedAt={item.child_generated_at} highWater={item.source_high_water}/>,
      accessibleText: (item) => item.state === "MISSING"
        ? "No attention snapshot contributed."
        : `${item.child_snapshot_id || "Unavailable"}. ${formatHighWater(item.source_high_water)}`,
    },
    {
      id: "posture-source",
      header: "CRO source",
      mobileLayout: "full-width",
      render: (item) => item.domain_state === "MISSING"
        ? <span>No CRO posture contributed.</span>
        : <SourceBasis sourceID={item.domain_source_id} generatedAt={item.domain_generated_at} highWater={item.domain_source_high_water}/>,
      accessibleText: (item) => item.domain_state === "MISSING"
        ? "No CRO posture contributed."
        : `${item.domain_source_id || "Unavailable"}. ${formatHighWater(item.domain_source_high_water)}`,
    },
  ];

  if (state === "unavailable" && !value) {
    return <section className="group-oversight-page">
      <EmptyState
        population="Authorized Group posture"
        title="Group posture unavailable"
        description="Refresh or return to the current legal entity."
        action={<Button variant="secondary" onPress={() => setRetry((current) => current + 1)}>Try again</Button>}
        role="alert"
      />
    </section>;
  }

  return <section className="group-oversight-page" aria-labelledby="group-oversight-heading">
    <header className="topbar group-oversight-header">
      <div>
        <span className="eyebrow">{organizationName}</span>
        <h1 id="group-oversight-heading">Group Home</h1>
        <p>Authorized OpCo aggregates only. Open an OpCo before viewing record detail.</p>
      </div>
    </header>

    <Tabs ariaLabel="Group Home views" compactLabel="Group Home view" items={homeTabs} selectedKey={selectedHomeTab} onSelectionChange={selectHomeTab}>
      {(tab) => <div className="group-home-panel">
        {tab === "oversight" && <GroupPostureView
          value={value}
          state={state}
          metric={postureMetric}
          onMetricChange={setPostureMetric}
          rows={postureRows}
          onOpenLegalEntity={onOpenLegalEntity}
        />}
        {tab === "attention" && <GroupAttentionView
          value={value}
          state={state}
          selected={selectedAttention}
          rows={attentionRows}
          columns={attentionColumns}
          onMetricChange={chooseAttentionMetric}
          onOpenLegalEntity={onOpenLegalEntity}
        />}
        {tab === "my-work" && <EmptyState
          population="Group scope"
          title="Assigned work stays within an OpCo"
          description="Open an OpCo to work its queue, or return to your current OpCo work list."
          action={onOpenWork ? <Button onPress={onOpenWork}>Open current OpCo My work</Button> : undefined}
        />}
      </div>}
    </Tabs>

    {value && <details className="oversight-data-freshness group-data-basis" onToggle={(event) => setBasisOpen(event.currentTarget.open)}>
      <summary>Data basis · {value.coverage.authorized_children} authorized {value.coverage.authorized_children === 1 ? "OpCo" : "OpCos"}</summary>
      {basisOpen && <div>
        <p>Attention and CRO posture retain separate child source revisions.</p>
        <dl className="group-data-basis__group">
          <div><dt>Group revision</dt><dd><code>{value.revision_id}</code></dd></div>
          <div><dt>Generated</dt><dd><time dateTime={value.generated_at}>{formatGroupTime(value.generated_at)}</time></dd></div>
          <div><dt>Projection</dt><dd>{value.projection_version}</dd></div>
        </dl>
        <DataTable
          ariaLabel="Group data basis"
          rows={value.children}
          rowKey={(item) => item.legal_entity_id}
          rowName={(item) => `${item.legal_entity_name}, CRO ${groupChildStateLabel(item.domain_state)}, Attention ${groupChildStateLabel(item.state)}`}
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
  onMetricChange,
  rows,
  onOpenLegalEntity,
}: {
  value?: GroupOversightSnapshot;
  state: LoadState;
  metric: GroupPostureMetric;
  onMetricChange: (metric: GroupPostureMetric) => void;
  rows: RankedBarItem[];
  onOpenLegalEntity: (legalEntityID: string) => void;
}) {
  const completeness = value ? groupPostureCompleteness(value) : "UNKNOWN";
  return <>
    {value && <GroupPostureCoverageNotice value={value}/>}
    <div className="oversight-counts" aria-label="Group CRO posture" aria-busy={state === "loading" || undefined}>
      {postureMetrics.map((definition) => {
        const metricValue = value?.posture[definition.id];
        const active = metric === definition.id;
        return <MetricCard
          key={definition.id}
          label={definition.label}
          value={metricValue ?? "—"}
          detail={definition.detail}
          meta={value ? groupPostureMeta(value) : undefined}
          tone={metricValue === undefined ? "neutral" : groupPostureTone(definition.id, metricValue, completeness)}
          quality={value ? homeMetricQuality({ freshness: value.posture_freshness, completeness }) : "unknown"}
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
          <p>{value ? `${value.posture_coverage.included_children} of ${value.posture_coverage.authorized_children} OpCos contributing` : "Loading authorized OpCos…"}</p>
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
          : <EmptyState population="Authorized Group legal entities" title="No CRO posture available" description="No authorized OpCo returned a current domain posture source."/>}
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
}: {
  value?: GroupOversightSnapshot;
  state: LoadState;
  selected: HomeMetricFilter;
  rows: GroupChild[];
  columns: readonly DataColumn<GroupChild>[];
  onMetricChange: (filter: HomeMetricFilter) => void;
  onOpenLegalEntity: (legalEntityID: string) => void;
}) {
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

function entityColumn(): DataColumn<GroupChild> {
  return {
    id: "entity",
    header: "OpCo",
    mobileLayout: "full-width",
    render: (item) => <span className="group-opco__identity"><strong>{item.legal_entity_name}</strong><small>{item.jurisdiction || item.legal_entity_code || "Legal entity"}</small></span>,
    accessibleText: (item) => `${item.legal_entity_name}, ${item.jurisdiction || item.legal_entity_code || "Legal entity"}`,
  };
}

function SourceBasis({
  sourceID,
  generatedAt,
  highWater,
}: {
  sourceID?: string;
  generatedAt?: string;
  highWater?: Record<string, string>;
}) {
  return <span className="group-data-basis__snapshot">
    <code>{sourceID || "Unavailable"}</code>
    <small>{generatedAt ? formatGroupTime(generatedAt) : "Capture time unavailable"}</small>
    <small>{formatHighWater(highWater)}</small>
  </span>;
}

function GroupPostureCoverageNotice({ value }: { value: GroupOversightSnapshot }) {
  if (value.posture_coverage.missing_children > 0) {
    return <Notice tone="warning">{value.posture_coverage.missing_children} {value.posture_coverage.missing_children === 1 ? "OpCo has" : "OpCos have"} no current CRO posture. Group posture is incomplete.</Notice>;
  }
  if (value.posture_coverage.stale_children > 0) {
    return <Notice tone="warning">{value.posture_coverage.stale_children} {value.posture_coverage.stale_children === 1 ? "OpCo has" : "OpCos have"} stale CRO posture.</Notice>;
  }
  return <p className="group-oversight-quality">{value.posture_coverage.authorized_children} OpCos · current CRO posture</p>;
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

function groupPostureCompleteness(value: GroupOversightSnapshot): MetricCompleteness {
  if (value.posture_coverage.missing_children > 0) return "UNKNOWN";
  if (value.posture_coverage.stale_children > 0) return "PARTIAL";
  return "COMPLETE";
}

function groupAttentionCompleteness(value: GroupOversightSnapshot): MetricCompleteness {
  if (value.coverage.missing_children > 0 || value.record_coverage.unknown === undefined) return "UNKNOWN";
  if (value.coverage.stale_children > 0 || value.record_coverage.unknown > 0) return "PARTIAL";
  return "COMPLETE";
}

function groupPostureMeta(value: GroupOversightSnapshot) {
  return `${value.posture_coverage.included_children} contributing · ${value.posture_coverage.missing_children} missing`;
}

function groupAttentionMetricMeta(value: GroupOversightSnapshot) {
  return `${value.record_coverage.population} checked · ${knownCount(value.record_coverage.excluded)} excluded · ${knownCount(value.record_coverage.unknown)} unknown`;
}

function groupPostureTone(metric: GroupPostureMetric, value: number, completeness: MetricCompleteness): StatusTone {
  if (completeness !== "COMPLETE") return "neutral";
  if (value === 0) return "success";
  return metric === "indicator_breaches" ? "warning" : "error";
}

function groupPostureValue(item: GroupChild, metric: GroupPostureMetric) {
  return item.domain_state === "MISSING" ? 0 : item.domain_posture[metric];
}

function postureMetricLabel(metric: GroupPostureMetric) {
  if (metric === "risks_outside_appetite") return "Outside appetite";
  if (metric === "indicator_breaches") return "Indicator breaches";
  return "Assurance failures";
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
