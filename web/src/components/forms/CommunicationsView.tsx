import { useEffect, useMemo, useState } from "react";
import {
  createCommunicationProfile, createCommunicationTemplate, impactCommunicationTemplate, loadCommunicationProfiles, loadCommunicationTemplates,
  previewCommunicationTemplate, rollbackCommunicationProfile, rollbackCommunicationTemplate, testSendCommunicationTemplate,
  transitionCommunicationProfile, transitionCommunicationTemplate,
  type CommunicationImpact, type CommunicationPreview, type CommunicationProfile, type CommunicationStatus, type CommunicationTemplate,
} from "../../formsCommunicationApi";
import { CommunicationTemplateEditor, type CommunicationTemplateDraft } from "./CommunicationTemplateEditor";

export default function CommunicationsView() {
  const [profiles, setProfiles] = useState<CommunicationProfile[]>([]);
  const [templates, setTemplates] = useState<CommunicationTemplate[]>([]);
  const [selected, setSelected] = useState<CommunicationTemplate>();
  const [editorOpen, setEditorOpen] = useState(false);
  const [profileOpen, setProfileOpen] = useState(false);
  const [preview, setPreview] = useState<CommunicationPreview>();
  const [impact, setImpact] = useState<CommunicationImpact>();
  const [testAddress, setTestAddress] = useState("");
  const [busy, setBusy] = useState<string>();
  const [error, setError] = useState<string>();
  const [notice, setNotice] = useState<string>();

  useEffect(() => { void refresh(); }, []);
  async function refresh() {
    setError(undefined);
    try {
      const [profileValues, templateValues] = await Promise.all([loadCommunicationProfiles(), loadCommunicationTemplates()]);
      setProfiles(profileValues); setTemplates(templateValues);
      setSelected((current) => templateValues.find((value) => current && sameTemplate(value, current)) ?? templateValues[0]);
    } catch (cause) { setError(message(cause, "Communication configuration could not be loaded.")); }
  }
  async function saveTemplate(draft: CommunicationTemplateDraft) {
    if (busy) return;
    setBusy("save-template"); setError(undefined);
    try {
      const value = await createCommunicationTemplate(draft);
      setNotice(`Draft ${value.action} · ${value.locale} · v${value.version} created.`); setEditorOpen(false); await refresh(); setSelected(value);
    } catch (cause) { setError(message(cause, "Communication template could not be saved.")); } finally { setBusy(undefined); }
  }
  async function templateTransition(to: CommunicationStatus) {
    if (!selected || busy) return;
    setBusy("template-transition"); setError(undefined);
    try { const value = await transitionCommunicationTemplate(selected, to); setSelected(value); setNotice(`Template moved to ${label(to)}.`); await refresh(); }
    catch (cause) { setError(message(cause, "Template lifecycle state could not be changed.")); } finally { setBusy(undefined); }
  }
  async function templateRollback() {
    if (!selected || busy) return;
    setBusy("template-rollback");
    try { const value = await rollbackCommunicationTemplate(selected); setNotice(`Rollback created draft v${value.version}.`); await refresh(); setSelected(value); }
    catch (cause) { setError(message(cause, "Template rollback could not be created.")); } finally { setBusy(undefined); }
  }
  async function serverPreview() {
    if (!selected || busy) return;
    setBusy("preview"); setError(undefined);
    try { setPreview(await previewCommunicationTemplate(selected)); } catch (cause) { setError(message(cause, "Communication preview could not be rendered.")); } finally { setBusy(undefined); }
  }
  async function serverImpact() {
    if (!selected || busy) return;
    setBusy("impact"); setError(undefined);
    try { setImpact(await impactCommunicationTemplate(selected)); } catch (cause) { setError(message(cause, "Server impact could not be calculated.")); } finally { setBusy(undefined); }
  }
  async function testSend() {
    if (!selected || !testAddress.trim() || busy) return;
    setBusy("test-send"); setError(undefined);
    try { await testSendCommunicationTemplate(selected, testAddress.trim()); setNotice("Test email queued for delivery."); }
    catch (cause) { setError(message(cause, "Test send failed.")); } finally { setBusy(undefined); }
  }

  const profile = useMemo(() => profiles.find((value) => value.status === "ACTIVE") ?? profiles[0], [profiles]);
  if (editorOpen) return <CommunicationTemplateEditor initial={selected} busy={busy === "save-template"} onSave={saveTemplate}/>;
  if (profileOpen) return <ProfileEditor initial={profile} busy={Boolean(busy)} onCancel={() => setProfileOpen(false)} onSaved={async () => { setProfileOpen(false); await refresh(); }}/>

  return <section className="forms-task-shell" aria-labelledby="communications-title">
    <div className="forms-task-heading"><div><span>Message delivery</span><h2 id="communications-title">Communications</h2><p>Manage the sender identity and message templates used for form invitations and reminders.</p></div><div className="forms-inline-actions"><button type="button" onClick={() => setProfileOpen(true)}>{profile ? "New profile revision" : "Create profile"}</button><button className="forms-primary" type="button" onClick={() => setEditorOpen(true)}>{selected ? "New template revision" : "Create template"}</button></div></div>
    {error && <div className="forms-message error" role="alert">{error}</div>}{notice && <div className="forms-message" role="status">{notice}</div>}
    <div className="forms-communication-summary"><div><span>Active profile</span><strong>{profile?.bank_name || "Not configured"}</strong><small>{profile ? `${profile.default_locale} · ${label(profile.status)} · v${profile.version}` : "Create a legal-entity communication profile."}</small></div><div><span>Template revisions</span><strong>{templates.length}</strong><small>{templates.filter((value) => value.status === "ACTIVE").length} active</small></div></div>
    <div className="forms-task-layout">
      <div className="forms-communication-list"><h3>Templates</h3><ul>{templates.map((value) => <li key={`${value.action}:${value.locale}:${value.version}`}><button type="button" className={selected && sameTemplate(value, selected) ? "selected" : ""} onClick={() => { setSelected(value); setPreview(undefined); setImpact(undefined); }}><strong>{label(value.action)} · {value.locale}</strong><span>{label(value.status)} · v{value.version}</span><small>{value.subject_template}</small></button></li>)}</ul>{templates.length === 0 && <div className="forms-task-empty"><strong>No communication templates</strong><span>Create the first governed draft revision.</span></div>}</div>
      <main className="forms-communication-detail">{selected ? <><span className="forms-detail-kicker">{selected.action} · {selected.locale}</span><h3>{selected.subject_template}</h3><p>v{selected.version} · {label(selected.status)} · effective {formatDate(selected.effective_from)}</p><div className="forms-inline-actions"><button type="button" disabled={Boolean(busy)} onClick={() => void serverPreview()}>{busy === "preview" ? "Rendering…" : "Preview"}</button><button type="button" disabled={Boolean(busy)} onClick={() => void serverImpact()}>{busy === "impact" ? "Checking…" : "Impact"}</button><button type="button" onClick={() => setEditorOpen(true)}>Edit as new revision</button></div>{preview && <section className="forms-server-preview" aria-label="Communication preview"><span>Preview</span><h4>{preview.subject}</h4><p>{preview.plain_text}</p></section>}{impact && <dl className="forms-impact"><div><dt>Subject changed</dt><dd>{yesNo(impact.subject_changed)}</dd></div><div><dt>Document changed</dt><dd>{yesNo(impact.document_changed)}</dd></div><div><dt>Window changed</dt><dd>{yesNo(impact.effective_window_changed)}</dd></div></dl>}<div className="forms-detail-actions">{selected.status === "DRAFT" && <button type="button" disabled={Boolean(busy)} onClick={() => void templateTransition("PENDING_APPROVAL")}>Send for approval</button>}{selected.status === "PENDING_APPROVAL" && <button className="forms-primary" type="button" disabled={Boolean(busy)} onClick={() => void templateTransition("ACTIVE")}>Activate</button>}{selected.status === "ACTIVE" && <button type="button" disabled={Boolean(busy)} onClick={() => void templateTransition("RETIRED")}>Retire</button>}<button type="button" disabled={Boolean(busy)} onClick={() => void templateRollback()}>Create rollback draft</button></div><div className="forms-test-send"><label><span>Test recipient</span><input type="email" value={testAddress} onChange={(event) => setTestAddress(event.target.value)}/></label><button type="button" disabled={Boolean(busy) || !testAddress.trim()} onClick={() => void testSend()}>{busy === "test-send" ? "Sending…" : "Test send"}</button></div></> : <div className="forms-task-empty"><strong>Select or create a template</strong><span>Choose a message template or create one for invitations and reminders.</span></div>}</main>
      <aside className="forms-task-detail"><span className="forms-detail-kicker">Profile</span>{profile ? <><h3>{profile.bank_name}</h3><dl><div><dt>Status</dt><dd>{label(profile.status)}</dd></div><div><dt>Locale</dt><dd>{profile.default_locale}</dd></div><div><dt>Support</dt><dd>{profile.support_contact || "Not set"}</dd></div><div><dt>Version</dt><dd>{profile.version}</dd></div></dl><ProfileLifecycle profile={profile} busy={Boolean(busy)} onTransition={async (to) => { setBusy("profile-transition"); try { await transitionCommunicationProfile(profile.version, to); await refresh(); } catch (cause) { setError(message(cause, "Profile lifecycle state could not be changed.")); } finally { setBusy(undefined); } }} onRollback={async () => { setBusy("profile-rollback"); try { await rollbackCommunicationProfile(profile.version); await refresh(); } catch (cause) { setError(message(cause, "Profile rollback could not be created.")); } finally { setBusy(undefined); } }}/></> : <><h3>No communication profile</h3><p>Create a profile before activating message revisions.</p></>}</aside>
    </div>
  </section>;
}

