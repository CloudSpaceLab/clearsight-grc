import { useEffect, useMemo, useState } from "react";
import { loadFormTemplatePage, loadFormTemplateRevision } from "../formsApi";
import type { FormTemplate } from "../formsTypes";
import { apiErrorKind } from "../http";
import { applyMatterFormRemediation, createMatterFormRemediation, loadMatterFormRemediations, sendMatterFormRemediation } from "../matterFormRemediationApi";
import type { MatterFormRemediationState } from "../matterFormRemediationApi";
import type { MatterAggregate } from "../types";
import type { MatterOperation } from "../matterOperationsApi";
import { Button, FocusedSheet, Notice, SelectField, TextArea, TextField } from "./ui";

type Props = {
  aggregate: MatterAggregate;
  operations: MatterOperation[];
  onUpdated: (value: MatterAggregate) => void | Promise<void>;
  onOpenRequest?: (requestID: string) => void;
  onMappingsChange: (items: string[]) => void;
};

function factKey(label: string) { return label.trim().toLowerCase().replace(/[^a-z0-9]+/g, "_").replace(/^_+|_+$/g, ""); }
function dateInput(days: number) { const value = new Date(Date.now() + days * 86_400_000); return value.toISOString().slice(0, 10); }
function endOfDay(value: string) { return new Date(`${value}T23:59:59.000Z`).toISOString(); }
function routeExpiry(deadline: string) { const due = new Date(endOfDay(deadline)); const bounded = new Date(Math.min(due.getTime(), Date.now() + 7 * 86_400_000)); return bounded.toISOString(); }

