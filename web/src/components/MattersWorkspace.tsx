import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { FormEvent } from "react";
import { loadMatter, loadMatterSummaries } from "../api";
import type { MatterSummary } from "../summaryTypes";
import type { MatterAggregate } from "../types";
import { EmptyState } from "./EmptyState";
import { MatterWorkCommandPanel } from "./MatterWorkCommandPanel";
import { MatterRecordWorkspace } from "./MatterRecordWorkspace";
import { MatterSetupWorkspace } from "./MatterSetupWorkspace";
import { VendorRelationshipLinks } from "./VendorRelationshipLinks";
import { VendorWorkPanel } from "./VendorWorkPanel";

type LoadState = "loading" | "live" | "unavailable";
type Props = { targetID?: string; openFirst?: boolean; onBack?: () => void };

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
    default: return humanizeKey(value);
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
  return value.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function formatFact(value: unknown) {
  if (value === null || value === undefined || value === "") return "Not recorded";
  if (typeof value === "object") return JSON.stringify(value, null, 2);
  return String(value);
}

function latestResult(aggregate: MatterAggregate, contractID: string) {
  return (aggregate.verification_results ?? [])
    .filter((result) => result.contract_id === contractID)
    .sort((left, right) => Date.parse(right.observed_at) - Date.parse(left.observed_at))[0];
}

function latestOutcome(aggregate: MatterAggregate) {
  return [...aggregate.verification_results].sort((left, right) => Date.parse(right.observed_at) - Date.parse(left.observed_at))[0];
}

function resultLabel(value?: string) {
  switch (value) {
    case "PASS": return "Outcome confirmed";
    case "FAIL": return "Outcome not achieved";
    case "INCONCLUSIVE": return "More evidence needed";
    default: return "Not checked yet";
  }
}

function summaryFromAggregate(detail: MatterAggregate): MatterSummary {
  const latest = latestOutcome(detail);
  const programIDs = new Set(detail.links.map((link) => link.program_id).filter((value): value is string => Boolean(value)));
  return {
    matter: detail.matter,
    type_label: detail.type_label,
    status_label: detail.status_label,
    next_action: detail.next_action,
    program_count: programIDs.size,
    open_action_count: detail.actions.filter((action) => !["IMPLEMENTED", "CANCELLED"].includes(action.status)).length,
    outcome_check_count: detail.verification_contracts.length,
    latest_outcome: latest?.result,
    latest_outcome_at: latest?.observed_at,
  };
}

export function MattersWorkspace({ targetID, openFirst = false, onBack }: Props) {
  if (targetID) return <MatterRecordWorkspace matterID={targetID} onBack={onBack ?? (() => { window.location.hash = "#work"; })}/>;
  return <MatterListWorkspace openFirst={openFirst}/>;
}

