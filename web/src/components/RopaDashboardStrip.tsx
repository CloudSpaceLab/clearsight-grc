import { Button, MetricCard, Notice, StatusBadge, type MetricCardQuality, type StatusTone } from "./ui";
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
  const noPopulation = counts.total === 0;
  const quality: MetricCardQuality = noPopulation || summary.coverage.population === undefined ||
    summary.coverage.excluded == null || summary.coverage.unknown == null ? "unknown" :
    stale ? "stale" : summary.coverage.excluded > 0 || summary.coverage.unknown > 0 ? "partial" : "current";
  const cards: ReadonlyArray<{ label: string; value: number; tone: StatusTone }> = [
    { label: "Processing activities", value: counts.total, tone: "neutral" },
    { label: "In progress", value: counts.open, tone: "info" },
    { label: "Reviews overdue", value: counts.review_overdue, tone: counts.review_overdue > 0 ? "warning" : "success" },
    { label: "Missing lawful basis", value: counts.missing_lawful_basis, tone: counts.missing_lawful_basis > 0 ? "warning" : "success" },
    { label: "Missing owner", value: counts.missing_owner, tone: counts.missing_owner > 0 ? "warning" : "success" },
    { label: "Missing data subjects", value: counts.no_data_subjects, tone: counts.no_data_subjects > 0 ? "warning" : "success" },
  ];

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
      {cards.map((card) => <MetricCard
        key={card.label}
        label={card.label}
        value={noPopulation && card.label !== "Processing activities" ? "—" : formatCount(card.value)}
        tone={card.tone}
        quality={quality}
        qualityLabel={noPopulation ? "No assessed population" : undefined}
      />)}
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
