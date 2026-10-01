import { EmptyState } from "./EmptyState";
import { WorkItemIcon } from "./WorkItemIcon";
import type { AttentionItem, InterventionClass } from "../types";

export type AssignedWorkState = "loading" | "live" | "unavailable";
type FocusedAttentionItem = AttentionItem & { action_target_sub_id?: string };

export type AssignedWorkQueueProps = {
  items: AttentionItem[];
  state: AssignedWorkState;
  onOpenItem: (item: AttentionItem) => void;
  onInspectAuthority?: (item: AttentionItem) => void;
  onRetry?: () => void;
  headingLevel?: "h2" | "h3";
};

export function AssignedWorkQueue({ items, state, onOpenItem, onInspectAuthority, onRetry = () => window.location.reload(), headingLevel = "h2" }: AssignedWorkQueueProps) {
  const count = items.length === 1 ? "1 item needs your action" : `${items.length} items need your action`;
  const title = state === "loading" ? "Loading assigned work" : state === "unavailable" ? "Assigned work is unavailable" : count;
  const Heading = headingLevel;

  return <section className="intervention-brief" aria-labelledby="assigned-work-heading">
    <header className="intervention-heading">
      <div><span className="eyebrow">Assigned work</span><Heading id="assigned-work-heading">{title}</Heading></div>
    </header>
    {state === "loading"
      ? <div className="workspace-loading" aria-live="polite" aria-busy="true">Loading assigned work…</div>
      : state === "unavailable"
        ? <EmptyState kind="unavailable" label="Assigned work" title="Assigned work could not be loaded" description="Retry." action="Try again" onAction={onRetry}/>
        : items.length
          ? <div className="intervention-list" id="attention-list">{items.map((item) => <InterventionRow key={item.id} item={item} onOpen={onOpenItem} onInspectAuthority={onInspectAuthority}/>)}</div>
          : <div id="attention-list"><EmptyState label="Assigned work" title="Nothing needs your action right now" description="No open assigned work."/></div>}
  </section>;
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
      {canOpen ? <button className="primary-button" type="button" onClick={() => openItem(item, onOpen)}>{openLabel(targetType, item.action_target_id)}</button> : <small>No linked record is available.</small>}
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
  if (focused.action_target_type === "CONFIGURE" && focused.action_target_id) {
    window.location.hash = `configure/${encodeURIComponent(focused.action_target_id)}`;
    return;
  }
  fallback(item);
}

function VerificationContext({ item }: { item: AttentionItem }) {
  const verification = item.verification;
  if (!verification) return null;
  const timing = formatCheckTime(verification.next_check_at);
  return <details className="intervention-verification">
    <summary>Outcome check details</summary>
    <dl>
      <div><dt>Expected outcome</dt><dd>{verification.expected_outcome || "Not provided"}</dd></div>
      <div><dt>Method</dt><dd>{verification.method || "Outcome review"}</dd></div>
      <div><dt>Check</dt><dd>{timing}</dd></div>
    </dl>
  </details>;
}

function openLabel(target?: string, targetID?: string) {
  if (target === "PROGRAM") return "Open program";
  if (target === "MATTER") return "Open issue";
  if (target === "EVIDENCE_REQUEST") return "Open request";
  if (target === "DOCUMENT_IMPORT") return "Open proposal";
  if (target === "CONFIGURE") return targetID === "operations" ? "Open system operations" : "Open access settings";
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

function formatCheckTime(value?: string) {
  const parsed = Date.parse(value ?? "");
  if (!Number.isFinite(parsed) || new Date(parsed).getUTCFullYear() < 2000) return "Not scheduled";
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(parsed));
}