function MatterListWorkspace({ openFirst = false }: Pick<Props, "openFirst">) {
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
  const [setupOpen, setSetupOpen] = useState(false);
  const [creationNotice, setCreationNotice] = useState("");
  const requestID = useRef(0);
  const handledTarget = useRef("");
  const mounted = useRef(true);
  const targetScrollTimer = useRef<number | null>(null);

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      requestID.current += 1;
      if (targetScrollTimer.current !== null) window.clearTimeout(targetScrollTimer.current);
    };
  }, []);

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

  async function fetchDetail(id: string): Promise<MatterAggregate | null> {
    if (detailState[id] === "loading") return details[id] ?? null;
    setDetailState((current) => ({ ...current, [id]: "loading" }));
    try {
      const value = await loadMatter(id);
      if (!mounted.current) return null;
      setDetails((current) => ({ ...current, [id]: value }));
      setDetailState((current) => ({ ...current, [id]: "live" }));
      return value;
    } catch {
      if (!mounted.current) return null;
      setDetailState((current) => ({ ...current, [id]: "unavailable" }));
      return null;
    }
  }

  function applyDetailUpdate(value: MatterAggregate) {
    setDetails((current) => ({ ...current, [value.matter.id]: value }));
    setItems((current) => current.map((item) => item.matter.id === value.matter.id ? summaryFromAggregate(value) : item));
  }

  function applyCreatedMatter(value: MatterAggregate) {
    setSearchDraft("");
    setSearch("");
    setStatus("OPEN");
    setDetails((current) => ({ ...current, [value.matter.id]: value }));
    setDetailState((current) => ({ ...current, [value.matter.id]: "live" }));
    setItems((current) => [summaryFromAggregate(value), ...current.filter((item) => item.matter.id !== value.matter.id)]);
    setOpenID(value.matter.id);
    setSetupOpen(false);
    setCreationNotice("Issue or change created.");
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
    if (state !== "live") return;
    const id = openFirst ? items[0]?.matter.id : undefined;
    if (!id || handledTarget.current === id) return;
    handledTarget.current = id;

    void (async () => {
      const inPage = items.some((item) => item.matter.id === id);
      const detail = details[id] ?? await fetchDetail(id);
      if (!mounted.current) return;
      if (detail && !inPage) {
        setItems((current) => current.some((item) => item.matter.id === id) ? current : [summaryFromAggregate(detail), ...current]);
      }
      if (detail || inPage) setOpenID(id);
      if (targetScrollTimer.current !== null) window.clearTimeout(targetScrollTimer.current);
      targetScrollTimer.current = window.setTimeout(() => {
        targetScrollTimer.current = null;
        document.getElementById(`matter-${id}`)?.scrollIntoView({ behavior: "smooth", block: "center" });
      }, 80);
    })();
  }, [state, items, openFirst]);

  if (state === "loading") return <section id="matters-workspace" className="workspace-loading" aria-live="polite" aria-busy="true">Loading issues and changes…</section>;
  if (state === "unavailable") return <div id="matters-workspace"><EmptyState label="Issues and changes" title="Issues and changes could not be loaded" description="The service is unavailable. No current work totals are shown." action="Try again" onAction={() => void load(true)}/></div>;

  return <div id="matters-workspace">
    <section className="workspace-brief">
      <div><span className="eyebrow">Issues and changes</span><h2>{items.length ? `${items.length} loaded item${items.length === 1 ? "" : "s"}` : "No open items in this view"}</h2><p>Open an item to review its current handoff, facts, actions and outcome checks.</p></div>
      <div className="workspace-brief-side"><div className="workspace-brief-facts" aria-label="Loaded work summary"><span><strong>{summary.decisions}</strong> decisions</span><span><strong>{summary.overdue}</strong> overdue</span><span><strong>{summary.checking}</strong> outcome checks</span></div>{!setupOpen && <button className="primary-button" type="button" onClick={() => { setCreationNotice(""); setSetupOpen(true); }}>New issue or change</button>}</div>
    </section>
    {setupOpen && <MatterSetupWorkspace onCreated={applyCreatedMatter} onClose={() => setSetupOpen(false)}/>}
    {creationNotice && <p className="inline-success" role="status">{creationNotice}</p>}
    <form className="workspace-toolbar" role="search" onSubmit={submitSearch}>
      <label><span>Search issues and changes</span><input value={searchDraft} onChange={(event) => setSearchDraft(event.target.value)} placeholder="Reference, title, summary or type"/></label>
      <label><span>Status</span><select value={status} onChange={(event) => setStatus(event.target.value)}><option value="OPEN">Open</option><option value="DECISION_REQUIRED">Decision needed</option><option value="ACTION_IN_PROGRESS">Work in progress</option><option value="VERIFICATION">Confirming outcome</option><option value="CLOSED">Closed</option><option value="">All statuses</option></select></label>
      <button className="secondary-button" type="submit">Search</button>
      {(search || status !== "OPEN") && <button className="text-button" type="button" onClick={clearFilters}>Clear filters</button>}
    </form>
      {!setupOpen && !items.length ? <EmptyState label="Issues and changes" title={search || status !== "OPEN" ? "No items match these filters" : "No open issues or changes"} description={search || status !== "OPEN" ? "Change the search or status filter to see other work." : "There are no open changes, gaps, findings, exceptions or responses in your current access scope."} action={search || status !== "OPEN" ? "Clear filters" : "Create issue or change"} onAction={search || status !== "OPEN" ? clearFilters : () => { setCreationNotice(""); setSetupOpen(true); }}/> : items.length ? <section className="matter-list">{items.map((summaryItem) => {
      const matter = summaryItem.matter;
      const isOpen = openID === matter.id;
      const detail = details[matter.id];
      const currentDetailState = detailState[matter.id];
      const knownFacts = detail?.matter.known_facts ?? {};
      const missingFacts = detail?.matter.missing_facts ?? [];
      const contradictions = detail?.matter.contradictions ?? [];
      const actions = detail?.actions ?? [];
      const decisions = detail?.decisions ?? [];
      const responses = detail?.response_packages ?? [];
      const verificationContracts = detail?.verification_contracts ?? [];
      const closure = detail?.closure ?? { ready: false, reasons: [] };
      return <article className="matter-card" id={`matter-${matter.id}`} key={matter.id}>
        <button type="button" className="matter-card-main" aria-expanded={isOpen} aria-controls={`matter-detail-${matter.id}`} onClick={() => void toggleDetail(matter.id)}>
          <span className="matter-icon"><MatterIcon type={matter.type}/></span>
          <span className="matter-primary"><span className="matter-kicker">{summaryItem.type_label} · {matter.reference}</span><strong>{matter.title}</strong><small>{matter.summary}</small></span>
          <span className="matter-meta"><span>{priorityLabel(matter.priority)} priority</span><span>{matter.due_at ? `${Date.parse(matter.due_at) < Date.now() ? "Overdue" : "Due"} ${new Date(matter.due_at).toLocaleDateString()}` : "No due date"}</span><span>{summaryItem.open_action_count} open action{summaryItem.open_action_count === 1 ? "" : "s"}</span></span>
          <span className={`matter-status status-${matter.status.toLowerCase().replaceAll("_", "-")}`}><strong>{summaryItem.status_label}</strong><small>{summaryItem.next_action}</small></span>
          <span className="expand-indicator" aria-hidden="true">{isOpen ? "−" : "+"}</span>
        </button>
        <div className="record-open-actions"><button className="secondary-button" type="button" onClick={() => { window.location.hash = `#work/matters/${encodeURIComponent(matter.id)}`; }}>Open issue workspace</button></div>
        {isOpen && <div className="matter-detail progressive-detail" id={`matter-detail-${matter.id}`}>
          {currentDetailState === "loading" && <p aria-live="polite">Loading issue details…</p>}
          {currentDetailState === "unavailable" && <div className="inline-error"><p>Issue details could not be loaded.</p><button className="secondary-button" onClick={() => void fetchDetail(matter.id)}>Try again</button></div>}
          {detail && <>
            <section className="handoff-summary"><h3>Current handoff</h3><strong>{summaryItem.next_action}</strong><p>{matter.summary}</p><div className="handoff-flags">{missingFacts.length > 0 && <span>{missingFacts.length} missing fact{missingFacts.length === 1 ? "" : "s"}</span>}{contradictions.length > 0 && <span>{contradictions.length} contradiction{contradictions.length === 1 ? "" : "s"}</span>}{verificationContracts.length > 0 && <span>{verificationContracts.length} outcome check{verificationContracts.length === 1 ? "" : "s"}</span>}</div></section>
            <MatterWorkCommandPanel aggregate={detail} onUpdated={applyDetailUpdate}/>
            <VendorRelationshipLinks targetType="MATTER" targetID={matter.id}/>
            <VendorWorkPanel targetType="MATTER" targetID={matter.id} onOpenRequest={onOpenRequest}/>
            <details className="progressive-section"><summary><span>Evidence and facts</span><strong>{Object.keys(knownFacts).length + missingFacts.length + contradictions.length}</strong></summary><div>{Object.keys(knownFacts).length ? <dl>{Object.entries(knownFacts).map(([key, value]) => <div key={key}><dt>{humanizeKey(key)}</dt><dd><pre>{formatFact(value)}</pre></dd></div>)}</dl> : <p>No facts have been recorded.</p>}{missingFacts.length ? <div className="closure-note"><strong>Information still needed</strong><ul>{missingFacts.map((fact, index) => <li key={`${index}-${formatFact(fact)}`}>{formatFact(fact)}</li>)}</ul></div> : null}{contradictions.length ? <div className="closure-note warning"><strong>Contradictions to resolve</strong><ul>{contradictions.map((fact, index) => <li key={`${index}-${formatFact(fact)}`}>{formatFact(fact)}</li>)}</ul></div> : null}</div></details>
            <details className="progressive-section"><summary><span>Decision and response history</span><strong>{decisions.length + responses.length}</strong></summary><div><section><h3>Decisions</h3>{decisions.length ? decisions.map((decision) => <div className="detail-row" key={decision.id}><div><strong>{humanizeKey(decision.type)}</strong><small>{decision.rationale}</small></div><span>{humanizeKey(decision.status)}{decision.selected_option ? ` · ${humanizeKey(decision.selected_option)}` : ""}</span></div>) : <p>No decisions have been recorded.</p>}</section><section><h3>External responses</h3>{responses.length ? responses.map((response) => <div className="detail-row" key={response.id}><div><strong>{response.purpose}</strong><small>{response.audience}</small></div><span>{humanizeKey(response.status)}</span></div>) : <p>No response package has been recorded.</p>}</section></div></details>
            <details className="progressive-section"><summary><span>Actions and outcome checks</span><strong>{actions.length + verificationContracts.length}</strong></summary><div><section><h3>Actions</h3>{actions.length ? actions.map((action) => <div className="detail-row" key={action.id}><div><strong>{action.title}</strong><small>{action.description}</small></div><span>{actionStatusLabel(action.status)}</span></div>) : <p>No actions have been recorded.</p>}</section><section><h3>Outcome checks</h3>{verificationContracts.length ? verificationContracts.map((contract) => { const result = latestResult(detail, contract.id); return <div className="detail-row" key={contract.id}><div><strong>{contract.expected_outcome}</strong>{result?.rationale && <small>{result.rationale}</small>}</div><span>{resultLabel(result?.result)}</span></div>; }) : <p>No outcome check has been defined.</p>}{closure.ready ? <div className="closure-note ready"><strong>Ready to close</strong><p>All recorded closure requirements are satisfied.</p></div> : closure.reasons.length > 0 && <div className="closure-note"><strong>Before this can close</strong><ul>{closure.reasons.map((reason) => <li key={reason}>{reason}</li>)}</ul></div>}</section></div></details>
          </>}
        </div>}
      </article>;
    })}</section> : null}
    {nextCursor && <div className="load-more"><button className="secondary-button" disabled={loadingMore} onClick={() => void load(false, nextCursor)}>{loadingMore ? "Loading…" : "Load more items"}</button></div>}
  </div>;
}
