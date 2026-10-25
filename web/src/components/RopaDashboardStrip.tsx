import { Button, EmptyState, MetricCard, Notice, RankedBarList, StatusBadge, Surface, type MetricCardQuality, type StatusTone } from "./ui";
import type { ProcessingActivityStatus, RegisterSummary } from "../ropaTypes";
import "./ropa.css";
import "./ropa/ropa-overview.css";

export type RopaDashboardStripProps = {
  summary: RegisterSummary;
  legalEntityName?: string;
  onRetry?: () => void;
  onOpenStatus?: (status: ProcessingActivityStatus | undefined) => void;
};

export function RopaDashboardStrip({ summary, legalEntityName, onRetry, onOpenStatus }: RopaDashboardStripProps) {
  const stale = summary.freshness !== "CURRENT";
  const counts = summary.counts;
  const noPopulation = counts.total === 0;
  const quality: MetricCardQuality = noPopulation || summary.coverage.population === undefined ||
    summary.coverage.excluded == null || summary.coverage.unknown == null ? "unknown" :
    stale ? "stale" : summary.coverage.excluded > 0 || summary.coverage.unknown > 0 ? "partial" : "current";
  const metrics: ReadonlyArray<{ label: string; value: number; tone: StatusTone; status?: ProcessingActivityStatus; canOpen?: boolean }> = [
    { label: "Processing activities", value: counts.total, tone: "neutral", canOpen: true },
    { label: "In progress", value: counts.open, tone: "info", status: "OPEN", canOpen: true },
    { label: "Reviews overdue", value: counts.review_overdue, tone: counts.review_overdue > 0 ? "warning" : "success" },
    { label: "Missing lawful basis", value: counts.missing_lawful_basis, tone: counts.missing_lawful_basis > 0 ? "warning" : "success" },
  ];
  const gaps = [
    { label: "Missing owner", value: counts.missing_owner },
    { label: "Missing data subjects", value: counts.no_data_subjects },
  ];
  const statuses = [
    { id: "NEW", label: "Not started", value: counts.new },
    { id: "OPEN", label: "In progress", value: counts.open },
    { id: "CLOSED", label: "Complete", value: counts.closed },
  ];

  return <section className="ropa-dashboard-strip ropa-overview" role="region" aria-label="Processing activity status and coverage">
    {stale && <Notice tone="warning"><strong>Summary may be outdated.</strong> {onRetry && <Button variant="secondary" size="compact" onPress={onRetry}>Retry</Button>}</Notice>}
    <div className="ropa-dashboard-strip__metrics" aria-label="Stored processing activity counts">
      {metrics.map((metric) => <MetricCard key={metric.label}
        label={metric.label}
        value={noPopulation && metric.label !== "Processing activities" ? "—" : formatCount(metric.value)}
        tone={metric.tone}
        quality={quality}
        qualityLabel={noPopulation ? "No assessed population" : undefined}
        onPress={metric.canOpen && onOpenStatus ? () => onOpenStatus(metric.status) : undefined}
        actionLabel={metric.canOpen && onOpenStatus ? "Open ROPA" : undefined}
      />)}
    </div>
    <div className="ropa-overview__layout">
      <div className="ropa-overview__primary">
        <section className="ropa-overview__panel" aria-labelledby="ropa-status-heading"><Surface>
          <header><h2 id="ropa-status-heading">Activity status</h2><p>{legalEntityName || "Processing activities"} · All recorded activities</p></header>
          {noPopulation ? <EmptyState population={legalEntityName || "Current register"} title="No activities" description="No processing activities have been recorded for this legal entity."/> : <RankedBarList
            ariaLabel="Processing activity status counts" layout="inline" items={statuses}
            onAction={onOpenStatus ? (item) => onOpenStatus(item.id as ProcessingActivityStatus) : undefined}
          />}
          {onOpenStatus && <Button variant="quiet" size="compact" onPress={() => onOpenStatus(undefined)}>All activities</Button>}
        </Surface></section>
        <section className="ropa-overview__panel" aria-labelledby="ropa-completeness-heading"><Surface>
          <header><h2 id="ropa-completeness-heading">Record completeness</h2><p>Missing processing activity details</p></header>
          <div className="ropa-overview__gaps">{gaps.map((gap) => <MetricCard key={gap.label} label={gap.label}
            value={noPopulation ? "—" : formatCount(gap.value)} tone={gap.value > 0 ? "warning" : "success"}
            quality={quality} qualityLabel={noPopulation ? "No assessed population" : undefined}
          />)}</div>
        </Surface></section>
      </div>
      <aside className="ropa-overview__coverage" aria-labelledby="ropa-coverage-heading"><Surface>
        <h2 id="ropa-coverage-heading">Register coverage</h2>
        <StatusBadge tone={stale ? "warning" : "neutral"}>{stale ? "Stale" : "Current"}</StatusBadge>
        <p>{coverageSummary(summary.coverage)}</p>
        <p>Includes {formatCount(counts.retired)} ended {counts.retired === 1 ? "activity" : "activities"}.</p>
        <dl>
          <div><dt>Updated</dt><dd><time dateTime={summary.generated_at}>{formatDateTime(summary.generated_at)}</time></dd></div>
          <div><dt>Data through</dt><dd><time dateTime={summary.source_high_water}>{formatDateTime(summary.source_high_water)}</time></dd></div>
        </dl>
      </Surface></aside>
    </div>
  </section>;
}

function coverageSummary(coverage: RegisterSummary["coverage"]): string {
  const population = formatPopulation(coverage.population);
  const excluded = formatCoverage(coverage.excluded);
  const unknown = formatCoverage(coverage.unknown);
  const excludedText = excluded === "unknown" ? "excluded: unknown" : `${excluded} excluded`;
  const unknownText = unknown === "unknown" ? "unknown records: unknown" : `${unknown} unknown`;
  return `${population} checked · ${excludedText} · ${unknownText}`;
}

function formatPopulation(value: number | undefined): string {
  return value === undefined || !Number.isFinite(value) ? "unknown" : value.toLocaleString("en-GB");
}

function formatCoverage(value: number | null | undefined): string {
  return value === null || value === undefined || !Number.isFinite(value) ? "unknown" : String(value);
}

function formatCount(value: number): string {
  return Number.isFinite(value) ? value.toLocaleString("en-GB") : "unknown";
}

function formatDateTime(value: string): string {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return "Unknown";
  return new Intl.DateTimeFormat("en-GB", { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit", timeZone: "UTC" }).format(date);
}
