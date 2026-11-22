import { useEffect, useRef, useState } from "react";
import { apiErrorKind } from "../http";
import { loadFormTemplatePage } from "../formsApi";
import type { FormLibraryFacets, FormLibraryItem, FormTemplateQuery } from "../formsTypes";
import { FormFilters, formStatusCounts } from "./FormFilters";
import "./forms-register.css";

export function FormsRegister() {
  const [query, setQuery] = useState<FormTemplateQuery>({limit:20});
  const [facets, setFacets] = useState<FormLibraryFacets>();
  const [items, setItems] = useState<FormLibraryItem[]>([]);
  const [cursor, setCursor] = useState<string>();
  const [status, setStatus] = useState<"loading" | "live" | "error">("loading");
  const [busy, setBusy] = useState(false);
  const [moreError, setMoreError] = useState(false);
  const [retry, setRetry] = useState(0);
  const generation = useRef(0);
  const moreLock = useRef(false);
  const seenCursors = useRef(new Set<string>());
  useEffect(() => {
    let active = true; generation.current++; moreLock.current = false; seenCursors.current.clear(); setStatus("loading"); setItems([]); setCursor(undefined); setFacets(undefined); setBusy(false); setMoreError(false);
    loadFormTemplatePage(query, undefined, {statusFacets:true}).then((page) => { if (active) { setItems(page.items); setCursor(page.next_cursor); if (page.next_cursor) seenCursors.current.add(page.next_cursor); setFacets(page.facets); setStatus("live"); } }).catch(() => { if (active) setStatus("error"); });
    return () => { active = false; };
  }, [query, retry]);
  async function more() {
    if (!cursor || moreLock.current) return;
    moreLock.current = true; const request = generation.current; setBusy(true); setMoreError(false);
    try { const page = await loadFormTemplatePage({ ...query, cursor }, undefined, {statusFacets:true}); if (generation.current === request) { if (page.next_cursor && seenCursors.current.has(page.next_cursor)) { setMoreError(true); return; } if (page.next_cursor) seenCursors.current.add(page.next_cursor); setItems((previous) => [...previous, ...page.items]); setCursor(page.next_cursor); } }
    catch (error) {
      if (generation.current === request) {
        if (["unauthorized", "forbidden"].includes(apiErrorKind(error))) {
          setItems([]); setCursor(undefined); setFacets(undefined); setStatus("error");
        } else setMoreError(true);
      }
    } finally { if (generation.current === request) { moreLock.current = false; setBusy(false); } }
  }
  const counts = formStatusCounts(facets);
  const draftCount = counts?.DRAFT ?? 0;
  const overviewTitle = query.search || query.tag ? "Matching forms by status" : "Forms by status";
  const summaries = [{label:"Active forms", value:counts && `${counts.ACTIVE ?? 0} active`}, {label:"Draft forms", value:counts && `${draftCount} ${draftCount === 1 ? "draft" : "drafts"}`}, {label:"Awaiting approval", value:counts && `${counts.PENDING_APPROVAL ?? 0} awaiting approval`}];
  return <section className="v-forms-register"><header className="v-page-heading"><div><p className="v-eyebrow">Assessment workspace</p><h1>Forms & assessments</h1></div><a className="v-primary" href="#/forms/new">Create form</a></header>
    <nav className="v-local-nav" aria-label="Forms views"><a href="#/forms" aria-current="page">Forms</a><a href="#/forms/responses">Responses</a><a href="#/documents">Documents</a></nav>
    <section className="v-forms-overview" aria-label={overviewTitle}><h2>{overviewTitle}</h2><div className="v-forms-overview-grid">{summaries.map((summary) => <article className="v-panel v-form-summary" key={summary.label}><span className="v-form-summary-icon" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6"><path d="M6 3h12v18H6zM9 8h6M9 12h6M9 16h4"/></svg></span><h3>{summary.label}</h3><p>{summary.value ?? "—"}</p></article>)}</div></section>
    <div className="v-forms-list-title"><h2>Form library</h2>{status === "live" && <span>{items.length} loaded{cursor ? " · More available" : ""}</span>}</div>
    <FormFilters query={query} facets={facets} onChange={setQuery}/>
    {status === "loading" && <p role="status">Loading forms…</p>}
    {status === "error" && <div className="v-panel"><p role="alert">Forms unavailable.</p><button onClick={() => setRetry(retry + 1)}>Try again</button></div>}
    {status === "live" && <div className="v-panel v-program-table v-forms-register-table">{items.length === 0 ? <>{query.search || query.tag || query.status ? <><p>No forms match these filters.</p><button type="button" onClick={() => setQuery({limit:20})}>Clear filters</button></> : <p>No forms found.</p>}</> : <table><thead><tr><th>Form</th><th>Questions</th><th>Status</th><th>Version</th><th><span className="v-forms-sr">Action</span></th></tr></thead><tbody>{items.map(({ template }) => <tr key={template.id}><td><a href={`#/forms/${encodeURIComponent(template.id)}`}>{template.name}</a>{template.purpose && <small>{template.purpose}</small>}</td><td><span className="v-mobile-label">Questions</span>{template.fields?.length ?? "Unknown"}</td><td><span className="v-mobile-label">Status</span><span className={`v-form-status ${template.status === "ACTIVE" ? "is-active" : template.status === "REJECTED" ? "is-rejected" : ""}`}>{template.status.toLowerCase().replaceAll("_", " ")}</span></td><td><span className="v-mobile-label">Version</span>{template.version}</td><td><a className="v-form-open" href={`#/forms/${encodeURIComponent(template.id)}`} aria-label={`Open ${template.name}`}>Open</a></td></tr>)}</tbody></table>}{moreError && <p role="alert">More forms unavailable.</p>}{cursor && <button disabled={busy} onClick={() => void more()}>{busy ? "Loading…" : "Load more"}</button>}</div>}
  </section>;
}
