import { useEffect, useRef, useState } from "react";
import { getRCSACycle, listRCSACycles } from "../rcsaApi";
import type { RCSACycleDetail, RCSACycleSummary } from "../rcsaTypes";
import "./risk-workspace.css";

export function RCSAWorkspace({ id, canViewLosses = false }: { id?: string; canViewLosses?: boolean }) { return id ? <CycleDetail key={id} id={id}/> : <CycleList canViewLosses={canViewLosses}/>; }
function CycleList({ canViewLosses }: { canViewLosses: boolean }) {
  const [items, setItems] = useState<RCSACycleSummary[]>([]); const [cursor, setCursor] = useState<string>(); const [complete, setComplete] = useState(false);
  const [state, setState] = useState<"loading" | "ready" | "error">("loading"); const [busy, setBusy] = useState(false); const [moreError, setMoreError] = useState(false); const [attempt, setAttempt] = useState(0); const generation = useRef(0);
  useEffect(() => { const controller = new AbortController(); const token = ++generation.current; setState("loading"); setItems([]); setCursor(undefined); setMoreError(false); setBusy(false);
    listRCSACycles({ limit: 25 }, controller.signal).then((page) => { if (token === generation.current) { setItems(page.items); setCursor(page.next_cursor); setComplete(page.complete); setState("ready"); } }).catch(() => { if (token === generation.current && !controller.signal.aborted) setState("error"); });
    return () => { controller.abort(); generation.current++; };
  }, [attempt]);
  async function more() { if (!cursor || busy) return; const token = generation.current; setBusy(true); setMoreError(false);
    try { const page = await listRCSACycles({ limit: 25, cursor }); if (token === generation.current) { if (page.next_cursor === cursor) { setMoreError(true); return; } setItems((current) => [...current, ...page.items]); setCursor(page.next_cursor); setComplete((current) => current && page.complete); } }
    catch { if (token === generation.current) setMoreError(true); } finally { if (token === generation.current) setBusy(false); }
  }
  return <section><header className="v-page-heading"><div><p className="v-eyebrow">Risk &amp; Losses</p><h1>Risk assessments</h1></div></header><nav className="v-local-nav" aria-label="Risk views"><a href="#/risks">Risk register</a><a href="#/risks/rcsa" aria-current="page">RCSA</a><a href="#/risks/indicators">Key indicators</a>{canViewLosses && <a href="#/risks/losses">Loss events</a>}</nav>
    {state === "loading" && <p role="status">Loading assessments…</p>}{state === "error" && <div className="v-panel"><p role="alert">Assessments unavailable.</p><button onClick={() => setAttempt((value) => value + 1)}>Try again</button></div>}
    {state === "ready" && <div className="v-panel v-program-table">{items.length === 0 ? <p>{complete ? "No assessment cycles found." : "Assessment list unavailable."}</p> : <table><thead><tr><th>Assessment</th><th>Risks</th><th>Controls</th><th>Status</th></tr></thead><tbody>{items.map(({ cycle, risk_count, control_count }) => <tr key={cycle.id}><td><a href={`#/risks/rcsa/${encodeURIComponent(cycle.id)}`}>{cycle.name}</a><small>{cycle.code}</small></td><td><span className="v-mobile-label">Risks</span>{risk_count}</td><td><span className="v-mobile-label">Controls</span>{control_count}</td><td><span className="v-mobile-label">Status</span>{label(cycle.status)}</td></tr>)}</tbody></table>}{items.length > 0 && !complete && <p>Assessment list may be incomplete.</p>}{moreError && <p role="alert">More assessments unavailable.</p>}{cursor && <button disabled={busy} onClick={() => void more()}>{busy ? "Loading…" : "Load more"}</button>}</div>}
  </section>;
}
function CycleDetail({ id }: { id: string }) {
  const [data, setData] = useState<RCSACycleDetail>(); const [state, setState] = useState<"loading" | "ready" | "error">("loading"); const [attempt, setAttempt] = useState(0);
  useEffect(() => { const controller = new AbortController(); setData(undefined); setState("loading"); getRCSACycle(id, controller.signal).then((value) => { if (!controller.signal.aborted) { if (value.cycle.id !== id) { setState("error"); return; } setData(value); setState("ready"); } }).catch(() => { if (!controller.signal.aborted) setState("error"); }); return () => controller.abort(); }, [id, attempt]);
  if (state === "loading") return <p role="status">Loading assessment…</p>;
  if (state === "error" || !data) return <section><h1>Assessment unavailable</h1><button onClick={() => setAttempt((value) => value + 1)}>Try again</button><p><a href="#/risks/rcsa">Risk assessments</a></p></section>;
  const { cycle } = data;
  return <section className="v-risk-detail"><a className="v-back" href="#/risks/rcsa">← Risk assessments</a><header className="v-page-heading"><div><p className="v-eyebrow">{cycle.code} · Version {cycle.version}</p><h1>{cycle.name}</h1></div><span className="v-status">{label(cycle.status)}</span></header><div className="v-panel"><h2>Assessment details</h2><dl><div><dt>Phase</dt><dd>{data.phase?.label || "Unknown"}</dd></div><div><dt>Owner</dt><dd>{data.first_line_owner_display_name || "Name unavailable"}</dd></div></dl></div><div className="v-panel"><h2>Risks</h2>{!data.complete && <p>Assessment details may be incomplete.</p>}{data.risks?.length ? <ul className="v-attention">{data.risks.map((risk) => <li key={risk.risk_id}><div><strong>{risk.name}</strong><small>{risk.code} · Version {risk.risk_version}</small></div></li>)}</ul> : data.complete ? <p>No risks recorded.</p> : null}</div><div className="v-panel"><h2>Controls</h2>{data.controls?.length ? <ul className="v-attention">{data.controls.map((control) => <li key={control.risk_control_link_id}><div><strong>{control.implementation_name}</strong><small>{control.definition_name} · {control.implementation_status.toLowerCase().replaceAll("_", " ")}</small></div></li>)}</ul> : data.complete ? <p>No controls recorded.</p> : <p>Control details may be incomplete.</p>}</div></section>;
}
function label(value: string) { return ({ DRAFT: "Draft", ASSESSMENT_OPEN: "Assessment open", AWAITING_CHALLENGE: "Awaiting challenge", COMPLETED: "Completed", CANCELLED: "Cancelled" } as Record<string, string>)[value] ?? "Status unknown"; }
