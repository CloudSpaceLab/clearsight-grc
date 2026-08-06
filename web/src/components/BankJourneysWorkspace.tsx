import { useCallback, useEffect, useState } from "react";
import { loadBankJourneys, loadEvidenceRequest, loadMatter, loadProgram } from "../api";
import type { EvidenceRequest, MatterAggregate, ProgramAggregate } from "../types";
import type { BankJourney, JourneyActionTarget } from "../verticalTypes";
import { EmptyState } from "./EmptyState";
import { PremiumIllustration } from "./PremiumIllustration";
import { WorkItemIcon } from "./WorkItemIcon";

type LoadState = "loading" | "live" | "unavailable";
type InspectorState =
  | { state: "closed" }
  | { state: "loading"; type: JourneyActionTarget; id: string; title: string }
  | { state: "unavailable"; type: JourneyActionTarget; id: string; title: string }
  | { state: "live"; type: "PROGRAM"; id: string; title: string; value: ProgramAggregate }
  | { state: "live"; type: "MATTER"; id: string; title: string; value: MatterAggregate }
  | { state: "live"; type: "EVIDENCE_REQUEST"; id: string; title: string; value: EvidenceRequest };

function statusClass(journey: BankJourney) {
  if (journey.status === "CLOSED" || journey.status_label === "Up to date") return "status-good";
  if (["OVERDUE", "GAP_IDENTIFIED"].includes(journey.status)) return "status-critical";
  if (["NOT_SET_UP", "STATUS_PENDING"].includes(journey.status)) return "status-neutral";
  return "status-warning";
}

function dueLabel(value?: string, status?: string) {
  if (status === "CLOSED") return "Completed";
  if (!value) return "No current deadline";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Deadline unavailable";
  return `Due ${date.toLocaleDateString(undefined, { day: "numeric", month: "short", year: "numeric" })}`;
}

function readableValue(value: unknown): string {
  if (value === null || value === undefined || value === "") return "Not recorded";
  if (Array.isArray(value)) return value.map(readableValue).join(", ");
  if (typeof value === "object") return Object.entries(value as Record<string, unknown>).map(([key, item]) => `${key.replaceAll("_", " ")}: ${readableValue(item)}`).join("; ");
  return String(value);
}

export function BankJourneysWorkspace() {
  const [items, setItems] = useState<BankJourney[]>([]);
  const [state, setState] = useState<LoadState>("loading");
  const [sample, setSample] = useState(false);
  const [inspector, setInspector] = useState<InspectorState>({ state: "closed" });
  const load = useCallback(async () => {
    setState("loading");
    try {
      const response = await loadBankJourneys();
      setItems(response.items);
      setSample(response.sample);
      setState("live");
    } catch {
      setItems([]);
      setSample(false);
      setState("unavailable");
    }
  }, []);
  useEffect(() => { void load(); }, [load]);

  async function openRecord(type: JourneyActionTarget, id: string, title: string) {
    setInspector({ state: "loading", type, id, title });
    try {
      if (type === "PROGRAM") {
        setInspector({ state: "live", type, id, title, value: await loadProgram(id) });
      } else if (type === "MATTER") {
        setInspector({ state: "live", type, id, title, value: await loadMatter(id) });
      } else {
        setInspector({ state: "live", type, id, title, value: await loadEvidenceRequest(id) });
      }
    } catch {
      setInspector({ state: "unavailable", type, id, title });
    }
  }

  if (state === "loading") return <section className="workspace-loading" aria-live="polite" aria-busy="true">Loading bank journeys…</section>;
  if (state === "unavailable") return <EmptyState label="Bank journeys" title="Bank journeys could not be loaded" description="The service is unavailable. No journey status is shown." action="Try again" onAction={() => void load()}/>;
  if (!items.length) return <EmptyState label="Bank journeys" title="No journeys are available in this scope" description="No configured bank journeys are visible to the signed-in user."/>;

  return <>
    <section className="journey-hero">
      <div>
        <span className="eyebrow">Bank compliance journeys</span>
        <h2>Move each requirement from source to confirmed outcome</h2>
        <p>Start with the recorded next action. Open the exact Program, issue or evidence request without needing to know internal system codes.</p>
        {sample && <div className="sample-notice" role="note"><strong>Reference data</strong><span>These Nigerian-bank records demonstrate product behaviour. The bank must review current legal and regulatory requirements before operational use.</span></div>}
      </div>
      <PremiumIllustration variant="routing"/>
    </section>
    <section className="journey-grid" aria-label="Bank compliance journeys">
      {items.map((journey) => {
        const progress = journey.total_steps ? Math.round((journey.completed_steps / journey.total_steps) * 100) : 0;
        return <article className={`journey-card ${journey.sensitive ? "restricted" : ""}`} key={journey.code}>
          <div className="journey-card-header">
            <span className="journey-icon"><WorkItemIcon type={journey.code}/></span>
            <div><span className="journey-owner">Accountable function · {journey.owner}</span><h3>{journey.title}</h3></div>
            {journey.sensitive && <span className="restricted-label">Restricted record</span>}
          </div>
          <p className="journey-summary">{journey.summary}</p>
          <div className="journey-state">
            <span className={`program-state ${statusClass(journey)}`}><strong>{journey.status_label}</strong></span>
            <span>{journey.completed_steps} of {journey.total_steps} stages complete</span>
          </div>
          <div className="journey-progress" role="progressbar" aria-label={`${journey.title} completion`} aria-valuemin={0} aria-valuemax={journey.total_steps} aria-valuenow={journey.completed_steps}><span style={{ width: `${progress}%` }}/></div>
          <section className="journey-next" aria-label={`Next action for ${journey.title}`}>
            <span>Next action</span>
            <strong>{journey.next_action}</strong>
            <small>{dueLabel(journey.due_at, journey.status)}</small>
            {journey.action_available && journey.action_target_type && journey.action_target_id
              ? <button className="primary-button journey-primary-action" type="button" onClick={() => void openRecord(journey.action_target_type!, journey.action_target_id!, journey.title)}>{journey.action_label || "Open record"}</button>
              : <p className="journey-action-unavailable">{journey.action_unavailable_reason || "Ask the accountable function to configure the required record."}</p>}
          </section>
          <details className="journey-details">
            <summary aria-label={`View stages and sources for ${journey.title}`}>Stages and supporting records</summary>
            <ol>{journey.steps.map((step) => <li className={step.complete ? "complete" : "open"} key={step.code}><span aria-hidden="true">{step.complete ? "✓" : "○"}</span><span>{step.label}</span></li>)}</ol>
            <div className="journey-sources"><strong>Recorded sources</strong>{journey.source_names.length ? <ul>{journey.source_names.map((source) => <li key={source}>{source}</li>)}</ul> : <p>No source is recorded.</p>}</div>
            <div className="journey-record-links" aria-label={`Linked records for ${journey.title}`}>
              {journey.program_id && <button type="button" onClick={() => void openRecord("PROGRAM", journey.program_id!, journey.title)}>Open linked Program</button>}
              {journey.matter_id && <button type="button" onClick={() => void openRecord("MATTER", journey.matter_id!, journey.title)}>Open linked issue</button>}
              {journey.evidence_request_id && <button type="button" onClick={() => void openRecord("EVIDENCE_REQUEST", journey.evidence_request_id!, journey.title)}>Open linked evidence request</button>}
              {!journey.program_id && !journey.matter_id && !journey.evidence_request_id && <span>No linked record</span>}
            </div>
          </details>
        </article>;
      })}
    </section>
    {inspector.state !== "closed" && <div className="journey-inspector-backdrop" onMouseDown={() => setInspector({ state: "closed" })}>
      <aside className="journey-inspector" role="dialog" aria-modal="true" aria-label={`Connected record for ${inspector.title}`} onMouseDown={(event) => event.stopPropagation()}>
        <button className="panel-close" type="button" onClick={() => setInspector({ state: "closed" })} aria-label="Close connected record">×</button>
        {inspector.state === "loading" && <p aria-live="polite" aria-busy="true">Loading the connected record…</p>}
        {inspector.state === "unavailable" && <div className="inline-error"><h2>Record could not be opened</h2><p>Your access may have changed, or the record is temporarily unavailable.</p><button className="secondary-button" type="button" onClick={() => void openRecord(inspector.type, inspector.id, inspector.title)}>Try again</button></div>}
        {inspector.state === "live" && <InspectorContent inspector={inspector}/>} 
      </aside>
    </div>}
  </>;
}

