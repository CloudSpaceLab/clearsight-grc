import { homeMetricQuality } from "../../homeMetricPresentation";
import { reportingPeriodLabel, type ReportingPeriodQuery } from "../../reportingPeriod";
import type { ReportDefinition, ReportRun } from "../../reportingTypes";
import { formatLossMoneyCompact } from "../losses/lossPresentation";
import { LossMovement } from "../oversight/LossMovement";
import { OrganizationRiskSummary } from "../oversight/OrganizationRiskSummary";
import { Button, DataTable, EmptyState, MetricCard, Notice, type DataColumn } from "../ui";
import { formatDateTime, isReportAvailable } from "./reportRunPresentation";
import { useReportsOverviewMetrics } from "./useReportsOverviewMetrics";
import "../../oversight.css";
import "./reports-overview.css";

type LoadState = "loading" | "live" | "unavailable";
type Props = {
  metricPeriod?: ReportingPeriodQuery;
  metricState: "idle" | LoadState;
  refreshKey: number;
  scopeAuthorized: boolean;
  legalEntityID?: string;
  organizationScopeID?: string;
  legalEntityName?: string;
  runs: ReportRun[];
  nextCursor?: string;
  runState: "loading" | "live" | "error";
  definitionsByID: Map<string, ReportDefinition>;
  definitionState: "loading" | "live" | "error";
  definitionCount: number;
  columns: DataColumn<ReportRun>[];
  onSelectRun: (run: ReportRun) => void;
  onViewReports: () => void;
  onOpenExplore: () => void;
  onOpenBuilder: () => void;
  onOpenWork?: () => void;
  onOpenPrograms?: () => void;
};

