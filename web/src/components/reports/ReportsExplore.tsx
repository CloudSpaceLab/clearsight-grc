import { useEffect, useId, useMemo, useState } from "react";
import {
  loadDomainMetricMembers, loadDomainMetricOrganizationBreakdown, loadLossPeriodMetricMembers,
  type DomainMetricBundle, type HomeMetric, type HomeMetricMemberPage,
  type LossCurrencyFlow, type LossPeriodBundle, type OrganizationMetricBreakdown,
  type OrganizationMetricBucket,
} from "../../metricApi";
import type { ReportingPeriodQuery } from "../../reportingPeriod";
import { useRiskLossMetrics, type RiskLossMetricOptions } from "../insights/useRiskLossMetrics";
import { formatLossMoneyExact } from "../losses/lossPresentation";
import { MetricMemberDrill } from "../oversight/MetricMemberDrill";
import { bigintAbs, bigintMax, normalizedMinor, parseMoneyMinor } from "../oversight/lossOverviewPresentation";
import { Button, DataTable, EmptyState, FocusedSheet, Notice, RankedBarList, SelectField, StatusBadge, type DataColumn } from "../ui";
import "./reports-explore.css";

type LoadState = "loading" | "live" | "unavailable";
type Measure = "risk" | "net-loss" | "loss-events";
type Row = { id: string; label: string; value: number; exact: string; note: string };
type Drill = { key: string; kind: "risk" | "loss"; id: string; label: string; source: string; revision: string; count: number };

export type ReportsExploreProps = {
  period?: ReportingPeriodQuery;
  periodState?: "idle" | LoadState;
  scopeAuthorized: boolean;
  legalEntityID?: string;
  organizationScopeID?: string;
  refreshKey?: number;
  onOpenRisk?: (id: string) => void;
  onOpenLoss?: (id: string) => void;
  onOpenInsights?: () => void;
  loadDomain?: RiskLossMetricOptions["loadDomain"];
  loadLoss?: RiskLossMetricOptions["loadLoss"];
  loadBreakdown?: typeof loadDomainMetricOrganizationBreakdown;
  loadMembers?: typeof loadDomainMetricMembers;
  loadLossMembers?: typeof loadLossPeriodMetricMembers;
};

const measures = [
  { id: "risk", label: "Risks outside appetite" },
  { id: "net-loss", label: "Net operational loss" },
  { id: "loss-events", label: "Loss events" },
] as const;

