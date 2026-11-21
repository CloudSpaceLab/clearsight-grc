import { useEffect, useRef, useState } from "react";
import { loadFrameworkClauses, loadFrameworkEditionPage, type FrameworkClause, type FrameworkEditionOption } from "../programOperationsApi";
import { FrameworkAdoption } from "./FrameworkAdoption";
import "./frameworks.css";

export function Frameworks() {
  const [jurisdiction, setJurisdiction] = useState("GLOBAL");
  const [jurisdictionDraft, setJurisdictionDraft] = useState("GLOBAL");
  const [search, setSearch] = useState("");
  const [items, setItems] = useState<FrameworkEditionOption[]>([]);
  const [selected, setSelected] = useState<string>();
  const [adopting, setAdopting] = useState(false);
  const [adoptionPending, setAdoptionPending] = useState(false);
  const [cursor, setCursor] = useState("");
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [busy, setBusy] = useState(false);
  const [moreError, setMoreError] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const generation = useRef(0);
  useEffect(() => {
    const token = ++generation.current;
    setState("loading"); setItems([]); setSelected(undefined); setAdopting(false); setCursor(""); setMoreError(false); setBusy(false);
    loadFrameworkEditionPage(jurisdiction, "").then((page) => {
      if (token === generation.current) { setItems(page.items); setCursor(page.next_after_id); setState("ready"); }
    }).catch(() => { if (token === generation.current) setState("error"); });
    return () => { generation.current++; };
  }, [jurisdiction, attempt]);
  async function more() {
    if (!cursor || busy) return;
    const token = generation.current; setBusy(true); setMoreError(false);
    try {
      const page = await loadFrameworkEditionPage(jurisdiction, cursor);
      if (token === generation.current) { setItems((current) => [...current, ...page.items]); setCursor(page.next_after_id); }
    } catch { if (token === generation.current) setMoreError(true); }
    finally { if (token === generation.current) setBusy(false); }
  }
  const filtered = items.filter((item) => [item.title, item.framework_code, item.edition_code, item.publisher_code].some((value) => value.toLowerCase().includes(search.trim().toLowerCase())));
  const detail = items.find((item) => item.id === selected);
  const source = detail ? safeSource(detail.source_url) : undefined;
  return <section className="v-frameworks">
    <header className="v-page-heading"><div><p className="v-eyebrow">Frameworks</p><h1>Framework library</h1></div></header>
    <form className="v-framework-filters" onSubmit={(event) => { event.preventDefault(); const next = jurisdictionDraft.trim().toUpperCase(); if (next && /^[A-Z0-9-]{1,30}$/.test(next)) setJurisdiction(next); }}>
      <label>Jurisdiction<input disabled={adoptionPending} value={jurisdictionDraft} onChange={(event) => setJurisdictionDraft(event.target.value)} maxLength={30} required pattern="[A-Za-z0-9-]+"/></label><button type="submit" disabled={adoptionPending}>Apply jurisdiction</button>
      <label>Search loaded editions<input disabled={adoptionPending} type="search" value={search} onChange={(event) => setSearch(event.target.value)}/></label>
    </form>
    {state === "loading" && <p role="status">Loading framework editions…</p>}
    {state === "error" && <div className="v-panel"><p role="alert">Framework editions unavailable.</p><button onClick={() => setAttempt((value) => value + 1)}>Try again</button></div>}
    {state === "ready" && <div className="v-framework-layout"><div>
      {items.length === 0 ? <div className="v-panel"><p>No published editions found for {jurisdiction}.</p></div> : filtered.length === 0 ? <div className="v-panel"><p>No loaded editions match this search.</p></div> : <ul className="v-framework-list">{filtered.map((item) => <li key={item.id}><button type="button" disabled={adoptionPending} aria-pressed={selected === item.id} onClick={() => { setSelected(item.id); setAdopting(false); }}><strong>{item.title}</strong><span>{item.publisher_code} · {item.edition_code}</span><small>{item.jurisdiction}</small></button></li>)}</ul>}
      {moreError && <p role="alert">More editions unavailable. Try again.</p>}
      {cursor && <button type="button" className="v-framework-more" disabled={busy} onClick={() => void more()}>{busy ? "Loading…" : "Load more editions"}</button>}
    </div><aside className="v-panel v-framework-detail" aria-label="Framework edition details"><h2>{detail?.title ?? "Edition details"}</h2>{detail ? <>
      <dl><div><dt>Publisher</dt><dd>{detail.publisher_code}</dd></div><div><dt>Edition</dt><dd>{detail.edition_code}</dd></div><div><dt>Jurisdiction</dt><dd>{detail.jurisdiction}</dd></div><div><dt>Published</dt><dd>{detail.source_published_on || "Unknown"}</dd></div><div><dt>Source rights</dt><dd>{rights(detail.license_basis)}</dd></div>{detail.effective_on && <div><dt>Effective</dt><dd>{detail.effective_on}</dd></div>}{detail.license_reference && <div><dt>Rights reference</dt><dd>{detail.license_reference}</dd></div>}</dl>
      {source ? <a href={source} target="_blank" rel="noopener noreferrer">View published source</a> : <p>Source unavailable.</p>}
      <FrameworkClauses key={detail.id} editionID={detail.id}/>
      {source && ["CITATIONS_ONLY", "SUMMARIES_PERMITTED"].includes(detail.license_basis) && !adopting && <button className="v-primary v-framework-propose" type="button" onClick={() => setAdopting(true)}>Select program</button>}
      {adopting && <FrameworkAdoption key={detail.id} edition={detail} onCancel={() => setAdopting(false)} onPendingChange={setAdoptionPending}/>}
    </> : <p>Select an edition to review its source.</p>}</aside></div>}
  </section>;
}

