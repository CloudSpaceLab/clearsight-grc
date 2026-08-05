import type { Readiness } from "../types";
import { PremiumIllustration } from "./PremiumIllustration";

export function ReadinessPanel({ readiness }: { readiness: Readiness | null }) {
  if (!readiness) return <section className="readiness-card loading-card"><span>Loading continuous readiness…</span></section>;
  const dimensions: Array<[string, number | string]> = [
    ["Current", readiness.baseline_known ? readiness.dimensions.current : "—"],
    ["Aging", readiness.dimensions.aging],
    ["At risk", readiness.dimensions.at_risk],
    ["Unknown", readiness.dimensions.unknown],
    ["Routing blocked", readiness.dimensions.blocked_routing],
    ["Human judgment", readiness.dimensions.pending_human],
  ];
  return <section className="readiness-card" id="readiness-panel">
    <div className="readiness-copy">
      <span className="eyebrow">Continuous readiness</span>
      <div className="readiness-title"><h2>{readiness.status.replaceAll("_", " ")}</h2><span>{new Date(readiness.generated_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</span></div>
      <p>{readiness.baseline_known ? "ClearSight is continuously comparing requirements, evidence, source health and routing coverage." : "The active drift is visible, but a governed denominator has not yet been connected. ClearSight will not fabricate a current count."}</p>
      <div className="readiness-dimensions">{dimensions.map(([label, value]) => <div key={label}><strong>{value}</strong><span>{label}</span></div>)}</div>
    </div>
    <PremiumIllustration variant="readiness"/>
    <div className="readiness-actions"><h3>Recommended handling</h3>{readiness.recommended_actions.map((action) => <p key={action}>{action}</p>)}</div>
  </section>;
}