export function ReportsExplore({
  period, periodState = period ? "live" : "unavailable", scopeAuthorized, legalEntityID, organizationScopeID,
  refreshKey = 0, onOpenRisk, onOpenLoss, onOpenInsights, loadDomain, loadLoss,
  loadBreakdown = loadDomainMetricOrganizationBreakdown,
  loadMembers = loadDomainMetricMembers, loadLossMembers = loadLossPeriodMetricMembers,
}: ReportsExploreProps) {
  const helpID = useId();
  const [measure, setMeasure] = useState<Measure>("risk");
  const [selectedCurrency, setSelectedCurrency] = useState<string>();
  const [retry, setRetry] = useState(0);
  const [openedDrill, setOpenedDrill] = useState<string>();
  const { domain, loss, domainState, lossState } = useRiskLossMetrics({
    metricPeriod: period, metricState: periodState, scopeAuthorized, legalEntityID, organizationScopeID,
    refreshKey: refreshKey + retry, loadDomain, loadLoss,
  });
  const candidate = domain?.items.find((item) => item.id === "risks_outside_appetite");
  const metric = validRisk(candidate, domain) ? candidate : undefined;
  const request = useMemo(() => measure === "risk" && domain && metric && metric.population > 0
    ? { domain, metric, organizationScopeID, loadBreakdown } : undefined,
  [measure, domain, metric, organizationScopeID, loadBreakdown]);
  const [breakdown, setBreakdown] = useState<{ request: typeof request; state: LoadState; value?: OrganizationMetricBreakdown }>();

  useEffect(() => {
    if (!request) return;
    const controller = new AbortController();
    setBreakdown({ request, state: "loading" });
    void request.loadBreakdown(request.metric.id, request.domain.source_id, request.metric.definition_revision,
      request.organizationScopeID, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      const valid = value?.source_id === request.domain.source_id && value.metric_id === request.metric.id
        && value.definition_revision === request.metric.definition_revision && value.count === request.metric.value
        && (value.scope_id ?? "") === (request.organizationScopeID ?? "") && validRiskBuckets(value);
      setBreakdown({ request, state: valid ? "live" : "unavailable", value: valid ? value : undefined });
    }).catch(() => {
      if (!controller.signal.aborted) setBreakdown({ request, state: "unavailable" });
    });
    return () => controller.abort();
  }, [request]);

  const currentBreakdown = request && breakdown?.request === request ? breakdown : undefined;
  const riskState = domainState !== "live" ? domainState : !metric ? "unavailable"
    : metric.population === 0 ? "live" : currentBreakdown?.state ?? "loading";
  const checkedLoss = loss && validLossBreakdown(loss) ? loss : undefined;
  const currencies = checkedLoss?.currencies.map((item) => item.currency) ?? [];
  const currency = selectedCurrency && currencies.includes(selectedCurrency) ? selectedCurrency : currencies[0];
  const label = measures.find((item) => item.id === measure)!.label;
  const state = measure === "risk" ? riskState : lossState === "live" && !checkedLoss ? "unavailable" : lossState;
  const rows = measure === "risk" ? riskRows(currentBreakdown?.value)
    : lossRows(checkedLoss, measure, currency);
  const total = measure === "risk" ? metric && metric.population > 0 ? String(metric.value) : "Unassessed"
    : measure === "loss-events" ? String(checkedLoss?.event_count ?? "—")
      : currency ? formatLossMoneyExact(checkedLoss!.currencies.find((item) => item.currency === currency)!.net.minor_units, currency) : "—";
  const drill = makeDrill(measure, label, metric, domain, checkedLoss, organizationScopeID);
  const drillKey = state === "live" ? drill?.key : undefined;
  useEffect(() => {
    if (openedDrill && openedDrill !== drillKey) setOpenedDrill(undefined);
  }, [openedDrill, drillKey]);

  const columns: readonly DataColumn<Row>[] = [
    { id: "area", header: "Organization area", mobileLayout: "full-width", render: (row) => <strong>{row.label}</strong>, accessibleText: (row) => row.label },
    { id: "value", header: label, kind: "number", render: (row) => row.exact, accessibleText: (row) => row.exact },
    { id: "context", header: "Context", render: (row) => row.note, accessibleText: (row) => row.note },
  ];
  const periodPending = periodState === "loading" || periodState === "idle";
  const empty = measure === "risk" ? metric?.population === 0 || metric?.value === 0
    : measure === "loss-events" ? checkedLoss?.event_count === 0 : currencies.length === 0;

  return <section className="reports-explore" aria-label="Explore measures">
    <header className="reports-explore__heading">
      <div><p className="reports-explore__eyebrow">Analysis</p><h2>Explore measures</h2>
        <p>Review risk assessments and loss activity by organization area.</p></div>
      {onOpenInsights && <Button variant="secondary" onPress={onOpenInsights}
        isDisabled={!scopeAuthorized || periodState !== "live"}>Open Insights workspace</Button>}
    </header>
    <div className="reports-explore__layout">
      <section className="reports-explore__filters" aria-label="Analysis data">
        <h3>Data</h3>
        <div className="reports-explore__fields">
          <SelectField label="Measure" placeholder="Choose a measure" value={measure} options={measures} allowsEmpty={false}
            onChange={(value) => { if (value) setMeasure(value); }}/>
          <div className="reports-explore__fixed-field" role="group" aria-label="Breakdown" aria-describedby={`${helpID}-breakdown`}>
            <SelectField label="Breakdown" placeholder="Organization area" value="organization" allowsEmpty={false}
              options={[{ id: "organization", label: "Organization area" }]} isDisabled onChange={() => undefined}/>
            <p id={`${helpID}-breakdown`}>Measures are available by organization area.</p>
          </div>
          <div className="reports-explore__fixed-field" role="group" aria-label="Comparison" aria-describedby={`${helpID}-comparison`}>
            <SelectField label="Comparison" placeholder="No comparison" value="none" allowsEmpty={false}
              options={[{ id: "none", label: "No comparison" }]} isDisabled onChange={() => undefined}/>
            <p id={`${helpID}-comparison`}>Previous-period values by organization area are unavailable.</p>
          </div>
          {measure === "net-loss" && currencies.length > 1 && <SelectField label="Currency" placeholder="Choose a currency"
            value={currency} options={currencies.map((value) => ({ id: value, label: value }))} allowsEmpty={false}
            onChange={setSelectedCurrency} description="Amounts in different currencies stay separate."/>}
        </div>
        <p className="reports-explore__scope">{organizationScopeID ? "Selected organization area" : "Selected legal entity"}.
          {measure === "risk" ? " Risk posture reflects the retained assessment source; the reporting period does not change it."
            : period ? ` Loss activity from ${period.start_date} to ${period.end_date}.` : " A verified reporting period is required."}</p>
      </section>
      <div className="reports-explore__results">
        <section className="reports-explore__chart" aria-label={`${label} chart`} aria-busy={state === "loading" || undefined}
          data-state={!scopeAuthorized ? "denied" : periodPending ? "loading" : state}
          data-quality={measure === "risk" ? metric?.completeness.toLowerCase() : undefined}>
          <header><div><h3>{label}</h3><p>By organization area</p></div>
            {state === "live" && measure === "risk" && metric && <StatusBadge tone={metric.freshness === "STALE" ? "warning" : "neutral"}>
              {metric.freshness === "STALE" ? "Source stale" : "Source current"}</StatusBadge>}</header>
          {!scopeAuthorized ? <Notice tone="warning">This organization scope is not authorized. Select an authorized area to review its measures.</Notice>
            : periodPending || state === "loading" ? <p role="status">Loading {label.toLowerCase()}…</p>
              : state === "unavailable" ? <Notice tone="warning">The {label.toLowerCase()} measure is unavailable for this scope and reporting period.
                {periodState === "live" && <Button variant="secondary" onPress={() => setRetry((value) => value + 1)}>Try again</Button>}</Notice>
                : <>
                  {measure === "risk" && metric && <p className="reports-explore__coverage">
                    {metric.population} risks in scope · {metric.unknown === undefined ? "Unknown count unavailable" : `${metric.unknown} unknown`} · {metric.excluded === undefined ? "Excluded count unavailable" : `${metric.excluded} excluded`}.
                    {metric.completeness !== "COMPLETE" && ` ${metric.completeness === "PARTIAL" ? "Risk coverage is partial." : "Risk coverage is unknown."}`}
                  </p>}
                  {measure === "net-loss" && checkedLoss?.mixed_currencies && <p className="reports-explore__coverage">Showing {currency ?? "separate currency amounts"} only. Amounts are not converted or combined.</p>}
                  {measure === "net-loss" && rows.some((row) => row.note.includes("Net recovery")) && <p className="reports-explore__coverage">Bars show the size of each amount. Negative amounts represent net recoveries.</p>}
                  {empty ? <ExploreEmpty measure={measure} metric={metric}/>
                    : <RankedBarList ariaLabel={`${label} by organization area`} layout="inline"
                      items={rows.map((row) => ({ id: row.id, label: row.label, value: row.value, displayValue: row.exact, meta: row.note }))}/>}
                  <footer className="reports-explore__total"><p>Total <strong>{total}</strong></p>
                    {drill && drill.count > 0 && <Button variant="secondary" onPress={() => setOpenedDrill(drill.key)}>Review underlying records</Button>}</footer>
                  {measure === "risk" && domain && metric
                    ? <p className="reports-explore__source">Risk as of {sourceDate(domain.posture_as_of)} · Source revision {domain.source_revision} · Definition {metric.definition_revision}</p>
                    : checkedLoss && <p className="reports-explore__source">Prepared {sourceDate(checkedLoss.generated_at)} · Source revision {checkedLoss.source_revision} · Definition {checkedLoss.definition_revision}</p>}
                </>}
        </section>
        {state === "live" && scopeAuthorized && !periodPending && <section className="reports-explore__table" aria-label="Breakdown data">
          <header><h3>Breakdown</h3><p>{label}{measure === "net-loss" && currency ? ` · ${currency}` : ""}</p></header>
          {rows.length > 0 && !(measure === "risk" && metric?.population === 0)
            ? <DataTable ariaLabel="Explore measure breakdown" rows={rows} rowKey={(row) => row.id} rowName={(row) => `${row.label}, ${row.exact}`} columns={columns}/>
            : <p>No organization-area values are available for this measure.</p>}
        </section>}
      </div>
    </div>
    {drill && openedDrill !== undefined && openedDrill === drillKey && <FocusedSheet label={`${label} records`} onClose={() => setOpenedDrill(undefined)} size="wide">
      <h2>{label} records</h2>
      {measure === "net-loss" && checkedLoss?.mixed_currencies && <Notice tone="info">Underlying records include all currencies in this reporting period.</Notice>}
      <ExploreMembers key={drill.key} drill={drill} organizationScopeID={organizationScopeID}
        loadMembers={loadMembers} loadLossMembers={loadLossMembers} onOpenRisk={onOpenRisk} onOpenLoss={onOpenLoss}/>
    </FocusedSheet>}
  </section>;
}

