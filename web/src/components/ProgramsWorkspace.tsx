import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { loadProgram, loadProgramSummaries } from "../api";
import type { ProgramSummary } from "../summaryTypes";
import type { ProgramAggregate, ProgramState } from "../types";
import { EmptyState } from "./EmptyState";

type LoadState = "loading" | "live" | "unavailable";
type Props = { targetID?: string; openFirst?: boolean };

function ProgramIcon() {
  return <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="M6 3h9l3 3v15H6z"/><path d="M15 3v4h4M9 11h6M9 15h6M9 19h4"/></svg>;
}

function requirementStatusLabel(value: string) {
  switch (value) {
    case "APPROVED": return "Approved";
    case "PROPOSED": return "Awaiting review";
    case "SUPERSEDED": return "Replaced";
    case "RETIRED": return "Ended";
    default: return value.replaceAll("_", " ").toLowerCase();
  }
}

function stateClass(value?: ProgramState) {
  switch (value) {
    case "CURRENT": return "status-good";
    case "GAP_IDENTIFIED": case "OVERDUE": return "status-critical";
    case "AT_RISK": case "EVIDENCE_INSUFFICIENT": case "IMPLEMENTATION_PENDING": case "UNDER_REVIEW": return "status-warning";
    default: return "status-neutral";
  }
}

