import { useEffect, useMemo, useState } from "react";
import {
  loadDomainMetricMembers,
  type DomainMetricBundle,
  type HomeMetric,
  type HomeMetricMemberPage,
} from "../../metricApi";
import { homeMetricQuality } from "../../homeMetricPresentation";
import { MetricCard, Notice } from "../ui";
import type { StatusTone } from "../ui";
import { MetricMemberDrill } from "./MetricMemberDrill";

type LoadState = "loading" | "live" | "unavailable";

type Props = {
  bundle: DomainMetricBundle | null;
  state: LoadState;
  organizationScopeID?: string;
  loadMembers?: typeof loadDomainMetricMembers;
  onOpenRisk?: (id: string) => void;
  onOpenLoss?: (id: string) => void;
};

const postureOrder = [
  "risks_outside_appetite",
  "indicator_breaches",
  "assurance_failures",
  "losses_without_issue",
] as const;

const postureDetails: Record<(typeof postureOrder)[number], string> = {
  risks_outside_appetite: "Active risks beyond approved appetite",
  indicator_breaches: "High or critical KRI/KCI breaches",
  assurance_failures: "Risks with failed current assurance",
  losses_without_issue: "Active losses without a linked intervention",
};

export function DomainPostureSummary({
  bundle,
  state,
  organizationScopeID,
  loadMembers = loadDomainMetricMembers,
  onOpenRisk,
  onOpenLoss,
}: Props) {
  const [selectedID, setSelectedID] = useState<string>();
  const [memberPage, setMemberPage] = useState<HomeMetricMemberPage | null>(null);
  const [memberState, setMemberState] = useState<"idle" | LoadState>("idle");
  const [memberCursors, setMemberCursors] = useState<string[]>([]);
  const [retry, setRetry] = useState(0);

  const metrics = useMemo(() => postureOrder.map((id) => bundle?.items.find((item) => item.id === id)), [bundle]);
  const selected = selectedID ? bundle?.items.find((item) => item.id === selectedID) : undefined;
  const cursor = memberCursors[memberCursors.length - 1];

  useEffect(() => {
    setSelectedID(undefined);
    setMemberPage(null);
    setMemberCursors([]);
    setMemberState("idle");
  }, [bundle?.source_id, organizationScopeID]);

  useEffect(() => {
    if (!bundle || !selected) return;
    const controller = new AbortController();
    setMemberState("loading");
    void loadMembers(
      selected.id,
      bundle.source_id,
      selected.definition_revision,
      organizationScopeID,
      cursor,
      50,
      controller.signal,
    ).then((page) => {
      if (controller.signal.aborted) return;
      const valid = page.source_id === bundle.source_id
        && page.metric_id === selected.id
        && page.definition_revision === selected.definition_revision
        && page.count === selected.value;
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
  }, [bundle, cursor, loadMembers, organizationScopeID, retry, selected]);

  function selectMetric(metric: HomeMetric) {
    if (selectedID === metric.id) {
      setSelectedID(undefined);
      setMemberPage(null);
      setMemberCursors([]);
      setMemberState("idle");
      return;
    }
    setSelectedID(metric.id);
    setMemberPage(null);
    setMemberCursors([]);
  }

  return <section className="domain-posture" aria-labelledby="domain-posture-heading">
    <div className="section-header">
      <div>
        <span className="eyebrow">Current posture</span>
        <h2 id="domain-posture-heading">Material risk signals</h2>
        <p>Current governed posture for this scope.</p>
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
          onPress={() => selectMetric(metric)}
        />;
      })}
    </div>

    {state === "unavailable" && <Notice tone="warning">Current risk posture is unavailable.</Notice>}

    {selected && bundle && <div id="domain-posture-members" className="domain-posture__members">
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

function postureLabel(id: (typeof postureOrder)[number]) {
  if (id === "risks_outside_appetite") return "Outside appetite";
  if (id === "indicator_breaches") return "Indicator breaches";
  if (id === "assurance_failures") return "Assurance failures";
  return "Losses without issue";
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
