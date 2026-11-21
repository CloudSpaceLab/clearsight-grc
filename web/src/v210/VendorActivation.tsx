import { useEffect, useRef, useState, type FormEvent } from "react";
import { ApiError } from "../http";
import { activateVendorRelationship, loadVendorActivation } from "../vendorApi";
import type { VendorActivationResult, VendorRelationship } from "../vendorTypes";
import { checkVendorAuthority } from "./vendorAuthority";

export function VendorActivation({ relationshipId, relationshipScope, onActivated, onRefreshed }: { relationshipId: string; relationshipScope: Pick<VendorRelationship, "id" | "tenant_id" | "legal_entity_id" | "version"> & { status: string }; onActivated: (relationship: VendorRelationship) => void; onRefreshed?: (relationship: VendorRelationship) => void }) {
  const [result, setResult] = useState<VendorActivationResult>();
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [authorization, setAuthorization] = useState<"checking" | "allowed" | "unavailable">("checking");
  const [attempt, setAttempt] = useState(0); const [form, setForm] = useState(false);
  const [rationale, setRationale] = useState(""); const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  const [outcome, setOutcome] = useState<"unknown" | "rejected" | "activated">();
  const [refreshFailed, setRefreshFailed] = useState(false);
  const generation = useRef(0); const commandPending = useRef(false);
  const refreshed = useRef(onRefreshed); refreshed.current = onRefreshed;
  useEffect(() => { setRationale(""); }, [relationshipId, relationshipScope?.id, relationshipScope?.tenant_id, relationshipScope?.legal_entity_id, relationshipScope?.version, relationshipScope?.status]);
  useEffect(() => {
    generation.current++; commandPending.current = false;
    let active = true; setState("loading"); setResult(undefined); setAuthorization("checking");
    setForm(false); setBusy(false); setError(""); setOutcome(undefined); setRefreshFailed(false);
    if (!relationshipScope || relationshipScope.id !== relationshipId || !relationshipScope.tenant_id || !relationshipScope.legal_entity_id || relationshipScope.version <= 0) { setState("error"); return () => { active = false; generation.current++; }; }
    if (relationshipScope.status === "ACTIVE") return () => { active = false; generation.current++; };
    loadVendorActivation(relationshipId).then((value) => {
      if (!active) return;
      if (!value.relationship || value.relationship.id !== relationshipId || value.relationship.tenant_id !== relationshipScope.tenant_id || value.relationship.legal_entity_id !== relationshipScope.legal_entity_id || value.relationship.version < relationshipScope.version || (value.relationship.version === relationshipScope.version && value.relationship.status !== relationshipScope.status) || !value.policy || !Array.isArray(value.gates)) { setState("error"); return; }
      setResult(value); setState("ready");
      if (value.relationship.status === "ACTIVE") { try { refreshed.current?.(value.relationship); } catch { if (active) setRefreshFailed(true); } }
    }).catch(() => { if (active) setState("error"); });
    checkVendorAuthority("activate", relationshipId).then((value) => { if (active) setAuthorization(value ? "allowed" : "unavailable"); }).catch(() => { if (active) setAuthorization("unavailable"); });
    return () => { active = false; generation.current++; };
  }, [relationshipId, relationshipScope?.id, relationshipScope?.tenant_id, relationshipScope?.legal_entity_id, relationshipScope?.version, relationshipScope?.status, attempt]);
  const selectedActive = Boolean(relationshipScope && relationshipScope.id === relationshipId && relationshipScope.tenant_id && relationshipScope.legal_entity_id && relationshipScope.version > 0 && relationshipScope.status === "ACTIVE");
  const loadedActive = Boolean(!selectedActive && state === "ready" && result?.relationship.status === "ACTIVE" && !outcome);
  const ready = Boolean(result && !selectedActive && result.relationship.id === relationshipId && result.relationship.tenant_id === relationshipScope?.tenant_id && result.relationship.legal_entity_id === relationshipScope?.legal_entity_id && result.relationship.version >= relationshipScope?.version && result.relationship.status !== "ACTIVE" && result.eligible && result.gates.length > 0 && result.gates.every((gate) => gate.satisfied) && result.policy.status === "ACTIVE" && result.relationship.version > 0);
  function reload() { setForm(false); setError(""); setOutcome(undefined); setAttempt((value) => value + 1); }
  async function submit(event: FormEvent) {
    event.preventDefault(); if (!result || !ready || authorization !== "allowed" || commandPending.current || !rationale.trim()) return;
    commandPending.current = true; const commandGeneration = generation.current;
    setBusy(true); setError("");
    try {
      const next = await activateVendorRelationship(relationshipId, { expected_version: result.relationship.version, intended_effective_at: "0001-01-01T00:00:00Z", rationale: rationale.trim() });
      if (generation.current !== commandGeneration) return;
      if (next.relationship?.id !== relationshipId || next.relationship.tenant_id !== result.relationship.tenant_id || next.relationship.legal_entity_id !== result.relationship.legal_entity_id || next.relationship.status !== "ACTIVE" || next.relationship.version <= result.relationship.version || !next.receipt?.id || next.receipt.relationship_version !== next.relationship.version) { setOutcome("unknown"); setForm(false); return; }
      setResult(next); setOutcome("activated"); setForm(false);
      try { onActivated(next.relationship); }
      catch { if (generation.current === commandGeneration) setRefreshFailed(true); }
    } catch (failure) {
      if (generation.current !== commandGeneration) return;
      setForm(false);
      if (failure instanceof ApiError && failure.status >= 400 && failure.status < 500) { setError(failure.message); setOutcome("rejected"); }
      else setOutcome("unknown");
    } finally { if (generation.current === commandGeneration) { commandPending.current = false; setBusy(false); } }
  }
  return <section className="v-panel v-vendor-activation"><h2>Activation</h2>
    {(selectedActive || loadedActive) && <p>Relationship active.</p>}
    {loadedActive && refreshFailed && <p role="alert"><span className="v-status v-status--high" style={{ display: "inline-block" }}>Relationship details could not be refreshed. Reload the relationship.</span></p>}
    {!selectedActive && state === "loading" && <p role="status">Loading activation requirements…</p>}
    {!selectedActive && state === "error" && <><p role="alert">Activation requirements unavailable.</p><button onClick={reload}>Reload requirements</button></>}
    {!selectedActive && !loadedActive && state === "ready" && result && <>
      {(!outcome || outcome === "activated") && <><p>Policy {result.policy.policy_number} · Version {result.policy.version} · Effective from <time dateTime={result.policy.effective_from}>{formatPolicyDate(result.policy.effective_from)}</time></p>
      {result.gates.length === 0 ? <p>Activation requirements unavailable.</p> : <ul>{result.gates.map((gate) => <li key={gate.code}><strong>{gateLabel(gate.code)}</strong> · {gate.satisfied ? "Met" : "Not met"} · <span>{gate.explanation}</span></li>)}</ul>}</>}
      {outcome === "activated" && <p role="status">Relationship activated.</p>}
      {outcome === "activated" && refreshFailed && <p role="alert"><span className="v-status v-status--high" style={{ display: "inline-block" }}>Relationship details could not be refreshed. Reload the relationship.</span></p>}
      {outcome === "unknown" && <p role="alert">Activation status could not be confirmed. Reload the requirements before trying again.</p>}
      {outcome === "rejected" && <p role="alert">{error}</p>}
      {(outcome === "unknown" || outcome === "rejected") && <button onClick={reload}>Reload requirements</button>}
      {!outcome && ready && authorization === "checking" && <p role="status">Checking activation access…</p>}
      {!outcome && ready && authorization === "unavailable" && <p>Activation unavailable. Check your role or reload the requirements.</p>}
      {!outcome && ready && authorization === "unavailable" && <button onClick={reload}>Reload requirements</button>}
      {!outcome && ready && authorization === "allowed" && !form && <button className="v-primary" onClick={() => setForm(true)}>Activate relationship</button>}
      {!outcome && ready && authorization === "allowed" && form && <form className="v-intake" onSubmit={(event) => void submit(event)}><p>Requirements are checked again before activation.</p><label>Activation rationale<textarea required maxLength={2000} value={rationale} onChange={(event) => setRationale(event.target.value)}/></label><div className="v-intake-actions"><button type="button" disabled={busy} onClick={() => setForm(false)}>Cancel</button><button className="v-primary" disabled={busy || !rationale.trim()}>{busy ? "Activating…" : "Confirm activation"}</button></div></form>}
    </>}
  </section>;
}

function gateLabel(code: string) {
  const labels: Record<string, string> = { RELATIONSHIP_STATE: "Relationship state", CURRENT_ASSESSMENT: "Current onboarding assessment", ASSESSMENT_CONCLUSION: "Assessment conclusion", REQUIRED_DECISIONS: "Required decisions", DECISION_AUTHORITY: "Decision authority", ADDRESS_OUTCOME: "Address verification", CONDITIONS: "Recorded conditions", BLOCKING_ISSUES: "Blocking issues", CONTRADICTIONS: "Evidence contradictions" };
  return labels[code] ?? code.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
function formatPolicyDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "Date unavailable" : date.toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric", timeZone: "UTC" }); }
