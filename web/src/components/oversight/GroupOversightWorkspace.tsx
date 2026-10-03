import { useEffect, useMemo, useState } from "react";
import { loadGroupOversight, type GroupChild, type GroupOversightSnapshot } from "../../groupOversightApi";
import { homeMetricDetail, homeMetricQuality, homeMetricTone, headlineMetricDefinitions, type HomeMetricFilter } from "../../homeMetricPresentation";
import type { MetricCompleteness } from "../../metricApi";
import { Button, DataTable, EmptyState, MetricCard, Notice, StatusBadge, type DataColumn } from "../ui";
import "./group-oversight.css";

type Props = {
  organizationName: string;
  initialSnapshot?: GroupOversightSnapshot;
  metricFilter?: HomeMetricFilter;
  onMetricFilterChange?: (filter: HomeMetricFilter) => void;
  onOpenLegalEntity: (legalEntityID: string) => void;
  loadGroup?: (signal?: AbortSignal) => Promise<GroupOversightSnapshot>;
};

type LoadState = "loading" | "live" | "unavailable";

export function GroupOversightWorkspace({
  organizationName,
  initialSnapshot,
  metricFilter = "all",
  onMetricFilterChange,
  onOpenLegalEntity,
  loadGroup = loadGroupOversight,
}: Props) {
  const [value, setValue] = useState<GroupOversightSnapshot | undefined>(initialSnapshot);
  const [state, setState] = useState<LoadState>(initialSnapshot ? "live" : "loading");
  const [retry, setRetry] = useState(0);
  const [localFilter, setLocalFilter] = useState<HomeMetricFilter>(metricFilter);
  const selected = onMetricFilterChange ? metricFilter : localFilter;

  useEffect(() => { setLocalFilter(metricFilter); }, [metricFilter]);
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

  function chooseMetric(filter: HomeMetricFilter) {
    const next = selected === filter ? "all" : filter;
    if (onMetricFilterChange) onMetricFilterChange(next);
    else setLocalFilter(next);
    requestAnimationFrame(() => document.getElementById("group-opcos")?.focus());
  }

  const rows = useMemo(() => {
    const items = [...(value?.children ?? [])];
    return items.sort((left, right) => {
      const leftValue = groupMetricValue(left, selected);
      const rightValue = groupMetricValue(right, selected);
      if (leftValue !== rightValue) return rightValue - leftValue;
      if (left.state === "MISSING" && right.state !== "MISSING") return 1;
      if (right.state === "MISSING" && left.state !== "MISSING") return -1;
      return left.legal_entity_name.localeCompare(right.legal_entity_name);
    });
  }, [selected, value?.children]);

  const columns: readonly DataColumn<GroupChild>[] = [
    {
      id: "entity",
      header: "OpCo",
      mobileLayout: "full-width",
      render: (item) => <span className="group-opco__identity"><strong>{item.legal_entity_name}</strong><small>{item.jurisdiction || item.legal_entity_code || "Legal entity"}</small></span>,
      accessibleText: (item) => `${item.legal_entity_name}, ${item.jurisdiction || item.legal_entity_code || "Legal entity"}`,
    },
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
      render: (item) => <StatusBadge tone={item.state === "AVAILABLE" ? "success" : item.state === "STALE" ? "warning" : "neutral"}>{groupChildStateLabel(item.state)}</StatusBadge>,
      accessibleText: (item) => groupChildStateLabel(item.state),
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

  const completeness = value ? groupCompleteness(value) : "UNKNOWN";

  return <section className="group-oversight-page" aria-labelledby="group-oversight-heading">
    <header className="topbar group-oversight-header">
      <div>
        <span className="eyebrow">{organizationName}</span>
        <h1 id="group-oversight-heading">Group posture</h1>
        <p>Authorized OpCos only. Open an OpCo before viewing record detail.</p>
      </div>
    </header>

    {value && <GroupCoverageNotice value={value}/>}

    <div className="oversight-counts" aria-label="Group risk metrics" aria-busy={state === "loading" || undefined}>
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
          meta={groupMetricMeta(value)}
          tone={homeMetricTone({ id: definition.id, condition: metricValue > 0 ? "ATTENTION" : "CLEAR" })}
          quality={homeMetricQuality({ freshness: value.freshness, completeness })}
          actionLabel={active ? "Show all OpCos" : "Compare OpCos"}
          isSelected={active}
          ariaControls="group-opcos"
          onPress={() => chooseMetric(filter)}
        />;
      })}
    </div>

    <section id="group-opcos" className="group-opcos" aria-labelledby="group-opcos-heading" tabIndex={-1}>
      <div className="section-header">
        <div>
          <span className="eyebrow">Legal entities</span>
          <h2 id="group-opcos-heading">{selected === "all" ? "OpCo comparison" : groupFilterLabel(selected)}</h2>
          <p>{value ? `${value.coverage.included_children} of ${value.coverage.authorized_children} OpCos contributing` : "Loading authorized OpCos…"}</p>
        </div>
      </div>
      {value?.children.length
        ? <DataTable
          ariaLabel="Group OpCo posture"
          rows={rows}
          rowKey={(item) => item.legal_entity_id}
          rowName={(item) => `${item.legal_entity_name}, ${groupChildStateLabel(item.state)}`}
          columns={columns}
          onRowAction={(item) => onOpenLegalEntity(item.legal_entity_id)}
          rowActionLabel="Open OpCo"
          isLoading={state === "loading"}
        />
        : state === "loading"
          ? <p className="group-oversight-status" role="status">Loading Group posture…</p>
          : <EmptyState population="Authorized Group legal entities" title="No OpCo posture available" description="No authorized legal entity returned a Group posture row."/>}
    </section>
  </section>;
}

function GroupCoverageNotice({ value }: { value: GroupOversightSnapshot }) {
  if (value.coverage.missing_children > 0) {
    return <Notice tone="warning">{value.coverage.missing_children} {value.coverage.missing_children === 1 ? "OpCo has" : "OpCos have"} no current snapshot. Group totals are incomplete.</Notice>;
  }
  if (value.coverage.stale_children > 0) {
    return <Notice tone="warning">{value.coverage.stale_children} {value.coverage.stale_children === 1 ? "OpCo has" : "OpCos have"} stale posture data.</Notice>;
  }
  return <p className="group-oversight-quality">{value.coverage.authorized_children} OpCos · {value.record_coverage.population} issues checked</p>;
}

function groupCompleteness(value: GroupOversightSnapshot): MetricCompleteness {
  if (value.coverage.missing_children > 0 || value.record_coverage.unknown === undefined) return "UNKNOWN";
  if (value.coverage.stale_children > 0 || value.record_coverage.unknown > 0) return "PARTIAL";
  return "COMPLETE";
}

function groupMetricMeta(value: GroupOversightSnapshot) {
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

function groupMetricValue(item: GroupChild, filter: HomeMetricFilter) {
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
  return "OpCo comparison";
}

function groupChildStateLabel(state: GroupChild["state"]) {
  if (state === "AVAILABLE") return "Current";
  if (state === "STALE") return "Stale";
  return "No snapshot";
}

function knownCount(value: number | undefined) {
  return value === undefined ? "unknown" : String(value);
}
