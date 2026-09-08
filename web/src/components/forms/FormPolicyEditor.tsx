import { useEffect, useState, type FormEvent } from "react";

import type { CreateFormResponsePolicyInput, FormPolicyRollout, FormAutomationChoice } from "../../formPoliciesApi";

import { Button, CheckboxField, Notice, SelectField, TextArea, TextField } from "../ui";



export type PolicyFormChoice = { id: string; name: string; code: string; version: number; requiresBankAssessment?: boolean };



type Props = {

  onCancel: () => void;

  onCreate: (input: CreateFormResponsePolicyInput) => void | Promise<void>;

  forms: PolicyFormChoice[];

  automationChoices?: FormAutomationChoice[];

  automationLoading?: boolean;

  automationError?: string;

  onSelectForm?: (id: string, version: number) => void;

  onRetryAutomation?: () => void;

  initialInput?: CreateFormResponsePolicyInput;

  formsLoading?: boolean;

  formsError?: string;

  onRetryForms?: () => void;

  busy?: boolean;

};



export function FormPolicyEditor({ onCancel, onCreate, forms, formsLoading = false, formsError = "", onRetryForms, automationChoices = [], automationLoading = false, automationError = "", onSelectForm, onRetryAutomation, initialInput, busy = false }: Props) {

  const [value, setValue] = useState<CreateFormResponsePolicyInput>(() => initialInput ?? selectDefaultForm(defaultInput, forms));

  const [error, setError] = useState("");

  function patch<K extends keyof CreateFormResponsePolicyInput>(key: K, next: CreateFormResponsePolicyInput[K]) { setValue((current) => ({ ...current, [key]: next, ...(["eligibility", "blast_radius", "outcome_contract", "rollout", "effective_from", "effective_until"].includes(key) && !current.create_automation_policy ? {create_automation_policy:true,automation_policy_id:"",automation_policy_version:1} : {}) })); setError(""); }

  useEffect(() => {

    setValue((current) => forms.some((form) => form.id === current.eligibility.form_template_id && form.version === current.eligibility.form_template_version)

      ? current

      : selectDefaultForm(current, forms));

  }, [forms]);

  function submit(event: FormEvent) {

    event.preventDefault();

    const missing = requiredFields(value).find((item) => !item.value.trim());

    if (missing) { setError(`${missing.label} is required before this policy draft can be created.`); return; }

    const selected = forms.find((form) => form.id === value.eligibility.form_template_id && form.version === value.eligibility.form_template_version);

    if (!selected) { setError("Choose a currently available approved scoring form revision before creating this policy draft."); return; }

    if (!value.create_automation_policy && !automationChoices.some((choice) => choice.id === value.automation_policy_id && choice.version === value.automation_policy_version && choice.status === "ACTIVE")) { setError("Choose an active automation policy or create guardrails for this response policy."); return; }

    const effectiveFrom = toISOString(value.effective_from);

    const effectiveUntil = toISOString(value.effective_until);

    if (effectiveFrom && effectiveUntil && effectiveUntil <= effectiveFrom) {

      setError("Effective until must be later than effective from.");

      return;

    }

    void onCreate({ ...value, code: normalizedCode(value.code), name: value.name.trim(), purpose: value.purpose.trim(), effective_from: effectiveFrom, effective_until: effectiveUntil });

  }

  const selectedForm = formRefKey(value.eligibility.form_template_id, value.eligibility.form_template_version);

  return <form className="forms-policy-editor" noValidate onSubmit={submit}>

    <header><p>New response policy</p><h2>Create a response policy</h2><span>Define which completed responses create an issue. Activation requires a current simulation and independent approval.</span></header>

    {error && <Notice tone="error">{error}</Notice>}

    <section aria-labelledby="policy-identity-title"><h3 id="policy-identity-title">Policy purpose</h3><div className="forms-policy-control-grid">

      <TextField label="Policy name" value={value.name} onChange={(name) => patch("name", name)} maxLength={160}/>

      <TextField label="Policy code" description="Use a stable lowercase code; spaces become hyphens." value={value.code} onChange={(code) => patch("code", code)} maxLength={64}/>

      <div className="forms-policy-control-grid__full"><TextArea label="Purpose" value={value.purpose} onChange={(purpose) => patch("purpose", purpose)} rows={3} maxLength={1000}/></div>

    </div></section>

    <section aria-labelledby="policy-population-title"><h3 id="policy-population-title">Response population</h3><p>Choose the approved form revision and the bank records whose responses this policy covers.</p><div className="forms-policy-control-grid">

      {formsLoading ? <div className="forms-policy-control-grid__full"><Notice>Loading approved scoring forms…</Notice></div>

        : formsError ? <div className="forms-policy-control-grid__full"><Notice tone="error">{formsError}</Notice>{onRetryForms && <Button variant="quiet" type="button" onPress={onRetryForms}>Retry form choices</Button>}</div>

          : forms.length === 0 ? <div className="forms-policy-control-grid__full"><Notice tone="warning">No active approved scoring forms are available. Approve and activate a scored form before creating a response policy.</Notice></div>

            : <SelectField label="Approved form revision" value={selectedForm} placeholder="Choose approved scored form" allowsEmpty={false} options={forms.map((form) => ({ id: formRefKey(form.id, form.version), label: `${form.name} · ${form.code} · revision ${form.version}` }))} onChange={(next) => {

                const selected = forms.find((form) => formRefKey(form.id, form.version) === next);

                if (selected) { setValue((current) => ({...current, create_automation_policy: true, automation_policy_id: "", eligibility: {...current.eligibility, form_template_id: selected.id, form_template_version: selected.version, result_basis: selected.requiresBankAssessment ? "BANK_ASSESSED" : "AUTOMATIC"}})); onSelectForm?.(selected.id, selected.version); }

              }}/>} 

      <SelectField label="Responses about" value={value.eligibility.subject_types[0] ?? ""} placeholder="Choose bank records" allowsEmpty={false} options={[{id:"VENDOR_RELATIONSHIP",label:"Vendor services"},{id:"PROGRAM",label:"Programs"},{id:"MATTER",label:"Issues and changes"}]} onChange={(subject) => { if(subject) patch("eligibility", {...value.eligibility,subject_types:[subject]}); }}/>

      <SelectField label="Result used for issue handling" value={value.eligibility.result_basis ?? "AUTOMATIC"} placeholder="Choose result" allowsEmpty={false} options={[{id:"AUTOMATIC",label:"Automatic submission result"},{id:"BANK_ASSESSED",label:"Completed bank assessment"}]} onChange={(basis) => {if(basis) patch("eligibility",{...value.eligibility,result_basis:basis as "AUTOMATIC" | "BANK_ASSESSED"}); }}/>

      {value.eligibility.result_basis === "BANK_ASSESSED" && <p>Required bank reviews must be complete before this policy can create an issue. Minimum coverage applies to the assessed result.</p>}

      <NumberField label="Minimum answer coverage (%)" value={Math.round(value.eligibility.minimum_coverage * 100)} min={0} max={100} onChange={(minimum_coverage) => patch("eligibility", { ...value.eligibility, minimum_coverage: minimum_coverage / 100 })}/>

    </div><div className="forms-policy-checks"><CheckboxField label="Use only the current response revision" description="Earlier response revisions remain reconstructable but cannot trigger this policy." isSelected={value.eligibility.current_only} onChange={(current_only) => patch("eligibility", { ...value.eligibility, current_only })}/>{(["HIGH", "CRITICAL"] as const).map((band) => <CheckboxField key={band} label={`${title(band)} concern responses`} isSelected={value.eligibility.bands?.includes(band) ?? false} onChange={(selected) => patch("eligibility", { ...value.eligibility, bands: selected ? [...(value.eligibility.bands ?? []), band] : (value.eligibility.bands ?? []).filter((item) => item !== band) })}/>)}</div></section>

    <section aria-labelledby="policy-automation-title"><h3 id="policy-automation-title">Automation guardrail</h3><div className="forms-policy-control-grid">

      <SelectField label="Automation policy" value={value.create_automation_policy ? "create" : value.automation_policy_id} placeholder="Choose automation policy" allowsEmpty={false} options={[{id:"create",label:"Create guardrails for this response policy"},...automationChoices.filter((choice) => choice.status === "ACTIVE" && (!choice.effective_until || new Date(choice.effective_until).getTime() > Date.now())).map((choice) => ({id:choice.id,label:`${choice.name} · ${title(choice.status)} · revision ${choice.version}`}))]} onChange={(selected) => {

        if(selected === "create") { patch("create_automation_policy",true); return; }

        const choice = automationChoices.find((item) => item.id === selected);

        if(choice) setValue((current) => ({...current,create_automation_policy:false,automation_policy_id:choice.id,automation_policy_version:choice.version,eligibility:choice.eligibility,blast_radius:choice.blast_radius,outcome_contract:choice.outcome_contract,rollout:choice.rollout,effective_from:choice.effective_from?.slice(0,16),effective_until:choice.effective_until?.slice(0,16)}));

      }}/>

      <div className="forms-policy-control-grid__full">{value.create_automation_policy ? <p>The limits and outcome check below will be saved with this policy. Simulation and independent approval apply to both before activation.</p> : <p>{automationChoices.find((choice) => choice.id === value.automation_policy_id)?.purpose} Used by {automationChoices.find((choice) => choice.id === value.automation_policy_id)?.use_count ?? "an unknown number of"} response policy revisions.</p>}

      {automationLoading && <p role="status">Checking automation policies for this form revision…</p>}

      {automationError && <><Notice tone="warning">{automationError} You can still create guardrails with this response policy.</Notice>{onRetryAutomation && <Button type="button" variant="quiet" onPress={onRetryAutomation}>Retry automation policies</Button>}</>}

      {!automationLoading && !automationError && automationChoices.length === 0 && <p>No automation policies were found for this form revision. Create guardrails with this response policy.</p>}

      </div>

      <NumberField label="Maximum new issues per run" value={value.blast_radius.per_run} min={1} max={1_000} onChange={(per_run) => patch("blast_radius", { ...value.blast_radius, per_run, per_day: Math.max(per_run, value.blast_radius.per_day) })}/>

      <NumberField label="Maximum new issues per day" value={value.blast_radius.per_day} min={value.blast_radius.per_run} max={10_000} onChange={(per_day) => patch("blast_radius", { ...value.blast_radius, per_day })}/>

      <SelectField label="Rollout mode" value={value.rollout} placeholder="Choose rollout" allowsEmpty={false} options={[{ id: "SHADOW", label: "Shadow — record impact only" }, { id: "ENFORCE", label: "Enforce — create governed issues" }]} onChange={(rollout) => { if (rollout) patch("rollout", rollout as FormPolicyRollout); }}/>

      <TextField label="Effective from" description="Leave blank to make the approved policy eligible as soon as it is activated." type="datetime-local" value={value.effective_from ?? ""} onChange={(effective_from) => patch("effective_from", effective_from || undefined)}/>

      <TextField label="Effective until" description="Leave blank when the approved policy has no planned end date." type="datetime-local" value={value.effective_until ?? ""} onChange={(effective_until) => patch("effective_until", effective_until || undefined)}/>

    </div></section>

    <section aria-labelledby="policy-matter-title"><h3 id="policy-matter-title">Issue and outcome</h3><div className="forms-policy-control-grid">

      <SelectField label="Issue type" value={value.action.type} placeholder="Choose issue type" allowsEmpty={false} options={[{ id: "VENDOR_DEFICIENCY", label: "Vendor deficiency" }, { id: "VENDOR_REVIEW", label: "Vendor review" }, { id: "CONTROL_GAP", label: "Control gap" }, { id: "RISK_SITUATION", label: "Risk situation" }, { id: "AUDIT_FINDING", label: "Audit finding" }, { id: "EXCEPTION", label: "Exception" }, { id: "FAILED_VERIFICATION", label: "Failed verification" }, { id: "EVIDENCE_CONTRADICTION", label: "Evidence contradiction" }, { id: "KRI_BREACH", label: "KRI breach" }]} onChange={(type) => { if (type) patch("action", { ...value.action, type }); }}/>

      <TextField label="Issue title" value={value.action.title_template} onChange={(title_template) => patch("action", { ...value.action, title_template })}/>

      <NumberField label="Issue priority" value={value.action.priority} min={1} max={5} onChange={(priority) => patch("action", { ...value.action, priority })}/>

      <div className="forms-policy-control-grid__full"><TextArea label="Issue summary" value={value.action.summary_template} onChange={(summary_template) => patch("action", { ...value.action, summary_template })} rows={2}/></div>

      <div className="forms-policy-control-grid__full"><TextArea label="Required handling" value={value.action.requested_handling} onChange={(requested_handling) => patch("action", { ...value.action, requested_handling })} rows={2}/></div>

      <div className="forms-policy-control-grid__full"><TextArea label="Expected outcome" value={value.outcome_contract.expected_outcome} onChange={(expected_outcome) => patch("outcome_contract", { ...value.outcome_contract, expected_outcome })} rows={2}/></div>

      <NumberField label="Check outcome after (minutes)" value={value.outcome_contract.check_after_minutes} min={1} max={525_600} onChange={(check_after_minutes) => patch("outcome_contract", { ...value.outcome_contract, check_after_minutes })}/>

      <SelectField label="If the outcome is not achieved" value={value.outcome_contract.failure_response} placeholder="Choose follow-up" allowsEmpty={false} options={[{ id: "ESCALATE", label: "Escalate using the current hierarchy" }, { id: "REOPEN", label: "Reopen the issue for handling" }, { id: "CREATE_MATTER", label: "Create a failed-verification issue" }, { id: "BLOCK_CLOSE", label: "Block closure until verification succeeds" }]} onChange={(failure_response) => { if (failure_response) patch("outcome_contract", { ...value.outcome_contract, failure_response }); }}/>

    </div></section>

    <footer><Button variant="primary" type="submit" isLoading={busy} isDisabled={formsLoading || Boolean(formsError) || forms.length === 0}>Create policy draft</Button><Button variant="quiet" type="button" onPress={onCancel} isDisabled={busy}>Cancel</Button></footer>

  </form>;

}



