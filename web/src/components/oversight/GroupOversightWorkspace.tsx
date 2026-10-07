import { useEffect, useMemo, useState } from "react";
import type { HomeTab } from "../../appRouting";
import { loadGroupOversight, type GroupChild, type GroupOversightSnapshot } from "../../groupOversightApi";
import { loadGroupPosture, type GroupPostureBundle } from "../../groupPostureApi";
import type { HomeMetricFilter } from "../../homeMetricPresentation";
import { startDateForDays, type ReportingPeriod, type ReportingPeriodQuery } from "../../reportingPeriod";
import { Button, DataTable, EmptyState, Tabs, type RankedBarItem, type DataColumn } from "../ui";
import { OversightPeriodPicker } from "./OversightPeriodPicker";
import { GroupAttentionView, groupAttentionColumns, groupAttentionBasisColumns, groupAttentionMetricValue, groupChildStateLabel, formatGroupTime, formatHighWater } from "./GroupAttentionPanel";
import { GroupPostureView, groupPostureCoverage, groupPostureValue, postureRowMeta, type GroupPostureMetric } from "./GroupPosturePanel";
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

const homeTabs = [
  { id: "oversight", label: "Oversight" },
  { id: "attention", label: "Attention" },
  { id: "my-work", label: "My work" },
] as const;

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
    setPosture(null);
    setPostureState("loading");
    setPeriod({
      start_date: next.start_date,
      end_date: next.end_date,
      mode: "CURRENT_WINDOW",
      max_days: 365,
      historical_end_supported: false,
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
      {selectedHomeTab === "oversight" && posture && <OversightPeriodPicker
        period={period}
        freshness={postureState === "live" && posture && groupPostureCoverage(posture, postureMetric).complete ? "CURRENT" : "STALE"}
        generatedAt={posture.generated_at}
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

    {selectedHomeTab === "attention" && attention && <details className="oversight-data-freshness group-data-basis" onToggle={(event) => setBasisOpen(event.currentTarget.open)}>
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


function reportingPeriod(endDate: string, days: number): ReportingPeriod {
  return {
    start_date: startDateForDays(endDate, days),
    end_date: endDate,
    mode: "CURRENT_WINDOW",
    max_days: 365,
      historical_end_supported: false,
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

