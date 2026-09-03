import { useEffect, useState } from "react";
import { activateVendorRelationship, loadVendorActivation } from "../vendorApi";
import type { VendorActivationResult, VendorRelationship } from "../vendorTypes";
import { apiErrorKind } from "../http";

export function VendorActivationPanel({ relationship, onActivated }: { relationship: VendorRelationship; onActivated: (relationship: VendorRelationship) => void }) {
  const [state, setState] = useState<"loading" | "ready" | "unavailable">(relationship.status === "ACTIVE" ? "ready" : "loading");
  const [eligibility, setEligibility] = useState<VendorActivationResult>();
  const [rationale, setRationale] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    let current = true;
    setError("");
    setRationale("");
    if (relationship.status === "ACTIVE") {
      setEligibility(undefined);
      setState("ready");
      return () => { current = false; };
    }
    setState("loading");
    void loadVendorActivation(relationship.id).then((value) => {
      if (!current) return;
      setEligibility(value);
      setState("ready");
    }).catch((caught) => {
      if (!current) return;
      setEligibility(undefined);
      setState("unavailable");
      setError(apiErrorKind(caught) === "conflict" ? "No approved activation policy applies to this legal entity at the current time." : "Activation checks could not be loaded. The relationship remains unchanged.");
    });
    return () => { current = false; };
  }, [relationship.id, relationship.version, relationship.status]);

  async function activate() {
    if (!eligibility?.eligible || rationale.trim().length < 20) return;
    setBusy(true);
    setError("");
    try {
      const result = await activateVendorRelationship(relationship.id, {
        expected_version: relationship.version,
        intended_effective_at: new Date().toISOString(),
        rationale: rationale.trim(),
      });
      setEligibility(result);
      onActivated(result.relationship);
    } catch (caught) {
      const kind = apiErrorKind(caught);
      setError(kind === "conflict" ? "The relationship or activation policy changed. Reload the activation checks before continuing." : kind === "validation" ? "One or more activation gates are no longer satisfied. Review the current checks below." : kind === "forbidden" || kind === "unauthorized" ? "You are not permitted to record this activation decision." : "The activation command did not complete. The relationship remains unchanged.");
    } finally {
      setBusy(false);
    }
  }

  if (relationship.status === "ACTIVE") return <section className="vendor-activation-panel" aria-labelledby="vendor-activation-title"><span className="eyebrow">Activation complete</span><h3 id="vendor-activation-title">Vendor relationship active</h3><p>{relationship.service_name} may now receive certification requests. Vendor uploads still require separate bank review.</p></section>;
  return <section className="vendor-activation-panel" aria-labelledby="vendor-activation-title" aria-busy={state === "loading"}>
    <div className="vendor-activation-heading"><div><span className="eyebrow">Activation decision</span><h3 id="vendor-activation-title">Activate vendor relationship</h3></div>{eligibility && <span className={eligibility.eligible ? "vendor-activation-ready" : "vendor-activation-pending"}>{eligibility.eligible ? "Ready for authorization" : "Checks incomplete"}</span>}</div>
    {state === "loading" && <p>Checking the current policy, assessment, decisions, address outcome and blocking issues…</p>}
    {state === "unavailable" && <div role="status"><p>{error}</p><button type="button" className="secondary-button" onClick={() => { setState("loading"); setError(""); void loadVendorActivation(relationship.id).then((value) => { setEligibility(value); setState("ready"); }).catch(() => { setState("unavailable"); setError("Activation checks remain unavailable. The relationship has not changed."); }); }}>Reload activation checks</button></div>}
    {state === "ready" && eligibility && <>
      <p>Policy {eligibility.policy.policy_number}, version {eligibility.policy.version} applies from {formatDate(eligibility.policy.effective_from)}.</p>
      <ul className="vendor-activation-gates">{eligibility.gates.map((gate) => <li key={gate.code} data-satisfied={gate.satisfied}><span aria-hidden="true">{gate.satisfied ? "✓" : "–"}</span><div><strong>{gateLabel(gate.code)}</strong><p>{gate.explanation}</p></div></li>)}</ul>
      {eligibility.eligible && <div className="vendor-activation-action"><label htmlFor="vendor-activation-rationale">Activation rationale</label><textarea id="vendor-activation-rationale" rows={3} maxLength={2000} value={rationale} onChange={(event) => setRationale(event.target.value)} placeholder="Record why the current evidence and decisions support activation."/><small>{rationale.trim().length < 20 ? "Enter at least 20 characters for the activation record." : "This rationale will be stored with the activation receipt."}</small><button type="button" className="primary-button" disabled={busy || rationale.trim().length < 20} onClick={() => void activate()}>{busy ? "Activating…" : "Activate vendor relationship"}</button></div>}
      {!eligibility.eligible && <p className="inline-notice">Complete the first unsatisfied check above. Submission or upload alone cannot activate this relationship.</p>}
    </>}
    {error && state === "ready" && <p role="alert" className="inline-error">{error}</p>}
  </section>;
}

function gateLabel(code: string) {
  const labels: Record<string, string> = { RELATIONSHIP_STATE: "Relationship state", CURRENT_ASSESSMENT: "Current onboarding assessment", ASSESSMENT_CONCLUSION: "Assessment conclusion", REQUIRED_DECISIONS: "Required decisions", ADDRESS_OUTCOME: "Address verification", CONDITIONS: "Recorded conditions", BLOCKING_ISSUES: "Blocking issues", CONTRADICTIONS: "Evidence contradictions" };
  return labels[code] ?? code.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function formatDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "an unavailable date" : date.toLocaleString(); }
