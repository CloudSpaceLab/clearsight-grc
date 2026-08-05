import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { loadMatter, loadMatterSummaries } from "../api";
import type { MatterSummary } from "../summaryTypes";
import type { MatterAggregate } from "../types";
import { EmptyState } from "./EmptyState";
import { PremiumIllustration } from "./PremiumIllustration";

type LoadState = "loading" | "live" | "unavailable";

function MatterIcon({ type }: { type: string }) {
  const common = { width: 21, height: 21, viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: 1.7, strokeLinecap: "round" as const, strokeLinejoin: "round" as const, "aria-hidden": true };
  if (type === "REGULATORY_CHANGE" || type === "AUTHORITY_REQUEST") return <svg {...common}><path d="M4 9h16M6 9v11M10 9v11M14 9v11M18 9v11M3 20h18M12 3 3 8h18z"/></svg>;
  if (type === "EVIDENCE_CONTRADICTION") return <svg {...common}><path d="m8 5 8 14M16 5 8 19"/><circle cx="12" cy="12" r="9"/></svg>;
  return <svg {...common}><path d="M12 3 3 20h18z"/><path d="M12 9v5M12 18h.01"/></svg>;
}

function actionStatusLabel(value: string) {
  switch (value) {
    case "PLANNED": return "Not started";
    case "IN_PROGRESS": return "In progress";
    case "BLOCKED": return "Blocked";
    case "IMPLEMENTED": return "Work completed; outcome not confirmed";
    case "CANCELLED": return "Cancelled";
    default: return value.replaceAll("_", " ").toLowerCase();
  }
}

function priorityLabel(value: number) {
  if (value >= 5) return "Critical";
  if (value === 4) return "High";
  if (value === 3) return "Medium";
  if (value === 2) return "Normal";
  return "Low";
}

