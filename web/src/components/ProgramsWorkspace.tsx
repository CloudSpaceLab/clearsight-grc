import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { FormEvent } from "react";
import type { ProgramItemTarget, ProgramSection } from "../appRouting";
import { loadProgramSummaries } from "../api";
import type { ProgramSummary } from "../summaryTypes";
import type { ProgramAggregate, ProgramState } from "../types";
import { EmptyState } from "./EmptyState";
import { ProgramSetupWorkspace } from "./ProgramSetupWorkspace";
import { ProgramRecordWorkspace } from "./ProgramRecordWorkspace";
import { readWorkspaceFilters, replaceWorkspaceHash, workspaceHash } from "../workspaceFilters";

type LoadState = "loading" | "live" | "unavailable";
type ProgramListSummary = Omit<ProgramSummary, "open_matter_count"> & { open_matter_count?: number };
type Props = { targetID?: string; targetSection?: ProgramSection; programItem?: ProgramItemTarget; onSectionChange?: (programID: string, section: ProgramSection) => void; openFirst?: boolean; actorPrincipalID?: string; canConfigureSources?: boolean; onOpenRequest?: (requestID: string) => void; onOpenForm?: (formID: string) => void };

function ProgramIcon() {
  return <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="M6 3h9l3 3v15H6z"/><path d="M15 3v4h4M9 11h6M9 15h6M9 19h4"/></svg>;
}

function stateClass(value?: ProgramState) {
  switch (value) {
    case "CURRENT": return "status-good";
    case "GAP_IDENTIFIED": case "OVERDUE": return "status-critical";
    case "AT_RISK": case "EVIDENCE_INSUFFICIENT": case "IMPLEMENTATION_PENDING": case "UNDER_REVIEW": return "status-warning";
    default: return "status-neutral";
  }
}

function hasAssessment(summary: ProgramListSummary) {
  return Number.isInteger(summary.assessed_program_version) && summary.assessed_program_version > 0;
}

function needsAssessment(summary: ProgramListSummary) {
  return !hasAssessment(summary) || summary.projection_stale || summary.assessed_program_version !== summary.program_version;
}

function summaryFromAggregate(detail: ProgramAggregate): ProgramListSummary {
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
    open_matter_count: current?.open_matter_count,
    requirement_count: detail.requirements.length,
    safeguard_count: detail.control_implementations.length,
    evidence_check_count: detail.evidence_contracts.length,
    program_version: detail.program.version,
    assessed_program_version: assessedVersion,
    projection_version: current?.projection_version ?? 0,
    projection_stale: !current || assessedVersion !== detail.program.version,
    state_generated_at: current?.generated_at,
  };
}

export function ProgramsWorkspace(props: Props) {
  if (props.targetID) return <ProgramRecordWorkspace programID={props.targetID} section={props.targetSection} programItem={props.programItem} onSectionChange={(section) => props.onSectionChange?.(props.targetID!, section)} actorPrincipalID={props.actorPrincipalID} canConfigureSources={props.canConfigureSources} onOpenRequest={props.onOpenRequest} onOpenForm={props.onOpenForm} onBack={() => { window.location.hash = workspaceHash("#programs", readWorkspaceFilters(window.location.hash)); }}/>;
  return <ProgramListWorkspace {...props}/>;
}

function humanizeProgramState(value: string) {
  const labels: Record<string, string> = {
    CURRENT: "Up to date",
    AT_RISK: "At risk",
    GAP_IDENTIFIED: "Gap identified",
    EVIDENCE_INSUFFICIENT: "Evidence incomplete",
    IMPLEMENTATION_PENDING: "Implementation pending",
    OVERDUE: "Overdue",
    UNDER_REVIEW: "Under review",
    UNKNOWN: "Not assessed",
  };
  return labels[value] ?? value.replaceAll("_", " ").toLowerCase();
}

