import type { ProgramSection } from "../appRouting";
import type { ProgramAggregate } from "../types";

type Reason = NonNullable<ProgramAggregate["current_state"]>["reasons"][number];
type Group = { key: string; title: string; action: string; section: ProgramSection; reasons: Reason[] };

export function ProgramAttention({ aggregate, onNavigate }: { aggregate: ProgramAggregate; onNavigate: (section: ProgramSection) => void }) {
  const reasons = aggregate.current_state?.reasons ?? [];
  const groups = new Map<string, Group>();
  for (const reason of reasons) {
    const assessment = reason.object_type === "EVIDENCE_CONTRACT";
    const source = reason.code.includes("SOURCE");
    const issues = reason.code === "OPEN_MATTERS";
    const key = assessment ? reason.code : source ? "sources" : issues ? "issues" : "other";
    const group = groups.get(key) ?? {
      key, reasons: [],
      title: assessment ? reason.code === "EVIDENCE_EXPIRED" ? "Assessments need renewal" : reason.code === "EVIDENCE_NOT_ASSESSED" ? "Assessments missing" : "Assessment gaps" : source ? "Source updates needed" : issues ? "Open issues" : "Other follow-up",
      action: assessment ? "Review assessments" : source ? "Review collection" : issues ? "Review issues" : "Review requirements",
      section: assessment ? "evidence-results" : source ? "monitoring" : issues ? "issues-actions" : "requirements-controls",
    } satisfies Group;
    group.reasons.push(reason);
    groups.set(key, group);
  }
  if (!reasons.length) return null;
  return <div className="program-attention" aria-label="Program follow-up">{[...groups.values()].map(group => <div className="program-attention-row" key={group.key}>
    <div><strong>{group.key === "issues" ? `${aggregate.current_state?.open_matter_count ?? "Unknown"} open issues` : `${group.reasons.length} · ${group.title}`}</strong>
      {group.key.startsWith("EVIDENCE_") && <p>{group.key === "EVIDENCE_EXPIRED" ? "Validity has ended for these internal assessments." : "Review the supporting information and record the assessment."}</p>}
      {group.key !== "issues" && <details><summary>Affected checks and requirements</summary><ul>{group.reasons.map((reason, index) => <li key={`${reason.code}-${reason.object_id ?? index}`}>{aggregate.evidence_contracts?.find(contract => contract.id === reason.object_id)?.name ?? reason.summary}</li>)}</ul></details>}
    </div><button type="button" className="text-button" onClick={() => onNavigate(group.section)}>{group.action} <span aria-hidden="true">→</span></button>
  </div>)}</div>;
}
