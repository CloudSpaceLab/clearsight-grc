import { useEffect, useState } from "react";
import { apiErrorKind } from "../../http";
import {
  activateFormResponsePolicy, approveFormResponsePolicy, createFormResponsePolicy, listFormResponsePolicies,
  rollbackFormResponsePolicy, simulateFormResponsePolicy, submitFormResponsePolicy, suspendFormResponsePolicy,
  type CreateFormResponsePolicyInput, type FormPolicySimulation, type FormResponsePolicy,
} from "../../formPoliciesApi";
import { Button, EmptyState, FocusedDialog, Notice, SelectableRecord, StatusBadge, Surface } from "../ui";
import { FormPolicyEditor } from "./FormPolicyEditor";
import "./form-policies.css";

type LoadState = "loading" | "live" | "sign-in-required" | "error";

export function FormPoliciesView() {
  const [items, setItems] = useState<FormResponsePolicy[]>([]);
  const [selectedID, setSelectedID] = useState<string>();
  const [state, setState] = useState<LoadState>("loading");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [creating, setCreating] = useState(false);
  const [busy, setBusy] = useState("");
  const [simulations, setSimulations] = useState<Record<string, FormPolicySimulation>>({});
  const selected = items.find((item) => item.id === selectedID) ?? items[0];
  const rollbackTarget = selected ? items.filter((item) => item.code === selected.code && item.version < selected.version).sort((left, right) => right.version - left.version)[0] : undefined;

  async function refresh() {
    setState("loading"); setError("");
    try {
      const values = await listFormResponsePolicies();
      setItems(values); setSelectedID((current) => values.some((item) => item.id === current) ? current : values[0]?.id); setState("live");
    } catch (cause) {
      setError(message(cause, "Response policies cannot be checked right now."));
      setState(apiErrorKind(cause) === "unauthorized" ? "sign-in-required" : items.length ? "live" : "error");
    }
  }
  useEffect(() => { void refresh(); }, []);

  async function create(input: CreateFormResponsePolicyInput) {
    setBusy("create"); setError("");
    try { const value = await createFormResponsePolicy(input); setCreating(false); setNotice("Policy draft created. Simulate its stored response population before requesting approval."); await refresh(); setSelectedID(value.id); }
    catch (cause) { setError(message(cause, "The policy draft could not be created.")); }
    finally { setBusy(""); }
  }
  async function act(policy: FormResponsePolicy, rollbackTargetID?: string) {
    if (busy) return;
    const simulation = simulations[policy.id];
    setBusy(policy.id); setError(""); setNotice("");
    try {
      if (policy.status === "DRAFT" && !simulation) {
        const receipt = await simulateFormResponsePolicy(policy.id, policy.record_version); setSimulations((current) => ({ ...current, [policy.id]: receipt })); setNotice("Simulation completed against the current stored response population."); return;
      }
      let next: FormResponsePolicy;
      if (policy.status === "DRAFT") next = await submitFormResponsePolicy(policy.id, policy.record_version, simulation!.id);
      else if (policy.status === "PENDING_APPROVAL") next = await approveFormResponsePolicy(policy.id, policy.record_version, policy.approved_simulation_id ?? simulation?.id ?? "");
      else if (policy.status === "APPROVED") next = await activateFormResponsePolicy(policy.id, policy.record_version);
      else if (policy.status === "ACTIVE") next = await suspendFormResponsePolicy(policy.id, policy.record_version);
      else if (policy.status === "SUSPENDED" && rollbackTargetID) next = await rollbackFormResponsePolicy(policy.id, policy.record_version, rollbackTargetID);
      else return;
      setItems((current) => current.map((item) => item.id === next.id ? next : item));
      setNotice(actionNotice(next.status));
    } catch (cause) { setError(message(cause, "The policy command could not be completed. No policy state was changed.")); }
    finally { setBusy(""); }
  }

  return <section className="forms-policies" aria-labelledby="form-policies-title">
    <header className="forms-policies__heading"><div><p>Governed automation</p><h2 id="form-policies-title">Response policies</h2><span>Create an issue from a completed response that meets an approved concern threshold. Simulation and independent approval are required before activation.</span></div>{state === "live" && <Button variant="primary" onPress={() => setCreating(true)}>Create policy</Button>}</header>
    {error && state === "live" && <Notice tone="error">{error} Policies already shown remain available.</Notice>}
    {notice && <Notice>{notice}</Notice>}
    {state === "loading" && <Surface><p role="status">Loading response policies for this legal entity…</p></Surface>}
    {state === "sign-in-required" && (
      <EmptyState population="Response policies in this legal entity" title="Sign in to review response policies" description="Your session ended before the policy population could be loaded." action={<Button onPress={() => void refresh()}>Retry loading policies</Button>}/>
    )}
    {state === "error" && <div role="alert"><EmptyState population="Response policies in this legal entity" title="Response policies could not be loaded" description={error} action={<Button onPress={() => void refresh()}>Retry loading policies</Button>}/></div>}
    {state === "live" && items.length === 0 && (
      <EmptyState population="Response policies in this legal entity" title="No response policies have been created" description="Create a draft to select the approved form, eligible subjects, issue handling and outcome check." action={<Button variant="primary" onPress={() => setCreating(true)}>Create policy</Button>}/>
    )}
    {state === "live" && items.length > 0 && <div className="forms-policies__layout">
      <nav className="forms-policies__list" aria-label="Response policies">{items.map((policy) => <SelectableRecord key={policy.id} title={policy.name} metadata={`${statusLabel(policy.status)} · ${policy.code} · form revision ${policy.eligibility.form_template_version}`} description={policy.rollout === "SHADOW" ? "Shadow impact only" : "Creates governed issues"} isSelected={policy.id === selected?.id} onPress={() => setSelectedID(policy.id)}/>)}</nav>
      {selected && (
        <PolicyDetail policy={selected} simulation={simulations[selected.id]} rollbackTarget={rollbackTarget} busy={busy === selected.id} onAction={() => void act(selected, rollbackTarget?.id)}/>
      )}
    </div>}
    {creating && <FocusedDialog label="Create response policy" size="wide" onClose={() => setCreating(false)}><FormPolicyEditor onCancel={() => setCreating(false)} onCreate={create} busy={busy === "create"}/></FocusedDialog>}
  </section>;
}

