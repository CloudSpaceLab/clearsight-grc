import type { Readiness } from "../types";
import { PremiumIllustration } from "./PremiumIllustration";

export function ReadinessPanel({ readiness }: { readiness: Readiness | null }) {
  if (!readiness) return <section className="readiness-card loading-card"><span>Loading readiness data…</span></section>;
  const dimensions: Array<[string, number | string]> = [
    ["Current", readiness.baseline_known ? readiness.dimensions.current : "—"],
    ["Aging", readiness.dimensions.aging],
    ["At risk", readiness.dimensions.at_risk],
    ["Unknown", readiness.dimensions.unknown],
    ["Routing blocked", readiness.dimensions.blocked_routing],
    ["Awaiting review", readiness.dimensions.pending_human],
  ];
  const summary = readiness.baseline_known
    ? "Requirement, evidence, source-health and routing checks are included in the current assessment."
    : "Active exceptions are shown below. A complete governed population has not been connected, so no current count is displayed.";
  return <section className="readiness-card" id="readiness-panel">
    <div className="readiness-copy">
      <span className="eyebrow">Readiness</span>
      <div className="readiness-title"><h2>{readiness.status.replaceAll("_", " ")}</h2><span>Updated {new Date(readiness.generated_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</span></div>
      <p>{summary}</p>
      <div className="readiness-dimensions">{dimensions.map(([label, value]) => <div key={label}><strong>{value}</strong><span>{label}</span></div>)}</div>
    </div>
    <PremiumIllustration variant="readiness"/>
    <div className="readiness-actions"><h3>Required actions</h3>{readiness.recommended_actions.length ? readiness.recommended_actions.map((action) => <p key={action}>{action}</p>) : <p>No action is required.</p>}</div>
  </section>;
}