function humanizeKey(value: string) {
  return value.replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function formatFact(value: unknown) {
  if (value === null || value === undefined || value === "") return "Not recorded";
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

function latestResult(aggregate: MatterAggregate, contractID: string) {
  return aggregate.verification_results
    .filter((result) => result.contract_id === contractID)
    .sort((left, right) => Date.parse(right.observed_at) - Date.parse(left.observed_at))[0];
}

function resultLabel(value?: string) {
  switch (value) {
    case "PASS": return "Outcome confirmed";
    case "FAIL": return "Outcome not achieved";
    case "INCONCLUSIVE": return "More evidence needed";
    default: return "Not checked yet";
  }
}

export function MattersWorkspace() {
  const [items, setItems] = useState<MatterSummary[]>([]);
  const [state, setState] = useState<LoadState>("loading");
  const [nextCursor, setNextCursor] = useState("");
  const [searchDraft, setSearchDraft] = useState("");
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("OPEN");
  const [openID, setOpenID] = useState<string | null>(null);
  const [details, setDetails] = useState<Record<string, MatterAggregate>>({});
  const [detailState, setDetailState] = useState<Record<string, LoadState>>({});
  const [loadingMore, setLoadingMore] = useState(false);
  const requestID = useRef(0);

  const load = useCallback(async (reset: boolean, cursor = "") => {
    const currentRequest = ++requestID.current;
    if (reset) setState("loading"); else setLoadingMore(true);
    try {
      const page = await loadMatterSummaries({ q: search, status, cursor, limit: 20 });
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
    decisions: items.filter((item) => item.matter.status === "DECISION_REQUIRED").length,
    overdue: items.filter((item) => item.matter.due_at && Date.parse(item.matter.due_at) < Date.now() && !["CLOSED", "CANCELLED"].includes(item.matter.status)).length,
    checking: items.filter((item) => item.matter.status === "VERIFICATION").length,
  }), [items]);

  function submitSearch(event: FormEvent) {
    event.preventDefault();
    setSearch(searchDraft.trim());
  }

  function clearFilters() {
    setSearchDraft("");
    setSearch("");
    setStatus("OPEN");
  }

  async function fetchDetail(id: string) {
    if (detailState[id] === "loading") return;
    setDetailState((current) => ({ ...current, [id]: "loading" }));
    try {
      const value = await loadMatter(id);
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

  if (state === "loading") return <section className="workspace-loading">Loading issues and changes…</section>;
  if (state === "unavailable") return <EmptyState label="Issues and changes" title="Issues and changes could not be loaded" description="The service is unavailable. No current work totals are shown." action="Try again" onAction={() => void load(true)}/>;
  if (!items.length) return <EmptyState label="Issues and changes" title={search || status !== "OPEN" ? "No items match these filters" : "No open issues or changes"} description={search || status !== "OPEN" ? "Change the search or status filter to see other work." : "There are no recorded open changes, gaps, findings, exceptions or response items in the connected scope."} action={search || status !== "OPEN" ? "Clear filters" : undefined} onAction={clearFilters}/>;

  return <>
    <section className="matter-hero"><div><span className="eyebrow">Issues and changes</span><h2>{items.length} loaded item{items.length === 1 ? "" : "s"}</h2><p>Specific changes, gaps, findings, requests and exceptions that need a decision, action or outcome check.</p></div><PremiumIllustration variant="routing"/></section>
    <form className="workspace-toolbar" role="search" onSubmit={submitSearch}>
      <label><span>Search issues and changes</span><input value={searchDraft} onChange={(event) => setSearchDraft(event.target.value)} placeholder="Reference, title, summary or type"/></label>
      <label><span>Status</span><select value={status} onChange={(event) => setStatus(event.target.value)}><option value="OPEN">Open</option><option value="DECISION_REQUIRED">Decision needed</option><option value="ACTION_IN_PROGRESS">Work in progress</option><option value="VERIFICATION">Confirming outcome</option><option value="CLOSED">Closed</option><option value="">All statuses</option></select></label>
      <button className="secondary-button" type="submit">Search</button>
    </form>
    <section className="matter-summary" aria-label="Loaded issue and change summary"><div><span>Decision needed</span><strong>{summary.decisions}</strong><small>Waiting for an authorized decision</small></div><div><span>Overdue</span><strong>{summary.overdue}</strong><small>Past the recorded due date</small></div><div><span>Confirming outcome</span><strong>{summary.checking}</strong><small>Work is complete; the result still needs confirmation</small></div></section>
    <section className="matter-list">{items.map((summaryItem) => {
      const matter = summaryItem.matter;
      const isOpen = openID === matter.id;
      const detail = details[matter.id];
      const currentDetailState = detailState[matter.id];
      return <article className="matter-card" key={matter.id}>
        <button type="button" className="matter-card-main" aria-expanded={isOpen} onClick={() => void toggleDetail(matter.id)}>
          <span className="matter-icon"><MatterIcon type={matter.type}/></span>
          <span className="matter-primary"><span className="matter-kicker">{summaryItem.type_label} · {matter.reference}</span><strong>{matter.title}</strong><small>{matter.summary}</small></span>
          <span className="matter-meta"><span>{priorityLabel(matter.priority)} priority</span><span>{matter.due_at ? `${Date.parse(matter.due_at) < Date.now() ? "Overdue" : "Due"} ${new Date(matter.due_at).toLocaleDateString()}` : "No due date"}</span><span>{summaryItem.open_action_count} open action{summaryItem.open_action_count === 1 ? "" : "s"}</span></span>
          <span className={`matter-status status-${matter.status.toLowerCase().replaceAll("_", "-")}`}><strong>{summaryItem.status_label}</strong><small>{summaryItem.next_action}</small></span>
          <span className="expand-indicator" aria-hidden="true">{isOpen ? "−" : "+"}</span>
        </button>
        {isOpen && <div className="matter-detail">
          {currentDetailState === "loading" && <p>Loading issue details…</p>}
          {currentDetailState === "unavailable" && <div className="inline-error"><p>Issue details could not be loaded.</p><button className="secondary-button" onClick={() => void fetchDetail(matter.id)}>Try again</button></div>}
          {detail && <>
            <section><h3>What we know</h3>{Object.keys(detail.matter.known_facts ?? {}).length ? <dl>{Object.entries(detail.matter.known_facts).slice(0, 5).map(([key, value]) => <div key={key}><dt>{humanizeKey(key)}</dt><dd>{formatFact(value)}</dd></div>)}</dl> : <p>No facts have been recorded.</p>}{detail.matter.missing_facts?.length ? <div className="closure-note"><strong>Information still needed</strong><ul>{detail.matter.missing_facts.slice(0, 5).map((fact, index) => <li key={`${index}-${formatFact(fact)}`}>{formatFact(fact)}</li>)}</ul></div> : null}{detail.matter.contradictions?.length ? <p>{detail.matter.contradictions.length} conflicting item{detail.matter.contradictions.length === 1 ? " is" : "s are"} recorded.</p> : null}</section>
            <section><h3>Actions</h3>{detail.actions.length ? detail.actions.map((action) => <div className="detail-row" key={action.id}><strong>{action.title}</strong><span>{actionStatusLabel(action.status)}</span></div>) : <p>No actions have been recorded.</p>}</section>
            <section><h3>Outcome checks</h3>{detail.verification_contracts.length ? detail.verification_contracts.map((contract) => { const result = latestResult(detail, contract.id); return <div className="detail-row" key={contract.id}><strong>{contract.expected_outcome}</strong><span>{resultLabel(result?.result)}</span></div>; }) : <p>No outcome check has been defined.</p>}{!detail.closure.ready && detail.closure.reasons.length > 0 && <div className="closure-note"><strong>Before this can close</strong><ul>{detail.closure.reasons.map((reason) => <li key={reason}>{reason}</li>)}</ul></div>}</section>
          </>}
        </div>}
      </article>;
    })}</section>
    {nextCursor && <div className="load-more"><button className="secondary-button" disabled={loadingMore} onClick={() => void load(false, nextCursor)}>{loadingMore ? "Loading…" : "Load more items"}</button></div>}
  </>;
}
