import { useEffect, useMemo, useState } from "react";
import {
  loadDomainMetricMembers,
  loadLossPeriodMetricMembers,
  type DomainMetricBundle,
  type HomeMetric,
  type HomeMetricMemberPage,
  type LossPeriodBundle,
} from "../../metricApi";
import { homeMetricQuality } from "../../homeMetricPresentation";
import { lossComparisonLabel, lossPeriodValue } from "./lossOverviewPresentation";
import { MetricCard, Notice } from "../ui";
import type { StatusTone } from "../ui";
import { MetricMemberDrill } from "./MetricMemberDrill";

type LoadState = "loading" | "live" | "unavailable";
type SelectedMetric = {
  id: string;
  label: string;
  sourceID: string;
  definitionRevision: string;
  expectedCount: number;
  kind: "domain" | "loss";
};

type Props = {
  bundle: DomainMetricBundle | null;
  state: LoadState;
  lossBundle?: LossPeriodBundle | null;
  lossState?: LoadState;
  organizationScopeID?: string;
  loadMembers?: typeof loadDomainMetricMembers;
  loadLossMembers?: typeof loadLossPeriodMetricMembers;
  onOpenRisk?: (id: string) => void;
  onOpenLoss?: (id: string) => void;
};

const postureOrder = [
  "risks_outside_appetite",
  "indicator_breaches",
  "assurance_failures",
] as const;

const postureDetails: Record<(typeof postureOrder)[number], string> = {
  risks_outside_appetite: "Active risks beyond approved appetite",
  indicator_breaches: "High or critical KRI/KCI breaches",
  assurance_failures: "Risks with failed current assurance",
};