function ProgramListWorkspace({ targetID, openFirst = false, actorPrincipalID = "", canConfigureSources = false }: Props) {
  const initialFilters = useMemo(() => readWorkspaceFilters(window.location.hash), []);
  const [items, setItems] = useState<ProgramListSummary[]>([]);
  const [state, setState] = useState<LoadState>("loading");
  const [nextCursor, setNextCursor] = useState("");
  const [searchDraft, setSearchDraft] = useState(initialFilters.q ?? "");
  const [search, setSearch] = useState(initialFilters.q ?? "");
  const [statusDraft, setStatusDraft] = useState(initialFilters.status ?? "");
  const [status, setStatus] = useState(initialFilters.status ?? "");
  const [overallStateDraft, setOverallStateDraft] = useState(initialFilters.overall_state ?? "");
  const [overallState, setOverallState] = useState(initialFilters.overall_state ?? "");
  const [jurisdictionDraft, setJurisdictionDraft] = useState(initialFilters.jurisdiction ?? "");
  const [jurisdiction, setJurisdiction] = useState(initialFilters.jurisdiction ?? "");
  const [assignedDraft, setAssignedDraft] = useState(initialFilters.assigned_to_me === "true");
  const [assignedToMe, setAssignedToMe] = useState(initialFilters.assigned_to_me === "true");
  const [loadingMore, setLoadingMore] = useState(false);
  const [setupOpen, setSetupOpen] = useState(false);
  const requestID = useRef(0);
  const handledTarget = useRef("");

  useEffect(() => {
    return () => {
      requestID.current += 1;
    };
  }, []);

  const load = useCallback(async (reset: boolean, cursor = "") => {
    const currentRequest = ++requestID.current;
    if (reset) setState("loading"); else setLoadingMore(true);
    try {
      const page = await loadProgramSummaries({ q: search, status, overallState, jurisdiction, assignedToMe, cursor, limit: 20 });
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
  }, [assignedToMe, jurisdiction, overallState, search, status]);

  useEffect(() => { void load(true); }, [load]);

  const summary = useMemo(() => ({
    current: items.filter((item) => !needsAssessment(item) && item.overall_state === "CURRENT").length,
    attention: items.filter((item) => !needsAssessment(item) && ["AT_RISK", "GAP_IDENTIFIED", "EVIDENCE_INSUFFICIENT", "IMPLEMENTATION_PENDING", "OVERDUE"].includes(item.overall_state)).length,
    setup: items.filter((item) => needsAssessment(item) || item.program.status === "DRAFT" || ["UNKNOWN", "UNDER_REVIEW"].includes(item.overall_state)).length,
  }), [items]);

  function submitSearch(event: FormEvent) {
    event.preventDefault();
    const nextSearch = searchDraft.trim();
    const nextJurisdiction = jurisdictionDraft.trim();
    setSearch(nextSearch);
    setStatus(statusDraft);
    setOverallState(overallStateDraft);
    setJurisdiction(nextJurisdiction);
    setAssignedToMe(assignedDraft);
    replaceWorkspaceHash(workspaceHash("#programs", { q: nextSearch, status: statusDraft, overall_state: overallStateDraft, jurisdiction: nextJurisdiction, assigned_to_me: assignedDraft }));
  }

  function clearFilters() {
    setSearchDraft("");
    setSearch("");
    setStatusDraft("");
    setStatus("");
    setOverallStateDraft("");
    setOverallState("");
    setJurisdictionDraft("");
    setJurisdiction("");
    setAssignedDraft(false);
    setAssignedToMe(false);
    replaceWorkspaceHash("#programs");
  }

  function applyCreatedProgram(value: ProgramAggregate) {
    setItems((current) => {
      const next = summaryFromAggregate(value);
      return current.some((item) => item.program.id === value.program.id) ? current.map((item) => item.program.id === value.program.id ? next : item) : [next, ...current];
    });
  }

  useEffect(() => {
    if (state !== "live") return;
    const id = targetID ?? (openFirst ? items[0]?.program.id : undefined);
    if (!id || handledTarget.current === id) return;
    handledTarget.current = id;
    window.location.hash = workspaceHash(`#programs/${encodeURIComponent(id)}`, { q: search, status, overall_state: overallState, jurisdiction, assigned_to_me: assignedToMe });
  }, [state, items, targetID, openFirst, search, status, overallState, jurisdiction, assignedToMe]);

  if (state === "loading") return <section id="programs-workspace" className="workspace-loading" aria-live="polite" aria-busy="true">Loading programs…</section>;
  if (state === "unavailable") return <div id="programs-workspace"><EmptyState label="Programs" title="Programs could not be loaded" description="The service is unavailable. No program totals are shown." action="Try again" onAction={() => void load(true)}/></div>;

  const briefTitle = summary.attention > 0
    ? `${summary.attention} loaded program${summary.attention === 1 ? " requires" : "s require"} follow-up`
    : summary.setup > 0
      ? `${summary.setup} loaded program${summary.setup === 1 ? " needs" : "s need"} setup, review or a current assessment`
      : items.length ? "No recorded gaps or overdue work in the loaded programs" : "No programs in this scope";
  const filtersActive = Boolean(search || status || overallState || jurisdiction || assignedToMe);
  const programFilters = { q: search, status, overall_state: overallState, jurisdiction, assigned_to_me: assignedToMe };

  return <div id="programs-workspace">
    <section className="workspace-brief">
        <div><span className="eyebrow">Ongoing compliance</span><h2>{briefTitle}</h2><p>Open a Program to review its requirements, collected data, evidence results and open issues.</p></div>
      <div className="workspace-brief-side"><div className="workspace-brief-facts" aria-label="Loaded Program status"><span><strong>{summary.attention}</strong> follow-up</span><span><strong>{summary.current}</strong> current</span><span><strong>{summary.setup}</strong> setup, review or assessment needed</span></div><button className="primary-button" type="button" onClick={() => setSetupOpen((current) => !current)}>{setupOpen ? "Close setup" : "New Program"}</button></div>
    </section>
    {setupOpen && <ProgramSetupWorkspace actorPrincipalID={actorPrincipalID} canConfigureSources={canConfigureSources} onCreated={applyCreatedProgram} onClose={() => setSetupOpen(false)}/>}
    <form className="workspace-toolbar" role="search" onSubmit={submitSearch}>
      <label className="workspace-search-field"><span>Search programs</span><input value={searchDraft} onChange={(event) => setSearchDraft(event.target.value)} placeholder="Name, code, function or jurisdiction"/></label>
      <label><span>Status</span><select value={statusDraft} onChange={(event) => setStatusDraft(event.target.value)}><option value="">All statuses</option><option value="ACTIVE">Active</option><option value="PAUSED">Paused</option><option value="DRAFT">Setup in progress</option><option value="RETIRED">Ended</option></select></label>
      <label><span>Operating state</span><select value={overallStateDraft} onChange={(event) => setOverallStateDraft(event.target.value)}><option value="">All operating states</option><option value="CURRENT">Up to date</option><option value="AT_RISK">At risk</option><option value="GAP_IDENTIFIED">Gap identified</option><option value="EVIDENCE_INSUFFICIENT">Evidence incomplete</option><option value="IMPLEMENTATION_PENDING">Implementation pending</option><option value="OVERDUE">Overdue</option><option value="UNDER_REVIEW">Under review</option><option value="UNKNOWN">Not assessed</option></select></label>
      <label><span>Jurisdiction</span><input value={jurisdictionDraft} onChange={(event) => setJurisdictionDraft(event.target.value)} placeholder="Country or regulator"/></label>
      <label className="workspace-check-filter"><input type="checkbox" checked={assignedDraft} onChange={(event) => setAssignedDraft(event.target.checked)}/><span>Assigned to me</span></label>
      <button className="secondary-button" type="submit">Apply filters</button>
      {filtersActive && <button className="text-button" type="button" onClick={clearFilters}>Clear filters</button>}
    </form>
    {filtersActive && <div className="workspace-filter-chips" aria-label="Applied Program filters">{search && <span>Search: {search}</span>}{status && <span>{status === "DRAFT" ? "Setup in progress" : status[0] + status.slice(1).toLowerCase()}</span>}{overallState && <span>{humanizeProgramState(overallState)}</span>}{jurisdiction && <span>{jurisdiction}</span>}{assignedToMe && <span>Assigned to me</span>}</div>}
      {!items.length ? <EmptyState label="Programs" title={filtersActive ? "No programs match these filters" : "No programs in this scope"} description={filtersActive ? "Change or clear the filters to see other Programs in your access scope." : "There are no ongoing compliance or control Programs in your current access scope."} action={filtersActive ? "Clear filters" : undefined} onAction={clearFilters}/> : items.length ? <section className="program-list">
      {items.map((summaryItem) => {
        const program = summaryItem.program;
        const assessmentMissing = !hasAssessment(summaryItem);
        const assessmentStale = needsAssessment(summaryItem);
        const displayState = assessmentStale ? "UNKNOWN" : summaryItem.overall_state;
        const displayLabel = assessmentMissing ? "Unknown" : assessmentStale ? "Out of date" : summaryItem.state_label || "Unknown";
        const openIssues = summaryItem.open_matter_count;
        const knownOpenIssues = !assessmentMissing && typeof openIssues === "number" && Number.isInteger(openIssues) && openIssues >= 0;
        return <article className={targetID === program.id ? "program-card targeted" : "program-card"} id={`program-${program.id}`} key={program.id}>
          <a className="program-card-main" href={workspaceHash(`#programs/${encodeURIComponent(program.id)}`, programFilters)}>
            <span className="program-icon"><ProgramIcon/></span>
            <span className="program-primary"><span className="program-kicker">{program.code} · {program.owning_function}</span><strong>{program.name}</strong>{program.jurisdiction && <small>{program.jurisdiction}</small>}</span>
            <span className="program-counts"><span><b>{summaryItem.requirement_count}</b> requirements</span><span><b>{summaryItem.evidence_check_count}</b> evidence checks</span><span><b>{knownOpenIssues ? openIssues : "Unknown"}</b> open issues{knownOpenIssues && assessmentStale ? " (last calculation)" : ""}</span></span>
            <span className={`program-state ${stateClass(displayState)}`}><strong>{displayLabel}</strong></span>
          </a>
        </article>;
      })}
    </section> : null}
    {nextCursor && <div className="load-more"><button className="secondary-button" disabled={loadingMore} onClick={() => void load(false, nextCursor)}>{loadingMore ? "Loading…" : "Load more programs"}</button></div>}
  </div>;
}
