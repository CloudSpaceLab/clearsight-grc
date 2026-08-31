import { useState, type FormEvent } from "react";
import type { IdentityGroup, IdentityRole } from "../../identityAccessApi";
import { FocusedSheet } from "../FocusedSheet";

type SourceInput = { code: string; identity_issuer?: string; subject_attribute: "externalId" | "userName" };
type BindingInput = { group_id: string; role_template_id: string; department_path: string[] };

type SourceProps = {
  open: boolean;
  busy: boolean;
  onClose: () => void;
  onCreate: (input: SourceInput) => Promise<boolean>;
};

export function ProvisioningSourceComposer({ open, busy, onClose, onCreate }: SourceProps) {
  const [code, setCode] = useState("");
  const [issuer, setIssuer] = useState("");
  const [subjectAttribute, setSubjectAttribute] = useState<"externalId" | "userName">("externalId");
  if (!open) return null;

  async function submit(event: FormEvent) {
    event.preventDefault();
    const created = await onCreate({ code: code.trim(), identity_issuer: issuer.trim() || undefined, subject_attribute: subjectAttribute });
    if (!created) return;
    setCode("");
    setIssuer("");
    setSubjectAttribute("externalId");
    onClose();
  }

  return <FocusedSheet label="Add provisioning source" closeLabel="Close provisioning source form" panelClassName="access-composer-sheet" onClose={onClose}>
    <section className="access-composer" aria-labelledby="identity-source-composer-title">
      <span className="eyebrow">Directory provisioning</span>
      <h3 id="identity-source-composer-title">Add provisioning source</h3>
      <p>Connect a SCIM source without changing approval authority. The provisioning token is shown only once after creation.</p>
      <form onSubmit={(event) => void submit(event)}>
        <label>Code<input value={code} onChange={(event) => setCode(event.target.value)} required maxLength={64} placeholder="ENTRA"/></label>
        <label>OIDC issuer <span>(optional correlation)</span><input value={issuer} onChange={(event) => setIssuer(event.target.value)} maxLength={512} placeholder="https://id.example.com"/></label>
        <label>Stable subject<select value={subjectAttribute} onChange={(event) => setSubjectAttribute(event.target.value as "externalId" | "userName")}><option value="externalId">externalId</option><option value="userName">userName</option></select></label>
        <button type="submit" disabled={busy || !code.trim()}>{busy ? "Creating source…" : "Create source"}</button>
      </form>
    </section>
  </FocusedSheet>;
}

type BindingProps = {
  open: boolean;
  busy: boolean;
  groups: IdentityGroup[];
  roles: IdentityRole[];
  onClose: () => void;
  onCreate: (input: BindingInput) => Promise<boolean>;
};

export function GroupRoleBindingComposer({ open, busy, groups, roles, onClose, onCreate }: BindingProps) {
  const [groupID, setGroupID] = useState("");
  const [roleID, setRoleID] = useState("");
  const [department, setDepartment] = useState("");
  if (!open) return null;

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!groupID || !roleID) return;
    const created = await onCreate({ group_id: groupID, role_template_id: roleID, department_path: parseDepartmentPath(department) });
    if (!created) return;
    setGroupID("");
    setRoleID("");
    setDepartment("");
    onClose();
  }

  return <FocusedSheet label="Add group role mapping" closeLabel="Close group role mapping form" panelClassName="access-composer-sheet" onClose={onClose}>
    <section className="access-composer" aria-labelledby="identity-binding-composer-title">
      <span className="eyebrow">Workspace access</span>
      <h3 id="identity-binding-composer-title">Add group → role mapping</h3>
      <p>Map one connected directory group to a workspace role. Material decision authority remains governed separately.</p>
      <form onSubmit={(event) => void submit(event)}>
        <label>Directory group<select required value={groupID} onChange={(event) => setGroupID(event.target.value)}><option value="">Choose group</option>{groups.map((group) => <option value={group.id} key={group.id}>{group.display_name} · {group.source_code}</option>)}</select></label>
        <label>Role<select required value={roleID} onChange={(event) => setRoleID(event.target.value)}><option value="">Choose role</option>{roles.map((role) => <option value={role.id} key={role.id}>{role.code} · {role.capabilities.join(", ") || "no workspace permissions"}</option>)}</select></label>
        <label>Department path <span>(optional)</span><input value={department} onChange={(event) => setDepartment(event.target.value)} maxLength={512} placeholder="BANK / RISK / OPERATIONS"/></label>
        <button type="submit" disabled={busy || !groupID || !roleID}>{busy ? "Adding mapping…" : "Add mapping"}</button>
      </form>
    </section>
  </FocusedSheet>;
}

function parseDepartmentPath(value: string): string[] {
  return value.split(/[/>]/).map((part) => part.trim()).filter(Boolean);
}