export function ProgramsWorkspace({ targetID, openFirst = false }: Props) {
  const [items, setItems] = useState<ProgramSummary[]>([]);
  const [state, setState] = useState<LoadState>("loading");
  const [nextCursor, setNextCursor] = useState("");
  const [searchDraft, setSearchDraft] = useState("");
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");
  const [openID, setOpenID] = useState<string | null>(null);
  const [details, setDetails] = useState<Record<string, ProgramAggregate>>({});
  const [detailState, setDetailState] = useState<Record<string, LoadState>>({});
  const [loadingMore, setLoadingMore] = useState(false);
  const requestID = useRef(0);
  const handledTarget = useRef("");

  const load = useCallback(async (reset: boolean, cursor = "") => {
    const currentRequest = ++requestID.current;
    if (reset) setState("loading"); else setLoadingMore(true);
    try {
      const page = await loadProgramSummaries({ q: search, status, cursor, limit: 20 });
      if (currentRequest !== requestID.current) return;
      setItems((current) => reset ? page.items : [...current, ...page.items]);
      setNextCursor(page.next_cursor ?? "");
      setState("live");
    } catch {
      if (currentRequest !== requestID.current) return;
      if (reset) setState("unavailable");
    } finally {
      if (currentRequest === requestID.current) setLoadingMore(false);
    }
  }, [search, status]);

  useEffect(() => { void load(true); }, [load]);

  const summary = useMemo(() => ({
    current: items.filter((item) => item.overall_state === "CURRENT").length,
    attention: items.filter((item) => ["AT_RISK", "GAP_IDENTIFIED", "EVIDENCE_INSUFFICIENT", "IMPLEMENTATION_PENDING", "OVERDUE"].includes(item.overall_state)).length,
    setup: items.filter((item) => item.program.status === "DRAFT" || ["UNKNOWN", "UNDER_REVIEW"].includes(item.overall_state)).length,
  }), [items]);

  function submitSearch(event: FormEvent) {
    event.preventDefault();
    setSearch(searchDraft.trim());
  }

  function clearFilters() {
    setSearchDraft("");
    setSearch("");
    setStatus("");
  }

  async function fetchDetail(id: string) {
    if (detailState[id] === "loading") return;
    setDetailState((current) => ({ ...current, [id]: "loading" }));
    try {
      const value = await loadProgram(id);
      setDetails((current) => ({ ...current, [id]: value }));
      setDetailState((current) => ({ ...current, [id]: "live" }));
    } catch {
      setDetailState((current) => ({ ...current, [id]: "unavailable" }));
    }
  }

  async function toggleDetail(id: string) {
    if (openID === id) {
      setOpenID(null);
      return;
    }
    setOpenID(id);
    if (!details[id]) await fetchDetail(id);
  }

  useEffect(() => {
    if (state !== "live" || !items.length) return;
    const id = targetID ?? (openFirst ? items[0]?.program.id : undefined);
    if (!id || handledTarget.current === id) return;
    handledTarget.current = id;
    setOpenID(id);
    void fetchDetail(id).finally(() => window.setTimeout(() => document.getElementById(`program-${id}`)?.scrollIntoView({ behavior: "smooth", block: "center" }), 80));
  }, [state, items, targetID, openFirst]);

  if (state === "loading") return <section id="programs-workspace" className="workspace-loading" aria-live="polite" aria-busy="true">Loading programs…</section>;
  if (state === "unavailable") return <div id="programs-workspace"><EmptyState label="Programs" title="Programs could not be loaded" description="The service is unavailable. No program totals are shown." action="Try again" onAction={() => void load(true)}/></div>;

  const briefTitle = summary.attention > 0
    ? `${summary.attention} loaded program${summary.attention === 1 ? " requires" : "s require"} follow-up`
    : summary.setup > 0
      ? `${summary.setup} loaded program${summary.setup === 1 ? " is" : "s are"} still being set up`
      : items.length ? "No recorded gaps or overdue work in the loaded programs" : "No programs in this scope";

  return <div id="programs-workspace">
    <section className="workspace-brief">
      <div><span className="eyebrow">Ongoing compliance</span><h2>{briefTitle}</h2><p>Open a Program only when you need the status reason or deeper source context.</p></div>
      <div className="workspace-brief-facts" aria-label="Loaded Program status"><span><strong>{summary.attention}</strong> follow-up</span><span><strong>{summary.current}</strong> current</span><span><strong>{summary.setup}</strong> setup or review</span></div>
    </section>
    <form className="workspace-toolbar" role="search" onSubmit={submitSearch}>
      <label><span>Search programs</span><input value={searchDraft} onChange={(event) => setSearchDraft(event.target.value)} placeholder="Name, code, function or jurisdiction"/></label>
      <label><span>Status</span><select value={status} onChange={(event) => setStatus(event.target.value)}><option value="">All statuses</option><option value="ACTIVE">Active</option><option value="PAUSED">Paused</option><option value="DRAFT">Setup in progress</option><option value="RETIRED">Ended</option></select></label>
      <button className="secondary-button" type="submit">Search</button>
      {(search || status) && <button className="text-button" type="button" onClick={clearFilters}>Clear filters</button>}
    </form>
    {!items.length ? <EmptyState label="Programs" title={search || status ? "No programs match these filters" : "No programs in this scope"} description={search || status ? "Change the search or status filter to see other programs." : "There are no ongoing compliance or control programs in the connected bank scope."} action={search || status ? "Clear filters" : undefined} onAction={clearFilters}/> : <section className="program-list">
      {items.map((summaryItem) => {
        const program = summaryItem.program;
        const isOpen = openID === program.id;
        const detail = details[program.id];
        const currentDetailState = detailState[program.id];
        return <article className={targetID === program.id ? "program-card targeted" : "program-card"} id={`program-${program.id}`} key={program.id}>
          <button className="program-card-main" type="button" aria-expanded={isOpen} aria-controls={`program-detail-${program.id}`} onClick={() => void toggleDetail(program.id)}>
            <span className="program-icon"><ProgramIcon/></span>
            <span className="program-primary"><span className="program-kicker">{program.code} · {program.owning_function}</span><strong>{program.name}</strong>{program.jurisdiction && <small>{program.jurisdiction}</small>}</span>
            <span className="program-counts"><span><b>{summaryItem.requirement_count}</b> requirements</span><span><b>{summaryItem.evidence_check_count}</b> evidence checks</span><span><b>{summaryItem.open_matter_count}</b> open issues</span></span>
            <span className={`program-state ${stateClass(summaryItem.overall_state)}`}><strong>{summaryItem.state_label || "Not assessed"}</strong><small>{summaryItem.reasons[0]?.summary ?? "Open for current status reasons"}</small></span>
            <span className="expand-indicator" aria-hidden="true">{isOpen ? "−" : "+"}</span>
          </button>
          {isOpen && <div className="program-detail progressive-detail" id={`program-detail-${program.id}`}>
            {currentDetailState === "loading" && <p aria-live="polite">Loading program details…</p>}
            {currentDetailState === "unavailable" && <div className="inline-error"><p>Program details could not be loaded.</p><button className="secondary-button" onClick={() => void fetchDetail(program.id)}>Try again</button></div>}
            {detail && <>
              <section className="status-reasons"><h3>Why this status</h3>{detail.current_state?.reasons?.length ? <ul>{detail.current_state.reasons.map((reason) => <li key={`${reason.code}-${reason.object_id ?? ""}`}>{reason.summary}</li>)}</ul> : <p>No status reasons are recorded for the latest assessment.</p>}</section>
              <details className="progressive-section"><summary><span>Requirements</span><strong>{detail.requirements.length}</strong></summary><div>{detail.requirements.length ? detail.requirements.map((requirement) => <div className="detail-row" key={requirement.id}><div><strong>{requirement.title}</strong><small>{requirement.statement}</small>{requirement.source_anchor && <small>Source: {requirement.source_anchor}</small>}</div><span>{requirementStatusLabel(requirement.status)}</span></div>) : <p>No approved requirements have been added.</p>}</div></details>
              <details className="progressive-section"><summary><span>Evidence expectations</span><strong>{detail.evidence_contracts.length}</strong></summary><div>{detail.evidence_contracts.length ? detail.evidence_contracts.map((contract) => <div className="detail-row" key={contract.id}><div><strong>{contract.name}</strong><small>{contract.claim}</small></div><span>Required coverage: {Math.round(contract.minimum_coverage * 100)}%</span></div>) : <p>No evidence checks have been defined.</p>}</div></details>
            </>}
          </div>}
        </article>;
      })}
    </section>}
    {nextCursor && <div className="load-more"><button className="secondary-button" disabled={loadingMore} onClick={() => void load(false, nextCursor)}>{loadingMore ? "Loading…" : "Load more programs"}</button></div>}
  </div>;
}
