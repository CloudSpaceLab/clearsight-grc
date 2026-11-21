import { useEffect, useRef, useState } from "react";
import { loadIndicatorPopulation } from "../indicatorApi";
import type { IndicatorPopulationItem } from "../indicatorTypes";
import type { MonitoringNativeMeasurement, SourceOperator } from "../monitoringTypes";
import type { RiskIndicatorKind } from "../riskTypes";
import "./key-indicators.css";

export function KeyIndicators({ canViewLosses = false }: { canViewLosses?: boolean }) {
  const [items, setItems] = useState<IndicatorPopulationItem[]>([]);
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [kind, setKind] = useState<"" | RiskIndicatorKind>(""); const [search, setSearch] = useState("");
  const [cursor, setCursor] = useState<string>(); const [complete, setComplete] = useState(true); const [busy, setBusy] = useState(false); const [moreError, setMoreError] = useState(false); const [attempt, setAttempt] = useState(0);
  const generation = useRef(0);
  useEffect(() => {
    const token = ++generation.current; const controller = new AbortController(); setState("loading"); setItems([]); setCursor(undefined); setComplete(true); setMoreError(false); setBusy(false);
    loadIndicatorPopulation({ limit: 25, ...(kind ? { kind } : {}) }, controller.signal).then((page) => {
      if (token !== generation.current || controller.signal.aborted) return;
      if (!validItems(page.items)) { setState("error"); return; }
      setItems(page.items); setCursor(page.next_cursor); setComplete(page.complete === true && (!page.truncated || Boolean(page.next_cursor))); setState("ready");
    }).catch(() => { if (token === generation.current && !controller.signal.aborted) setState("error"); });
    return () => { controller.abort(); generation.current++; };
  }, [kind, attempt]);
  async function more() {
    if (!cursor || busy) return; const token = generation.current; setBusy(true); setMoreError(false);
    try {
      const page = await loadIndicatorPopulation({ limit: 25, ...(kind ? { kind } : {}), cursor });
      if (token !== generation.current) return;
      if (!validItems(page.items) || page.next_cursor === cursor) { setMoreError(true); return; }
      setItems((current) => [...current, ...page.items]); setCursor(page.next_cursor); setComplete((value) => value && page.complete === true && (!page.truncated || Boolean(page.next_cursor)));
    } catch { if (token === generation.current) setMoreError(true); }
    finally { if (token === generation.current) setBusy(false); }
  }
  const query = search.trim().toLocaleLowerCase();
  const visible = items.filter(({ indicator, risks }) => [indicator.check_name, indicator.check_code, indicator.program_name, ...risks.map((risk) => risk.name)].some((value) => value?.toLocaleLowerCase().includes(query)));
  return <section className="v-indicators"><header className="v-page-heading"><div><p className="v-eyebrow">Risk &amp; Losses</p><h1>Key indicators</h1></div></header>
    <nav className="v-local-nav" aria-label="Risk views"><a href="#/risks">Risk register</a><a href="#/risks/rcsa">RCSA</a><a href="#/risks/indicators" aria-current="page">Key indicators</a>{canViewLosses && <a href="#/risks/losses">Loss events</a>}</nav>
    <div className="v-indicator-filters"><label>Indicator type<select value={kind} onChange={(event) => setKind(event.target.value as "" | RiskIndicatorKind)}><option value="">All types</option><option value="KRI">Risk indicators</option><option value="KCI">Control indicators</option></select></label><label>Search loaded indicators<input type="search" value={search} onChange={(event) => setSearch(event.target.value)}/></label></div>
    {state === "loading" && <p role="status">Loading indicators…</p>}{state === "error" && <div className="v-panel"><p role="alert">Indicators unavailable.</p><button onClick={() => setAttempt((value) => value + 1)}>Try again</button></div>}
    {state === "ready" && <section className="v-panel v-indicator-register">{!complete && <p role="status">Some indicators could not be loaded.</p>}{items.length === 0 ? <p>{complete ? "No indicators found." : "Indicator list unavailable."}</p> : visible.length === 0 ? <p>No loaded indicators match this search.</p> : <ul>{visible.map(({ indicator, risks, risks_truncated, kind_conflict }) => <li key={`${indicator.check_id}:${indicator.link.kind}`}><div><strong>{indicator.check_name}</strong><small>{indicator.check_code} · {indicator.link.kind} · {indicator.program_name}</small>{risks.length > 0 && <small>{risks.map((risk) => risk.name).join(", ")}</small>}{risks_truncated && <small>More linked risks not shown.</small>}{kind_conflict && <small>Indicator type requires review.</small>}</div><div><span className="v-indicator-mobile-label">Latest observation</span><strong>{measurementValue(indicator.native_measurement)}</strong>{indicator.score !== undefined && <small>Concern: {indicator.score} / {indicator.denominator} points</small>}<small>{indicator.evaluated_at ? `Evaluated ${formatDate(indicator.evaluated_at)}` : "No evaluation date"}</small></div><div><span className="v-indicator-mobile-label">Approved limit</span><strong>{limitText(indicator.native_measurement)}</strong></div><div><span className={`v-indicator-state ${stateClass(indicator.state)}`}>{stateLabel(indicator.state)}</span>{indicator.reason && <small>{indicator.reason}</small>}{indicator.freshness_minutes > 0 && <small>Freshness limit: {freshness(indicator.freshness_minutes)}</small>}</div></li>)}</ul>}{moreError && <p role="alert">More indicators unavailable. Try again.</p>}{cursor && <button type="button" disabled={busy} onClick={() => void more()}>{busy ? "Loading…" : "Load more indicators"}</button>}</section>}
  </section>;
}

