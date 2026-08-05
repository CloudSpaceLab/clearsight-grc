import { useCallback, useEffect, useState } from "react";
import { loadBankJourneys } from "../api";
import type { BankJourney } from "../verticalTypes";
import { EmptyState } from "./EmptyState";
import { PremiumIllustration } from "./PremiumIllustration";
import { WorkItemIcon } from "./WorkItemIcon";

type LoadState = "loading" | "live" | "unavailable";

function statusClass(journey: BankJourney) {
  if (journey.status === "CLOSED" || journey.status_label === "Up to date") return "status-good";
  if (["OVERDUE", "GAP_IDENTIFIED"].includes(journey.status)) return "status-critical";
  if (["NOT_SET_UP", "STATUS_PENDING"].includes(journey.status)) return "status-neutral";
  return "status-warning";
}

function dueLabel(value?: string) {
  if (!value) return "No current deadline";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Deadline unavailable";
  return `Due ${date.toLocaleDateString(undefined, { day: "numeric", month: "short", year: "numeric" })}`;
}

export function BankJourneysWorkspace() {
  const [items, setItems] = useState<BankJourney[]>([]);
  const [state, setState] = useState<LoadState>("loading");
  const [sample, setSample] = useState(false);
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

  if (state === "loading") return <section className="workspace-loading">Loading bank journeys…</section>;
  if (state === "unavailable") return <EmptyState label="Bank journeys" title="Bank journeys could not be loaded" description="The service is unavailable. No journey status is shown." action="Try again" onAction={() => void load()}/>;
  if (!items.length) return <EmptyState label="Bank journeys" title="No journeys are available in this scope" description="No configured bank journeys are visible to the signed-in user."/>;

  return <>
    <section className="journey-hero">
      <div>
        <span className="eyebrow">Connected bank workflows</span>
        <h2>Compliance work from source to confirmed outcome</h2>
        <p>Each journey shows the records already connected, completed stages and next recorded action.</p>
        {sample && <div className="sample-notice"><strong>Reference data</strong><span>These Nigerian-bank records demonstrate product behaviour. The bank must review current legal and regulatory requirements before operational use.</span></div>}
      </div>
      <PremiumIllustration variant="routing"/>
    </section>
    <section className="journey-grid" aria-label="Bank reference journeys">
      {items.map((journey) => <article className={`journey-card ${journey.sensitive ? "restricted" : ""}`} key={journey.code}>
        <div className="journey-card-header">
          <span className="journey-icon"><WorkItemIcon type={journey.code}/></span>
          <div><span className="journey-owner">{journey.owner}</span><h3>{journey.title}</h3></div>
          {journey.sensitive && <span className="restricted-label">Restricted</span>}
        </div>
        <p>{journey.summary}</p>
        <div className="journey-status-row">
          <span className={`program-state ${statusClass(journey)}`}><strong>{journey.status_label}</strong></span>
          <span>{journey.completed_steps} of {journey.total_steps} stages complete</span>
        </div>
        <div className="journey-progress" aria-label={`${journey.completed_steps} of ${journey.total_steps} stages complete`}><span style={{ width: `${journey.total_steps ? Math.round((journey.completed_steps / journey.total_steps) * 100) : 0}%` }}/></div>
        <div className="journey-next"><span>Next action</span><strong>{journey.next_action}</strong><small>{dueLabel(journey.due_at)}</small></div>
        <details className="journey-details">
          <summary>View journey record</summary>
          <ol>{journey.steps.map((step) => <li className={step.complete ? "complete" : "open"} key={step.code}><span aria-hidden="true">{step.complete ? "✓" : "○"}</span><span>{step.label}</span></li>)}</ol>
          <div className="journey-sources"><strong>Recorded sources</strong>{journey.source_names.length ? <ul>{journey.source_names.map((source) => <li key={source}>{source}</li>)}</ul> : <p>No source is recorded.</p>}</div>
          <div className="journey-record-links"><span>{journey.program_id ? "Program record available" : journey.matter_id ? "Issue record available" : "No linked record"}</span>{journey.evidence_request_id && <span>Evidence request available</span>}</div>
        </details>
      </article>)}
    </section>
  </>;
}