function InspectorContent({ inspector }: { inspector: Extract<InspectorState, { state: "live" }> }) {
  if (inspector.type === "PROGRAM") {
    const value = inspector.value;
    return <div className="inspector-content"><span className="eyebrow">Program</span><h2>{value.program.name}</h2><p>{value.state_label || "Status not assessed"}</p><section><h3>Why this status</h3>{value.current_state?.reasons?.length ? <ul>{value.current_state.reasons.map((reason) => <li key={`${reason.code}-${reason.object_id ?? ""}`}>{reason.summary}</li>)}</ul> : <p>No current status reason is recorded.</p>}</section><section><h3>Requirements and evidence</h3><p>{value.requirements.length} requirements · {value.evidence_contracts.length} evidence checks · {value.current_state?.open_matter_count ?? 0} open issues</p></section></div>;
  }
  if (inspector.type === "EVIDENCE_REQUEST") {
    const value = inspector.value;
    return <div className="inspector-content"><span className="eyebrow">Evidence request</span><h2>{value.title}</h2><p>{value.purpose}</p><dl><div><dt>Why you</dt><dd>{value.why_you}</dd></div><div><dt>Deadline</dt><dd>{dueLabel(value.deadline)}</dd></div><div><dt>Status</dt><dd>{value.status.replaceAll("_", " ").toLowerCase()}</dd></div></dl><section><h3>What to provide</h3><ul>{value.fields.map((field) => <li key={field.id}><strong>{field.label}</strong>{field.description ? ` — ${field.description}` : ""}{field.required ? " (required)" : ""}</li>)}</ul></section></div>;
  }
  const value = inspector.value;
  const facts = Object.entries(value.matter.known_facts ?? {});
  return <div className="inspector-content"><span className="eyebrow">Issue or change</span><h2>{value.matter.title}</h2><p>{value.matter.summary}</p><dl><div><dt>Current state</dt><dd>{value.status_label}</dd></div><div><dt>Next action</dt><dd>{value.next_action}</dd></div><div><dt>Deadline</dt><dd>{dueLabel(value.matter.due_at, value.matter.status)}</dd></div></dl><section><h3>Recorded facts</h3>{facts.length ? <ul>{facts.map(([key, item]) => <li key={key}><strong>{key.replaceAll("_", " ")}</strong>: {readableValue(item)}</li>)}</ul> : <p>No facts are recorded.</p>}</section>{value.matter.missing_facts?.length ? <section><h3>Information still needed</h3><ul>{value.matter.missing_facts.map((fact, index) => <li key={`${index}-${readableValue(fact)}`}>{readableValue(fact)}</li>)}</ul></section> : null}<section><h3>Work and outcome</h3><p>{value.actions.length} actions · {value.verification_contracts.length} outcome checks</p>{!value.closure.ready && value.closure.reasons.length > 0 && <ul>{value.closure.reasons.map((reason) => <li key={reason}>{reason}</li>)}</ul>}</section></div>;
}
