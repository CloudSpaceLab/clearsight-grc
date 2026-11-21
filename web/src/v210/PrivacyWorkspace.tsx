import { useEffect, useRef, useState, type FormEvent } from "react";
import { fetchProcessingActivity, listProcessingActivities } from "../ropaApi";
import type { ProcessingActivity, ProcessingActivityResponse } from "../ropaTypes";
import "./privacy-workspace.css";

export function PrivacyWorkspace({ id }: { id?: string }) { return id ? <ActivityDetail key={id} id={id}/> : <ActivityRegister/>; }
function ActivityRegister() {
  const [draft, setDraft] = useState(""); const [search, setSearch] = useState(""); const [attempt, setAttempt] = useState(0);
  const [items, setItems] = useState<ProcessingActivity[]>([]); const [cursor, setCursor] = useState<string>(); const [hasMore, setHasMore] = useState(false);
  const [state, setState] = useState<"loading" | "ready" | "error">("loading"); const [busy, setBusy] = useState(false); const [moreError, setMoreError] = useState(false); const generation = useRef(0);
  useEffect(() => { const controller = new AbortController(); const token = ++generation.current; setState("loading"); setItems([]); setCursor(undefined); setHasMore(false); setBusy(false); setMoreError(false);
    listProcessingActivities({ search: search || undefined }, controller.signal).then((page) => { if (generation.current === token) { setItems(page.rows); setCursor(page.next_cursor); setHasMore(page.has_more); setState("ready"); } }).catch(() => { if (generation.current === token && !controller.signal.aborted) setState("error"); });
    return () => { controller.abort(); generation.current++; };
  }, [search, attempt]);
  async function more() { if (!cursor || !hasMore || busy) return; const token = generation.current; setBusy(true); setMoreError(false);
    try { const page = await listProcessingActivities({ search: search || undefined, cursor }); if (generation.current === token) { if (page.next_cursor === cursor) { setMoreError(true); return; } setItems((current) => [...current, ...page.rows]); setCursor(page.next_cursor); setHasMore(page.has_more); } }
    catch { if (generation.current === token) setMoreError(true); } finally { if (generation.current === token) setBusy(false); }
  }
  function submit(event: FormEvent) { event.preventDefault(); setSearch(draft.trim()); }
  return <section><header className="v-page-heading"><div><p className="v-eyebrow">Privacy</p><h1>Processing activities</h1></div></header><form className="v-program-search" onSubmit={submit}><label htmlFor="v-privacy-search">Find a processing activity</label><div><input id="v-privacy-search" maxLength={200} value={draft} onChange={(event) => setDraft(event.target.value)}/><button>Search</button></div></form>
    {state === "loading" && <p role="status">Loading processing activities…</p>}{state === "error" && <div className="v-panel"><p role="alert">Processing activities unavailable.</p><button onClick={() => setAttempt((value) => value + 1)}>Try again</button></div>}
    {state === "ready" && <div className="v-panel v-program-table">{items.length === 0 ? <p>No processing activities match this search.</p> : <table><thead><tr><th>Activity</th><th>Owner</th><th>Next review</th><th>Status</th></tr></thead><tbody>{items.map((item) => <tr key={item.id}><td><a href={`#/privacy/${encodeURIComponent(item.id)}`}>{item.name}</a><small>{item.code}</small></td><td><span className="v-mobile-label">Owner</span>{item.owner_display_name || (item.owner_principal_id ? "Name unavailable" : "Not recorded")}</td><td><span className="v-mobile-label">Next review</span>{item.next_review_date ? formatDate(item.next_review_date) : "Not recorded"}</td><td><span className="v-mobile-label">Status</span>{label(item.status)}</td></tr>)}</tbody></table>}{moreError && <p role="alert">More activities unavailable.</p>}{hasMore && cursor && <button disabled={busy} onClick={() => void more()}>{busy ? "Loading…" : "Load more"}</button>}{hasMore && !cursor && <p role="alert">More activities unavailable. Reload the register.</p>}</div>}
  </section>;
}
function ActivityDetail({ id }: { id: string }) {
  const [data, setData] = useState<ProcessingActivityResponse>(); const [state, setState] = useState<"loading" | "ready" | "error">("loading"); const [attempt, setAttempt] = useState(0);
  useEffect(() => { const controller = new AbortController(); setData(undefined); setState("loading"); fetchProcessingActivity(id, controller.signal).then((value) => { if (!controller.signal.aborted) { if (value.activity.id !== id) { setState("error"); return; } setData(value); setState("ready"); } }).catch(() => { if (!controller.signal.aborted) setState("error"); }); return () => controller.abort(); }, [id, attempt]);
  if (state === "loading") return <p role="status">Loading processing activity…</p>;
  if (state === "error" || !data) return <section><h1>Processing activity unavailable</h1><button onClick={() => setAttempt((value) => value + 1)}>Try again</button><p><a href="#/privacy">Processing activities</a></p></section>;
  const activity = data.activity;
  const details = [["Purpose", activity.purpose], ["Lawful basis", activity.lawful_basis], ["Controller", activity.controller], ["Processor", activity.processor], ["Data subjects", activity.data_subject_categories], ["Personal data", activity.personal_data_categories], ["Retention", activity.retention_period], ["Owner", activity.owner_display_name || (activity.owner_principal_id ? "Name unavailable" : "Not recorded")]];
  return <section className="v-privacy-detail"><a className="v-back" href="#/privacy">← Processing activities</a><header className="v-page-heading"><div><p className="v-eyebrow">{activity.code} · Version {activity.version}</p><h1>{activity.name}</h1></div><span className="v-status">{data.state_label || label(activity.status)}</span></header><div className="v-panel"><h2>Processing details</h2><dl>{details.map(([title, value]) => <div key={title}><dt>{title}</dt><dd>{value || "Not recorded"}</dd></div>)}</dl></div>{data.closure_blockers?.length ? <div className="v-panel"><h2>Closure requirements</h2><ul>{data.closure_blockers.map((item, index) => <li key={`${index}:${item}`}>{item}</li>)}</ul></div> : null}</section>;
}
function label(value: string) { return value.toLowerCase().replaceAll("_", " "); }
function formatDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "date unavailable" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeZone: "UTC" }).format(date); }