const defaultInput: CreateFormResponsePolicyInput = {

  create_automation_policy: true, code: "", name: "", purpose: "", automation_policy_id: "", automation_policy_version: 1,

  eligibility: { form_template_id: "", form_template_version: 1, subject_types: ["VENDOR_RELATIONSHIP"], current_only: true, minimum_coverage: 0.8, bands: ["HIGH", "CRITICAL"] },

  action: { type: "VENDOR_DEFICIENCY", priority: 2, title_template: "", summary_template: "A completed response met the approved concern threshold.", requested_handling: "" },

  blast_radius: { per_run: 10, per_day: 50 }, outcome_contract: { expected_outcome: "", check_after_minutes: 1440, failure_response: "ESCALATE" }, rollout: "SHADOW",

};

function requiredFields(value: CreateFormResponsePolicyInput) { return [{ label: "Policy name", value: value.name }, { label: "Policy code", value: value.code }, { label: "Purpose", value: value.purpose }, { label: "Approved form revision", value: value.eligibility.form_template_id }, { label: "Issue title", value: value.action.title_template }, { label: "Required handling", value: value.action.requested_handling }, { label: "Expected outcome", value: value.outcome_contract.expected_outcome }]; }

function NumberField({ label, value, min, max, onChange }: { label: string; value: number; min: number; max: number; onChange: (value: number) => void }) { return <TextField label={label} type="number" value={String(value)} min={min} max={max} onChange={(raw) => onChange(Math.min(max, Math.max(min, Number(raw) || min)))}/>; }

function selectDefaultForm(value: CreateFormResponsePolicyInput, forms: PolicyFormChoice[]): CreateFormResponsePolicyInput { const form = forms[0]; return { ...value, eligibility: { ...value.eligibility, form_template_id: form?.id ?? "", form_template_version: form?.version ?? 1, result_basis: form?.requiresBankAssessment ? "BANK_ASSESSED" : "AUTOMATIC" } }; }

function formRefKey(id: string, version: number) { return id ? `${id}@${version}` : ""; }

function normalizedCode(value: string) { return value.trim().toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, ""); }

function title(value: string) { return value.charAt(0) + value.slice(1).toLowerCase(); }

function toISOString(value?: string) { return value ? new Date(value).toISOString() : undefined; }
