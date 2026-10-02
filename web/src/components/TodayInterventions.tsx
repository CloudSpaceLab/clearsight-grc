import { AssignedWorkQueue } from "./AssignedWorkQueue";
import type { AttentionItem, Readiness } from "../types";

type ConnectionState = "loading" | "live" | "unavailable";
type ReadinessState = "loading" | "live" | "unavailable";

type Props = {
  items: AttentionItem[];
  connection: ConnectionState;
  readiness: Readiness | null;
  readinessState: ReadinessState;
  onOpenItem: (item: AttentionItem) => void;
  onInspectAuthority?: (item: AttentionItem) => void;
  onRetry?: () => void;
};

export function TodayInterventions({ items, connection, readiness, readinessState, onOpenItem, onInspectAuthority, onRetry }: Props) {
  return <>
    <div id="today-brief">
      <AssignedWorkQueue items={items} state={connection} onOpenItem={onOpenItem} onInspectAuthority={onInspectAuthority} onRetry={onRetry}/>
    </div>
    <StatusChecks readiness={readiness} state={readinessState}/>
  </>;
}

function StatusChecks({ readiness, state }: { readiness: Readiness | null; state: ReadinessState }) {
  if (state === "loading") return <div className="continuous-checks quiet" aria-live="polite">Status checks are loading…</div>;
  if (state === "unavailable" || !readiness) return <div className="continuous-checks quiet"><strong>Status checks unavailable</strong><span>Retry.</span></div>;
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
        <div><dt>Awaiting review</dt><dd>{dimensions.pending_human}</dd></div>
      </dl>
      {readiness.recommended_actions.length > 0 && <div><h3>Suggested follow-up</h3><ul>{readiness.recommended_actions.map((action) => <li key={action}>{action}</li>)}</ul></div>}
      {!readiness.baseline_known && <p>Coverage incomplete.</p>}
    </div>
  </details>;
}