function PolicyDetail({ policy, simulation, rollbackTarget, busy, onAction }: { policy: FormResponsePolicy; simulation?: FormPolicySimulation; rollbackTarget?: FormResponsePolicy; busy: boolean; onAction: () => void }) {
  const action = dominantAction(policy, simulation, rollbackTarget);
  return <article className="forms-policy-detail" aria-labelledby={`policy-${policy.id}`}>
    <header><div><StatusBadge tone={statusTone(policy.status)}>{statusLabel(policy.status)}</StatusBadge><h3 id={`policy-${policy.id}`}>{policy.name}</h3><p>{policy.purpose}</p></div>{action && <Button variant="primary" isLoading={busy} onPress={onAction}>{action}</Button>}</header>
    <dl className="forms-policy-facts"><div><dt>Form scope</dt><dd>{policy.eligibility.form_template_id} · revision {policy.eligibility.form_template_version}</dd></div><div><dt>Concern threshold</dt><dd>{policy.eligibility.bands?.map(statusLabel).join(", ") || "Bounded score threshold"}</dd></div><div><dt>Rollout</dt><dd>{policy.rollout === "SHADOW" ? "Shadow impact only" : "Create governed issues"}</dd></div><div><dt>Blast radius</dt><dd>{policy.blast_radius.per_run} per run · {policy.blast_radius.per_day} per day</dd></div><div><dt>Outcome check</dt><dd>After {policy.outcome_contract.check_after_minutes} minutes</dd></div><div><dt>Policy revision</dt><dd>{policy.version}</dd></div></dl>
    <section><h4>Issue handling</h4><strong>{policy.action.title_template}</strong><p>{policy.action.requested_handling}</p></section>
    {simulation ? <section className="forms-policy-impact" aria-label="Latest simulation impact"><div><strong>{simulation.would_create_count} new issues</strong><span>{simulation.eligible_count} eligible responses from {simulation.population_count} checked</span></div><ul><li>{simulation.would_reuse_count} existing issues reused</li><li>{simulation.blast_suppressed_count} responses held by the blast-radius limit</li><li>{simulation.restricted_excluded_count} restricted responses excluded</li></ul><small>Observed {new Date(simulation.observed_at).toLocaleString()} · expires {new Date(simulation.expires_at).toLocaleString()}</small></section> : <section className="forms-policy-impact forms-policy-impact--empty"><h4>No current simulation</h4><p>Simulate this exact policy revision before requesting approval.</p></section>}
    {policy.status === "SUSPENDED" && !rollbackTarget && <Notice tone="warning">No earlier version of this policy is available for rollback. Create a new policy draft instead.</Notice>}
  </article>;
}
function dominantAction(policy: FormResponsePolicy, simulation?: FormPolicySimulation, rollbackTarget?: FormResponsePolicy) { if (policy.status === "DRAFT") return simulation ? "Send for approval" : "Simulate impact"; if (policy.status === "PENDING_APPROVAL") return "Approve policy"; if (policy.status === "APPROVED") return "Activate policy"; if (policy.status === "ACTIVE") return "Suspend policy"; if (policy.status === "SUSPENDED" && rollbackTarget) return `Create rollback from revision ${rollbackTarget.version}`; return undefined; }
function statusLabel(value: string) { return value.split("_").map((part) => part.charAt(0) + part.slice(1).toLowerCase()).join(" "); }
function statusTone(status: string): "neutral" | "info" | "success" | "warning" { return status === "ACTIVE" ? "success" : status === "PENDING_APPROVAL" || status === "APPROVED" ? "info" : status === "SUSPENDED" ? "warning" : "neutral"; }
function actionNotice(status: string) { return status === "PENDING_APPROVAL" ? "Policy sent for independent approval." : status === "APPROVED" ? "Policy approved. A permitted user must still activate it." : status === "ACTIVE" ? "Policy activated for its approved response population." : status === "SUSPENDED" ? "Policy suspended. No new responses will be enforced by it." : "A rollback draft was created for review."; }
function message(cause: unknown, fallback: string) { return cause instanceof Error && cause.message.trim() && !/^Request failed with \d+$/i.test(cause.message.trim()) ? cause.message : fallback; }

export default FormPoliciesView;
