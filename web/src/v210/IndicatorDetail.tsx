import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { loadMonitoringResults } from "../monitoringApi";
import type { MonitoringNativeMeasurement, MonitoringResult } from "../monitoringTypes";
import type { RiskIndicatorDetail } from "../riskTypes";
import { apiErrorKind } from "../http";
import { limitText, measurementValue, validConcern } from "./indicatorFormatting";
import "./indicator-detail.css";

export function IndicatorDetail({ indicator, loadResults = loadMonitoringResults, onAccessDenied, inSheet = false }: { indicator: RiskIndicatorDetail; loadResults?: typeof loadMonitoringResults; onAccessDenied?: () => void; inSheet?: boolean }) {
  const [items, setItems] = useState<MonitoringResult[]>([]); const [state, setState] = useState<"loading" | "ready" | "error" | "denied">("loading");
  const [attempt, setAttempt] = useState(0); const [selected, setSelected] = useState<MonitoringResult>();
  const detail = useRef<HTMLElement>(null); const origin = useRef<HTMLButtonElement>(null);
  useLayoutEffect(() => { if (selected) { detail.current?.focus({ preventScroll: true }); detail.current?.scrollIntoView?.({ block: "nearest" }); } }, [selected]);
  useEffect(() => {
    let active = true; setItems([]); setSelected(undefined); setState("loading");
    if (!indicator.check_id || !indicator.program_id || indicator.link.monitoring_check_id !== indicator.check_id || indicator.link.monitoring_check_version !== indicator.check_version || indicator.link.program_id !== indicator.program_id || !Number.isSafeInteger(indicator.check_version) || indicator.check_version < 1) { setState("error"); return () => { active = false; }; }
    loadResults(indicator.check_id, indicator.check_version).then((results) => {
      if (!active) return;
      if (!Array.isArray(results) || results.length > 20 || results.some((result) => !result?.id || result.program_id !== indicator.program_id || result.monitoring_check_id !== indicator.check_id || result.monitoring_check_version !== indicator.check_version || !result.evaluation) || new Set(results.map((result) => result.id)).size !== results.length) throw new Error("Invalid evaluation history");
      setItems(results); setState("ready");
    }).catch((error: unknown) => { if (active) { if (["unauthorized", "forbidden", "not_found"].includes(apiErrorKind(error))) { setItems([]); setState("denied"); onAccessDenied?.(); } else setState("error"); } });
    return () => { active = false; };
  }, [indicator.check_id, indicator.check_version, indicator.program_id, indicator.link.monitoring_check_id, indicator.link.monitoring_check_version, indicator.link.program_id, loadResults, attempt]);
  if (state === "denied") return <p role="alert">Indicator unavailable.</p>;
  const trend = trendData(items, indicator.native_measurement);
  return <section className="v-indicator-detail"><header>{!inSheet && <h2>{indicator.check_name}</h2>}<p>{indicator.check_code} · Version {indicator.check_version}</p></header>
    <div className="v-indicator-detail-links"><a href={`#/programs/${encodeURIComponent(indicator.program_id)}`}>View program</a>{indicator.intervention?.matter_id && <a href={`#/work/issues/${encodeURIComponent(indicator.intervention.matter_id)}`}>View issue{indicator.intervention.reference ? ` ${indicator.intervention.reference}` : ""}</a>}</div>{indicator.intervention?.matter_id && <p>Issue status: {issueStatus(indicator.intervention.status)}</p>}<p><strong>{status(indicator.state)}</strong>{indicator.reason && ` · ${indicator.reason}`}</p><p>{indicator.evaluated_at ? `Evaluated ${date(indicator.evaluated_at)}` : "No evaluation date"}</p>
    <dl><div><dt>Latest value</dt><dd>{measurementValue(indicator.native_measurement)}</dd></div><div><dt>Approved limit</dt><dd>{limitText(indicator.native_measurement)}</dd></div>{period(indicator.native_measurement) && <div><dt>Reporting period</dt><dd>{period(indicator.native_measurement)}</dd></div>}<div><dt>Owner</dt><dd>{indicator.owner_display_name || "Name unavailable"}</dd></div><div><dt>Reviewer</dt><dd>{indicator.reviewer_display_name || "Name unavailable"}</dd></div>{validConcern(indicator.score, indicator.denominator) && <div><dt>Concern</dt><dd>{indicator.score} / {indicator.denominator} concern</dd></div>}</dl>
    <h3>Evaluation history</h3>{state === "loading" && <p role="status">Loading evaluations…</p>}{state === "error" && <><p role="alert">Evaluation history unavailable.</p><button type="button" onClick={() => setAttempt((value) => value + 1)}>Try again</button></>}{state === "ready" && <>{items.length === 0 ? <p>No evaluations recorded for this version.</p> : <>{trend && <><svg role="img" aria-label={`${indicator.check_name} evaluation trend`} viewBox="0 0 100 40" preserveAspectRatio="none"><polyline points={trend.points} fill="none" stroke="currentColor" strokeWidth="1.5"/></svg><p>Range: {measurementValue(trend.min)}–{measurementValue(trend.max)}</p></>}<p>Showing up to 20 evaluations.</p><div className="v-indicator-history-wrap"><table aria-label={`${indicator.check_name} evaluation history`}><thead><tr><th>Evaluated</th><th>Value</th><th>Limit</th><th>Result</th><th>Concern</th><th>Details</th></tr></thead><tbody>{items.map((result) => <tr key={result.id}><td data-label="Evaluated">{date(result.evaluated_at)}</td><td data-label="Value">{measurementValue(result.evaluation.measurement)}</td><td data-label="Limit">{limitText(result.evaluation.measurement)}</td><td data-label="Result">{condition(result.evaluation.measurement?.condition)}</td><td data-label="Concern">{band(result.evaluation.band)}</td><td><button type="button" onClick={(event) => { origin.current = event.currentTarget; setSelected(result); }}>View evaluation for {date(result.evaluated_at)}</button></td></tr>)}</tbody></table></div></>}</>}
    {selected && <section ref={detail} tabIndex={-1} className="v-indicator-evaluation"><h3>Evaluation detail</h3><p>Evaluated {date(selected.evaluated_at)}</p>{selected.evaluation.rule_results?.length ? <ul>{selected.evaluation.rule_results.map((rule, index) => <li key={`${rule.rule_id ?? rule.field_id}:${index}`}>{rule.reason || "Result detail unavailable."}</li>)}</ul> : <p>No evaluation details recorded.</p>}<button type="button" onClick={() => { setSelected(undefined); origin.current?.focus(); }}>Close detail</button></section>}
  </section>;
}

