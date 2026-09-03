import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { FormEvent, ReactNode } from "react";
import type { ProgramSection } from "../appRouting";
import { loadProgram, loadProgramSummaries } from "../api";
import type { ProgramSummary } from "../summaryTypes";
import type { ProgramAggregate, ProgramState } from "../types";
import { EmptyState } from "./EmptyState";
import { ProgramLifecycleControls } from "./ProgramLifecycleControls";
import { ProgramReviewDigest } from "./ProgramReviewDigest";
import { ProgramSetupWorkspace } from "./ProgramSetupWorkspace";
import { MonitoringSetup } from "./MonitoringSetup";
import { ProgramDetailSections } from "./ProgramDetailSections";

type LoadState = "loading" | "live" | "unavailable";
type Props = { targetID?: string; targetSection?: ProgramSection; onSectionChange?: (programID: string, section: ProgramSection) => void; openFirst?: boolean; actorPrincipalID?: string; canConfigureSources?: boolean };

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

function summaryFromAggregate(detail: ProgramAggregate): ProgramSummary {
  const current = detail.current_state;
  const assessedVersion = current?.program_version ?? 0;
  const reasons = current?.reasons ?? [];
  return {
    program: detail.program,
    state_label: detail.state_label,
    overall_state: current?.overall_state ?? current?.overall ?? "UNKNOWN",
    reasons,
    reasons_total: reasons.length,
    reasons_omitted: 0,
    open_matter_count: current?.open_matter_count ?? 0,
    requirement_count: detail.requirements.length,
    safeguard_count: detail.control_implementations.length,
    evidence_check_count: detail.evidence_contracts.length,
    program_version: detail.program.version,
    assessed_program_version: assessedVersion,
    projection_version: current?.projection_version ?? 0,
    projection_stale: !current || assessedVersion < detail.program.version,
    state_generated_at: current?.generated_at,
  };
}

function formatProgramTime(value: string) {
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(parsed)) : "Not available";
}

