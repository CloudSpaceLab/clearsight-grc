import { EmptyState } from "./EmptyState";
import { WorkItemIcon } from "./WorkItemIcon";
import type { AttentionItem, InterventionClass, Readiness } from "../types";

type ConnectionState = "loading" | "live" | "sample" | "unavailable";
type ReadinessState = "loading" | "live" | "unavailable";
type FocusedAttentionItem = AttentionItem & { action_target_sub_id?: string };

type Props = {
  items: AttentionItem[];
  connection: ConnectionState;
  readiness: Readiness | null;
  readinessState: ReadinessState;
  onOpenItem: (item: AttentionItem) => void;
  onInspectAuthority?: (item: AttentionItem) => void;
};

export function TodayInterventions({ items, connection, readiness, readinessState, onOpenItem, onInspectAuthority }: Props) {
  const heading = items.length === 1 ? "1 item needs your action" : `${items.length} items need your action`;
  const title = connection === "loading" ? "Loading Today" : connection === "unavailable" ? "Today is unavailable" : heading;

  return <>
    <section className="intervention-brief" id="today-brief" aria-labelledby="intervention-heading">
      <header className="intervention-heading">
        <div><span className="eyebrow">Today</span><h2 id="intervention-heading">{title}</h2><p>Reviews, approvals and evidence requests assigned to you.</p></div>
      </header>
      {connection === "loading"
        ? <div className="workspace-loading" aria-live="polite" aria-busy="true">Loading Today…</div>
        : connection === "unavailable"
          ? <EmptyState kind="unavailable" label="Today" title="Today could not be loaded" description="Try again before relying on this list."/>
          : items.length
            ? <div className="intervention-list" id="attention-list">{items.map((item) => <InterventionRow key={item.id} item={item} onOpen={onOpenItem} onInspectAuthority={onInspectAuthority}/>)}</div>
            : <div id="attention-list"><EmptyState label="Today" title="Nothing needs your action right now" description="There are no open reviews, approvals or evidence requests assigned to you in this scope."/></div>}
    </section>
    <StatusChecks readiness={readiness} state={readinessState}/>
  </>;
}

function InterventionRow({ item, onOpen, onInspectAuthority }: { item: AttentionItem; onOpen: (item: AttentionItem) => void; onInspectAuthority?: (item: AttentionItem) => void }) {
  const due = formatDue(item.due_at);
  const nextAction = item.recommendation?.proposed_action || item.primary_action;
  const nextActionLabel = item.recommendation ? "Recommended action" : "Next action";
  const conclusion = item.material_conclusion || item.why_now;
  const targetType = item.action_target_type as string | undefined;
  const canOpen = Boolean(targetType && item.action_target_id);
  const canInspectAuthority = Boolean(canOpen && targetType !== "DOCUMENT_IMPORT" && item.authority && onInspectAuthority);
  return <article className="intervention-row">
    <div className="intervention-main">
      <div className="intervention-kicker"><span className="intervention-kind"><WorkItemIcon type={item.type}/>{gateLabel(item.intervention_class, targetType)}</span><span>{item.state}</span><time>{due}</time></div>
      <h3>{item.title}</h3>
      <p className="intervention-conclusion">{conclusion}</p>
      {item.change_summary && item.change_summary !== conclusion && <p className="intervention-change"><strong>Changed:</strong> {item.change_summary}</p>}
      <div className="intervention-meta"><span>{item.scope}</span><span>{item.evidence}</span><span>{item.owner}</span></div>
    </div>
    <div className="intervention-next">
      <span>{nextActionLabel}</span>
      <strong>{nextAction}</strong>
      {item.recommendation?.rationale && item.recommendation.rationale !== conclusion && <small>{item.recommendation.rationale}</small>}
      {item.verification && <VerificationContext item={item}/>} 
      {canOpen ? <button className="primary-button" type="button" onClick={() => openItem(item, onOpen)}>{openLabel(targetType)}</button> : <small>No linked record is available.</small>}
      {canInspectAuthority && <button className="text-button" type="button" onClick={() => onInspectAuthority?.(item)}>Check authority</button>}
    </div>
  </article>;
}

