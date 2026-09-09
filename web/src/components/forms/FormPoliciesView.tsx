import { useEffect, useRef, useState } from "react";
import { apiErrorKind } from "../../http";
import {
  listFormPolicyExecutions, type FormPolicyExecution, listFormPolicyAutomationChoices, type FormAutomationChoice, activateFormResponsePolicy, approveFormResponsePolicy, createFormResponsePolicy, listFormResponsePolicies,
  rollbackFormResponsePolicy, simulateFormResponsePolicy, submitFormResponsePolicy, suspendFormResponsePolicy,
  type CreateFormResponsePolicyInput, type FormPolicySimulation, type FormResponsePolicy,
} from "../../formPoliciesApi";
import { loadFormTemplates } from "../../monitoringApi";
import { Button, EmptyState, FocusedDialog, Notice, SelectableRecord, StatusBadge, Surface } from "../ui";
import { FormPolicyEditor, type PolicyFormChoice } from "./FormPolicyEditor";
import "./form-policies.css";

type LoadState = "loading" | "live" | "sign-in-required" | "error";

export function FormPoliciesView() {
  const [items, setItems] = useState<FormResponsePolicy[]>([]);
  const [selectedID, setSelectedID] = useState<string>();
  const [state, setState] = useState<LoadState>("loading");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [editingInput, setEditingInput] = useState<CreateFormResponsePolicyInput>();
  const [automationChoices, setAutomationChoices] = useState<FormAutomationChoice[]>([]);
  const [automationLoading, setAutomationLoading] = useState(false);
  const [automationError, setAutomationError] = useState("");
  const automationRequest = useRef(0);
  const automationForm = useRef<{id: string; version: number} | undefined>(undefined);
  const [creating, setCreating] = useState(false);
  const [busy, setBusy] = useState("");
  const [simulations, setSimulations] = useState<Record<string, FormPolicySimulation>>({});
  const [forms, setForms] = useState<PolicyFormChoice[]>([]);
  const [formsLoading, setFormsLoading] = useState(true);
  const [formsError, setFormsError] = useState("");
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
  async function refreshForms() {
    setFormsLoading(true); setFormsError("");
    try {
      const values = await loadFormTemplates();
      setForms(values.filter((form) => form.status === "ACTIVE" && (form.scoring_mode !== undefined && form.scoring_mode !== "NONE" || form.fields.some((field) => (field.assessment?.mode === "MANUAL" || field.assessment?.mode === "AUTOMATIC_REVIEW") && field.assessment.weight > 0 && (field.assessment.rubric?.length ?? 0) > 0))).map((form) => ({
        id: form.id, name: form.name, code: form.code, version: form.version, requiresBankAssessment: form.fields.some((field) => field.assessment?.mode === "MANUAL" || field.assessment?.mode === "AUTOMATIC_REVIEW"),
      })));
    } catch (cause) {
      setForms([]);
      setFormsError(message(cause, "Approved scoring forms cannot be checked right now."));
    } finally {
      setFormsLoading(false);
    }
  }
  useEffect(() => { void refresh(); void refreshForms(); }, []);

  async function refreshAutomation(id: string, version: number) {
    const request = ++automationRequest.current;
    automationForm.current = {id,version}; setAutomationChoices([]); setAutomationLoading(true); setAutomationError("");
    try { const values = await listFormPolicyAutomationChoices(id, version); if(request === automationRequest.current) setAutomationChoices(values); }
    catch(cause) { if(request === automationRequest.current) setAutomationError(message(cause,"Automation policies could not be checked.")); }
    finally { if(request === automationRequest.current) setAutomationLoading(false); }
  }
  function openEditor(policy?: FormResponsePolicy) {
    setEditingInput(policy ? draftInput(policy) : undefined);
    const form = policy ? {id:policy.eligibility.form_template_id,version:policy.eligibility.form_template_version} : forms[0];
    if(form) void refreshAutomation(form.id,form.version);
    setCreating(true);
  }

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
    } catch (cause) { setError(message(cause, "The policy command could not be confirmed. Reload the policy before retrying.")); }
    finally { setBusy(""); }
  }

  return <section className="forms-policies" aria-labelledby="form-policies-title">
    <header className="forms-policies__heading"><div><p>Governed automation</p><h2 id="form-policies-title">Response policies</h2><span>Create an issue from a completed response that meets an approved concern threshold. Simulation and independent approval are required before activation.</span></div>{state === "live" && <Button variant="primary" onPress={() => openEditor()}>Create policy</Button>}</header>
    {error && state === "live" && <Notice tone="error">{error} Policies already shown remain available.</Notice>}
    {notice && <Notice>{notice}</Notice>}
    {state === "loading" && <Surface><p role="status">Loading response policies for this legal entity…</p></Surface>}
    {state === "sign-in-required" && (
      <EmptyState population="Response policies in this legal entity" title="Sign in to review response policies" description="Your session ended before the policy population could be loaded." action={<Button onPress={() => void refresh()}>Retry loading policies</Button>}/>
    )}
    {state === "error" && <div role="alert"><EmptyState population="Response policies in this legal entity" title="Response policies could not be loaded" description={error} action={<Button onPress={() => void refresh()}>Retry loading policies</Button>}/></div>}
    {state === "live" && items.length === 0 && (
      <EmptyState population="Response policies in this legal entity" title="No response policies have been created" description="Create a draft to select the approved form, eligible subjects, issue handling and outcome check." action={<Button variant="primary" onPress={() => openEditor()}>Create policy</Button>}/>
    )}
    {state === "live" && items.length > 0 && <div className="forms-policies__layout">
      <nav className="forms-policies__list" aria-label="Response policies">{items.map((policy) => <SelectableRecord key={policy.id} title={policy.name} metadata={`${statusLabel(policy.status)} · ${policy.code} · form revision ${policy.eligibility.form_template_version}`} description={policy.rollout === "SHADOW" ? "Simulation only" : "Creates governed issues"} isSelected={policy.id === selected?.id} onPress={() => setSelectedID(policy.id)}/>)}</nav>
      {selected && (
        <PolicyDetail policy={selected} simulation={simulations[selected.id]} rollbackTarget={rollbackTarget} busy={busy === selected.id} onAction={() => void act(selected, rollbackTarget?.id)} onRevise={() => openEditor(selected)} forms={forms}/>
      )}
    </div>}
    {creating && <FocusedDialog label="Create response policy" size="wide" onClose={() => setCreating(false)}><FormPolicyEditor initialInput={editingInput} automationChoices={automationChoices} automationLoading={automationLoading} automationError={automationError} onSelectForm={(id,version) => void refreshAutomation(id,version)} onRetryAutomation={() => { const form=automationForm.current; if(form) void refreshAutomation(form.id,form.version); }} forms={forms} formsLoading={formsLoading} formsError={formsError} onRetryForms={() => void refreshForms()} onCancel={() => setCreating(false)} onCreate={create} busy={busy === "create"}/></FocusedDialog>}
  </section>;
}

