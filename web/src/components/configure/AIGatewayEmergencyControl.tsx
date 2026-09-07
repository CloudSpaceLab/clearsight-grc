import { useEffect, useState } from "react";
import { setGatewayEmergencyControl, type GatewayEmergencyControl, type GatewayEnvironment, type GatewayRuntimeStatus } from "../../aiGatewayTransportApi";
import "./AIGatewayEmergencyControl.css";

type EmergencyAction = "freeze" | "restore";

export function AIGatewayEmergencyControl({
  environment,
  control,
  runtimeStatus,
  canConfigure,
  loading,
  onChanged,
}: {
  environment: GatewayEnvironment;
  control: GatewayEmergencyControl;
  runtimeStatus: GatewayRuntimeStatus | null;
  canConfigure: boolean;
  loading: boolean;
  onChanged: () => Promise<void>;
}) {
  const [action, setAction] = useState<EmergencyAction | null>(null);
  const [reason, setReason] = useState("");
  const [confirmed, setConfirmed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  useEffect(() => {
    setAction(null);
    setReason("");
    setConfirmed(false);
    setMessage("");
  }, [environment, control.record_version, control.frozen]);

  const desired = control.frozen ? "Frozen" : "Enabled";
  const propagation = propagationState(control, runtimeStatus, loading);

  async function submit() {
    if (!action || busy || !confirmed || !reason.trim()) return;
    const nextFrozen = action === "freeze";
    setBusy(true);
    setMessage("");
    try {
      await setGatewayEmergencyControl({
        environment,
        frozen: nextFrozen,
        reason,
        expectedVersion: control.record_version,
      });
      setMessage(nextFrozen
        ? "Emergency freeze recorded. Gateway propagation is verified separately below."
        : "Outbound AI restoration recorded. Gateway propagation is verified separately below.");
      await onChanged();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Emergency gateway control could not be updated.");
    } finally {
      setBusy(false);
    }
  }

  return <section className="ai-gateway-emergency" aria-labelledby="ai-gateway-emergency-heading">
    <div className="ai-gateway-emergency__header">
      <div>
        <span className="eyebrow">Incident control</span>
        <h4 id="ai-gateway-emergency-heading">Emergency outbound AI</h4>
        <p>Stop new provider-bound requests without deleting provider or model configuration. This control is independent of routing revision lifecycle.</p>
      </div>
      <div className="ai-gateway-emergency__state" aria-label={`Desired outbound AI state: ${desired}`}>
        <span>Desired state</span>
        <strong>{desired}</strong>
        <small>Control v{control.record_version}</small>
      </div>
    </div>

    <div className="ai-gateway-emergency__propagation">
      <div><span>Gateway propagation</span><strong>{propagation.label}</strong></div>
      <p>{propagation.note}</p>
    </div>

    {control.reason && <p className="ai-gateway-emergency__last-change"><strong>Last reason:</strong> {control.reason}</p>}

    {canConfigure && !action && <div className="ai-gateway-emergency__actions">
      <button className={control.frozen ? "primary-button" : "secondary-button"} type="button" disabled={loading} onClick={() => setAction(control.frozen ? "restore" : "freeze")}>
        {control.frozen ? "Restore outbound AI" : "Emergency freeze"}
      </button>
    </div>}

    {canConfigure && action && <div className="ai-gateway-emergency__confirm" role="group" aria-label={action === "freeze" ? "Confirm emergency freeze" : "Confirm outbound AI restoration"}>
      <div>
        <strong>{action === "freeze" ? "Confirm emergency freeze" : "Confirm restoration"}</strong>
        <p>{action === "freeze"
          ? "After the gateway observes this control, new provider-bound requests fail closed. Provider configuration remains intact."
          : "Restore provider-bound requests only after the incident condition has been resolved and the active routing configuration is safe."}</p>
      </div>
      <label><span>Reason</span><textarea rows={3} maxLength={1000} value={reason} onChange={(event) => setReason(event.target.value)} placeholder={action === "freeze" ? "Describe the incident or containment reason" : "Describe why outbound AI is safe to restore"}/></label>
      <label className="ai-gateway-emergency__ack"><input type="checkbox" checked={confirmed} onChange={(event) => setConfirmed(event.target.checked)}/><span>{action === "freeze" ? "I understand this stops new provider-bound requests after propagation; already in-flight provider calls are not canceled." : "I confirm the incident condition has been addressed and outbound AI may resume."}</span></label>
      <div className="ai-gateway-emergency__actions">
        <button className="primary-button" type="button" disabled={busy || !confirmed || !reason.trim()} onClick={() => void submit()}>{busy ? "Applying…" : action === "freeze" ? "Freeze outbound AI" : "Restore outbound AI"}</button>
        <button className="text-button" type="button" disabled={busy} onClick={() => setAction(null)}>Cancel</button>
      </div>
    </div>}

    {!canConfigure && <p className="ai-gateway-emergency__readonly">Configuration write permission is required to change emergency outbound state.</p>}
    {message && <p className="ai-gateway-emergency__message" aria-live="polite">{message}</p>}
  </section>;
}

function propagationState(control: GatewayEmergencyControl, runtime: GatewayRuntimeStatus | null, loading: boolean) {
  if (loading) return { label: "Checking…", note: "Reading the gateway's independently reported emergency-control state." };
  if (!runtime?.configured) return { label: "Not connected", note: "The server-to-server gateway operations bridge is not configured, so propagation cannot be verified here." };
  if (!runtime.available) return { label: "Unavailable", note: "The operations bridge is configured, but the gateway's current emergency-control state could not be read." };
  if (!runtime.emergency_supported) return { label: "Unsupported", note: "This gateway process does not report emergency-control enforcement. Do not treat the desired state as applied." };
  if (runtime.emergency_revision !== control.record_version || runtime.outbound_frozen !== control.frozen) {
    return { label: "Propagating", note: `ClearSight expects control v${control.record_version}; the gateway reports v${runtime.emergency_revision}. Until these agree, applied state is not confirmed.` };
  }
  if (control.frozen) return { label: "Frozen · applied", note: "The gateway reports this exact freeze revision. New provider-bound requests are blocked; already in-flight provider calls are not canceled." };
  return { label: "Enabled · applied", note: "The gateway reports this exact control revision and provider-bound requests may proceed subject to normal governance and routing." };
}
