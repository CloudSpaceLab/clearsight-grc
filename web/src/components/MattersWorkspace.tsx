import { useMemo, useState } from "react";
import type { MatterAggregate } from "../types";
import { EmptyState } from "./EmptyState";
import { PremiumIllustration } from "./PremiumIllustration";

type Props = { matters: MatterAggregate[]; state: "loading" | "live" | "unavailable" };

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
    case "IMPLEMENTED": return "Completed; outcome not yet confirmed";
    case "VERIFIED": return "Outcome confirmed";
    case "CANCELLED": return "Cancelled";
    default: return value.replaceAll("_", " ").toLowerCase();
  }
}

function priorityLabel(value: number) {
  if (value >= 5) return "Critical";
  if (value === 4) return "High";
  if (value === 3) return "Medium";
  return "Low";
}

export function MattersWorkspace({ matters, state }: Props) {
  const [openID, setOpenID] = useState<string | null>(null);
  const summary = useMemo(() => ({
    decisions: matters.filter((item) => item.matter.status === "DECISION_REQUIRED").length,
    overdue: matters.filter((item) => item.matter.due_at && Date.parse(item.matter.due_at) < Date.now() && !["CLOSED", "CANCELLED"].includes(item.matter.status)).length,
    checking: matters.filter((item) => item.matter.status === "VERIFICATION").length,
  }), [matters]);

  if (state === "loading") return <section className="workspace-loading">Loading issues and changes…</section>;
  if (state === "unavailable") return <EmptyState label="Issues and changes" title="Issues could not be loaded" description="Try again when the service is available. Existing records have not been changed."/>;
  if (!matters.length) return <EmptyState label="Issues and changes" title="No open issues or changes" description="There are no recorded changes, gaps, findings, exceptions or response items in the current scope."/>;

  return <>
    <section className="matter-hero"><div><span className="eyebrow">Issues and changes</span><h2>{matters.length} open issue{matters.length === 1 ? " or change" : "s or changes"}</h2><p>Specific changes, gaps, findings, requests and exceptions that need a decision, action or outcome check.</p></div><PremiumIllustration variant="routing"/></section>
    <section className="matter-summary" aria-label="Matter summary"><div><span>Decision needed</span><strong>{summary.decisions}</strong><small>Waiting for an authorized decision</small></div><div><span>Overdue</span><strong>{summary.overdue}</strong><small>Past the recorded due date</small></div><div><span>Confirming outcome</span><strong>{summary.checking}</strong><small>Work is complete; the result still needs confirmation</small></div></section>
    <section className="matter-list">{matters.map((aggregate) => {
      const matter = aggregate.matter;
      const isOpen = openID === matter.id;
      return <article className="matter-card" key={matter.id}>
        <button type="button" className="matter-card-main" aria-expanded={isOpen} onClick={() => setOpenID(isOpen ? null : matter.id)}>
          <span className="matter-icon"><MatterIcon type={matter.type}/></span>
          <span className="matter-primary"><span className="matter-kicker">{aggregate.type_label} · {matter.reference}</span><strong>{matter.title}</strong><small>{matter.summary}</small></span>
          <span className="matter-meta"><span>{priorityLabel(matter.priority)} priority</span><span>{matter.due_at ? `Due ${new Date(matter.due_at).toLocaleDateString()}` : "No due date"}</span></span>
          <span className={`matter-status status-${matter.status.toLowerCase().replaceAll("_", "-")}`}><strong>{aggregate.status_label}</strong><small>{aggregate.next_action}</small></span>
          <span className="expand-indicator" aria-hidden="true">{isOpen ? "−" : "+"}</span>
        </button>
        {isOpen && <div className="matter-detail">
          <section><h3>What we know</h3><dl><div><dt>Known facts</dt><dd>{Object.keys(matter.known_facts ?? {}).length || "None recorded"}</dd></div><div><dt>Missing facts</dt><dd>{matter.missing_facts?.length ?? 0}</dd></div><div><dt>Conflicting information</dt><dd>{matter.contradictions?.length ?? 0}</dd></div></dl></section>
          <section><h3>Actions</h3>{aggregate.actions.length ? aggregate.actions.map((action) => <div className="detail-row" key={action.id}><strong>{action.title}</strong><span>{actionStatusLabel(action.status)}</span></div>) : <p>No actions have been recorded.</p>}</section>
          <section><h3>Result checks</h3>{aggregate.verification_contracts.length ? aggregate.verification_contracts.map((contract) => <div className="detail-row" key={contract.id}><strong>{contract.expected_outcome}</strong><span>{aggregate.verification_results.some((result) => result.contract_id === contract.id && result.result === "PASS") ? "Confirmed" : "Not checked yet"}</span></div>) : <p>No result check has been defined.</p>}{!aggregate.closure.ready && aggregate.closure.reasons.length > 0 && <div className="closure-note"><strong>Before this can close</strong><ul>{aggregate.closure.reasons.map((reason) => <li key={reason}>{reason}</li>)}</ul></div>}</section>
        </div>}
      </article>;
    })}</section>
  </>;
}
