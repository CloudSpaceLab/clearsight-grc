import { useEffect, useRef, useState } from "react";
import { activateVendorRelationship, loadVendorActivation } from "../vendorApi";
import type { VendorActivationResult, VendorRelationship } from "../vendorTypes";
import { apiErrorKind } from "../http";

type VendorActivationPanelProps = {
  relationship: VendorRelationship;
  onActivated: (relationship: VendorRelationship) => void;
  onRefreshed?: (relationship: VendorRelationship) => void;
};

export function VendorActivationPanel(props: VendorActivationPanelProps) {
  const { relationship } = props;
  // A new scope or material version retires every pending read and command.
  return <ActivationChecks key={JSON.stringify([relationship.tenant_id, relationship.legal_entity_id, relationship.id, relationship.version, relationship.status])} {...props}/>;
}

function ActivationChecks({ relationship, onActivated, onRefreshed }: VendorActivationPanelProps) {
  const [state, setState] = useState<"loading" | "ready" | "unavailable">(relationship.status === "ACTIVE" ? "ready" : "loading");
  const [eligibility, setEligibility] = useState<VendorActivationResult>();
  const [rationale, setRationale] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [reload, setReload] = useState(0);
  const generation = useRef(0);
  const commandPending = useRef(false);

  useEffect(() => {
    const current = ++generation.current;
    setError("");
    setEligibility(undefined);
    if (relationship.status === "ACTIVE") {
      setEligibility(undefined);
      setState("ready");
      return () => { generation.current++; };
    }
    setState("loading");
    void loadVendorActivation(relationship.id).then((value) => {
      if (current !== generation.current) return;
      if (!sameRelationship(value.relationship, relationship) || value.relationship.version < relationship.version) throw new Error("Activation relationship mismatch");
      setEligibility(value);
      setState("ready");
      if (value.relationship.status === "ACTIVE") onRefreshed?.(value.relationship);
    }).catch((caught) => {
      if (current !== generation.current) return;
      setEligibility(undefined);
      setState("unavailable");
      setError(apiErrorKind(caught) === "conflict" ? "No approved activation policy applies to this legal entity at the current time. Reload activation checks after the policy is reviewed." : "Activation checks could not be loaded. Reload the checks to confirm the relationship's current status.");
    });
    return () => { generation.current++; };
  // The keyed owner fixes relationship scope/version for this component lifetime.
  // Callback identity changes must not restart reads or retire commands.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [reload]);

  async function activate() {
    if (commandPending.current || state !== "ready" || !eligibility?.eligible || eligibility.relationship.status === "ACTIVE" || rationale.trim().length < 20) return;
    commandPending.current = true;
    const current = generation.current;
    setBusy(true);
    setError("");
    try {
      const result = await activateVendorRelationship(relationship.id, {
        expected_version: eligibility.relationship.version,
        intended_effective_at: new Date().toISOString(),
        rationale: rationale.trim(),
      });
      if (current !== generation.current) return;
      if (!sameRelationship(result.relationship, relationship) || result.relationship.status !== "ACTIVE") throw new Error("Activation could not be confirmed");
      setEligibility(result);
      onActivated(result.relationship);
    } catch (caught) {
      if (current !== generation.current) return;
      const kind = apiErrorKind(caught);
      setEligibility(undefined);
      setState("unavailable");
      setError(kind === "conflict" ? "The relationship or activation policy changed. Reload the activation checks before continuing." : kind === "validation" ? "One or more activation checks are no longer satisfied. Reload the activation checks before continuing." : kind === "forbidden" || kind === "unauthorized" ? "You are not permitted to record this activation decision. Reload the activation checks to confirm your current access." : "Activation could not be confirmed. Reload the activation checks to confirm the relationship's current status before trying again.");
    } finally {
      if (current === generation.current) {
        commandPending.current = false;
        setBusy(false);
      }
    }
  }

  if (relationship.status === "ACTIVE" || eligibility?.relationship.status === "ACTIVE") return <section className="vendor-activation-panel" aria-labelledby="vendor-activation-title"><span className="eyebrow">Activation complete</span><h3 id="vendor-activation-title">Vendor relationship active</h3><p>{relationship.service_name} may now receive certification requests. Vendor uploads still require separate bank review.</p></section>;
  return <section className="vendor-activation-panel" aria-labelledby="vendor-activation-title" aria-busy={state === "loading"}>
    <div className="vendor-activation-heading"><div><span className="eyebrow">Activation decision</span><h3 id="vendor-activation-title">Activate vendor relationship</h3></div>{eligibility && <span className={eligibility.eligible ? "vendor-activation-ready" : "vendor-activation-pending"}>{eligibility.eligible ? "Ready for authorization" : "Checks incomplete"}</span>}</div>
    {state === "loading" && <p>Checking the current policy, assessment, decisions, address outcome and blocking issues…</p>}
    {state === "unavailable" && <div role="status"><p>{error}</p><button type="button" className="secondary-button" onClick={() => { setEligibility(undefined); setState("loading"); setError(""); setReload((value) => value + 1); }}>Reload activation checks</button></div>}
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
  const labels: Record<string, string> = { RELATIONSHIP_STATE: "Relationship state", CURRENT_ASSESSMENT: "Current onboarding assessment", ASSESSMENT_CONCLUSION: "Assessment conclusion", REQUIRED_DECISIONS: "Required decisions", DECISION_AUTHORITY: "Decision authority", ADDRESS_OUTCOME: "Address verification", CONDITIONS: "Recorded conditions", BLOCKING_ISSUES: "Blocking issues", CONTRADICTIONS: "Evidence contradictions" };
  return labels[code] ?? code.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function formatDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "an unavailable date" : date.toLocaleString(); }

function sameRelationship(actual: VendorRelationship, expected: VendorRelationship) {
  return actual.id === expected.id && actual.tenant_id === expected.tenant_id && actual.legal_entity_id === expected.legal_entity_id;
}