function openItem(item: AttentionItem, fallback: (item: AttentionItem) => void) {
  const focused = item as FocusedAttentionItem;
  if ((focused.action_target_type as string | undefined) === "DOCUMENT_IMPORT" && focused.action_target_id) {
    const proposal = focused.action_target_sub_id ? `/${encodeURIComponent(focused.action_target_sub_id)}` : "";
    window.location.hash = `imports/${encodeURIComponent(focused.action_target_id)}${proposal}`;
    return;
  }
  fallback(item);
}

function VerificationContext({ item }: { item: AttentionItem }) {
  const verification = item.verification;
  if (!verification) return null;
  const timing = verification.next_check_at ? formatCheckTime(verification.next_check_at) : "Ready now";
  return <details className="intervention-verification">
    <summary>Outcome check details</summary>
    <dl>
      <div><dt>Expected outcome</dt><dd>{verification.expected_outcome || "Not provided"}</dd></div>
      <div><dt>Method</dt><dd>{verification.method || "Outcome review"}</dd></div>
      <div><dt>Check</dt><dd>{timing}</dd></div>
    </dl>
  </details>;
}

function StatusChecks({ readiness, state }: { readiness: Readiness | null; state: ReadinessState }) {
  if (state === "loading") return <div className="continuous-checks quiet" aria-live="polite">Status checks are loading…</div>;
  if (state === "unavailable" || !readiness) return <div className="continuous-checks quiet"><strong>Status checks unavailable</strong><span>The latest status checks could not be loaded. Try again before reviewing readiness.</span></div>;
  const dimensions = readiness.dimensions;
  const active = dimensions.aging + dimensions.at_risk + dimensions.unknown + dimensions.blocked_routing + dimensions.pending_human;
  const status = readiness.status.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
  const summary = readiness.baseline_known
    ? active ? `${active} current exception${active === 1 ? "" : "s"}` : "No current exceptions recorded"
    : active ? `${active} known exception${active === 1 ? "" : "s"}; coverage is incomplete` : "Coverage is incomplete";
  return <details className="continuous-checks">
    <summary><div><span className="eyebrow">Status checks</span><strong>{summary}</strong></div><div><span>{readiness.baseline_known ? status : "Coverage incomplete"}</span><time>Updated {new Date(readiness.generated_at).toLocaleString()}</time></div></summary>
    <div className="continuous-checks-detail">
      <dl>
        <div><dt>Current</dt><dd>{readiness.baseline_known ? dimensions.current : "—"}</dd></div>
        <div><dt>Aging</dt><dd>{dimensions.aging}</dd></div>
        <div><dt>At risk</dt><dd>{dimensions.at_risk}</dd></div>
        <div><dt>Unknown</dt><dd>{dimensions.unknown}</dd></div>
        <div><dt>Routing blocked</dt><dd>{dimensions.blocked_routing}</dd></div>
        <div><dt>Waiting for review</dt><dd>{dimensions.pending_human}</dd></div>
      </dl>
      {readiness.recommended_actions.length > 0 && <div><h3>Suggested follow-up</h3><ul>{readiness.recommended_actions.map((action) => <li key={action}>{action}</li>)}</ul></div>}
      {!readiness.baseline_known && <p>Coverage is incomplete, so these counts are not a complete view of compliance status.</p>}
    </div>
  </details>;
}

function openLabel(target?: string) {
  if (target === "PROGRAM") return "Open program";
  if (target === "MATTER") return "Open issue";
  if (target === "EVIDENCE_REQUEST") return "Open request";
  if (target === "DOCUMENT_IMPORT") return "Open proposal";
  return "Open item";
}

function gateLabel(value?: InterventionClass, target?: string) {
  switch (value) {
    case "DECISION": return "Decision";
    case "AUTHORIZATION": return "Approval";
    case "EVIDENCE_EXCEPTION": return "Evidence needed";
    case "ESCALATION": return "Escalated";
    case "VERIFICATION": return "Outcome check";
    case "EXTERNAL_REPRESENTATION": return "External response";
    case "REVIEW": return "Review";
    default: return target === "EVIDENCE_REQUEST" ? "Evidence request" : "Review";
  }
}

function formatDue(value: string) {
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) return "No deadline";
  const date = new Date(parsed);
  if (date.getUTCFullYear() < 2000) return "No deadline";
  return new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" }).format(date);
}

function formatCheckTime(value: string) {
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) return "Ready now";
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(parsed));
}
