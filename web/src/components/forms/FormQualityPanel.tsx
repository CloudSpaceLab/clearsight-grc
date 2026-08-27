import type { FormScoringMode } from "../../formsTypes";
import { hasRequiredSignOff, type AuthoringField, type AuthoringSection } from "./formAuthoring";
import type { FormQualityIssue } from "./formQuality";

type Props = { scoringMode: FormScoringMode; sections: AuthoringSection[]; fields: AuthoringField[]; issues: FormQualityIssue[]; onAddRequiredSignOff: () => void };

export function FormQualityPanel({ scoringMode, sections, fields, issues, onAddRequiredSignOff }: Props) {
  const blocking = issues.filter((issue) => issue.blocking);
  const compliance = scoringMode === "COMPLIANCE" ? complianceState(sections, fields) : [];
  const signed = hasRequiredSignOff(fields);
  return <aside className="form-quality-panel" aria-labelledby="form-quality-title">
    <div className="form-quality-heading"><div><span className="eyebrow">Quality gate</span><h5 id="form-quality-title">Approval readiness</h5></div><strong className={blocking.length ? "quality-count blocking" : "quality-count ready"}>{blocking.length ? `${blocking.length} blocking` : "Ready"}</strong></div>
    {blocking.length ? <ul className="form-quality-issues">{blocking.map((issue) => <li key={issue.id}>{issue.message}</li>)}</ul> : <p className="form-quality-ready">The current draft satisfies the deterministic contract checks required before approval.</p>}
    {compliance.length > 0 && <div className="form-quality-compliance" aria-label="Compliance weight allocation">{compliance.map((item) => <div key={item.id}><div><span>{item.label}</span><strong>{item.value}%</strong></div><progress max={100} value={Math.min(100, item.value)} aria-label={`${item.label} ${item.value}%`}/></div>)}</div>}
    <div className="form-quality-signoff"><div><strong>{signed ? "Required sign-off included" : "No required sign-off yet"}</strong><span>{signed ? "A required attestation or signature is present." : "Add one when the responder must explicitly attest to the submitted information."}</span></div>{!signed && <button className="secondary-button" type="button" onClick={onAddRequiredSignOff}>Add required sign-off</button>}</div>
  </aside>;
}

function complianceState(sections: AuthoringSection[], fields: AuthoringField[]) {
  const result: Array<{ id: string; label: string; value: number }> = [];
  let sectionWeight = 0;
  for (const section of sections) {
    const scored = fields.filter((field) => field.section_id === section.id && field.scoring);
    if (!scored.length) continue;
    result.push({ id: `fields:${section.id}`, label: `${section.title || "Section"} question allocation`, value: scored.reduce((sum, field) => sum + (field.scoring?.weight ?? 0), 0) });
    sectionWeight += section.weight ?? 0;
  }
  result.unshift({ id: "sections", label: "Scored section allocation", value: sectionWeight });
  return result;
}
