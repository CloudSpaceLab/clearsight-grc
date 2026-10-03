import { useCallback, useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { loadContext } from "../../api";
import type { AIGovernancePolicy, AIGovernanceWorkload } from "../../types";
import { loadGatewayBaselines, type GatewayBaselinePolicy } from "../../aiGovernanceControlApi";
import {
  createGatewayExceptionEnforcementRevision,
  createGatewayExceptionShadowDraft,
  gatewayExceptionScope,
  isGatewayExceptionPolicy,
  transitionGatewayException,
  type GatewayExceptionTransition,
} from "../../aiGatewayExceptionApi";
import "./AIGatewayExceptionControl.css";

type LoadState = "loading" | "live" | "unavailable";
type WaivableRule = { id: string; label: string };

export function AIGatewayExceptionControl({
  policies,
  policyState,
  workloads,
  workloadState,
  onChanged,
}: {
  policies: AIGovernancePolicy[];
  policyState: LoadState;
  workloads: AIGovernanceWorkload[];
  workloadState: LoadState;
  onChanged?: () => void;
}) {
  const [canConfigure, setCanConfigure] = useState(false);
  const [actorId, setActorId] = useState("");
  const [baselines, setBaselines] = useState<GatewayBaselinePolicy[]>([]);
  const [loading, setLoading] = useState(true);
  const [name, setName] = useState("Temporary AI baseline exception");
  const [environment, setEnvironment] = useState("PRODUCTION");
  const [workloadIds, setWorkloadIds] = useState<string[]>([]);
  const [ruleIds, setRuleIds] = useState<string[]>([]);
  const [justification, setJustification] = useState("");
  const [effectiveUntil, setEffectiveUntil] = useState(defaultExpiry());
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [context, baselineItems] = await Promise.all([loadContext(), loadGatewayBaselines()]);
      setCanConfigure(Boolean(context.capabilities?.config_write));
      setActorId(context.actor.id);
      setBaselines(baselineItems);
    } catch {
      setBaselines([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  const activeBaseline = baselines.find((policy) => policy.status === "ACTIVE" && policy.rollout_mode === "ENFORCE");
  const waivableRules = useMemo(() => restrictiveRules(activeBaseline), [activeBaseline]);
  const workloadCandidates = useMemo(
    () => workloads.filter((workload) => workload.state === "ACTIVE" && workload.environment.toUpperCase() === environment),
    [environment, workloads],
  );
  const exceptions = useMemo(
    () => policies.filter(isGatewayExceptionPolicy).sort((left, right) => String(right.effective_until ?? "").localeCompare(String(left.effective_until ?? ""))),
    [policies],
  );
  const expiry = new Date(effectiveUntil);
  const expiryValid = Number.isFinite(expiry.getTime()) && expiry.getTime() > Date.now() && expiry.getTime() <= Date.now() + 30 * 24 * 60 * 60 * 1000;
  const canCreate = canConfigure && !busy && Boolean(activeBaseline) && workloadIds.length > 0 && workloadIds.length <= 16 && ruleIds.length > 0 && ruleIds.length <= 16 && justification.trim().length > 0 && expiryValid;

  useEffect(() => {
    setWorkloadIds((current) => current.filter((id) => workloadCandidates.some((workload) => workload.id === id)));
  }, [workloadCandidates]);

  useEffect(() => {
    setRuleIds((current) => current.filter((id) => waivableRules.some((rule) => rule.id === id)));
  }, [waivableRules]);

  async function createDraft(event: FormEvent) {
    event.preventDefault();
    if (!canCreate || !activeBaseline) return;
    await runCommand(async () => {
      await createGatewayExceptionShadowDraft({
        name,
        baseline: activeBaseline,
        workloadRecordIds: workloadIds,
        environments: [environment],
        waivedRuleIds: ruleIds,
        justification,
        effectiveUntil: expiry.toISOString(),
      });
      setJustification("");
      setWorkloadIds([]);
      setRuleIds([]);
      return "Shadow exception draft created. It does not weaken live traffic until an independently approved enforcement revision is activated.";
    });
  }

  async function transition(policy: AIGovernancePolicy, action: GatewayExceptionTransition) {
    await runCommand(async () => {
      await transitionGatewayException(policy.id, action, policy.record_version);
      switch (action) {
        case "submit": return "Exception submitted for independent approval.";
        case "approve": return "Exception revision approved. A checker can now activate it.";
        case "activate": return policy.rollout_mode === "ENFORCE" ? "Bounded exception enforcement activated." : "Shadow exception activated for observation only.";
        case "suspend": return "Exception suspended. The full organization baseline applies again.";
        default: return "Exception lifecycle updated.";
      }
    });
  }

  async function createEnforcement(source: AIGovernancePolicy) {
    await runCommand(async () => {
      await createGatewayExceptionEnforcementRevision(source);
      return "Enforcement revision created from the exact Shadow exception. Submit it for independent approval.";
    });
  }

  async function runCommand(command: () => Promise<string>) {
    if (busy) return;
    setBusy(true);
    setMessage("");
    try {
      setMessage(await command());
      await load();
      onChanged?.();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "The gateway exception could not be updated.");
    } finally {
      setBusy(false);
    }
  }

  return <article className="configure-context-panel ai-gateway-exception" aria-labelledby="gateway-exception-heading">
    <div className="configure-subheader ai-gateway-exception__header">
      <div>
        <span className="eyebrow">Gateway · bounded exceptions</span>
        <h3 id="gateway-exception-heading">Temporary baseline exceptions</h3>
        <p>Waive only named rules on one exact baseline revision for explicitly selected workloads. Exceptions require maker/checker approval, expire automatically, and never bypass source-fact failures or the emergency outbound freeze.</p>
      </div>
      <span className="ai-gateway-exception__state">{loading ? "Checking…" : exceptions.some((policy) => policy.status === "ACTIVE" && policy.rollout_mode === "ENFORCE") ? "Exception active" : "Full baseline"}</span>
    </div>

    {exceptions.length > 0 && <section className="ai-gateway-exception__list" aria-label="Recent gateway baseline exceptions">
      {exceptions.slice(0, 6).map((policy) => <ExceptionRow
        key={policy.id}
        policy={policy}
        actorId={actorId}
        busy={busy}
        siblings={exceptions.filter((candidate) => candidate.code === policy.code)}
        onTransition={transition}
        onCreateEnforcement={createEnforcement}
      />)}
    </section>}

    {!canConfigure ? <div className="calm-empty"><span>↗</span><div><strong>Read-only access</strong><p>Configuration permission is required to create or change baseline exceptions.</p></div></div>
      : !activeBaseline ? <div className="calm-empty"><span>◎</span><div><strong>No enforcing organization baseline</strong><p>Activate an enforcing organization baseline before creating an exception.</p></div></div>
      : <form className="ai-gateway-exception__form" onSubmit={createDraft}>
        <div className="ai-gateway-exception__target">
          <span>Exact baseline target</span>
          <strong>{activeBaseline.name} · v{activeBaseline.version}</strong>
          <small>{activeBaseline.code} · enforcement</small>
        </div>

        <div className="ai-gateway-exception__grid">
          <label><span>Exception name</span><input value={name} onChange={(event) => setName(event.target.value)} maxLength={160} required/></label>
          <label><span>Environment</span><select value={environment} onChange={(event) => setEnvironment(event.target.value)} aria-label="Exception environment"><option value="PRODUCTION">Production</option><option value="TEST">Test</option><option value="DEVELOPMENT">Development</option></select></label>
          <label><span>Expires at</span><input type="datetime-local" value={effectiveUntil} onChange={(event) => setEffectiveUntil(event.target.value)} required/><small>Maximum lifetime · 30 days.</small></label>
        </div>

        <fieldset className="ai-gateway-exception__choices">
          <legend>Workloads</legend>
          {workloadState === "unavailable" ? <p>Workload inventory is unavailable.</p> : workloadCandidates.length === 0 ? <p>No active workloads are registered in this environment.</p> : workloadCandidates.map((workload) => <label key={workload.id}><input type="checkbox" checked={workloadIds.includes(workload.id)} onChange={() => setWorkloadIds(toggle(workloadIds, workload.id, 16))}/><span><strong>{workload.name}</strong><small>{workload.code} · policy v{workload.policy_version}</small></span></label>)}
        </fieldset>

        <fieldset className="ai-gateway-exception__choices">
          <legend>Baseline rules to waive</legend>
          {waivableRules.length === 0 ? <p>No restrictive baseline rules are eligible for exception.</p> : waivableRules.map((rule) => <label key={rule.id}><input type="checkbox" checked={ruleIds.includes(rule.id)} onChange={() => setRuleIds(toggle(ruleIds, rule.id, 16))}/><span><strong>{rule.label}</strong><small>{rule.id}</small></span></label>)}
        </fieldset>

        <label className="ai-gateway-exception__justification"><span>Justification</span><textarea value={justification} onChange={(event) => setJustification(event.target.value)} rows={3} maxLength={1000} required/><small>Stored as governed scope metadata for reviewer and audit reconstruction.</small></label>

        <div className="ai-gateway-exception__actions">
          <button className="primary-button" type="submit" disabled={!canCreate}>{busy ? "Creating…" : "Create Shadow exception"}</button>
          <span>{expiryValid ? "Independent approval required before enforcement." : "Choose a future expiry within 30 days."}</span>
        </div>
      </form>}

    {policyState === "unavailable" && <p className="ai-gateway-exception__message" aria-live="polite">Policy inventory is unavailable; existing exception lifecycle cannot be shown.</p>}
    {message && <p className="ai-gateway-exception__message" aria-live="polite">{message}</p>}
  </article>;
}

function ExceptionRow({
  policy,
  actorId,
  busy,
  siblings,
  onTransition,
  onCreateEnforcement,
}: {
  policy: AIGovernancePolicy;
  actorId: string;
  busy: boolean;
  siblings: AIGovernancePolicy[];
  onTransition: (policy: AIGovernancePolicy, action: GatewayExceptionTransition) => Promise<void>;
  onCreateEnforcement: (policy: AIGovernancePolicy) => Promise<void>;
}) {
  const scope = gatewayExceptionScope(policy);
  const independentChecker = Boolean(actorId && policy.maker_id !== actorId);
  const hasOpenEnforcement = siblings.some((candidate) => candidate.rollout_mode === "ENFORCE" && ["DRAFT", "PENDING_APPROVAL", "APPROVED"].includes(candidate.status));
  return <div className="ai-gateway-exception__row">
    <div><strong>{policy.name}</strong><span>{humanize(policy.status)} · {policy.rollout_mode.toLowerCase()} · v{policy.version}</span><small>{scope ? `${scope.workload_record_ids.length} workload${scope.workload_record_ids.length === 1 ? "" : "s"} · ${scope.waived_rule_ids.length} rule${scope.waived_rule_ids.length === 1 ? "" : "s"} · expires ${formatTime(policy.effective_until)}` : "Invalid scope metadata"}</small></div>
    <div className="ai-gateway-exception__row-actions">
      {policy.status === "DRAFT" && <button className="primary-button" type="button" disabled={busy} onClick={() => void onTransition(policy, "submit")}>Submit</button>}
      {policy.status === "PENDING_APPROVAL" && independentChecker && <button className="primary-button" type="button" disabled={busy} onClick={() => void onTransition(policy, "approve")}>Approve</button>}
      {policy.status === "PENDING_APPROVAL" && !independentChecker && <span>Awaiting independent checker</span>}
      {(policy.status === "APPROVED" || policy.status === "SUSPENDED") && independentChecker && <button className="primary-button" type="button" disabled={busy} onClick={() => void onTransition(policy, "activate")}>Activate</button>}
      {policy.status === "ACTIVE" && policy.rollout_mode === "SHADOW" && !hasOpenEnforcement && <button className="primary-button" type="button" disabled={busy} onClick={() => void onCreateEnforcement(policy)}>Create enforcement revision</button>}
      {policy.status === "ACTIVE" && policy.rollout_mode === "ENFORCE" && <button className="secondary-button" type="button" disabled={busy} onClick={() => void onTransition(policy, "suspend")}>Suspend</button>}
    </div>
  </div>;
}

function restrictiveRules(baseline?: GatewayBaselinePolicy): WaivableRule[] {
  if (!baseline) return [];
  const rules = baseline.definition.rules ?? [];
  return rules.flatMap((rule) => {
    const id = typeof rule.id === "string" ? rule.id : "";
    const action = typeof rule.action === "string" ? rule.action : "";
    const reason = typeof rule.reason_code === "string" ? rule.reason_code : id;
    const obligations = Array.isArray(rule.obligations) ? rule.obligations : [];
    const instructionRule = obligations.some((value) => typeof value === "object" && value !== null && "code" in value && (value as { code?: unknown }).code === "ORG_INSTRUCTION");
    if (!id || instructionRule || !["DENY", "REQUIRE_APPROVAL", "MODIFY", "ROUTE"].includes(action)) return [];
    return [{ id, label: `${humanize(reason)} · ${humanize(action)}` }];
  });
}

function toggle(values: string[], value: string, max: number) {
  if (values.includes(value)) return values.filter((candidate) => candidate !== value);
  if (values.length >= max) return values;
  return [...values, value];
}

function defaultExpiry() {
  const date = new Date(Date.now() + 24 * 60 * 60 * 1000);
  return new Date(date.getTime() - date.getTimezoneOffset() * 60_000).toISOString().slice(0, 16);
}

function formatTime(value?: string) {
  if (!value) return "unknown";
  const date = new Date(value);
  return Number.isFinite(date.getTime()) ? date.toLocaleString() : "unknown";
}

function humanize(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replaceAll("-", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