function ProfileEditor({ initial, busy, onCancel, onSaved }: { initial?: CommunicationProfile; busy: boolean; onCancel: () => void; onSaved: () => Promise<void> }) {
  const [bank, setBank] = useState(initial?.bank_name ?? ""); const [locale, setLocale] = useState(initial?.default_locale ?? "en"); const [support, setSupport] = useState(initial?.support_contact ?? ""); const [brand, setBrand] = useState(initial?.brand_asset_id ?? ""); const [from, setFrom] = useState(toLocal(initial?.effective_from) || toLocal(new Date().toISOString())); const [until, setUntil] = useState(toLocal(initial?.effective_until)); const [error, setError] = useState<string>();
  async function save() { try { await createCommunicationProfile({ bank_name: bank.trim(), default_locale: locale.trim(), support_contact: support.trim() || undefined, brand_asset_id: brand.trim() || undefined, effective_from: new Date(from).toISOString(), effective_until: until ? new Date(until).toISOString() : undefined }); await onSaved(); } catch (cause) { setError(message(cause, "Communication profile could not be saved.")); } }
  return <section className="forms-task-card"><div className="forms-task-heading"><div><span>New governed revision</span><h2>Communication profile</h2><p>Editing creates another immutable profile revision.</p></div><button type="button" onClick={onCancel}>Close</button></div>{error && <div className="forms-message error" role="alert">{error}</div>}<div className="forms-task-grid"><label><span>Bank or organization name</span><input value={bank} onChange={(event) => setBank(event.target.value)}/></label><label><span>Default locale</span><input value={locale} onChange={(event) => setLocale(event.target.value)}/></label><label><span>Support contact</span><input value={support} onChange={(event) => setSupport(event.target.value)}/></label><label><span>Inspected brand asset ID</span><input value={brand} onChange={(event) => setBrand(event.target.value)}/></label><label><span>Effective from</span><input type="datetime-local" value={from} onChange={(event) => setFrom(event.target.value)}/></label><label><span>Effective until</span><input type="datetime-local" value={until} onChange={(event) => setUntil(event.target.value)}/></label></div><div className="forms-task-actions"><button className="forms-primary" type="button" disabled={busy || !bank.trim() || !locale.trim() || !from} onClick={() => void save()}>Save profile revision</button></div></section>;
}
function ProfileLifecycle({ profile, busy, onTransition, onRollback }: { profile: CommunicationProfile; busy: boolean; onTransition: (to: CommunicationStatus) => Promise<void>; onRollback: () => Promise<void> }) { return <div className="forms-detail-actions">{profile.status === "DRAFT" && <button type="button" disabled={busy} onClick={() => void onTransition("PENDING_APPROVAL")}>Send for approval</button>}{profile.status === "PENDING_APPROVAL" && <button type="button" disabled={busy} onClick={() => void onTransition("ACTIVE")}>Activate</button>}{profile.status === "ACTIVE" && <button type="button" disabled={busy} onClick={() => void onTransition("RETIRED")}>Retire</button>}<button type="button" disabled={busy} onClick={() => void onRollback()}>Rollback draft</button></div>; }
function sameTemplate(a: CommunicationTemplate, b: CommunicationTemplate) { return a.action === b.action && a.locale === b.locale && a.version === b.version; }
function label(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (part) => part.toUpperCase()); }
function formatDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "Unknown" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(date); }
function toLocal(value?: string) { if (!value) return ""; const date = new Date(value); if (Number.isNaN(date.getTime())) return ""; const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000); return local.toISOString().slice(0, 16); }
function yesNo(value: boolean) { return value ? "Yes" : "No"; }
function message(cause: unknown, fallback: string) { return cause instanceof Error ? cause.message : fallback; }