function ExploreEmpty({ measure, metric }: { measure: Measure; metric?: HomeMetric }) {
  if (measure === "risk") {
    const complete = metric && metric.population > 0 && metric.freshness === "CURRENT" && metric.completeness === "COMPLETE"
      && metric.unknown === 0 && metric.excluded === 0;
    return <EmptyState compact population="Risks in the selected scope"
      title={metric?.population === 0 ? "No assessed risk population" : complete ? "No risks outside appetite" : "No identified appetite breaches"}
      description={complete ? "No assessed risks exceed approved appetite in this source."
        : "Current, complete assessment coverage is not confirmed. Review the assessments before drawing an appetite conclusion."}/>;
  }
  return <EmptyState compact population="Loss activity in the selected period"
    title={measure === "loss-events" ? "No new loss events" : "No monetary loss flow"}
    description={measure === "loss-events" ? "No new loss events were recorded in this scope and period. Recoveries can still affect net loss."
      : "No gross loss, recovery or reversal amounts were recorded in this scope and period."}/>;
}

function validRisk(value: HomeMetric | undefined, domain: DomainMetricBundle | null): value is HomeMetric {
  return Boolean(value && domain && nonEmpty(domain.source_id) && nonEmpty(domain.source_revision)
    && value.source_revision === domain.source_revision && nonEmpty(value.definition_revision) && count(value.value) && count(value.population)
    && value.value <= value.population && value.unit === "COUNT" && value.basis === "CURRENT_POSTURE"
    && ["CURRENT", "STALE"].includes(value.freshness) && ["COMPLETE", "PARTIAL", "UNKNOWN"].includes(value.completeness)
    && (value.unknown === undefined || count(value.unknown)) && (value.excluded === undefined || count(value.excluded)));
}

