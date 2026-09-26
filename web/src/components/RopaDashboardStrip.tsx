import { Button, Notice, StatusBadge, type StatusTone } from "./ui";
import type { ProcessingActivityStatus, RegisterSummary } from "../ropaTypes";
import "./ropa.css";

export type RopaDashboardStripProps = {
  summary: RegisterSummary;
  legalEntityName?: string;
  onRetry?: () => void;
  onOpenStatus?: (status: ProcessingActivityStatus | undefined) => void;
};

export function RopaDashboardStrip({ summary, onRetry, onOpenStatus }: RopaDashboardStripProps) {
  const stale = summary.freshness !== "CURRENT";
  const counts = summary.counts;

  return <section className="ropa-dashboard-strip" role="region" aria-label="Processing activity status and coverage">
    <div className="ropa-dashboard-strip__header">
      <div>
        <span className="eyebrow">Register status</span>
        <h2>Processing activity status</h2>
      </div>
      <div className="ropa-dashboard-strip__freshness">
        <StatusBadge tone={stale ? "warning" : "success"}>{stale ? "Stale" : "Current"}</StatusBadge>
        <span>Updated <time dateTime={summary.generated_at}>{formatDateTime(summary.generated_at)}</time></span>
      </div>
    </div>

    {stale && <Notice tone="warning">
      <strong>Summary may be outdated.</strong>
      {onRetry && <Button variant="secondary" size="compact" onPress={onRetry}>Retry</Button>}
    </Notice>}

    <div className="ropa-dashboard-strip__metrics" aria-label="Stored processing activity counts">
      <CountMetric label="Processing activities" value={counts.total} />
      <CountMetric label="In progress" value={counts.open} tone="info" />
      <CountMetric label="Reviews overdue" value={counts.review_overdue} tone={counts.review_overdue > 0 ? "warning" : "success"} />
      <ExceptionMetric label="Missing lawful basis" value={counts.missing_lawful_basis} />
      <ExceptionMetric label="Missing owner" value={counts.missing_owner} />
      <ExceptionMetric label="Missing data subjects" value={counts.no_data_subjects} />
    </div>

    <div className="ropa-dashboard-strip__coverage" aria-label="Register coverage">
      <p>{coverageSummary(summary.coverage)}</p>
      <p>Data through <time dateTime={summary.source_high_water}>{formatDateTime(summary.source_high_water)}</time></p>
    </div>

    {onOpenStatus && <div className="ropa-dashboard-strip__links" aria-label="Open matching register rows">
      <Button variant="quiet" size="compact" onPress={() => onOpenStatus(undefined)}>All activities</Button>
      <Button variant="quiet" size="compact" onPress={() => onOpenStatus("OPEN")}>In progress</Button>
    </div>}
  </section>;
}

function CountMetric({ label, value, tone = "neutral" }: { label: string; value: number; tone?: StatusTone }) {
  return <div className="ropa-dashboard-strip__metric">
    <span>{label}</span>
    <strong>{formatCount(value)}</strong>
    <StatusBadge tone={tone}>{label}</StatusBadge>
  </div>;
}

function ExceptionMetric({ label, value }: { label: string; value: number }) {
  return <div className="ropa-dashboard-strip__metric ropa-dashboard-strip__metric--exception">
    <span>Exception</span>
    <strong>{formatCount(value)}</strong>
    <StatusBadge tone={value > 0 ? "warning" : "success"}>{`${formatCount(value)} ${label.toLowerCase()}`}</StatusBadge>
  </div>;
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