export function DomainPostureSummary({
  bundle,
  state,
  lossBundle = null,
  lossState = "loading",
  organizationScopeID,
  loadMembers = loadDomainMetricMembers,
  loadLossMembers = loadLossPeriodMetricMembers,
  onOpenRisk,
  onOpenLoss,
}: Props) {
  const [selectedID, setSelectedID] = useState<string>();
  const [memberPage, setMemberPage] = useState<HomeMetricMemberPage | null>(null);
  const [memberState, setMemberState] = useState<"idle" | LoadState>("idle");
  const [memberCursors, setMemberCursors] = useState<string[]>([]);
  const [retry, setRetry] = useState(0);

  const metrics = useMemo(
    () => postureOrder.map((id) => bundle?.items.find((item) => item.id === id)),
    [bundle],
  );
  const selectedDomain = selectedID
    ? bundle?.items.find((item) => item.id === selectedID)
    : undefined;
  const lossCard = useMemo(() => lossCardModel(lossBundle), [lossBundle]);
  const selected = useMemo<SelectedMetric | undefined>(() => {
    if (selectedDomain && bundle) {
      return {
        id: selectedDomain.id,
        label: selectedDomain.label,
        sourceID: bundle.source_id,
        definitionRevision: selectedDomain.definition_revision,
        expectedCount: selectedDomain.value,
        kind: "domain",
      };
    }
    if (lossCard && selectedID === lossCard.metricID && lossBundle) {
      return {
        id: lossCard.metricID,
        label: "Net operational loss",
        sourceID: lossBundle.source_id,
        definitionRevision: lossBundle.definition_revision,
        expectedCount: lossCard.expectedCount,
        kind: "loss",
      };
    }
    return undefined;
  }, [bundle, lossBundle, lossCard, selectedDomain, selectedID]);
  const cursor = memberCursors[memberCursors.length - 1];

  useEffect(() => {
    setSelectedID(undefined);
    setMemberPage(null);
    setMemberCursors([]);
    setMemberState("idle");
  }, [bundle?.source_id, lossBundle?.source_id, organizationScopeID]);

  useEffect(() => {
    if (!selected) return;
    const controller = new AbortController();
    setMemberState("loading");
    const request = selected.kind === "loss"
      ? loadLossMembers(
        selected.id as "operational_loss_events" | "operational_loss_net",
        selected.sourceID,
        selected.definitionRevision,
        organizationScopeID,
        cursor,
        50,
        controller.signal,
      )
      : loadMembers(
        selected.id,
        selected.sourceID,
        selected.definitionRevision,
        organizationScopeID,
        cursor,
        50,
        controller.signal,
      );

    void request.then((page) => {
      if (controller.signal.aborted) return;
      const valid = page.source_id === selected.sourceID
        && page.metric_id === selected.id
        && page.definition_revision === selected.definitionRevision
        && page.count === selected.expectedCount;
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
    cursor,
    loadLossMembers,
    loadMembers,
    organizationScopeID,
    retry,
    selected,
  ]);

  function toggleMetric(id: string) {
    if (selectedID === id) {
      setSelectedID(undefined);
      setMemberPage(null);
      setMemberCursors([]);
      setMemberState("idle");
      return;
    }
    setSelectedID(id);
    setMemberPage(null);
    setMemberCursors([]);
  }

  return <section className="domain-posture" aria-labelledby="domain-posture-heading">
    <div className="section-header">
      <div>
        <span className="eyebrow">Current posture</span>
        <h2 id="domain-posture-heading">Material risk signals</h2>
        <p>Current posture and period Loss flow for this scope.</p>
      </div>
    </div>

    <div className="oversight-counts" aria-label="Current risk posture" aria-busy={state === "loading" || undefined}>
      {postureOrder.map((id, index) => {
        const metric = metrics[index];
        if (!metric) {
          return <MetricCard
            key={id}
            label={postureLabel(id)}
            value="—"
            detail={postureDetails[id]}
            quality="unknown"
            qualityLabel={state === "loading" ? "Loading" : "Unavailable"}
          />;
        }
        const active = selectedID === metric.id;
        return <MetricCard
          key={metric.id}
          label={metric.label}
          value={metric.value}
          detail={postureDetails[id]}
          meta={metricMeta(metric)}
          tone={postureTone(metric)}
          quality={homeMetricQuality(metric)}
          actionLabel={active ? "Close records" : "Review records"}
          isSelected={active}
          ariaControls="domain-posture-members"
          onPress={() => toggleMetric(metric.id)}
        />;
      })}

      {!lossCard || !lossBundle
        ? <MetricCard
          label="Net operational loss"
          value="—"
          detail="Loss flow in the selected period"
          quality="unknown"
          qualityLabel={lossState === "loading" ? "Loading" : "Unavailable"}
        />
        : <MetricCard
          label="Net operational loss"
          value={lossCard.value}
          delta={lossCard.delta}
          detail={lossCard.detail}
          meta={lossCard.meta}
          tone="neutral"
          quality="current"
          qualityLabel="Period flow"
          actionLabel={lossCard.expectedCount > 0
            ? selectedID === lossCard.metricID ? "Close records" : "Review records"
            : undefined}
          isSelected={selectedID === lossCard.metricID}
          ariaControls="domain-posture-members"
          onPress={lossCard.expectedCount > 0 ? () => toggleMetric(lossCard.metricID) : undefined}
        />}
    </div>

    {state === "unavailable" && <Notice tone="warning">Current risk posture is unavailable.</Notice>}
    {lossState === "unavailable" && <Notice tone="warning">Loss flow is unavailable for this period.</Notice>}

    {selected && <div id="domain-posture-members" className="domain-posture__members">
      <MetricMemberDrill
        label={selected.label}
        state={memberState === "idle" ? "loading" : memberState}
        page={memberPage}
        hasPrevious={memberCursors.length > 0}
        onPrevious={() => setMemberCursors((value) => value.slice(0, -1))}
        onNext={() => {
          if (!memberPage?.next_cursor) return;
          setMemberCursors((value) => [...value, memberPage.next_cursor!]);
        }}
        onRetry={() => setRetry((value) => value + 1)}
        onOpenRisk={onOpenRisk}
        onOpenLoss={onOpenLoss}
      />
    </div>}
  </section>;
}

function lossCardModel(bundle: LossPeriodBundle | null) {
  if (!bundle) return undefined;
  if (bundle.mixed_currencies) {
    return {
      metricID: "operational_loss_events" as const,
      expectedCount: bundle.event_count,
      value: lossPeriodValue(bundle),
      delta: lossComparisonLabel(bundle),
      detail: "Mixed currencies; amounts kept separate",
      meta: `${bundle.currencies.length} currencies · ${bundle.contributing_loss_count} contributing Losses`,
    };
  }
  if (bundle.net_loss) {
    return {
      metricID: "operational_loss_net" as const,
      expectedCount: bundle.contributing_loss_count,
      value: lossPeriodValue(bundle),
      delta: lossComparisonLabel(bundle),
      detail: "Net Loss flow in the selected period",
      meta: `${bundle.event_count} new ${bundle.event_count === 1 ? "event" : "events"} · ${bundle.contributing_loss_count} contributing Losses`,
    };
  }
  return {
    metricID: "operational_loss_events" as const,
    expectedCount: bundle.event_count,
    value: lossPeriodValue(bundle),
    delta: lossComparisonLabel(bundle),
    detail: "No monetary Loss flow in this period",
    meta: "No gross Loss or recovery activity",
  };
}

function postureLabel(id: (typeof postureOrder)[number]) {
  if (id === "risks_outside_appetite") return "Outside appetite";
  if (id === "indicator_breaches") return "Indicator breaches";
  return "Assurance failures";
}

function postureTone(metric: HomeMetric): StatusTone {
  if (metric.condition === "CLEAR") return "success";
  if (metric.id === "risks_outside_appetite" || metric.id === "assurance_failures") return "error";
  return "warning";
}

function metricMeta(metric: HomeMetric) {
  const unknown = metric.unknown === undefined ? "unknown" : String(metric.unknown);
  return `${metric.population} checked · ${unknown} unknown`;
}

function isAbortError(error: unknown) {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}