export function MatterFormRemediationPanel({ aggregate, operations, onUpdated, onOpenRequest, onMappingsChange }: Props) {
  const [states, setStates] = useState<MatterFormRemediationState[]>([]);
  const [forms, setForms] = useState<FormTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [formKey, setFormKey] = useState("");
  const [contractID, setContractID] = useState("");
  const [email, setEmail] = useState("");
  const [deadline, setDeadline] = useState(dateInput(14));
  const [fieldByMissing, setFieldByMissing] = useState<Record<string, string>>({});
  const [rationaleByBinding, setRationaleByBinding] = useState<Record<string, string>>({});
  const ownerCanAct = operations.some((operation) => operation.command === "matter.context.change" && operation.can_act);
  const reviewerCanAct = operations.some((operation) => operation.command === "matter.outcome.record" && operation.can_act);
  const programID = aggregate.links.find((link) => link.program_id)?.program_id ?? "";
  const missing = aggregate.matter.missing_facts.map(String);
  const selectedForm = forms.find((form) => `${form.id}:${form.version}` === formKey);
  const activeContracts = aggregate.verification_contracts.filter((contract) => contract.status === "ACTIVE");

  async function reload() {
    setLoading(true); setError("");
    try {
      const values = await loadMatterFormRemediations(aggregate.matter.id);
      setStates(values);
      onMappingsChange(values.flatMap((state) => state.binding.mappings.map((mapping) => mapping.missing_item)));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Linked form requests could not be loaded.");
    } finally { setLoading(false); }
  }

  useEffect(() => { void reload(); }, [aggregate.matter.id]);
  useEffect(() => {
    if (!creating) return;
    let cancelled = false;
    void (async () => {
      try {
        const page = await loadFormTemplatePage({ status: "ACTIVE", use: "MATTER_REMEDIATION", limit: 100 });
        const revisions = await Promise.all(page.items.flatMap((item) => item.active_status === "ACTIVE" && item.active_version ? [loadFormTemplateRevision(item.template.id, item.active_version)] : []));
        if (!cancelled) setForms(revisions.filter((form) => form.is_current && form.status === "ACTIVE"));
      } catch (cause) { if (!cancelled) setError(cause instanceof Error ? cause.message : "Approved forms could not be loaded."); }
    })();
    return () => { cancelled = true; };
  }, [creating]);

  const mappedItems = useMemo(() => new Set(states.flatMap((state) => state.binding.mappings.map((mapping) => mapping.missing_item.toLowerCase()))), [states]);
  const unmapped = missing.filter((item) => !mappedItems.has(item.toLowerCase()));

  async function createAndSend() {
    if (!selectedForm || !programID || !contractID || !email || !deadline || unmapped.some((item) => !fieldByMissing[item])) return;
    setBusy(true); setError("");
    try {
      const binding = await createMatterFormRemediation(aggregate.matter.id, {
        legalEntityID: aggregate.matter.legal_entity_id ?? "", expectedMatterVersion: aggregate.matter.version, programID,
        formTemplateID: selectedForm.id, formTemplateVersion: selectedForm.version,
        mappings: unmapped.map((item) => ({ field_id: fieldByMissing[item]!, missing_item: item, fact_key: factKey(item) })),
        verificationContractID: contractID,
      });
      await sendMatterFormRemediation(aggregate.matter.id, binding.id, { bindingVersion: binding.version, email, deadline: endOfDay(deadline), routeExpiresAt: routeExpiry(deadline) });
      setCreating(false); setFormKey(""); setContractID(""); setEmail(""); setFieldByMissing({});
      await reload();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The linked form could not be sent.");
    } finally { setBusy(false); }
  }

  async function applyResponse(state: MatterFormRemediationState) {
    if (!state.response) return;
    setBusy(true); setError("");
    try {
      const result = await applyMatterFormRemediation(aggregate.matter.id, state.binding.id, {
        bindingVersion: state.binding.version, expectedMatterVersion: aggregate.matter.version,
        responseRevisionID: state.response.id, rationale: rationaleByBinding[state.binding.id] ?? "",
      });
      await onUpdated(result.matter); await reload();
    } catch (cause) {
      setError(apiErrorKind(cause) === "conflict" ? "This issue changed after the response loaded. Reload the issue before applying the response." : cause instanceof Error ? cause.message : "The response could not be applied.");
    } finally { setBusy(false); }
  }

  return <article className="matter-record-panel matter-form-remediation-panel">
    <div className="matter-record-section-heading"><div><span className="eyebrow">Linked evidence</span><h2>Approved form requests</h2><p>Use one approved form response to supply mapped missing information. The outcome check and closure remain separate.</p></div>
      {ownerCanAct && unmapped.length > 0 && <Button variant="primary" onPress={() => { setCreating(true); setError(""); }}>Send linked form</Button>}
    </div>
    {loading && <p role="status">Checking linked requests and responses…</p>}
    {!loading && states.length === 0 && <p>No approved form request is linked to this issue. {ownerCanAct ? "Send one to collect the missing information without creating a second evidence record." : "The current issue owner can link an approved form."}</p>}
    {states.length > 0 && <ul className="matter-form-remediation-list">{states.map((state) => <li key={state.binding.id}>
      <div><strong>{state.request?.title ?? `Form revision ${state.binding.form_template_version}`}</strong><span>{state.next_action} · {state.binding.mappings.length} mapped item{state.binding.mappings.length === 1 ? "" : "s"}</span></div>
      {state.request && onOpenRequest && state.next_action === "Open response" && <Button variant="secondary" onPress={() => onOpenRequest(state.request!.id)}>Open response</Button>}
      {state.request && state.next_action === "Request correction" && <div className="matter-form-review-action"><Notice tone="warning">This response does not meet the binding's score threshold. The issue remains open and the mapped information has not been applied.</Notice>{onOpenRequest && <Button variant="secondary" onPress={() => onOpenRequest(state.request!.id)}>Review response</Button>}</div>}
      {state.response && !state.application && reviewerCanAct && state.next_action === "Review evidence" && <div className="matter-form-review-action"><TextArea label="Review basis" value={rationaleByBinding[state.binding.id] ?? ""} onChange={(value) => setRationaleByBinding((current) => ({ ...current, [state.binding.id]: value }))} description="Explain why this exact final response supplies the mapped information."/><Button variant="primary" isDisabled={busy || (rationaleByBinding[state.binding.id]?.trim().length ?? 0) < 20} onPress={() => void applyResponse(state)}>Apply response</Button></div>}
      {state.application && <span className="status-pill success">Response applied · outcome check required</span>}
    </li>)}</ul>}
    {error && <Notice tone="error"><strong>Linked form action could not be completed.</strong> {error}</Notice>}
    {creating && <FocusedSheet label="Send linked form" closeLabel={busy ? "Linked form is being sent" : "Close linked form setup"} size="wide" isDismissable={!busy} onClose={() => setCreating(false)}>
      <div className="cs-sheet-heading"><span className="eyebrow">Issue evidence</span><h2>Send an approved form</h2><p>Map every remaining item to one response field. This exact revision and mapping cannot change after send.</p></div>
      {!programID && <Notice tone="error"><strong>Link a Program first.</strong> This issue must be linked to the Program that owns the remediation.</Notice>}
      <SelectField label="Approved form revision" value={formKey || undefined} placeholder={forms.length ? "Choose an active form" : "No active remediation form available"} isRequired allowsEmpty={false} options={forms.map((form) => ({ id: `${form.id}:${form.version}`, label: `${form.name} · v${form.version}` }))} onChange={(value) => { setFormKey(value ?? ""); setFieldByMissing({}); }}/>
      {selectedForm && unmapped.map((item) => <SelectField key={item} label={`Response field for ${item}`} value={fieldByMissing[item]} placeholder="Choose one response field" isRequired allowsEmpty={false} options={selectedForm.fields.filter((field) => !Object.values(fieldByMissing).includes(field.id) || fieldByMissing[item] === field.id).map((field) => ({ id: field.id, label: field.label }))} onChange={(value) => setFieldByMissing((current) => ({ ...current, [item]: value ?? "" }))}/>)}
      <SelectField label="Outcome check" value={contractID || undefined} placeholder={activeContracts.length ? "Choose the check that reviews this response" : "Define an outcome check first"} isRequired allowsEmpty={false} options={activeContracts.map((contract) => ({ id: contract.id, label: contract.expected_outcome }))} onChange={(value) => setContractID(value ?? "")}/>
      <div className="matter-form-remediation-delivery"><TextField label="Recipient email" type="email" value={email} onChange={setEmail} isRequired autoComplete="email" placeholder="evidence.contact@example.com"/><TextField label="Due date" type="date" min={dateInput(1)} value={deadline} onChange={setDeadline} isRequired/></div>
      <div className="matter-form-actions"><Button variant="primary" isDisabled={busy || !selectedForm || !programID || !contractID || !email || !deadline || unmapped.length === 0 || unmapped.some((item) => !fieldByMissing[item])} onPress={() => void createAndSend()}>{busy ? "Sending…" : "Send linked form"}</Button><Button variant="quiet" onPress={() => setCreating(false)}>Cancel</Button></div>
    </FocusedSheet>}
  </article>;
}
