import { useEffect, useMemo, useState, type ReactNode } from "react";
import { loadReusableFormTemplateRefs } from "../../formsApi";
import type { ReusableFormTemplateRef } from "../../formsTypes";
import {
  createDistribution,
  loadRecipientCandidates,
  type CreateDistributionRecipient,
  type DistributionAccessPolicy,
  type DistributionDetail,
  type RecipientCandidate,
  type CreateDistributionInput,
} from "../../formsDistributionApi";
import { SelectField, TextArea, TextField } from "../ui";
import "../../forms-task11.css";

const policies: Array<{ value: DistributionAccessPolicy; label: string; detail: string }> = [
  { value: "DIRECT_LINK_EMAIL_OTP", label: "Email verification", detail: "The vendor opens a private link and verifies their email." },
  { value: "DIRECT_MAGIC_LINK", label: "Secure link", detail: "The vendor opens a private link without an additional email code." },
  { value: "SHARED_LINK_EMAIL_OTP", label: "Shared link with email verification", detail: "The vendor selects their address and verifies their email." },
];

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export type DistributionSettings = Omit<CreateDistributionInput, "subject_type" | "subject_id" | "recipients">;
type Props = { onCreated?: (value: DistributionDetail) => void; onCancel?: () => void; scopedDelivery?: {
  label: string; recipients: ReactNode; ready: boolean; submitLabel: string; onSubmit: (settings: DistributionSettings) => Promise<void>;
} };

