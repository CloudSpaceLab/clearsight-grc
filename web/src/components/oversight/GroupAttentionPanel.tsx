import { homeMetricDetail, homeMetricQuality, homeMetricTone, headlineMetricDefinitions, type HomeMetricFilter } from "../../homeMetricPresentation";
import type { GroupChild, GroupOversightSnapshot } from "../../groupOversightApi";
import type { MetricCompleteness } from "../../metricApi";
import { Button, DataTable, EmptyState, MetricCard, Notice, StatusBadge, type DataColumn, type StatusTone } from "../ui";

type LoadState = "loading" | "live" | "unavailable";

export function GroupAttentionView({
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

export function groupAttentionColumns(): readonly DataColumn<GroupChild>[] {
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

export function groupAttentionBasisColumns(): readonly DataColumn<GroupChild>[] {
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

function GroupAttentionCoverageNotice({ value }: { value: GroupOversightSnapshot }) {
  if (value.coverage.missing_children > 0) {
    return <Notice tone="warning">{value.coverage.missing_children} {value.coverage.missing_children === 1 ? "OpCo has" : "OpCos have"} no current Attention snapshot. Group totals are incomplete.</Notice>;
  }
  if (value.coverage.stale_children > 0) {
    return <Notice tone="warning">{value.coverage.stale_children} {value.coverage.stale_children === 1 ? "OpCo has" : "OpCos have"} stale Attention data.</Notice>;
  }
  return <p className="group-oversight-quality">{value.coverage.authorized_children} OpCos · {value.record_coverage.population} issues checked</p>;
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

export function groupAttentionMetricValue(item: GroupChild, filter: HomeMetricFilter) {
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

export function groupChildStateLabel(state: GroupChild["state"]) {
  if (state === "AVAILABLE") return "Current";
  if (state === "STALE") return "Stale";
  return "No snapshot";
}

function groupChildTone(state: GroupChild["state"]): StatusTone {
  if (state === "AVAILABLE") return "success";
  if (state === "STALE") return "warning";
  return "neutral";
}

export function formatGroupTime(value: string) {
  const parsed = Date.parse(value);
  return Number.isFinite(parsed)
    ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(parsed))
    : "Unavailable";
}

export function formatHighWater(values: Record<string, string> | undefined) {
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