function PolicyDetail({ policy, simulation, rollbackTarget, busy, onAction, onRevise, forms }: { forms: PolicyFormChoice[]; onRevise: () => void; policy: FormResponsePolicy; simulation?: FormPolicySimulation; rollbackTarget?: FormResponsePolicy; busy: boolean; onAction: () => void }) {
  const action = dominantAction(policy, simulation, rollbackTarget);
  const [history,setHistory]=useState<FormPolicyExecution[]>([]);
  const [historyState,setHistoryState]=useState<"loading"|"live"|"error">("loading");
  const [historyReload,setHistoryReload]=useState(0);
  useEffect(() => {let current=true;setHistoryState("loading");setHistory([]);
    void listFormPolicyExecutions(policy.id).then((values) => {if(current){setHistory(values);setHistoryState("live");}},() => {if(current)setHistoryState("error");});
    return () => {current=false;};
  },[policy.id,policy.record_version,historyReload]);
  return <article className="forms-policy-detail" aria-labelledby={`policy-${policy.id}`}>
    <header><div><StatusBadge tone={statusTone(policy.status)}>{statusLabel(policy.status)}</StatusBadge><h3 id={`policy-${policy.id}`}>{policy.name}</h3><p>{policy.purpose}</p></div><Button variant="quiet" onPress={onRevise}>Revise policy</Button>{action && <Button variant="primary" isLoading={busy} onPress={onAction}>{action}</Button>}</header>
    <dl className="forms-policy-facts"><div><dt>Form scope</dt><dd>{policy.eligibility.form_template_id} · revision {policy.eligibility.form_template_version}</dd></div><div><dt>Result used</dt><dd>{policy.eligibility.result_basis === "BANK_ASSESSED" ? "Completed assessment" : "Automatic submission result"}</dd></div><div><dt>Concern threshold</dt><dd>{policy.eligibility.bands?.map(statusLabel).join(", ") || "Score threshold"}</dd></div><div><dt>Rollout</dt><dd>{policy.rollout === "SHADOW" ? "Simulation only" : "Create governed issues"}</dd></div><div><dt>Blast radius</dt><dd>{policy.blast_radius.per_run} per run · {policy.blast_radius.per_day} per day</dd></div><div><dt>Outcome check</dt><dd>After {policy.outcome_contract.check_after_minutes} minutes</dd></div><div><dt>Policy revision</dt><dd>{policy.version}</dd></div></dl>
    <section><h4>Approval and timing</h4><p>{policy.checker_id ? "Independent approval recorded." : "Independent approval is pending."} The issue owner and outcome reviewer follow the current approved responsibilities.</p><p>{policy.effective_from ? `Eligible from ${new Date(policy.effective_from).toLocaleString()}.` : "Eligible after activation."} {policy.effective_until ? `Expires ${new Date(policy.effective_until).toLocaleString()}.` : "No planned expiry."}</p></section><section><h4>Issue handling</h4><strong>{policy.action.title_template}</strong><p>{policy.action.requested_handling}</p></section>
    {simulation ? <section className="forms-policy-impact" aria-label="Latest simulation impact"><div><strong>{simulation.would_create_count} new issues</strong><span>{(simulation.result_basis ?? policy.eligibility.result_basis) === "BANK_ASSESSED" ? "Completed assessments checked" : "Automatic submission results checked"}</span><span>{simulation.eligible_count} eligible responses from {simulation.population_count} checked</span></div><ul><li>{simulation.would_reuse_count} existing issues reused</li><li>{simulation.blast_suppressed_count} responses held by the blast-radius limit</li><li>{simulation.restricted_excluded_count} restricted responses excluded</li></ul><small>Observed {new Date(simulation.observed_at).toLocaleString()} · expires {new Date(simulation.expires_at).toLocaleString()}</small></section> : <section className="forms-policy-impact forms-policy-impact--empty"><h4>No current simulation</h4><p>Simulate this exact policy revision before requesting approval.</p></section>}
    <section aria-label="Policy execution history"><h4>Latest policy activity</h4>
      <p>The latest 50 execution records for this policy revision show which result was checked and what happened.</p>
      {historyState === "loading" && <p role="status">Loading policy activity...</p>}
      {historyState === "error" && <><Notice tone="warning">Policy activity could not be checked.</Notice><Button variant="quiet" onPress={() => setHistoryReload((value) => value+1)}>Retry policy activity</Button></>}
      {historyState === "live" && (history.length ? <ul>{history.map((item) => <li key={item.id}><strong>{executionLabel(item.state)}</strong> · {item.result_basis === "BANK_ASSESSED" ? `Assessment revision ${item.assessment_version}` : "Automatic submission result"} · {new Date(item.created_at).toLocaleString()}</li>)}</ul> : <p>No execution records were found for this policy revision. Activate the approved policy to check new results.</p>)}
    </section>
    {policy.status === "SUSPENDED" && !rollbackTarget && <Notice tone="warning">No earlier version of this policy is available for rollback. Create a new policy draft instead.</Notice>}
  </article>;
}
function dominantAction(policy: FormResponsePolicy, simulation?: FormPolicySimulation, rollbackTarget?: FormResponsePolicy) { if (policy.status === "DRAFT") return simulation ? "Send for approval" : "Simulate impact"; if (policy.status === "PENDING_APPROVAL") return "Approve policy"; if (policy.status === "APPROVED") return "Activate policy"; if (policy.status === "ACTIVE") return "Suspend policy"; if (policy.status === "SUSPENDED" && rollbackTarget) return `Create rollback from revision ${rollbackTarget.version}`; return undefined; }
function statusLabel(value: string) { return value.split("_").map((part) => part.charAt(0) + part.slice(1).toLowerCase()).join(" "); }
function statusTone(status: string): "neutral" | "info" | "success" | "warning" { return status === "ACTIVE" ? "success" : status === "PENDING_APPROVAL" || status === "APPROVED" ? "info" : status === "SUSPENDED" ? "warning" : "neutral"; }
function actionNotice(status: string) { return status === "PENDING_APPROVAL" ? "Policy sent for independent approval." : status === "APPROVED" ? "Policy approved. A permitted user must still activate it." : status === "ACTIVE" ? "Policy activated for its approved response population." : status === "SUSPENDED" ? "Policy suspended. No new responses will be enforced by it." : "A rollback draft was created for review."; }
function message(cause: unknown, fallback: string) { return cause instanceof Error && cause.message.trim() && !/^Request failed with \d+$/i.test(cause.message.trim()) ? cause.message : fallback; }

export default FormPoliciesView;

function draftInput(policy: FormResponsePolicy): CreateFormResponsePolicyInput {
  return {code:policy.code,name:policy.name,purpose:policy.purpose,create_automation_policy:true,automation_policy_id:"",automation_policy_version:1,eligibility:policy.eligibility,action:policy.action,blast_radius:policy.blast_radius,outcome_contract:policy.outcome_contract,rollout:policy.rollout,effective_from:policy.effective_from?.slice(0,16),effective_until:policy.effective_until?.slice(0,16)};
}

function executionLabel(state: FormPolicyExecution["state"]) {return {NOT_MATCHED:"Threshold not met",SHADOW:"Matched in shadow mode",APPLIED:"Issue created",REUSED:"Existing issue reused",BLAST_SUPPRESSED:"Issue limit reached",FAILED:"Handling failed"}[state];}
