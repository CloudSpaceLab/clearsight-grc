import { useEffect, useMemo, useState } from "react";
import {
  loadDomainMetricOrganizationBreakdown,
  type DomainMetricBundle,
  type OrganizationMetricBreakdown,
  type OrganizationMetricBucket,
} from "../../metricApi";
import { EmptyState, Notice, RankedBarList, type RankedBarItem } from "../ui";

type LoadState = "idle" | "loading" | "live" | "unavailable";

type Props = {
  bundle: DomainMetricBundle | null;
  organizationScopeID?: string;
  loadBreakdown?: typeof loadDomainMetricOrganizationBreakdown;
  onOpenScope?: (scopeID: string) => void;
};

export function OrganizationRiskSummary({
  bundle,
  organizationScopeID,
  loadBreakdown = loadDomainMetricOrganizationBreakdown,
  onOpenScope,
}: Props) {
  const metric = bundle?.items.find((item) => item.id === "risks_outside_appetite");
  const [value, setValue] = useState<OrganizationMetricBreakdown | null>(null);
  const [state, setState] = useState<LoadState>("idle");

  useEffect(() => {
    if (!bundle || !metric) {
      setValue(null);
      setState("idle");
      return;
    }
    const controller = new AbortController();
    setState("loading");
    setValue(null);
    void loadBreakdown(
      metric.id,
      bundle.source_id,
      metric.definition_revision,
      organizationScopeID,
      controller.signal,
    ).then((next) => {
      if (controller.signal.aborted) return;
      const valid = next.source_id === bundle.source_id
        && next.metric_id === metric.id
        && next.definition_revision === metric.definition_revision
        && next.count === metric.value
        && (next.scope_id ?? "") === (organizationScopeID ?? "");
      if (!valid) {
        setState("unavailable");
        return;
      }
      setValue(next);
      setState("live");
    }).catch((error: unknown) => {
      if (controller.signal.aborted || isAbortError(error)) return;
      setValue(null);
      setState("unavailable");
    });
    return () => controller.abort();
  }, [bundle, loadBreakdown, metric, organizationScopeID]);

  const displayItems = useMemo(() => compactBuckets(value?.items ?? []), [value?.items]);
  const rankedItems = useMemo<RankedBarItem[]>(() => displayItems.map((item) => ({
    id: item.key,
    label: item.label,
    value: item.value,
    meta: bucketMeta(item),
    actionLabel: item.scope_id ? `Open ${item.label}` : undefined,
    isDisabled: !item.scope_id,
  })), [displayItems]);

  return <section className="organization-risk-summary" aria-labelledby="organization-risk-summary-heading">
    <div className="section-header">
      <div>
        <span className="eyebrow">Where risk is concentrated</span>
        <h2 id="organization-risk-summary-heading">Risk concentration</h2>
        <p>Outside-appetite risks by area.</p>
      </div>
    </div>

    {state === "loading" && <p className="oversight-today-status" role="status" aria-busy="true">Loading risk concentration…</p>}
    {state === "unavailable" && <Notice tone="warning">Risk concentration is unavailable for this scope.</Notice>}
    {state === "live" && value?.count === 0 && <EmptyState
      population="Current outside-appetite risk population"
      title="No outside-appetite risks"
      description="No current risk in this scope is outside appetite."
    />}
    {state === "live" && value && value.count > 0 && <RankedBarList
      ariaLabel="Outside-appetite risks by organization area"
      items={rankedItems}
      valueLabel={(count) => `${count}`}
      onAction={onOpenScope ? (item) => {
        const bucket = displayItems.find((candidate) => candidate.key === item.id);
        if (bucket?.scope_id) onOpenScope(bucket.scope_id);
      } : undefined}
    />}
  </section>;
}

function compactBuckets(items: OrganizationMetricBucket[]) {
  const areas = items.filter((item) => item.kind === "ORGANIZATION_SCOPE");
  const residual = items.filter((item) => item.kind !== "ORGANIZATION_SCOPE");
  if (areas.length <= 6) return [...areas, ...residual];

  const visible = areas.slice(0, 6);
  const otherValue = areas.slice(6).reduce((sum, item) => sum + item.value, 0);
  return [
    ...visible,
    { key: "other-areas", label: "Other areas", kind: "UNAVAILABLE" as const, value: otherValue },
    ...residual,
  ];
}

function bucketMeta(item: OrganizationMetricBucket) {
  if (item.kind === "DIRECT") return "Recorded directly at this scope";
  if (item.kind === "UNATTRIBUTED") return "No organization area recorded";
  if (item.kind === "UNAVAILABLE") return item.key === "other-areas" ? "Remaining organization areas" : "Area no longer available";
  return "Open this area";
}

function isAbortError(error: unknown) {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}
