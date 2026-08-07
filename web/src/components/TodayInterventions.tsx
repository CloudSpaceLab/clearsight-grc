import { EmptyState } from "./EmptyState";
import { WorkItemIcon } from "./WorkItemIcon";
import type { AttentionItem, InterventionClass, Readiness } from "../types";

type ConnectionState = "loading" | "live" | "sample" | "unavailable";
type ReadinessState = "loading" | "live" | "unavailable";

type Props = {
  items: AttentionItem[];
  connection: ConnectionState;
  readiness: Readiness | null;
  readinessState: ReadinessState;
  onOpenItem: (item: AttentionItem) => void;
};

export function TodayInterventions({ items, connection, readiness, readinessState, onOpenItem }: Props) {
  const heading = items.length === 1 ? "1 item requires your action" : `${items.length} items require your action`;
  const title = connection === "loading"
    ? "Loading assigned work"
    : connection === "unavailable"
      ? "Assigned work is unavailable"
      : heading;

  return <>
    <section className="intervention-brief" id="today-brief" aria-labelledby="intervention-heading">
      <header className="intervention-heading">
        <div><span className="eyebrow">Human intervention</span><h2 id="intervention-heading">{title}</h2><p>Review only the decisions, evidence exceptions and outcome checks that require your role.</p></div>
      </header>
      {connection === "loading"
        ? <div className="workspace-loading" aria-live="polite" aria-busy="true">Loading assigned work…</div>
        : connection === "unavailable"
          ? <EmptyState label="Assigned work" title="Assigned work could not be loaded" description="The service is unavailable. No current work count is shown."/>
          : items.length
            ? <div className="intervention-list" id="attention-list">{items.map((item) => <InterventionRow key={item.id} item={item} onOpen={onOpenItem}/>)}</div>
            : <div id="attention-list"><EmptyState label="Assigned work" title="No assigned items" description="There are no open reviews, approvals or evidence requests assigned to you in the connected scope."/></div>}
    </section>
    <ContinuousChecks readiness={readiness} state={readinessState}/>
  </>;
}

function InterventionRow({ item, onOpen }: { item: AttentionItem; onOpen: (item: AttentionItem) => void }) {
  const due = formatDue(item.due_at);
  const recommendation = item.recommendation?.proposed_action || item.primary_action;
  const conclusion = item.material_conclusion || item.why_now;
  const canOpen = Boolean(item.action_target_type && item.action_target_id);
  return <article className="intervention-row">
    <div className="intervention-main">
      <div className="intervention-kicker"><span className="intervention-kind"><WorkItemIcon type={item.type}/>{gateLabel(item.intervention_class, item.action_target_type)}</span><span>{item.state}</span><time>{due}</time></div>
      <h3>{item.title}</h3>
      <p className="intervention-conclusion">{conclusion}</p>
      {item.change_summary && item.change_summary !== conclusion && <p className="intervention-change"><strong>Changed:</strong> {item.change_summary}</p>}
      <div className="intervention-meta"><span>{item.scope}</span><span>{item.evidence}</span><span>{item.owner}</span></div>
    </div>
    <div className="intervention-next">
      <span>Prepared next step</span>
      <strong>{recommendation}</strong>
      {item.recommendation?.rationale && item.recommendation.rationale !== conclusion && <small>{item.recommendation.rationale}</small>}
      {canOpen ? <button className="primary-button" type="button" onClick={() => onOpen(item)}>Review and act</button> : <small>No linked record is available.</small>}
    </div>
  </article>;
}

function ContinuousChecks({ readiness, state }: { readiness: Readiness | null; state: ReadinessState }) {
  if (state === "loading") return <div className="continuous-checks quiet" aria-live="polite">Continuous checks are loading…</div>;
  if (state === "unavailable" || !readiness) return <div className="continuous-checks quiet"><strong>Continuous checks unavailable</strong><span>No readiness claim is shown while the service is unavailable.</span></div>;
  const dimensions = readiness.dimensions;
  const active = dimensions.aging + dimensions.at_risk + dimensions.unknown + dimensions.blocked_routing + dimensions.pending_human;
  const status = readiness.status.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
  return <details className="continuous-checks">
    <summary><div><span className="eyebrow">Continuous checks</span><strong>{active ? `${active} current exception${active === 1 ? "" : "s"}` : "No current exception is recorded"}</strong></div><div><span>{status}</span><time>Updated {new Date(readiness.generated_at).toLocaleString()}</time></div></summary>
    <div className="continuous-checks-detail">
      <dl>
        <div><dt>Current</dt><dd>{readiness.baseline_known ? dimensions.current : "—"}</dd></div>
        <div><dt>Aging</dt><dd>{dimensions.aging}</dd></div>
        <div><dt>At risk</dt><dd>{dimensions.at_risk}</dd></div>
        <div><dt>Unknown</dt><dd>{dimensions.unknown}</dd></div>
        <div><dt>Routing blocked</dt><dd>{dimensions.blocked_routing}</dd></div>
        <div><dt>Awaiting human review</dt><dd>{dimensions.pending_human}</dd></div>
      </dl>
      {readiness.recommended_actions.length > 0 && <div><h3>System-recommended follow-up</h3><ul>{readiness.recommended_actions.map((action) => <li key={action}>{action}</li>)}</ul></div>}
      {!readiness.baseline_known && <p>A complete governed population is not connected, so current coverage is not represented as complete.</p>}
    </div>
  </details>;
}

function gateLabel(value?: InterventionClass, target?: AttentionItem["action_target_type"]) {
  switch (value) {
    case "DECISION": return "Decision";
    case "AUTHORIZATION": return "Authorization";
    case "EVIDENCE_EXCEPTION": return "Evidence exception";
    case "ESCALATION": return "Escalation";
    case "VERIFICATION": return "Outcome check";
    case "EXTERNAL_REPRESENTATION": return "External approval";
    case "REVIEW": return "Review";
    default: return target === "EVIDENCE_REQUEST" ? "Evidence response" : "Review";
  }
}

function formatDue(value: string) {
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) return "No deadline";
  const date = new Date(parsed);
  if (date.getUTCFullYear() < 2000) return "No deadline";
  return new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" }).format(date);
}