function date(value: string) { const parsed = new Date(value); return Number.isNaN(parsed.getTime()) ? "Date unavailable" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short", timeZone: "UTC" }).format(parsed) + " UTC"; }
function period(value?: MonitoringNativeMeasurement) { const start = value?.reporting_period_start; const end = value?.reporting_period_end; if (!start && !end) return undefined; return start && end ? `${date(start)} – ${date(end)}` : start ? `From ${date(start)}` : `Until ${date(end!)}`; }
function condition(value?: string) { return ({ WITHIN: "Within limit", BREACHED: "Outside limit", UNKNOWN: "Unknown" } as Record<string, string>)[value ?? ""] ?? "Unknown"; }
function status(value?: string) { return ({ NORMAL: "Normal", WATCH: "Watch", BREACH: "Breach", UNKNOWN: "Unknown" } as Record<string, string>)[value ?? ""] ?? "Unknown"; }
function issueStatus(value?: string) { return ({ DRAFT: "Draft", TRIAGE: "Triage", ASSESSMENT: "Assessment", DECISION_REQUIRED: "Decision needed", ACTION_IN_PROGRESS: "Action in progress", RESPONSE_PREPARATION: "Response in preparation", VERIFICATION: "Verification", CLOSED: "Closed", CANCELLED: "Cancelled" } as Record<string, string>)[value ?? ""] ?? "Unavailable"; }
function band(value?: string) { return ({ LOW: "Low concern", MODERATE: "Moderate concern", HIGH: "High concern", CRITICAL: "Critical concern" } as Record<string, string>)[value ?? ""] ?? "Concern unavailable"; }
function trendData(results: MonitoringResult[], current?: MonitoringNativeMeasurement): { points: string; min: MonitoringNativeMeasurement; max: MonitoringNativeMeasurement } | undefined {
  if (!current?.field || !current.unit) return undefined;
  const times = results.map((result) => Date.parse(result.evaluated_at));
  if (times.some((value) => !Number.isFinite(value)) || new Set(times).size !== times.length) return undefined;
  const values = [...results].sort((a, b) => Date.parse(a.evaluated_at) - Date.parse(b.evaluated_at)).map((result) => result.evaluation.measurement).filter((value): value is MonitoringNativeMeasurement => Boolean(value?.value));
  const signature = (value: MonitoringNativeMeasurement) => {
    if (value.limits !== undefined && (!Array.isArray(value.limits) || value.limits.some((limit) => !limit || typeof limit.operator !== "string" || typeof limit.expected !== "string"))) return undefined;
    return JSON.stringify([value.field, value.unit, value.currency ?? "", value.duration_unit ?? "", value.precision ?? 0, (value.limits ?? []).map((limit) => [limit.operator, limit.expected])]);
  };
  const expected = signature(current);
  if (!expected || values.length < 2 || values.length !== results.length || values.some((value) => !["WITHIN", "BREACHED"].includes(value.condition ?? "") || signature(value) !== expected)) return undefined;
  if (values.some((value) => !/^[+-]?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?$/.test(value.value ?? ""))) return undefined;
  const numbers = values.map((value) => Number(value.value));
  if (numbers.some((value) => !Number.isFinite(value) || Math.abs(value) > 1e12)) return undefined;
  if (numbers.some((value, index) => numbers.some((other, otherIndex) => index !== otherIndex && value === other && values[index]?.value !== values[otherIndex]?.value))) return undefined;
  const minIndex = numbers.indexOf(Math.min(...numbers)); const maxIndex = numbers.indexOf(Math.max(...numbers)); const min = numbers[minIndex]!; const max = numbers[maxIndex]!;
  if (min === max) return undefined;
  return { min: values[minIndex]!, max: values[maxIndex]!, points: numbers.map((value, index) => `${index * 100 / (numbers.length - 1)},${38 - ((value - min) / (max - min)) * 36}`).join(" ") };
}

