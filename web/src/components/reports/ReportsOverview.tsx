import { useEffect, useState } from "react";
import {
  loadDomainMetrics,
  loadLossPeriodMetrics,
  type DomainMetricBundle,
  type HomeMetricBundle,
  type LossPeriodBundle,
} from "../../metricApi";
import type { ReportingPeriodQuery } from "../../reportingPeriod";
import type { ReportDefinition, ReportRun } from "../../reportingTypes";
import { formatLossMoneyExact } from "../losses/lossPresentation";
import { LossMovement } from "../oversight/LossMovement";
import { OrganizationRiskSummary } from "../oversight/OrganizationRiskSummary";
import { Button, DataTable, EmptyState, MetricCard, Notice, type DataColumn } from "../ui";

type LoadState = "loading" | "live" | "unavailable";
type Props = {
  metricBundle?: HomeMetricBundle;
  metricPeriod?: ReportingPeriodQuery;
  metricState: "idle" | LoadState;
  scopeAuthorized: boolean;
  legalEntityID?: string;
  organizationScopeID?: string;
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
  onOpenRisk?: (id: string) => void;
  onOpenLoss?: (id: string) => void;
};

/** The approved Reports Overview layout uses only governed source measures. */
export function ReportsOverview({
  metricBundle, metricPeriod, metricState, scopeAuthorized, legalEntityID, organizationScopeID,
  runs, nextCursor, runState, definitionsByID, definitionState, definitionCount,
  columns, onSelectRun, onViewReports, onOpenExplore, onOpenBuilder, onOpenWork, onOpenPrograms,
}: Props) {
  const [domain, setDomain] = useState<DomainMetricBundle | null>(null);
  const [loss, setLoss] = useState<LossPeriodBundle | null>(null);
  const [domainState, setDomainState] = useState<LoadState>("loading");
  const [lossState, setLossState] = useState<LoadState>("loading");

  useEffect(() => {
    // Invalidate both panels with scope/period changes so stale source quality
    // never follows the next authorized selection.
    setDomain(null);
    setLoss(null);
    setDomainState(metricState === "loading" ? "loading" : "unavailable");
    setLossState(metricState === "loading" ? "loading" : "unavailable");
    if (!scopeAuthorized || metricState !== "live" || !metricPeriod) return;
    const controller = new AbortController();
    setDomainState("loading");
    setLossState("loading");
    void loadDomainMetrics(undefined, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      if (value.scope_kind !== "LEGAL_ENTITY" || (legalEntityID && value.scope_id !== legalEntityID)) {
        setDomainState("unavailable");
        return;
      }
      setDomain(value);
      setDomainState("live");
    }).catch(() => { if (!controller.signal.aborted) setDomainState("unavailable"); });

    void loadLossPeriodMetrics(metricPeriod, organizationScopeID, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      const expectedScope = organizationScopeID ?? legalEntityID;
      if (value.period_start.slice(0, 10) !== metricPeriod.start_date ||
          value.period_end.slice(0, 10) !== metricPeriod.end_date ||
          value.scope_kind !== (organizationScopeID ? "ORGANIZATION_SCOPE" : "LEGAL_ENTITY") ||
          (expectedScope && value.scope_id !== expectedScope)) {
        setLossState("unavailable");
        return;
      }
      setLoss(value);
      setLossState("live");
    }).catch(() => { if (!controller.signal.aborted) setLossState("unavailable"); });
    return () => controller.abort();
  }, [scopeAuthorized, legalEntityID, organizationScopeID, metricState, metricPeriod?.start_date, metricPeriod?.end_date]);

  const risk = metricBundle?.items?.find((item) => item.id === "risks_outside_appetite");
  const riskQuality = !risk ? "unknown"
    : risk.freshness === "STALE" ? "stale"
    : risk.completeness !== "COMPLETE" || (risk.unknown ?? 0) > 0 || (risk.excluded ?? 0) > 0
      ? "partial" : "current";
  const riskValue = scopeAuthorized && metricState === "live" && risk ? risk.value : "—";
  const readyCount = runState === "live" ? runs.filter((item) => item.status === "READY").length : undefined;
  const net = lossState === "live" && loss && !loss.mixed_currencies && loss.net_loss
    ? formatLossMoneyExact(loss.net_loss.minor_units, loss.net_loss.currency) : null;
  const recentRuns = [...runs].sort((a, b) => Date.parse(b.created_at) - Date.parse(a.created_at)).slice(0, 5);

  return <div className="reports-v2-overview" role="region" aria-label="Reports Overview">
    {runState === "error" && <Notice tone="warning">Report status totals unavailable until report runs load.</Notice>}
    <div className="reports-v2-columns">
      <div className="reports-v2-primary">
        <section className="reports-v2-stats" aria-label="Reporting measures">
          <MetricCard label="Outside appetite" value={riskValue} quality={riskValue === "—" ? "unknown" : riskQuality}
            detail="Current risk posture" meta={risk ? `Population: ${risk.population}` : "Risk measure unavailable"}
            tone="error" onPress={onOpenExplore} actionLabel="Explore risks"/>
          <MetricCard label="Requirements assessed" value="—" quality="unknown"
            detail="Assessed / applicable" meta="Framework assessment totals unavailable"
            onPress={onOpenPrograms} actionLabel={onOpenPrograms ? "View Programs" : undefined}/>
          <MetricCard label="Net operational loss" value={net ?? "—"}
            quality={net ? "current" : "unknown"}
            detail={metricPeriod ? `${metricPeriod.start_date} – ${metricPeriod.end_date}` : "Period unavailable"}
            meta={loss?.mixed_currencies ? "Currencies reported separately" : lossState === "unavailable" ? "Loss data unavailable" : undefined}
            onPress={onOpenExplore} actionLabel="Explore losses"/>
          <MetricCard label="Reports ready" value={readyCount ?? "—"}
            quality={readyCount === undefined ? "unknown" : nextCursor ? "partial" : "current"}
            detail={nextCursor ? "Loaded reports · more available" : "Loaded report runs"}
            onPress={onViewReports} actionLabel="View reports"/>
        </section>

        <div className="reports-v2-analysis">
          <section className="reports-v2-panel" aria-label="Risk concentration panel">
            <OrganizationRiskSummary bundle={domain} organizationScopeID={organizationScopeID}/>
            {domainState === "unavailable" && <Notice tone="warning">Risk source unavailable.</Notice>}
            <Button variant="quiet" onPress={onOpenExplore}>Explore risks</Button>
          </section>
          <section className="reports-v2-panel reports-v2-coverage" aria-labelledby="reports-v2-coverage-heading">
            <div className="reports-v2-panel-heading"><h2 id="reports-v2-coverage-heading">Assessment coverage</h2>
              <Button variant="quiet" onPress={onOpenPrograms} isDisabled={!onOpenPrograms}>View Programs</Button></div>
            <p>Assessed requirements / applicable requirements</p>
            <div className="reports-v2-empty-measure">—</div>
            <Notice tone="warning">Framework-level assessment denominators are unavailable for this scope.</Notice>
            <p>Coverage is not a compliance conclusion.</p>
          </section>
        </div>
        <div className="reports-v2-analysis">
          <section className="reports-v2-panel" aria-label="Loss movement panel">
            <LossMovement bundle={loss} state={lossState === "loading" ? "loading" : lossState}/>
            <Button variant="quiet" onPress={onOpenExplore}>Explore losses</Button>
          </section>
          <section className="reports-v2-panel reports-v2-findings" aria-labelledby="reports-v2-findings-heading">
            <div className="reports-v2-panel-heading"><h2 id="reports-v2-findings-heading">Priority findings</h2>
              {onOpenWork && <Button variant="quiet" onPress={onOpenWork}>Review all</Button>}</div>
            <p>Assigned review and remediation</p>
            <Notice tone="info">Current findings are available in Work. This report has no independent reviewed finding population.</Notice>
          </section>
        </div>
        <section className="reports-v2-panel reports-overview__recent" aria-labelledby="reports-recent-heading">
          <header>
            <div><h2 id="reports-recent-heading">Recent reports</h2>
              <p>{recentRuns.length} recent loaded {recentRuns.length === 1 ? "report" : "reports"}{nextCursor ? " · More available" : ""}</p></div>
            <Button variant="quiet" onPress={onViewReports}>View all</Button>
          </header>
          {runState === "loading" && <p role="status">Loading reports…</p>}
          {runState === "error" && <Notice tone="warning">Recent reports are unavailable.</Notice>}
          {runState === "live" && recentRuns.length === 0 && <EmptyState
            population="generated reports" title="No reports yet" description="Generate a report to get started."/>}
          {recentRuns.length > 0 && <DataTable ariaLabel="Recent reports" rows={recentRuns}
            rowKey={(run) => run.id}
            rowName={(run) => definitionsByID.get(run.definition_id)?.name || run.definition_code}
            columns={columns} onRowAction={onSelectRun} rowActionLabel="View" isLoading={runState === "loading"}/>}
        </section>
      </div>
      <aside className="reports-v2-rail" aria-label="Reporting context">
        <section className="reports-v2-panel reports-v2-context">
          <h2>Reporting context</h2>
          <p>Risk figures show current posture. Loss figures use the selected reporting period.</p>
          <dl>
            <div><dt>Risk source</dt><dd>{domainState === "live" ? "Current source" : "Unavailable"}</dd></div>
            <div><dt>Loss period</dt><dd>{metricPeriod ? `${metricPeriod.start_date} – ${metricPeriod.end_date}` : "Unavailable"}</dd></div>
            <div><dt>Organization</dt><dd>{organizationScopeID ? "Selected area" : "Legal entity"}</dd></div>
          </dl>
          {!scopeAuthorized && <Notice tone="warning">Organization scope is not authorized.</Notice>}
          <Button variant="secondary" onPress={onOpenExplore}>Explore analysis</Button>
        </section>
        <section className="reports-v2-panel reports-v2-saved">
          <h2>Saved report setups</h2>
          <p>{definitionState === "live" ? `${definitionCount} available loaded setups` : "Setups unavailable"}</p>
          <Button variant="secondary" onPress={onOpenBuilder}>Build report</Button>
        </section>
      </aside>
    </div>
  </div>;
}
