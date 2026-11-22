import { useEffect, useState, type FormEvent } from "react";
import { loadActorWorkDueCounts, type ActorWorkDueCounts, type RuntimeContext } from "../api";
import { validWorkCounts } from "../workSourceReceipt";
import { loadHomeMetrics, type HomeMetricBundle } from "../metricApi";
import { loadOversight, type OversightSnapshot } from "../oversightApi";
import { loadIndicatorPopulation } from "../indicatorApi";
import type { IndicatorPopulationPage } from "../indicatorTypes";
import { minimumReportingStart, type ReportingPeriodQuery } from "../reportingPeriod";
import { isAdministrativeHome } from "./homePresentation";
import "./home.css";

type HomeState = { kind: "loading" } | { kind: "live"; metrics: HomeMetricBundle; oversight: OversightSnapshot } | { kind: "unavailable" };
export function Home({ context }: { context: RuntimeContext }) {
  return <HomeContent key={JSON.stringify([context.tenant.id, context.legal_entity.id, context.actor.id])} context={context}/>;
}
function HomeContent({ context }: { context: RuntimeContext }) {
  const [state, setState] = useState<HomeState>({ kind: "loading" });
  const [period, setPeriod] = useState<ReportingPeriodQuery>();
  const [draft, setDraft] = useState("");
  const [attempt, setAttempt] = useState(0);
  const [indicators, setIndicators] = useState<IndicatorPopulationPage>();
  const [indicatorsState, setIndicatorsState] = useState<"loading" | "ready" | "error">("loading");
  const [assignments, setAssignments] = useState<ActorWorkDueCounts>();
  const [assignmentsState, setAssignmentsState] = useState<"loading" | "ready" | "error">("loading");
  const [assignmentsAttempt, setAssignmentsAttempt] = useState(0);
  const visibleAssignments = assignments && validWorkCounts(assignments, { tenantID: context.tenant.id, legalEntityID: context.legal_entity.id, principalID: context.actor.id }) ? assignments : undefined;
  useEffect(() => {
    let active = true;
    setAssignments(undefined); setAssignmentsState("loading");
    const scope = { tenantID: context.tenant.id, legalEntityID: context.legal_entity.id, principalID: context.actor.id };
    if (!scope.tenantID || !scope.legalEntityID || scope.legalEntityID === "*" || !scope.principalID) { setAssignmentsState("error"); return () => { active = false; }; }
    loadActorWorkDueCounts().then((value) => {
      if (!active) return;
      if (!validWorkCounts(value, scope)) { setAssignmentsState("error"); return; }
      setAssignments(value); setAssignmentsState("ready");
    }).catch(() => { if (active) setAssignmentsState("error"); });
    return () => { active = false; };
  }, [context.tenant.id, context.legal_entity.id, context.actor.id, assignmentsAttempt]);
  useEffect(() => {
    const controller = new AbortController(); setIndicators(undefined); setIndicatorsState("loading");
    loadIndicatorPopulation({ kind: "KRI", limit: 3 }, controller.signal).then((page) => {
      if (!controller.signal.aborted) { if (!Array.isArray(page.items) || page.organization_scope_id) { setIndicatorsState("error"); return; } setIndicators(page); setIndicatorsState("ready"); }
    }).catch(() => { if (!controller.signal.aborted) setIndicatorsState("error"); });
    return () => controller.abort();
  }, [context.tenant.id, context.legal_entity.id, context.actor.id]);
  useEffect(() => {
    let active = true; setState({ kind: "loading" });
    Promise.all([loadHomeMetrics(period), loadOversight(period)]).then(([metrics, oversight]) => {
      if (!active) return;
      const left = metrics.reporting_period, right = oversight.reporting_period;
      const matches = metrics.scope_kind === "LEGAL_ENTITY" && metrics.scope_id === context.legal_entity.id && !oversight.organization_scope_id && left && right && /^\d{4}-\d{2}-\d{2}$/.test(left.start_date) && /^\d{4}-\d{2}-\d{2}$/.test(left.end_date) && Number.isFinite(Date.parse(`${left.end_date}T00:00:00Z`)) && Number.isInteger(left.max_days) && left.max_days > 0 && left.start_date === right.start_date && left.end_date === right.end_date && left.mode === right.mode && metrics.source_revision && metrics.source_revision === oversight.projection_version && (!period || (left.start_date === period.start_date && left.end_date === period.end_date));
      if (!matches) { setState({ kind: "unavailable" }); return; }
      setDraft(left.start_date); setState({ kind: "live", metrics, oversight });
    }).catch(() => { if (active) setState({ kind: "unavailable" }); });
    return () => { active = false; };
  }, [context.tenant.id, context.legal_entity.id, context.actor.id, period, attempt]);
  function apply(event: FormEvent) {
    event.preventDefault(); if (state.kind !== "live") return;
    const current = state.metrics.reporting_period;
    if (draft < minimumReportingStart(current) || draft > current.end_date) return;
    setPeriod({ start_date: draft, end_date: current.end_date });
  }
  const admin = isAdministrativeHome(context);
  const assignmentPanel = <section className="v-panel v-home-assignments"><header><h2>My assignments</h2><a href="#/work">View my work</a></header>
      {assignmentsState === "loading" && <p role="status">Loading assignments…</p>}
      {assignmentsState === "error" && <><p role="alert">Assignments unavailable.</p><button type="button" onClick={() => setAssignmentsAttempt((value) => value + 1)}>Retry assignments</button></>}
      {assignmentsState === "ready" && visibleAssignments && <><ul><li>{visibleAssignments.total.toLocaleString()} open</li><li><span className={visibleAssignments.overdue > 0 ? "v-status v-status--high" : undefined}>{visibleAssignments.overdue.toLocaleString()} overdue</span></li><li><span className={visibleAssignments.due_soon > 0 ? "v-home-indicator v-home-indicator--watch" : undefined}>{visibleAssignments.due_soon.toLocaleString()} due within 7 days</span></li></ul><p>Workflow assignments · As of {formatDateTime(visibleAssignments.as_of)}</p></>}
    </section>;
  const indicatorPanel = <section className="v-panel v-home-indicators"><h2>Current risk indicators</h2>{indicatorsState === "loading" && <p role="status">Loading indicators…</p>}{indicatorsState === "error" && <p>Indicators unavailable.</p>}{indicatorsState === "ready" && indicators && (indicators.items.length ? <><ul className="v-attention">{indicators.items.map(({ indicator }) => <li key={indicator.check_id}><div><strong>{indicator.check_name}</strong><small>{indicator.reason}</small>{indicator.evaluated_at && <small>Checked {formatDate(indicator.evaluated_at)}</small>}</div><span className={`v-home-indicator v-home-indicator--${indicator.state.toLowerCase()}`}>{indicatorLabel(indicator.state)}</span></li>)}</ul>{(indicators.next_cursor || !indicators.complete || indicators.truncated) && <p>More indicators may be available.</p>}</> : <p>{indicators.complete ? "No linked risk indicators recorded." : "Indicator results are incomplete."}</p>)}</section>;
  return <section className="v-home"><header className="v-page-heading"><div><p className="v-eyebrow">{state.kind === "live" && state.metrics.posture_as_of ? `As of ${formatDate(state.metrics.posture_as_of)}` : context.legal_entity.name}</p><h1>{admin ? "Dashboard" : "Overview"}</h1></div></header>
    {!admin && assignmentPanel}
    {state.kind === "loading" && <p role="status">{admin ? "Loading dashboard…" : "Loading overview…"}</p>}
    {state.kind === "unavailable" && <div className="v-panel"><p role="alert">{admin ? "Dashboard data unavailable." : "Home data unavailable."}</p><button onClick={() => setAttempt(attempt + 1)}>Try again</button>{period && <button type="button" onClick={() => setPeriod(undefined)}>Reset dates</button>}</div>}
    {admin && state.kind !== "live" && assignmentPanel}
    {state.kind === "live" && <><form className="v-home-period" onSubmit={apply}><label>From<input type="date" required min={minimumReportingStart(state.metrics.reporting_period)} max={state.metrics.reporting_period.end_date} value={draft} onChange={(event) => setDraft(event.target.value)}/></label><div><span>To</span><strong>{formatDate(state.metrics.reporting_period.end_date)}</strong></div><button type="submit">Apply dates</button></form><p className="v-home-period-note">Current posture is unaffected by the date range.</p>
      <div className="v-metrics">{state.metrics.items.slice(0, 4).map((metric) => <article className="v-metric" key={metric.id}><p>{metric.label}</p><strong>{metric.completeness === "UNKNOWN" ? "—" : metric.value.toLocaleString()}</strong>{(metric.freshness === "STALE" || metric.completeness !== "COMPLETE") && <small>{[metric.freshness === "STALE" ? "Outdated" : "", metric.completeness === "PARTIAL" ? "Incomplete coverage" : metric.completeness === "UNKNOWN" ? "Coverage unknown" : ""].filter(Boolean).join(" · ")}</small>}</article>)}</div>
      <div className="v-home-grid"><section className="v-panel"><header><h2>Needs attention</h2></header>{state.oversight.interventions.length === 0 ? <p>No priority items in the current scope.</p> : <ul className="v-attention v-home-priorities">{state.oversight.interventions.slice(0, 10).map((item) => <li key={`${item.target_type}:${item.target_id}`}><div><strong>{item.title}</strong><small>{item.owner_name || "Owner not recorded"}{item.due_at ? ` · Due ${formatDate(item.due_at)}` : ""}</small></div><div className="v-home-priority-actions"><span className={item.priority >= 4 ? "v-status v-status--high" : "v-status"}>{item.priority === 5 ? "Critical" : item.priority === 4 ? "High" : item.state.replaceAll("_", " ").toLowerCase()}</span>{priorityMatterHref(item) && <a href={priorityMatterHref(item)} aria-label={`Open ${item.title}`}>Open issue</a>}</div></li>)}</ul>}</section><div className="v-home-side">{admin && assignmentPanel}{indicatorPanel}</div></div>
    </>}
  </section>;
}
function formatDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "date unavailable" : new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" }).format(date); }
function formatDateTime(value: string) { return new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit", timeZone: "UTC", timeZoneName: "short" }).format(new Date(value)); }
function indicatorLabel(value: string) { return ({ NORMAL: "Normal", WATCH: "Watch", BREACH: "Breach", UNKNOWN: "Status unknown" } as Record<string, string>)[value] ?? "Status unknown"; }
function priorityMatterHref(item: OversightSnapshot["interventions"][number]) { const id = item.target_id; return item.target_type === "MATTER" && typeof id === "string" && id.trim() && id.length <= 200 && !/[\u0000-\u001f\u007f]/.test(id) ? `#/work/issues/${encodeURIComponent(id)}` : undefined; }