function validRiskBuckets(value: OrganizationMetricBreakdown) {
  return Array.isArray(value.items) && value.items.every((item) => validBucket(item) && count(item.value))
    && new Set(value.items.map((item) => item.key)).size === value.items.length
    && value.items.reduce((sum, item) => sum + item.value, 0) === value.count;
}

function validLossBreakdown(value: LossPeriodBundle) {
  return nonEmpty(value.source_id) && nonEmpty(value.source_revision) && nonEmpty(value.definition_revision)
    && count(value.event_count) && count(value.contributing_loss_count) && validCurrencyBasis(value)
    && (value.contributing_loss_count === 0 || value.currencies.length > 0)
    && Array.isArray(value.organization_breakdown) && value.organization_breakdown.every((item) => validBucket(item)
      && count(item.loss_event_count) && count(item.contributing_loss_count) && validCurrencyBasis(item)
      && (item.contributing_loss_count === 0 || item.currencies.length > 0)
      && item.currencies.every((flow) => value.currencies.some((total) => total.currency === flow.currency)))
    && new Set(value.organization_breakdown.map((item) => item.key)).size === value.organization_breakdown.length
    && value.organization_breakdown.reduce((sum, item) => sum + item.loss_event_count, 0) === value.event_count
    && value.organization_breakdown.reduce((sum, item) => sum + item.contributing_loss_count, 0) === value.contributing_loss_count
    && value.currencies.every((total) => value.organization_breakdown.some((item) => item.currencies.some((flow) => flow.currency === total.currency))
      && value.organization_breakdown.reduce((sum, item) => {
      const flow = item.currencies.find((candidate) => candidate.currency === total.currency);
      return sum + (flow ? parseMoneyMinor(flow.net, total.currency)! : 0n);
    }, 0n) === parseMoneyMinor(total.net, total.currency));
}

