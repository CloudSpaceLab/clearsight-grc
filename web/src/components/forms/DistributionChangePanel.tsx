import { useEffect, useMemo, useState } from "react";
import { loadReusableFormTemplateRefs } from "../../formsApi";
import {
  amendDistribution,
  loadRecipientCandidates,
  previewDistributionSupersession,
  supersedeDistribution,
  type CreateDistributionRecipient,
  type DistributionDetail,
  type DistributionSupersessionPreview,
  type RecipientCandidate,
} from "../../formsDistributionApi";
import "./distribution-change.css";

type Props = {
  detail: DistributionDetail;
  mode: "amend" | "supersede";
  onCancel: () => void;
  onSaved: (value: DistributionDetail, notice: string) => void;
};

export function DistributionChangePanel({ detail, mode, onCancel, onSaved }: Props) {
  if (mode === "amend") return <AmendPanel detail={detail} onCancel={onCancel} onSaved={onSaved}/>;
  return <SupersedePanel detail={detail} onCancel={onCancel} onSaved={onSaved}/>;
}

function AmendPanel({ detail, onCancel, onSaved }: Omit<Props, "mode">) {
  const [deadline, setDeadline] = useState(() => localDateTime(detail.distribution.deadline));
  const [expiry, setExpiry] = useState(() => localDateTime(detail.distribution.route_expires_at));
  const [revoked, setRevoked] = useState<string[]>([]);
  const [recipients, setRecipients] = useState<CreateDistributionRecipient[]>([]);
  const [internalQuery, setInternalQuery] = useState("");
  const [candidates, setCandidates] = useState<RecipientCandidate[]>([]);
  const [externalAddress, setExternalAddress] = useState("");
  const [externalLabel, setExternalLabel] = useState("");
  const [recipientRole, setRecipientRole] = useState<"TO" | "CC">("TO");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const valid = validDates(deadline, expiry);

  useEffect(() => {
    if (internalQuery.trim().length < 2) { setCandidates([]); return; }
    let active = true;
    const timer = window.setTimeout(() => {
      void loadRecipientCandidates(internalQuery, 12).then((page) => active && setCandidates(page.items)).catch(() => active && setCandidates([]));
    }, 200);
    return () => { active = false; window.clearTimeout(timer); };
  }, [internalQuery]);

  function addInternal(candidate: RecipientCandidate) {
    if (recipients.some((value) => value.type === "INTERNAL_PRINCIPAL" && value.principal_id === candidate.principal_id && value.role === recipientRole)) return;
    setRecipients((current) => [...current, { role: recipientRole, type: "INTERNAL_PRINCIPAL", principal_id: candidate.principal_id, contact_label: candidate.display_name }]);
    setInternalQuery(""); setCandidates([]);
  }

  function addExternal() {
    const address = externalAddress.trim().toLowerCase();
    if (!emailPattern.test(address)) { setError("Enter a valid external email address."); return; }
    if (recipients.some((value) => value.type === "EXTERNAL_AUDIENCE" && value.address === address && value.role === recipientRole)) return;
    setRecipients((current) => [...current, { role: recipientRole, type: "EXTERNAL_AUDIENCE", address, contact_label: externalLabel.trim() || undefined }]);
    setExternalAddress(""); setExternalLabel(""); setError("");
  }

  async function save(event: React.FormEvent) {
    event.preventDefault();
    if (!valid || busy) return;
    setBusy(true); setError("");
    try {
      const result = await amendDistribution(detail.distribution.id, {
        expected_version: detail.distribution.version,
        deadline: new Date(deadline).toISOString(),
        route_expires_at: new Date(expiry).toISOString(),
        add_recipients: recipients,
        revoke_recipient_ids: revoked,
      });
      const changes = [result.impact.deadline_changed && "deadline", result.impact.route_expiry_changed && "access expiry", result.impact.recipients_added && `${result.impact.recipients_added} recipient${result.impact.recipients_added === 1 ? "" : "s"} added`, result.impact.recipients_revoked && `${result.impact.recipients_revoked} recipient${result.impact.recipients_revoked === 1 ? "" : "s"} revoked`].filter(Boolean).join(", ");
      onSaved(result.detail, changes ? `Updated ${changes}.` : "No distribution changes were needed.");
    } catch (cause) { setError(message(cause, "The distribution could not be amended. Refresh it and try again.")); }
    finally { setBusy(false); }
  }

  return <section className="forms-task-card" aria-labelledby="amend-distribution-title">
    <div className="forms-task-heading"><div><span>Update sent form</span><h2 id="amend-distribution-title">Amend distribution</h2><p>Change the deadline or access expiry, add recipients, or revoke access that is no longer required.</p></div><button type="button" onClick={onCancel}>Close</button></div>
    {error && <div className="forms-message error" role="alert">{error}</div>}
    <form onSubmit={(event) => void save(event)}>
      <div className="forms-task-grid">
        <label><span>Response deadline</span><input type="datetime-local" value={deadline} onChange={(event) => setDeadline(event.target.value)} required/></label>
        <label><span>Access expiry</span><input type="datetime-local" value={expiry} onChange={(event) => setExpiry(event.target.value)} required/><small>Access must expire no later than the response deadline.</small></label>
      </div>
      <fieldset className="forms-change-recipients"><legend>Add recipients</legend><p>New To recipients receive a response task. CC recipients receive the communication without a response task.</p>
        <div className="forms-task-grid">
          <label><span>Recipient role</span><select value={recipientRole} onChange={(event) => setRecipientRole(event.target.value as "TO" | "CC")}><option value="TO">To</option><option value="CC">CC</option></select></label>
          <label><span>Find internal recipient</span><input type="search" value={internalQuery} placeholder="Name or identifier" onChange={(event) => setInternalQuery(event.target.value)}/>{candidates.length > 0 && <div className="forms-candidate-list" role="listbox" aria-label="Internal recipient candidates">{candidates.map((candidate) => <button type="button" role="option" key={candidate.principal_id} onClick={() => addInternal(candidate)}><strong>{candidate.display_name}</strong><span>{candidate.context_label || candidate.principal_id}</span></button>)}</div>}</label>
          <label><span>External email</span><input type="email" value={externalAddress} onChange={(event) => setExternalAddress(event.target.value)}/></label>
          <label><span>Contact label</span><input value={externalLabel} maxLength={160} onChange={(event) => setExternalLabel(event.target.value)}/></label>
        </div>
        <button type="button" disabled={!externalAddress.trim()} onClick={addExternal}>Add external {recipientRole === "TO" ? "To" : "CC"}</button>
        <ul className="forms-recipient-list">{recipients.map((recipient, index) => <li key={`${recipient.type}:${recipient.principal_id || recipient.address}:${recipient.role}:${index}`}><div><strong>{recipient.contact_label || recipient.principal_id || maskAddress(recipient.address)}</strong><span>{recipient.role} · {recipient.type === "INTERNAL_PRINCIPAL" ? "Internal" : "External protected"}</span></div><button type="button" aria-label={`Remove new recipient ${index + 1}`} onClick={() => setRecipients((current) => current.filter((_, itemIndex) => itemIndex !== index))}>Remove</button></li>)}</ul>
      </fieldset>
      <fieldset className="forms-change-recipients"><legend>Current recipients</legend><p>Revoking a recipient ends their access. Their earlier submitted response remains in the audit history.</p>
        <ul className="forms-recipient-list">{detail.recipients.filter((recipient) => recipient.state !== "REVOKED").map((recipient) => <li key={recipient.id}><div><strong>{recipient.contact_label || recipient.principal_id || recipient.audience_hint || "Protected recipient"}</strong><span>{recipient.role} · {recipient.type === "INTERNAL_PRINCIPAL" ? "Internal" : "External"} · {label(recipient.state)}</span></div><label><input type="checkbox" checked={revoked.includes(recipient.id)} onChange={(event) => setRevoked((current) => event.target.checked ? [...current, recipient.id] : current.filter((id) => id !== recipient.id))}/> Revoke</label></li>)}</ul>
      </fieldset>
      <div className="forms-task-actions"><button className="forms-primary" type="submit" disabled={!valid || busy}>{busy ? "Saving…" : "Save amendment"}</button><button type="button" onClick={onCancel} disabled={busy}>Cancel</button>{!valid && <small>Set future dates with access expiry no later than the response deadline.</small>}</div>
    </form>
  </section>;
}

