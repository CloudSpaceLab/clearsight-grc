import { useEffect, useMemo, useState } from "react";
import { apiErrorKind } from "../../http";
import {
  loadDomainMetricOrganizationTrend,
  loadDomainMetricTrend,
  type DomainMetricBundle,
  type MetricTrendSeries,
  type OrganizationTrendSeries,
} from "../../metricApi";
import { Button, MetricTrend, type MetricTrendDatum } from "../ui";

type RangeDays = 7 | 30 | 90;
type LoadState = "loading" | "live" | "missing" | "unavailable";

type Props = {
  bundle: DomainMetricBundle | null;
  organizationScopeID?: string;
  loadLegalEntityTrend?: typeof loadDomainMetricTrend;
  loadOrganizationTrend?: typeof loadDomainMetricOrganizationTrend;
};

const ranges: readonly RangeDays[] = [7, 30, 90];

export function RiskMovement({
  bundle,
  organizationScopeID,
  loadLegalEntityTrend = loadDomainMetricTrend,
  loadOrganizationTrend = loadDomainMetricOrganizationTrend,
}: Props) {
  const [range, setRange] = useState<RangeDays>(30);
  const [state, setState] = useState<LoadState>("loading");
  const [legalTrend, setLegalTrend] = useState<MetricTrendSeries | null>(null);
  const [organizationTrend, setOrganizationTrend] = useState<OrganizationTrendSeries | null>(null);
  const current = bundle?.items.find((item) => item.id === "risks_outside_appetite");

  useEffect(() => {
    if (!bundle || !current) {
      setState("loading");
      setLegalTrend(null);
      setOrganizationTrend(null);
      return;
    }

    const controller = new AbortController();
    const today = utcDay(new Date());
    const baselineDate = addUTCDays(today, -range);
    const requestedStart = organizationScopeID || range !== 7 ? baselineDate : addUTCDays(baselineDate, -1);
    const startDate = formatDate(requestedStart);
    const endDate = formatDate(today);

    setState("loading");
    setLegalTrend(null);
    setOrganizationTrend(null);

    const request = organizationScopeID
      ? loadOrganizationTrend(current.id, organizationScopeID, startDate, endDate, controller.signal)
        .then((value) => ({ kind: "organization" as const, value }))
      : loadLegalEntityTrend(current.id, startDate, endDate, controller.signal)
        .then((value) => ({ kind: "legal" as const, value }));

    void request.then((result) => {
      if (controller.signal.aborted) return;
      if (result.kind === "organization") {
        setOrganizationTrend(result.value);
        setLegalTrend(null);
      } else {
        setLegalTrend(result.value);
        setOrganizationTrend(null);
      }
      setState("live");
    }).catch((error: unknown) => {
      if (controller.signal.aborted || isAbortError(error)) return;
      setLegalTrend(null);
      setOrganizationTrend(null);
      setState(apiErrorKind(error) === "not_found" ? "missing" : "unavailable");
    });

    return () => controller.abort();
  }, [bundle?.source_id, current, loadLegalEntityTrend, loadOrganizationTrend, organizationScopeID, range]);

  const model = useMemo(() => {
    if (!bundle || !current) return null;

    const today = utcDay(new Date());
    const baselineDate = addUTCDays(today, -range);
    const baselineKey = formatDate(baselineDate);
    const todayKey = formatDate(today);

    const history = organizationTrend
      ? organizationTrend.points.map((point) => ({
          id: `history-${point.date}`,
          at: point.at,
          date: point.date,
          value: point.value,
          complete: point.source_complete,
        }))
      : (legalTrend?.points ?? []).map((point) => ({
          id: `history-${point.at}`,
          at: point.at,
          date: point.at.slice(0, 10),
          value: point.value,
          complete: metricPointComplete(point),
        }));

    const relevant = history.filter((point) => point.date >= baselineKey && point.date < todayKey);
    const baseline = relevant.find((point) => point.date === baselineKey);
    const currentComplete = current.freshness === "CURRENT"
      && current.completeness === "COMPLETE"
      && (current.excluded ?? 0) === 0
      && (current.unknown ?? 0) === 0;
    const comparable = Boolean(baseline?.complete && currentComplete);

    const chartPoints: MetricTrendDatum[] = relevant.map((point) => ({
      id: point.id,
      at: point.at,
      label: shortDate(point.at),
      value: point.value,
    }));
    chartPoints.push({
      id: `current-${bundle.source_id}`,
      at: bundle.generated_at,
      label: "Now",
      value: current.value,
    });

    return {
      chartPoints,
      comparison: comparisonLabel(current.value, baseline?.value, range, comparable),
      baselineValue: comparable ? baseline?.value : undefined,
      currentValue: current.value,
    };
  }, [bundle, current, legalTrend, organizationTrend, range]);

  return <section className="risk-movement" aria-labelledby="risk-movement-heading">
    <div className="section-header risk-movement__header">
      <div>
        <span className="eyebrow">What changed</span>
        <h2 id="risk-movement-heading">Risk movement</h2>
        <p>Outside-appetite risk posture over time.</p>
      </div>
      <div className="risk-movement__ranges" aria-label="Risk movement period">
        {ranges.map((days) => <Button
          key={days}
          size="compact"
          variant={range === days ? "primary" : "quiet"}
          aria-pressed={range === days}
          onPress={() => setRange(days)}
        >{days}d</Button>)}
      </div>
    </div>

    {model && <div className="risk-movement__summary" aria-live="polite">
      <strong>{model.currentValue} now</strong>
      <span>{model.comparison}</span>
    </div>}

    {state === "loading" && <p className="oversight-today-status" role="status" aria-busy="true">Loading risk movement…</p>}
    {(state === "missing" || state === "unavailable") && <p className="risk-movement__empty">No comparable history yet.</p>}
    {state === "live" && model && <MetricTrend
      ariaLabel={`Outside-appetite risk movement for the last ${range} days`}
      points={model.chartPoints}
    />}
  </section>;
}

function comparisonLabel(current: number, baseline: number | undefined, days: RangeDays, comparable: boolean) {
  if (!comparable || baseline === undefined) return "No comparable history";
  const delta = current - baseline;
  if (delta > 0) return `+${delta} worse vs ${days} days ago`;
  if (delta < 0) return `${delta} improved vs ${days} days ago`;
  return `No change vs ${days} days ago`;
}

function metricPointComplete(point: MetricTrendSeries["points"][number]) {
  return point.freshness === "CURRENT"
    && point.completeness === "COMPLETE"
    && (point.excluded ?? 0) === 0
    && (point.unknown ?? 0) === 0;
}

function utcDay(value: Date) {
  return new Date(Date.UTC(value.getUTCFullYear(), value.getUTCMonth(), value.getUTCDate()));
}

function addUTCDays(value: Date, days: number) {
  return new Date(value.getTime() + days * 24 * 60 * 60 * 1000);
}

function formatDate(value: Date) {
  return value.toISOString().slice(0, 10);
}

function shortDate(value: string) {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value.slice(0, 10);
  return new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric", timeZone: "UTC" }).format(parsed);
}

function isAbortError(error: unknown) {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}