function FrameworkClauses({ editionID }: { editionID: string }) {
  const [items, setItems] = useState<FrameworkClause[]>([]);
  const [cursor, setCursor] = useState("");
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [attempt, setAttempt] = useState(0);
  const [busy, setBusy] = useState(false);
  const [moreError, setMoreError] = useState(false);
  const generation = useRef(0);
  useEffect(() => {
    const token = ++generation.current; setState("loading"); setItems([]); setCursor(""); setMoreError(false);
    loadFrameworkClauses(editionID, "").then((page) => {
      if (token !== generation.current) return;
      if (page.edition_id !== editionID || !Array.isArray(page.items)) { setState("error"); return; }
      setItems(page.items); setCursor(page.next_after_code); setState("ready");
    }).catch(() => { if (token === generation.current) setState("error"); });
    return () => { generation.current++; };
  }, [editionID, attempt]);
  async function more() {
    if (!cursor || busy) return;
    const token = generation.current; setBusy(true); setMoreError(false);
    try {
      const page = await loadFrameworkClauses(editionID, cursor);
      if (token !== generation.current) return;
      if (page.edition_id !== editionID || !Array.isArray(page.items)) throw new Error("mismatch");
      setItems((current) => [...current, ...page.items]); setCursor(page.next_after_code);
    } catch { if (token === generation.current) setMoreError(true); }
    finally { if (token === generation.current) setBusy(false); }
  }
  return <section className="v-framework-clauses"><h3>Requirements</h3>
    {state === "loading" && <p role="status">Loading requirements…</p>}
    {state === "error" && <><p role="alert">Requirements unavailable.</p><button type="button" onClick={() => setAttempt((value) => value + 1)}>Try again</button></>}
    {state === "ready" && <>{items.length === 0 ? <p>No requirements recorded for this edition.</p> : <ul>{items.map((item) => <li key={item.code}><strong>{item.title}</strong><small>{item.code} · {item.source_anchor}</small>{item.summary && <p>{item.summary}</p>}</li>)}</ul>}{moreError && <p role="alert">More requirements unavailable. Try again.</p>}{cursor && <button type="button" disabled={busy} onClick={() => void more()}>{busy ? "Loading…" : "Load more requirements"}</button>}</>}
  </section>;
}

function safeSource(value: string): string | undefined { try { const url = new URL(value); return url.protocol === "https:" && url.hostname && !url.username && !url.password ? url.href : undefined; } catch { return undefined; } }
function rights(value: string): string { return value === "CITATIONS_ONLY" ? "Citations only" : value === "SUMMARIES_PERMITTED" ? "Summaries permitted" : "Not verified"; }