function SupersedePanel({ detail, onCancel, onSaved }: Omit<Props, "mode">) {
  const [versions, setVersions] = useState<Array<{ version: number; name: string }>>([]);
  const [targetVersion, setTargetVersion] = useState(0);
  const [preview, setPreview] = useState<DistributionSupersessionPreview>();
  const [carryForward, setCarryForward] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    void loadReusableFormTemplateRefs().then((items) => {
      if (!active) return;
      const eligible = items.filter((item) => item.id === detail.distribution.form_template_id && item.version !== detail.distribution.form_template_version).map((item) => ({ version: item.version, name: item.name }));
      setVersions(eligible); setTargetVersion(eligible[0]?.version ?? 0);
    }).catch((cause) => active && setError(message(cause, "Replacement form versions could not be loaded.")));
    return () => { active = false; };
  }, [detail.distribution.form_template_id, detail.distribution.form_template_version]);

  async function loadPreview() {
    if (!targetVersion || busy) return;
    setBusy(true); setError(""); setPreview(undefined); setCarryForward(false);
    try { setPreview(await previewDistributionSupersession(detail.distribution.id, detail.distribution.version, targetVersion)); }
    catch (cause) { setError(message(cause, "The replacement impact could not be prepared. Refresh the distribution and try again.")); }
    finally { setBusy(false); }
  }

  async function confirm() {
    if (!preview || busy) return;
    setBusy(true); setError("");
    try {
      const result = await supersedeDistribution(detail.distribution.id, {
        expected_version: preview.expected_version,
        expected_workspace_version: preview.expected_workspace_version,
        target_form_version: preview.target_form_version,
        carry_forward: carryForward,
        confirmed_field_ids: carryForward ? preview.compatible_fields.map((field) => field.field_id) : [],
      });
      onSaved(result.replacement, `Form version replaced with v${result.replacement.distribution.form_template_version}. ${result.carried_field_ids.length} compatible response${result.carried_field_ids.length === 1 ? " was" : "s were"} carried forward.`);
    } catch (cause) { setError(message(cause, "The form version could not be replaced. Preview the latest changes and try again.")); }
    finally { setBusy(false); }
  }

  const compatibleCount = preview?.compatible_fields.length ?? 0;
  return <section className="forms-task-card" aria-labelledby="supersede-distribution-title">
    <div className="forms-task-heading"><div><span>Replace sent form</span><h2 id="supersede-distribution-title">Replace form version</h2><p>Preview which submitted answers remain compatible before moving recipients to the approved replacement.</p></div><button type="button" onClick={onCancel}>Close</button></div>
    {error && <div className="forms-message error" role="alert">{error}</div>}
    <div className="forms-task-grid"><label><span>Approved replacement</span><select value={targetVersion || ""} onChange={(event) => { setTargetVersion(Number(event.target.value)); setPreview(undefined); }}><option value="">Select a different active version</option>{versions.map((item) => <option key={item.version} value={item.version}>{item.name} · v{item.version}</option>)}</select></label></div>
    {versions.length === 0 && !error && <div className="forms-task-empty"><strong>No approved replacement is available</strong><span>Activate a newer version of this form before replacing the sent form.</span></div>}
    <div className="forms-task-actions"><button type="button" onClick={() => void loadPreview()} disabled={!targetVersion || busy}>{busy && !preview ? "Checking…" : "Preview replacement"}</button></div>
    {preview && <section className="forms-change-preview" aria-label="Replacement impact"><h3>{compatibleCount} response{compatibleCount === 1 ? "" : "s"} can carry forward</h3><p>{preview.excluded_fields.length} response{preview.excluded_fields.length === 1 ? " needs" : "s need"} a new answer because the field changed or is no longer present.</p>{preview.excluded_fields.length > 0 && <ul>{preview.excluded_fields.map((field) => <li key={field.field_id}><strong>{field.field_id}</strong>{field.reason ? ` · ${field.reason}` : ""}</li>)}</ul>}<label className="forms-check-row"><input type="checkbox" checked={carryForward} disabled={compatibleCount === 0} onChange={(event) => setCarryForward(event.target.checked)}/> Carry compatible responses into the replacement</label><div className="forms-task-actions"><button className="forms-primary" type="button" onClick={() => void confirm()} disabled={busy}>{busy ? "Replacing…" : "Confirm replacement"}</button><button type="button" onClick={onCancel} disabled={busy}>Cancel</button></div></section>}
  </section>;
}

function localDateTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 16);
}
function validDates(deadline: string, expiry: string) {
  const deadlineTime = new Date(deadline).getTime(); const expiryTime = new Date(expiry).getTime();
  return Number.isFinite(deadlineTime) && Number.isFinite(expiryTime) && deadlineTime > Date.now() && expiryTime > Date.now() && expiryTime <= deadlineTime;
}
function label(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (part) => part.toUpperCase()); }
function message(cause: unknown, fallback: string) { return cause instanceof Error ? cause.message : fallback; }
const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
function maskAddress(value?: string) { const [local, domain] = value?.split("@") ?? []; return domain ? `${(local ?? "").slice(0, 1)}***@${domain}` : "External recipient"; }
