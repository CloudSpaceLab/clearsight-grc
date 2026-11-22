import { useEffect, useRef, useState } from "react";
import { loadProgramSummaries, type RuntimeContext } from "../api";
import { apiErrorKind } from "../http";
import type { ProgramSummary } from "../summaryTypes";
import { ProgramCreateLink } from "./ProgramCreator";
import { programCount, programStatus } from "./programProjection";
import { programFilterQuery, programStates, readProgramFilters } from "./programFilters";
import { ShellIcon } from "./ShellIcon";
import "./program-register.css";
import "./program-register-v215.css";

function statusClass(item: ProgramSummary) {
  const status = programStatus(item);
  if (status === "Unknown" || status === "Out of date") return "is-unknown";
  if (["GAP_IDENTIFIED", "OVERDUE"].includes(item.overall_state)) return "is-critical";
  if (["AT_RISK", "EVIDENCE_INSUFFICIENT", "IMPLEMENTATION_PENDING", "UNDER_REVIEW"].includes(item.overall_state)) return "is-attention";
  return item.overall_state === "CURRENT" ? "is-current" : "is-unknown";
}

export function ProgramsRegister({ context }: { context: RuntimeContext }) {
  const [initial] = useState(() => readProgramFilters(window.location.hash));
  const [draft, setDraft] = useState(initial.search); const [search, setSearch] = useState(initial.search);
  const [draftState, setDraftState] = useState(initial.overallState); const [overallState, setOverallState] = useState(initial.overallState);
  const [items, setItems] = useState<ProgramSummary[]>([]); const [cursor, setCursor] = useState<string>();
  const [status, setStatus] = useState<"loading" | "live" | "error">("loading");
  const [loadingMore, setLoadingMore] = useState(false); const [moreError, setMoreError] = useState(false); const [retry, setRetry] = useState(0);
  const generation = useRef(0); const moreLock = useRef(false);
  useEffect(() => {
    let active = true; generation.current++; moreLock.current = false; setStatus("loading"); setItems([]); setCursor(undefined); setLoadingMore(false); setMoreError(false);
    loadProgramSummaries({ q: search, ...(overallState ? { overallState } : {}), limit: 20 }).then((page) => { if (active) { setItems(page.items); setCursor(page.next_cursor); setStatus("live"); } }).catch(() => { if (active) setStatus("error"); });
    return () => { active = false; };
  }, [search, overallState, retry]);
  useEffect(() => {
    const restore = () => { if (!window.location.hash.match(/^#\/?programs(?:\?|$)/)) return; const filters = readProgramFilters(window.location.hash); setDraft(filters.search); setSearch(filters.search); setDraftState(filters.overallState); setOverallState(filters.overallState); };
    window.addEventListener("popstate", restore); window.addEventListener("hashchange", restore);
    return () => { window.removeEventListener("popstate", restore); window.removeEventListener("hashchange", restore); };
  }, []);
  async function more() {
    if (!cursor || moreLock.current) return;
    moreLock.current = true; const request = generation.current; setLoadingMore(true); setMoreError(false);
    try { const next = await loadProgramSummaries({ q: search, ...(overallState ? { overallState } : {}), limit: 20, cursor }); if (generation.current === request) { if (next.next_cursor === cursor) { setMoreError(true); return; } setItems((current) => { const ids = new Set(current.map((item) => item.program.id)); return [...current, ...next.items.filter((item) => { if (ids.has(item.program.id)) return false; ids.add(item.program.id); return true; })]; }); setCursor(next.next_cursor); } }
    catch (error) { if (generation.current === request) { if (["unauthorized", "forbidden"].includes(apiErrorKind(error))) { setItems([]); setCursor(undefined); setStatus("error"); } else setMoreError(true); } }
    finally { if (generation.current === request) { moreLock.current = false; setLoadingMore(false); } }
  }
  const issueCountsValid = items.every((item) => Number.isSafeInteger(item.open_matter_count) && item.open_matter_count >= 0);
  const issuePrograms = issueCountsValid ? String(items.filter((item) => item.open_matter_count > 0).length) : "Unknown";
  const stalePrograms = items.filter((item) => programStatus(item) === "Out of date").length;
  const unknownPrograms = items.filter((item) => programStatus(item) === "Unknown").length;
  return <section className="v-programs"><header className="v-page-heading v-program-register-heading"><div><p className="v-eyebrow">Programs</p><h1>Compliance programs</h1></div><ProgramCreateLink context={context}/></header>
    {status === "live" && <section className="v-program-summary" aria-label="Loaded Program summary"><article className="v-panel"><div><span>Programs loaded</span><ShellIcon name="programs"/></div><strong>{items.length}</strong><small>{cursor ? "More programs available" : "Results complete"}</small></article><article className="v-panel"><div><span>With open issues</span><ShellIcon name="work"/></div><strong>{issuePrograms}</strong><small>Loaded programs</small></article><article className="v-panel"><div><span>Confirmed out of date</span><ShellIcon name="risks"/></div><strong>{stalePrograms}</strong><small>Loaded programs</small></article><article className="v-panel"><div><span>Assessment unknown</span><ShellIcon name="forms"/></div><strong>{unknownPrograms}</strong><small>Loaded programs</small></article></section>}
    {status === "live" && <form className="v-program-search" onSubmit={(event) => { event.preventDefault(); const nextSearch = draft.trim(); setSearch(nextSearch); setOverallState(draftState); window.history.replaceState(window.history.state, "", `#/programs${programFilterQuery(nextSearch, draftState)}`); }}><div className="v-program-filter-fields"><div className="v-program-filter-field"><label htmlFor="v-program-search">Search programs</label><input id="v-program-search" value={draft} onChange={(event) => setDraft(event.target.value)} placeholder="Program name"/></div><div className="v-program-filter-field"><label htmlFor="v-program-status">Assessed status</label><select id="v-program-status" value={draftState} onChange={(event) => setDraftState(event.target.value)}><option value="">All statuses</option>{programStates.map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></div><button type="submit">Apply filters</button></div><p className="v-program-filter-note">Results may include out-of-date assessments.</p></form>}
    {status === "loading" && <p role="status">Loading programs…</p>}
    {status === "error" && <div className="v-panel"><p role="alert">Programs unavailable.</p><button onClick={() => setRetry(retry + 1)}>Try again</button>{(search || overallState) && <button onClick={() => { setDraft(""); setDraftState(""); setSearch(""); setOverallState(""); window.history.replaceState(window.history.state, "", "#/programs"); }}>Clear filters</button>}</div>}
    {status === "live" && <><p className="v-program-result-count">{items.length} loaded{cursor ? " · More available" : ""}</p><div className="v-panel v-program-table v-program-register-table">{items.length === 0 ? <p>No programs match these filters.</p> : <table><thead><tr><th>Program</th><th>Function</th><th>Requirements</th><th title="Evidence checks">Assurance checks</th><th>Visible open issues</th><th>Status</th><th><span className="v-visually-hidden">Action</span></th></tr></thead><tbody>{items.map((item) => <tr key={item.program.id}><td><a href={`#/programs/${encodeURIComponent(item.program.id)}${programFilterQuery(search, overallState)}`}>{item.program.name}</a><small>{item.program.code}</small></td><td><span className="v-mobile-label">Function</span>{item.program.owning_function || "Not recorded"}</td><td><span className="v-mobile-label">Requirements</span>{programCount(item.requirement_count)}</td><td title="Evidence checks"><span className="v-mobile-label">Assurance checks</span>{programCount(item.evidence_check_count)}</td><td><span className="v-mobile-label">Visible open issues</span>{Number.isSafeInteger(item.open_matter_count) && item.open_matter_count > 0 ? <a className="v-program-issue-link" href={`#/programs/${encodeURIComponent(item.program.id)}/issues${programFilterQuery(search, overallState)}`} aria-label={`View ${item.open_matter_count} visible open issues for ${item.program.name}`}>{item.open_matter_count}</a> : programCount(item.open_matter_count)}</td><td><span className="v-mobile-label">Status</span><span className={`v-program-state ${statusClass(item)}`}>{programStatus(item)}</span>{item.state_generated_at && Number.isFinite(Date.parse(item.state_generated_at)) && <small>Status as of <time dateTime={item.state_generated_at}>{new Date(item.state_generated_at).toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric", timeZone: "UTC" })}</time></small>}</td><td><a className="v-program-open" href={`#/programs/${encodeURIComponent(item.program.id)}${programFilterQuery(search, overallState)}`} aria-label={`Open ${item.program.name}`}>Open <span aria-hidden="true">→</span></a></td></tr>)}</tbody></table>}{moreError && <p role="alert">More programs unavailable.</p>}{cursor && <button disabled={loadingMore} onClick={() => void more()}>{loadingMore ? "Loading…" : "Load more"}</button>}</div></>}
  </section>;
}