function validCurrencyBasis(value: Pick<LossPeriodBundle, "currencies" | "mixed_currencies" | "net_loss">) {
  if (!validCurrencies(value.currencies) || value.mixed_currencies !== (value.currencies.length > 1)) return false;
  if (value.net_loss === undefined) return true;
  const only = value.currencies.length === 1 ? value.currencies[0] : undefined;
  return Boolean(only && value.net_loss && nonEmpty(value.net_loss.minor_units)
    && parseMoneyMinor(value.net_loss, only.currency) === parseMoneyMinor(only.net, only.currency));
}

function validCurrencies(values: LossCurrencyFlow[]) {
  return Array.isArray(values) && values.every((item) => item && nonEmpty(item.currency)
    && item.net && nonEmpty(item.net.minor_units) && parseMoneyMinor(item.net, item.currency) !== undefined)
    && new Set(values.map((item) => item.currency)).size === values.length;
}

function validBucket(item: Pick<OrganizationMetricBucket, "key" | "label" | "kind"> | null) {
  return Boolean(item && nonEmpty(item.key) && nonEmpty(item.label)
    && ["ORGANIZATION_SCOPE", "DIRECT", "UNATTRIBUTED", "UNAVAILABLE"].includes(item.kind));
}
function count(value: unknown): value is number { return Number.isSafeInteger(value) && Number(value) >= 0; }
function nonEmpty(value: unknown): value is string { return typeof value === "string" && value.trim().length > 0; }
function sourceDate(value: string) { return Number.isFinite(Date.parse(value)) ? new Date(value).toLocaleString(undefined, { timeZone: "UTC" }) + " UTC" : "time unavailable"; }
function bucketNote(kind: OrganizationMetricBucket["kind"]) {
  if (kind === "DIRECT") return "Recorded directly in this scope";
  if (kind === "UNATTRIBUTED") return "No organization area recorded";
  if (kind === "UNAVAILABLE") return "Organization area unavailable";
  return "Organization area";
}
function riskRows(value: OrganizationMetricBreakdown | undefined): Row[] {
  return value?.items.map((item) => ({ id: item.key, label: item.label, value: item.value, exact: String(item.value), note: bucketNote(item.kind) })) ?? [];
}
function lossRows(value: LossPeriodBundle | undefined, measure: "net-loss" | "loss-events", currency?: string): Row[] {
  if (!value || measure === "net-loss" && !currency) return [];
  const minor = value.organization_breakdown.map((item) => {
    const flow = item.currencies.find((candidate) => candidate.currency === currency);
    return flow ? parseMoneyMinor(flow.net, currency)! : 0n;
  });
  const maximum = minor.reduce((max, amount) => bigintMax(max, bigintAbs(amount)), 0n);
  return value.organization_breakdown.map((item, index) => ({
    id: item.key, label: item.label,
    value: measure === "loss-events" ? item.loss_event_count : normalizedMinor(bigintAbs(minor[index]!), maximum),
    exact: measure === "loss-events" ? String(item.loss_event_count) : formatLossMoneyExact(minor[index]!, currency!),
    note: bucketNote(item.kind) + (measure === "net-loss" && minor[index]! < 0n ? " · Net recovery" : ""),
  }));
}

