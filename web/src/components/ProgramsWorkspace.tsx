import { useMemo, useState } from "react";
import type { ProgramAggregate, ProgramState } from "../types";
import { EmptyState } from "./EmptyState";
import { PremiumIllustration } from "./PremiumIllustration";

type Props = { programs: ProgramAggregate[]; state: "loading" | "live" | "unavailable" };

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

export function ProgramsWorkspace({ programs, state }: Props) {
  const [openID, setOpenID] = useState<string | null>(null);
  const summary = useMemo(() => ({
    current: programs.filter((item) => item.current_state?.overall_state === "CURRENT" || item.current_state?.overall === "CURRENT").length,
    attention: programs.filter((item) => ["AT_RISK", "GAP_IDENTIFIED", "EVIDENCE_INSUFFICIENT", "IMPLEMENTATION_PENDING", "OVERDUE"].includes(item.current_state?.overall_state ?? item.current_state?.overall ?? "")).length,
    setup: programs.filter((item) => item.program.status === "DRAFT" || !item.current_state || ["UNKNOWN", "UNDER_REVIEW"].includes(item.current_state.overall_state ?? item.current_state.overall ?? "")).length,
  }), [programs]);
  const heroTitle = summary.attention > 0
    ? `${summary.attention} program${summary.attention === 1 ? " has" : "s have"} gaps, incomplete evidence or overdue work`
    : summary.setup > 0
      ? `${summary.setup} program${summary.setup === 1 ? " is" : "s are"} still being set up`
      : "No recorded gaps or overdue work in the loaded programs";

  if (state === "loading") return <section className="workspace-loading">Loading programs…</section>;
  if (state === "unavailable") return <EmptyState label="Programs" title="Programs could not be loaded" description="Try again when the service is available. Existing records have not been changed."/>;
  if (!programs.length) return <EmptyState label="Programs" title="No programs in this scope" description="There are no ongoing compliance or control programs in the current bank scope."/>;

  return <>
    <section className="program-hero">
      <div><span className="eyebrow">Ongoing compliance</span><h2>{heroTitle}</h2><p>Requirements, safeguards, evidence checks and open issues for ongoing responsibilities.</p></div>
      <PremiumIllustration variant="readiness"/>
    </section>
    <section className="program-summary" aria-label="Program summary">
      <div><span>Programs</span><strong>{programs.length}</strong><small>Current bank scope</small></div>
      <div><span>Up to date</span><strong>{summary.current}</strong><small>Based on the recorded requirements and evidence</small></div>
      <div><span>Open gaps or overdue work</span><strong>{summary.attention}</strong><small>Includes incomplete evidence and work in progress</small></div>
    </section>
    <section className="program-list">
      {programs.map((aggregate) => {
        const program = aggregate.program;
        const currentState = aggregate.current_state;
        const isOpen = openID === program.id;
        const overall = currentState?.overall_state ?? currentState?.overall;
        return <article className="program-card" key={program.id}>
          <button className="program-card-main" type="button" aria-expanded={isOpen} onClick={() => setOpenID(isOpen ? null : program.id)}>
            <span className="program-icon"><ProgramIcon/></span>
            <span className="program-primary"><span className="program-kicker">{program.code} · {program.owning_function}</span><strong>{program.name}</strong>{program.jurisdiction && <small>{program.jurisdiction}</small>}</span>
            <span className="program-counts"><span><b>{aggregate.requirements.length}</b> requirements recorded</span><span><b>{aggregate.control_implementations.length}</b> safeguards</span><span><b>{aggregate.evidence_contracts.length}</b> evidence checks</span></span>
            <span className={`program-state ${stateClass(overall)}`}><strong>{aggregate.state_label || "Not assessed"}</strong><small>{currentState?.open_matter_count ?? 0} open issue{currentState?.open_matter_count === 1 ? "" : "s"}</small></span>
            <span className="expand-indicator" aria-hidden="true">{isOpen ? "−" : "+"}</span>
          </button>
          {isOpen && <div className="program-detail">
            <section><h3>Why this status</h3>{currentState?.reasons?.length ? <ul>{currentState.reasons.slice(0, 6).map((reason) => <li key={`${reason.code}-${reason.object_id ?? ""}`}>{reason.summary}</li>)}</ul> : <p>No status reasons are recorded for the latest assessment.</p>}</section>
            <section><h3>Requirements</h3>{aggregate.requirements.length ? aggregate.requirements.slice(0, 5).map((requirement) => <div className="detail-row" key={requirement.id}><strong>{requirement.title}</strong><span>{requirementStatusLabel(requirement.status)}</span></div>) : <p>No approved requirements have been added.</p>}</section>
            <section><h3>Required evidence</h3>{aggregate.evidence_contracts.length ? aggregate.evidence_contracts.slice(0, 5).map((contract) => <div className="detail-row" key={contract.id}><strong>{contract.name}</strong><span>Required coverage: {Math.round(contract.minimum_coverage * 100)}%</span></div>) : <p>No evidence checks have been defined.</p>}</section>
          </div>}
        </article>;
      })}
    </section>
  </>;
}
