import { useCallback, useEffect, useState } from "react";
import { loadContext } from "../../api";
import {
  createGatewayTransportRevision,
  loadGatewayTransportState,
  transitionGatewayTransport,
  type GatewayEmergencyControl,
  type GatewayEnvironment,
  type GatewayProxyInfo,
  type GatewayRuntimeStatus,
  type GatewayTransportDefinition,
  type GatewayTransportRevision,
  type GatewayTransportTransition,
} from "../../aiGatewayTransportApi";
import { AIGatewayEmergencyControl } from "./AIGatewayEmergencyControl";
import { AIGatewayProxyCard } from "./AIGatewayProxyCard";
import { AIGatewayTransportDraftForm } from "./AIGatewayTransportDraftForm";
import "./AIGatewayTransportControl.css";

const environments: GatewayEnvironment[] = ["PRODUCTION", "TEST", "DEVELOPMENT"];
const emptyProxy: GatewayProxyInfo = { configured: false, ingress: [] };
const emptyEmergency = (environment: GatewayEnvironment): GatewayEmergencyControl => ({ tenant_id: "", environment, frozen: false, record_version: 0 });

export function AIGatewayTransportControl() {
  const [environment, setEnvironment] = useState<GatewayEnvironment>("PRODUCTION");
  const [actorId, setActorId] = useState("");
  const [canConfigure, setCanConfigure] = useState(false);
  const [revisions, setRevisions] = useState<GatewayTransportRevision[]>([]);
  const [runtimeStatus, setRuntimeStatus] = useState<GatewayRuntimeStatus | null>(null);
  const [proxy, setProxy] = useState<GatewayProxyInfo>(emptyProxy);
  const [emergencyControl, setEmergencyControl] = useState<GatewayEmergencyControl>(() => emptyEmergency("PRODUCTION"));
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [context, state] = await Promise.all([loadContext(), loadGatewayTransportState(environment)]);
      setActorId(context.actor.id);
      setCanConfigure(Boolean(context.capabilities?.config_write));
      setRevisions(state.revisions);
      setRuntimeStatus(state.runtimeStatus);
      setProxy(state.proxy);
      setEmergencyControl(state.emergencyControl);
    } catch (error) {
      setRevisions([]);
      setRuntimeStatus(null);
      setProxy(emptyProxy);
      setEmergencyControl(emptyEmergency(environment));
      setMessage(error instanceof Error ? error.message : "Gateway routing configuration could not be loaded.");
    } finally {
      setLoading(false);
    }
  }, [environment]);

  useEffect(() => { void load(); }, [load]);

  const latest = revisions[0];
  const active = revisions.find((revision) => revision.status === "ACTIVE");
  const openRevision = revisions.find((revision) => ["DRAFT", "PENDING_APPROVAL", "APPROVED"].includes(revision.status));

  async function create(definition: GatewayTransportDefinition, changeReason: string) {
    await run(async () => {
      await createGatewayTransportRevision({ environment, definition, changeReason });
      return "Routing draft created. Submit for independent approval.";
    });
  }

  async function transition(revision: GatewayTransportRevision, action: GatewayTransportTransition) {
    await run(async () => {
      await transitionGatewayTransport(revision.id, action, revision.record_version);
      switch (action) {
        case "submit": return "Routing revision submitted for independent approval.";
        case "approve": return "Routing approved. Activate to replace the current route.";
        case "activate": return "Routing activated. Gateways may still be applying the change; check their status before relying on the new route.";
        case "suspend": return "Routing suspended. Check gateway status to confirm the change.";
        default: return "Routing revision retired and retained for reconstruction.";
      }
    });
  }

  async function run(command: () => Promise<string>) {
    if (busy) return;
    setBusy(true);
    setMessage("");
    try {
      setMessage(await command());
      await load();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Gateway routing configuration could not be updated.");
    } finally {
      setBusy(false);
    }
  }

  const latestIsIndependent = Boolean(latest && actorId && latest.maker_id !== actorId);

  return <article className="configure-context-panel ai-gateway-transport" aria-labelledby="gateway-routing-heading">
    <div className="configure-subheader ai-gateway-transport__header">
      <div>
        <span className="eyebrow">Gateway · provider routing</span>
        <h3 id="gateway-routing-heading">Organization AI proxy</h3>
        <p>Manage provider routes and model aliases. Credentials stay outside ClearSight.</p>
      </div>
      <label className="ai-gateway-transport__environment"><span>Environment</span><select value={environment} onChange={(event) => setEnvironment(event.target.value as GatewayEnvironment)}>{environments.map((value) => <option value={value} key={value}>{title(value)}</option>)}</select></label>
    </div>

    <div className="ai-gateway-transport__status-grid" aria-label="Gateway configuration status">
      <Status label="Desired authority" value={active ? `v${active.version} · active` : loading ? "Checking…" : "Not configured"} note={active ? shortChecksum(active.checksum) : "No database transport revision is active."}/>
      <Status label="Providers" value={active ? String(active.definition.providers.filter((provider) => provider.state === "ENABLED").length) : "—"} note={active ? `${active.definition.providers.length} configured in the active revision` : "No active revision"}/>
      <Status label="Logical models" value={active ? String(active.definition.models.length) : "—"} note="Applications select model aliases."/>
      <Status label="Runtime apply" value={runtimeValue(runtimeStatus, active, loading)} note={runtimeNote(runtimeStatus, active)}/>
    </div>

    <AIGatewayProxyCard proxy={proxy} runtimeStatus={runtimeStatus} loading={loading}/>
    <AIGatewayEmergencyControl environment={environment} control={emergencyControl} runtimeStatus={runtimeStatus} canConfigure={canConfigure} loading={loading} onChanged={load}/>

    {latest && <section className="ai-gateway-transport__revision" aria-label="Latest gateway routing revision">
      <div><span className="eyebrow">Latest revision</span><h4>v{latest.version} · {title(latest.status)}</h4><p>{latest.change_reason}</p></div>
      <div className="ai-gateway-transport__revision-meta"><span>{latest.definition.providers.length} provider{latest.definition.providers.length === 1 ? "" : "s"}</span><span>{latest.definition.models.length} alias{latest.definition.models.length === 1 ? "" : "es"}</span><span>{shortChecksum(latest.checksum)}</span></div>
      {canConfigure && <div className="ai-gateway-transport__actions">
        {latest.status === "DRAFT" && <button className="primary-button" type="button" disabled={busy} onClick={() => void transition(latest, "submit")}>Submit for approval</button>}
        {latest.status === "PENDING_APPROVAL" && latestIsIndependent && <button className="primary-button" type="button" disabled={busy} onClick={() => void transition(latest, "approve")}>Approve routing</button>}
        {latest.status === "PENDING_APPROVAL" && !latestIsIndependent && <span>Awaiting an independent checker</span>}
        {(latest.status === "APPROVED" || latest.status === "SUSPENDED") && latestIsIndependent && <button className="primary-button" type="button" disabled={busy} onClick={() => void transition(latest, "activate")}>Activate revision</button>}
        {latest.status === "ACTIVE" && <button className="secondary-button" type="button" disabled={busy} onClick={() => void transition(latest, "suspend")}>Suspend routing</button>}
      </div>}
    </section>}

    {!canConfigure ? <div className="calm-empty"><span>↗</span><div><strong>Read-only gateway access</strong><p>View-only access. Ask an administrator to change routing.</p></div></div>
      : !openRevision ? <AIGatewayTransportDraftForm environment={environment} busy={busy} disabled={loading} onSubmit={create}/>
      : <div className="calm-empty"><span>◌</span><div><strong>Finish the open revision first</strong><p>{title(openRevision.status)} v{openRevision.version} is already in the governed lifecycle for {title(environment)}. Complete or retire it before starting another routing change.</p></div></div>}

    {message && <p className="ai-gateway-transport__message" aria-live="polite">{message}</p>}
  </article>;
}

