import { useEffect, useMemo, useState } from "react";
import { loadGroupOversight, type GroupChild, type GroupOversightResponse } from "../../groupOversightApi";
import { homeMetricDetail, homeMetricFilter, homeMetricMeta, homeMetricQuality, homeMetricTone, headlineMetricDefinitions, type HomeMetricFilter } from "../../homeMetricPresentation";
import { Button, DataTable, EmptyState, MetricCard, Notice, StatusBadge, type DataColumn } from "../ui";
import "./group-oversight.css";

type Props = {
  organizationName: string;
  metricFilter?: HomeMetricFilter;
  onMetricFilterChange?: (filter: HomeMetricFilter) => void;
  onOpenLegalEntity: (legalEntityID: string) => void;
  loadGroup?: (signal?: AbortSignal) => Promise<GroupOversightResponse>;
};

type LoadState = "loading" | "live" | "unavailable";

export function GroupOversightWorkspace({
  organizationName,
  metricFilter = "all",
  onMetricFilterChange,
  onOpenLegalEntity,
  loadGroup = loadGroupOversight,
}: Props) {
  const [value, setValue] = useState<GroupOversightResponse>();
  const [state, setState] = useState<LoadState>("loading");
  const [retry, setRetry] = useState(0);
  const [localFilter, setLocalFilter] = useState<HomeMetricFilter>(metricFilter);
  const selected = onMetricFilterChange ? metricFilter : localFilter;

  useEffect(() => { setLocalFilter(metricFilter); }, [metricFilter]);
  useEffect(() => {
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
  }, [loadGroup, retry]);

  function chooseMetric(filter: HomeMetricFilter) {
    const next = selected === filter ? "all" : filter;
    if (onMetricFilterChange) onMetricFilterChange(next);
    else setLocalFilter(next);
    requestAnimationFrame(() => document.getElementById("group-opcos")?.focus());
  }

  const rows = useMemo(() => {
    const items = [...(value?.snapshot.children ?? [])];
    return items.sort((left, right) => {
      const leftValue = groupMetricValue(left, selected);
      const rightValue = groupMetricValue(right, selected);
      if (leftValue !== rightValue) return rightValue - leftValue;
      if (left.state === "MISSING" && right.state !== "MISSING") return 1;
      if (right.state === "MISSING" && left.state !== "MISSING") return -1;
      return left.name.localeCompare(right.name);
    });
  }, [selected, value?.snapshot.children]);

  const columns: readonly DataColumn<GroupChild>[] = [
    {
      id: "entity",
      header: "OpCo",
      mobileLayout: "full-width",
      render: (item) => <span className="group-opco__identity"><strong>{item.name}</strong><small>{item.jurisdiction || item.code || "Legal entity"}</small></span>,
      accessibleText: (item) => `${item.name}, ${item.jurisdiction || item.code || "Legal entity"}`,
    },
    {
      id: "critical",
      header: "Critical & high",
      kind: "number",
      render: (item) => item.counts?.critical_high ?? "—",
      accessibleText: (item) => item.counts ? String(item.counts.critical_high) : "Unavailable",
    },
    {
      id: "overdue",
      header: "Overdue",
      kind: "number",
      render: (item) => item.counts?.overdue ?? "—",
      accessibleText: (item) => item.counts ? String(item.counts.overdue) : "Unavailable",
    },
    {
      id: "routing",
      header: "Routing gaps",
      kind: "number",
      render: (item) => item.counts?.routing_failures ?? "—",
      accessibleText: (item) => item.counts ? String(item.counts.routing_failures) : "Unavailable",
    },
    {
      id: "outcomes",
      header: "Outcome failures",
      kind: "number",
      render: (item) => item.counts?.outcome_failures ?? "—",
      accessibleText: (item) => item.counts ? String(item.counts.outcome_failures) : "Unavailable",
    },
    {
      id: "quality",
      header: "Data",
      kind: "status",
      render: (item) => <StatusBadge tone={item.state === "CURRENT" ? "success" : item.state === "STALE" ? "warning" : "neutral"}>{groupChildStateLabel(item.state)}</StatusBadge>,
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
        const metric = value?.metrics.items.find((item) => item.id === definition.id);
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
          actionLabel={filter === "all" ? undefined : active ? "Show all OpCos" : "Compare OpCos"}
          isSelected={active}
          ariaControls={filter === "all" ? undefined : "group-opcos"}
          onPress={filter === "all" ? undefined : () => chooseMetric(filter)}
        />;
      })}
    </div>

    <section id="group-opcos" className="group-opcos" aria-labelledby="group-opcos-heading" tabIndex={-1}>
      <div className="section-header">
        <div>
          <span className="eyebrow">Legal entities</span>
          <h2 id="group-opcos-heading">{selected === "all" ? "OpCo comparison" : groupFilterLabel(selected)}</h2>
          <p>{value ? `${value.snapshot.coverage.contributing_children} of ${value.snapshot.coverage.authorized_children} OpCos contributing` : "Loading authorized OpCos…"}</p>
        </div>
      </div>
      {value?.snapshot.children.length
        ? <DataTable
          ariaLabel="Group OpCo posture"
          rows={rows}
          rowKey={(item) => item.legal_entity_id}
          rowName={(item) => `${item.name}, ${groupChildStateLabel(item.state)}`}
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

function GroupCoverageNotice({ value }: { value: GroupOversightResponse }) {
  const coverage = value.snapshot.coverage;
  if (coverage.missing_children > 0) {
    return <Notice tone="warning">{coverage.missing_children} {coverage.missing_children === 1 ? "OpCo has" : "OpCos have"} no current snapshot. Group totals are incomplete.</Notice>;
  }
  if (coverage.stale_children > 0) {
    return <Notice tone="warning">{coverage.stale_children} {coverage.stale_children === 1 ? "OpCo has" : "OpCos have"} stale posture data.</Notice>;
  }
  return <p className="group-oversight-quality">{coverage.authorized_children} OpCos · {coverage.population} issues checked</p>;
}

function groupMetricValue(item: GroupChild, filter: HomeMetricFilter) {
  if (!item.counts || filter === "all") return 0;
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
  if (state === "CURRENT") return "Current";
  if (state === "STALE") return "Stale";
  return "No snapshot";
}