function measurementValue(value?: MonitoringNativeMeasurement) {
  if (!value) return "Native value unavailable";
  if (!value.value) return "No observation";
  const numeric = exactDecimal(value.value, value.precision);
  if (value.unit === "PERCENT") return `${numeric}%`;
  if (value.unit === "MONEY") return `${value.currency || "Currency"} ${numeric}`;
  if (value.unit === "DURATION") return `${numeric} ${duration(value.duration_unit)}`;
  return numeric;
}
// Preserve source digits and avoid floating-point conversion or unbounded expansion.
function exactDecimal(value: string, precision?: number) {
  const match = /^([+-]?)(\d+)(?:\.(\d+))?(?:[eE]([+-]?\d+))?$/.exec(value);
  if (!match) return value;
  const exponent = Number(match[4] ?? "0");
  if (!Number.isSafeInteger(exponent) || Math.abs(exponent) > 100 || (precision !== undefined && (!Number.isInteger(precision) || precision < 0 || precision > 20))) return value;
  const digits = match[2] + (match[3] ?? "");
  const point = match[2]!.length + exponent;
  const expanded = point <= 0 ? `0.${"0".repeat(-point)}${digits}` : point >= digits.length ? `${digits}${"0".repeat(point - digits.length)}` : `${digits.slice(0, point)}.${digits.slice(point)}`;
  const [integer, sourceFraction = ""] = expanded.split(".");
  const fraction = sourceFraction.padEnd(Math.max(sourceFraction.length, precision ?? 0), "0");
  return `${match[1]}${integer!.replace(/\B(?=(\d{3})+(?!\d))/g, ",")}${fraction ? `.${fraction}` : ""}`;
}
function freshness(minutes: number) { if (minutes % 1440 === 0) return `${minutes / 1440} ${minutes === 1440 ? "day" : "days"}`; if (minutes % 60 === 0) return `${minutes / 60} ${minutes === 60 ? "hour" : "hours"}`; return `${minutes} ${minutes === 1 ? "minute" : "minutes"}`; }
function duration(value?: string) { return ({ SECONDS: "seconds", MINUTES: "minutes", HOURS: "hours", DAYS: "days" } as Record<string, string>)[value ?? ""] ?? "duration"; }
function operator(value: SourceOperator) { return ({ EQUALS: "=", NOT_EQUALS: "≠", GREATER_THAN: ">", GREATER_OR_EQUAL: "≥", LESS_THAN: "<", LESS_OR_EQUAL: "≤" } as Partial<Record<SourceOperator, string>>)[value]; }
function limitText(value?: MonitoringNativeMeasurement) { if (!value?.limits?.length) return "Not recorded"; const limits = value.limits.map((limit) => { const symbol = operator(limit.operator); return symbol && limit.expected ? `${symbol} ${measurementValue({ ...value, value: limit.expected })}` : "Limit unavailable"; }); return limits.join(" · "); }
function stateLabel(value: string) { return ({ NORMAL: "Normal", WATCH: "Watch", BREACH: "Breach", UNKNOWN: "Unknown" } as Record<string, string>)[value] ?? "Unknown"; }
function stateClass(value: string) { return value === "BREACH" ? "is-breach" : value === "WATCH" ? "is-watch" : value === "NORMAL" ? "is-normal" : "is-unknown"; }
function formatDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "date unavailable" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date); }
function validItems(items: unknown): items is IndicatorPopulationItem[] { return Array.isArray(items) && items.every((item) => item && typeof item === "object" && item.indicator && typeof item.indicator.check_id === "string" && typeof item.indicator.check_name === "string" && item.indicator.link && typeof item.indicator.link.kind === "string" && Array.isArray(item.risks)); }
