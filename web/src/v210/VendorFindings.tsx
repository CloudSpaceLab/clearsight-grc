import type { VendorAssessmentFinding } from "../vendorAssessmentTypes";

export function VendorFindings({ matters, mayCreate }: { matters?: VendorAssessmentFinding[]; mayCreate: boolean }) {
  const findings = matters?.filter((matter) => matter.type === "VENDOR_DEFICIENCY");
  return <section className="v-panel v-vendor-findings"><h2>Findings</h2>
    {!findings ? <p>Findings unavailable.</p> : findings.length === 0 ? <p>No linked findings returned for this assessment.</p> : <ul>{findings.map((finding) => <li key={finding.matter_id}><strong>{finding.title}</strong><span>{statusLabel(finding.status)}</span></li>)}</ul>}
    {mayCreate && <p>Finding creation unavailable.</p>}
  </section>;
}

function statusLabel(value: string) { const text = value.toLowerCase().replaceAll("_", " "); return text.charAt(0).toUpperCase() + text.slice(1); }