/** The approved Reports Overview layout uses only governed source measures. */
export function ReportsOverview({
  metricPeriod, metricState, refreshKey, scopeAuthorized, legalEntityID, organizationScopeID, legalEntityName,
  runs, nextCursor, runState, definitionsByID, definitionState, definitionCount,
  columns, onSelectRun, onViewReports, onOpenExplore, onOpenBuilder, onOpenWork, onOpenPrograms,
}: Props) {
  const { domain, loss, domainState, lossState } = useReportsOverviewMetrics({
    metricPeriod, metricState, refreshKey, scopeAuthorized, legalEntityID, organizationScopeID,
  });
  const risk = domain?.items.find((item) => item.id === "risks_outside_appetite");
  const riskQuality = !risk || risk.population === 0 ? "unknown" : homeMetricQuality({
    freshness: risk.freshness,
    completeness: risk.completeness === "COMPLETE" && ((risk.unknown ?? 0) > 0 || (risk.excluded ?? 0) > 0)
      ? "PARTIAL" : risk.completeness,
  });
  const riskValue = risk && risk.population > 0 ? risk.value : "—";
  const readyCount = runState === "live" ? runs.filter(isReportAvailable).length : undefined;
  const net = loss && !loss.mixed_currencies && loss.net_loss
    ? formatLossMoneyCompact(loss.net_loss.minor_units, loss.net_loss.currency) : null;
  const recentRuns = [...runs].sort((a, b) => Date.parse(b.created_at) - Date.parse(a.created_at)).slice(0, 5);

  return <div className="reports-v2-overview" role="region" aria-label="Reports Overview">
    {runState === "error" && <Notice tone="warning">Report totals unavailable.</Notice>}
    <div className="reports-v2-columns">
      <div className="reports-v2-primary">
        <section className="reports-v2-stats" aria-label="Reporting measures">
          <MetricCard label="Outside appetite" value={riskValue} quality={riskQuality}
            qualityLabel={domainState === "loading" ? "Loading" : !risk ? "Unavailable" : risk.population === 0 ? "Unassessed" : undefined}
            detail={risk ? `${risk.population} in scope` : "Current risk posture"}
            tone={risk && risk.value > 0 ? "error" : "neutral"} onPress={onOpenExplore}/>
          <MetricCard label="Requirements assessed" value="—" quality="unknown" qualityLabel="Unavailable"
            detail="Assessed / applicable" onPress={onOpenPrograms}/>
          <MetricCard label="Net operational loss" value={net ?? "—"} quality={net ? "current" : "unknown"}
            qualityLabel={lossState === "loading" ? "Loading" : net ? "Period flow" : "Unavailable"}
            detail={loss?.mixed_currencies ? "Separate currencies" : metricPeriod ? reportingPeriodLabel(metricPeriod) : "Period unavailable"}
            onPress={onOpenExplore}/>
          <MetricCard label="Reports ready" value={readyCount ?? "—"}
            quality={readyCount === undefined ? "unknown" : nextCursor ? "partial" : "current"}
            qualityLabel={runState === "loading" ? "Loading" : runState === "error" ? "Unavailable" : nextCursor ? "Partial list" : "Available"}
            detail={nextCursor ? "More reports available" : "Loaded report runs"} onPress={onViewReports}/>
        </section>

        <div className="reports-v2-analysis reports-v2-analysis--primary">
          <section className="reports-v2-panel" aria-label="Risk concentration panel">
            <OrganizationRiskSummary bundle={domain} organizationScopeID={organizationScopeID} embedded
              action={<Button variant="quiet" onPress={onOpenExplore}>Explore risks</Button>}/>
            {domainState === "loading" && <p role="status">Loading risk concentration…</p>}
            {(domainState === "unavailable" || (domainState === "live" && !risk)) && <Notice tone="warning">Risk concentration unavailable.</Notice>}
          </section>
          <section className="reports-v2-panel reports-v2-coverage" aria-labelledby="reports-v2-coverage-heading">
            <div className="reports-v2-panel-heading"><h2 id="reports-v2-coverage-heading">Assessment coverage</h2>
              <Button variant="quiet" onPress={onOpenPrograms} isDisabled={!onOpenPrograms}>View Programs</Button></div>
            <p>Assessed requirements / applicable requirements</p>
            <div className="reports-v2-empty-measure">—</div>
            <Notice tone="warning">Assessment totals unavailable for this scope.</Notice>
          </section>
        </div>
        <div className="reports-v2-analysis reports-v2-analysis--secondary">
          <section className="reports-v2-panel" aria-label="Loss movement panel">
            <LossMovement bundle={loss} state={lossState} embedded
              action={<Button variant="quiet" onPress={onOpenExplore}>Explore losses</Button>}/>
          </section>
          <section className="reports-v2-panel reports-v2-findings" aria-labelledby="reports-v2-findings-heading">
            <div className="reports-v2-panel-heading"><h2 id="reports-v2-findings-heading">Priority findings</h2>
              {onOpenWork && <Button variant="quiet" onPress={onOpenWork}>Review all</Button>}</div>
            <p>Assigned review and remediation</p>
            <Notice tone="info">Open Work to review assigned findings.</Notice>
          </section>
        </div>
        <section className="reports-v2-panel reports-overview__recent" aria-labelledby="reports-recent-heading">
          <header>
            <div><h2 id="reports-recent-heading">Recent reports</h2>
              <p>{runState === "live" ? recentRuns.length : "—"} recent {recentRuns.length === 1 ? "report" : "reports"}{nextCursor ? " · More available" : ""}</p></div>
            <Button variant="quiet" onPress={onViewReports}>View all</Button>
          </header>
          {runState === "loading" && <p role="status">Loading reports…</p>}
          {runState === "error" && <Notice tone="warning">Recent reports are unavailable.</Notice>}
          {runState === "live" && recentRuns.length === 0 && <EmptyState
            population="generated reports" title="No reports yet" description="Generate a report to get started."/>}
          {runState === "live" && recentRuns.length > 0 && <DataTable ariaLabel="Recent reports" rows={recentRuns}
            rowKey={(run) => run.id}
            rowName={(run) => definitionsByID.get(run.definition_id)?.name || run.definition_code}
            columns={columns} onRowAction={onSelectRun} rowActionLabel="View"/>}
        </section>
      </div>
      <aside className="reports-v2-rail" aria-label="Reporting context">
        <section className="reports-v2-panel reports-v2-context">
          <h2>Reporting context</h2>
          <p>{organizationScopeID ? "Analysis for the selected area" : "Analysis for the legal entity"}</p>
          <dl>
            <div><dt>Risk as of</dt><dd>{domain ? formatDateTime(domain.posture_as_of) : domainState === "loading" ? "Loading…" : "Unavailable"}</dd></div>
            <div><dt>Loss period</dt><dd>{metricPeriod ? `${metricPeriod.start_date} – ${metricPeriod.end_date}` : "Unavailable"}</dd></div>
            <div><dt>Report scope</dt><dd>{legalEntityName || "Legal entity"}</dd></div>
          </dl>
          {!scopeAuthorized && <Notice tone="warning">Organization scope is not authorized.</Notice>}
          <Button variant="secondary" onPress={onOpenExplore}>Explore analysis</Button>
        </section>
        <section className="reports-v2-panel reports-v2-saved">
          <h2>Saved report setups</h2>
          <p>{definitionState === "live" ? `${definitionCount} loaded setups` : "Setups unavailable"}</p>
          <Button variant="secondary" onPress={onOpenBuilder}>Build report</Button>
        </section>
      </aside>
    </div>
  </div>;
}