function Status({ label, value, note }: { label: string; value: string; note: string }) {
  return <div><span>{label}</span><strong>{value}</strong><small>{note}</small></div>;
}

function runtimeValue(status: GatewayRuntimeStatus | null, active: GatewayTransportRevision | undefined, loading: boolean) {
  if (loading) return "Checking…";
  if (!status) return "Unavailable";
  if (!status.configured) return "Not connected";
  if (!status.available) return "Unavailable";
  if (!status.applied_revision) return "Not applied";
  if (active && status.applied_revision !== active.version) return `Pending · v${status.applied_revision} → v${active.version}`;
  if (status.degraded || status.desired_revision !== status.applied_revision) return `Degraded · v${status.applied_revision} → v${status.desired_revision || "?"}`;
  return `Applied · v${status.applied_revision}`;
}

function runtimeNote(status: GatewayRuntimeStatus | null, active: GatewayTransportRevision | undefined) {
  if (!status) return "Runtime apply status could not be loaded.";
  if (!status.configured) return "The API-to-gateway operations bridge is not configured for this deployment.";
  if (!status.available) return "The operations bridge is configured but the gateway status endpoint is unavailable.";
  if (status.error_code) return `Gateway reports ${title(status.error_code)}.`;
  if (active && status.applied_revision !== active.version) return "Gateway has not applied the latest routing revision. Previous routing remains active.";
  if (!status.applied_revision) return "No routing configuration applied to this gateway.";
  return status.applied_checksum ? shortChecksum(status.applied_checksum) : "The gateway reports the active revision without a checksum.";
}

function title(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function shortChecksum(value: string) {
  return value ? `sha256 · ${value.slice(0, 10)}…` : "Checksum unavailable";
}
