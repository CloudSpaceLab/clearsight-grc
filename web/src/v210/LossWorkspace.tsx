import { useEffect, useRef, useState, type FormEvent } from "react";
import { getLoss, listLosses } from "../lossApi";
import { apiErrorKind } from "../http";
import type { LossAggregate, LossSummary } from "../lossTypes";
import "./risk-workspace.css";

export function LossWorkspace({ id }: { id?: string }) { return id ? <LossDetail key={id} id={id}/> : <LossList/>; }
function LossList() {
  const [draft, setDraft] = useState(""); const [search, setSearch] = useState(""); const [items, setItems] = useState<LossSummary[]>([]); const [cursor, setCursor] = useState<string>(); const [attempt, setAttempt] = useState(0);
  const [state, setState] = useState<"loading" | "ready" | "error">("loading"); const [busy, setBusy] = useState(false); const [moreError, setMoreError] = useState(false); const generation = useRef(0); const moreLock = useRef(false);
  useEffect(() => { const controller = new AbortController(); const token = ++generation.current; moreLock.current = false; setState("loading"); setItems([]); setCursor(undefined); setBusy(false); setMoreError(false);
    listLosses({ search: search || undefined, limit: 25 }, controller.signal).then((page) => { if (generation.current === token) { setItems(page.items); setCursor(page.next_cursor); setState("ready"); } }).catch(() => { if (generation.current === token && !controller.signal.aborted) setState("error"); });
    return () => { controller.abort(); generation.current++; };
  }, [search, attempt]);
  async function more() { if (!cursor || moreLock.current) return; moreLock.current = true; const token = generation.current; setBusy(true); setMoreError(false);
    try { const page = await listLosses({ search: search || undefined, limit: 25, cursor }); if (generation.current === token) { if (page.next_cursor === cursor) { setMoreError(true); return; } setItems((current) => [...current, ...page.items]); setCursor(page.next_cursor); } }
    catch (error) { if (generation.current === token) { if (["unauthorized", "forbidden"].includes(apiErrorKind(error))) { setItems([]); setCursor(undefined); setState("error"); } else setMoreError(true); } } finally { if (generation.current === token) { moreLock.current = false; setBusy(false); } }
  }
  function submit(event: FormEvent) { event.preventDefault(); setSearch(draft.trim()); }
  return <section><header className="v-page-heading"><div><p className="v-eyebrow">Risk &amp; Losses</p><h1>Loss events</h1></div></header><nav className="v-local-nav" aria-label="Risk views"><a href="#/risks">Risk register</a><a href="#/risks/rcsa">RCSA</a><a href="#/risks/indicators">Key indicators</a><a href="#/risks/losses" aria-current="page">Loss events</a></nav><form className="v-program-search" onSubmit={submit}><label htmlFor="v-loss-search">Find a loss event</label><div><input id="v-loss-search" value={draft} maxLength={200} onChange={(event) => setDraft(event.target.value)}/><button>Search</button></div></form>
    {state === "loading" && <p role="status">Loading loss events…</p>}{state === "error" && <div className="v-panel"><p role="alert">Loss events unavailable.</p><button onClick={() => setAttempt((value) => value + 1)}>Try again</button></div>}
    {state === "ready" && <div className="v-panel v-program-table">{items.length === 0 ? <p>No loss events match this search.</p> : <table><thead><tr><th>Loss event</th><th>Occurred</th><th>Net loss</th><th>Status</th></tr></thead><tbody>{items.map(({ loss, totals }) => <tr key={loss.id}><td><a href={`#/risks/losses/${encodeURIComponent(loss.id)}`}>{loss.title}</a><small>{loss.code}</small></td><td><span className="v-mobile-label">Occurred</span>{formatDate(loss.occurred_at)}</td><td><span className="v-mobile-label">Net loss</span>{totals.currency === loss.currency ? formatMinor(totals.net_loss_minor, totals.currency) : "Amount unavailable"}</td><td><span className="v-mobile-label">Status</span>{label(loss.status)}</td></tr>)}</tbody></table>}{moreError && <p role="alert">More loss events unavailable.</p>}{cursor && <button disabled={busy} onClick={() => void more()}>{busy ? "Loading…" : "Load more"}</button>}</div>}
  </section>;
}
function LossDetail({ id }: { id: string }) {
  const [data, setData] = useState<LossAggregate>(); const [state, setState] = useState<"loading" | "ready" | "error">("loading"); const [attempt, setAttempt] = useState(0);
  useEffect(() => { const controller = new AbortController(); setData(undefined); setState("loading"); getLoss(id, controller.signal).then((value) => { if (!controller.signal.aborted) { if (value.loss.id !== id) { setState("error"); return; } setData(value); setState("ready"); } }).catch(() => { if (!controller.signal.aborted) setState("error"); }); return () => controller.abort(); }, [id, attempt]);
  if (state === "loading") return <p role="status">Loading loss event…</p>;
  if (state === "error" || !data) return <section><h1>Loss unavailable</h1><button onClick={() => setAttempt((value) => value + 1)}>Try again</button><p><a href="#/risks/losses">Loss events</a></p></section>;
  const { loss, totals } = data; const compatible = totals.currency === loss.currency;
  return <section className="v-risk-detail"><a className="v-back" href="#/risks/losses">← Loss events</a><header className="v-page-heading"><div><p className="v-eyebrow">{loss.code} · Version {loss.version}</p><h1>{loss.title}</h1></div><span className="v-status">{label(loss.status)}</span></header><div className="v-panel"><h2>Loss details</h2><dl><div><dt>Event type</dt><dd>{label(loss.event_type)}</dd></div><div><dt>Occurred</dt><dd>{formatDate(loss.occurred_at)}</dd></div><div><dt>Gross loss</dt><dd>{compatible ? formatMinor(totals.gross_amount_minor, totals.currency) : "Amount unavailable"}</dd></div><div><dt>Recovered</dt><dd>{compatible ? formatMinor(totals.recovered_amount_minor, totals.currency) : "Amount unavailable"}</dd></div><div><dt>Net loss</dt><dd>{compatible ? formatMinor(totals.net_loss_minor, totals.currency) : "Amount unavailable"}</dd></div><div><dt>Owner</dt><dd>{data.owner_display_name || (loss.owner_principal_id ? "Name unavailable" : "Not recorded")}</dd></div></dl></div><div className="v-panel"><h2>Recovery history</h2>{data.recoveries?.length ? <ul className="v-attention">{data.recoveries.map((item) => <li key={item.id}><div><strong>{label(item.kind)}</strong><small>{formatDate(item.recovered_at)}{item.loss_version !== loss.version ? " · Earlier loss version" : ""}</small></div><span>{item.currency === loss.currency ? formatMinor(item.amount_minor, item.currency) : "Amount unavailable"}</span></li>)}</ul> : <p>No recoveries recorded.</p>}</div></section>;
}
export function formatMinor(minor: number, currency: string) {
  if (!Number.isSafeInteger(minor) || !/^[A-Z]{3}$/.test(currency)) return "Amount unavailable";
  try {
    if (!Intl.supportedValuesOf("currency").includes(currency)) return "Amount unavailable";
    const digits = new Intl.NumberFormat("en", { style: "currency", currency }).resolvedOptions().maximumFractionDigits;
    if (digits === undefined) return "Amount unavailable";
    const amount = BigInt(minor); const absolute = amount < 0n ? -amount : amount; const scale = 10n ** BigInt(digits);
    const whole = new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(absolute / scale);
    const decimal = digits ? `${new Intl.NumberFormat(undefined).formatToParts(1.1).find((part) => part.type === "decimal")?.value ?? "."}${(absolute % scale).toString().padStart(digits, "0")}` : "";
    return `${amount < 0n ? "−" : ""}${currency} ${whole}${decimal}`;
  } catch { return "Amount unavailable"; }
}
function label(value: string) { return value.toLowerCase().replaceAll("_", " "); }
function formatDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "date unavailable" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeZone: "UTC" }).format(date); }