export function ProgramsWorkspace({ targetID, targetSection = "overview", onSectionChange, openFirst = false, actorPrincipalID = "", canConfigureSources = false }: Props) {
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
  const [setupOpen, setSetupOpen] = useState(false);
  const [localSections, setLocalSections] = useState<Record<string, ProgramSection>>({});
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
    current: items.filter((item) => !item.projection_stale && item.overall_state === "CURRENT").length,
    attention: items.filter((item) => !item.projection_stale && ["AT_RISK", "GAP_IDENTIFIED", "EVIDENCE_INSUFFICIENT", "IMPLEMENTATION_PENDING", "OVERDUE"].includes(item.overall_state)).length,
    setup: items.filter((item) => item.projection_stale || item.program.status === "DRAFT" || ["UNKNOWN", "UNDER_REVIEW"].includes(item.overall_state)).length,
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

  async function fetchDetail(id: string): Promise<ProgramAggregate | null> {
    if (detailState[id] === "loading") return details[id] ?? null;
    setDetailState((current) => ({ ...current, [id]: "loading" }));
    try {
      const value = await loadProgram(id);
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

  function applyDetailUpdate(value: ProgramAggregate) {
    setDetails((current) => ({ ...current, [value.program.id]: value }));
    setItems((current) => current.map((item) => item.program.id === value.program.id ? summaryFromAggregate(value) : item));
  }

  function applyCreatedProgram(value: ProgramAggregate) {
    setDetails((current) => ({ ...current, [value.program.id]: value }));
    setDetailState((current) => ({ ...current, [value.program.id]: "live" }));
    setItems((current) => {
      const next = summaryFromAggregate(value);
      return current.some((item) => item.program.id === value.program.id) ? current.map((item) => item.program.id === value.program.id ? next : item) : [next, ...current];
    });
    setOpenID(value.program.id);
  }

  function selectProgramSection(programID: string, section: ProgramSection) {
    setLocalSections((current) => ({ ...current, [programID]: section }));
    onSectionChange?.(programID, section);
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
    const id = targetID ?? (openFirst ? items[0]?.program.id : undefined);
    if (!id || handledTarget.current === id) return;
    handledTarget.current = id;

    void (async () => {
      const inPage = items.some((item) => item.program.id === id);
      const detail = details[id] ?? await fetchDetail(id);
      if (!mounted.current) return;
      if (detail && !inPage) {
        setItems((current) => current.some((item) => item.program.id === id) ? current : [summaryFromAggregate(detail), ...current]);
      }
      if (detail || inPage) setOpenID(id);
      if (targetScrollTimer.current !== null) window.clearTimeout(targetScrollTimer.current);
      targetScrollTimer.current = window.setTimeout(() => {
        targetScrollTimer.current = null;
        document.getElementById(`program-${id}`)?.scrollIntoView({ behavior: "smooth", block: "center" });
      }, 80);
    })();
  }, [state, items, targetID, openFirst]);

  if (state === "loading") return <section id="programs-workspace" className="workspace-loading" aria-live="polite" aria-busy="true">Loading programs…</section>;
  if (state === "unavailable") return <div id="programs-workspace"><EmptyState label="Programs" title="Programs could not be loaded" description="The service is unavailable. No program totals are shown." action="Try again" onAction={() => void load(true)}/></div>;

  const briefTitle = summary.attention > 0
    ? `${summary.attention} loaded program${summary.attention === 1 ? " requires" : "s require"} follow-up`
    : summary.setup > 0
      ? `${summary.setup} loaded program${summary.setup === 1 ? " is" : "s are"} still being set up or reassessed`
      : items.length ? "No recorded gaps or overdue work in the loaded programs" : "No programs in this scope";
  const targetInList = !targetID || items.some((item) => item.program.id === targetID);
  const targetLoading = Boolean(targetID && !targetInList && detailState[targetID] === "loading");
  const targetUnavailable = Boolean(targetID && !targetInList && detailState[targetID] === "unavailable");

  return <div id="programs-workspace">
    <section className="workspace-brief">
        <div><span className="eyebrow">Ongoing compliance</span><h2>{briefTitle}</h2><p>Open a Program to review status, recent changes, requirements, evidence and available actions.</p></div>
      <div className="workspace-brief-side"><div className="workspace-brief-facts" aria-label="Loaded Program status"><span><strong>{summary.attention}</strong> follow-up</span><span><strong>{summary.current}</strong> current</span><span><strong>{summary.setup}</strong> setup or reassessing</span></div><button className="primary-button" type="button" onClick={() => setSetupOpen((current) => !current)}>{setupOpen ? "Close setup" : "New Program"}</button></div>
    </section>
    {setupOpen && <ProgramSetupWorkspace actorPrincipalID={actorPrincipalID} canConfigureSources={canConfigureSources} onCreated={applyCreatedProgram} onClose={() => setSetupOpen(false)}/>}
    <form className="workspace-toolbar" role="search" onSubmit={submitSearch}>
      <label><span>Search programs</span><input value={searchDraft} onChange={(event) => setSearchDraft(event.target.value)} placeholder="Name, code, function or jurisdiction"/></label>
      <label><span>Status</span><select value={status} onChange={(event) => setStatus(event.target.value)}><option value="">All statuses</option><option value="ACTIVE">Active</option><option value="PAUSED">Paused</option><option value="DRAFT">Setup in progress</option><option value="RETIRED">Ended</option></select></label>
      <button className="secondary-button" type="submit">Search</button>
      {(search || status) && <button className="text-button" type="button" onClick={clearFilters}>Clear filters</button>}
    </form>
    {targetLoading && <div className="workspace-loading compact" aria-live="polite" aria-busy="true">Loading requested Program…</div>}
    {targetUnavailable && <EmptyState label="Requested Program" title="The requested Program could not be loaded" description="It may be outside your current access scope or no longer available."/>}
      {!items.length && !targetLoading && !targetUnavailable ? <EmptyState label="Programs" title={search || status ? "No programs match these filters" : "No programs in this scope"} description={search || status ? "Change the search or status filter to see other programs." : "There are no ongoing compliance or control Programs in your current access scope."} action={search || status ? "Clear filters" : undefined} onAction={clearFilters}/> : items.length ? <section className="program-list">
      {items.map((summaryItem) => {
        const program = summaryItem.program;
        const isOpen = openID === program.id;
        const detail = details[program.id];
        const currentDetailState = detailState[program.id];
        const displayState = summaryItem.projection_stale ? "UNKNOWN" : summaryItem.overall_state;
        const displayLabel = summaryItem.projection_stale ? "Updating status" : summaryItem.state_label || "Not assessed";
        const displayReason = summaryItem.projection_stale
          ? `Last assessed at version ${summaryItem.assessed_program_version}; Program is version ${summaryItem.program_version}.`
          : summaryItem.reasons[0]?.summary ?? "Open for current status reasons";
        return <article className={targetID === program.id ? "program-card targeted" : "program-card"} id={`program-${program.id}`} key={program.id}>
          <button className="program-card-main" type="button" aria-expanded={isOpen} aria-controls={`program-detail-${program.id}`} onClick={() => void toggleDetail(program.id)}>
            <span className="program-icon"><ProgramIcon/></span>
            <span className="program-primary"><span className="program-kicker">{program.code} · {program.owning_function}</span><strong>{program.name}</strong>{program.jurisdiction && <small>{program.jurisdiction}</small>}</span>
            <span className="program-counts"><span><b>{summaryItem.requirement_count}</b> requirements</span><span><b>{summaryItem.evidence_check_count}</b> evidence checks</span><span><b>{summaryItem.open_matter_count}</b> open issues</span></span>
            <span className={`program-state ${stateClass(displayState)}`}><strong>{displayLabel}</strong><small>{displayReason}</small></span>
            <span className="expand-indicator" aria-hidden="true">{isOpen ? "−" : "+"}</span>
          </button>
          {isOpen && <div className="program-detail progressive-detail" id={`program-detail-${program.id}`}>
            {currentDetailState === "loading" && <p aria-live="polite">Loading program details…</p>}
            {currentDetailState === "unavailable" && <div className="inline-error"><p>Program details could not be loaded.</p><button className="secondary-button" onClick={() => void fetchDetail(program.id)}>Try again</button></div>}
            {detail && (() => {
              const selectedSection = targetID === program.id ? targetSection : localSections[program.id] ?? "overview";
              const contractNames = new Map(detail.evidence_contracts.map((contract) => [contract.id, contract.name]));
              const panels = {
                overview: <div className="program-section-stack">
                  <ProgramReviewDigest aggregate={detail}/>
                  <ProgramLifecycleControls aggregate={detail} onUpdated={applyDetailUpdate}/>
                  <section className="status-reasons"><h4>Why this status</h4>{detail.current_state?.reasons?.length ? <ul>{detail.current_state.reasons.map((reason) => <li key={`${reason.code}-${reason.object_id ?? ""}`}>{reason.summary}</li>)}</ul> : <p>No status reasons are recorded for the latest assessment.</p>}{summaryItem.reasons_omitted > 0 && <p>{summaryItem.reasons_omitted} additional status reason{summaryItem.reasons_omitted === 1 ? " is" : "s are"} available in Program history.</p>}</section>
                </div>,
                "requirements-controls": <div className="program-section-stack">
                  <section><div className="program-section-heading"><h4>Requirements</h4><span>{detail.requirements.length}</span></div>{detail.requirements.length ? detail.requirements.map((requirement) => <div className="detail-row" key={requirement.id}><div><strong>{requirement.title}</strong><small>{requirement.statement}</small>{requirement.source_anchor && <small>Source: {requirement.source_anchor}</small>}</div><span>{requirementStatusLabel(requirement.status)}</span></div>) : <p>No approved requirements have been added to this Program.</p>}</section>
                  <section><div className="program-section-heading"><h4>Controls</h4><span>{detail.control_implementations.length}</span></div>{detail.control_implementations.length ? detail.control_implementations.map((control) => <div className="detail-row" key={control.id}><div><strong>{control.name}</strong><small>{control.description}</small></div><span>{requirementStatusLabel(control.status)}</span></div>) : <p>No control implementations have been added to this Program.</p>}</section>
                </div>,
                monitoring: <MonitoringSetup aggregate={detail} actorPrincipalID={actorPrincipalID} canConfigureSources={canConfigureSources}/>,
                "evidence-results": <div className="program-section-stack">
                  <section><div className="program-section-heading"><h4>Evidence expectations</h4><span>{detail.evidence_contracts.length}</span></div>{detail.evidence_contracts.length ? detail.evidence_contracts.map((contract) => <div className="detail-row" key={contract.id}><div><strong>{contract.name}</strong><small>{contract.claim}</small></div><span>Required coverage: {Math.round(contract.minimum_coverage * 100)}%</span></div>) : <p>No evidence expectations have been defined for this Program.</p>}</section>
                  <section><div className="program-section-heading"><h4>Assessment results</h4><span>{detail.evidence_assessments.length}</span></div>{detail.evidence_assessments.length ? detail.evidence_assessments.map((assessment) => <div className="detail-row" key={assessment.id}><div><strong>{contractNames.get(assessment.contract_id) ?? "Evidence expectation"}</strong><small>Assessed <time dateTime={assessment.assessed_at}>{formatProgramTime(assessment.assessed_at)}</time></small></div><span>{requirementStatusLabel(assessment.conclusion)} · {Math.round(assessment.coverage * 100)}% coverage</span></div>) : <p>No evidence assessment results have been recorded for this Program.</p>}</section>
                </div>,
                "issues-actions": <section className="program-section-empty"><h4>Open issues and actions</h4>{(detail.current_state?.open_matter_count ?? 0) > 0 ? <p>{detail.current_state?.open_matter_count} open issue{detail.current_state?.open_matter_count === 1 ? " or change is" : "s or changes are"} recorded in the latest Program status. Review the Work workspace for assigned decisions and actions.</p> : <p>No open issues or changes are recorded in the latest Program status.</p>}</section>,
                history: <div className="program-section-stack">
                  <dl className="program-history-facts"><div><dt>Program version</dt><dd>{detail.program.version}</dd></div><div><dt>Created</dt><dd><time dateTime={detail.program.created_at}>{formatProgramTime(detail.program.created_at)}</time></dd></div><div><dt>Last changed</dt><dd><time dateTime={detail.program.updated_at}>{formatProgramTime(detail.program.updated_at)}</time></dd></div><div><dt>Latest status calculated</dt><dd>{detail.current_state?.generated_at ? <time dateTime={detail.current_state.generated_at}>{formatProgramTime(detail.current_state.generated_at)}</time> : "Not available"}</dd></div></dl>
                  <section><div className="program-section-heading"><h4>Recorded triggers</h4><span>{detail.triggers.length}</span></div>{detail.triggers.length ? detail.triggers.map((trigger) => <div className="detail-row" key={trigger.id}><div><strong>{requirementStatusLabel(trigger.type)}</strong><small>{trigger.source}</small></div><time dateTime={trigger.observed_at}>{formatProgramTime(trigger.observed_at)}</time></div>) : <p>No Program triggers have been recorded.</p>}</section>
                </div>,
              } satisfies Record<ProgramSection, ReactNode>;
              return <>
                {summaryItem.projection_stale && <div className="inline-notice" role="status">The Program changed after the latest assessment. The last known reasons remain available while status is recalculated.</div>}
                <ProgramDetailSections section={selectedSection} panels={panels} onSectionChange={(section) => selectProgramSection(program.id, section)}/>
              </>;
            })()}
          </div>}
        </article>;
      })}
    </section> : null}
    {nextCursor && <div className="load-more"><button className="secondary-button" disabled={loadingMore} onClick={() => void load(false, nextCursor)}>{loadingMore ? "Loading…" : "Load more programs"}</button></div>}
  </div>;
}