export function DistributionComposer({ onCreated, onCancel, scopedDelivery }: Props) {
  const [templates, setTemplates] = useState<ReusableFormTemplateRef[]>([]);
  const [templateKey, setTemplateKey] = useState("");
  const [subjectType, setSubjectType] = useState("CONTROL");
  const [subjectID, setSubjectID] = useState("");
  const [title, setTitle] = useState("");
  const [purpose, setPurpose] = useState("");
  const [policy, setPolicy] = useState<DistributionAccessPolicy>("DIRECT_LINK_EMAIL_OTP");
  const [estimatedMinutes, setEstimatedMinutes] = useState(15);
  const [deadline, setDeadline] = useState(() => futureLocalDateTime(21));
  const [routeExpiry, setRouteExpiry] = useState(() => futureLocalDateTime(7));
  const [internalQuery, setInternalQuery] = useState("");
  const [candidates, setCandidates] = useState<RecipientCandidate[]>([]);
  const [recipients, setRecipients] = useState<CreateDistributionRecipient[]>([]);
  const [externalAddress, setExternalAddress] = useState("");
  const [externalLabel, setExternalLabel] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    void loadReusableFormTemplateRefs().then((values) => {
      if (!active) return;
      setTemplates(values);
      if (values[0]) {
        setTemplateKey(`${values[0].id}:${values[0].version}`);
        setTitle(values[0].name);
        setPurpose(`Provide the information and evidence requested in ${values[0].name}.`);
      }
    }).catch((cause) => active && setError(cause instanceof Error ? cause.message : "Active form revisions could not be loaded."));
    return () => { active = false; };
  }, []);

  useEffect(() => {
    if (internalQuery.trim().length < 2) {
      setCandidates([]);
      return;
    }
    let active = true;
    const timer = window.setTimeout(() => {
      void loadRecipientCandidates(internalQuery, 12).then((page) => active && setCandidates(page.items)).catch(() => active && setCandidates([]));
    }, 200);
    return () => { active = false; window.clearTimeout(timer); };
  }, [internalQuery]);

  const selectedTemplate = useMemo(() => templates.find((value) => `${value.id}:${value.version}` === templateKey), [templateKey, templates]);
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
  const targetReady = scopedDelivery ? scopedDelivery.ready : Boolean(subjectType.trim() && subjectID.trim() && recipients.some((value) => value.role === "TO"));
  const ready = Boolean(selectedTemplate && targetReady && title.trim() && purpose.trim() && validDates(deadline, routeExpiry) && estimatedMinutes >= 1 && estimatedMinutes <= 60);

  function addInternal(candidate: RecipientCandidate, role: "TO" | "CC" = "TO") {
    if (recipients.length >= 500 || recipients.some((value) => value.type === "INTERNAL_PRINCIPAL" && value.principal_id === candidate.principal_id && value.role === role)) return;
    setRecipients((current) => [...current, { role, type: "INTERNAL_PRINCIPAL", principal_id: candidate.principal_id, contact_label: candidate.display_name }]);
    setInternalQuery("");
    setCandidates([]);
  }

  function addExternal() {
    const address = externalAddress.trim().toLowerCase();
    if (!emailPattern.test(address)) {
      setError("Enter a valid external email address.");
      return;
    }
    if (recipients.length >= 500 || recipients.some((value) => value.type === "EXTERNAL_AUDIENCE" && value.address === address && value.role === "TO")) return;
    setRecipients((current) => [...current, { role: "TO", type: "EXTERNAL_AUDIENCE", address, contact_label: externalLabel.trim() || undefined }]);
    setExternalAddress("");
    setExternalLabel("");
    setError(null);
  }

  async function submit() {
    if (!ready || !selectedTemplate || busy) return;
    setBusy(true);
    setError(null);
    try {
      if (scopedDelivery) {
        await scopedDelivery.onSubmit({ form_template_id: selectedTemplate.id, form_template_version: selectedTemplate.version, title: title.trim(), purpose: purpose.trim(), access_policy: policy, estimated_minutes: estimatedMinutes, deadline: new Date(deadline).toISOString(), route_expires_at: new Date(routeExpiry).toISOString() });
        return;
      }
      const value = await createDistribution({
        form_template_id: selectedTemplate.id,
        form_template_version: selectedTemplate.version,
        subject_type: subjectType.trim(), subject_id: subjectID.trim(), title: title.trim(), purpose: purpose.trim(),
        access_policy: policy, estimated_minutes: estimatedMinutes,
        deadline: new Date(deadline).toISOString(), route_expires_at: new Date(routeExpiry).toISOString(), recipients,
      });
      onCreated?.(value);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The distribution could not be created.");
    } finally {
      setBusy(false);
    }
  }

  function selectTemplate(value?: string) {
    setTemplateKey(value ?? "");
    const next = templates.find((item) => `${item.id}:${item.version}` === value);
    if (next) {
      setTitle(next.name);
      setPurpose(`Provide the information and evidence requested in ${next.name}.`);
    }
  }

  const deliverySettings = <>
    <TextField label="Estimated time" type="number" min={1} max={60} value={String(estimatedMinutes)} onChange={(value) => setEstimatedMinutes(Number(value))} description="Minutes shown to the vendor."/>
    <TextField label="Link expiry" type="datetime-local" value={routeExpiry} onChange={setRouteExpiry} description="Must be no later than the response deadline."/>
    <div className="forms-task-span"><SelectField label="Recipient verification" value={policy} placeholder="Choose verification" description={policies.find((item) => item.value === policy)?.detail} allowsEmpty={false} options={policies.map((item) => ({ id: item.value, label: item.label }))} onChange={(value) => { if (value) setPolicy(value); }}/></div>
  </>;

  return <section className="forms-task-card forms-composer" aria-labelledby="distribution-composer-title">
    <div className="forms-task-heading"><div><span>Form request</span><h2 id="distribution-composer-title">{scopedDelivery ? "Request form" : "Create form distribution"}</h2><p>{scopedDelivery ? "Choose the form, confirm the deadline and add the vendor contact." : "Choose the approved form, who must respond, the deadline and how recipients verify access."}</p></div>{onCancel && <button type="button" onClick={onCancel}>Close</button>}</div>
    {error && <div className="forms-message error" role="alert">{error}</div>}
    <div className="forms-task-grid">
      <SelectField label="Form" value={templateKey || undefined} placeholder="Select form" description={selectedTemplate ? `Version ${selectedTemplate.version}` : undefined} options={templates.map((item) => ({ id: `${item.id}:${item.version}`, label: item.name }))} onChange={selectTemplate}/>
      {!scopedDelivery && <><TextField label="Subject type" value={subjectType} maxLength={80} onChange={setSubjectType}/>
      <TextField label="Subject identifier" value={subjectID} maxLength={160} onChange={setSubjectID}/>
      <div className="forms-task-span"><TextField label="Title" value={title} maxLength={240} onChange={setTitle}/></div>
      <div className="forms-task-span"><TextArea label="Purpose" value={purpose} maxLength={1600} rows={3} onChange={setPurpose}/></div></>}
      <TextField label="Response deadline" type="datetime-local" value={deadline} onChange={setDeadline} description={`${timezone} timezone`}/>
      {!scopedDelivery && deliverySettings}
    </div>

    {scopedDelivery && <details className="forms-composer__advanced"><summary>More options</summary><div className="forms-task-grid"><div className="forms-task-span"><TextField label="Request title" value={title} maxLength={240} onChange={setTitle}/></div><div className="forms-task-span"><TextArea label="Message to vendor" value={purpose} maxLength={1600} rows={3} onChange={setPurpose}/></div>{deliverySettings}</div></details>}

    {scopedDelivery ? scopedDelivery.recipients : <div className="forms-recipient-panel">
      <div><h3>Recipients</h3><p>Add at least one To recipient to complete the form. CC recipients receive the communication without a response task.</p></div>
      <div className="forms-task-grid">
        <label><span>Find internal recipient</span><input type="search" value={internalQuery} placeholder="Name or identifier" onChange={(event) => setInternalQuery(event.target.value)}/>{candidates.length > 0 && <div className="forms-candidate-list" role="listbox" aria-label="Internal recipient candidates">{candidates.map((candidate) => <button type="button" role="option" key={candidate.principal_id} onClick={() => addInternal(candidate)}><strong>{candidate.display_name}</strong><span>{candidate.context_label || candidate.principal_id}</span></button>)}</div>}</label>
        <div><label><span>External email</span><input type="email" value={externalAddress} onChange={(event) => setExternalAddress(event.target.value)}/></label><label><span>Contact label</span><input value={externalLabel} maxLength={160} onChange={(event) => setExternalLabel(event.target.value)}/></label><button type="button" disabled={!externalAddress.trim() || recipients.length >= 500} onClick={addExternal}>Add external To</button></div>
      </div>
      <ul className="forms-recipient-list">{recipients.map((recipient, index) => <li key={`${recipient.type}:${recipient.principal_id || recipient.address}:${recipient.role}:${index}`}><div><strong>{recipient.contact_label || recipient.principal_id || maskAddress(recipient.address)}</strong><span>{recipient.role} · {recipient.type === "INTERNAL_PRINCIPAL" ? "Internal" : "External protected"}</span></div><SelectField label={`Role for recipient ${index + 1}`} isLabelHidden value={recipient.role} placeholder="Choose role" allowsEmpty={false} options={[{ id: "TO", label: "To" }, { id: "CC", label: "CC" }]} onChange={(role) => { if (role) setRecipients((current) => current.map((value, i) => i === index ? { ...value, role } : value)); }}/><button type="button" aria-label={`Remove recipient ${index + 1}`} onClick={() => setRecipients((current) => current.filter((_, i) => i !== index))}>Remove</button></li>)}</ul>
      {recipients.length === 0 && <p className="forms-muted">No recipients selected.</p>}
    </div>}

    {!scopedDelivery && <div className="forms-readonly-scope"><span>Owner</span><strong>Current signed-in sender</strong><span>Timezone</span><strong>{timezone}</strong></div>}
    <div className="forms-task-actions"><button className="forms-primary" type="button" disabled={!ready || busy} onClick={() => void submit()}>{busy ? scopedDelivery ? "Preparing review…" : "Creating…" : scopedDelivery?.submitLabel ?? "Create and dispatch"}</button>{!ready && <small>{scopedDelivery ? "Complete the highlighted fields." : "Add an active revision, scoped subject, valid dates, purpose and at least one To recipient."}</small>}</div>
  </section>;
}

function validDates(deadline: string, expiry: string) {
  const deadlineTime = new Date(deadline).getTime();
  const expiryTime = new Date(expiry).getTime();
  return Number.isFinite(deadlineTime) && Number.isFinite(expiryTime) && deadlineTime > Date.now() && expiryTime > Date.now() && expiryTime <= deadlineTime;
}
function maskAddress(value?: string) {
  if (!value) return "External recipient";
  const [local, domain] = value.split("@");
  return domain ? `${(local ?? "").slice(0, 1)}***@${domain}` : "External recipient";
}

function futureLocalDateTime(days: number) {
  const value = new Date(Date.now() + days * 24 * 60 * 60 * 1000);
  value.setMinutes(value.getMinutes() - value.getTimezoneOffset());
  return value.toISOString().slice(0, 16);
}
