import { useEffect, useMemo, useState } from "react";
import { loadReusableFormTemplateRefs } from "../../formsApi";
import type { ReusableFormTemplateRef } from "../../formsTypes";
import {
  createDistribution,
  loadRecipientCandidates,
  type CreateDistributionRecipient,
  type DistributionAccessPolicy,
  type DistributionDetail,
  type RecipientCandidate,
} from "../../formsDistributionApi";
import { SelectField } from "../ui";

const policies: Array<{ value: DistributionAccessPolicy; label: string; detail: string }> = [
  { value: "DIRECT_MAGIC_LINK", label: "Direct magic link", detail: "Possession of the recipient-specific link grants access." },
  { value: "SHARED_LINK_EMAIL_OTP", label: "Shared link + email OTP", detail: "A shared route requires recipient selection and email verification." },
  { value: "DIRECT_LINK_EMAIL_OTP", label: "Direct link + email OTP", detail: "A recipient-specific route also requires email verification." },
];

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

type Props = { onCreated?: (value: DistributionDetail) => void; onCancel?: () => void };

export function DistributionComposer({ onCreated, onCancel }: Props) {
  const [templates, setTemplates] = useState<ReusableFormTemplateRef[]>([]);
  const [templateKey, setTemplateKey] = useState("");
  const [subjectType, setSubjectType] = useState("CONTROL");
  const [subjectID, setSubjectID] = useState("");
  const [title, setTitle] = useState("");
  const [purpose, setPurpose] = useState("");
  const [policy, setPolicy] = useState<DistributionAccessPolicy>("DIRECT_LINK_EMAIL_OTP");
  const [estimatedMinutes, setEstimatedMinutes] = useState(15);
  const [deadline, setDeadline] = useState("");
  const [routeExpiry, setRouteExpiry] = useState("");
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
      if (values[0]) setTemplateKey(`${values[0].id}:${values[0].version}`);
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
  const ready = Boolean(selectedTemplate && subjectType.trim() && subjectID.trim() && title.trim() && purpose.trim() && recipients.some((value) => value.role === "TO") && validDates(deadline, routeExpiry) && estimatedMinutes >= 1 && estimatedMinutes <= 60);

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

  return <section className="forms-task-card forms-composer" aria-labelledby="distribution-composer-title">
    <div className="forms-task-heading"><div><span>Send form</span><h2 id="distribution-composer-title">Create form distribution</h2><p>Choose the approved form, who must respond, the deadline and how recipients verify access.</p></div>{onCancel && <button type="button" onClick={onCancel}>Close</button>}</div>
    {error && <div className="forms-message error" role="alert">{error}</div>}
    <div className="forms-task-grid">
      <SelectField label="Active form revision" value={templateKey || undefined} placeholder="Select active revision" description="The selected form version cannot change after sending." options={templates.map((item) => ({ id: `${item.id}:${item.version}`, label: `${item.name} · ${item.code} · v${item.version}` }))} onChange={(value) => setTemplateKey(value ?? "")}/>
      <label><span>Subject type</span><input value={subjectType} maxLength={80} onChange={(event) => setSubjectType(event.target.value)}/></label>
      <label><span>Subject identifier</span><input value={subjectID} maxLength={160} onChange={(event) => setSubjectID(event.target.value)}/></label>
      <label><span>Estimated minutes</span><input type="number" min={1} max={60} value={estimatedMinutes} onChange={(event) => setEstimatedMinutes(Number(event.target.value))}/></label>
      <label className="forms-task-span"><span>Title</span><input value={title} maxLength={240} onChange={(event) => setTitle(event.target.value)}/></label>
      <label className="forms-task-span"><span>Purpose</span><textarea value={purpose} maxLength={1600} rows={3} onChange={(event) => setPurpose(event.target.value)}/></label>
      <label><span>Deadline</span><input type="datetime-local" value={deadline} onChange={(event) => setDeadline(event.target.value)}/><small>The saved deadline includes the {timezone} timezone.</small></label>
      <label><span>Access route expiry</span><input type="datetime-local" value={routeExpiry} onChange={(event) => setRouteExpiry(event.target.value)}/><small>Must be no later than the deadline.</small></label>
      <div className="forms-task-span"><SelectField label="Access policy" value={policy} placeholder="Choose access policy" description={policies.find((item) => item.value === policy)?.detail} allowsEmpty={false} options={policies.map((item) => ({ id: item.value, label: item.label }))} onChange={(value) => { if (value) setPolicy(value); }}/></div>
    </div>

    <div className="forms-recipient-panel">
      <div><h3>Recipients</h3><p>Add at least one To recipient to complete the form. CC recipients receive the communication without a response task.</p></div>
      <div className="forms-task-grid">
        <label><span>Find internal recipient</span><input type="search" value={internalQuery} placeholder="Name or identifier" onChange={(event) => setInternalQuery(event.target.value)}/>{candidates.length > 0 && <div className="forms-candidate-list" role="listbox" aria-label="Internal recipient candidates">{candidates.map((candidate) => <button type="button" role="option" key={candidate.principal_id} onClick={() => addInternal(candidate)}><strong>{candidate.display_name}</strong><span>{candidate.context_label || candidate.principal_id}</span></button>)}</div>}</label>
        <div><label><span>External email</span><input type="email" value={externalAddress} onChange={(event) => setExternalAddress(event.target.value)}/></label><label><span>Contact label</span><input value={externalLabel} maxLength={160} onChange={(event) => setExternalLabel(event.target.value)}/></label><button type="button" disabled={!externalAddress.trim() || recipients.length >= 500} onClick={addExternal}>Add external To</button></div>
      </div>
      <ul className="forms-recipient-list">{recipients.map((recipient, index) => <li key={`${recipient.type}:${recipient.principal_id || recipient.address}:${recipient.role}:${index}`}><div><strong>{recipient.contact_label || recipient.principal_id || maskAddress(recipient.address)}</strong><span>{recipient.role} · {recipient.type === "INTERNAL_PRINCIPAL" ? "Internal" : "External protected"}</span></div><SelectField label={`Role for recipient ${index + 1}`} isLabelHidden value={recipient.role} placeholder="Choose role" allowsEmpty={false} options={[{ id: "TO", label: "To" }, { id: "CC", label: "CC" }]} onChange={(role) => { if (role) setRecipients((current) => current.map((value, i) => i === index ? { ...value, role } : value)); }}/><button type="button" aria-label={`Remove recipient ${index + 1}`} onClick={() => setRecipients((current) => current.filter((_, i) => i !== index))}>Remove</button></li>)}</ul>
      {recipients.length === 0 && <p className="forms-muted">No recipients selected.</p>}
    </div>

    <div className="forms-readonly-scope"><span>Owner</span><strong>Current signed-in sender</strong><span>Timezone</span><strong>{timezone}</strong></div>
    <div className="forms-task-actions"><button className="forms-primary" type="button" disabled={!ready || busy} onClick={() => void submit()}>{busy ? "Creating…" : "Create and dispatch"}</button>{!ready && <small>Add an active revision, scoped subject, valid dates, purpose and at least one To recipient.</small>}</div>
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