function makeDrill(measure: Measure, label: string, metric: HomeMetric | undefined, domain: DomainMetricBundle | null,
  loss: LossPeriodBundle | undefined, scope?: string): Drill | undefined {
  const value = measure === "risk" ? metric && domain && metric.population > 0
    ? { kind: "risk" as const, id: metric.id, source: domain.source_id, revision: metric.definition_revision, count: metric.value } : undefined
    : loss ? { kind: "loss" as const, id: measure === "net-loss" ? "operational_loss_net" : "operational_loss_events",
      source: loss.source_id, revision: loss.definition_revision, count: measure === "net-loss" ? loss.contributing_loss_count : loss.event_count } : undefined;
  return value ? { ...value, label, key: JSON.stringify([value.kind, value.id, value.source, value.revision, value.count, scope]) } : undefined;
}

function ExploreMembers({ drill, organizationScopeID, loadMembers, loadLossMembers, onOpenRisk, onOpenLoss }: {
  drill: Drill; organizationScopeID?: string; loadMembers: typeof loadDomainMetricMembers;
  loadLossMembers: typeof loadLossPeriodMetricMembers; onOpenRisk?: (id: string) => void; onOpenLoss?: (id: string) => void;
}) {
  const [cursors, setCursors] = useState<string[]>([]);
  const [retry, setRetry] = useState(0);
  const cursor = cursors[cursors.length - 1];
  const request = useMemo(() => ({ key: drill.key, cursor, retry, loadMembers, loadLossMembers }), [drill.key, cursor, retry, loadMembers, loadLossMembers]);
  const [result, setResult] = useState<{ request: typeof request; state: LoadState; page: HomeMetricMemberPage | null }>();
  useEffect(() => {
    const controller = new AbortController();
    setResult({ request, state: "loading", page: null });
    const read = drill.kind === "risk" ? loadMembers : loadLossMembers;
    void read(drill.id as "operational_loss_net" | "operational_loss_events", drill.source, drill.revision,
      organizationScopeID, cursor, 50, controller.signal).then((page) => {
      if (controller.signal.aborted) return;
      const valid = page?.source_id === drill.source && page.metric_id === drill.id
        && page.definition_revision === drill.revision && page.count === drill.count
        && Array.isArray(page.items) && page.items.every((item) => item && nonEmpty(item.member_id)
          && typeof item.target_title === "string" && typeof item.state === "string" && typeof item.accessible === "boolean"
          && ["RISK", "LOSS", "MATTER", "PROGRAM"].includes(item.target_type));
      setResult({ request, state: valid ? "live" : "unavailable", page: valid ? page : null });
    }).catch(() => {
      if (!controller.signal.aborted) setResult({ request, state: "unavailable", page: null });
    });
    return () => controller.abort();
  }, [request, drill.key, drill.id, drill.kind, drill.source, drill.revision, drill.count, organizationScopeID, cursor, loadMembers, loadLossMembers]);
  const current = result?.request === request ? result : undefined;
  return <MetricMemberDrill label={drill.label} state={current?.state ?? "loading"} page={current?.page ?? null}
    hasPrevious={cursors.length > 0} onPrevious={() => setCursors((values) => values.slice(0, -1))}
    onNext={() => { if (current?.page?.next_cursor) setCursors((values) => [...values, current.page!.next_cursor!]); }}
    onRetry={() => setRetry((value) => value + 1)} onOpenRisk={onOpenRisk} onOpenLoss={onOpenLoss}/>;
}
