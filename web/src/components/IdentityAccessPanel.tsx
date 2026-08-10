import { FormEvent, useEffect, useMemo, useState } from "react";
import { ApiError } from "../http";
import {
  createGroupRoleBinding,
  createIdentitySource,
  loadIdentityAccessOverview,
  previewEscalation,
  retireGroupRoleBinding,
  revokeIdentitySource,
  rotateIdentitySourceToken,
  type EscalationPreview,
  type IdentityAccessOverview,
} from "../identityAccessApi";

export function IdentityAccessPanel() {
  const [overview, setOverview] = useState<IdentityAccessOverview | null>(null);
  const [state, setState] = useState<"loading" | "live" | "restricted" | "unavailable">("loading");
  const [busy, setBusy] = useState("");
  const [notice, setNotice] = useState("");
  const [token, setToken] = useState("");
  const [sourceCode, setSourceCode] = useState("");
  const [issuer, setIssuer] = useState("");
  const [subjectAttribute, setSubjectAttribute] = useState<"externalId" | "userName">("externalId");
  const [groupID, setGroupID] = useState("");
  const [roleID, setRoleID] = useState("");
  const [department, setDepartment] = useState("");
  const [policyID, setPolicyID] = useState("");
  const [sequenceID, setSequenceID] = useState("");
  const [previewDepartment, setPreviewDepartment] = useState("");
  const [preview, setPreview] = useState<EscalationPreview | null>(null);

  async function reload() {
    setState("loading");
    try {
      const next = await loadIdentityAccessOverview();
      setOverview(next);
      setState("live");
    } catch (error) {
      setOverview(null);
      setState(error instanceof ApiError && (error.status === 401 || error.status === 403) ? "restricted" : "unavailable");
    }
  }

  useEffect(() => { void reload(); }, []);

  const activeGroups = useMemo(() => overview?.groups.filter((group) => group.source_state === "ACTIVE") ?? [], [overview]);
  const activeRoles = overview?.roles ?? [];
  const selectedPolicy = overview?.escalation_policies.find((policy) => policy.policy_id === policyID);
  const sequences = selectedPolicy?.sequences ?? [];

  useEffect(() => {
    if (!overview?.escalation_policies.length) return;
    if (!policyID) setPolicyID(overview.escalation_policies[0].policy_id);
  }, [overview, policyID]);

  useEffect(() => {
    const first = sequences[0]?.ID;
    if (first && !sequences.some((sequence) => sequence.ID === sequenceID)) setSequenceID(first);
  }, [sequences, sequenceID]);

  async function run(label: string, action: () => Promise<void>) {
    setBusy(label); setNotice("");
    try { await action(); }
    catch (error) { setNotice(error instanceof Error ? error.message : "The change could not be completed."); }
    finally { setBusy(""); }
  }

  async function createSource(event: FormEvent) {
    event.preventDefault();
    await run("source-create", async () => {
      const result = await createIdentitySource({ code: sourceCode, identity_issuer: issuer || undefined, subject_attribute: subjectAttribute });
      setToken(result.token); setSourceCode(""); setIssuer(""); setNotice("Provisioning source created. Copy the token now; it will not be shown again.");
      await reload();
    });
  }

  async function createBinding(event: FormEvent) {
    event.preventDefault();
    await run("binding-create", async () => {
      await createGroupRoleBinding({ group_id: groupID, role_template_id: roleID, department_path: parseDepartmentPath(department) });
      setNotice("Directory group role mapping added. Material decision authority is unchanged."); setDepartment("");
      await reload();
    });
  }

  async function loadPreview(event: FormEvent) {
    event.preventDefault(); setBusy("preview"); setNotice("");
    try { setPreview(await previewEscalation({ policy_id: policyID, sequence_id: sequenceID, department_path: parseDepartmentPath(previewDepartment) })); }
    catch (error) { setPreview(null); setNotice(error instanceof Error ? error.message : "Escalation preview could not be loaded."); }
    finally { setBusy(""); }
  }

  if (state === "loading") return <section className="identity-access-panel" aria-busy="true"><div className="section-header"><div><span className="eyebrow">Identity & access</span><h2>Enterprise access</h2><p>Loading sign-in, provisioning and role mappings…</p></div></div></section>;
  if (state === "restricted") return <section className="identity-access-panel"><div className="section-header"><div><span className="eyebrow">Identity & access</span><h2>Enterprise access</h2><p>Your current role does not include identity administration visibility.</p></div></div></section>;
  if (state === "unavailable" || !overview) return <section className="identity-access-panel"><div className="section-header"><div><span className="eyebrow">Identity & access</span><h2>Enterprise access unavailable</h2><p>No identity, provisioning or role-mapping state is inferred while the service is unavailable.</p></div><button className="secondary-button" type="button" onClick={() => void reload()}>Retry</button></div></section>;

  return <section className="identity-access-panel" aria-label="Identity and access configuration">
    <div className="section-header identity-access-heading"><div><span className="eyebrow">Identity & access</span><h2>Enterprise access</h2><p>Sign-in, directory provisioning, coarse role eligibility and escalation routing. Material authority remains governed separately.</p></div><span className="identity-health">{overview.sources.filter((source) => source.status === "ACTIVE").length} active source{overview.sources.filter((source) => source.status === "ACTIVE").length === 1 ? "" : "s"}</span></div>
    {notice && <div className="inline-notice" role="status">{notice}</div>}
    {token && <div className="identity-token" role="status"><div><strong>Provisioning token — shown once</strong><p>Copy this token into the upstream SCIM provider now. ClearSight stores only its digest.</p><code>{token}</code></div><div className="identity-token-actions"><button className="secondary-button" type="button" onClick={() => void navigator.clipboard?.writeText(token)}>Copy</button><button className="text-button" type="button" onClick={() => setToken("")}>Hide</button></div></div>}

    <div className="identity-access-grid">
      <article className="config-card identity-source-card"><div className="section-header"><div><h3>Sign-in & provisioning</h3><p>{overview.sign_in.mode === "oidc" ? `OIDC${overview.sign_in.issuer ? ` · ${overview.sign_in.issuer}` : ""}` : humanize(overview.sign_in.mode)} · {overview.sign_in.assurance_level || "assurance not reported"}</p></div></div>
        <div className="identity-list">{overview.sources.length ? overview.sources.map((source) => <div className="identity-row" key={source.id}><div><strong>{source.code}</strong><span>{source.active_users} users · {source.active_groups} groups · {source.subject_attribute}</span></div><mark>{humanize(source.status)}</mark>{overview.can_configure && source.status === "ACTIVE" && <div className="identity-row-actions"><button className="text-button" type="button" disabled={busy !== ""} onClick={() => void run(`rotate-${source.id}`, async () => { const next = await rotateIdentitySourceToken(source.id); setToken(next.token); setNotice("Token rotated. The previous token is no longer valid."); })}>Rotate token</button><button className="text-button danger-text" type="button" disabled={busy !== ""} onClick={() => { if (window.confirm(`Revoke ${source.code}? Source-derived access will stop on the next request.`)) void run(`revoke-${source.id}`, async () => { await revokeIdentitySource(source.id); setNotice("Provisioning source revoked. Historical principals remain recorded, but source-derived eligibility is disabled."); await reload(); }); }}>Revoke</button></div>}</div>) : <p className="muted-copy">No SCIM source has been configured.</p>}</div>
        {overview.can_configure && <form className="identity-inline-form" onSubmit={(event) => void createSource(event)}><h4>Add provisioning source</h4><label>Code<input value={sourceCode} onChange={(event) => setSourceCode(event.target.value)} required placeholder="ENTRA"/></label><label>OIDC issuer <span>(optional correlation)</span><input value={issuer} onChange={(event) => setIssuer(event.target.value)} placeholder="https://id.example.com"/></label><label>Stable subject<select value={subjectAttribute} onChange={(event) => setSubjectAttribute(event.target.value as "externalId" | "userName")}><option value="externalId">externalId</option><option value="userName">userName</option></select></label><button className="secondary-button" disabled={busy !== ""} type="submit">Create source</button></form>}
      </article>

      <article className="config-card"><div className="section-header"><div><h3>Group → role mappings</h3><p>Directory groups grant coarse capability eligibility only.</p></div></div>
        <div className="identity-list">{overview.bindings.length ? overview.bindings.map((binding) => <div className="identity-row" key={binding.id}><div><strong>{binding.group_name} → {binding.role_code}</strong><span>{binding.department_path.length ? binding.department_path.join(" / ") : "Legal entity wide"} · {binding.legal_entity}</span></div>{overview.can_configure && <button className="text-button" disabled={busy !== ""} type="button" onClick={() => { if (window.confirm("Retire this group role mapping?")) void run(`retire-${binding.id}`, async () => { await retireGroupRoleBinding(binding.id); setNotice("Group role mapping retired."); await reload(); }); }}>Retire</button>}</div>) : <p className="muted-copy">No active directory group role mappings in this legal entity.</p>}</div>
        {overview.can_configure && <form className="identity-inline-form" onSubmit={(event) => void createBinding(event)}><h4>Add mapping</h4><label>Directory group<select required value={groupID} onChange={(event) => setGroupID(event.target.value)}><option value="">Choose group</option>{activeGroups.map((group) => <option value={group.id} key={group.id}>{group.display_name} · {group.source_code}</option>)}</select></label><label>Role<select required value={roleID} onChange={(event) => setRoleID(event.target.value)}><option value="">Choose role</option>{activeRoles.map((role) => <option value={role.id} key={role.id}>{role.code} · {role.capabilities.join(", ") || "no coarse capabilities"}</option>)}</select></label><label>Department path <span>(optional)</span><input value={department} onChange={(event) => setDepartment(event.target.value)} placeholder="BANK / RISK / OPERATIONS"/></label><button className="secondary-button" disabled={busy !== ""} type="submit">Add mapping</button></form>}
      </article>

      <article className="config-card"><div className="section-header"><div><h3>People & groups</h3><p>Bounded directory inspection, not a second directory console.</p></div></div><div className="identity-directory-columns"><div><h4>People</h4>{overview.people.slice(0, 8).map((person) => <div className="identity-mini-row" key={person.id}><strong>{person.display_name}</strong><span>{person.source_code ? `${person.source_code} · ${person.source_state}` : "Local principal"}</span></div>)}{!overview.people.length && <p className="muted-copy">No people found.</p>}</div><div><h4>Groups</h4>{overview.groups.slice(0, 8).map((group) => <div className="identity-mini-row" key={group.id}><strong>{group.display_name}</strong><span>{group.member_count} members · {group.source_code} · {group.source_state}</span></div>)}{!overview.groups.length && <p className="muted-copy">No directory groups found.</p>}</div></div></article>

      <article className="config-card identity-escalation-card"><div className="section-header"><div><h3>Escalation runtime</h3><p>Current OVERDUE routing health and hierarchy preview.</p></div></div><div className="identity-metrics"><div><strong>{overview.escalation.escalated_tasks}</strong><span>Escalated work</span></div><div><strong>{overview.escalation.pending_timers}</strong><span>Pending levels</span></div><div><strong>{overview.escalation.unresolved_24h}</strong><span>Unresolved · 24h</span></div><div><strong>{overview.escalation.failed_timers}</strong><span>Failed timers</span></div></div>
        {overview.escalation_policies.length ? <form className="identity-inline-form" onSubmit={(event) => void loadPreview(event)}><h4>Preview hierarchy</h4><label>Policy<select value={policyID} onChange={(event) => { setPolicyID(event.target.value); setPreview(null); }}><option value="">Choose policy</option>{overview.escalation_policies.map((policy) => <option key={policy.policy_id} value={policy.policy_id}>{policy.code} · v{policy.version}</option>)}</select></label><label>Sequence<select value={sequenceID} onChange={(event) => { setSequenceID(event.target.value); setPreview(null); }}><option value="">Choose sequence</option>{sequences.map((sequence) => <option key={sequence.ID} value={sequence.ID}>{sequence.ID} · {sequence.Trigger}</option>)}</select></label><label>Starting department<input value={previewDepartment} onChange={(event) => setPreviewDepartment(event.target.value)} placeholder="BANK / RISK / OPERATIONS"/></label><button className="secondary-button" disabled={!policyID || !sequenceID || busy !== ""} type="submit">Preview levels</button></form> : <p className="muted-copy">No active escalation sequence is configured.</p>}
        {preview && <ol className="identity-preview">{preview.steps.map((step) => <li key={step.index}><span>After {step.after}</span><strong>{humanize(step.responsibility)}</strong><small>{step.scope === "DEPARTMENT" ? step.department_path?.join(" / ") : step.scope === "LEGAL_ENTITY" ? "Legal entity scope" : "Department ancestry unavailable"}</small></li>)}</ol>}
        <p className="identity-footnote">Preview shows policy order and department scope only. The actual person is resolved from current authority, delegation, grants, segregation and visibility when the level fires.</p>
      </article>
    </div>
  </section>;
}

function parseDepartmentPath(value: string): string[] {
  return value.split(/[/>]/).map((part) => part.trim()).filter(Boolean);
}
function humanize(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()); }
