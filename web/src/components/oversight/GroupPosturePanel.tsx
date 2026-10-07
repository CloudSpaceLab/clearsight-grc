import type { ReportingPeriod } from "../../reportingPeriod";
import { utcDayDifference } from "../../reportingPeriod";
import type { GroupPostureBundle, GroupPostureChild, GroupPostureCounts } from "../../groupPostureApi";
import { Button, EmptyState, MetricCard, Notice, RankedBarList, type RankedBarItem, type StatusTone } from "../ui";

type LoadState = "loading" | "live" | "unavailable";
export type GroupPostureMetric = keyof GroupPostureCounts;

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

export function GroupPostureView({
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
  if (metric === "loss_events") return "neutral";
  if (value === 0) return "success";
  if (metric === "indicator_breaches" || metric === "loss_events") return "warning";
  return "error";
}

export function groupPostureValue(item: GroupPostureChild, metric: GroupPostureMetric) {
  return item.counts[metric];
}

export function postureRowMeta(item: GroupPostureChild, metric: GroupPostureMetric) {
  if (metric === "loss_events") return item.jurisdiction || item.legal_entity_code || "OpCo";
  return [item.jurisdiction || item.legal_entity_code || "OpCo", groupRiskStateLabel(item.risk_state)].join(" · ");
}

function postureMetricLabel(metric: GroupPostureMetric) {
  if (metric === "risks_outside_appetite") return "Outside appetite";
  if (metric === "indicator_breaches") return "Indicator breaches";
  if (metric === "assurance_failures") return "Assurance failures";
  return "Loss events";
}

function groupRiskStateLabel(state: GroupPostureChild["risk_state"]) {
  if (state === "AVAILABLE") return "Current";
  if (state === "STALE") return "Stale";
  return "No Risk posture";
}


function daysInPeriod(period: ReportingPeriod) {
  return utcDayDifference(period.start_date, period.end_date) + 1;
}
